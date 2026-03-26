package gateway

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kashalls/waifu/apps/listener/internal/config"
	"github.com/kashalls/waifu/apps/listener/internal/publisher"
)

// Session wraps a discordgo session and forwards all raw gateway dispatch
// events to the Redis publisher.
type Session struct {
	dg        *discordgo.Session
	publisher *publisher.Publisher
	cfg       *config.Config
}

func New(cfg *config.Config) (*Session, error) {
	pub, err := publisher.New(cfg.RedisURL, cfg.StreamPrefix, cfg.ShardID)
	if err != nil {
		return nil, err
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		_ = pub.Close()
		return nil, err
	}

	// Set shard identity provided by discord.js-kuberscaler.
	dg.ShardID = cfg.ShardID
	dg.ShardCount = cfg.ShardCount

	// Request all intents. Privileged intents (GUILD_MEMBERS, PRESENCE_UPDATE,
	// MESSAGE_CONTENT) must be enabled in the Discord Developer Portal.
	dg.Identify.Intents = discordgo.IntentsAll

	s := &Session{dg: dg, publisher: pub, cfg: cfg}
	dg.AddHandler(s.onRawEvent)
	dg.AddHandler(s.onConnect)
	dg.AddHandler(s.onDisconnect)
	dg.AddHandler(s.onResume)

	return s, nil
}

// onRawEvent is called for every Discord gateway DISPATCH event (op=0).
// discordgo dispatches *discordgo.Event to handlers registered for that type,
// giving us a single catch-all hook with the raw JSON payload intact.
func (s *Session) onRawEvent(_ *discordgo.Session, e *discordgo.Event) {
	if e.Type == "" {
		return
	}
	ctx := context.Background()
	if err := s.publisher.Publish(ctx, e.Type, e.Sequence, e.RawData); err != nil {
		slog.Error("failed to publish event", "type", e.Type, "error", err)
	}
}

func (s *Session) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	slog.Info("connected to discord gateway", "shard_id", s.cfg.ShardID, "shard_count", s.cfg.ShardCount)
}

func (s *Session) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	slog.Warn("disconnected from discord gateway", "shard_id", s.cfg.ShardID)
}

func (s *Session) onResume(_ *discordgo.Session, _ *discordgo.Resumed) {
	slog.Info("resumed discord gateway session", "shard_id", s.cfg.ShardID)
}

func (s *Session) Open() error {
	return s.dg.Open()
}

func (s *Session) Close() {
	_ = s.dg.Close()
	_ = s.publisher.Close()
}
