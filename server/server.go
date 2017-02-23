package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	//"github.com/dpapathanasiou/go-recaptcha"
	"github.com/gin-gonic/gin"
	"github.com/itsjamie/gin-cors"
	"github.com/mirror-media/mm-rest/gingo"

	"golang.org/x/net/context"
	// Imports the Stackdriver Logging client package
	"cloud.google.com/go/logging"
)

var (
	projectId    = flag.String("project-id", "mirrormedia-1470651750304", "Your Google Cloud Platform project ID")
	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
	redisPrimary = flag.String("redis-primary", ":6379", "Address to the Redis Primary server")
	redisAuth    = flag.String("redis-auth", "", "Password to the Redis server")
	secret       = flag.String("secret", "", "Secret for the Google recaptcha")
	name         = ""
	key          = ""
	ErrNil       = errors.New("redigo: nil returned")
	//redisPrimary := gingo.NewRedisStore(*redisPrimary, *redisAuth)
	redisClient = gingo.NewRedisStore(*redisAddress, *redisAuth)
)

type Error string

func main() {
	flag.Parse()
	router := gin.Default()
	log.Printf("redis address is %s\n", *redisAddress)
	log.Printf("redis auth is %s\n", *redisAuth)
	router.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		Credentials:     true,
		ValidateHeaders: false,
	}))
	router.GET("/ready", ready)
	router.GET("/weblog", weblog)

	router.Run()
}

func ready(c *gin.Context) {
	c.JSON(200, gin.H{
		"result": "ok",
	})
	return
}

func weblog(c *gin.Context) {
	payload := c.Query("log")
	ctx := context.Background()
	client, err := logging.NewClient(ctx, *projectId)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	logName := "analytics"

	logger := client.Logger(logName)
	logger.Log(logging.Entry{Payload: payload})

	err = client.Close()
	if err != nil {
		log.Fatalf("Failer to close client: %v", err)
	}

	fmt.Printf("Logged: %v", payload)

}
