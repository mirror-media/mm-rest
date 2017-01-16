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
)

var (
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

// Values is a helper that converts an array command reply to a []interface{}.
// If err is not equal to nil, then Values returns nil, err. Otherwise, Values
// converts the reply as follows:
//
//  Reply type      Result
//  array           reply, nil
//  nil             nil, ErrNil
//  other           nil, error
func Values(reply interface{}, err error) ([]interface{}, error) {
	if err != nil {
		return nil, err
	}
	switch reply := reply.(type) {
	case []interface{}:
		return reply, nil
	case nil:
		return nil, ErrNil
	}
	return nil, fmt.Errorf("redigo: unexpected type for Values, got type %T", reply)
}

func Strings(reply interface{}, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	switch reply := reply.(type) {
	case []interface{}:
		result := make([]string, len(reply))
		for i := range reply {
			if reply[i] == nil {
				continue
			}
			p, ok := reply[i].([]byte)
			if !ok {
				return nil, fmt.Errorf("redigo: unexpected element type for Strings, got type %T", reply[i])
			}
			result[i] = string(p)
		}
		return result, nil
	case nil:
		return nil, ErrNil
	}
	return nil, fmt.Errorf("redigo: unexpected type for Strings, got type ")
}

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
	qid := c.Query("qid")
	ret, err := Strings(redisClient.Do("MGET", qid))
	if err != nil {
		c.JSON(500, gin.H{
			"message": err,
			"_error":  err,
		})
		return
	}
	c.JSON(200, gin.H{
		"message": ret,
		"result":  ret,
	})
}
