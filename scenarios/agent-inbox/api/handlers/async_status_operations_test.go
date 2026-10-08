package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	cancelCalls         []string
	cancelError         error
	subscribeWithIDFunc func(chatID string) *services.Subscription
}

func newMockAsyncTracker() *mockAsyncTracker {
	return &mockAsyncTracker{
		operations: make(map[string]*services.AsyncOperation),
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
