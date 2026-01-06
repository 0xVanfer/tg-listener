// Package config defines configuration structures for tgwrapper.
package config

import (
	"context"

	"github.com/mymmrac/telego"
)

// HandlerRegistry holds all handler functions that can be referenced by configuration.
// This allows defining handler logic in code while referencing them by name in config.
type HandlerRegistry struct {
	// CommandHandlers maps command handler names to their implementations.
	CommandHandlers map[string]CommandHandlerFunc

	// CallbackHandlers maps callback handler names to their implementations.
	CallbackHandlers map[string]CallbackHandlerFunc

	// StepHandlers maps step handler names to their implementations.
	StepHandlers map[string]StepHandlerFunc

	// KeyboardProviders maps keyboard provider names to their implementations.
	KeyboardProviders map[string]KeyboardProviderFunc

	// Validators maps validator names to their implementations.
	Validators map[string]ValidatorFunc

	// AuthFunc is the authentication function for user authorization.
	AuthFunc AuthFunc

	// OnConversationStart is called when a conversation starts.
	OnConversationStart ConversationHookFunc

	// OnConversationEnd is called when a conversation ends.
	OnConversationEnd ConversationHookFunc

	// OnStepChange is called when a conversation step changes.
	OnStepChange StepChangeHookFunc
}

// CommandHandlerFunc is the function signature for command handlers.
type CommandHandlerFunc func(ctx context.Context, msg telego.Message) error

// CallbackHandlerFunc is the function signature for callback handlers.
type CallbackHandlerFunc func(ctx context.Context, query telego.CallbackQuery) error

// StepHandlerFunc is the function signature for step completion handlers.
// The Conversation type is defined in the conv package, but we use interface{}
// here to avoid circular imports. It will be type-asserted in the wrapper.
type StepHandlerFunc func(ctx context.Context, conv interface{}) error

// KeyboardProviderFunc is the function signature for dynamic keyboard providers.
type KeyboardProviderFunc func(ctx context.Context, conv interface{}) []ButtonData

// ValidatorFunc is the function signature for custom validators.
type ValidatorFunc func(value string, conv interface{}) error

// AuthFunc is the function signature for authentication.
type AuthFunc func(ctx context.Context, userID int64, username string) bool

// ConversationHookFunc is the function signature for conversation lifecycle hooks.
type ConversationHookFunc func(ctx context.Context, conv interface{})

// StepChangeHookFunc is the function signature for step change hooks.
type StepChangeHookFunc func(ctx context.Context, conv interface{}, from, to string)

// NewHandlerRegistry creates a new empty handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		CommandHandlers:   make(map[string]CommandHandlerFunc),
		CallbackHandlers:  make(map[string]CallbackHandlerFunc),
		StepHandlers:      make(map[string]StepHandlerFunc),
		KeyboardProviders: make(map[string]KeyboardProviderFunc),
		Validators:        make(map[string]ValidatorFunc),
	}
}

// RegisterCommand registers a command handler by name.
func (r *HandlerRegistry) RegisterCommand(name string, handler CommandHandlerFunc) *HandlerRegistry {
	r.CommandHandlers[name] = handler
	return r
}

// RegisterCallback registers a callback handler by name.
func (r *HandlerRegistry) RegisterCallback(name string, handler CallbackHandlerFunc) *HandlerRegistry {
	r.CallbackHandlers[name] = handler
	return r
}

// RegisterStepHandler registers a step completion handler by name.
func (r *HandlerRegistry) RegisterStepHandler(name string, handler StepHandlerFunc) *HandlerRegistry {
	r.StepHandlers[name] = handler
	return r
}

// RegisterKeyboardProvider registers a keyboard provider by name.
func (r *HandlerRegistry) RegisterKeyboardProvider(name string, provider KeyboardProviderFunc) *HandlerRegistry {
	r.KeyboardProviders[name] = provider
	return r
}

// RegisterValidator registers a validator by name.
func (r *HandlerRegistry) RegisterValidator(name string, validator ValidatorFunc) *HandlerRegistry {
	r.Validators[name] = validator
	return r
}

// SetAuthFunc sets the authentication function.
func (r *HandlerRegistry) SetAuthFunc(fn AuthFunc) *HandlerRegistry {
	r.AuthFunc = fn
	return r
}

// SetOnConversationStart sets the conversation start hook.
func (r *HandlerRegistry) SetOnConversationStart(fn ConversationHookFunc) *HandlerRegistry {
	r.OnConversationStart = fn
	return r
}

// SetOnConversationEnd sets the conversation end hook.
func (r *HandlerRegistry) SetOnConversationEnd(fn ConversationHookFunc) *HandlerRegistry {
	r.OnConversationEnd = fn
	return r
}

// SetOnStepChange sets the step change hook.
func (r *HandlerRegistry) SetOnStepChange(fn StepChangeHookFunc) *HandlerRegistry {
	r.OnStepChange = fn
	return r
}
