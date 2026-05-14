// Package events provides structured lifecycle event emission for the
// web-console API. The Logger collects events in a bounded ring buffer,
// emits a JSON log line for each, and fans them out to non-blocking
// subscribers for real-time consumption (e.g. the /api/v1/events stream).
//
// [REQ:P1-004a] Structured Event Logging
package events

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Event type constants. Strings are stable wire identifiers and must not
// change without coordinating with consumers.
const (
	SessionCreated      = "session.created"
	SessionConnected    = "session.connected"
	SessionDisconnected = "session.disconnected"
	SessionTerminated   = "session.terminated"
	SessionDeleted      = "session.deleted"
	PaneResized         = "pane.resized"
	AIGenerate          = "ai.generate"
	AISuggest           = "ai.suggest"
	SessionPolicyUpdate = "session.policy_updated"
	SessionReattach     = "session.reattach"
	SessionRecovered    = "session.recovered"
	SessionRecoveryFail = "session.recovery_failed"

	// Workspace layout events.
	WorkspaceLayoutUpdated = "workspace.layout_updated"
	PaneUpdated            = "pane.updated"
	TabGroupCreated        = "group.created"
	TabGroupUpdated        = "group.updated"
	TabGroupDeleted        = "group.deleted"
)

// Event is a structured lifecycle event emitted by the system.
type Event struct {
	Type      string            `json:"type"`
	SessionID string            `json:"session_id"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// Logger provides structured event emission. Events are logged as JSON
// and optionally forwarded to subscribers for real-time consumption.
type Logger struct {
	mu          sync.RWMutex
	subscribers []chan Event
	history     []Event
	maxHistory  int
}

// NewLogger creates an event logger with a bounded history buffer.
func NewLogger(maxHistory int) *Logger {
	if maxHistory <= 0 {
		maxHistory = 1000
	}
	return &Logger{
		maxHistory: maxHistory,
	}
}

// Emit records a structured event, logs it, and notifies subscribers.
func (el *Logger) Emit(eventType, sessionID string, details map[string]string) {
	evt := Event{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Details:   details,
	}

	if data, err := json.Marshal(evt); err == nil {
		log.Printf("[EVENT] %s", string(data))
	}

	el.mu.Lock()
	el.history = append(el.history, evt)
	if len(el.history) > el.maxHistory {
		el.history = el.history[len(el.history)-el.maxHistory:]
	}
	for _, ch := range el.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}
	el.mu.Unlock()
}

// Recent returns the last n events from history.
func (el *Logger) Recent(n int) []Event {
	el.mu.RLock()
	defer el.mu.RUnlock()
	if n <= 0 || n > len(el.history) {
		n = len(el.history)
	}
	start := len(el.history) - n
	result := make([]Event, n)
	copy(result, el.history[start:])
	return result
}

// Count returns the total number of events recorded.
func (el *Logger) Count() int {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return len(el.history)
}

// Subscribe registers a channel to receive events as they are emitted.
// The channel must be buffered; emits are non-blocking and drop on a
// full channel. Returns an unsubscribe func.
func (el *Logger) Subscribe(ch chan Event) func() {
	el.mu.Lock()
	el.subscribers = append(el.subscribers, ch)
	el.mu.Unlock()
	return func() {
		el.mu.Lock()
		defer el.mu.Unlock()
		for i, c := range el.subscribers {
			if c == ch {
				el.subscribers = append(el.subscribers[:i], el.subscribers[i+1:]...)
				return
			}
		}
	}
}
