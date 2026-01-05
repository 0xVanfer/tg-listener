// Package core provides message formatting builder functionality.
package core

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/mymmrac/telego"
)

// Builder is a message builder for constructing formatted Telegram messages.
// It uses message entities for formatting instead of parse mode,
// which provides more precise control over text formatting.
type Builder struct {
	text     strings.Builder        // Text content accumulator
	entities []telego.MessageEntity // Formatting entities
}

// NewBuilder creates a new message builder instance.
func NewBuilder() *Builder {
	return &Builder{
		entities: make([]telego.MessageEntity, 0),
	}
}

// getCurrentOffset returns the current UTF-16 offset.
// Telegram uses UTF-16 for entity offset calculation.
func (b *Builder) getCurrentOffset() int {
	return len(utf16.Encode([]rune(b.text.String())))
}

// addEntity adds a formatting entity and appends the text.
func (b *Builder) addEntity(entityType string, text string) *Builder {
	offset := b.getCurrentOffset()
	length := len(utf16.Encode([]rune(text)))

	b.entities = append(b.entities, telego.MessageEntity{
		Type:   entityType,
		Offset: offset,
		Length: length,
	})

	b.text.WriteString(text)
	return b
}

// Text appends plain text without formatting.
func (b *Builder) Text(text string) *Builder {
	b.text.WriteString(text)
	return b
}

// Line appends text followed by a newline.
func (b *Builder) Line(text string) *Builder {
	b.text.WriteString(text)
	b.text.WriteString("\n")
	return b
}

// Ln appends a newline character.
func (b *Builder) Ln() *Builder {
	b.text.WriteString("\n")
	return b
}

// Bold appends bold formatted text.
func (b *Builder) Bold(text string) *Builder {
	return b.addEntity("bold", text)
}

// BoldLine appends bold text followed by a newline.
func (b *Builder) BoldLine(text string) *Builder {
	b.Bold(text)
	b.Ln()
	return b
}

// Italic appends italic formatted text.
func (b *Builder) Italic(text string) *Builder {
	return b.addEntity("italic", text)
}

// Code appends inline code formatted text.
func (b *Builder) Code(text string) *Builder {
	return b.addEntity("code", text)
}

// Pre appends a code block with optional language highlighting.
func (b *Builder) Pre(text string, language string) *Builder {
	offset := b.getCurrentOffset()
	length := len(utf16.Encode([]rune(text)))

	entity := telego.MessageEntity{
		Type:     "pre",
		Offset:   offset,
		Length:   length,
		Language: language,
	}

	b.entities = append(b.entities, entity)
	b.text.WriteString(text)
	return b
}

// Link appends a text link (clickable text with URL).
func (b *Builder) Link(text, url string) *Builder {
	offset := b.getCurrentOffset()
	length := len(utf16.Encode([]rune(text)))

	b.entities = append(b.entities, telego.MessageEntity{
		Type:   "text_link",
		Offset: offset,
		Length: length,
		URL:    url,
	})

	b.text.WriteString(text)
	return b
}

// Mention appends an @ mention.
func (b *Builder) Mention(username string) *Builder {
	if !strings.HasPrefix(username, "@") {
		username = "@" + username
	}
	return b.addEntity("mention", username)
}

// UserMention appends a user mention by ID (clickable username).
func (b *Builder) UserMention(text string, userID int64) *Builder {
	offset := b.getCurrentOffset()
	length := len(utf16.Encode([]rune(text)))

	b.entities = append(b.entities, telego.MessageEntity{
		Type:   "text_mention",
		Offset: offset,
		Length: length,
		User: &telego.User{
			ID: userID,
		},
	})

	b.text.WriteString(text)
	return b
}

// Strikethrough appends strikethrough formatted text.
func (b *Builder) Strikethrough(text string) *Builder {
	return b.addEntity("strikethrough", text)
}

// Underline appends underlined text.
func (b *Builder) Underline(text string) *Builder {
	return b.addEntity("underline", text)
}

// Spoiler appends spoiler (hidden) text.
func (b *Builder) Spoiler(text string) *Builder {
	return b.addEntity("spoiler", text)
}

// Header appends a header (bold + newline + blank line).
func (b *Builder) Header(text string) *Builder {
	b.Bold(text)
	b.Ln()
	b.Ln()
	return b
}

// SubHeader appends a sub-header (bold + newline).
func (b *Builder) SubHeader(text string) *Builder {
	b.Bold(text)
	b.Ln()
	return b
}

// KeyValue appends a key-value pair (bold key + plain value).
func (b *Builder) KeyValue(key, value string) *Builder {
	b.Bold(key + ": ")
	b.Line(value)
	return b
}

// KeyValueCode appends a key-value pair with code-formatted value.
func (b *Builder) KeyValueCode(key, value string) *Builder {
	b.Bold(key + ": ")
	b.Code(value)
	b.Ln()
	return b
}

// List appends an unordered list with bullet points.
func (b *Builder) List(items ...string) *Builder {
	for _, item := range items {
		b.Text("• ")
		b.Line(item)
	}
	return b
}

// NumberedList appends an ordered (numbered) list.
// NumberedList appends an ordered (numbered) list.
func (b *Builder) NumberedList(items ...string) *Builder {
	for i, item := range items {
		b.Text(fmt.Sprintf("%d. ", i+1))
		b.Line(item)
	}
	return b
}

// Separator appends a horizontal separator line.
func (b *Builder) Separator() *Builder {
	b.Line("━━━━━━━━━━━━━━━")
	return b
}

// Build returns the final text and entities.
func (b *Builder) Build() (string, []telego.MessageEntity) {
	return b.text.String(), b.entities
}

// String returns the plain text content without entities.
func (b *Builder) String() string {
	return b.text.String()
}
