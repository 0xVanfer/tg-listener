// Package config defines configuration structures for tgwrapper.
// This package provides all the configuration types needed to define
// bot settings, menus, conversation flows, keyboards, and buttons.
package config

import "time"

// BotConfig defines the bot-level configuration settings.
// This includes authentication credentials, command registration,
// logging targets, and various behavioral options.
type BotConfig struct {
	// Token is the Telegram Bot API token obtained from @BotFather.
	// This is required for the bot to authenticate with Telegram servers.
	Token string `json:"token" yaml:"token" mapstructure:"token"`

	// Commands is the list of commands to register with Telegram.
	// These appear in the command menu when users type "/" in the chat.
	Commands []CmdConfig `json:"commands" yaml:"commands" mapstructure:"commands"`

	// WarningChat specifies the target chat for warning/alert messages.
	// Use this to send critical notifications to administrators.
	WarningChat *ChatConfig `json:"warning_chat" yaml:"warning_chat" mapstructure:"warning_chat"`

	// LogChat specifies the target chat for log messages.
	// Use this for general logging and debugging information.
	LogChat *ChatConfig `json:"log_chat" yaml:"log_chat" mapstructure:"log_chat"`

	// DefaultTTL is the default time-to-live for conversations.
	// Conversations that exceed this duration will be automatically cleaned up.
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl" mapstructure:"default_ttl"`

	// Debug enables debug mode for verbose logging.
	Debug bool `json:"debug" yaml:"debug" mapstructure:"debug"`

	// DeleteCommandsOnExit determines whether to delete all registered
	// commands when the bot stops. Useful for development/testing.
	DeleteCommandsOnExit bool `json:"delete_commands_on_exit" yaml:"delete_commands_on_exit" mapstructure:"delete_commands_on_exit"`

	// RegisterCommands determines whether to register commands on startup.
	// Defaults to true if nil. Set to false to skip command registration.
	RegisterCommands *bool `json:"register_commands" yaml:"register_commands" mapstructure:"register_commands"`
}

// ChatConfig defines a chat target for messages.
// Used for specifying where to send logs, warnings, or other notifications.
type ChatConfig struct {
	// ChatID is the Telegram chat identifier.
	// Can be a user ID, group ID, or channel ID.
	ChatID int64 `json:"chat_id" yaml:"chat_id" mapstructure:"chat_id"`

	// TopicID is the message thread ID for group topics (forum mode).
	// Set to 0 for regular chats without topics.
	TopicID int `json:"topic_id" yaml:"topic_id" mapstructure:"topic_id"`
}

// CmdConfig defines a single bot command configuration.
type CmdConfig struct {
	// Command is the command name without the leading slash.
	// For example, use "help" not "/help".
	Command string `json:"command" yaml:"command" mapstructure:"command"`

	// Description is shown in Telegram's command menu.
	// Keep it concise but descriptive.
	Description string `json:"description" yaml:"description" mapstructure:"description"`

	// Handler is the name of the handler function to call.
	// This is looked up in the registered handlers map.
	Handler string `json:"handler" yaml:"handler" mapstructure:"handler"`
}

// NewDefaultBotConfig creates a BotConfig with sensible default values.
// Returns a config with 10-minute default TTL and debug mode disabled.
func NewDefaultBotConfig() *BotConfig {
	return &BotConfig{
		DefaultTTL: 10 * time.Minute,
		Debug:      false,
	}
}

// Validate checks if the bot configuration is valid.
// Returns ErrEmptyToken if the token is not set.
func (c *BotConfig) Validate() error {
	if c.Token == "" {
		return ErrEmptyToken
	}
	return nil
}

// HasWarningChat returns true if a warning chat is configured.
func (c *BotConfig) HasWarningChat() bool {
	return c.WarningChat != nil && c.WarningChat.ChatID != 0
}

// HasLogChat returns true if a log chat is configured.
func (c *BotConfig) HasLogChat() bool {
	return c.LogChat != nil && c.LogChat.ChatID != 0
}
