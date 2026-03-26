package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kashalls/waifu/pkg/event"
	"github.com/redis/go-redis/v9"
)

// Publisher writes Discord events to per-event-type Redis streams.
type Publisher struct {
	client       *redis.Client
	streamPrefix string
	shardID      int
}

func New(redisURL, streamPrefix string, shardID int) (*Publisher, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	slog.Info("connected to redis", "stream_prefix", streamPrefix)
	return &Publisher{client: client, streamPrefix: streamPrefix, shardID: shardID}, nil
}

// Publish serialises a raw Discord dispatch event and appends it to the
// stream for that event type: {streamPrefix}{EVENT_TYPE}
// e.g. "discord:MESSAGE_CREATE".
func (p *Publisher) Publish(ctx context.Context, eventType string, sequence int64, rawData json.RawMessage) error {
	e := event.Event{
		Type:     eventType,
		ShardID:  p.shardID,
		Sequence: sequence,
		Data:     rawData,
	}

	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	streamKey := p.streamPrefix + strings.ToUpper(eventType)
	if err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{
			"shard_id": p.shardID,
			"payload":  string(payload),
		},
	}).Err(); err != nil {
		return fmt.Errorf("xadd %s: %w", streamKey, err)
	}

	slog.Debug("published event", "type", eventType, "stream", streamKey, "shard_id", p.shardID, "seq", sequence)
	return nil
}

func (p *Publisher) Close() error {
	return p.client.Close()
}
