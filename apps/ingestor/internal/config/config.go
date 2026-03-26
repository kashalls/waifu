package config

import (
	"fmt"
	"os"
)

type Config struct {
	RedisURL      string
	StreamPrefix  string
	ConsumerGroup string
	ConsumerName  string
}

func Load() (*Config, error) {
	redisURL := envString("REDIS_URL", "redis://localhost:6379")
	streamPrefix := envString("STREAM_PREFIX", "discord:")
	consumerGroup := envString("CONSUMER_GROUP", "ingestor")

	consumerName := os.Getenv("CONSUMER_NAME")
	if consumerName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("get hostname for consumer name: %w", err)
		}
		consumerName = hostname
	}

	return &Config{
		RedisURL:      redisURL,
		StreamPrefix:  streamPrefix,
		ConsumerGroup: consumerGroup,
		ConsumerName:  consumerName,
	}, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
