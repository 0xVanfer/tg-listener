// Package core provides core functionality for Telegram Bot operations.
// This package wraps the telego library to provide a simplified and
// opinionated interface for common bot operations.
package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

// AuthFunc is the authentication function type.
// Returns true if the user is authorized, false otherwise.
// Used to filter messages and callbacks from unauthorized users.
type AuthFunc func(ctx context.Context, userID int64, username string) bool

// Chat represents a chat target for sending messages.
// Combines ChatID and TopicID for complete message targeting.
type Chat struct {
	ChatID  int64 // Telegram chat ID (user, group, or channel)
	TopicID int   // Message thread ID for group topics (forum mode)
}

// NewChat creates a new Chat instance with the specified IDs.
func NewChat(chatID int64, topicID int) Chat {
	return Chat{
		ChatID:  chatID,
		TopicID: topicID,
	}
}

// Bot wraps telego.Bot to provide high-level message operations.
// All methods are safe for concurrent use.
type Bot struct {
	bot      *telego.Bot  // Underlying telego bot instance
	authFunc AuthFunc     // Authentication function for user filtering
	mu       sync.RWMutex // Mutex for thread-safe auth function access
}

// NewBot creates a new Bot instance with the given token.
// Returns an error if the token is invalid or bot creation fails.
func NewBot(token string) (*Bot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &Bot{
		bot: bot,
	}, nil
}

// SetAuthFunc sets the authentication function for user authorization.
// The function is called before processing any user interaction.
func (b *Bot) SetAuthFunc(fn AuthFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.authFunc = fn
}

// GetAuthFunc returns the current authentication function.
func (b *Bot) GetAuthFunc() AuthFunc {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.authFunc
}

// CheckAuth verifies if a user is authorized.
// Returns true if no auth function is set (default allow).
func (b *Bot) CheckAuth(ctx context.Context, userID int64, username string) bool {
	fn := b.GetAuthFunc()
	if fn == nil {
		return true // No auth function means allow all
	}
	return fn(ctx, userID, username)
}

// Telego returns the underlying telego.Bot instance for direct API access.
func (b *Bot) Telego() *telego.Bot {
	return b.bot
}

// SendMessage sends a text message to the specified chat.
// Supports MessageThreadID for group topics and message entities for formatting.
// Link previews are disabled by default following Telegram best practices.
func (b *Bot) SendMessage(ctx context.Context, chatID int64, topicID int, text string, entities ...telego.MessageEntity) (*telego.Message, error) {
	if b.bot == nil {
		return nil, nil
	}

	params := &telego.SendMessageParams{
		ChatID:    telegoutil.ID(chatID),
		Text:      text,
		ParseMode: "", // Don't set ParseMode when using entities
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: true, // Disable link preview
		},
	}

	if topicID > 0 {
		params.MessageThreadID = topicID
	}

	if len(entities) > 0 {
		params.Entities = entities
	}

	return b.bot.SendMessage(ctx, params)
}

// SendMessageWithKeyboard sends a message with an inline keyboard.
// Similar to SendMessage but includes keyboard markup.
func (b *Bot) SendMessageWithKeyboard(ctx context.Context, chatID int64, topicID int, text string, keyboard *telego.InlineKeyboardMarkup, entities ...telego.MessageEntity) (*telego.Message, error) {
	if b.bot == nil {
		return nil, nil
	}

	params := &telego.SendMessageParams{
		ChatID:    telegoutil.ID(chatID),
		Text:      text,
		ParseMode: "", // Don't set ParseMode when using entities
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: true,
		},
	}

	if topicID > 0 {
		params.MessageThreadID = topicID
	}

	if len(entities) > 0 {
		params.Entities = entities
	}

	if keyboard != nil {
		params.ReplyMarkup = keyboard
	}

	return b.bot.SendMessage(ctx, params)
}

// EditMessage edits the text of an existing message.
// Link previews are disabled by default.
func (b *Bot) EditMessage(ctx context.Context, chatID int64, messageID int, text string, entities ...telego.MessageEntity) (*telego.Message, error) {
	if b.bot == nil {
		return nil, nil
	}

	params := &telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(chatID),
		MessageID: messageID,
		Text:      text,
		ParseMode: "",
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: true,
		},
	}

	if len(entities) > 0 {
		params.Entities = entities
	}

	return b.bot.EditMessageText(ctx, params)
}

// EditMessageWithKeyboard edits both text and keyboard of an existing message.
// This is the preferred method for callback handlers to update message content.
func (b *Bot) EditMessageWithKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard *telego.InlineKeyboardMarkup, entities ...telego.MessageEntity) (*telego.Message, error) {
	if b.bot == nil {
		return nil, nil
	}

	params := &telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(chatID),
		MessageID: messageID,
		Text:      text,
		ParseMode: "",
		LinkPreviewOptions: &telego.LinkPreviewOptions{
			IsDisabled: true,
		},
	}

	if len(entities) > 0 {
		params.Entities = entities
	}

	if keyboard != nil {
		params.ReplyMarkup = keyboard
	}

	return b.bot.EditMessageText(ctx, params)
}

// EditKeyboard edits only the keyboard of an existing message.
// Use this when the text content doesn't need to change.
func (b *Bot) EditKeyboard(ctx context.Context, chatID int64, messageID int, keyboard *telego.InlineKeyboardMarkup) (*telego.Message, error) {
	if b.bot == nil {
		return nil, nil
	}

	params := &telego.EditMessageReplyMarkupParams{
		ChatID:      telegoutil.ID(chatID),
		MessageID:   messageID,
		ReplyMarkup: keyboard,
	}

	return b.bot.EditMessageReplyMarkup(ctx, params)
}

// DeleteMessage deletes a message from the chat.
func (b *Bot) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	if b.bot == nil {
		return nil
	}

	return b.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    telegoutil.ID(chatID),
		MessageID: messageID,
	})
}

// AnswerCallback responds to a callback query.
// Must be called for every callback query to prevent loading indicators.
func (b *Bot) AnswerCallback(ctx context.Context, callbackID string, text string) error {
	if b.bot == nil {
		return nil
	}

	return b.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
}

// AnswerCallbackWithAlert responds to a callback query with an alert popup.
// The user must dismiss the alert before continuing.
func (b *Bot) AnswerCallbackWithAlert(ctx context.Context, callbackID string, text string) error {
	if b.bot == nil {
		return nil
	}

	return b.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       true,
	})
}

// SetMyCommands registers the bot's command list with Telegram.
// These commands appear in the command menu when users type "/".
func (b *Bot) SetMyCommands(ctx context.Context, commands []telego.BotCommand) error {
	if b.bot == nil {
		return nil
	}

	return b.bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: commands,
	})
}

// GetMe retrieves information about the bot itself.
func (b *Bot) GetMe(ctx context.Context) (*telego.User, error) {
	if b.bot == nil {
		return nil, nil
	}
	return b.bot.GetMe(ctx)
}

// SendTo sends a message to the specified Chat.
// Convenience method that accepts a Chat struct.
func (b *Bot) SendTo(ctx context.Context, chat Chat, text string, entities ...telego.MessageEntity) (*telego.Message, error) {
	return b.SendMessage(ctx, chat.ChatID, chat.TopicID, text, entities...)
}

// SendToWithKeyboard sends a message with keyboard to the specified Chat.
func (b *Bot) SendToWithKeyboard(ctx context.Context, chat Chat, text string, keyboard *telego.InlineKeyboardMarkup, entities ...telego.MessageEntity) (*telego.Message, error) {
	return b.SendMessageWithKeyboard(ctx, chat.ChatID, chat.TopicID, text, keyboard, entities...)
}

// GetTopicID extracts the message thread ID from a message.
// Returns 0 if the message doesn't have a thread ID or is inaccessible.
func GetTopicID(msg telego.MaybeInaccessibleMessage) int {
	if msg == nil {
		return 0
	}

	if accessible, ok := msg.(*telego.Message); ok {
		return accessible.MessageThreadID
	}

	return 0
}

// ParseCallbackData removes a prefix from callback data.
// Returns the original data if the prefix doesn't match.
func ParseCallbackData(data string, prefix string) string {
	if len(data) >= len(prefix) && data[:len(prefix)] == prefix {
		return data[len(prefix):]
	}
	return data
}
