// Package tgwrapper provides a high-level wrapper for Telegram Bot based on telego.
//
// Features:
//   - Configuration-driven: Define menus, conversation flows, and buttons through struct configuration
//   - Edit-first messaging: Edit existing messages on callback handling to avoid keyboard accumulation
//   - Group topic support: All message operations support MessageThreadID for group topics
//   - Highly extensible: Support for custom handlers, validators, and middleware
//
// Basic usage:
//
//	cfg := config.NewConfig()
//	// ... configure menus and flows ...
//	wrapper, _ := tgwrapper.New(cfg)
//	wrapper.Start(ctx)
package tgwrapper

import (
	"context"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/0xVanfer/tg-listener/config"
	"github.com/0xVanfer/tg-listener/conv"
	"github.com/0xVanfer/tg-listener/core"
	"github.com/0xVanfer/tg-listener/handler"
	"github.com/0xVanfer/tg-listener/menu"
)

// Re-export commonly used types and functions for convenience.
type (
	// Builder is a message builder for constructing formatted Telegram messages with entities.
	Builder = core.Builder
	// KeyboardBuilder is a builder for creating inline keyboard markup.
	KeyboardBuilder = core.KeyboardBuilder
	// Conversation represents a conversation session with a user.
	Conversation = conv.Conversation
	// Config is the main configuration structure for the wrapper.
	Config = config.Config
	// ButtonData represents dynamic button data for keyboard generation.
	ButtonData = config.ButtonData
	// HandlerRegistry holds all handler functions that can be referenced by configuration.
	HandlerRegistry = config.HandlerRegistry
	// CommandHandlerFunc is the function signature for command handlers.
	CommandHandlerFunc = config.CommandHandlerFunc
	// CallbackHandlerFunc is the function signature for callback handlers.
	CallbackHandlerFunc = config.CallbackHandlerFunc
	// StepHandlerFunc is the function signature for step completion handlers.
	StepHandlerFunc = config.StepHandlerFunc
	// KeyboardProviderFunc is the function signature for dynamic keyboard providers.
	KeyboardProviderFunc = config.KeyboardProviderFunc
	// ValidatorFunc is the function signature for custom validators.
	ValidatorFunc = config.ValidatorFunc
)

// Re-export commonly used callback constants for handling user interactions.
const (
	// CallbackMainMenu is the callback data for returning to main menu.
	CallbackMainMenu = core.CallbackMainMenu
	// CallbackBack is the callback data for going back to the previous step.
	CallbackBack = core.CallbackBack
	// CallbackCancel is the callback data for canceling the current operation (same as back).
	CallbackCancel = core.CallbackCancel
	// CallbackPage is the prefix for pagination callback data.
	CallbackPage = core.CallbackPage
)

// Re-export commonly used functions for building messages and keyboards.
var (
	// NewBuilder creates a new message builder instance.
	NewBuilder = core.NewBuilder
	// NewKeyboard creates a new keyboard builder instance.
	NewKeyboard = core.NewKeyboard
	// Button creates an inline keyboard button with callback data.
	Button = core.Button
	// URLButton creates an inline keyboard button with a URL.
	URLButton = core.URLButton
	// ParseCallbackData extracts the data portion from callback data by removing the prefix.
	ParseCallbackData = core.ParseCallbackData
	// GetTopicID extracts the message thread ID from a message for group topic support.
	GetTopicID = core.GetTopicID
	// NewHandlerRegistry creates a new empty handler registry.
	NewHandlerRegistry = config.NewHandlerRegistry
)

// Wrapper is the main entry point of tgwrapper library.
// It orchestrates all components including bot, router, menu manager, and conversation engine.
// Use New() to create a new instance and Start() to begin processing updates.
type Wrapper struct {
	bot         *core.Bot        // Core bot instance for Telegram API operations
	config      *config.Config   // Configuration containing menus, flows, and bot settings
	router      *handler.Router  // Router for dispatching commands, callbacks, and messages
	menuManager *menu.Manager    // Manager for menu display and navigation
	convManager *conv.Manager    // Manager for conversation state and lifecycle
	flowEngine  *conv.FlowEngine // Engine for processing conversation flows and steps

	botHandler *th.BotHandler // Telego handler for update processing
	stopChan   chan struct{}  // Channel for signaling graceful shutdown
}

// New creates a new Wrapper instance with the provided configuration.
// It initializes all internal components including bot, router, conversation manager,
// flow engine, and menu manager.
//
// Parameters:
//   - cfg: The configuration containing bot token, menus, flows, and other settings
//
// Returns:
//   - *Wrapper: The initialized wrapper instance
//   - error: Error if configuration is invalid or bot creation fails
//
// Example:
//
//	cfg := config.NewConfig()
//	cfg.Bot = &config.BotConfig{Token: "your-bot-token"}
//	wrapper, err := tgwrapper.New(cfg)
func New(cfg *config.Config) (*Wrapper, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if cfg.Bot == nil || cfg.Bot.Token == "" {
		return nil, fmt.Errorf("bot token is not configured")
	}

	// Create the core bot instance
	bot, err := core.NewBot(cfg.Bot.Token)
	if err != nil {
		return nil, err
	}

	// Get TTL from configuration or use default
	ttl := 30 * time.Minute
	if cfg.Bot != nil && cfg.Bot.DefaultTTL > 0 {
		ttl = cfg.Bot.DefaultTTL
	}

	// Create conversation manager with configured TTL
	convManager := conv.NewManager(ttl)

	// Create flow engine for processing conversation flows
	flowEngine := conv.NewFlowEngine(cfg)

	// Create router for dispatching updates
	router := handler.NewRouter(bot, cfg, convManager, flowEngine)

	// Set debug mode from configuration
	if cfg.Bot != nil && cfg.Bot.Debug {
		router.SetDebug(true)
	}

	// Create menu manager for menu display
	menuManager := menu.NewManager(bot, cfg)

	w := &Wrapper{
		bot:         bot,
		config:      cfg,
		router:      router,
		menuManager: menuManager,
		convManager: convManager,
		flowEngine:  flowEngine,
		stopChan:    make(chan struct{}),
	}

	// Register internal callback handlers for built-in functionality
	w.setupInternalHandlers()

	// Set up step display function for router
	w.router.SetStepDisplayFunc(w.showStepPrompt)

	return w, nil
}

// NewWithHandlers creates a new Wrapper instance with the provided configuration and handler registry.
// This is the recommended way to create a Wrapper as it enables configuration-driven handler registration.
// All handlers defined in the registry are automatically registered and linked to the configuration.
//
// Parameters:
//   - cfg: The configuration containing bot token, menus, flows, and other settings
//   - registry: The handler registry containing all handler implementations
//
// Returns:
//   - *Wrapper: The initialized wrapper instance with all handlers registered
//   - error: Error if configuration is invalid or bot creation fails
//
// Example:
//
//	cfg, _ := config.LoadFromFile("config.yaml")
//	registry := tgwrapper.NewHandlerRegistry()
//	registry.RegisterCommand("menu", menuHandler)
//	registry.RegisterStepHandler("handleInput", inputHandler)
//	wrapper, err := tgwrapper.NewWithHandlers(cfg, registry)
func NewWithHandlers(cfg *config.Config, registry *config.HandlerRegistry) (*Wrapper, error) {
	w, err := New(cfg)
	if err != nil {
		return nil, err
	}

	// Apply handler registry
	w.applyHandlerRegistry(registry)

	// Register handlers from configuration
	w.registerConfiguredHandlers(registry)

	return w, nil
}

// applyHandlerRegistry applies all handlers from the registry to the wrapper.
func (w *Wrapper) applyHandlerRegistry(registry *config.HandlerRegistry) {
	if registry == nil {
		return
	}

	// Set authentication function
	if registry.AuthFunc != nil {
		w.bot.SetAuthFunc(func(ctx context.Context, userID int64, username string) bool {
			return registry.AuthFunc(ctx, userID, username)
		})
	}

	// Register step handlers with type conversion
	for name, handler := range registry.StepHandlers {
		h := handler // capture loop variable
		w.flowEngine.RegisterStepHandler(name, func(ctx context.Context, c *conv.Conversation) error {
			return h(ctx, c)
		})
	}

	// Register keyboard providers with type conversion
	for name, provider := range registry.KeyboardProviders {
		p := provider // capture loop variable
		w.flowEngine.RegisterKeyboardProvider(name, func(ctx context.Context, c *conv.Conversation) []config.ButtonData {
			return p(ctx, c)
		})
	}

	// Register validators with type conversion
	for name, validator := range registry.Validators {
		v := validator // capture loop variable
		w.flowEngine.RegisterValidator(name, func(value string, c *conv.Conversation) error {
			return v(value, c)
		})
	}

	// Set conversation lifecycle hooks
	if registry.OnConversationStart != nil {
		fn := registry.OnConversationStart
		w.convManager.SetOnStart(func(ctx context.Context, c *conv.Conversation) {
			fn(ctx, c)
		})
	}

	if registry.OnConversationEnd != nil {
		fn := registry.OnConversationEnd
		w.convManager.SetOnEnd(func(ctx context.Context, c *conv.Conversation) {
			fn(ctx, c)
		})
	}

	if registry.OnStepChange != nil {
		fn := registry.OnStepChange
		w.convManager.SetOnStepChange(func(ctx context.Context, c *conv.Conversation, from, to string) {
			fn(ctx, c, from, to)
		})
	}
}

// registerConfiguredHandlers registers handlers based on configuration.
// This connects command and callback configurations to their handler implementations.
func (w *Wrapper) registerConfiguredHandlers(registry *config.HandlerRegistry) {
	if w.config.Bot == nil {
		return
	}

	// Register command handlers from configuration
	for _, cmd := range w.config.Bot.Commands {
		cmdCfg := cmd // capture loop variable

		// If handler is specified, look it up in registry
		if cmdCfg.Handler != "" && registry != nil {
			if handler, ok := registry.CommandHandlers[cmdCfg.Handler]; ok {
				w.router.RegisterCommand(cmdCfg.Command, func(ctx context.Context, msg telego.Message) error {
					return handler(ctx, msg)
				})
				continue
			}
		}

		// Handle built-in actions
		switch cmdCfg.Action {
		case "show_menu":
			target := cmdCfg.Target
			w.router.RegisterCommand(cmdCfg.Command, func(ctx context.Context, msg telego.Message) error {
				return w.ShowMainMenu(ctx, msg.Chat.ID, msg.MessageThreadID, 0)
			})
			if target != "" && target != "main" {
				targetID := target
				w.router.RegisterCommand(cmdCfg.Command, func(ctx context.Context, msg telego.Message) error {
					return w.ShowMenu(ctx, msg.Chat.ID, msg.MessageThreadID, targetID, 0)
				})
			}
		case "start_flow":
			if cmdCfg.Target != "" {
				flowID := cmdCfg.Target
				w.router.RegisterCommand(cmdCfg.Command, func(ctx context.Context, msg telego.Message) error {
					_, err := w.StartConversation(ctx, msg.From.ID, msg.Chat.ID, msg.MessageThreadID, flowID, 0)
					if err != nil {
						return w.ShowMainMenu(ctx, msg.Chat.ID, msg.MessageThreadID, 0)
					}
					c := w.convManager.Get(msg.From.ID, msg.Chat.ID)
					if c != nil {
						return w.showStepPrompt(ctx, c)
					}
					return nil
				})
			}
		}
	}

	// Register callback handlers from configuration
	for _, cb := range w.config.Callbacks {
		cbCfg := cb // capture loop variable

		// If handler is specified, look it up in registry
		if cbCfg.Handler != "" && registry != nil {
			if handler, ok := registry.CallbackHandlers[cbCfg.Handler]; ok {
				if cbCfg.IsPrefix {
					w.router.RegisterCallbackPrefix(cbCfg.Callback, func(ctx context.Context, query telego.CallbackQuery) error {
						return handler(ctx, query)
					})
				} else {
					w.router.RegisterCallback(cbCfg.Callback, func(ctx context.Context, query telego.CallbackQuery) error {
						return handler(ctx, query)
					})
				}
				continue
			}
		}

		// Handle built-in actions
		switch cbCfg.Action {
		case "show_menu":
			target := cbCfg.Target
			answerText := cbCfg.AnswerText
			if cbCfg.IsPrefix {
				w.router.RegisterCallbackPrefix(cbCfg.Callback, func(ctx context.Context, query telego.CallbackQuery) error {
					_ = w.bot.AnswerCallback(ctx, query.ID, answerText)
					chatID := query.Message.GetChat().ID
					msgID := query.Message.GetMessageID()
					if target == "" || target == "main" {
						_, err := w.menuManager.EditToMainMenu(ctx, chatID, msgID, nil)
						return err
					}
					_, err := w.menuManager.EditToMenu(ctx, chatID, msgID, target, nil)
					return err
				})
			} else {
				w.router.RegisterCallback(cbCfg.Callback, func(ctx context.Context, query telego.CallbackQuery) error {
					_ = w.bot.AnswerCallback(ctx, query.ID, answerText)
					chatID := query.Message.GetChat().ID
					msgID := query.Message.GetMessageID()
					if target == "" || target == "main" {
						_, err := w.menuManager.EditToMainMenu(ctx, chatID, msgID, nil)
						return err
					}
					_, err := w.menuManager.EditToMenu(ctx, chatID, msgID, target, nil)
					return err
				})
			}
		case "start_flow":
			if cbCfg.Target != "" {
				flowID := cbCfg.Target
				answerText := cbCfg.AnswerText
				w.router.RegisterCallback(cbCfg.Callback, func(ctx context.Context, query telego.CallbackQuery) error {
					_ = w.bot.AnswerCallback(ctx, query.ID, answerText)
					chatID := query.Message.GetChat().ID
					topicID := core.GetTopicID(query.Message)
					msgID := query.Message.GetMessageID()
					_, err := w.StartConversation(ctx, query.From.ID, chatID, topicID, flowID, msgID)
					if err != nil {
						return w.ShowMainMenu(ctx, chatID, topicID, msgID)
					}
					c := w.convManager.Get(query.From.ID, chatID)
					if c != nil {
						return w.showStepPrompt(ctx, c)
					}
					return nil
				})
			}
		case "answer":
			answerText := cbCfg.AnswerText
			w.router.RegisterCallback(cbCfg.Callback, func(ctx context.Context, query telego.CallbackQuery) error {
				return w.bot.AnswerCallback(ctx, query.ID, answerText)
			})
		}
	}
}

// setupInternalHandlers registers internal handlers for built-in callbacks.
// This includes main menu navigation, menu jumping, flow starting, and step display.
func (w *Wrapper) setupInternalHandlers() {
	// Main menu handler - returns user to the main menu
	w.router.RegisterCallback(core.CallbackMainMenu+"_internal", func(ctx context.Context, query telego.CallbackQuery) error {
		chatID := query.Message.GetChat().ID
		msgID := query.Message.GetMessageID()
		_, err := w.menuManager.EditToMainMenu(ctx, chatID, msgID, nil)
		return err
	})

	// Menu navigation handler - jumps to a specific menu by ID
	w.router.RegisterCallbackPrefix("menu:", func(ctx context.Context, query telego.CallbackQuery) error {
		_ = w.bot.AnswerCallback(ctx, query.ID, "")
		menuID := core.ParseCallbackData(query.Data, "menu:")
		chatID := query.Message.GetChat().ID
		msgID := query.Message.GetMessageID()
		_, err := w.menuManager.EditToMenu(ctx, chatID, msgID, menuID, nil)
		return err
	})

	// Flow start handler - initiates a conversation flow
	w.router.RegisterCallbackPrefix("flow:", func(ctx context.Context, query telego.CallbackQuery) error {
		_ = w.bot.AnswerCallback(ctx, query.ID, "")
		flowID := core.ParseCallbackData(query.Data, "flow:")
		chatID := query.Message.GetChat().ID
		topicID := core.GetTopicID(query.Message)
		msgID := query.Message.GetMessageID()

		_, err := w.StartConversation(ctx, query.From.ID, chatID, topicID, flowID, msgID)
		if err != nil {
			return w.ShowMainMenu(ctx, chatID, topicID, msgID)
		}

		// Display the initial step prompt
		c := w.convManager.Get(query.From.ID, chatID)
		if c != nil {
			return w.showStepPrompt(ctx, c)
		}
		return nil
	})
}

// Start initializes and starts the Wrapper to begin processing Telegram updates.
// It registers bot commands (if configured), sets up long polling, and starts
// the handler and cleanup tasks.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - error: Error if starting long polling or handler creation fails
//
// The method performs the following operations:
// 1. Registers bot commands with Telegram (if RegisterCommands is true)
// 2. Starts long polling for updates
// 3. Creates and configures the bot handler
// 4. Starts periodic cleanup of expired conversations
func (w *Wrapper) Start(ctx context.Context) error {
	// Determine whether to register commands based on configuration
	shouldRegister := true
	if w.config.Bot != nil && w.config.Bot.RegisterCommands != nil {
		shouldRegister = *w.config.Bot.RegisterCommands
	}

	// Register bot commands with Telegram
	if shouldRegister && w.config.Bot != nil && len(w.config.Bot.Commands) > 0 {
		commands := make([]telego.BotCommand, len(w.config.Bot.Commands))
		for i, cmd := range w.config.Bot.Commands {
			commands[i] = telego.BotCommand{
				Command:     cmd.Command,
				Description: cmd.Description,
			}
		}
		_ = w.bot.SetMyCommands(ctx, commands)
	}

	// Start long polling to receive updates from Telegram
	updates, err := w.bot.Telego().UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start long polling: %w", err)
	}

	// Create the bot handler for processing updates
	w.botHandler, err = th.NewBotHandler(w.bot.Telego(), updates)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	// Configure routing for the handler
	w.router.SetupHandler(w.botHandler)

	// Start periodic cleanup task for expired conversations
	w.convManager.StartCleanupTask(ctx, 5*time.Minute)

	// Start processing updates in a goroutine
	go w.botHandler.Start()

	return nil
}

// Stop gracefully stops the Wrapper and releases all resources.
// It stops the bot handler and signals shutdown via the stop channel.
// Note: Long polling is stopped by canceling the context passed to Start().
func (w *Wrapper) Stop() {
	if w.botHandler != nil {
		_ = w.botHandler.Stop()
	}
	close(w.stopChan)
	if w.config.Bot.DeleteCommandsOnExit {
		_ = w.Bot().Telego().DeleteMyCommands(context.Background(), nil)
	}
}

// Bot returns the underlying core.Bot instance for direct Telegram API access.
func (w *Wrapper) Bot() *core.Bot {
	return w.bot
}

// Config returns the current configuration.
func (w *Wrapper) Config() *config.Config {
	return w.config
}

// Router returns the message router for registering custom handlers.
func (w *Wrapper) Router() *handler.Router {
	return w.router
}

// SetAuthFunc sets the authentication function for user authorization.
// The auth function is called before processing any command, callback, or message.
//
// Parameters:
//   - fn: Authentication function that returns true if user is authorized
func (w *Wrapper) SetAuthFunc(fn core.AuthFunc) {
	w.bot.SetAuthFunc(fn)
}

// RegisterCommand registers a handler for a specific bot command.
//
// Parameters:
//   - command: The command name without the leading slash (e.g., "menu" not "/menu")
//   - h: The handler function to execute when the command is received
func (w *Wrapper) RegisterCommand(command string, h handler.CommandHandler) {
	w.router.RegisterCommand(command, h)
}

// RegisterCallback registers a callback handler for a specific callback data prefix.
//
// Parameters:
//   - callback: The callback data prefix to match
//   - h: The handler function to execute when matching callback is received
func (w *Wrapper) RegisterCallback(callback string, h handler.CallbackHandler) {
	w.router.RegisterCallbackPrefix(callback, h)
}

// RegisterStepHandler registers a handler function for conversation step completion.
// Step handlers are called when the OnComplete field of a step configuration is set.
//
// Parameters:
//   - name: The handler name (referenced in flow configuration)
//   - h: The handler function to execute
func (w *Wrapper) RegisterStepHandler(name string, h conv.StepHandler) {
	w.flowEngine.RegisterStepHandler(name, h)
}

// RegisterKeyboardProvider registers a dynamic keyboard data provider.
// Providers are called when a step's keyboard has type "dynamic" and provider is set.
//
// Parameters:
//   - name: The provider name (referenced in keyboard configuration)
//   - provider: Function that returns button data for the keyboard
func (w *Wrapper) RegisterKeyboardProvider(name string, provider conv.KeyboardProvider) {
	w.flowEngine.RegisterKeyboardProvider(name, provider)
}

// RegisterValidator registers a custom input validator.
// Validators are called when a step's validation type is "custom".
//
// Parameters:
//   - name: The validator name (referenced in validation configuration)
//   - validator: Function that validates input and returns error if invalid
func (w *Wrapper) RegisterValidator(name string, validator conv.Validator) {
	w.flowEngine.RegisterValidator(name, validator)
}

// Use adds a middleware to the router's middleware chain.
// Middleware are executed in the order they are added.
func (w *Wrapper) Use(middleware handler.Middleware) {
	w.router.Use(middleware)
}

// OnConversationStart sets a callback function that is called when a conversation starts.
func (w *Wrapper) OnConversationStart(fn func(ctx context.Context, c *conv.Conversation)) {
	w.convManager.SetOnStart(fn)
}

// OnConversationEnd sets a callback function that is called when a conversation ends.
func (w *Wrapper) OnConversationEnd(fn func(ctx context.Context, c *conv.Conversation)) {
	w.convManager.SetOnEnd(fn)
}

// OnStepChange sets a callback function that is called when a conversation step changes.
func (w *Wrapper) OnStepChange(fn func(ctx context.Context, c *conv.Conversation, from, to string)) {
	w.convManager.SetOnStepChange(fn)
}

// StartConversation initiates a new conversation flow for a user.
// If the user already has an active conversation, it will be ended first.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: The Telegram user ID
//   - chatID: The chat ID where the conversation takes place
//   - topicID: The message thread ID (for group topics, 0 if not applicable)
//   - flowID: The ID of the flow to start (must be defined in configuration)
//   - keyboardMsgID: The message ID of the keyboard to edit (0 to send new message)
//
// Returns:
//   - *conv.Conversation: The started conversation instance
//   - error: Error if the flow doesn't exist
func (w *Wrapper) StartConversation(ctx context.Context, userID, chatID int64, topicID int, flowID string, keyboardMsgID int) (*conv.Conversation, error) {
	flow := w.config.GetFlow(flowID)
	if flow == nil {
		return nil, fmt.Errorf("flow %s does not exist", flowID)
	}

	c, err := w.convManager.Start(ctx, userID, chatID, topicID, flowID, flow.InitialStep, flow.TTL)
	if err != nil {
		return nil, err
	}

	if keyboardMsgID > 0 {
		c.SetKeyboardMsgID(keyboardMsgID)
	}

	return c, nil
}

// GetConversation retrieves an active conversation for a user in a specific chat.
// Returns nil if no active conversation exists or if the conversation has expired.
func (w *Wrapper) GetConversation(userID, chatID int64) *conv.Conversation {
	return w.convManager.Get(userID, chatID)
}

// EndConversation terminates an active conversation for a user.
// This will trigger the OnEnd callback if configured.
func (w *Wrapper) EndConversation(ctx context.Context, userID, chatID int64) {
	w.convManager.End(ctx, userID, chatID)
}

// ShowMainMenu displays the main menu to the user.
// If editMsgID is provided (> 0), the existing message is edited; otherwise, a new message is sent.
func (w *Wrapper) ShowMainMenu(ctx context.Context, chatID int64, topicID int, editMsgID int) error {
	if editMsgID > 0 {
		_, err := w.menuManager.EditToMainMenu(ctx, chatID, editMsgID, nil)
		return err
	}
	_, err := w.menuManager.ShowMainMenu(ctx, chatID, topicID, nil)
	return err
}

// ShowMenu displays a specific menu to the user.
// If editMsgID is provided (> 0), the existing message is edited; otherwise, a new message is sent.
func (w *Wrapper) ShowMenu(ctx context.Context, chatID int64, topicID int, menuID string, editMsgID int) error {
	if editMsgID > 0 {
		_, err := w.menuManager.EditToMenu(ctx, chatID, editMsgID, menuID, nil)
		return err
	}
	_, err := w.menuManager.ShowMenu(ctx, chatID, topicID, menuID, nil)
	return err
}

// SendTo sends a text message to a specific chat.
// Supports MessageThreadID for group topics and message entities for formatting.
func (w *Wrapper) SendTo(ctx context.Context, chatID int64, topicID int, text string, entities ...telego.MessageEntity) (*telego.Message, error) {
	return w.bot.SendMessage(ctx, chatID, topicID, text, entities...)
}

// SendToWithKeyboard sends a message with an inline keyboard to a specific chat.
func (w *Wrapper) SendToWithKeyboard(ctx context.Context, chatID int64, topicID int, text string, keyboard *telego.InlineKeyboardMarkup, entities ...telego.MessageEntity) (*telego.Message, error) {
	return w.bot.SendMessageWithKeyboard(ctx, chatID, topicID, text, keyboard, entities...)
}

// EditMessage edits the text of an existing message.
func (w *Wrapper) EditMessage(ctx context.Context, chatID int64, messageID int, text string, entities ...telego.MessageEntity) (*telego.Message, error) {
	return w.bot.EditMessage(ctx, chatID, messageID, text, entities...)
}

// EditMessageKeyboard edits both the text and keyboard of an existing message.
func (w *Wrapper) EditMessageKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard *telego.InlineKeyboardMarkup, entities ...telego.MessageEntity) (*telego.Message, error) {
	return w.bot.EditMessageWithKeyboard(ctx, chatID, messageID, text, keyboard, entities...)
}

// DeleteMessage deletes a message from a chat.
func (w *Wrapper) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	return w.bot.DeleteMessage(ctx, chatID, messageID)
}

// AnswerCallback responds to a callback query.
// This should be called to acknowledge the callback and optionally show a notification.
func (w *Wrapper) AnswerCallback(ctx context.Context, callbackID string, text string) error {
	return w.bot.AnswerCallback(ctx, callbackID, text)
}

// showStepPrompt displays the prompt for the current conversation step.
// It builds the keyboard (static and/or dynamic) and either edits the existing
// keyboard message or sends a new one.
//
// This is an internal method called when:
// - A new conversation flow starts
// - The conversation advances to a new step
// - User navigates back to a previous step
func (w *Wrapper) showStepPrompt(ctx context.Context, c *conv.Conversation) error {
	flow := w.config.GetFlow(c.FlowID)
	if flow == nil {
		return nil
	}

	step := flow.GetStep(c.StepID)
	if step == nil {
		return nil
	}

	// Build the keyboard based on step configuration
	var kb *telego.InlineKeyboardMarkup
	if step.Keyboard != nil {
		kbCfg := step.Keyboard

		// Fetch dynamic button data if required
		var dynamicButtons []config.ButtonData
		if kbCfg.NeedsDynamicData() && kbCfg.Provider != "" {
			provider := w.flowEngine.GetKeyboardProvider(kbCfg.Provider)
			if provider != nil {
				dynamicButtons = provider(ctx, c)
			}
		}

		// Build the keyboard using the keyboard builder
		kbBuilder := core.NewKeyboard()

		// Add static buttons from configuration
		for _, row := range kbCfg.Buttons {
			var buttons []telego.InlineKeyboardButton
			for _, btn := range row {
				if btn.URL != "" {
					buttons = append(buttons, core.URLButton(btn.Text, btn.URL))
				} else {
					buttons = append(buttons, core.Button(btn.Text, btn.Callback))
				}
			}
			if len(buttons) > 0 {
				kbBuilder.Row(buttons...)
			}
		}

		// Add dynamic buttons in a grid layout
		if len(dynamicButtons) > 0 {
			var buttons []telego.InlineKeyboardButton
			for _, btn := range dynamicButtons {
				callback := btn.Callback
				if kbCfg.CallbackPrefix != "" {
					callback = kbCfg.CallbackPrefix + callback
				}
				buttons = append(buttons, core.Button(btn.Text, callback))
			}
			kbBuilder.Grid(buttons, kbCfg.GetColumns())
		}

		// Add navigation buttons (back/main menu)
		if kbCfg.AddBack {
			kbBuilder.Back(kbCfg.GetBackText())
		}
		if kbCfg.AddMain {
			kbBuilder.MainMenu(kbCfg.GetMainText())
		}

		kb = kbBuilder.Build()
	}

	// Edit existing keyboard message or send new one
	if c.KeyboardMsgID > 0 {
		_, err := w.bot.EditMessageWithKeyboard(ctx, c.ChatID, c.KeyboardMsgID, step.PromptText, kb)
		return err
	}

	// Send a new message with the step prompt
	msg, err := w.bot.SendMessageWithKeyboard(ctx, c.ChatID, c.TopicID, step.PromptText, kb)
	if err != nil {
		return err
	}
	if msg != nil {
		c.SetKeyboardMsgID(msg.MessageID)
	}
	return nil
}
