package main

import (
	"flag"
	"log"

	"github.com/mirror-media/mm-rest/gingo"
)

var (
	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
	redisAuth    = flag.String("redis-auth", "", "Password to the Redis server")
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
				"message": err,
			})
			return
		}
		c.JSON(200, gin.H{
			"message": ret,
		})
	})

	router.Run()
}
