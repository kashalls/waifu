package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kashalls/waifu/apps/ingestor/internal/config"
	"github.com/kashalls/waifu/apps/ingestor/internal/consumer"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	c, err := consumer.New(cfg)
	if err != nil {
		slog.Error("failed to create consumer", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	slog.Info("starting ingestor",
		"stream_prefix", cfg.StreamPrefix,
		"group", cfg.ConsumerGroup,
		"consumer", cfg.ConsumerName,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := c.Run(ctx); err != nil {
		slog.Error("ingestor error", "error", err)
		os.Exit(1)
	}

	slog.Info("ingestor shutdown complete")
}
