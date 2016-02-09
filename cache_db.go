package main

import (
	"fmt"

	redis "gopkg.in/redis.v3"
)

// RedisClient is a wrapper for the Redis lib to encapsulate common methods
// used in the application.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new RedisClient.
func NewRedisClient() *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
	}
}

// SaveMessage stores a message string.
func (c *RedisClient) SaveMessage(channel, msg string) error {
	// Key should be in the format of channelName:messages
	key := fmt.Sprintf("%s:messages", channel)

	return c.client.LPush(key, msg).Err()
}

// GetAllMessages retreives all messages for the specified channel.
func (c *RedisClient) GetAllMessages(channel string) ([]string, error) {
	// Key should be in the format of channelName:messages
	key := fmt.Sprintf("%s:messages", channel)

	return c.client.LRange(key, 0, -1).Result()
}

// Ping is used to check if the Redis connection is active.
func (c *RedisClient) Ping() error {
	return c.client.Ping().Err()
}
