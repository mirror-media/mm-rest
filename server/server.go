package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	// "github.com/itsjamie/gin-cors"
)

// var (
// 	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
// 	redisPrimary = flag.String("redis-primary", ":6379", "Address to the Redis Primary server")
// 	redisAuth    = flag.String("redis-auth", "", "Password to the Redis server")
// 	secret       = flag.String("secret", "", "Secret for the Google recaptcha")
// 	name         = ""
// 	key          = ""
// 	ErrNil       = errors.New("redigo: nil returned")
// )

type epaperStruct struct {
	Name *string `json:"name"`
	ID   *string `json:"listID"`
}

type userStruct struct {
	Name         string         `form:"user" json:"user" binding:"required"`
	Subscription []epaperStruct `form:"subscription" json:"result" binding:"required"`
}

func main() {

	db, err := sql.Open("mysql", "root:12345@tcp(localhost:3306)/members")
	if err != nil {
		log.Fatal(err)
	}
	router := gin.Default()
	// router.Use(cors.Middleware(cors.Config{
	// 	Origins:         "*",
	// 	Methods:         "GET, PUT, POST, DELETE",
	// 	RequestHeaders:  "Origin, Authorization, Content-Type",
	// 	ExposedHeaders:  "",
	// 	MaxAge:          50 * time.Second,
	// 	Credentials:     true,
	// 	ValidateHeaders: false,
	// }))

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "healthy")
	})

	router.GET("/user/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Use one-time preparation because we only query once
		rows, err := db.Query("select email, name, listID from users u inner join user_epaper ue on u.id = ue.user_id inner join epapers e on e.listID = ue.epaper_id where email = ?", id)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		fmt.Printf("row length: %d\n", len(cols))

		var email string

		user := userStruct{}
		// subscription := make([]map[string]string, 0)
		for rows.Next() {
			epaper := epaperStruct{}
			// err := rows.Scan(&email, &epaperName, &epaperListid)
			err := rows.Scan(&email, &epaper.Name, &epaper.ID)
			if err != nil {
				log.Fatal(err)
			}

			// if (epaper{}) == epaper {
			// 	fmt.Println("not exist in database")
			// }

			if email == "" {
				email = id
			}
			fmt.Println(email)
			user.Name = email
			user.Subscription = append(user.Subscription, epaper)
			// subscription = append(subscription, epaper)
			// fmt.Println("after assign :", subscription)
		}

		// var msg struct {
		// 	Name         string              `json:"user"`
		// 	Subscription []map[string]string `json:"result"`
		// }

		// msg.Name = email
		// msg.Subscription = subscription

		// if email == "" {
		// 	email = id
		// }
		//fmt.Printf("The 2nd length of subscription: %d\n", len(subscription))
		// c.String(http.StatusOK, "Hello %s subscribe to %s", email, subscription)
		// c.JSON(200, gin.H{
		// 	"user":   email,
		// 	"result": subscription,
		// })
		c.JSON(200, user)
	})

	router.POST("user", func(c *gin.Context) {
		// id := c.Param("id")
		// fmt.Println(id)
		subscription := c.PostForm("item")
		email := c.PostForm("user")
		fmt.Println(email)
		fmt.Println(subscription)
	})

	router.Run()
}
