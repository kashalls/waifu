package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Discord settings
	DiscordToken string
	ShardID      int
	ShardCount   int

	// Redis settings
	RedisURL     string
	StreamPrefix string
}

func Load() (*Config, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is required")
	}

	shardID, err := envInt("SHARD_ID", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid SHARD_ID: %w", err)
	}

	shardCount, err := envInt("SHARD_COUNT", 1)
	if err != nil {
		return nil, fmt.Errorf("invalid SHARD_COUNT: %w", err)
	}

	redisURL := envString("REDIS_URL", "redis://localhost:6379")
	streamPrefix := envString("STREAM_PREFIX", "discord:")

	return &Config{
		DiscordToken: token,
		ShardID:      shardID,
		ShardCount:   shardCount,
		RedisURL:     redisURL,
		StreamPrefix: streamPrefix,
	}, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.Atoi(v)
}
