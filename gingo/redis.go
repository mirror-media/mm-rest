package gingo

import (
	"log"
	"time"

	"github.com/garyburd/redigo/redis"
)

type RedisStore struct {
	pool *redis.Pool
}

func NewRedisStore(ipport, auth string) *RedisStore {
	var pool = &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ipport)
			if err != nil {
				return nil, err
			}
			if auth != "" {
				if _, err := c.Do("AUTH", auth); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		// TOB
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return &RedisStore{pool}
}

func (c *RedisStore) Do(cmd string, args ...interface{}) (ret interface{}, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	ret, err = conn.Do(cmd, args...)
	if err != nil {
		log.Printf("do %s err: %v", cmd, err)
	}
	return ret, err
}
