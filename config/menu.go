// Package config defines configuration structures for tgwrapper.
package config

// MenuConfig defines a menu configuration.
// Menus are standalone message templates with buttons that can be displayed
// at any time, independent of conversation flows.
type MenuConfig struct {
	// ID is the unique identifier for this menu.
	// Referenced when navigating with menu_id.
	ID string `json:"id" yaml:"id" mapstructure:"id"`

	// Text is the message text to display.
	// Supports Markdown/HTML based on ParseMode.
	Text string `json:"text" yaml:"text" mapstructure:"text"`

	// Buttons defines button rows as a 2D array.
	// Each inner array represents a row of buttons.
	Buttons [][]ButtonConfig `json:"buttons" yaml:"buttons" mapstructure:"buttons"`

	// Pages defines pagination for large menus.
	// Each page can have its own set of buttons.
	Pages []PageConfig `json:"pages" yaml:"pages" mapstructure:"pages"`

	// Condition is an expression that determines when this menu should be shown.
	Condition string `json:"condition" yaml:"condition" mapstructure:"condition"`

	// ParseMode specifies the text formatting: Markdown, MarkdownV2, or HTML.
	ParseMode string `json:"parse_mode" yaml:"parse_mode" mapstructure:"parse_mode"`
}

// ButtonConfig defines a single button in a menu or keyboard.
type ButtonConfig struct {
	// Text is the button label shown to users.
	Text string `json:"text" yaml:"text" mapstructure:"text"`

	// Callback is the callback data sent when button is pressed.
	// Mutually exclusive with URL, FlowID, and MenuID.
	Callback string `json:"callback" yaml:"callback" mapstructure:"callback"`

	// URL opens a link when button is pressed.
	// Mutually exclusive with Callback, FlowID, and MenuID.
	URL string `json:"url" yaml:"url" mapstructure:"url"`

	// FlowID starts a conversation flow when button is pressed.
	// Mutually exclusive with Callback, URL, and MenuID.
	FlowID string `json:"flow_id" yaml:"flow_id" mapstructure:"flow_id"`

	// MenuID navigates to another menu when button is pressed.
	// Mutually exclusive with Callback, URL, and FlowID.
	MenuID string `json:"menu_id" yaml:"menu_id" mapstructure:"menu_id"`

	// Handler is the name of a custom handler function to call.
	Handler string `json:"handler" yaml:"handler" mapstructure:"handler"`

	// Condition is an expression that determines when this button should be shown.
	Condition string `json:"condition" yaml:"condition" mapstructure:"condition"`
}

// PageConfig defines a page within a paginated menu.
type PageConfig struct {
	// ID is the unique identifier for this page within the menu.
	ID string `json:"id" yaml:"id" mapstructure:"id"`

	// Name is a human-readable name for the page.
	Name string `json:"name" yaml:"name" mapstructure:"name"`

	// Buttons defines the button rows for this page.
	Buttons [][]ButtonConfig `json:"buttons" yaml:"buttons" mapstructure:"buttons"`

	// Condition is an expression that determines when this page should be shown.
	Condition string `json:"condition" yaml:"condition" mapstructure:"condition"`

	// NavButton configures the navigation button to this page.
	NavButton *ButtonConfig `json:"nav_button" yaml:"nav_button" mapstructure:"nav_button"`
}

// Validate checks if the menu configuration is valid.
func (m *MenuConfig) Validate() error {
	if m.ID == "" {
		return ErrInvalidMenu
	}
	if m.Text == "" && len(m.Buttons) == 0 && len(m.Pages) == 0 {
		return ErrInvalidMenu
	}
	return nil
}

// GetButtons returns the buttons for a specific page or the default buttons.
// If pageID is empty or not found, returns the main Buttons.
func (m *MenuConfig) GetButtons(pageID string) [][]ButtonConfig {
	if pageID == "" || len(m.Pages) == 0 {
		return m.Buttons
	}
	for _, p := range m.Pages {
		if p.ID == pageID {
			return p.Buttons
		}
	}
	return m.Buttons
}

// HasPages returns true if the menu has pagination.
func (m *MenuConfig) HasPages() bool {
	return len(m.Pages) > 0
}

// ButtonData represents dynamic button data returned by keyboard providers.
// Used for runtime button generation.
type ButtonData struct {
	// Text is the button label shown to users.
	Text string `json:"text" yaml:"text" mapstructure:"text"`

	// Callback is the callback data sent when button is pressed.
	Callback string `json:"callback" yaml:"callback" mapstructure:"callback"`

	// URL opens a link when button is pressed.
	URL string `json:"url" yaml:"url" mapstructure:"url"`
}

// ToButtonConfig converts ButtonData to ButtonConfig for unified processing.
func (b ButtonData) ToButtonConfig() ButtonConfig {
	return ButtonConfig{
		Text:     b.Text,
		Callback: b.Callback,
		URL:      b.URL,
	}
}
