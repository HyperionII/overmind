package main

import (
	redis "gopkg.in/redis.v3"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient() *RedisClient {
	return &RedisClient{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}
