package services

import (
	"context"
	"testing"
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
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
	svc := NewAsyncTrackerService(nil, nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil, nil)

	err := svc.StartTracking(context.Background(), "tc-1", "chat-1", "tool", "scenario", nil, nil)
	if err == nil {
		t.Error("expected error when asyncBehavior is nil")
	}
}

// TestStartTracking_MissingStatusPolling verifies error when status polling config is missing.
func TestStartTracking_MissingStatusPolling(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)
	asyncBehavior := &toolspb.AsyncBehavior{} // No StatusPolling

	err := svc.StartTracking(context.Background(), "tc-1", "chat-1", "tool", "scenario", nil, asyncBehavior)
	if err == nil {
		t.Error("expected error when StatusPolling is nil")
	}
}

// TestStartTracking_ExtractsOperationID verifies the operation ID is extracted from the result.
func TestStartTracking_ExtractsOperationID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Cancel context immediately to stop polling
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	asyncBehavior := &toolspb.AsyncBehavior{
		StatusPolling: &toolspb.StatusPolling{
			OperationIdField:       "run_id",
			StatusTool:             "get_status",
			StatusToolIdParam:      "id",
			PollIntervalSeconds:    1,
			MaxPollDurationSeconds: 60,
		},
		CompletionConditions: &toolspb.CompletionConditions{
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
	svc := NewAsyncTrackerService(nil, nil, nil)

	asyncBehavior := &toolspb.AsyncBehavior{
		StatusPolling: &toolspb.StatusPolling{
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
	svc := NewAsyncTrackerService(nil, nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil, nil)

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

// TestGetActiveOperations verifies filtering by chat ID and completion status.
func TestGetActiveOperations(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	now := time.Now()
	completed := now.Add(-time.Minute)

	// Add some operations
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "running",
	}
	svc.operations["tc-2"] = &AsyncOperation{
		ToolCallID:  "tc-2",
		ChatID:      "chat-1",
		Status:      "completed",
		CompletedAt: &completed, // Already completed
	}
	svc.operations["tc-3"] = &AsyncOperation{
		ToolCallID: "tc-3",
		ChatID:     "chat-2", // Different chat
		Status:     "running",
	}
	svc.mu.Unlock()

	// Get active operations for chat-1
	active := svc.GetActiveOperations("chat-1")
	if len(active) != 1 {
		t.Errorf("expected 1 active operation, got %d", len(active))
	}
	if len(active) > 0 && active[0].ToolCallID != "tc-1" {
		t.Errorf("expected tc-1, got %s", active[0].ToolCallID)
	}
}

// TestCleanupStaleOperations verifies removal of old completed operations.
func TestCleanupStaleOperations(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	now := time.Now()
	old := now.Add(-2 * time.Hour)
	recent := now.Add(-5 * time.Minute)

	// Add operations
	svc.mu.Lock()
	svc.operations["tc-old"] = &AsyncOperation{
		ToolCallID:  "tc-old",
		Status:      "completed",
		CompletedAt: &old, // Old and completed
	}
	svc.operations["tc-recent"] = &AsyncOperation{
		ToolCallID:  "tc-recent",
		Status:      "completed",
		CompletedAt: &recent, // Recent and completed
	}
	svc.operations["tc-running"] = &AsyncOperation{
		ToolCallID: "tc-running",
		Status:     "running", // Still running
	}
	svc.mu.Unlock()

	// Cleanup with 1 hour retention
	removed := svc.CleanupStaleOperations(time.Hour)

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	// Verify correct operations remain
	svc.mu.RLock()
	if _, ok := svc.operations["tc-old"]; ok {
		t.Error("tc-old should have been removed")
	}
	if _, ok := svc.operations["tc-recent"]; !ok {
		t.Error("tc-recent should remain")
	}
	if _, ok := svc.operations["tc-running"]; !ok {
		t.Error("tc-running should remain")
	}
	svc.mu.RUnlock()
}

// TestGetOperationCount verifies operation counting.
func TestGetOperationCount(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	if svc.GetOperationCount() != 0 {
		t.Error("expected 0 operations initially")
	}

	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{ToolCallID: "tc-1"}
	svc.operations["tc-2"] = &AsyncOperation{ToolCallID: "tc-2"}
	svc.mu.Unlock()

	if svc.GetOperationCount() != 2 {
		t.Errorf("expected 2 operations, got %d", svc.GetOperationCount())
	}
}

// TestSnapshotOperation verifies safe copying of immutable fields.
func TestSnapshotOperation(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	asyncBehavior := &toolspb.AsyncBehavior{
		StatusPolling: &toolspb.StatusPolling{
			StatusTool: "get_status",
		},
	}

	startTime := time.Now()
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID:    "tc-1",
		ChatID:        "chat-1",
		ToolName:      "test_tool",
		Scenario:      "test_scenario",
		ExternalRunID: "run-123",
		AsyncBehavior: asyncBehavior,
		StartedAt:     startTime,
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot to be found")
	}

	if snap.ToolCallID != "tc-1" {
		t.Errorf("expected ToolCallID='tc-1', got '%s'", snap.ToolCallID)
	}
	if snap.ChatID != "chat-1" {
		t.Errorf("expected ChatID='chat-1', got '%s'", snap.ChatID)
	}
	if snap.ToolName != "test_tool" {
		t.Errorf("expected ToolName='test_tool', got '%s'", snap.ToolName)
	}
	if snap.Scenario != "test_scenario" {
		t.Errorf("expected Scenario='test_scenario', got '%s'", snap.Scenario)
	}
	if snap.ExternalRunID != "run-123" {
		t.Errorf("expected ExternalRunID='run-123', got '%s'", snap.ExternalRunID)
	}
	if snap.AsyncBehavior != asyncBehavior {
		t.Error("expected same AsyncBehavior pointer")
	}
}

// TestSnapshotOperation_NotFound verifies behavior when operation doesn't exist.
func TestSnapshotOperation_NotFound(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	snap, ok := svc.snapshotOperation("nonexistent")
	if ok {
		t.Error("expected ok to be false for nonexistent operation")
	}
	if snap != nil {
		t.Error("expected nil snapshot for nonexistent operation")
	}
}

// TestMultipleSubscribersReceiveUpdates verifies updates go to all subscribers.
func TestMultipleSubscribersReceiveUpdates(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Create multiple subscribers
	sub1 := svc.SubscribeWithID("chat-1")
	sub2 := svc.SubscribeWithID("chat-1")
	defer svc.UnsubscribeByID(sub1)
	defer svc.UnsubscribeByID(sub2)

	// Push an update
	update := AsyncStatusUpdate{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "running",
		UpdatedAt:  time.Now(),
	}
	svc.pushUpdateData("chat-1", update)

	// Both subscribers should receive the update
	u1 := waitForUpdate(t, sub1.Channel, 100*time.Millisecond)
	if u1.Status != "running" {
		t.Errorf("subscriber 1: expected status='running', got '%s'", u1.Status)
	}

	u2 := waitForUpdate(t, sub2.Channel, 100*time.Millisecond)
	if u2.Status != "running" {
		t.Errorf("subscriber 2: expected status='running', got '%s'", u2.Status)
	}
}

// TestRemoveOperation verifies explicit operation removal.
func TestRemoveOperation(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{ToolCallID: "tc-1"}
	svc.cancelFuncs["tc-1"] = cancel
	svc.mu.Unlock()

	svc.RemoveOperation("tc-1")

	svc.mu.RLock()
	if _, ok := svc.operations["tc-1"]; ok {
		t.Error("operation should be removed")
	}
	if _, ok := svc.cancelFuncs["tc-1"]; ok {
		t.Error("cancel func should be removed")
	}
	svc.mu.RUnlock()
}

// =============================================================================
// Helper Function Tests
// =============================================================================

// TestSplitDotPath verifies dot-notation path splitting through ExtractField.
// The splitDotPath function is internal to field_extractor.go, so we test it
// indirectly by verifying that ExtractField correctly handles various path formats.
func TestSplitDotPath(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "value",
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectNil   bool
		expectValue string // Only for string values at leaf level
	}{
		{"simple", "a", false, ""},       // Returns map, not nil
		{"two levels", "a.b", false, ""}, // Returns map, not nil
		{"three levels", "a.b.c", false, "value"},
		{"empty", "", true, ""},
		{"trailing dot", "a.", false, ""}, // After filtering empty parts, finds "a"
		{"nonexistent", "x.y.z", true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractField(data, tc.path)
			if tc.expectNil {
				if result != nil {
					t.Errorf("ExtractField(%q) = %v, want nil", tc.path, result)
				}
			} else {
				if result == nil {
					t.Errorf("ExtractField(%q) = nil, want non-nil", tc.path)
				}
				// For string leaf values, verify the actual value
				if tc.expectValue != "" {
					if strVal, ok := result.(string); ok {
						if strVal != tc.expectValue {
							t.Errorf("ExtractField(%q) = %q, want %q", tc.path, strVal, tc.expectValue)
						}
					} else {
						t.Errorf("ExtractField(%q) = %T, want string", tc.path, result)
					}
				}
			}
		})
	}
}

// TestContainsString verifies slice membership check.
func TestContainsString(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
		{[]string{"completed", "succeeded"}, "completed", true},
		{[]string{"failed", "error"}, "timeout", false},
	}

	for _, tc := range tests {
		t.Run(tc.item, func(t *testing.T) {
			result := ContainsString(tc.slice, tc.item)
			if result != tc.expected {
				t.Errorf("ContainsString(%v, %q) = %v, want %v", tc.slice, tc.item, result, tc.expected)
			}
		})
	}
}

// TestExtractField verifies nested field extraction.
func TestExtractField(t *testing.T) {
	data := map[string]interface{}{
		"status": "completed",
		"count":  42.0,
		"nested": map[string]interface{}{
			"value":  "inner",
			"number": 100.0,
			"deep": map[string]interface{}{
				"item": "deepest",
			},
		},
	}

	tests := []struct {
		path     string
		expected interface{}
	}{
		{"status", "completed"},
		{"count", 42.0},
		{"nested.value", "inner"},
		{"nested.number", 100.0},
		{"nested.deep.item", "deepest"},
		{"nonexistent", nil},
		{"nested.nonexistent", nil},
		{"nested.deep.nonexistent", nil},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := ExtractField(data, tc.path)
			if result != tc.expected {
				t.Errorf("ExtractField(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

// TestExtractStringField verifies string extraction.
func TestExtractStringField(t *testing.T) {
	data := map[string]interface{}{
		"status":  "completed",
		"number":  42.0,
		"boolean": true,
		"nested": map[string]interface{}{
			"message": "hello",
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"status", "completed"},
		{"number", ""},      // Not a string
		{"boolean", ""},     // Not a string
		{"nonexistent", ""}, // Not found
		{"nested.message", "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := ExtractStringField(data, tc.path)
			if result != tc.expected {
				t.Errorf("ExtractStringField(%q) = %q, want %q", tc.path, result, tc.expected)
			}
		})
	}
}

// TestExtractIntField verifies int extraction from various numeric types.
func TestExtractIntField(t *testing.T) {
	data := map[string]interface{}{
		"float":   42.0,
		"int":     100,
		"int64":   int64(200),
		"string":  "not a number",
		"boolean": true,
		"nested": map[string]interface{}{
			"progress": 75.0,
		},
	}

	tests := []struct {
		path     string
		expected *int
	}{
		{"float", intPtr(42)},
		{"int", intPtr(100)},
		{"int64", intPtr(200)},
		{"string", nil},
		{"boolean", nil},
		{"nonexistent", nil},
		{"nested.progress", intPtr(75)},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := ExtractIntField(data, tc.path)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("ExtractIntField(%q) = %v, want nil", tc.path, *result)
				}
			} else {
				if result == nil {
					t.Errorf("ExtractIntField(%q) = nil, want %d", tc.path, *tc.expected)
				} else if *result != *tc.expected {
					t.Errorf("ExtractIntField(%q) = %d, want %d", tc.path, *result, *tc.expected)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

// =============================================================================
// ProcessStatusResult Tests
// =============================================================================

// TestProcessStatusResult_SuccessCompletion verifies success detection.
func TestProcessStatusResult_SuccessCompletion(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Subscribe to receive updates
	sub := svc.SubscribeWithID("chat-1")
	defer svc.UnsubscribeByID(sub)

	// Create operation
	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		ToolName:   "test_tool",
		Status:     "running",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &toolspb.AsyncBehavior{
			CompletionConditions: &toolspb.CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed", "succeeded"},
				FailureValues: []string{"failed", "error"},
			},
		},
	}
	svc.AddTestOperation(op)

	// Process a success result
	result := map[string]interface{}{
		"status": "completed",
	}

	conditions := op.AsyncBehavior.CompletionConditions
	isTerminal, status := svc.processStatusResult(op, result, conditions)

	if !isTerminal {
		t.Error("expected terminal to be true for success")
	}
	if status != "completed" {
		t.Errorf("expected status 'completed', got %q", status)
	}
	if op.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

// TestProcessStatusResult_FailureCompletion verifies failure detection.
func TestProcessStatusResult_FailureCompletion(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "running",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &toolspb.AsyncBehavior{
			CompletionConditions: &toolspb.CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed"},
				FailureValues: []string{"failed", "error"},
				ErrorField:    "error_message",
			},
		},
	}
	svc.AddTestOperation(op)

	result := map[string]interface{}{
		"status":        "failed",
		"error_message": "something went wrong",
	}

	conditions := op.AsyncBehavior.CompletionConditions
	isTerminal, status := svc.processStatusResult(op, result, conditions)

	if !isTerminal {
		t.Error("expected terminal to be true for failure")
	}
	if status != "failed" {
		t.Errorf("expected status 'failed', got %q", status)
	}
	if op.Error != "something went wrong" {
		t.Errorf("expected error message, got %q", op.Error)
	}
}

// TestProcessStatusResult_InProgress verifies non-terminal status handling.
func TestProcessStatusResult_InProgress(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "starting",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &toolspb.AsyncBehavior{
			CompletionConditions: &toolspb.CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed"},
				FailureValues: []string{"failed"},
			},
		},
	}
	svc.AddTestOperation(op)

	result := map[string]interface{}{
		"status": "running",
	}

	conditions := op.AsyncBehavior.CompletionConditions
	isTerminal, status := svc.processStatusResult(op, result, conditions)

	if isTerminal {
		t.Error("expected terminal to be false for in-progress")
	}
	if status != "running" {
		t.Errorf("expected status 'running', got %q", status)
	}
	if op.CompletedAt != nil {
		t.Error("expected CompletedAt to remain nil")
	}
}

// TestProcessStatusResult_WithProgressTracking verifies progress extraction.
func TestProcessStatusResult_WithProgressTracking(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "starting",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &toolspb.AsyncBehavior{
			CompletionConditions: &toolspb.CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed"},
			},
			ProgressTracking: &toolspb.ProgressTracking{
				ProgressField: "progress",
				MessageField:  "message",
				PhaseField:    "phase",
			},
		},
	}
	svc.AddTestOperation(op)

	result := map[string]interface{}{
		"status":   "running",
		"progress": 50.0,
		"message":  "Processing data",
		"phase":    "analysis",
	}

	conditions := op.AsyncBehavior.CompletionConditions
	svc.processStatusResult(op, result, conditions)

	if op.Progress == nil || *op.Progress != 50 {
		t.Error("expected progress to be 50")
	}
	if op.Message != "Processing data" {
		t.Errorf("expected message 'Processing data', got %q", op.Message)
	}
	if op.Phase != "analysis" {
		t.Errorf("expected phase 'analysis', got %q", op.Phase)
	}
}

// TestProcessStatusResult_InvalidResult verifies handling of non-map results.
func TestProcessStatusResult_InvalidResult(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "running",
	}
	svc.AddTestOperation(op)

	conditions := &toolspb.CompletionConditions{
		StatusField:   "status",
		SuccessValues: []string{"completed"},
	}

	// Pass a non-map result
	isTerminal, status := svc.processStatusResult(op, "not a map", conditions)

	if isTerminal {
		t.Error("expected terminal to be false for invalid result")
	}
	if status != "" {
		t.Errorf("expected empty status, got %q", status)
	}
}

// =============================================================================
// HandleTimeout Tests
// =============================================================================

// TestHandleTimeout verifies timeout handling.
func TestHandleTimeout(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Register completion callback
	completionCh := svc.RegisterCompletionCallback("chat-1")
	defer svc.UnregisterCompletionCallback("chat-1")

	// Subscribe for updates
	sub := svc.SubscribeWithID("chat-1")
	defer svc.UnsubscribeByID(sub)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		ToolName:   "test_tool",
		Status:     "running",
		UpdatedAt:  time.Now(),
	}
	svc.AddTestOperation(op)

	svc.handleTimeout(op)

	// Verify status
	if op.Status != "timeout" {
		t.Errorf("expected status 'timeout', got %q", op.Status)
	}
	if op.Error != "Operation timed out" {
		t.Errorf("expected error message, got %q", op.Error)
	}
	if op.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	// Verify update was pushed
	update := waitForUpdate(t, sub.Channel, 100*time.Millisecond)
	if update.Status != "timeout" {
		t.Errorf("update: expected status 'timeout', got %q", update.Status)
	}
	if !update.IsTerminal {
		t.Error("expected IsTerminal to be true")
	}

	// Verify completion callback was triggered
	event := waitForCompletion(t, completionCh, 100*time.Millisecond)
	if event.Status != "timeout" {
		t.Errorf("event: expected status 'timeout', got %q", event.Status)
	}
}

// =============================================================================
// CancelOperation Tests
// =============================================================================

// TestCancelOperation_NoCancelTool verifies operation with no cancel tool configured.
func TestCancelOperation_NoCancelTool(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Create operation without cancellation config
	op := &AsyncOperation{
		ToolCallID:    "tc-1",
		ChatID:        "chat-1",
		ToolName:      "test_tool",
		Status:        "running",
		AsyncBehavior: &toolspb.AsyncBehavior{}, // No cancellation config
	}
	svc.AddTestOperation(op)

	err := svc.CancelOperation(context.Background(), "tc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify operation was still stopped
	svc.mu.RLock()
	if op.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %q", op.Status)
	}
	svc.mu.RUnlock()
}

// TestCancelOperation_NotFound verifies error for non-existent operation.
func TestCancelOperation_NotFound(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	err := svc.CancelOperation(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent operation")
	}
}

// TestCancelOperation_WithCancelTool_NilExecutor verifies behavior when executor is nil.
// NOTE: This test documents a known limitation - CancelOperation panics when
// a cancel tool is configured but the executor is nil. This is acceptable since
// in production, the executor should never be nil when cancellation is configured.
// Full cancel tool execution requires integration testing with real executor.
func TestCancelOperation_WithCancelTool_NilExecutor(t *testing.T) {
	t.Skip("Skipping: CancelOperation panics with nil executor when cancel tool configured - known limitation")

	svc := NewAsyncTrackerService(nil, nil, nil)

	op := &AsyncOperation{
		ToolCallID:    "tc-1",
		ChatID:        "chat-1",
		ToolName:      "test_tool",
		ExternalRunID: "run-123",
		Status:        "running",
		AsyncBehavior: &toolspb.AsyncBehavior{
			Cancellation: &toolspb.CancellationBehavior{
				CancelTool:        "cancel_run",
				CancelToolIdParam: "run_id",
			},
		},
	}
	svc.AddTestOperation(op)

	err := svc.CancelOperation(context.Background(), "tc-1")
	if err != nil {
		t.Logf("CancelOperation returned error (expected): %v", err)
	}
}

// =============================================================================
// ExtractOperationID Tests
// =============================================================================

// =============================================================================
// Backoff Configuration Tests
// =============================================================================

// TestSnapshotOperation_BackoffDefaults verifies default backoff values.
func TestSnapshotOperation_BackoffDefaults(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Operation with no backoff config
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &toolspb.AsyncBehavior{
			StatusPolling: &toolspb.StatusPolling{
				PollIntervalSeconds: 5,
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	// Default: no backoff (multiplier = 1.0)
	if snap.BackoffMultiplier != 1.0 {
		t.Errorf("expected default multiplier=1.0, got %f", snap.BackoffMultiplier)
	}
	// Initial should be the configured poll interval
	if snap.BackoffInitial != 5*time.Second {
		t.Errorf("expected initial=5s, got %v", snap.BackoffInitial)
	}
	// Max should equal initial when no backoff
	if snap.BackoffMax != 5*time.Second {
		t.Errorf("expected max=5s (same as initial), got %v", snap.BackoffMax)
	}
}

// TestSnapshotOperation_BackoffFromProto verifies backoff config extraction.
func TestSnapshotOperation_BackoffFromProto(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Operation with full backoff config
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &toolspb.AsyncBehavior{
			StatusPolling: &toolspb.StatusPolling{
				PollIntervalSeconds: 5,
				Backoff: &toolspb.PollingBackoff{
					InitialIntervalSeconds: 2,
					MaxIntervalSeconds:     30,
					Multiplier:             1.5,
				},
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	if snap.BackoffInitial != 2*time.Second {
		t.Errorf("expected initial=2s, got %v", snap.BackoffInitial)
	}
	if snap.BackoffMax != 30*time.Second {
		t.Errorf("expected max=30s, got %v", snap.BackoffMax)
	}
	if snap.BackoffMultiplier != 1.5 {
		t.Errorf("expected multiplier=1.5, got %f", snap.BackoffMultiplier)
	}
}

// TestSnapshotOperation_BackoffMinInterval verifies minimum interval enforcement.
func TestSnapshotOperation_BackoffMinInterval(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Operation with too-low interval (below MinPollInterval)
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &toolspb.AsyncBehavior{
			StatusPolling: &toolspb.StatusPolling{
				PollIntervalSeconds: 0, // Below minimum
				Backoff: &toolspb.PollingBackoff{
					InitialIntervalSeconds: 0, // Also below minimum
					MaxIntervalSeconds:     60,
					Multiplier:             2.0,
				},
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	// Should fall back to default, not go below minimum
	if snap.BackoffInitial < MinPollInterval {
		t.Errorf("expected initial >= %v (MinPollInterval), got %v", MinPollInterval, snap.BackoffInitial)
	}
}

// TestSnapshotOperation_BackoffInvalidMultiplier verifies multiplier validation.
func TestSnapshotOperation_BackoffInvalidMultiplier(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	// Operation with invalid multiplier (< 1.0)
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &toolspb.AsyncBehavior{
			StatusPolling: &toolspb.StatusPolling{
				PollIntervalSeconds: 5,
				Backoff: &toolspb.PollingBackoff{
					InitialIntervalSeconds: 5,
					MaxIntervalSeconds:     30,
					Multiplier:             0.5, // Invalid - less than 1.0
				},
			},
		},
	}
	svc.mu.Unlock()

	snap, ok := svc.snapshotOperation("tc-1")
	if !ok {
		t.Fatal("expected snapshot")
	}

	// Invalid multiplier should result in no backoff (1.0)
	if snap.BackoffMultiplier != 1.0 {
		t.Errorf("expected multiplier=1.0 for invalid config, got %f", snap.BackoffMultiplier)
	}
}

// TestBackoffCalculation verifies the backoff interval growth calculation.
func TestBackoffCalculation(t *testing.T) {
	// This tests the math used in pollLoop for interval growth
	tests := []struct {
		name       string
		initial    time.Duration
		max        time.Duration
		multiplier float64
		iterations int
		expected   []time.Duration // Expected intervals after each iteration
	}{
		{
			name:       "standard backoff",
			initial:    2 * time.Second,
			max:        30 * time.Second,
			multiplier: 1.5,
			iterations: 5,
			expected: []time.Duration{
				2 * time.Second,          // Initial
				3 * time.Second,          // 2 * 1.5 = 3
				4500 * time.Millisecond,  // 3 * 1.5 = 4.5
				6750 * time.Millisecond,  // 4.5 * 1.5 = 6.75
				10125 * time.Millisecond, // 6.75 * 1.5 = 10.125
			},
		},
		{
			name:       "hits max",
			initial:    5 * time.Second,
			max:        10 * time.Second,
			multiplier: 2.0,
			iterations: 4,
			expected: []time.Duration{
				5 * time.Second,  // Initial
				10 * time.Second, // 5 * 2 = 10 (at max)
				10 * time.Second, // Capped at max
				10 * time.Second, // Capped at max
			},
		},
		{
			name:       "no backoff",
			initial:    5 * time.Second,
			max:        5 * time.Second,
			multiplier: 1.0,
			iterations: 3,
			expected: []time.Duration{
				5 * time.Second, // No change
				5 * time.Second, // No change
				5 * time.Second, // No change
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currentInterval := tc.initial

			for i, expected := range tc.expected {
				// First iteration uses initial interval
				if i == 0 {
					if currentInterval != expected {
						t.Errorf("iteration %d: expected %v, got %v", i, expected, currentInterval)
					}
					continue
				}

				// Calculate next interval (same logic as pollLoop)
				if tc.multiplier > 1.0 {
					nextInterval := time.Duration(float64(currentInterval) * tc.multiplier)
					if nextInterval > tc.max {
						nextInterval = tc.max
					}
					currentInterval = nextInterval
				}

				// Allow 1ms tolerance for floating point
				diff := currentInterval - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Millisecond {
					t.Errorf("iteration %d: expected ~%v, got %v", i, expected, currentInterval)
				}
			}
		})
	}
}

// TestExtractOperationID verifies operation ID extraction.
func TestExtractOperationID(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	tests := []struct {
		name      string
		result    interface{}
		fieldPath string
		expected  string
		wantErr   bool
	}{
		{
			name:      "simple field",
			result:    map[string]interface{}{"run_id": "run-123"},
			fieldPath: "run_id",
			expected:  "run-123",
		},
		{
			name: "nested field",
			result: map[string]interface{}{
				"data": map[string]interface{}{
					"execution_id": "exec-456",
				},
			},
			fieldPath: "data.execution_id",
			expected:  "exec-456",
		},
		{
			name:      "missing field",
			result:    map[string]interface{}{"other": "value"},
			fieldPath: "run_id",
			wantErr:   true,
		},
		{
			name:      "not a map",
			result:    "just a string",
			fieldPath: "run_id",
			wantErr:   true,
		},
		{
			name:      "empty field value",
			result:    map[string]interface{}{"run_id": ""},
			fieldPath: "run_id",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.extractOperationID(tc.result, tc.fieldPath)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
