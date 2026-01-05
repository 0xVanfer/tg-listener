// Package handler provides message routing and processing functionality.
// It handles routing of commands, callbacks, and messages to appropriate handlers,
// with support for middleware, conversation management, and flow processing.
package handler

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/0xVanfer/tg-listener/config"
	"github.com/0xVanfer/tg-listener/conv"
	"github.com/0xVanfer/tg-listener/core"
)

// CommandHandler is a function type for handling bot commands.
// It receives the context and the message containing the command.
type CommandHandler func(ctx context.Context, msg telego.Message) error

// CallbackHandler is a function type for handling callback queries.
// It receives the context and the callback query from inline keyboards.
type CallbackHandler func(ctx context.Context, query telego.CallbackQuery) error

// MessageHandler is a function type for handling text messages.
// It receives the context and the message.
type MessageHandler func(ctx context.Context, msg telego.Message) error

// PhotoHandler is a function type for handling photo messages.
type PhotoHandler func(ctx context.Context, msg telego.Message) error

// DocumentHandler is a function type for handling document messages.
type DocumentHandler func(ctx context.Context, msg telego.Message) error

// Middleware is a function type for request middleware.
// Middleware can intercept and modify request handling.
type Middleware func(next Handler) Handler

// Handler is a generic handler function type for processing updates.
type Handler func(ctx context.Context, update telego.Update) error

// StepDisplayFunc is a callback for displaying step prompts.
// Used internally to trigger step display from the wrapper.
type StepDisplayFunc func(ctx context.Context, c *conv.Conversation) error

// Router handles message routing and dispatching to appropriate handlers.
// It supports commands, callbacks, messages, middleware, and conversation flows.
type Router struct {
	bot         *core.Bot        // Bot instance for sending messages
	config      *config.Config   // Configuration
	convManager *conv.Manager    // Conversation manager
	flowEngine  *conv.FlowEngine // Flow engine for conversation flows

	commandHandlers  map[string]CommandHandler  // Command handlers by command name
	callbackHandlers map[string]CallbackHandler // Callback handlers by exact match
	prefixHandlers   map[string]CallbackHandler // Callback handlers by prefix match
	messageHandler   MessageHandler             // Default message handler
	photoHandler     PhotoHandler               // Photo message handler
	documentHandler  DocumentHandler            // Document message handler
	middlewares      []Middleware               // Middleware chain

	stepDisplayFunc StepDisplayFunc // Function to display step prompts
	debug           bool            // Enable debug logging

	mu sync.RWMutex // Mutex for thread-safe operations
}

// NewRouter creates a new message router with the given dependencies.
// Parameters:
//   - bot: Core bot instance for Telegram API operations
//   - cfg: Configuration containing menus and flows
//   - convManager: Conversation state manager
//   - flowEngine: Flow execution engine
func NewRouter(bot *core.Bot, cfg *config.Config, convManager *conv.Manager, flowEngine *conv.FlowEngine) *Router {
	return &Router{
		bot:              bot,
		config:           cfg,
		convManager:      convManager,
		flowEngine:       flowEngine,
		commandHandlers:  make(map[string]CommandHandler),
		callbackHandlers: make(map[string]CallbackHandler),
		prefixHandlers:   make(map[string]CallbackHandler),
	}
}

// SetConfig updates the router configuration.
func (r *Router) SetConfig(cfg *config.Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

// SetDebug enables or disables debug logging.
func (r *Router) SetDebug(debug bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.debug = debug
}

// SetStepDisplayFunc sets the function for displaying step prompts.
// This is called internally by the wrapper to enable step display.
func (r *Router) SetStepDisplayFunc(fn StepDisplayFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stepDisplayFunc = fn
}

// FlowEngine returns the flow engine instance.
func (r *Router) FlowEngine() *conv.FlowEngine {
	return r.flowEngine
}

// ConvManager returns the conversation manager instance.
func (r *Router) ConvManager() *conv.Manager {
	return r.convManager
}

// Use adds a middleware to the router's middleware chain.
// Middlewares are executed in the order they are added.
func (r *Router) Use(middleware Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, middleware)
}

// RegisterCommand registers a handler for a specific command.
// The command parameter can be with or without leading slash.
// Example: RegisterCommand("menu", handler) or RegisterCommand("/menu", handler)
func (r *Router) RegisterCommand(command string, handler CommandHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Remove leading slash if present
	command = strings.TrimPrefix(command, "/")
	r.commandHandlers[command] = handler
}

// RegisterCallback registers a callback handler for exact data match.
// Use this for specific callback data like "main_menu" or "confirm".
func (r *Router) RegisterCallback(callback string, handler CallbackHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbackHandlers[callback] = handler
}

// RegisterCallbackPrefix registers a callback handler for prefix match.
// Use this for callback data with dynamic parts like "user:" or "item:".
func (r *Router) RegisterCallbackPrefix(prefix string, handler CallbackHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prefixHandlers[prefix] = handler
}

// SetMessageHandler sets the default handler for text messages.
// This handler is called when no conversation is active and no other handler matches.
func (r *Router) SetMessageHandler(handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageHandler = handler
}

// SetPhotoHandler sets the handler for photo messages.
func (r *Router) SetPhotoHandler(handler PhotoHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.photoHandler = handler
}

// SetDocumentHandler sets the handler for document messages.
func (r *Router) SetDocumentHandler(handler DocumentHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.documentHandler = handler
}

// SetupHandler configures the telegohandler with routing rules.
// This method sets up all message, callback, and media handlers.
func (r *Router) SetupHandler(bh *th.BotHandler) {
	// Command handler - matches messages starting with /
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		r.handleCommand(ctx, message)
		return nil
	}, func(_ context.Context, update telego.Update) bool {
		return update.Message != nil &&
			len(update.Message.Text) > 0 &&
			update.Message.Text[0] == '/'
	})

	// Callback query handler
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		r.handleCallback(ctx, query)
		return nil
	})

	// Photo message handler
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		r.handlePhoto(ctx, message)
		return nil
	}, func(_ context.Context, update telego.Update) bool {
		return update.Message != nil && len(update.Message.Photo) > 0
	})

	// Document message handler
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		r.handleDocument(ctx, message)
		return nil
	}, func(_ context.Context, update telego.Update) bool {
		return update.Message != nil && update.Message.Document != nil
	})

	// Regular text message handler
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		r.handleMessage(ctx, message)
		return nil
	}, func(_ context.Context, update telego.Update) bool {
		return update.Message != nil &&
			len(update.Message.Text) > 0 &&
			update.Message.Text[0] != '/'
	})
}

// logDebug logs a debug message if debug mode is enabled.
func (r *Router) logDebug(format string, args ...interface{}) {
	r.mu.RLock()
	debug := r.debug
	r.mu.RUnlock()
	if debug {
		log.Printf("[Router] "+format, args...)
	}
}

// handleCommand processes incoming commands.
func (r *Router) handleCommand(ctx context.Context, msg telego.Message) {
	if msg.Text == "" || msg.Text[0] != '/' {
		return
	}

	// Parse command and arguments
	parts := strings.SplitN(msg.Text, " ", 2)
	command := strings.TrimPrefix(parts[0], "/")

	// Remove @botname suffix if present
	if idx := strings.Index(command, "@"); idx != -1 {
		command = command[:idx]
	}

	r.logDebug("Command received: /%s from user %d", command, msg.From.ID)

	// Authentication check
	if !r.bot.CheckAuth(ctx, msg.From.ID, msg.From.Username) {
		r.logDebug("User %d not authorized", msg.From.ID)
		return
	}

	// Look up handler
	r.mu.RLock()
	handler, ok := r.commandHandlers[command]
	r.mu.RUnlock()

	if ok {
		if err := handler(ctx, msg); err != nil {
			r.logDebug("Command handler error: %v", err)
		}
	} else {
		r.logDebug("No handler found for command: /%s", command)
	}
}

// handleCallback processes callback queries from inline keyboards.
func (r *Router) handleCallback(ctx context.Context, query telego.CallbackQuery) {
	// Authentication check
	if !r.bot.CheckAuth(ctx, query.From.ID, query.From.Username) {
		_ = r.bot.AnswerCallback(ctx, query.ID, "")
		return
	}

	data := query.Data
	r.logDebug("Callback received: %s from user %d", data, query.From.ID)

	// Handle built-in navigation callbacks
	switch data {
	case core.CallbackMainMenu:
		r.handleMainMenu(ctx, query)
		return
	case core.CallbackBack, core.CallbackCancel:
		r.handleBack(ctx, query)
		return
	}

	// Check for exact match handler
	r.mu.RLock()
	handler, ok := r.callbackHandlers[data]
	r.mu.RUnlock()

	if ok {
		if err := handler(ctx, query); err != nil {
			r.logDebug("Callback handler error: %v", err)
		}
		return
	}

	// Check for prefix match handler
	r.mu.RLock()
	for prefix, h := range r.prefixHandlers {
		if strings.HasPrefix(data, prefix) {
			handler = h
			break
		}
	}
	r.mu.RUnlock()

	if handler != nil {
		if err := handler(ctx, query); err != nil {
			r.logDebug("Prefix callback handler error: %v", err)
		}
		return
	}

	// Check if user is in a conversation
	chatID := query.Message.GetChat().ID
	c := r.convManager.Get(query.From.ID, chatID)
	if c != nil {
		r.handleConversationCallback(ctx, query, c)
		return
	}

	// No matching handler found - answer callback to prevent loading indicator
	_ = r.bot.AnswerCallback(ctx, query.ID, "")
}

// handleMessage processes regular text messages.
func (r *Router) handleMessage(ctx context.Context, msg telego.Message) {
	if msg.From == nil {
		return
	}

	// Authentication check
	if !r.bot.CheckAuth(ctx, msg.From.ID, msg.From.Username) {
		return
	}

	r.logDebug("Message received from user %d: %s", msg.From.ID, truncateString(msg.Text, 50))

	// Check if user is in a conversation
	c := r.convManager.Get(msg.From.ID, msg.Chat.ID)
	if c != nil {
		r.handleConversationMessage(ctx, msg, c)
		return
	}

	// Use default message handler
	r.mu.RLock()
	handler := r.messageHandler
	r.mu.RUnlock()

	if handler != nil {
		if err := handler(ctx, msg); err != nil {
			r.logDebug("Message handler error: %v", err)
		}
	}
}

// handlePhoto processes photo messages.
func (r *Router) handlePhoto(ctx context.Context, msg telego.Message) {
	if msg.From == nil {
		return
	}

	// Authentication check
	if !r.bot.CheckAuth(ctx, msg.From.ID, msg.From.Username) {
		return
	}

	r.logDebug("Photo received from user %d", msg.From.ID)

	// Check if user is in a conversation expecting photo input
	c := r.convManager.Get(msg.From.ID, msg.Chat.ID)
	if c != nil {
		step := r.flowEngine.GetStep(c.FlowID, c.StepID)
		if step != nil && step.InputType == config.InputTypePhoto {
			r.handleConversationPhoto(ctx, msg, c)
			return
		}
	}

	// Use photo handler
	r.mu.RLock()
	handler := r.photoHandler
	r.mu.RUnlock()

	if handler != nil {
		if err := handler(ctx, msg); err != nil {
			r.logDebug("Photo handler error: %v", err)
		}
	}
}

// handleDocument processes document messages.
func (r *Router) handleDocument(ctx context.Context, msg telego.Message) {
	if msg.From == nil {
		return
	}

	// Authentication check
	if !r.bot.CheckAuth(ctx, msg.From.ID, msg.From.Username) {
		return
	}

	r.logDebug("Document received from user %d: %s", msg.From.ID, msg.Document.FileName)

	// Check if user is in a conversation expecting document input
	c := r.convManager.Get(msg.From.ID, msg.Chat.ID)
	if c != nil {
		step := r.flowEngine.GetStep(c.FlowID, c.StepID)
		if step != nil && step.InputType == config.InputTypeDocument {
			r.handleConversationDocument(ctx, msg, c)
			return
		}
	}

	// Use document handler
	r.mu.RLock()
	handler := r.documentHandler
	r.mu.RUnlock()

	if handler != nil {
		if err := handler(ctx, msg); err != nil {
			r.logDebug("Document handler error: %v", err)
		}
	}
}

// handleMainMenu handles returning to the main menu.
func (r *Router) handleMainMenu(ctx context.Context, query telego.CallbackQuery) {
	_ = r.bot.AnswerCallback(ctx, query.ID, "")

	chatID := query.Message.GetChat().ID

	// End current conversation if any
	r.convManager.End(ctx, query.From.ID, chatID)

	// Trigger internal main menu handler
	r.mu.RLock()
	handler, ok := r.callbackHandlers[core.CallbackMainMenu+"_internal"]
	r.mu.RUnlock()

	if ok {
		_ = handler(ctx, query)
	}
}

// handleBack handles the back navigation.
func (r *Router) handleBack(ctx context.Context, query telego.CallbackQuery) {
	_ = r.bot.AnswerCallback(ctx, query.ID, "")

	chatID := query.Message.GetChat().ID
	c := r.convManager.Get(query.From.ID, chatID)

	if c != nil {
		// Get previous step from history
		prevStep := c.GetPreviousStep()
		if prevStep != "" {
			// Go back to previous step
			r.convManager.ChangeStep(ctx, query.From.ID, chatID, prevStep)
			r.displayStep(ctx, c)
		} else {
			// No previous step - end conversation and return to main menu
			r.convManager.End(ctx, query.From.ID, chatID)
			r.handleMainMenu(ctx, query)
		}
	} else {
		// Not in conversation - return to main menu
		r.handleMainMenu(ctx, query)
	}
}

// handleConversationCallback handles callback queries during a conversation.
func (r *Router) handleConversationCallback(ctx context.Context, query telego.CallbackQuery, c *conv.Conversation) {
	_ = r.bot.AnswerCallback(ctx, query.ID, "")

	step := r.flowEngine.GetStep(c.FlowID, c.StepID)
	if step == nil {
		return
	}

	// Verify step accepts callback input
	if step.InputType != config.InputTypeCallback && step.InputType != config.InputTypeAny {
		r.logDebug("Step %s does not accept callback input", c.StepID)
		return
	}

	// Store input data
	if step.StoreAs != "" {
		c.Set(step.StoreAs, query.Data)
	}
	c.AddHistory(c.StepID, query.Data)

	// Execute completion handler if specified
	if step.OnComplete != "" {
		if err := r.flowEngine.ExecuteStepHandler(ctx, c, step.OnComplete); err != nil {
			r.logDebug("Step handler error: %v", err)
		}
		return
	}

	// Determine and transition to next step
	nextStep := r.flowEngine.DetermineNextStep(ctx, c, query.Data)
	if nextStep != "" {
		r.convManager.ChangeStep(ctx, query.From.ID, c.ChatID, nextStep)
		r.displayStep(ctx, c)
	}
}

// handleConversationMessage handles text messages during a conversation.
func (r *Router) handleConversationMessage(ctx context.Context, msg telego.Message, c *conv.Conversation) {
	step := r.flowEngine.GetStep(c.FlowID, c.StepID)
	if step == nil {
		return
	}

	// Verify step accepts text input
	if step.InputType != config.InputTypeText && step.InputType != config.InputTypeAny {
		r.logDebug("Step %s does not accept text input", c.StepID)
		return
	}

	input := msg.Text

	// Validate input if validation is configured
	if err := r.flowEngine.ValidateInput(c, input); err != nil {
		_, _ = r.bot.SendMessage(ctx, msg.Chat.ID, msg.MessageThreadID, "❌ "+err.Error())
		return
	}

	// Store input data
	if step.StoreAs != "" {
		c.Set(step.StoreAs, input)
	}
	c.AddHistory(c.StepID, input)

	// Delete user message to keep chat clean (optional behavior)
	_ = r.bot.DeleteMessage(ctx, msg.Chat.ID, msg.MessageID)

	// Execute completion handler if specified
	if step.OnComplete != "" {
		if err := r.flowEngine.ExecuteStepHandler(ctx, c, step.OnComplete); err != nil {
			r.logDebug("Step handler error: %v", err)
		}
		return
	}

	// Determine and transition to next step
	nextStep := r.flowEngine.DetermineNextStep(ctx, c, input)
	if nextStep != "" {
		r.convManager.ChangeStep(ctx, msg.From.ID, c.ChatID, nextStep)
		r.displayStep(ctx, c)
	}
}

// handleConversationPhoto handles photo messages during a conversation.
func (r *Router) handleConversationPhoto(ctx context.Context, msg telego.Message, c *conv.Conversation) {
	step := r.flowEngine.GetStep(c.FlowID, c.StepID)
	if step == nil {
		return
	}

	// Get the largest photo (last in array)
	if len(msg.Photo) == 0 {
		return
	}
	photo := msg.Photo[len(msg.Photo)-1]

	// Store file ID
	if step.StoreAs != "" {
		c.Set(step.StoreAs, photo.FileID)
		c.Set(step.StoreAs+"_file_id", photo.FileID)
	}
	c.AddHistory(c.StepID, "photo:"+photo.FileID)

	// Execute completion handler if specified
	if step.OnComplete != "" {
		if err := r.flowEngine.ExecuteStepHandler(ctx, c, step.OnComplete); err != nil {
			r.logDebug("Step handler error: %v", err)
		}
		return
	}

	// Determine and transition to next step
	nextStep := r.flowEngine.DetermineNextStep(ctx, c, photo.FileID)
	if nextStep != "" {
		r.convManager.ChangeStep(ctx, msg.From.ID, c.ChatID, nextStep)
		r.displayStep(ctx, c)
	}
}

// handleConversationDocument handles document messages during a conversation.
func (r *Router) handleConversationDocument(ctx context.Context, msg telego.Message, c *conv.Conversation) {
	step := r.flowEngine.GetStep(c.FlowID, c.StepID)
	if step == nil || msg.Document == nil {
		return
	}

	// Store file info
	if step.StoreAs != "" {
		c.Set(step.StoreAs, msg.Document.FileID)
		c.Set(step.StoreAs+"_file_id", msg.Document.FileID)
		c.Set(step.StoreAs+"_file_name", msg.Document.FileName)
	}
	c.AddHistory(c.StepID, "doc:"+msg.Document.FileID)

	// Execute completion handler if specified
	if step.OnComplete != "" {
		if err := r.flowEngine.ExecuteStepHandler(ctx, c, step.OnComplete); err != nil {
			r.logDebug("Step handler error: %v", err)
		}
		return
	}

	// Determine and transition to next step
	nextStep := r.flowEngine.DetermineNextStep(ctx, c, msg.Document.FileID)
	if nextStep != "" {
		r.convManager.ChangeStep(ctx, msg.From.ID, c.ChatID, nextStep)
		r.displayStep(ctx, c)
	}
}

// displayStep triggers the step display function if configured.
func (r *Router) displayStep(ctx context.Context, c *conv.Conversation) {
	r.mu.RLock()
	fn := r.stepDisplayFunc
	r.mu.RUnlock()

	if fn != nil {
		if err := fn(ctx, c); err != nil {
			r.logDebug("Step display error: %v", err)
		}
	}
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
