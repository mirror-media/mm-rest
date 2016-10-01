package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mirror-media/mm-rest/gingo"
)

var (
	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
	redisPrimary = flag.String("redis-primary", ":6379", "Address to the Redis Primary server")
	redisAuth    = flag.String("redis-auth", "", "Password to the Redis server")
	name         = ""
	key          = ""
)

func main() {
	flag.Parse()
	router := gin.Default()
	log.Printf("redis address is %s\n", *redisAddress)
	log.Printf("redis auth is %s\n", *redisAuth)
	redisPrimary := gingo.NewRedisStore(*redisPrimary, *redisAuth)
	redis := gingo.NewRedisStore(*redisAddress, *redisAuth)
	router.GET("/ready", func(c *gin.Context) {
		ret, err := redis.Do("PING")
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
	router.GET("/check", func(c *gin.Context) {
		hasher := md5.New()
		name := c.Query("name")
		q1 := c.Query("q1")
		q3 := c.Query("q3")
		q4 := c.Query("q4")
		hasher.Write([]byte(name))
		redis_key := hex.EncodeToString(hasher.Sum(nil))
		log.Printf("redis key is %s\n", redis_key)
		ret, err := redis.Do("EXISTS", redis_key)
		if err != nil {
			c.JSON(500, gin.H{
				"_error": err,
			})
			return
		}
		ret, err = redis.Do("HGETALL", "listing-form")
		switch v := ret.(type) {
		case string:
			fmt.Println(v)
		case int, uint, int32, int64:
			fmt.Println(v)
		case byte:
			fmt.Println("byte")
		default:

		}
		//resp := "{ \"total\": " + total + ", \"q1r\": \"" + q1r + ", \"q3r\": " + q3r + ", \"q4r\": " + q4r + "}"
		c.JSON(200, gin.H{
			"result": ret,
		})
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
