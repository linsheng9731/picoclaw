package channels

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/config"
)

type TelegramCommander interface {
	Help(ctx context.Context, message telego.Message) error
	Start(ctx context.Context, message telego.Message) error
	Show(ctx context.Context, message telego.Message) error
	List(ctx context.Context, message telego.Message) error
	Switch(ctx context.Context, message telego.Message) error
}

// ModelSwitcher defines the interface for hot-reloading models.
type ModelSwitcher interface {
	SwitchModel(modelName string) error
}

type cmd struct {
	bot          *telego.Bot
	config       *config.Config
	configPath   string
	modelSwitcher ModelSwitcher
}

func NewTelegramCommands(bot *telego.Bot, cfg *config.Config, configPath string, switcher ModelSwitcher) TelegramCommander {
	return &cmd{
		bot:          bot,
		config:       cfg,
		configPath:   configPath,
		modelSwitcher: switcher,
	}
}

func commandArgs(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (c *cmd) Help(ctx context.Context, message telego.Message) error {
	msg := `/start - Start the bot
/help - Show this help message
/show [model|channel] - Show current configuration
/list [models|channels] - List available options
/switch <model_name> - Switch to a different model
	`
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   msg,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Start(ctx context.Context, message telego.Message) error {
	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "Hello! I am PicoClaw 🦞",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Show(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /show [model|channel]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "model":
		response = fmt.Sprintf("Current Model: %s (Provider: %s)",
			c.config.Agents.Defaults.GetModelName(),
			c.config.Agents.Defaults.Provider)
	case "channel":
		response = "Current Channel: telegram"
	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'model' or 'channel'.", args)
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   response,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) List(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /list [models|channels]",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	var response string
	switch args {
	case "models":
		currentModel := c.config.Agents.Defaults.GetModelName()
		var modelNames []string
		seen := make(map[string]bool)
		for _, m := range c.config.ModelList {
			if !seen[m.ModelName] {
				seen[m.ModelName] = true
				marker := ""
				if m.ModelName == currentModel {
					marker = " ✓"
				}
				modelNames = append(modelNames, fmt.Sprintf("• %s%s", m.ModelName, marker))
			}
		}
		if len(modelNames) == 0 {
			response = "No models configured in model_list"
		} else {
			response = fmt.Sprintf("Available Models:\n%s\n\nCurrent: %s", strings.Join(modelNames, "\n"), currentModel)
		}

	case "channels":
		var enabled []string
		if c.config.Channels.Telegram.Enabled {
			enabled = append(enabled, "telegram")
		}
		if c.config.Channels.WhatsApp.Enabled {
			enabled = append(enabled, "whatsapp")
		}
		if c.config.Channels.Feishu.Enabled {
			enabled = append(enabled, "feishu")
		}
		if c.config.Channels.Discord.Enabled {
			enabled = append(enabled, "discord")
		}
		if c.config.Channels.Slack.Enabled {
			enabled = append(enabled, "slack")
		}
		response = fmt.Sprintf("Enabled Channels:\n- %s", strings.Join(enabled, "\n- "))

	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'models' or 'channels'.", args)
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   response,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}

func (c *cmd) Switch(ctx context.Context, message telego.Message) error {
	args := commandArgs(message.Text)
	if args == "" {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Usage: /switch <model_name>\nUse /list models to see available models.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	modelName := strings.TrimSpace(args)

	// Check if model exists in model_list
	found := false
	for _, m := range c.config.ModelList {
		if m.ModelName == modelName {
			found = true
			break
		}
	}

	if !found {
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("Model '%s' not found. Use /list models to see available models.", modelName),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	// Update config
	c.config.Agents.Defaults.ModelName = modelName
	c.config.Agents.Defaults.Model = "" // Clear deprecated field

	// Save config
	if err := config.SaveConfig(c.configPath, c.config); err != nil {
		_, sendErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("Failed to save config: %v", err),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return sendErr
	}

	// Hot-reload the model if switcher is available
	if c.modelSwitcher != nil {
		if err := c.modelSwitcher.SwitchModel(modelName); err != nil {
			_, sendErr := c.bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text:   fmt.Sprintf("Config saved but hot-reload failed: %v\nRestart gateway to apply changes.", err),
				ReplyParameters: &telego.ReplyParameters{
					MessageID: message.MessageID,
				},
			})
			return sendErr
		}
		_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   fmt.Sprintf("✓ Switched to model: %s (hot-reloaded)", modelName),
			ReplyParameters: &telego.ReplyParameters{
				MessageID: message.MessageID,
			},
		})
		return err
	}

	_, err := c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   fmt.Sprintf("✓ Switched to model: %s\nRestart gateway to apply changes.", modelName),
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	return err
}
