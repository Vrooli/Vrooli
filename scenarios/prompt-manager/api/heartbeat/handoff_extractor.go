package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// HandoffExtractor extracts a structured handoff from raw run event data.
type HandoffExtractor interface {
	Extract(ctx context.Context, eventsJSON []byte) (string, error)
}

// SentinelExtractor scans the last assistant message for a ## HANDOFF section.
type SentinelExtractor struct{}

// NewSentinelExtractor creates a new SentinelExtractor.
func NewSentinelExtractor() *SentinelExtractor {
	return &SentinelExtractor{}
}

// handoffHeaderRe matches ## HANDOFF or ## 🔄 HANDOFF (case-insensitive, flexible whitespace).
var handoffHeaderRe = regexp.MustCompile(`(?im)^##\s+(?:🔄\s*)?handoff\s*$`)

// nextH2Re matches the start of the next ## section.
var nextH2Re = regexp.MustCompile(`(?m)^##\s+`)

// eventMessage represents a message within an event.
type eventMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// runEvent represents a single event from GetRunEvents.
type runEvent struct {
	EventType string       `json:"event_type"`
	Message   eventMessage `json:"message"`
}

// eventsEnvelope wraps the events array from GetRunEvents response.
type eventsEnvelope struct {
	Events []runEvent `json:"events"`
}

// Extract parses eventsJSON, finds the last assistant message, and extracts the HANDOFF section.
func (s *SentinelExtractor) Extract(_ context.Context, eventsJSON []byte) (string, error) {
	var envelope eventsEnvelope
	if err := json.Unmarshal(eventsJSON, &envelope); err != nil {
		// Try parsing as a plain array
		var events []runEvent
		if err2 := json.Unmarshal(eventsJSON, &events); err2 != nil {
			return "", fmt.Errorf("parsing events JSON: %w", err)
		}
		envelope.Events = events
	}

	// Find the last assistant message
	var lastAssistantContent string
	for i := len(envelope.Events) - 1; i >= 0; i-- {
		ev := envelope.Events[i]
		if ev.Message.Role == "assistant" && strings.TrimSpace(ev.Message.Content) != "" {
			lastAssistantContent = ev.Message.Content
			break
		}
	}

	if lastAssistantContent == "" {
		return "", nil
	}

	// Find the HANDOFF header
	loc := handoffHeaderRe.FindStringIndex(lastAssistantContent)
	if loc == nil {
		return "", nil
	}

	// Extract from the header to end or next ## section
	afterHeader := lastAssistantContent[loc[1]:]
	afterHeader = strings.TrimLeft(afterHeader, "\r\n")

	// Find next ## header (if any)
	nextLoc := nextH2Re.FindStringIndex(afterHeader)
	var body string
	if nextLoc != nil {
		body = afterHeader[:nextLoc[0]]
	} else {
		body = afterHeader
	}

	return strings.TrimSpace(body), nil
}

// ChainExtractor tries multiple extractors in order, returning the first non-empty result.
type ChainExtractor struct {
	extractors []HandoffExtractor
}

// NewChainExtractor creates a ChainExtractor from the given extractors.
func NewChainExtractor(extractors ...HandoffExtractor) *ChainExtractor {
	return &ChainExtractor{extractors: extractors}
}

// Extract tries each extractor in order.
func (c *ChainExtractor) Extract(ctx context.Context, eventsJSON []byte) (string, error) {
	for _, ext := range c.extractors {
		result, err := ext.Extract(ctx, eventsJSON)
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}
	}
	return "", nil
}
