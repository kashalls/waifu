package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kashalls/waifu/apps/ingestor/internal/config"
	"github.com/kashalls/waifu/pkg/event"
	"github.com/redis/go-redis/v9"
)

// Handler processes a single Discord event.
type Handler func(ctx context.Context, e event.Event) error

// Consumer reads Discord events from per-event-type Redis streams using a
// single consumer group shared across all streams.
type Consumer struct {
	client     *redis.Client
	cfg        *config.Config
	handlers   map[string]Handler // keyed by event type e.g. "MESSAGE_CREATE"
	streamKeys []string           // e.g. ["discord:MESSAGE_CREATE", ...]
}

func New(cfg *config.Config) (*Consumer, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	c := &Consumer{
		client:   client,
		cfg:      cfg,
		handlers: make(map[string]Handler),
	}
	c.registerHandlers()

	// Derive stream keys from registered event types and ensure consumer
	// groups exist on each stream.
	for eventType := range c.handlers {
		key := cfg.StreamPrefix + eventType
		c.streamKeys = append(c.streamKeys, key)
		err := client.XGroupCreateMkStream(context.Background(), key, cfg.ConsumerGroup, "$").Err()
		if err != nil && !isBusyGroup(err) {
			return nil, fmt.Errorf("create consumer group on %s: %w", key, err)
		}
	}

	slog.Info("connected to redis",
		"stream_prefix", cfg.StreamPrefix,
		"streams", len(c.streamKeys),
		"group", cfg.ConsumerGroup,
		"consumer", cfg.ConsumerName,
	)
	return c, nil
}

// registerHandlers maps Discord event type strings to their handler functions.
// Add new event types here as your application grows.
func (c *Consumer) registerHandlers() {
	c.handlers["MESSAGE_CREATE"] = c.handleMessageCreate
	c.handlers["MESSAGE_UPDATE"] = c.handleMessageUpdate
	c.handlers["MESSAGE_DELETE"] = c.handleMessageDelete
	c.handlers["MESSAGE_DELETE_BULK"] = c.handleMessageDeleteBulk
	c.handlers["GUILD_CREATE"] = c.handleGuildCreate
	c.handlers["GUILD_UPDATE"] = c.handleGuildUpdate
	c.handlers["GUILD_DELETE"] = c.handleGuildDelete
	c.handlers["GUILD_MEMBER_ADD"] = c.handleGuildMemberAdd
	c.handlers["GUILD_MEMBER_UPDATE"] = c.handleGuildMemberUpdate
	c.handlers["GUILD_MEMBER_REMOVE"] = c.handleGuildMemberRemove
	c.handlers["GUILD_BAN_ADD"] = c.handleGuildBanAdd
	c.handlers["GUILD_BAN_REMOVE"] = c.handleGuildBanRemove
	c.handlers["PRESENCE_UPDATE"] = c.handlePresenceUpdate
	c.handlers["VOICE_STATE_UPDATE"] = c.handleVoiceStateUpdate
	c.handlers["INTERACTION_CREATE"] = c.handleInteractionCreate
	c.handlers["CHANNEL_CREATE"] = c.handleChannelCreate
	c.handlers["CHANNEL_UPDATE"] = c.handleChannelUpdate
	c.handlers["CHANNEL_DELETE"] = c.handleChannelDelete
	c.handlers["THREAD_CREATE"] = c.handleThreadCreate
	c.handlers["THREAD_UPDATE"] = c.handleThreadUpdate
	c.handlers["THREAD_DELETE"] = c.handleThreadDelete
	c.handlers["REACTION_ADD"] = c.handleReactionAdd
	c.handlers["REACTION_REMOVE"] = c.handleReactionRemove
}

// Run blocks, continuously reading from all registered streams until ctx is
// cancelled. XREADGROUP is called with all stream keys in a single blocking
// call so a single goroutine is sufficient.
func (c *Consumer) Run(ctx context.Context) error {
	slog.Info("ingestor running",
		"streams", len(c.streamKeys),
		"group", c.cfg.ConsumerGroup,
	)

	// XReadGroup expects: streams = [key1, key2, ..., ">", ">", ...]
	streamArgs := make([]string, len(c.streamKeys)*2)
	copy(streamArgs, c.streamKeys)
	for i := range c.streamKeys {
		streamArgs[len(c.streamKeys)+i] = ">"
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.cfg.ConsumerGroup,
			Consumer: c.cfg.ConsumerName,
			Streams:  streamArgs,
			Count:    10,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("xreadgroup error", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			// Derive the event type from the stream key: "discord:MESSAGE_CREATE" → "MESSAGE_CREATE"
			eventType := strings.TrimPrefix(stream.Stream, c.cfg.StreamPrefix)
			handler, ok := c.handlers[eventType]

			for _, msg := range stream.Messages {
				if !ok {
					slog.Debug("no handler for stream", "stream", stream.Stream)
					_ = c.client.XAck(ctx, stream.Stream, c.cfg.ConsumerGroup, msg.ID).Err()
					continue
				}
				if err := c.processMessage(ctx, stream.Stream, msg, handler); err != nil {
					slog.Error("failed to process message", "stream", stream.Stream, "id", msg.ID, "error", err)
					// Do not ack on error so the message can be reclaimed.
					continue
				}
				if err := c.client.XAck(ctx, stream.Stream, c.cfg.ConsumerGroup, msg.ID).Err(); err != nil {
					slog.Error("failed to ack message", "stream", stream.Stream, "id", msg.ID, "error", err)
				}
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, streamKey string, msg redis.XMessage, handler Handler) error {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return fmt.Errorf("missing payload field in message %s", msg.ID)
	}

	var e event.Event
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	return handler(ctx, e)
}

func (c *Consumer) Close() error {
	return c.client.Close()
}

// isBusyGroup returns true when the Redis error indicates the consumer group
// already exists — which is safe to ignore.
func isBusyGroup(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}

// ---------------------------------------------------------------------------
// Event handlers — implement your business logic here.
// ---------------------------------------------------------------------------

func (c *Consumer) handleMessageCreate(ctx context.Context, e event.Event) error {
	slog.Info("MESSAGE_CREATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleMessageUpdate(ctx context.Context, e event.Event) error {
	slog.Info("MESSAGE_UPDATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleMessageDelete(ctx context.Context, e event.Event) error {
	slog.Info("MESSAGE_DELETE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleMessageDeleteBulk(ctx context.Context, e event.Event) error {
	slog.Info("MESSAGE_DELETE_BULK", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildCreate(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_CREATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildUpdate(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_UPDATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildDelete(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_DELETE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildMemberAdd(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_MEMBER_ADD", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildMemberUpdate(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_MEMBER_UPDATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildMemberRemove(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_MEMBER_REMOVE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildBanAdd(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_BAN_ADD", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleGuildBanRemove(ctx context.Context, e event.Event) error {
	slog.Info("GUILD_BAN_REMOVE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handlePresenceUpdate(ctx context.Context, e event.Event) error {
	slog.Debug("PRESENCE_UPDATE", "shard_id", e.ShardID)
	return nil
}

func (c *Consumer) handleVoiceStateUpdate(ctx context.Context, e event.Event) error {
	slog.Info("VOICE_STATE_UPDATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleInteractionCreate(ctx context.Context, e event.Event) error {
	slog.Info("INTERACTION_CREATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleChannelCreate(ctx context.Context, e event.Event) error {
	slog.Info("CHANNEL_CREATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleChannelUpdate(ctx context.Context, e event.Event) error {
	slog.Info("CHANNEL_UPDATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleChannelDelete(ctx context.Context, e event.Event) error {
	slog.Info("CHANNEL_DELETE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleThreadCreate(ctx context.Context, e event.Event) error {
	slog.Info("THREAD_CREATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleThreadUpdate(ctx context.Context, e event.Event) error {
	slog.Info("THREAD_UPDATE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleThreadDelete(ctx context.Context, e event.Event) error {
	slog.Info("THREAD_DELETE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleReactionAdd(ctx context.Context, e event.Event) error {
	slog.Info("REACTION_ADD", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}

func (c *Consumer) handleReactionRemove(ctx context.Context, e event.Event) error {
	slog.Info("REACTION_REMOVE", "shard_id", e.ShardID, "seq", e.Sequence)
	return nil
}
