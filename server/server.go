package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mirror-media/mm-rest/gingo"
)

var (
	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
	redisAuth    = flag.String("redis-auth", "", "Password to the Redis server")
	name         = ""
	email        = ""
	key          = ""
)

func main() {
	flag.Parse()
	router := gin.Default()
	log.Printf("redis address is %s\n", *redisAddress)
	log.Printf("redis auth is %s\n", *redisAuth)
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
		email := c.Query("email")
		hasher.Write([]byte(name + email))
		redis_key := hex.EncodeToString(hasher.Sum(nil))
		log.Printf("redis key is %s\n", redis_key)
		ret, err := redis.Do("EXISTS", redis_key)
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

	router.Run()
}
