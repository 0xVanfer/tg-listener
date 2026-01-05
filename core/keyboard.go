// Package core provides keyboard building functionality.
package core

import (
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

// Callback constants for common navigation actions.
const (
	// CallbackMainMenu returns to the main menu.
	CallbackMainMenu = "main_menu"
	// CallbackBack returns to the previous step.
	CallbackBack = "back"
	// CallbackCancel cancels the operation (functionally same as back).
	CallbackCancel = "cancel"
	// CallbackPage is the prefix for pagination callbacks.
	CallbackPage = "page:"
)

// KeyboardBuilder provides a fluent interface for building inline keyboards.
type KeyboardBuilder struct {
	rows [][]telego.InlineKeyboardButton
}

// NewKeyboard creates a new keyboard builder instance.
func NewKeyboard() *KeyboardBuilder {
	return &KeyboardBuilder{
		rows: make([][]telego.InlineKeyboardButton, 0),
	}
}

// Row adds a row of buttons to the keyboard.
func (kb *KeyboardBuilder) Row(buttons ...telego.InlineKeyboardButton) *KeyboardBuilder {
	if len(buttons) > 0 {
		kb.rows = append(kb.rows, buttons)
	}
	return kb
}

// Button adds a single callback button as a new row.
func (kb *KeyboardBuilder) Button(text, callback string) *KeyboardBuilder {
	return kb.Row(Button(text, callback))
}

// URLButton adds a single URL button as a new row.
func (kb *KeyboardBuilder) URLButton(text, url string) *KeyboardBuilder {
	return kb.Row(URLButton(text, url))
}

// Grid arranges buttons in a grid with specified number of columns.
func (kb *KeyboardBuilder) Grid(buttons []telego.InlineKeyboardButton, columns int) *KeyboardBuilder {
	if columns <= 0 {
		columns = 2
	}

	for i := 0; i < len(buttons); i += columns {
		end := min(i+columns, len(buttons))
		kb.rows = append(kb.rows, buttons[i:end])
	}
	return kb
}

// Back adds a back navigation button.
func (kb *KeyboardBuilder) Back(text string) *KeyboardBuilder {
	if text == "" {
		text = "⬅️ Back"
	}
	return kb.Row(Button(text, CallbackBack))
}

// MainMenu adds a main menu navigation button.
func (kb *KeyboardBuilder) MainMenu(text string) *KeyboardBuilder {
	if text == "" {
		text = "🏠 Main Menu"
	}
	return kb.Row(Button(text, CallbackMainMenu))
}

// Navigation adds a row with both back and main menu buttons.
func (kb *KeyboardBuilder) Navigation(backText, mainText string) *KeyboardBuilder {
	if backText == "" {
		backText = "⬅️ Back"
	}
	if mainText == "" {
		mainText = "🏠 Main Menu"
	}
	return kb.Row(
		Button(backText, CallbackBack),
		Button(mainText, CallbackMainMenu),
	)
}

// Pagination adds pagination navigation buttons.
func (kb *KeyboardBuilder) Pagination(currentPage, totalPages int, prefix string) *KeyboardBuilder {
	if totalPages <= 1 {
		return kb
	}

	var buttons []telego.InlineKeyboardButton

	// Previous page button
	if currentPage > 1 {
		buttons = append(buttons, Button("⬅️", prefix+string(rune('0'+currentPage-1))))
	}

	// Page indicator
	buttons = append(buttons, Button(
		string(rune('0'+currentPage))+"/"+string(rune('0'+totalPages)),
		"noop",
	))

	// Next page button
	if currentPage < totalPages {
		buttons = append(buttons, Button("➡️", prefix+string(rune('0'+currentPage+1))))
	}

	if len(buttons) > 0 {
		kb.rows = append(kb.rows, buttons)
	}
	return kb
}

// Build constructs and returns the InlineKeyboardMarkup.
// Returns nil if no buttons were added.
func (kb *KeyboardBuilder) Build() *telego.InlineKeyboardMarkup {
	if len(kb.rows) == 0 {
		return nil
	}
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: kb.rows,
	}
}

// Button creates a callback button with the given text and callback data.
func Button(text, callback string) telego.InlineKeyboardButton {
	return telegoutil.InlineKeyboardButton(text).WithCallbackData(callback)
}

// URLButton creates a URL button that opens a link when pressed.
func URLButton(text, url string) telego.InlineKeyboardButton {
	return telegoutil.InlineKeyboardButton(text).WithURL(url)
}

// SwitchInlineButton creates a button that switches to inline query mode.
func SwitchInlineButton(text, query string) telego.InlineKeyboardButton {
	return telegoutil.InlineKeyboardButton(text).WithSwitchInlineQueryCurrentChat(query)
}

// WebAppButton creates a button that opens a Web App.
func WebAppButton(text, url string) telego.InlineKeyboardButton {
	return telegoutil.InlineKeyboardButton(text).WithWebApp(&telego.WebAppInfo{URL: url})
}

// BuildFromConfig builds a keyboard from button configuration.
// This is a convenience function for converting config to keyboard.
func BuildFromConfig(buttons [][]ButtonConfig) *telego.InlineKeyboardMarkup {
	kb := NewKeyboard()

	for _, row := range buttons {
		var rowButtons []telego.InlineKeyboardButton
		for _, btn := range row {
			if btn.URL != "" {
				rowButtons = append(rowButtons, URLButton(btn.Text, btn.URL))
			} else if btn.Callback != "" {
				rowButtons = append(rowButtons, Button(btn.Text, btn.Callback))
			} else if btn.FlowID != "" {
				rowButtons = append(rowButtons, Button(btn.Text, "flow:"+btn.FlowID))
			} else if btn.MenuID != "" {
				rowButtons = append(rowButtons, Button(btn.Text, "menu:"+btn.MenuID))
			}
		}
		if len(rowButtons) > 0 {
			kb.Row(rowButtons...)
		}
	}

	return kb.Build()
}

// ButtonConfig is a simplified button configuration to avoid circular imports.
// Used by BuildFromConfig for converting config to keyboard.
type ButtonConfig struct {
	Text     string // Button display text
	Callback string // Callback data
	URL      string // URL for link buttons
	FlowID   string // Flow ID to start
	MenuID   string // Menu ID to navigate to
}
