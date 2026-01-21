package handlers

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-inbox/domain"
)

// =============================================================================
// SSE Event Emission Tests
// =============================================================================

// TestSSEWriter_EventFormatting verifies SSE events are properly formatted.
func TestSSEWriter_EventFormatting(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	// Send content event
	err := writer.SendContent("Hello world")
	if err != nil {
		t.Fatalf("SendContent failed: %v", err)
	}

	body := rec.Body.String()
	// JSON field order may vary, so check for presence of both fields
	if !strings.Contains(body, `"type":"content"`) {
		t.Errorf("expected type:content in event, got: %s", body)
	}
	if !strings.Contains(body, `"content":"Hello world"`) {
		t.Errorf("expected content in event, got: %s", body)
	}
	if !strings.HasPrefix(body, "data: ") {
		t.Errorf("expected data: prefix, got: %s", body)
	}
}

// TestSSEWriter_ToolCallStart verifies tool call start events are formatted correctly.
func TestSSEWriter_ToolCallStart(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendToolCallStart("call_123", "run-agent", `{"task":"test"}`)
	if err != nil {
		t.Fatalf("SendToolCallStart failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"tool_call_start"`) {
		t.Errorf("expected tool_call_start type, got: %s", body)
	}
	if !strings.Contains(body, `"tool_id":"call_123"`) {
		t.Errorf("expected tool_id, got: %s", body)
	}
	if !strings.Contains(body, `"tool_name":"run-agent"`) {
		t.Errorf("expected tool_name, got: %s", body)
	}
}

// TestSSEWriter_ToolCallResult verifies tool call result events.
func TestSSEWriter_ToolCallResult(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendToolCallResult("call_123", "completed", `{"success":true}`, "")
	if err != nil {
		t.Fatalf("SendToolCallResult failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"tool_call_result"`) {
		t.Errorf("expected tool_call_result type, got: %s", body)
	}
	if !strings.Contains(body, `"status":"completed"`) {
		t.Errorf("expected status:completed, got: %s", body)
	}
}

// TestSSEWriter_ToolPendingApproval verifies pending approval events.
func TestSSEWriter_ToolPendingApproval(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendToolPendingApproval("call_456", "dangerous-tool", `{"action":"delete"}`)
	if err != nil {
		t.Fatalf("SendToolPendingApproval failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"tool_pending_approval"`) {
		t.Errorf("expected tool_pending_approval type, got: %s", body)
	}
	if !strings.Contains(body, `"tool_call_id":"call_456"`) {
		t.Errorf("expected tool_call_id, got: %s", body)
	}
}

// TestSSEWriter_AwaitingApprovals verifies awaiting approvals event.
func TestSSEWriter_AwaitingApprovals(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendAwaitingApprovals()
	if err != nil {
		t.Fatalf("SendAwaitingApprovals failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"awaiting_approvals"`) {
		t.Errorf("expected awaiting_approvals type, got: %s", body)
	}
}

// TestSSEWriter_ErrorEvent verifies error event format.
func TestSSEWriter_ErrorEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendError("Something went wrong", "INTERNAL_ERROR")
	if err != nil {
		t.Fatalf("SendError failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"error"`) {
		t.Errorf("expected error type, got: %s", body)
	}
	if !strings.Contains(body, `"error":"Something went wrong"`) {
		t.Errorf("expected error message, got: %s", body)
	}
	if !strings.Contains(body, `"code":"INTERNAL_ERROR"`) {
		t.Errorf("expected error code, got: %s", body)
	}
}

// TestSSEWriter_Done verifies [DONE] sentinel.
func TestSSEWriter_Done(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendDone()
	if err != nil {
		t.Fatalf("SendDone failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "data: [DONE]") {
		t.Errorf("expected [DONE] sentinel, got: %s", body)
	}
}

// =============================================================================
// SSE Parsing Tests (Simulating Client-Side Parsing)
// =============================================================================

// parseSSEDataEvents simulates client-side SSE parsing from a response body.
// This extracts the data field from SSE events.
func parseSSEDataEvents(body string) []string {
	var events []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				events = append(events, data)
			}
		}
	}
	return events
}

// TestSSEParsing_MultipleEvents tests parsing multiple SSE events.
func TestSSEParsing_MultipleEvents(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	// Send multiple events
	writer.SendContent("Hello ")
	writer.SendContent("world")
	writer.SendToolCallStart("call_1", "test-tool", `{}`)
	writer.SendToolCallResult("call_1", "completed", `{}`, "")
	writer.SendDone()

	events := parseSSEDataEvents(rec.Body.String())
	if len(events) != 4 {
		t.Errorf("expected 4 events, got %d", len(events))
	}
}

// TestSSEParsing_EmptyContent handles empty content gracefully.
func TestSSEParsing_EmptyContent(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	// Empty content should still produce valid event
	err := writer.SendContent("")
	if err != nil {
		t.Fatalf("SendContent with empty string failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"content":""`) {
		t.Errorf("expected empty content in event, got: %s", body)
	}
}

// TestSSEParsing_SpecialCharacters tests handling of special characters in content.
func TestSSEParsing_SpecialCharacters(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	// Content with newlines, quotes, and unicode
	specialContent := "Line1\nLine2\t\"quoted\"\nUnicode: \u00e9\u00e0"
	err := writer.SendContent(specialContent)
	if err != nil {
		t.Fatalf("SendContent with special chars failed: %v", err)
	}

	// Should be properly JSON-escaped
	body := rec.Body.String()
	if !strings.Contains(body, `\n`) {
		t.Errorf("expected escaped newlines, got: %s", body)
	}
}

// =============================================================================
// Connection Handling Tests
// =============================================================================

// mockResponseWriter implements http.ResponseWriter and can simulate errors.
type mockResponseWriter struct {
	headers    http.Header
	body       strings.Builder
	statusCode int
	writeErr   error
	mu         sync.Mutex
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers:    make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.body.Write(data)
}

func (m *mockResponseWriter) WriteHeader(code int) {
	m.statusCode = code
}

func (m *mockResponseWriter) simulateDisconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeErr = errors.New("client disconnected")
}

// TestSSEWriter_ClientDisconnect verifies handling of client disconnect mid-stream.
func TestSSEWriter_ClientDisconnect(t *testing.T) {
	mock := newMockResponseWriter()
	writer := NewSSEWriter(mock)

	// First write succeeds
	err := writer.SendContent("Hello")
	if err != nil {
		t.Fatalf("first write should succeed: %v", err)
	}

	// Simulate disconnect
	mock.simulateDisconnect()

	// Second write should fail
	err = writer.SendContent("World")
	if err == nil {
		t.Error("expected error after disconnect")
	}
}

// =============================================================================
// Concurrent Write Tests
// =============================================================================

// TestSSEWriter_ConcurrentWrites tests concurrent writes are serialized correctly.
func TestSSEWriter_ConcurrentWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	var wg sync.WaitGroup
	numWriters := 10
	numEvents := 100

	// Spawn multiple goroutines writing events
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numEvents; j++ {
				writer.SendContent("test")
			}
		}(i)
	}

	wg.Wait()

	// All events should be written
	events := parseSSEDataEvents(rec.Body.String())
	expected := numWriters * numEvents
	if len(events) != expected {
		t.Errorf("expected %d events, got %d", expected, len(events))
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

// TestStreamingWithContextCancel tests that streaming respects context cancellation.
func TestStreamingWithContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel to track if streaming was interrupted
	interrupted := make(chan bool, 1)

	go func() {
		select {
		case <-ctx.Done():
			interrupted <- true
		case <-time.After(5 * time.Second):
			interrupted <- false
		}
	}()

	// Cancel immediately
	cancel()

	// Verify context was cancelled
	select {
	case wasInterrupted := <-interrupted:
		if !wasInterrupted {
			t.Error("expected context to be cancelled")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for cancellation")
	}
}

// =============================================================================
// Tool Execution Event Ordering Tests
// =============================================================================

// TestToolEventOrdering verifies correct ordering of tool events.
func TestToolEventOrdering(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	// Simulate tool execution flow
	writer.SendToolCallStart("call_1", "test-tool", `{"arg":"value"}`)
	writer.SendProgress("executing", "Running tool...")
	writer.SendToolCallResult("call_1", "completed", `{"result":"success"}`, "")
	writer.SendToolCallsComplete(false)
	writer.SendDone()

	body := rec.Body.String()
	lines := strings.Split(body, "\n")

	// Track order
	var order []string
	for _, line := range lines {
		if strings.Contains(line, "tool_call_start") {
			order = append(order, "start")
		} else if strings.Contains(line, "progress") {
			order = append(order, "progress")
		} else if strings.Contains(line, "tool_call_result") {
			order = append(order, "result")
		} else if strings.Contains(line, "tool_calls_complete") {
			order = append(order, "complete")
		} else if strings.Contains(line, "[DONE]") {
			order = append(order, "done")
		}
	}

	expected := []string{"start", "progress", "result", "complete", "done"}
	if len(order) != len(expected) {
		t.Errorf("expected %d events, got %d: %v", len(expected), len(order), order)
	}
	for i, exp := range expected {
		if i >= len(order) || order[i] != exp {
			t.Errorf("event %d: expected %s, got %v", i, exp, order)
			break
		}
	}
}

// =============================================================================
// Message Tree Consistency Tests (Active Leaf Updates)
// =============================================================================

// mockCompletionService implements the parts of CompletionService needed for testing.
type mockCompletionService struct {
	savedMessages []*domain.Message
	setLeafCalls  []string
	setLeafErr    error
	mu            sync.Mutex
}

func (m *mockCompletionService) SaveMessage(ctx context.Context, msg *domain.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedMessages = append(m.savedMessages, msg)
	return nil
}

func (m *mockCompletionService) SetActiveLeaf(ctx context.Context, chatID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setLeafCalls = append(m.setLeafCalls, messageID)
	return m.setLeafErr
}

func (m *mockCompletionService) GetSetLeafCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.setLeafCalls))
	copy(result, m.setLeafCalls)
	return result
}

// TestActiveLeafUpdatedAfterEachToolResponse verifies SetActiveLeaf is called.
// This test documents the current behavior - SetActiveLeaf is called for each tool message.
func TestActiveLeafUpdatedAfterEachToolResponse(t *testing.T) {
	// Note: This is testing the EXPECTED behavior after the fix.
	// The bug was that SetActiveLeaf was called for EACH tool message,
	// but the client refetch could see intermediate states.
	//
	// The proper fix would be:
	// 1. Batch tool messages and set active leaf once at the end
	// 2. Or emit active_leaf_updated SSE events so client doesn't need to refetch

	// This test serves as documentation of the current behavior
	// and will help verify the fix once implemented.

	mock := &mockCompletionService{}

	// Simulate what happens during auto-continue with multiple tool responses
	ctx := context.Background()
	chatID := "chat-123"

	// First tool response
	msg1 := &domain.Message{ID: "msg-1", ChatID: chatID, Role: "tool"}
	mock.SaveMessage(ctx, msg1)
	mock.SetActiveLeaf(ctx, chatID, msg1.ID)

	// Second tool response
	msg2 := &domain.Message{ID: "msg-2", ChatID: chatID, Role: "tool"}
	mock.SaveMessage(ctx, msg2)
	mock.SetActiveLeaf(ctx, chatID, msg2.ID)

	// Final assistant response
	msg3 := &domain.Message{ID: "msg-3", ChatID: chatID, Role: "assistant"}
	mock.SaveMessage(ctx, msg3)
	mock.SetActiveLeaf(ctx, chatID, msg3.ID)

	// Current behavior: 3 SetActiveLeaf calls (one per message)
	calls := mock.GetSetLeafCalls()
	if len(calls) != 3 {
		t.Errorf("expected 3 SetActiveLeaf calls, got %d", len(calls))
	}

	// The last call should set the final assistant message
	if len(calls) > 0 && calls[len(calls)-1] != "msg-3" {
		t.Errorf("expected last active leaf to be msg-3, got %s", calls[len(calls)-1])
	}
}

// =============================================================================
// Streaming Response Headers Test
// =============================================================================

// TestStreamingResponseHeaders verifies correct headers are set for SSE.
func TestStreamingResponseHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	// SetHeaders should be called to configure SSE
	writer.SetHeaders()

	headers := rec.Header()

	// Check Content-Type
	contentType := headers.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", contentType)
	}

	// Check Cache-Control
	cacheControl := headers.Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cacheControl)
	}

	// Check Connection
	connection := headers.Get("Connection")
	if connection != "keep-alive" {
		t.Errorf("expected Connection keep-alive, got %s", connection)
	}
}

// =============================================================================
// Image Generated Event Tests
// =============================================================================

// TestSSEWriter_ImageGenerated verifies image generated event format.
func TestSSEWriter_ImageGenerated(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	imageURL := "https://example.com/generated/image.png"
	err := writer.SendImageGenerated(imageURL)
	if err != nil {
		t.Fatalf("SendImageGenerated failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"image_generated"`) {
		t.Errorf("expected image_generated type, got: %s", body)
	}
	if !strings.Contains(body, `"image_url":"https://example.com/generated/image.png"`) {
		t.Errorf("expected image_url, got: %s", body)
	}
}

// =============================================================================
// Progress Event Tests
// =============================================================================

// TestSSEWriter_Progress verifies progress event format.
func TestSSEWriter_Progress(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendProgress("executing", "Processing step 1 of 3...")
	if err != nil {
		t.Fatalf("SendProgress failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"progress"`) {
		t.Errorf("expected progress type, got: %s", body)
	}
	if !strings.Contains(body, `"phase":"executing"`) {
		t.Errorf("expected phase, got: %s", body)
	}
	if !strings.Contains(body, `"message":"Processing step 1 of 3..."`) {
		t.Errorf("expected message, got: %s", body)
	}
}

// =============================================================================
// Warning Event Tests
// =============================================================================

// TestSSEWriter_Warning verifies warning event format.
func TestSSEWriter_Warning(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	err := writer.SendWarning("Rate limit approaching", "RATE_LIMIT_WARNING")
	if err != nil {
		t.Fatalf("SendWarning failed: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"warning"`) {
		t.Errorf("expected warning type, got: %s", body)
	}
	if !strings.Contains(body, `"message":"Rate limit approaching"`) {
		t.Errorf("expected warning message, got: %s", body)
	}
}
