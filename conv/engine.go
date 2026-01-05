// Package conv provides the flow engine for executing conversation flows.
package conv

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/0xVanfer/tg-listener/config"
)

// StepHandler is a function type for handling step completion.
// Called when a step is completed to perform custom business logic.
type StepHandler func(ctx context.Context, conv *Conversation) error

// KeyboardProvider is a function type for providing dynamic keyboard data.
// Called when a step needs dynamically generated buttons.
type KeyboardProvider func(ctx context.Context, conv *Conversation) []config.ButtonData

// Validator is a function type for custom input validation.
// Called to validate user input with custom rules.
type Validator func(value string, conv *Conversation) error

// ConditionEvaluator is a function type for evaluating conditions.
// Used for evaluating complex conditions in flow branching.
type ConditionEvaluator func(ctx context.Context, conv *Conversation, condition string) bool

// FlowEngine manages flow execution, step handlers, and validation.
// It provides the core logic for multi-step conversation flows.
type FlowEngine struct {
	config             *config.Config              // Configuration containing flow definitions
	stepHandlers       map[string]StepHandler      // Registered step completion handlers
	keyboardProviders  map[string]KeyboardProvider // Registered dynamic keyboard providers
	validators         map[string]Validator        // Registered custom validators
	conditionEvaluator ConditionEvaluator          // Custom condition evaluator

	mu sync.RWMutex // Mutex for thread-safe operations
}

// NewFlowEngine creates a new flow engine with the given configuration.
func NewFlowEngine(cfg *config.Config) *FlowEngine {
	return &FlowEngine{
		config:            cfg,
		stepHandlers:      make(map[string]StepHandler),
		keyboardProviders: make(map[string]KeyboardProvider),
		validators:        make(map[string]Validator),
	}
}

// SetConfig updates the flow engine configuration.
func (e *FlowEngine) SetConfig(cfg *config.Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = cfg
}

// RegisterStepHandler registers a step completion handler by name.
// The handler will be called when the step's OnComplete field matches the name.
func (e *FlowEngine) RegisterStepHandler(name string, handler StepHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stepHandlers[name] = handler
}

// RegisterKeyboardProvider registers a dynamic keyboard data provider.
// The provider will be called when a step's keyboard has type "dynamic" and matching Provider field.
func (e *FlowEngine) RegisterKeyboardProvider(name string, provider KeyboardProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.keyboardProviders[name] = provider
}

// RegisterValidator registers a custom validator by name.
// The validator will be called when validation type is "custom" with matching Custom field.
func (e *FlowEngine) RegisterValidator(name string, validator Validator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.validators[name] = validator
}

// SetConditionEvaluator sets a custom condition evaluator.
// Used for evaluating complex conditions that the built-in evaluator cannot handle.
func (e *FlowEngine) SetConditionEvaluator(evaluator ConditionEvaluator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.conditionEvaluator = evaluator
}

// GetStepHandler retrieves a registered step handler by name.
func (e *FlowEngine) GetStepHandler(name string) StepHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stepHandlers[name]
}

// GetKeyboardProvider retrieves a registered keyboard provider by name.
func (e *FlowEngine) GetKeyboardProvider(name string) KeyboardProvider {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.keyboardProviders[name]
}

// GetValidator retrieves a registered validator by name.
func (e *FlowEngine) GetValidator(name string) Validator {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.validators[name]
}

// GetFlow retrieves a flow configuration by ID.
func (e *FlowEngine) GetFlow(flowID string) *config.FlowConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.config == nil {
		return nil
	}
	return e.config.GetFlow(flowID)
}

// GetStep retrieves a step configuration from a flow.
func (e *FlowEngine) GetStep(flowID, stepID string) *config.StepConfig {
	flow := e.GetFlow(flowID)
	if flow == nil {
		return nil
	}
	return flow.GetStep(stepID)
}

// ValidateInput validates user input against the step's validation rules.
// Returns an error if validation fails, nil if valid or no validation configured.
func (e *FlowEngine) ValidateInput(conv *Conversation, input string) error {
	step := e.GetStep(conv.FlowID, conv.StepID)
	if step == nil {
		return nil
	}

	if step.Validation == nil {
		return nil
	}

	validation := step.Validation

	switch validation.Type {
	case "number":
		return e.validateNumber(input, validation)
	case "address":
		return e.validateAddress(input, validation)
	case "email":
		return e.validateEmail(input, validation)
	case "regex":
		return e.validateRegex(input, validation)
	case "custom":
		return e.validateCustom(input, validation, conv)
	default:
		return nil
	}
}

// validateNumber validates numeric input with optional min/max constraints.
func (e *FlowEngine) validateNumber(input string, validation *config.ValidationConfig) error {
	num, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return errors.New(getErrorMsg(validation.ErrorMsg, "Please enter a valid number"))
	}

	if validation.Min != "" {
		min, err := strconv.ParseFloat(validation.Min, 64)
		if err == nil && num < min {
			return errors.New(getErrorMsg(validation.ErrorMsg, "Number cannot be less than "+validation.Min))
		}
	}

	if validation.Max != "" {
		max, err := strconv.ParseFloat(validation.Max, 64)
		if err == nil && num > max {
			return errors.New(getErrorMsg(validation.ErrorMsg, "Number cannot be greater than "+validation.Max))
		}
	}

	return nil
}

// getErrorMsg returns the configured error message or a default if not configured.
func getErrorMsg(configured, defaultMsg string) string {
	if configured != "" {
		return configured
	}
	return defaultMsg
}

// validateAddress validates Ethereum-style addresses (0x + 40 hex chars).
func (e *FlowEngine) validateAddress(input string, validation *config.ValidationConfig) error {
	if len(input) != 42 || !strings.HasPrefix(input, "0x") {
		return errors.New(getErrorMsg(validation.ErrorMsg, "Please enter a valid Ethereum address"))
	}

	// Check if it contains valid hexadecimal characters
	_, err := strconv.ParseUint(input[2:], 16, 64)
	if err != nil {
		// Address is too long for single uint64, check character by character
		for _, c := range input[2:] {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return errors.New(getErrorMsg(validation.ErrorMsg, "Please enter a valid Ethereum address"))
			}
		}
	}

	return nil
}

// validateEmail validates email format using regex.
func (e *FlowEngine) validateEmail(input string, validation *config.ValidationConfig) error {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, input)
	if !matched {
		return errors.New(getErrorMsg(validation.ErrorMsg, "Please enter a valid email address"))
	}
	return nil
}

// validateRegex validates input against a custom regex pattern.
func (e *FlowEngine) validateRegex(input string, validation *config.ValidationConfig) error {
	if validation.Pattern == "" {
		return nil
	}

	matched, err := regexp.MatchString(validation.Pattern, input)
	if err != nil {
		return errors.New("Validation rule configuration error")
	}

	if !matched {
		return errors.New(getErrorMsg(validation.ErrorMsg, "Input format is incorrect"))
	}

	return nil
}

// validateCustom validates input using a registered custom validator.
func (e *FlowEngine) validateCustom(input string, validation *config.ValidationConfig, conv *Conversation) error {
	if validation.Custom == "" {
		return nil
	}

	validator := e.GetValidator(validation.Custom)
	if validator == nil {
		return nil
	}

	return validator(input, conv)
}

// EvaluateCondition evaluates a condition expression.
// Uses custom evaluator if set, otherwise falls back to built-in simple evaluation.
func (e *FlowEngine) EvaluateCondition(ctx context.Context, conv *Conversation, condition string) bool {
	if condition == "" {
		return true
	}

	// Use custom evaluator if available
	if e.conditionEvaluator != nil {
		return e.conditionEvaluator(ctx, conv, condition)
	}

	// Fall back to simple built-in condition evaluation
	return e.simpleEvaluate(conv, condition)
}

// simpleEvaluate provides basic condition evaluation.
// Supports simple equality (==) and inequality (!=) comparisons.
func (e *FlowEngine) simpleEvaluate(conv *Conversation, condition string) bool {
	// Support simple data.key == "value" format
	parts := strings.Split(condition, "==")
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		expected := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		// Remove data. prefix
		key = strings.TrimPrefix(key, "data.")

		actual := conv.GetString(key)
		return actual == expected
	}

	// Support data.key != "value" format
	parts = strings.Split(condition, "!=")
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		expected := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		key = strings.TrimPrefix(key, "data.")

		actual := conv.GetString(key)
		return actual != expected
	}

	return true
}

// DetermineNextStep determines the next step based on input and branch conditions.
// Evaluates branch conditions in order and returns the first matching next step,
// or falls back to the default NextStep if no branches match.
func (e *FlowEngine) DetermineNextStep(ctx context.Context, conv *Conversation, input string) string {
	step := e.GetStep(conv.FlowID, conv.StepID)
	if step == nil {
		return ""
	}

	// Check branch conditions
	for _, branch := range step.Branches {
		if e.evaluateBranchCondition(conv, branch.Condition, input) {
			return branch.NextStep
		}
	}

	// Return default next step
	return step.NextStep
}

// evaluateBranchCondition evaluates a branch condition against the input.
// Supports input == "xxx" format and general condition evaluation.
func (e *FlowEngine) evaluateBranchCondition(conv *Conversation, condition, input string) bool {
	// Support input == "xxx" format
	if strings.HasPrefix(condition, "input") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			expected := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			return input == expected
		}
	}

	// Use general condition evaluation
	return e.simpleEvaluate(conv, condition)
}

// ExecuteStepHandler executes a registered step handler by name.
// Returns nil if no handler is registered for the given name.
func (e *FlowEngine) ExecuteStepHandler(ctx context.Context, conv *Conversation, handlerName string) error {
	handler := e.GetStepHandler(handlerName)
	if handler == nil {
		return nil
	}
	return handler(ctx, conv)
}

// GetDynamicKeyboardData retrieves dynamic keyboard data from a registered provider.
// Returns nil if no provider is registered for the given name.
func (e *FlowEngine) GetDynamicKeyboardData(ctx context.Context, conv *Conversation, providerName string) []config.ButtonData {
	provider := e.GetKeyboardProvider(providerName)
	if provider == nil {
		return nil
	}
	return provider(ctx, conv)
}
