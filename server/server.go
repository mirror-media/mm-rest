package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/itsjamie/gin-cors"
)

var (
	sqlAddress     = flag.String("sql-address", "127.0.0.1", "Address to the SQL server")
	sqlAuth        = flag.String("sql-auth", "", "Password to SQL server")
	benchmarkToken = flag.String("benchmark-token", "", "Token used for benchmark service")
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

const benchmark = "https://apidocs.benchmarkemail.com/app/website/services/api.php?method=%s"

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

func main() {
	flag.Parse()
	fmt.Printf("sql address:%s, auth:%s \n", *sqlAddress, *sqlAuth)

	// db, err := sql.Open("mysql", "root:12345@tcp(localhost:3306)/members")
	db, err := sql.Open("mysql", fmt.Sprintf("root:%s@tcp(%s)/members", *sqlAuth, *sqlAddress))
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
		c.String(http.StatusOK, "healthy")
	})

	router.GET("/user/:id", func(c *gin.Context) {
		id := c.Param("id")

		user, status := getUser(id, db)
		c.JSON(status, user)
	})

	router.POST("/user", func(c *gin.Context) {
		// item := c.PostForm("item")
		// user := c.PostForm("user")
		post := postStruct{}
		c.Bind(&post)

		if post.Item == "" || post.User == "" {
			c.String(400, "Bad Request - Either request body is invalid")
			return
		}

		var (
			userID      int64
			userEmail   string
			listID      string
			epaperTitle string
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
			fmt.Printf("No user email %s\n", post.User)
		case err != nil:
			log.Fatal(err)
		// User exists
		default:
			fmt.Printf("User exist id: %d, email:%s\n", userID, userEmail)
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
