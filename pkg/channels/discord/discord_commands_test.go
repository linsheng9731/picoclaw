package discord

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestThinkPublishesNormalizedCommand(t *testing.T) {
	msgBus := bus.NewMessageBus()
	t.Cleanup(msgBus.Close)

	commander := &cmd{
		config: &config.Config{},
		bus:    msgBus,
	}

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-1",
			ChannelID: "ch-1",
			GuildID:   "guild-1",
			Content:   "/thinking high",
			Author: &discordgo.User{
				ID:       "u-1",
				Username: "bob",
			},
		},
	}

	if err := commander.Think(context.Background(), nil, msg); err != nil {
		t.Fatalf("Think() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	got, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound think command")
	}
	if got.Content != "/think high" {
		t.Fatalf("inbound content = %q, want %q", got.Content, "/think high")
	}
	if got.Channel != "discord" || got.ChatID != "ch-1" {
		t.Fatalf("unexpected routing: channel=%q chat=%q", got.Channel, got.ChatID)
	}
}
