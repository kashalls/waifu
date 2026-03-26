package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kashalls/waifu/apps/listener/internal/config"
	"github.com/kashalls/waifu/apps/listener/internal/gateway"
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

	slog.Info("starting listener", "shard_id", cfg.ShardID, "shard_count", cfg.ShardCount)

	session, err := gateway.New(cfg)
	if err != nil {
		slog.Error("failed to create gateway session", "error", err)
		os.Exit(1)
	}

	if err := session.Open(); err != nil {
		slog.Error("failed to open gateway connection", "error", err)
		os.Exit(1)
	}
	defer session.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutting down listener", "shard_id", cfg.ShardID)
}
