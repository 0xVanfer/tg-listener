// Package config defines configuration structures for tgwrapper.
package config

import "errors"

// Configuration-related error definitions.
// These errors are returned during configuration validation and lookup operations.
var (
	// ErrEmptyToken is returned when the bot token is empty or not provided.
	ErrEmptyToken = errors.New("bot token is empty")

	// ErrInvalidFlow is returned when a flow configuration is malformed.
	ErrInvalidFlow = errors.New("invalid flow configuration")

	// ErrInvalidStep is returned when a step configuration is malformed.
	ErrInvalidStep = errors.New("invalid step configuration")

	// ErrInvalidMenu is returned when a menu configuration is malformed.
	ErrInvalidMenu = errors.New("invalid menu configuration")

	// ErrFlowNotFound is returned when a referenced flow does not exist.
	ErrFlowNotFound = errors.New("flow not found")

	// ErrStepNotFound is returned when a referenced step does not exist.
	ErrStepNotFound = errors.New("step not found")

	// ErrMenuNotFound is returned when a referenced menu does not exist.
	ErrMenuNotFound = errors.New("menu not found")

	// ErrHandlerNotFound is returned when a referenced handler function is not registered.
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrProviderNotFound is returned when a referenced keyboard provider is not registered.
	ErrProviderNotFound = errors.New("provider not found")

	// ErrValidatorNotFound is returned when a referenced validator is not registered.
	ErrValidatorNotFound = errors.New("validator not found")
)
