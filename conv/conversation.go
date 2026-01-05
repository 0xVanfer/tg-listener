// Package conv provides conversation management functionality.
// It handles multi-step user interactions through state tracking and session management.
package conv

import (
	"context"
	"sync"
	"time"
)

// ConversationState represents the current state of a conversation.
type ConversationState int

const (
	// StateIdle indicates the conversation is idle with no active flow.
	StateIdle ConversationState = iota
	// StateWaiting indicates the conversation is waiting for user input.
	StateWaiting
	// StateProcessing indicates the conversation is processing user input.
	StateProcessing
	// StateCompleted indicates the conversation has been completed successfully.
	StateCompleted
	// StateCancelled indicates the conversation was cancelled by the user.
	StateCancelled
)

// Conversation represents a conversation session with a user.
// It tracks the current flow, step, collected data, and session metadata.
type Conversation struct {
	UserID        int64                  // User ID of the participant
	ChatID        int64                  // Chat ID where the conversation takes place
	TopicID       int                    // Topic ID for group topic support
	FlowID        string                 // Current flow ID being executed
	StepID        string                 // Current step ID within the flow
	State         ConversationState      // Current conversation state
	Data          map[string]interface{} // Key-value storage for collected data
	KeyboardMsgID int                    // Message ID of the last keyboard message (for editing)
	CreatedAt     time.Time              // Timestamp when conversation was created
	UpdatedAt     time.Time              // Timestamp of last update
	ExpiresAt     time.Time              // Expiration timestamp for auto-cleanup
	History       []HistoryEntry         // History of steps and inputs

	mu sync.RWMutex // Mutex for thread-safe operations
}

// HistoryEntry represents a single step in the conversation history.
type HistoryEntry struct {
	StepID    string    // ID of the step that was executed
	Input     string    // User input received at this step
	Timestamp time.Time // When this entry was recorded
}

// NewConversation creates a new conversation session.
// Parameters:
//   - userID: Telegram user ID
//   - chatID: Telegram chat ID
//   - topicID: Message thread ID for group topics (0 if not in topic)
//   - flowID: ID of the flow to start
//   - stepID: Initial step ID
//   - ttl: Time-to-live duration for auto-expiration
func NewConversation(userID, chatID int64, topicID int, flowID, stepID string, ttl time.Duration) *Conversation {
	now := time.Now()
	return &Conversation{
		UserID:    userID,
		ChatID:    chatID,
		TopicID:   topicID,
		FlowID:    flowID,
		StepID:    stepID,
		State:     StateWaiting,
		Data:      make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(ttl),
		History:   make([]HistoryEntry, 0),
	}
}

// Set stores a value in the conversation data.
// Thread-safe for concurrent access.
func (c *Conversation) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data[key] = value
	c.UpdatedAt = time.Now()
}

// Get retrieves a value from the conversation data.
// Returns the value and a boolean indicating if the key exists.
func (c *Conversation) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.Data[key]
	return v, ok
}

// GetString retrieves a string value from the conversation data.
// Returns empty string if key doesn't exist or value is not a string.
func (c *Conversation) GetString(key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetInt retrieves an integer value from the conversation data.
// Handles int, int64, and float64 types. Returns 0 if not found or invalid type.
func (c *Conversation) GetInt(key string) int {
	v, ok := c.Get(key)
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// SetStep updates the current step ID.
// Thread-safe and automatically updates the UpdatedAt timestamp.
func (c *Conversation) SetStep(stepID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.StepID = stepID
	c.UpdatedAt = time.Now()
}

// SetKeyboardMsgID sets the message ID of the keyboard message.
// Used for editing the keyboard message instead of sending new ones.
func (c *Conversation) SetKeyboardMsgID(msgID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.KeyboardMsgID = msgID
}

// AddHistory adds a new entry to the conversation history.
// Records the step ID, user input, and timestamp.
func (c *Conversation) AddHistory(stepID, input string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.History = append(c.History, HistoryEntry{
		StepID:    stepID,
		Input:     input,
		Timestamp: time.Now(),
	})
}

// GetPreviousStep returns the step ID before the current one.
// Useful for implementing back navigation. Returns empty string if no previous step.
func (c *Conversation) GetPreviousStep() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.History) >= 2 {
		return c.History[len(c.History)-2].StepID
	}
	return ""
}

// IsExpired checks if the conversation has expired.
// Returns true if current time is after the expiration time.
func (c *Conversation) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().After(c.ExpiresAt)
}

// Refresh extends the conversation expiration time.
// Typically called after each user interaction to keep the conversation alive.
func (c *Conversation) Refresh(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ExpiresAt = time.Now().Add(ttl)
	c.UpdatedAt = time.Now()
}

// Complete marks the conversation as successfully completed.
func (c *Conversation) Complete() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = StateCompleted
	c.UpdatedAt = time.Now()
}

// Cancel marks the conversation as cancelled.
func (c *Conversation) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = StateCancelled
	c.UpdatedAt = time.Now()
}

// conversationKey generates a unique key for the conversation.
// Uses combination of user ID and chat ID for uniqueness.
func conversationKey(userID, chatID int64) string {
	return string(rune(userID)) + ":" + string(rune(chatID))
}

// Manager handles conversation lifecycle and provides
// methods for starting, ending, and querying conversations.
type Manager struct {
	conversations map[string]*Conversation // Active conversations indexed by key
	defaultTTL    time.Duration            // Default time-to-live for new conversations
	mu            sync.RWMutex             // Mutex for thread-safe operations

	// Lifecycle callback functions
	onStart      func(ctx context.Context, c *Conversation)                  // Called when conversation starts
	onEnd        func(ctx context.Context, c *Conversation)                  // Called when conversation ends
	onStepChange func(ctx context.Context, c *Conversation, from, to string) // Called when step changes
}

// NewManager creates a new conversation manager.
// defaultTTL specifies the default expiration time for conversations (default: 30 minutes).
func NewManager(defaultTTL time.Duration) *Manager {
	if defaultTTL <= 0 {
		defaultTTL = 30 * time.Minute
	}
	return &Manager{
		conversations: make(map[string]*Conversation),
		defaultTTL:    defaultTTL,
	}
}

// SetOnStart sets the callback function for when a conversation starts.
func (m *Manager) SetOnStart(fn func(ctx context.Context, c *Conversation)) {
	m.onStart = fn
}

// SetOnEnd sets the callback function for when a conversation ends.
func (m *Manager) SetOnEnd(fn func(ctx context.Context, c *Conversation)) {
	m.onEnd = fn
}

// SetOnStepChange sets the callback function for when step changes.
func (m *Manager) SetOnStepChange(fn func(ctx context.Context, c *Conversation, from, to string)) {
	m.onStepChange = fn
}

// Start begins a new conversation for a user in a chat.
// If a conversation already exists for this user/chat, it will be ended first.
// Parameters:
//   - ctx: Context for cancellation and callbacks
//   - userID: Telegram user ID
//   - chatID: Telegram chat ID
//   - topicID: Message thread ID (0 if not in topic)
//   - flowID: ID of the flow to execute
//   - initialStep: Starting step ID
//   - ttl: Time-to-live (uses default if <= 0)
func (m *Manager) Start(ctx context.Context, userID, chatID int64, topicID int, flowID, initialStep string, ttl time.Duration) (*Conversation, error) {
	if ttl <= 0 {
		ttl = m.defaultTTL
	}

	key := conversationKey(userID, chatID)

	m.mu.Lock()
	// End existing conversation if present
	if existing, ok := m.conversations[key]; ok {
		if m.onEnd != nil {
			m.onEnd(ctx, existing)
		}
	}

	conv := NewConversation(userID, chatID, topicID, flowID, initialStep, ttl)
	m.conversations[key] = conv
	m.mu.Unlock()

	if m.onStart != nil {
		m.onStart(ctx, conv)
	}

	return conv, nil
}

// Get retrieves an active conversation for a user/chat.
// Returns nil if no conversation exists or if it has expired.
func (m *Manager) Get(userID, chatID int64) *Conversation {
	key := conversationKey(userID, chatID)

	m.mu.RLock()
	conv, ok := m.conversations[key]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	// Auto-cleanup expired conversations
	if conv.IsExpired() {
		m.End(context.Background(), userID, chatID)
		return nil
	}

	return conv
}

// End terminates a conversation and removes it from the manager.
// Triggers the onEnd callback if set.
func (m *Manager) End(ctx context.Context, userID, chatID int64) {
	key := conversationKey(userID, chatID)

	m.mu.Lock()
	conv, ok := m.conversations[key]
	if ok {
		delete(m.conversations, key)
	}
	m.mu.Unlock()

	if ok && m.onEnd != nil {
		m.onEnd(ctx, conv)
	}
}

// ChangeStep changes the current step of a conversation.
// Triggers the onStepChange callback with old and new step IDs.
func (m *Manager) ChangeStep(ctx context.Context, userID, chatID int64, newStep string) {
	conv := m.Get(userID, chatID)
	if conv == nil {
		return
	}

	oldStep := conv.StepID
	conv.SetStep(newStep)

	if m.onStepChange != nil {
		m.onStepChange(ctx, conv, oldStep, newStep)
	}
}

// Cleanup removes all expired conversations.
// Returns the number of conversations that were cleaned up.
func (m *Manager) Cleanup(ctx context.Context) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for key, conv := range m.conversations {
		if conv.IsExpired() {
			if m.onEnd != nil {
				m.onEnd(ctx, conv)
			}
			delete(m.conversations, key)
			count++
		}
	}

	return count
}

// StartCleanupTask starts a background goroutine that periodically cleans up expired conversations.
// The goroutine will stop when the context is cancelled.
func (m *Manager) StartCleanupTask(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.Cleanup(ctx)
			}
		}
	}()
}

// Count returns the number of active conversations.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conversations)
}
