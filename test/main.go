// Package main provides a test application for tg-listener library.
// This test script demonstrates loading configuration from YAML file
// and registering custom handlers for testing all features.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	tgwrapper "github.com/0xVanfer/tg-listener"
	"github.com/0xVanfer/tg-listener/config"
	"github.com/0xVanfer/tg-listener/conv"
	"github.com/0xVanfer/tg-listener/core"
)

// Global wrapper reference for step handlers
// Note: In production, consider using dependency injection or context values
var globalWrapper *tgwrapper.Wrapper

func main() {
	// Get config file path from command line or use default
	configPath := "test/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Load configuration from file
	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Get bot token from environment variable if not set in config
	if cfg.Bot.Token == "" || cfg.Bot.Token == "${BOT_TOKEN}" {
		token := os.Getenv("BOT_TOKEN")
		if token == "" {
			log.Fatal("BOT_TOKEN environment variable is not set")
		}
		cfg.Bot.Token = token
	}

	// Initialize the wrapper
	wrapper, err := tgwrapper.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create wrapper: %v", err)
	}

	// Store wrapper reference for step handlers
	globalWrapper = wrapper

	// Set authentication function - allows all users for testing
	wrapper.SetAuthFunc(func(ctx context.Context, userID int64, username string) bool {
		log.Printf("✅ User authenticated: userID=%d, username=%s", userID, username)
		return true
	})

	// Register all handlers
	registerHandlers(wrapper)

	// Create context with interrupt signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start the bot
	log.Println("🚀 Starting test Bot...")
	if err := wrapper.Start(ctx); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	log.Println("✅ Bot started!")
	log.Println("📋 Available commands:")
	log.Println("   /menu - Show main menu")
	log.Println("   /help - Show help information")
	log.Println("   /test - Quick test")
	log.Println("")
	log.Println("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("🛑 Stopping Bot...")

	// Cleanup
	if cfg.Bot.DeleteCommandsOnExit {
		log.Println("🗑️  Deleting registered commands...")
		_ = wrapper.Bot().Telego().DeleteMyCommands(context.Background(), nil)
	}

	wrapper.Stop()
	log.Println("👋 Bot stopped")
}

// loadConfig loads configuration from YAML or JSON file.
func loadConfig(path string) (*config.Config, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	log.Printf("📂 Loading config from: %s", absPath)

	cfg, err := config.LoadFromFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	log.Printf("✅ Config loaded: %d menus, %d flows", len(cfg.Menus), len(cfg.Flows))
	return cfg, nil
}

// registerHandlers registers all custom handlers for the test bot.
func registerHandlers(wrapper *tgwrapper.Wrapper) {
	// ===========================================
	// Command Handlers
	// ===========================================

	// /menu command - displays the main menu
	wrapper.RegisterCommand("menu", func(ctx context.Context, msg telego.Message) error {
		log.Printf("📨 /menu from user %d", msg.From.ID)
		return wrapper.ShowMainMenu(ctx, msg.Chat.ID, msg.MessageThreadID, 0)
	})

	// /help command - displays help information
	wrapper.RegisterCommand("help", func(ctx context.Context, msg telego.Message) error {
		log.Printf("📨 /help from user %d", msg.From.ID)

		b := core.NewBuilder()
		b.Header("❓ Help Information")
		b.Line("This is the tg-listener feature test Bot.")
		b.Ln()
		b.BoldLine("🧪 Test Features:")
		b.List(
			"Multi-level menu navigation (4 levels)",
			"Link buttons",
			"Text input and keyboard interaction",
			"Back to previous level/main menu",
			"Real external API calls",
			"Dynamic button data from API",
		)
		b.Ln()
		b.Line("Configuration loaded from: test/config.yaml")
		text, entities := b.Build()

		_, err := wrapper.SendTo(ctx, msg.Chat.ID, msg.MessageThreadID, text, entities...)
		return err
	})

	// /test command - displays quick test info
	wrapper.RegisterCommand("test", func(ctx context.Context, msg telego.Message) error {
		log.Printf("📨 /test from user %d", msg.From.ID)

		b := core.NewBuilder()
		b.Header("🧪 Quick Test")
		b.KeyValue("User ID", fmt.Sprintf("%d", msg.From.ID))
		b.KeyValue("Username", msg.From.Username)
		b.KeyValue("Chat Type", string(msg.Chat.Type))
		b.KeyValue("Time", time.Now().Format("2006-01-02 15:04:05"))
		text, entities := b.Build()

		kb := core.NewKeyboard()
		kb.Row(core.Button("📱 Open Main Menu", "main_menu"))
		kb.Row(core.Button("🌐 Test API", "flow:api_test"))

		_, err := wrapper.SendToWithKeyboard(ctx, msg.Chat.ID, msg.MessageThreadID, text, kb.Build(), entities...)
		return err
	})

	// ===========================================
	// Callback Handlers
	// ===========================================

	// show_info callback
	wrapper.RegisterCallback("show_info", func(ctx context.Context, query telego.CallbackQuery) error {
		_ = wrapper.AnswerCallback(ctx, query.ID, "ℹ️ This is Level 3-B information")
		return nil
	})

	// level_4_action callback
	wrapper.RegisterCallback("level_4_action", func(ctx context.Context, query telego.CallbackQuery) error {
		_ = wrapper.AnswerCallback(ctx, query.ID, "🎉 Action executed!")

		chatID := query.Message.GetChat().ID
		msgID := query.Message.GetMessageID()

		b := core.NewBuilder()
		b.Header("🎯 Action Executed Successfully")
		b.Line("You have successfully executed the Level 4 action!")
		b.Ln()
		b.KeyValue("Time", time.Now().Format("15:04:05"))
		b.KeyValue("User", query.From.Username)
		text, entities := b.Build()

		kb := core.NewKeyboard()
		kb.Row(core.Button("⬅️ Back to Level 4", "menu:level_4"))
		kb.Row(core.Button("🏠 Main Menu", "main_menu"))

		_, err := wrapper.EditMessageKeyboard(ctx, chatID, msgID, text, kb.Build(), entities...)
		return err
	})

	// ===========================================
	// Keyboard Providers
	// ===========================================

	// getCryptoPrices - fetches real crypto prices from API
	wrapper.RegisterKeyboardProvider("getCryptoPrices", func(ctx context.Context, c *conv.Conversation) []config.ButtonData {
		log.Println("🔄 Fetching cryptocurrency prices...")

		prices, err := fetchCryptoPrices()
		if err != nil {
			log.Printf("❌ Failed to fetch prices: %v", err)
			return []config.ButtonData{
				{Text: "❌ Load failed", Callback: "retry"},
			}
		}

		var buttons []config.ButtonData
		for _, p := range prices {
			text := fmt.Sprintf("%s $%.2f", strings.ToUpper(p.Symbol), p.Price)
			buttons = append(buttons, config.ButtonData{
				Text:     text,
				Callback: p.ID,
			})
		}

		log.Printf("✅ Fetched %d cryptocurrency prices", len(buttons))
		return buttons
	})

	// ===========================================
	// Step Handlers
	// ===========================================

	// handleTextInputAction - processes text input flow
	wrapper.RegisterStepHandler("handleTextInputAction", func(ctx context.Context, c *conv.Conversation) error {
		action := core.ParseCallbackData(c.GetString("selectedAction"), "action:")
		userName := c.GetString("userName")
		userMessage := c.GetString("userMessage")

		log.Printf("📝 Text input: name=%s, message=%s, action=%s", userName, userMessage, action)

		if action == "restart" {
			c.SetStep("input_name")
			c.Set("userName", "")
			c.Set("userMessage", "")
			return nil
		}

		// Show result
		b := core.NewBuilder()
		b.Header("📤 Your Input")
		b.KeyValue("Name", userName)
		b.KeyValue("Message", userMessage)
		b.Ln()
		b.Line("✅ Text input test completed!")
		text, entities := b.Build()

		kb := core.NewKeyboard()
		kb.Row(core.Button("🔄 Try Again", "flow:text_input"))
		kb.Row(core.Button("🏠 Main Menu", "main_menu"))

		if globalWrapper != nil && c.KeyboardMsgID > 0 {
			_, _ = globalWrapper.EditMessageKeyboard(ctx, c.ChatID, c.KeyboardMsgID, text, kb.Build(), entities...)
		}

		return nil
	})

	// executeAPICall - executes API calls
	wrapper.RegisterStepHandler("executeAPICall", func(ctx context.Context, c *conv.Conversation) error {
		api := core.ParseCallbackData(c.GetString("selectedAPI"), "api:")
		log.Printf("🌐 API call: %s", api)

		var result string
		var err error

		switch api {
		case "cat":
			result, err = fetchCatImage()
		case "dog":
			result, err = fetchDogImage()
		case "joke":
			result, err = fetchJoke()
		case "crypto":
			result, err = fetchCryptoSummary()
		default:
			result = "Unknown API"
		}

		if err != nil {
			result = fmt.Sprintf("❌ Error: %v", err)
		}

		b := core.NewBuilder()
		b.Header("🌐 API Result")
		b.KeyValue("API", api)
		b.Ln()
		b.Line(result)
		text, entities := b.Build()

		kb := core.NewKeyboard()
		kb.Row(core.Button("🔄 Try Another", "flow:api_test"))
		kb.Row(core.Button("🏠 Main Menu", "main_menu"))

		if globalWrapper != nil && c.KeyboardMsgID > 0 {
			_, _ = globalWrapper.EditMessageKeyboard(ctx, c.ChatID, c.KeyboardMsgID, text, kb.Build(), entities...)
		}

		return nil
	})

	// showCryptoDetail - shows crypto details
	wrapper.RegisterStepHandler("showCryptoDetail", func(ctx context.Context, c *conv.Conversation) error {
		crypto := core.ParseCallbackData(c.GetString("selectedCrypto"), "crypto:")
		log.Printf("🎯 Selected crypto: %s", crypto)

		b := core.NewBuilder()
		b.Header("📊 Cryptocurrency Details")
		b.KeyValue("Selected", strings.ToUpper(crypto))
		b.Ln()
		b.Line("You selected this cryptocurrency from dynamic buttons!")
		b.Line("The button data was fetched from CoinGecko API.")
		text, entities := b.Build()

		kb := core.NewKeyboard()
		kb.Row(core.Button("🔄 Select Another", "flow:dynamic_buttons"))
		kb.Row(core.Button("🏠 Main Menu", "main_menu"))

		if globalWrapper != nil && c.KeyboardMsgID > 0 {
			_, _ = globalWrapper.EditMessageKeyboard(ctx, c.ChatID, c.KeyboardMsgID, text, kb.Build(), entities...)
		}

		return nil
	})

	// Set lifecycle hooks
	wrapper.OnConversationStart(func(ctx context.Context, c *conv.Conversation) {
		log.Printf("🟢 Conversation started: flow=%s, user=%d", c.FlowID, c.UserID)
	})

	wrapper.OnConversationEnd(func(ctx context.Context, c *conv.Conversation) {
		log.Printf("🔴 Conversation ended: flow=%s, user=%d", c.FlowID, c.UserID)
	})
}

// ===========================================
// API Helper Functions
// ===========================================

// CryptoPrice represents cryptocurrency price data from CoinGecko API.
type CryptoPrice struct {
	ID     string  `json:"id"`
	Symbol string  `json:"symbol"`
	Price  float64 `json:"current_price"`
}

func fetchCryptoPrices() ([]CryptoPrice, error) {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=6&page=1")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var prices []CryptoPrice
	if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, err
	}

	return prices, nil
}

func fetchCatImage() (string, error) {
	resp, err := http.Get("https://api.thecatapi.com/v1/images/search")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result []struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result) > 0 {
		return "🐱 " + result[0].URL, nil
	}
	return "No image found", nil
}

func fetchDogImage() (string, error) {
	resp, err := http.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return "🐕 " + result.Message, nil
}

func fetchJoke() (string, error) {
	req, _ := http.NewRequest("GET", "https://icanhazdadjoke.com/", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Joke string `json:"joke"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return "😂 " + result.Joke, nil
}

func fetchCryptoSummary() (string, error) {
	prices, err := fetchCryptoPrices()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("📊 Top Cryptocurrencies:\n\n")
	for _, p := range prices {
		sb.WriteString(fmt.Sprintf("• %s: $%.2f\n", strings.ToUpper(p.Symbol), p.Price))
	}

	return sb.String(), nil
}
