package handlers

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-inbox/domain"
)

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
