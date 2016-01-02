package main

import (
	"fmt"

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

func (c *RedisClient) SaveMessage(channel, msg string) error {
	// Key should be in the format of channelName:messages
	key := fmt.Sprintf("%s:messages", channel)

	return c.client.LPush(key, msg).Err()
}

func (c *RedisClient) GetAllMessages(channel string) ([]string, error) {
	// Key should be in the format of channelName:messages
	key := fmt.Sprintf("%s:messages", channel)

	return c.client.LRange(key, 0, -1)
}
