// Package config defines configuration structures for tgwrapper.
package config

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the complete configuration structure for the tgwrapper library.
// It contains all settings for the bot, menus, conversation flows, and environment.
type Config struct {
	// Bot contains the bot-level configuration including token and commands.
	Bot *BotConfig `json:"bot" yaml:"bot" mapstructure:"bot"`

	// Menus is a map of menu configurations keyed by menu ID.
	// Use AddMenu() to add menus programmatically.
	Menus map[string]*MenuConfig `json:"menus" yaml:"menus" mapstructure:"menus"`

	// Flows is a map of conversation flow configurations keyed by flow ID.
	// Use AddFlow() to add flows programmatically.
	Flows map[string]*FlowConfig `json:"flows" yaml:"flows" mapstructure:"flows"`

	// MainMenuID specifies which menu to use as the main/home menu.
	// If empty, the menu with ID "main" is used by default.
	MainMenuID string `json:"main_menu_id" yaml:"main_menu_id" mapstructure:"main_menu_id"`

	// Environment contains custom variables for conditional logic.
	// These can be accessed in condition expressions.
	Environment map[string]interface{} `json:"environment" yaml:"environment" mapstructure:"environment"`
}

// NewConfig creates a new empty configuration with initialized maps.
func NewConfig() *Config {
	return &Config{
		Bot:   NewDefaultBotConfig(),
		Menus: make(map[string]*MenuConfig),
		Flows: make(map[string]*FlowConfig),
	}
}

// LoadFromFile loads configuration from a file.
// Supports both JSON and YAML formats based on file extension.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadFromBytes(data, path)
}

// LoadFromBytes loads configuration from byte data.
// The path parameter is used to determine the file format (JSON or YAML).
func LoadFromBytes(data []byte, path string) (*Config, error) {
	cfg := NewConfig()

	// Determine parsing method based on file extension
	if len(path) > 5 && path[len(path)-5:] == ".json" {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		// Default to YAML parsing
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Validate checks if all configuration components are valid.
func (c *Config) Validate() error {
	if c.Bot != nil {
		if err := c.Bot.Validate(); err != nil {
			return err
		}
	}

	for _, menu := range c.Menus {
		if err := menu.Validate(); err != nil {
			return err
		}
	}

	for _, flow := range c.Flows {
		if err := flow.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetMenu retrieves a menu configuration by ID.
// Returns nil if the menu doesn't exist.
func (c *Config) GetMenu(id string) *MenuConfig {
	return c.Menus[id]
}

// GetFlow retrieves a flow configuration by ID.
// Returns nil if the flow doesn't exist.
func (c *Config) GetFlow(id string) *FlowConfig {
	return c.Flows[id]
}

// GetMainMenu retrieves the main menu configuration.
// Falls back to the "main" menu if MainMenuID is not set.
func (c *Config) GetMainMenu() *MenuConfig {
	if c.MainMenuID == "" {
		return c.Menus["main"]
	}
	return c.Menus[c.MainMenuID]
}

// AddMenu adds a menu to the configuration.
// The menu's ID is used as the map key.
func (c *Config) AddMenu(menu *MenuConfig) {
	if c.Menus == nil {
		c.Menus = make(map[string]*MenuConfig)
	}
	c.Menus[menu.ID] = menu
}

// AddFlow adds a flow to the configuration.
// The flow's ID is used as the map key.
func (c *Config) AddFlow(flow *FlowConfig) {
	if c.Flows == nil {
		c.Flows = make(map[string]*FlowConfig)
	}
	c.Flows[flow.ID] = flow
}

// GetEnv retrieves an environment variable by key.
// Returns nil if the key doesn't exist.
func (c *Config) GetEnv(key string) interface{} {
	if c.Environment == nil {
		return nil
	}
	return c.Environment[key]
}

// GetEnvBool retrieves a boolean environment variable.
// Returns false if the key doesn't exist or is not a boolean.
func (c *Config) GetEnvBool(key string) bool {
	v := c.GetEnv(key)
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetEnvString retrieves a string environment variable.
// Returns empty string if the key doesn't exist or is not a string.
func (c *Config) GetEnvString(key string) string {
	v := c.GetEnv(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
