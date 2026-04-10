package handlers

import (
	"bufio"
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// TestStreamAsyncStatus_SetsSSEHeaders verifies correct SSE headers are set.
func TestStreamAsyncStatus_SetsSSEHeaders(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil, nil),
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-status", h.StreamAsyncStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/chats/chat-123/async-status", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Check headers
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got '%s'", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got '%s'", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("Connection") != "keep-alive" {
		t.Errorf("expected Connection 'keep-alive', got '%s'", w.Header().Get("Connection"))
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS header '*', got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// TestStreamAsyncStatus_SendsConnectedEvent verifies connected event is sent.
func TestStreamAsyncStatus_SendsConnectedEvent(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil, nil),
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-status", h.StreamAsyncStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/chats/chat-123/async-status", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Errorf("expected 'event: connected' in response, got: %s", body)
	}
	if !strings.Contains(body, `"chat_id":"chat-123"`) {
		t.Errorf("expected chat_id in connected event, got: %s", body)
	}
}

// TestStreamAsyncStatus_SendsActiveOperations verifies initial operations are sent.
func TestStreamAsyncStatus_SendsActiveOperations(t *testing.T) {
	tracker := services.NewAsyncTrackerService(nil, nil, nil)
	h := &Handlers{
		AsyncTracker: tracker,
	}

	// Add an active operation
	tracker.AddTestOperation(&services.AsyncOperation{
		ToolCallID: "tc-active",
		ChatID:     "chat-123",
		ToolName:   "test-tool",
		Status:     "running",
		UpdatedAt:  time.Now(),
	})

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-status", h.StreamAsyncStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/chats/chat-123/async-status", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "event: status") {
		t.Errorf("expected 'event: status' in response, got: %s", body)
	}
	if !strings.Contains(body, "tc-active") {
		t.Errorf("expected tool call ID in status event, got: %s", body)
	}
}

// TestStreamAsyncStatus_ClientDisconnect verifies cleanup on client disconnect.
func TestStreamAsyncStatus_ClientDisconnect(t *testing.T) {
	tracker := services.NewAsyncTrackerService(nil, nil, nil)
	h := &Handlers{
		AsyncTracker: tracker,
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-status", h.StreamAsyncStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/chats/chat-123/async-status", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel the context to simulate client disconnect
	cancel()

	// Wait for handler to return
	select {
	case <-done:
		// Success - handler returned
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after context cancellation")
	}
}

// TestOperationToUpdate verifies the conversion function.
func TestOperationToUpdate(t *testing.T) {
	now := time.Now()
	completed := now.Add(-time.Minute)
	progress := 50

	tests := []struct {
		name         string
		op           *services.AsyncOperation
		wantTerminal bool
	}{
		{
			name: "running operation",
			op: &services.AsyncOperation{
				ToolCallID: "tc-1",
				ChatID:     "chat-1",
				ToolName:   "tool-1",
				Status:     "running",
				Progress:   &progress,
				Message:    "Processing...",
				Phase:      "execution",
				UpdatedAt:  now,
			},
			wantTerminal: false,
		},
		{
			name: "completed operation",
			op: &services.AsyncOperation{
				ToolCallID:  "tc-2",
				ChatID:      "chat-2",
				ToolName:    "tool-2",
				Status:      "completed",
				Result:      map[string]string{"data": "result"},
				CompletedAt: &completed,
				UpdatedAt:   completed,
			},
			wantTerminal: true,
		},
		{
			name: "failed operation",
			op: &services.AsyncOperation{
				ToolCallID:  "tc-3",
				ChatID:      "chat-3",
				ToolName:    "tool-3",
				Status:      "failed",
				Error:       "Something went wrong",
				CompletedAt: &completed,
				UpdatedAt:   completed,
			},
			wantTerminal: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			update := services.BuildUpdateFromOperation(tc.op)

			if update.ToolCallID != tc.op.ToolCallID {
				t.Errorf("ToolCallID mismatch: got %s, want %s", update.ToolCallID, tc.op.ToolCallID)
			}
			if update.ChatID != tc.op.ChatID {
				t.Errorf("ChatID mismatch: got %s, want %s", update.ChatID, tc.op.ChatID)
			}
			if update.ToolName != tc.op.ToolName {
				t.Errorf("ToolName mismatch: got %s, want %s", update.ToolName, tc.op.ToolName)
			}
			if update.Status != tc.op.Status {
				t.Errorf("Status mismatch: got %s, want %s", update.Status, tc.op.Status)
			}
			if update.IsTerminal != tc.wantTerminal {
				t.Errorf("IsTerminal mismatch: got %v, want %v", update.IsTerminal, tc.wantTerminal)
			}
		})
	}
}

// parseSSEEvents parses SSE events from a response body.
func parseSSEEvents(body string) []map[string]string {
	var events []map[string]string
	scanner := bufio.NewScanner(strings.NewReader(body))

	current := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(current) > 0 {
				events = append(events, current)
				current = make(map[string]string)
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			current["event"] = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			current["data"] = strings.TrimPrefix(line, "data: ")
		}
	}
	if len(current) > 0 {
		events = append(events, current)
	}
	return events
}

// TestParseSSEEvents verifies our test helper.
func TestParseSSEEvents(t *testing.T) {
	body := `event: connected
data: {"chat_id":"123"}

event: status
data: {"status":"running"}

`
	events := parseSSEEvents(body)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0]["event"] != "connected" {
		t.Errorf("expected first event 'connected', got '%s'", events[0]["event"])
	}
	if events[1]["event"] != "status" {
		t.Errorf("expected second event 'status', got '%s'", events[1]["event"])
	}
}

// Ensure mock fulfills requirements - compile-time check
var _ interface {
	GetActiveOperations(chatID string) []*services.AsyncOperation
	GetOperation(toolCallID string) *services.AsyncOperation
	CancelOperation(ctx context.Context, toolCallID string) error
	SubscribeWithID(chatID string) *services.Subscription
	UnsubscribeByID(sub *services.Subscription)
} = (*mockAsyncTracker)(nil)

// Ensure errors is imported (used in mock)
var _ = errors.New
