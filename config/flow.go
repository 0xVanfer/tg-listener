// Package config defines configuration structures for tgwrapper.
package config

import "time"

// FlowConfig defines a conversation flow configuration.
// A flow represents a multi-step interaction with the user,
// consisting of prompts, user inputs, validations, and branching logic.
type FlowConfig struct {
	// ID is the unique identifier for this flow.
	// Referenced when starting a conversation with flow_id.
	ID string `json:"id" yaml:"id" mapstructure:"id"`

	// Name is a human-readable name for the flow.
	Name string `json:"name" yaml:"name" mapstructure:"name"`

	// InitialStep is the ID of the first step to execute.
	// Must reference a valid step in the Steps map.
	InitialStep string `json:"initial_step" yaml:"initial_step" mapstructure:"initial_step"`

	// Steps is a map of step configurations keyed by step ID.
	Steps map[string]*StepConfig `json:"steps" yaml:"steps" mapstructure:"steps"`

	// TTL is the time-to-live for conversations in this flow.
	// Overrides the default TTL if set.
	TTL time.Duration `json:"ttl" yaml:"ttl" mapstructure:"ttl"`

	// OnStart is the name of a hook function to call when the flow starts.
	OnStart string `json:"on_start" yaml:"on_start" mapstructure:"on_start"`

	// OnEnd is the name of a hook function to call when the flow ends.
	OnEnd string `json:"on_end" yaml:"on_end" mapstructure:"on_end"`
}

// InputType defines what kind of input a step expects from the user.
type InputType string

const (
	// InputTypeText expects text message input from the user.
	InputTypeText InputType = "text"

	// InputTypeCallback expects callback button press from the user.
	InputTypeCallback InputType = "callback"

	// InputTypeAny accepts either text or callback input.
	InputTypeAny InputType = "any"

	// InputTypeNone requires no input and auto-advances.
	InputTypeNone InputType = "none"

	// InputTypePhoto expects a photo upload from the user.
	InputTypePhoto InputType = "photo"

	// InputTypeDocument expects a document upload from the user.
	InputTypeDocument InputType = "document"
)

// StepConfig defines a single step within a conversation flow.
// Each step can display a prompt, show a keyboard, validate input,
// and determine the next step based on conditions or user input.
type StepConfig struct {
	// ID is the unique identifier for this step within the flow.
	ID string `json:"id" yaml:"id" mapstructure:"id"`

	// PromptText is the message text to display to the user.
	// Supports template variables like {{.data.key}}.
	PromptText string `json:"prompt_text" yaml:"prompt_text" mapstructure:"prompt_text"`

	// PromptTemplate is an advanced template for complex formatting.
	PromptTemplate string `json:"prompt_template" yaml:"prompt_template" mapstructure:"prompt_template"`

	// Keyboard defines the inline keyboard to display with the prompt.
	Keyboard *KeyboardConfig `json:"keyboard" yaml:"keyboard" mapstructure:"keyboard"`

	// InputType specifies what type of input this step expects.
	InputType InputType `json:"input_type" yaml:"input_type" mapstructure:"input_type"`

	// Validation defines input validation rules.
	Validation *ValidationConfig `json:"validation" yaml:"validation" mapstructure:"validation"`

	// NextStep is the ID of the next step (for simple linear flows).
	NextStep string `json:"next_step" yaml:"next_step" mapstructure:"next_step"`

	// Branches defines conditional branching based on input.
	Branches []BranchConfig `json:"branches" yaml:"branches" mapstructure:"branches"`

	// OnEnter is the name of a hook function to call when entering this step.
	OnEnter string `json:"on_enter" yaml:"on_enter" mapstructure:"on_enter"`

	// OnComplete is the name of a handler function to call when the step completes.
	// If set, the handler is responsible for determining what happens next.
	OnComplete string `json:"on_complete" yaml:"on_complete" mapstructure:"on_complete"`

	// StoreAs is the key name under which to store user input in conversation data.
	StoreAs string `json:"store_as" yaml:"store_as" mapstructure:"store_as"`

	// SkipIf is a condition expression; if true, skip this step.
	SkipIf string `json:"skip_if" yaml:"skip_if" mapstructure:"skip_if"`

	// ParseMode specifies the Telegram parse mode for the prompt.
	ParseMode string `json:"parse_mode" yaml:"parse_mode" mapstructure:"parse_mode"`
}

// ValidationConfig defines input validation rules for a step.
type ValidationConfig struct {
	// Type is the validation type: number, address, email, regex, custom, required.
	Type string `json:"type" yaml:"type" mapstructure:"type"`

	// Pattern is the regex pattern (when Type is "regex").
	Pattern string `json:"pattern" yaml:"pattern" mapstructure:"pattern"`

	// ErrorMsg is the message shown when validation fails.
	ErrorMsg string `json:"error_msg" yaml:"error_msg" mapstructure:"error_msg"`

	// Min is the minimum value (when Type is "number").
	Min string `json:"min" yaml:"min" mapstructure:"min"`

	// Max is the maximum value (when Type is "number").
	Max string `json:"max" yaml:"max" mapstructure:"max"`

	// MinLength is the minimum text length (when Type is "text").
	MinLength int `json:"min_length" yaml:"min_length" mapstructure:"min_length"`

	// MaxLength is the maximum text length (when Type is "text").
	MaxLength int `json:"max_length" yaml:"max_length" mapstructure:"max_length"`

	// Custom is the name of a custom validator function.
	Custom string `json:"custom" yaml:"custom" mapstructure:"custom"`
}

// BranchConfig defines a conditional branch for step transitions.
type BranchConfig struct {
	// Condition is the expression to evaluate.
	// Supported formats:
	// - "data == 'xxx'" - exact match
	// - "data.startsWith('prefix:')" - prefix match
	// - "data.contains('keyword')" - contains match
	// - "custom:handlerName" - custom condition handler
	Condition string `json:"condition" yaml:"condition" mapstructure:"condition"`

	// NextStep is the step ID to transition to when condition is true.
	NextStep string `json:"next_step" yaml:"next_step" mapstructure:"next_step"`

	// Handler is the name of a handler function to execute when condition is true.
	Handler string `json:"handler" yaml:"handler" mapstructure:"handler"`
}

// Validate checks if the flow configuration is valid.
// Returns an error if the flow is missing required fields or has invalid references.
func (f *FlowConfig) Validate() error {
	if f.ID == "" {
		return ErrInvalidFlow
	}
	if f.InitialStep == "" {
		return ErrInvalidFlow
	}
	if len(f.Steps) == 0 {
		return ErrInvalidFlow
	}
	if _, ok := f.Steps[f.InitialStep]; !ok {
		return ErrStepNotFound
	}
	return nil
}

// GetStep retrieves a step configuration by ID.
// Returns nil if the step doesn't exist.
func (f *FlowConfig) GetStep(stepID string) *StepConfig {
	return f.Steps[stepID]
}

// GetTTL returns the flow's TTL or the provided default if not set.
func (f *FlowConfig) GetTTL(defaultTTL time.Duration) time.Duration {
	if f.TTL > 0 {
		return f.TTL
	}
	return defaultTTL
}
