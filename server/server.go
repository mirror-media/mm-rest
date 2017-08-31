package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/itsjamie/gin-cors"
)

var (
	sqlUser        = flag.String("sql-user", "root", "User account to SQL server")
	sqlAddress     = flag.String("sql-address", "127.0.0.1:3306", "Address to the SQL server")
	sqlAuth        = flag.String("sql-auth", "", "Password to SQL server")
	mailchimpToken = flag.String("mailchimp-token", "", "Token used for benchmark service")
)

type epaperStruct struct {
	Name *string `json:"name"`
	ID   *string `json:"id"`
}

type userStruct struct {
	Name         string         `form:"user" json:"user" binding:"required"`
	Subscription []epaperStruct `form:"subscription" json:"result" binding:"required"`
}

type postStruct struct {
	User string `json:"user"`
	Item string `json:"item"`
}

const mailchimpAPI = "https://us16.api.mailchimp.com/3.0/lists/"

func getUser(name string, db *sql.DB) (userStruct, int) {

	rows, err := db.Query("SELECT user_email, epaper_title, list_id FROM users u INNER JOIN user_epaper ue ON u.user_id = ue.user_id INNER JOIN epapers e ON e.list_id = ue.epaper_id WHERE user_email = ?", name)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var email string

	user := userStruct{}
	if rows.Next() {

		epaper := epaperStruct{}
		err := rows.Scan(&email, &epaper.Name, &epaper.ID)
		if err != nil {
			log.Fatal(err)
		}

		user.Name = email
		user.Subscription = append(user.Subscription, epaper)

		for rows.Next() {
			epaper := epaperStruct{}
			err := rows.Scan(&email, &epaper.Name, &epaper.ID)
			if err != nil {
				log.Fatal(err)
			}

			user.Name = email
			user.Subscription = append(user.Subscription, epaper)
		}
	} else {
		return user, 404
	}

	return user, 200
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func chimpWorker(method string, listID string, userEmail string, desiredStatus string, db *sql.DB) {

	fmt.Printf("request method:%s, desiredStatus:%s, user email:%s\n", method, desiredStatus, userEmail)
	retries := 3
retryLoop:
	for retries > 0 {

		urlSlice := []string{mailchimpAPI}
		if method == "POST" {
			urlSlice = append(urlSlice, listID, "/members/")
		} else if method == "PUT" {
			urlSlice = append(urlSlice, listID, "/members/", getMD5Hash(userEmail))
		}
		url := strings.Join(urlSlice, "")

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: tr}

		reqBody := map[string]string{"email_address": userEmail, "status": desiredStatus}
		reqMarshaled, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Println(err)
		}
		// Prepare https request
		req, err := http.NewRequest(method, url, bytes.NewBuffer(reqMarshaled))
		req.SetBasicAuth("anystring", *mailchimpToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
		}
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)

		body, _ := ioutil.ReadAll(resp.Body)
		// fmt.Println(string(body))
		// bs := string(body)
		// fmt.Println("response Body:", bs)

		respJSON := make(map[string]interface{})
		err = json.Unmarshal(body, &respJSON)
		// statusCode := int(respJSON["status"].(float64))

		retries--
		switch {
		case resp.Status == "200 OK":
			// Success
			fmt.Println("Return 200, Success!")
			_, err = db.Exec("UPDATE user_epaper ue INNER JOIN users u ON ue.user_id = u.user_id SET mailchimp = 1 WHERE u.user_email = ? AND ue.epaper_id = ?", userEmail, listID)
			if err != nil {
				fmt.Println(err)
			}
			break retryLoop
		case resp.Status == "400 Bad Request":
			fmt.Printf("error body:%s\n", string(body))
			fmt.Println("Return 400 Bad Request!")
			if method == "POST" && retries > 0 {
				method = "PUT"
				time.Sleep(time.Second * time.Duration(5*(3-retries)))
			} else if method == "POST" || retries <= 0 {
				fmt.Printf("method %s fail to insert mailchimp for %s in list %s", method, userEmail, listID)
			}
		default:
			fmt.Printf("error body:%s\n", string(body))
		} // End of switch
	} // End of for
}

func main() {
	flag.Parse()
	fmt.Printf("sql user:%s, sql address:%s, auth:%s \n", *sqlUser, *sqlAddress, *sqlAuth)

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/members", *sqlUser, *sqlAuth, *sqlAddress))

	if err != nil {
		log.Fatal(err)
	}
	router := gin.Default()
	router.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, POST",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		Credentials:     true,
		ValidateHeaders: false,
	}))

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "")
	})

	router.GET("/user/:id", func(c *gin.Context) {
		id := c.Param("id")

		user, status := getUser(id, db)
		c.JSON(status, user)
	})

	router.POST("/user", func(c *gin.Context) {
		post := postStruct{}
		c.Bind(&post)

		if post.Item == "" || post.User == "" {
			c.String(400, "Bad Request - Either user or item is empty. Both are required.")
			return
		}

		if post.Item != "people" || post.Item != "foodtravel" {
			c.String(400, "Bad Request - Requested item is invalid.")
			return
		}

		var (
			userID      int64
			userEmail   string
			listID      string
			epaperTitle string
			userExists  bool
		)
		// First get the email from SQL
		err := db.QueryRow("SELECT user_id, user_email FROM users WHERE user_email = ?", post.User).Scan(&userID, &userEmail)

		switch {
		// User with this id doesn't exist, INSERT
		case err == sql.ErrNoRows:
			res, err := db.Exec("INSERT INTO users (user_email) VALUES (?)", post.User)
			if err != nil {
				log.Fatal(err)
			} else {
				userID, err = res.LastInsertId()
				if err != nil {
					fmt.Println("Error:", err.Error())
				} else {
					fmt.Printf("LastInsertId:%d\n", userID)
				}
			}
			// fmt.Println(res)
			userExists = false
			userEmail = post.User
			fmt.Printf("No user email %s\n", post.User)
		case err != nil:
			log.Fatal(err)
		// User exists
		default:
			fmt.Printf("User exist id: %d, email:%s\n", userID, userEmail)
			userExists = true
		}

		err = db.QueryRow("SELECT list_id, epaper_title FROM epapers e INNER JOIN user_epaper ue ON ue.epaper_id = e.list_id WHERE ue.user_id = ? and epaper_title = ?", userID, post.Item).Scan(&listID, &epaperTitle)
		switch {
		// User with this id doesn't exist, INSERT
		case err == sql.ErrNoRows:
			fmt.Printf("No item name %s\n", post.Item)
			err := db.QueryRow("SELECT list_id FROM epapers WHERE epaper_title = ?", post.Item).Scan(&listID)
			if err != nil {
				log.Fatal(err)
			} else {
				_, err := db.Exec("INSERT INTO user_epaper (user_id, epaper_id) VALUES (?,?)", userID, listID)
				if err != nil {
					fmt.Println("Error:", err.Error())
				}
			}

			if userExists {
				go chimpWorker("PUT", listID, userEmail, "subscribed", db)
			} else {
				go chimpWorker("POST", listID, userEmail, "subscribed", db)
			}
		case err != nil:
			log.Fatal(err)
		// User subscribe this epaper, DELETE!
		default:
			_, err := db.Exec("DELETE FROM user_epaper WHERE user_id = ? AND epaper_id = ?", userID, listID)
			if err != nil {
				log.Fatal(err)
			} else {
				fmt.Printf("Delete %s for %s\n", post.Item, post.User)
			}
			go chimpWorker("PUT", listID, userEmail, "unsubscribed", db)

		}
		res, status := getUser(post.User, db)
		// Still return 200 for empty subscription in POST
		if status == 404 {
			status = 200
		}
		c.JSON(status, res)
	})

	router.Run()
}
