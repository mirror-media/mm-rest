package main

import (
	"flag"

	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
)

var (
	redisAddress   = flag.String("redis-address", ":6379", "Address to the Redis server")
	maxConnections = flag.Int("max-connections", 10, "Max connections to Redis")
)

func main() {
	router := gin.Default()
	router.GET("/ready", checkRedis)
	router.GET("/ready/:auth", checkRedisWithAuth)

	router.Run()
}

func checkRedisWithAuth(c *gin.Context) {
	auth := c.Param("auth")
	doRedis(c, "PING", auth)
}

func checkRedis(c *gin.Context) {
	doRedis(c, "PING", "")
}

func doRedis(c *gin.Context, cmd, auth string) {
	//client, err := redis.Dial("tcp", *redisAddress)
	redisPool := redis.NewPool(func() (redis.Conn, error) {
		client, err := redis.Dial("tcp", *redisAddress)
		if err != nil {
			c.JSON(500, gin.H{
				"message": err,
			})
		}
		return client, err
	}, *maxConnections)

	defer redisPool.Close()

	r := redisPool.Get()
	defer r.Close()

	if auth != "" {
		if _, err := r.Do("AUTH", auth); err != nil {
			c.JSON(400, gin.H{
				"message": "Invalid password",
			})
			return
		}
	}
	ret, err := r.Do(cmd)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err,
		})
	}
	c.JSON(200, gin.H{
		"message": ret,
	})
}
