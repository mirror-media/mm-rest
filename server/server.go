package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
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
	router.GET("/check", func(c *gin.Context) {
		hasher := md5.New()
		name := c.Query("name")
		q1 := c.Query("q1")
		q3 := c.Query("q3")
		q4 := c.Query("q4")
		hasher.Write([]byte(name))
		redis_key := hex.EncodeToString(hasher.Sum(nil))
		log.Printf("redis key is %s\n", redis_key)
		ret, err := redisClient.Do("EXISTS", redis_key)
		if err != nil {
			c.JSON(500, gin.H{
				"_error": err,
			})
			return
		}
		ret, err = Values(redisClient.Do("HGETALL", "listing-form"))
		if err != nil {
			c.JSON(500, gin.H{
				"_error": err,
			})
			return
		}
		value, err := Strings(ret, err)
		if err != nil {
			c.JSON(500, gin.H{
				"_error": err,
			})
			return
		}
		c.JSON(200, gin.H{
			"result": value,
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
