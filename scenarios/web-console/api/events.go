package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#observability
// [REQ:P1-004a] Structured Event Logging

// Event types for session lifecycle and operations.
const (
	EventSessionCreated      = "session.created"
	EventSessionConnected    = "session.connected"
	EventSessionDisconnected = "session.disconnected"
	EventSessionTerminated   = "session.terminated"
	EventSessionDeleted      = "session.deleted"
	EventPaneResized         = "pane.resized"
	EventAIGenerate          = "ai.generate"
	EventAISuggest           = "ai.suggest"
	EventSessionPolicyUpdate = "session.policy_updated"

	// Workspace layout events
	EventWorkspaceLayoutUpdated = "workspace.layout_updated"
	EventPaneUpdated            = "pane.updated"
	EventTabGroupCreated        = "group.created"
	EventTabGroupUpdated        = "group.updated"
	EventTabGroupDeleted        = "group.deleted"
)

// Event is a structured lifecycle event emitted by the system.
type Event struct {
	Type      string            `json:"type"`
	SessionID string            `json:"session_id"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// EventLogger provides structured event emission. Events are logged as JSON
// and optionally forwarded to subscribers for real-time consumption.
// [REQ:P1-004a] Structured Event Logging
type EventLogger struct {
	mu          sync.RWMutex
	subscribers []chan Event
	history     []Event
	maxHistory  int
}

// NewEventLogger creates an event logger with a bounded history buffer.
func NewEventLogger(maxHistory int) *EventLogger {
	if maxHistory <= 0 {
		maxHistory = 1000
	}
	return &EventLogger{
		maxHistory: maxHistory,
	}
}

// Emit records a structured event, logs it, and notifies subscribers.
func (el *EventLogger) Emit(eventType, sessionID string, details map[string]string) {
	evt := Event{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Details:   details,
	}

	// Structured JSON log line
	if data, err := json.Marshal(evt); err == nil {
		log.Printf("[EVENT] %s", string(data))
	}

	el.mu.Lock()
	// Append to bounded history
	el.history = append(el.history, evt)
	if len(el.history) > el.maxHistory {
		el.history = el.history[len(el.history)-el.maxHistory:]
	}
	// Fan-out to subscribers (non-blocking)
	for _, ch := range el.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}
	el.mu.Unlock()
}

// Recent returns the last n events from history.
func (el *EventLogger) Recent(n int) []Event {
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
func (el *EventLogger) Count() int {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return len(el.history)
}

// EventsResponse is the JSON shape returned by the /api/v1/events endpoint.
type EventsResponse struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
}

// handleEvents returns recent structured events from the in-memory ring buffer.
// GET /api/v1/events?limit=N (default 50, max 1000)
// [REQ:P1-004a] Structured Event Logging — HTTP surface for event history
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}
	events := s.events.Recent(limit)
	writeJSON(w, http.StatusOK, EventsResponse{
		Events: events,
		Total:  s.events.Count(),
	})
}
