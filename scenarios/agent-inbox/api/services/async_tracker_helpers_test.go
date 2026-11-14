package services

import (
	"context"
	"testing"
	"time"
)

// TestGetActiveOperations verifies filtering by chat ID and completion status.
func TestGetActiveOperations(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil)

	asyncBehavior := &AsyncBehavior{
		StatusPolling: &StatusPolling{
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
	svc := NewAsyncTrackerService(nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil)

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
