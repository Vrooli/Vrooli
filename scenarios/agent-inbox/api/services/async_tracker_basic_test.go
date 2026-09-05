package services

import (
	"context"
	"testing"
	"time"
)

// Test helpers (inlined to avoid import cycle with testutil)

func waitForUpdate(t *testing.T, ch <-chan AsyncStatusUpdate, timeout time.Duration) AsyncStatusUpdate {
	t.Helper()
	select {
	case update, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		return update
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for update after %v", timeout)
		return AsyncStatusUpdate{}
	}
}

func waitForCompletion(t *testing.T, ch <-chan AsyncCompletionEvent, timeout time.Duration) AsyncCompletionEvent {
	t.Helper()
	select {
	case event, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		return event
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for completion after %v", timeout)
		return AsyncCompletionEvent{}
	}
}

// TestNewAsyncTrackerService verifies the service initializes correctly.
func TestNewAsyncTrackerService(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.operations == nil {
		t.Error("expected operations map to be initialized")
	}
	if svc.subscriptions == nil {
		t.Error("expected subscriptions map to be initialized")
	}
	if svc.completionCallbacks == nil {
		t.Error("expected completionCallbacks map to be initialized")
	}
}

// TestStartTracking_MissingAsyncBehavior verifies error when no async config is provided.
func TestStartTracking_MissingAsyncBehavior(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	err := svc.StartTracking(context.Background(), "tc-1", "chat-1", "tool", "scenario", nil, nil)
	if err == nil {
		t.Error("expected error when asyncBehavior is nil")
	}
}

// TestStartTracking_MissingStatusPolling verifies error when status polling config is missing.
func TestStartTracking_MissingStatusPolling(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)
	asyncBehavior := &AsyncBehavior{} // No StatusPolling

	err := svc.StartTracking(context.Background(), "tc-1", "chat-1", "tool", "scenario", nil, asyncBehavior)
	if err == nil {
		t.Error("expected error when StatusPolling is nil")
	}
}

// TestStartTracking_ExtractsOperationID verifies the operation ID is extracted from the result.
func TestStartTracking_ExtractsOperationID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Cancel context immediately to stop polling
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	asyncBehavior := &AsyncBehavior{
		StatusPolling: &StatusPolling{
			OperationIdField:       "run_id",
			StatusTool:             "get_status",
			StatusToolIdParam:      "id",
			PollIntervalSeconds:    1,
			MaxPollDurationSeconds: 60,
		},
		CompletionConditions: &CompletionConditions{
			StatusField:   "status",
			SuccessValues: []string{"completed"},
			FailureValues: []string{"failed"},
		},
	}

	toolResult := map[string]interface{}{
		"run_id": "run-123",
		"status": "pending",
	}

	err := svc.StartTracking(ctx, "tc-1", "chat-1", "test_tool", "test_scenario", toolResult, asyncBehavior)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify operation was created with correct external ID
	op := svc.GetOperation("tc-1")
	if op == nil {
		t.Fatal("expected operation to be tracked")
	}
	if op.ExternalRunID != "run-123" {
		t.Errorf("expected ExternalRunID='run-123', got '%s'", op.ExternalRunID)
	}
	if op.Status != "pending" {
		t.Errorf("expected Status='pending', got '%s'", op.Status)
	}
}

// TestStartTracking_MissingOperationID verifies error when run_id field is missing.
func TestStartTracking_MissingOperationID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	asyncBehavior := &AsyncBehavior{
		StatusPolling: &StatusPolling{
			OperationIdField: "run_id",
			StatusTool:       "get_status",
		},
	}

	// Result missing run_id field
	toolResult := map[string]interface{}{
		"status": "pending",
	}

	err := svc.StartTracking(context.Background(), "tc-1", "chat-1", "tool", "scenario", toolResult, asyncBehavior)
	if err == nil {
		t.Error("expected error when operation ID field is missing")
	}
}

// TestSubscribeWithID verifies ID-based subscription tracking.
func TestSubscribeWithID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	sub := svc.SubscribeWithID("chat-1")
	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}
	if sub.ChatID != "chat-1" {
		t.Errorf("expected ChatID='chat-1', got '%s'", sub.ChatID)
	}
	if sub.ID == "" {
		t.Error("expected non-empty subscription ID")
	}
	if sub.Channel == nil {
		t.Error("expected non-nil channel")
	}

	// Verify subscription is tracked
	svc.mu.RLock()
	if _, ok := svc.subscriptions[sub.ID]; !ok {
		t.Error("subscription not found in subscriptions map")
	}
	if len(svc.chatSubs["chat-1"]) != 1 {
		t.Errorf("expected 1 subscription for chat, got %d", len(svc.chatSubs["chat-1"]))
	}
	svc.mu.RUnlock()

	// Cleanup
	svc.UnsubscribeByID(sub)
}

// TestUnsubscribeByID verifies subscription cleanup.
func TestUnsubscribeByID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	sub := svc.SubscribeWithID("chat-1")
	svc.UnsubscribeByID(sub)

	// Verify subscription is removed
	svc.mu.RLock()
	if _, ok := svc.subscriptions[sub.ID]; ok {
		t.Error("subscription should be removed from subscriptions map")
	}
	if len(svc.chatSubs["chat-1"]) != 0 {
		t.Errorf("expected 0 subscriptions for chat, got %d", len(svc.chatSubs["chat-1"]))
	}
	svc.mu.RUnlock()
}

// TestRegisterCompletionCallback verifies callback registration.
func TestRegisterCompletionCallback(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	ch := svc.RegisterCompletionCallback("chat-1")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Verify callback is registered
	svc.mu.RLock()
	if _, ok := svc.completionCallbacks["chat-1"]; !ok {
		t.Error("callback not found in completionCallbacks map")
	}
	svc.mu.RUnlock()

	// Cleanup
	svc.UnregisterCompletionCallback("chat-1")
}

// TestUnregisterCompletionCallback verifies callback cleanup.
func TestUnregisterCompletionCallback(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	svc.RegisterCompletionCallback("chat-1")
	svc.UnregisterCompletionCallback("chat-1")

	// Verify callback is removed
	svc.mu.RLock()
	if _, ok := svc.completionCallbacks["chat-1"]; ok {
		t.Error("callback should be removed from completionCallbacks map")
	}
	svc.mu.RUnlock()
}

// TestStopTracking verifies operation cancellation and callback trigger.
func TestStopTracking(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	// Register completion callback first
	completionCh := svc.RegisterCompletionCallback("chat-1")
	defer svc.UnregisterCompletionCallback("chat-1")

	// Create a tracked operation manually (bypassing StartTracking which needs executor)
	svc.mu.Lock()
	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		ToolName:   "test_tool",
		Status:     "running",
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	svc.operations["tc-1"] = op
	_, cancel := context.WithCancel(context.Background())
	svc.cancelFuncs["tc-1"] = cancel
	svc.mu.Unlock()

	// Stop tracking
	svc.StopTracking("tc-1")

	// Verify operation is marked as cancelled
	svc.mu.RLock()
	if op.Status != "cancelled" {
		t.Errorf("expected Status='cancelled', got '%s'", op.Status)
	}
	if op.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
	svc.mu.RUnlock()

	// Verify completion callback was triggered
	event := waitForCompletion(t, completionCh, 100*time.Millisecond)
	if event.Status != "cancelled" {
		t.Errorf("expected event Status='cancelled', got '%s'", event.Status)
	}

	// Verify cancel function was cleaned up
	svc.mu.RLock()
	if _, ok := svc.cancelFuncs["tc-1"]; ok {
		t.Error("cancel function should be removed")
	}
	svc.mu.RUnlock()
}
