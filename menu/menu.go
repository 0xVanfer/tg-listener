// Package menu provides menu management functionality.
package menu

import (
	"context"
	"sync"

	"github.com/mymmrac/telego"

	"github.com/0xVanfer/tg-listener/config"
	"github.com/0xVanfer/tg-listener/core"
)

// Menu represents a menu with optional pagination support.
type Menu struct {
	Config      *config.MenuConfig // Menu configuration
	CurrentPage int                // Current page number (1-based)
}

// NewMenu creates a new menu instance from configuration.
func NewMenu(cfg *config.MenuConfig) *Menu {
	return &Menu{
		Config:      cfg,
		CurrentPage: 1,
	}
}

// GetText returns the menu text.
func (m *Menu) GetText() string {
	return m.Config.Text
}

// GetKeyboard builds and returns the menu keyboard.
// The evaluator function is used to evaluate button visibility conditions.
func (m *Menu) GetKeyboard(ctx context.Context, evaluator func(condition string) bool) *telego.InlineKeyboardMarkup {
	kb := core.NewKeyboard()

	// If menu has pages, show current page keyboard
	if len(m.Config.Pages) > 0 {
		return m.getPageKeyboard(ctx, evaluator)
	}

	// Add regular buttons
	for _, row := range m.Config.Buttons {
		var buttons []telego.InlineKeyboardButton
		for _, btn := range row {
			// Check button condition
			if btn.Condition != "" && evaluator != nil && !evaluator(btn.Condition) {
				continue
			}
			buttons = append(buttons, m.buildButton(btn))
		}
		if len(buttons) > 0 {
			kb.Row(buttons...)
		}
	}

	return kb.Build()
}

// getPageKeyboard builds the keyboard for a paginated menu.
func (m *Menu) getPageKeyboard(ctx context.Context, evaluator func(condition string) bool) *telego.InlineKeyboardMarkup {
	if m.CurrentPage < 1 || m.CurrentPage > len(m.Config.Pages) {
		m.CurrentPage = 1
	}

	kb := core.NewKeyboard()
	page := m.Config.Pages[m.CurrentPage-1]

	// Add buttons for current page
	for _, row := range page.Buttons {
		var buttons []telego.InlineKeyboardButton
		for _, btn := range row {
			if btn.Condition != "" && evaluator != nil && !evaluator(btn.Condition) {
				continue
			}
			buttons = append(buttons, m.buildButton(btn))
		}
		if len(buttons) > 0 {
			kb.Row(buttons...)
		}
	}

	// Add pagination navigation
	if len(m.Config.Pages) > 1 {
		kb.Pagination(m.CurrentPage, len(m.Config.Pages), core.CallbackPage)
	}

	return kb.Build()
}

// buildButton creates a keyboard button from configuration.
func (m *Menu) buildButton(btn config.ButtonConfig) telego.InlineKeyboardButton {
	if btn.URL != "" {
		return core.URLButton(btn.Text, btn.URL)
	}
	if btn.FlowID != "" {
		return core.Button(btn.Text, "flow:"+btn.FlowID)
	}
	if btn.MenuID != "" {
		return core.Button(btn.Text, "menu:"+btn.MenuID)
	}
	return core.Button(btn.Text, btn.Callback)
}

// SetPage sets the current page number.
func (m *Menu) SetPage(page int) {
	if page >= 1 && page <= len(m.Config.Pages) {
		m.CurrentPage = page
	}
}

// Manager manages menu instances and provides menu display functionality.
type Manager struct {
	bot    *core.Bot        // Bot instance for sending messages
	config *config.Config   // Configuration
	menus  map[string]*Menu // Menu instances by ID
	mu     sync.RWMutex     // Mutex for thread-safe operations
}

// NewManager creates a new menu manager.
func NewManager(bot *core.Bot, cfg *config.Config) *Manager {
	m := &Manager{
		bot:    bot,
		config: cfg,
		menus:  make(map[string]*Menu),
	}

	// Initialize all menus from configuration
	if cfg != nil {
		for id, menuCfg := range cfg.Menus {
			m.menus[id] = NewMenu(menuCfg)
		}
	}

	return m
}

// SetConfig updates the manager configuration and reinitializes menus.
func (m *Manager) SetConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg

	// Reinitialize menus
	m.menus = make(map[string]*Menu)
	if cfg != nil {
		for id, menuCfg := range cfg.Menus {
			m.menus[id] = NewMenu(menuCfg)
		}
	}
}

// GetMenu retrieves a menu by ID.
func (m *Manager) GetMenu(menuID string) *Menu {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.menus[menuID]
}

// ShowMenu displays a menu by sending a new message.
func (m *Manager) ShowMenu(ctx context.Context, chatID int64, topicID int, menuID string, evaluator func(string) bool) (*telego.Message, error) {
	menu := m.GetMenu(menuID)
	if menu == nil {
		return nil, nil
	}

	text := menu.GetText()
	keyboard := menu.GetKeyboard(ctx, evaluator)

	return m.bot.SendMessageWithKeyboard(ctx, chatID, topicID, text, keyboard)
}

// EditToMenu edits an existing message to show a menu.
func (m *Manager) EditToMenu(ctx context.Context, chatID int64, messageID int, menuID string, evaluator func(string) bool) (*telego.Message, error) {
	menu := m.GetMenu(menuID)
	if menu == nil {
		return nil, nil
	}

	text := menu.GetText()
	keyboard := menu.GetKeyboard(ctx, evaluator)

	return m.bot.EditMessageWithKeyboard(ctx, chatID, messageID, text, keyboard)
}

// ShowMainMenu displays the main menu by sending a new message.
func (m *Manager) ShowMainMenu(ctx context.Context, chatID int64, topicID int, evaluator func(string) bool) (*telego.Message, error) {
	if m.config == nil || m.config.MainMenuID == "" {
		return nil, nil
	}
	return m.ShowMenu(ctx, chatID, topicID, m.config.MainMenuID, evaluator)
}

// EditToMainMenu edits an existing message to show the main menu.
func (m *Manager) EditToMainMenu(ctx context.Context, chatID int64, messageID int, evaluator func(string) bool) (*telego.Message, error) {
	if m.config == nil || m.config.MainMenuID == "" {
		return nil, nil
	}
	return m.EditToMenu(ctx, chatID, messageID, m.config.MainMenuID, evaluator)
}

// HandlePageChange handles pagination by changing the current page and refreshing the display.
func (m *Manager) HandlePageChange(ctx context.Context, chatID int64, messageID int, menuID string, page int, evaluator func(string) bool) (*telego.Message, error) {
	menu := m.GetMenu(menuID)
	if menu == nil {
		return nil, nil
	}

	menu.SetPage(page)
	return m.EditToMenu(ctx, chatID, messageID, menuID, evaluator)
}
