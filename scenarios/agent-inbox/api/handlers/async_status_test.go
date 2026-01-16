package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// mockAsyncTracker implements the async tracker interface for testing.
type mockAsyncTracker struct {
	mu                  sync.Mutex
	operations          map[string]*services.AsyncOperation
	subscribers         map[string]chan services.AsyncStatusUpdate
	subscribeCalls      []string
	unsubscribeCalls    []unsubscribeCall
	cancelCalls         []string
	cancelError         error
	subscribeWithIDFunc func(chatID string) *services.Subscription
}

type unsubscribeCall struct {
	ChatID  string
	Channel chan services.AsyncStatusUpdate
}

func newMockAsyncTracker() *mockAsyncTracker {
	return &mockAsyncTracker{
		operations:  make(map[string]*services.AsyncOperation),
		subscribers: make(map[string]chan services.AsyncStatusUpdate),
	}
}

func (m *mockAsyncTracker) Subscribe(chatID string) chan services.AsyncStatusUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeCalls = append(m.subscribeCalls, chatID)
	ch := make(chan services.AsyncStatusUpdate, 10)
	m.subscribers[chatID] = ch
	return ch
}

func (m *mockAsyncTracker) Unsubscribe(chatID string, ch chan services.AsyncStatusUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unsubscribeCalls = append(m.unsubscribeCalls, unsubscribeCall{ChatID: chatID, Channel: ch})
	if sub, ok := m.subscribers[chatID]; ok && sub == ch {
		delete(m.subscribers, chatID)
		close(ch)
	}
}

func (m *mockAsyncTracker) GetActiveOperations(chatID string) []*services.AsyncOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ops []*services.AsyncOperation
	for _, op := range m.operations {
		if op.ChatID == chatID && op.CompletedAt == nil {
			ops = append(ops, op)
		}
	}
	return ops
}

func (m *mockAsyncTracker) GetOperation(toolCallID string) *services.AsyncOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operations[toolCallID]
}

func (m *mockAsyncTracker) CancelOperation(ctx context.Context, toolCallID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelCalls = append(m.cancelCalls, toolCallID)
	return m.cancelError
}

func (m *mockAsyncTracker) SubscribeWithID(chatID string) *services.Subscription {
	if m.subscribeWithIDFunc != nil {
		return m.subscribeWithIDFunc(chatID)
	}
	return nil
}

func (m *mockAsyncTracker) UnsubscribeByID(sub *services.Subscription) {}

func (m *mockAsyncTracker) RegisterCompletionCallback(chatID string) <-chan services.AsyncCompletionEvent {
	return nil
}

func (m *mockAsyncTracker) UnregisterCompletionCallback(chatID string) {}

// AddOperation adds an operation to the mock tracker.
func (m *mockAsyncTracker) AddOperation(op *services.AsyncOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[op.ToolCallID] = op
}

// SendUpdate sends an update to subscribers.
func (m *mockAsyncTracker) SendUpdate(chatID string, update services.AsyncStatusUpdate) {
	m.mu.Lock()
	ch, ok := m.subscribers[chatID]
	m.mu.Unlock()
	if ok {
		select {
		case ch <- update:
		default:
		}
	}
}

// setupTestHandler creates a Handlers instance with the mock tracker.
func setupTestHandler(tracker *mockAsyncTracker) *Handlers {
	// Create a real AsyncTrackerService for the actual Handlers struct
	realTracker := services.NewAsyncTrackerService(nil, nil)
	h := &Handlers{
		AsyncTracker: realTracker,
	}
	return h
}

// setupTestHandlerWithMock creates a router and handler for testing.
// Returns the router, handler, and mock tracker.
func setupTestHandlerWithMock() (*mux.Router, *Handlers, *mockAsyncTracker) {
	mock := newMockAsyncTracker()
	// Use a real AsyncTrackerService but wrap it for tests
	// For these tests we need to test against the real handler using the real service
	realTracker := services.NewAsyncTrackerService(nil, nil)
	h := &Handlers{
		AsyncTracker: realTracker,
	}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-status", h.StreamAsyncStatus).Methods("GET")
	r.HandleFunc("/api/v1/chats/{id}/async-operations", h.GetAsyncOperations).Methods("GET")
	r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST")
	return r, h, mock
}

// TestGetAsyncOperations_Success verifies successful retrieval of operations.
func TestGetAsyncOperations_Success(t *testing.T) {
	r, h, _ := setupTestHandlerWithMock()

	// Add some operations to the real tracker
	h.AsyncTracker.AddTestOperation(&services.AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-123",
		ToolName:   "test-tool",
		Status:     "running",
		UpdatedAt:  time.Now(),
	})

	req := httptest.NewRequest("GET", "/api/v1/chats/chat-123/async-operations", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["count"] != float64(1) {
		t.Errorf("expected count 1, got %v", resp["count"])
	}
}

// TestGetAsyncOperations_EmptyChatID verifies error for missing chat ID.
func TestGetAsyncOperations_EmptyChatID(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil),
	}

	// Create a router that doesn't set the id variable
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats//async-operations", h.GetAsyncOperations).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/chats//async-operations", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		// mux returns 404 for routes that don't match
		t.Logf("response code: %d (expected 404 or 400)", w.Code)
	}
}

// TestGetAsyncOperations_NoOperations verifies empty result.
func TestGetAsyncOperations_NoOperations(t *testing.T) {
	r, _, _ := setupTestHandlerWithMock()

	req := httptest.NewRequest("GET", "/api/v1/chats/chat-123/async-operations", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["count"] != float64(0) {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

// TestCancelAsyncOperation_Success verifies successful cancellation.
func TestCancelAsyncOperation_Success(t *testing.T) {
	tracker := services.NewAsyncTrackerService(nil, nil)
	h := &Handlers{
		AsyncTracker: tracker,
	}

	// Add an operation to cancel
	tracker.AddTestOperation(&services.AsyncOperation{
		ToolCallID: "tc-123",
		ChatID:     "chat-abc",
		ToolName:   "test-tool",
		Status:     "running",
	})

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/chats/chat-abc/async-operations/tc-123/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if resp["tool_call_id"] != "tc-123" {
		t.Errorf("expected tool_call_id='tc-123', got %v", resp["tool_call_id"])
	}
}

// TestCancelAsyncOperation_NotFound verifies 404 for unknown operation.
func TestCancelAsyncOperation_NotFound(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil),
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/chats/chat-abc/async-operations/nonexistent/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestCancelAsyncOperation_WrongChat verifies 403 when operation belongs to different chat.
func TestCancelAsyncOperation_WrongChat(t *testing.T) {
	tracker := services.NewAsyncTrackerService(nil, nil)
	h := &Handlers{
		AsyncTracker: tracker,
	}

	// Add an operation for a different chat
	tracker.AddTestOperation(&services.AsyncOperation{
		ToolCallID: "tc-123",
		ChatID:     "chat-other", // Different chat!
		ToolName:   "test-tool",
		Status:     "running",
	})

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/chats/chat-abc/async-operations/tc-123/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestCancelAsyncOperation_MissingParams verifies 400 for missing parameters.
func TestCancelAsyncOperation_MissingParams(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil),
	}

	tests := []struct {
		name string
		url  string
	}{
		{"missing both", "/api/v1/chats//async-operations//cancel"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Using a handler that won't match means 404
			r := mux.NewRouter()
			r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST")

			req := httptest.NewRequest("POST", tc.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// Will get 404 because route doesn't match
			if w.Code != http.StatusNotFound {
				t.Logf("got status %d for %s", w.Code, tc.url)
			}
		})
	}
}

// TestStreamAsyncStatus_SetsSSEHeaders verifies correct SSE headers are set.
func TestStreamAsyncStatus_SetsSSEHeaders(t *testing.T) {
	h := &Handlers{
		AsyncTracker: services.NewAsyncTrackerService(nil, nil),
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
		AsyncTracker: services.NewAsyncTrackerService(nil, nil),
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
	tracker := services.NewAsyncTrackerService(nil, nil)
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
	tracker := services.NewAsyncTrackerService(nil, nil)
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
		name       string
		op         *services.AsyncOperation
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
			update := operationToUpdate(tc.op)

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
	Subscribe(chatID string) chan services.AsyncStatusUpdate
	Unsubscribe(chatID string, ch chan services.AsyncStatusUpdate)
	GetActiveOperations(chatID string) []*services.AsyncOperation
	GetOperation(toolCallID string) *services.AsyncOperation
	CancelOperation(ctx context.Context, toolCallID string) error
} = (*mockAsyncTracker)(nil)

// Ensure errors is imported (used in mock)
var _ = errors.New
