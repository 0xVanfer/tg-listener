# TGWrapper - Configuration-Driven Telegram Bot Library

A high-level wrapper library based on [telego](https://github.com/mymmrac/telego) that implements Telegram bot message sending, listening, and multi-turn conversation functionality through configuration-driven design.

## Features

-   **Configuration-Driven**: Define menus, conversation flows, and buttons through struct configurations
-   **Message Editing Priority**: Edit existing messages on callbacks to avoid keyboard accumulation
-   **Group Topic Support**: All message operations support MessageThreadID
-   **Link Preview Disabled**: All messages disable link preview by default
-   **High Extensibility**: Support for custom handlers, validators, and keyboard providers

## Installation

```bash
go get tgwrapper
```

## Quick Start

### 1. Create Configuration File (config.yaml)

```yaml
bot:
    token: "${BOT_TOKEN}"
    commands:
        - command: menu
          description: Show main menu

menus:
    main:
        id: main
        text: "🏠 *Main Menu*\n\nPlease select an option:"
        buttons:
            - - text: "📜 View Contracts"
                flow_id: contract_addresses
              - text: "📊 View Data"
                flow_id: view_data
            - - text: "⚙️ Settings"
                menu_id: settings

flows:
    contract_addresses:
        id: contract_addresses
        name: View Contract Addresses
        initial_step: select_chain
        steps:
            select_chain:
                prompt_text: "📜 *Contract Addresses*\n\nPlease select a chain:"
                keyboard:
                    type: dynamic
                    provider: getChains
                    columns: 2
                    add_back: true
                input_type: callback
                on_complete: showContractAddresses
```

### 2. Code Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    "tgwrapper"
    "tgwrapper/config"
    "tgwrapper/conv"
)

func main() {
    // Load configuration
    cfg := &config.Config{
        Bot: &config.BotConfig{
            Token: os.Getenv("BOT_TOKEN"),
            Commands: []config.CmdConfig{
                {Command: "menu", Description: "Show main menu"},
            },
        },
        MainMenuID: "main",
    }

    // Add menu configuration
    cfg.AddMenu(&config.MenuConfig{
        ID:   "main",
        Text: "🏠 *Main Menu*\n\nPlease select an option:",
        Buttons: [][]config.ButtonConfig{
            {
                {Text: "📜 View Contracts", FlowID: "contract_addresses"},
                {Text: "📊 View Data", FlowID: "view_data"},
            },
        },
    })

    // Add flow configuration
    cfg.AddFlow(&config.FlowConfig{
        ID:          "contract_addresses",
        Name:        "View Contract Addresses",
        InitialStep: "select_chain",
        Steps: map[string]*config.StepConfig{
            "select_chain": {
                PromptText: "📜 *Contract Addresses*\n\nPlease select a chain:",
                Keyboard: &config.KeyboardConfig{
                    Type:     config.KeyboardTypeDynamic,
                    Provider: "getChains",
                    Columns:  2,
                    AddBack:  true,
                },
                InputType:  config.InputTypeCallback,
                OnComplete: "showContractAddresses",
            },
        },
    })

    // Create Wrapper
    wrapper, err := tgwrapper.New(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Set authentication function
    wrapper.SetAuthFunc(func(ctx context.Context, userID int64, username string) bool {
        // Implement your authentication logic
        return true
    })

    // Register command handler
    wrapper.RegisterCommand("menu", func(ctx context.Context, msg telego.Message) error {
        return wrapper.ShowMainMenu(ctx, msg.Chat.ID, msg.MessageThreadID, 0)
    })

    // Register keyboard data provider
    wrapper.RegisterKeyboardProvider("getChains", func(ctx context.Context, c *conv.Conversation) []config.ButtonData {
        return []config.ButtonData{
            {Text: "Ethereum", Callback: "chain:1"},
            {Text: "BSC", Callback: "chain:56"},
            {Text: "Polygon", Callback: "chain:137"},
        }
    })

    // Register step handler
    wrapper.RegisterStepHandler("showContractAddresses", func(ctx context.Context, c *conv.Conversation) error {
        chainID := c.GetString("chain")
        // Handle logic...
        text := fmt.Sprintf("Contract addresses for chain %s:\n...", chainID)
        wrapper.EditMessage(ctx, c.ChatID, c.KeyboardMsgID, text)
        return wrapper.ShowMainMenu(ctx, c.ChatID, c.TopicID, 0)
    })

    // Start
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    if err := wrapper.Start(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Bot started...")
    <-ctx.Done()
    wrapper.Stop()
}
```

## Core Concepts

### Menu

Menus are used to display messages with inline keyboards, with pagination support.

```go
menu := &config.MenuConfig{
    ID:   "main",
    Text: "Welcome",
    Buttons: [][]config.ButtonConfig{
        {{Text: "Option 1", Callback: "opt1"}, {Text: "Option 2", Callback: "opt2"}},
    },
    Pages: []config.PageConfig{
        {ID: "page1", Name: "Page 1", Buttons: ...},
        {ID: "page2", Name: "Page 2", Buttons: ...},
    },
}
```

### Flow

Flows define the step sequence for multi-turn conversations.

```go
flow := &config.FlowConfig{
    ID:          "example_flow",
    Name:        "Example Flow",
    InitialStep: "step1",
    Steps: map[string]*config.StepConfig{
        "step1": {
            PromptText: "Please enter your name:",
            InputType:  config.InputTypeText,
            StoreAs:    "name",
            NextStep:   "step2",
        },
        "step2": {
            PromptText: "Please confirm:",
            Keyboard: &config.KeyboardConfig{
                Buttons: [][]config.ButtonConfig{
                    {{Text: "Confirm", Callback: "confirm"}},
                },
                AddBack: true,
            },
            InputType:  config.InputTypeCallback,
            OnComplete: "handleComplete",
        },
    },
}
```

### Keyboard

Supports both static and dynamic keyboards:

```go
// Static keyboard
kb := tgwrapper.NewKeyboard().
    Row(tgwrapper.Button("Option 1", "opt1"), tgwrapper.Button("Option 2", "opt2")).
    Back().
    Build()

// Dynamic keyboard - register provider
wrapper.RegisterKeyboardProvider("getOptions", func(ctx context.Context, c *conv.Conversation) []config.ButtonData {
    return []config.ButtonData{
        {Text: "Dynamic Option 1", Callback: "dyn1"},
        {Text: "Dynamic Option 2", Callback: "dyn2"},
    }
})
```

### Message Builder

Used to build formatted messages:

```go
b := tgwrapper.NewBuilder()
b.Header("Title")
b.KeyValue("Name", "Value")
b.KeyValueLink("Link", "Click here", "https://example.com")
b.List("Item 1", "Item 2", "Item 3")
text, entities := b.Build()
```

## Extension Points

### Custom Handlers

```go
// Step completion handler
wrapper.RegisterStepHandler("myHandler", func(ctx context.Context, c *conv.Conversation) error {
    // Handle logic
    return nil
})

// Keyboard data provider
wrapper.RegisterKeyboardProvider("myProvider", func(ctx context.Context, c *conv.Conversation) []config.ButtonData {
    return []config.ButtonData{...}
})

// Validator
wrapper.RegisterValidator("myValidator", func(value string, c *conv.Conversation) error {
    if len(value) < 3 {
        return errors.New("input is too short")
    }
    return nil
})
```

### Hook Functions

```go
wrapper.OnConversationStart(func(ctx context.Context, c *conv.Conversation) {
    log.Printf("Conversation started: %s", c.FlowID)
})

wrapper.OnConversationEnd(func(ctx context.Context, c *conv.Conversation) {
    log.Printf("Conversation ended: %s", c.FlowID)
})

wrapper.OnStepChange(func(ctx context.Context, c *conv.Conversation, from, to string) {
    log.Printf("Step changed: %s -> %s", from, to)
})
```

## Directory Structure

```
tgwrapper/
├── config/           # Configuration struct definitions
│   ├── bot.go        # Bot configuration
│   ├── menu.go       # Menu configuration
│   ├── flow.go       # Conversation flow configuration
│   ├── keyboard.go   # Keyboard configuration
│   ├── config.go     # Complete configuration
│   └── errors.go     # Error definitions
├── core/             # Core functionality
│   ├── bot.go        # Bot wrapper
│   ├── keyboard.go   # Keyboard builder
│   ├── builder.go    # Message formatting
│   └── message.go    # Message processing utilities
├── conv/             # Conversation management
│   ├── conversation.go  # Conversation state
│   └── engine.go        # Flow engine
├── handler/          # Handlers
│   └── router.go     # Route dispatching
├── menu/             # Menu system
│   └── menu.go       # Menu management
├── examples/         # Configuration examples
│   ├── config.yaml   # YAML configuration example
│   └── config.json   # JSON configuration example
├── tgwrapper.go      # Entry point
├── go.mod
└── README.md
```

## Configuration Examples

The library supports both YAML and JSON configuration formats. See the `examples/` directory for complete configuration examples:

-   [config.yaml](examples/config.yaml) - Complete YAML configuration with all options documented
-   [config.json](examples/config.json) - Complete JSON configuration with all options

### Validation Types

The library supports several built-in validation types:

| Type      | Description                | Parameters                |
| --------- | -------------------------- | ------------------------- |
| `number`  | Validates numeric input    | `min`, `max`              |
| `email`   | Validates email format     | -                         |
| `address` | Validates Ethereum address | -                         |
| `regex`   | Custom regex pattern       | `pattern`                 |
| `custom`  | Custom validator function  | `custom` (validator name) |

### Input Types

| Type       | Description                      |
| ---------- | -------------------------------- |
| `text`     | Accepts text message input       |
| `callback` | Accepts inline keyboard callback |
| `any`      | Accepts both text and callback   |

### Keyboard Types

| Type      | Description                                         |
| --------- | --------------------------------------------------- |
| `static`  | Static buttons defined in configuration             |
| `dynamic` | Buttons generated by a registered provider function |

## Best Practices

1. **Message Editing Priority**: Prefer `EditMessage` over `SendTo` in callback handlers
2. **Topic Support**: Always pass `TopicID` to support group topics
3. **Error Handling**: Provide user-friendly error messages
4. **Configuration Separation**: Separate configuration from code using YAML or JSON configuration files
5. **Graceful Degradation**: Return nil instead of error when bot is not initialized

## API Reference

### Wrapper Methods

| Method                                            | Description                 |
| ------------------------------------------------- | --------------------------- |
| `New(cfg)`                                        | Create new wrapper instance |
| `Start(ctx)`                                      | Start the bot               |
| `Stop()`                                          | Stop the bot                |
| `SetAuthFunc(fn)`                                 | Set authentication function |
| `RegisterCommand(cmd, handler)`                   | Register command handler    |
| `RegisterCallback(data, handler)`                 | Register callback handler   |
| `RegisterStepHandler(name, handler)`              | Register step handler       |
| `RegisterKeyboardProvider(name, provider)`        | Register keyboard provider  |
| `RegisterValidator(name, validator)`              | Register validator          |
| `ShowMainMenu(ctx, chatID, topicID, msgID)`       | Show main menu              |
| `StartFlow(ctx, chatID, userID, topicID, flowID)` | Start conversation flow     |
| `EndConversation(ctx, userID, chatID)`            | End conversation            |

### Builder Methods

| Method                         | Description                         |
| ------------------------------ | ----------------------------------- |
| `Header(text)`                 | Add bold header                     |
| `Line(text)`                   | Add text line                       |
| `Ln()`                         | Add empty line                      |
| `Bold(text)`                   | Add bold text                       |
| `Italic(text)`                 | Add italic text                     |
| `Code(text)`                   | Add monospace text                  |
| `CodeBlock(text, lang)`        | Add code block                      |
| `KeyValue(key, value)`         | Add key-value pair                  |
| `KeyValueCode(key, value)`     | Add key-value with code value       |
| `KeyValueLink(key, text, url)` | Add key-value with link             |
| `List(items...)`               | Add bullet list                     |
| `Link(text, url)`              | Add hyperlink                       |
| `Build()`                      | Build and return text with entities |

## License

MIT License
