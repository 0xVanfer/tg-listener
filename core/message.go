// Package core provides message processing functionality.
package core

import (
	"unicode/utf16"

	"github.com/mymmrac/telego"
)

// MaxMessageLength is the maximum character length for a Telegram message.
const MaxMessageLength = 4096

// SplitMessage splits a long message into multiple parts.
// It preserves message entity offsets correctly across splits.
// Attempts to split at newlines for cleaner breaks.
func SplitMessage(text string, entities []telego.MessageEntity) []MessagePart {
	// If text is within limit, return as single part
	textRunes := []rune(text)
	if len(textRunes) <= MaxMessageLength {
		return []MessagePart{{Text: text, Entities: entities}}
	}

	var parts []MessagePart
	start := 0

	for start < len(textRunes) {
		end := min(start+MaxMessageLength, len(textRunes))

		// Try to split at a newline for cleaner breaks
		if end < len(textRunes) {
			for i := end - 1; i > start+MaxMessageLength/2; i-- {
				if textRunes[i] == '\n' {
					end = i + 1
					break
				}
			}
		}

		partText := string(textRunes[start:end])
		partEntities := adjustEntitiesForPart(entities, start, end)

		parts = append(parts, MessagePart{
			Text:     partText,
			Entities: partEntities,
		})

		start = end
	}

	return parts
}

// MessagePart represents a portion of a split message.
type MessagePart struct {
	Text     string                 // Text content for this part
	Entities []telego.MessageEntity // Adjusted entities for this part
}

// adjustEntitiesForPart adjusts entity offsets for a message part.
// Entities that span across the split point are truncated.
func adjustEntitiesForPart(entities []telego.MessageEntity, start, end int) []telego.MessageEntity {
	var result []telego.MessageEntity

	for _, entity := range entities {
		entityStart := int(entity.Offset)
		entityEnd := entityStart + int(entity.Length)

		// Check if entity is within this part
		if entityEnd <= start || entityStart >= end {
			continue
		}

		// Adjust entity range to fit within part
		newStart := max(entityStart-start, 0)
		newEnd := min(entityEnd-start, end-start)

		newEntity := entity
		newEntity.Offset = newStart
		newEntity.Length = newEnd - newStart

		if newEntity.Length > 0 {
			result = append(result, newEntity)
		}
	}

	return result
}

// UTF16Length calculates the UTF-16 encoded length of a string.
// Telegram uses UTF-16 for offset calculation in message entities.
func UTF16Length(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// UTF16Offset converts a rune offset to UTF-16 offset.
// Useful for calculating entity offsets from string positions.
func UTF16Offset(s string, runeOffset int) int {
	runes := []rune(s)
	if runeOffset > len(runes) {
		runeOffset = len(runes)
	}
	return len(utf16.Encode(runes[:runeOffset]))
}
