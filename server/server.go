package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/dpapathanasiou/go-recaptcha"
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
	redisPrimary := gingo.NewRedisStore(*redisPrimary, *redisAuth)
	redisClient := gingo.NewRedisStore(*redisAddress, *redisAuth)
	router.GET("/ready", func(c *gin.Context) {
		ret, err := redisClient.Do("PING")
		if err != nil {
			c.JSON(500, gin.H{
				"_error": err,
			})
			return
		}
		c.JSON(200, gin.H{
			"result": ret,
		})
	})

	router.GET("/tpe", func(c *gin.Context) {
		ret, err := Values(redisClient.Do("HGETALL", "tpe-form"))
		if err != nil {
			c.JSON(500, gin.H{
				"_error": "Internal Server Error",
			})
			return
		}
		value, err := Strings(ret, err)
		if err != nil {
			c.JSON(500, gin.H{
				"_error": "Internal Server Error",
			})
			return
		} else {
			c.JSON(200, gin.H{
				"result": value,
			})
		}
		q1 := c.Query("q1")
		q2 := c.Query("q2")
		q3 := c.Query("q3")
		q4 := c.Query("q4")
		captcha := c.Query("g-recaptcha-response")
		if captcha == "" {
			c.JSON(500, gin.H{
				"_error": "Internal Server Error",
			})
			return
		}
		recaptcha.Init(*secret)
		result := recaptcha.Confirm("", captcha)
		log.Printf("capcha return is %s\n", result)
		if result {
			redisPrimary.Do("HINCRBY", "tpe-form", "total", 1)
			if q1 == "1" {
				redisPrimary.Do("HINCRBY", "tpe-form", "q1r", 1)
			}
			if q2 == "1" {
				redisPrimary.Do("HINCRBY", "tpe-form", "q2r", 1)
			}
			if q3 == "1" {
				redisPrimary.Do("HINCRBY", "tpe-form", "q3r", 1)
			}
			if q4 == "1" {
				redisPrimary.Do("HINCRBY", "tpe-form", "q4r", 1)
			}
		}
	})

	router.GET("/check", func(c *gin.Context) {
		hasher := md5.New()
		ret, err := Values(redisClient.Do("HGETALL", "listing-form"))
		if err != nil {
			c.JSON(500, gin.H{
				"_error": "Internal Server Error",
			})
			return
		}
		value, err := Strings(ret, err)
		if err != nil {
			c.JSON(200, gin.H{
				"result": value,
			})
			return
		}
		name := c.Query("name")
		q1 := c.Query("q1")
		q3 := c.Query("q3")
		q4 := c.Query("q4")
		captcha := c.Query("g-recaptcha-response")
		recaptcha.Init(*secret)
		recaptcha.Confirm("", captcha)
		if err != nil || name == "" || q1 == "" || q3 == "" || q4 == "" {
			c.JSON(200, gin.H{
				"result": value,
			})
			return
		}
		hasher.Write([]byte(name))
		redis_key := hex.EncodeToString(hasher.Sum(nil))
		name_check, err := redisClient.Do("EXISTS", redis_key)
		if err != nil {
			c.JSON(500, gin.H{
				"_error": err,
			})
			return
		} else {
			c.JSON(200, gin.H{
				"result": value,
				"check":  name_check,
			})
		}
		redisPrimary.Do("HINCRBY", "listing-form", "total", 1)
		if q1 == "1" {
			redisPrimary.Do("HINCRBY", "listing-form", "q1r", 1)
		}
		if q3 == "1" {
			redisPrimary.Do("HINCRBY", "listing-form", "q3r", 1)
		}
		if q4 == "1" {
			redisPrimary.Do("HINCRBY", "listing-form", "q4r", 1)
		}
	})

	router.Run()
}
