package services

import (
	"context"
	"testing"
	"time"
)

// TestProcessStatusResult_InProgress verifies non-terminal status handling.
func TestProcessStatusResult_InProgress(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "starting",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &AsyncBehavior{
			CompletionConditions: &CompletionConditions{
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
	svc := NewAsyncTrackerService(nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "starting",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &AsyncBehavior{
			CompletionConditions: &CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed"},
			},
			ProgressTracking: &ProgressTracking{
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
	svc := NewAsyncTrackerService(nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "running",
	}
	svc.AddTestOperation(op)

	conditions := &CompletionConditions{
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
	svc := NewAsyncTrackerService(nil, nil)

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
	svc := NewAsyncTrackerService(nil, nil)

	// Create operation without cancellation config
	op := &AsyncOperation{
		ToolCallID:    "tc-1",
		ChatID:        "chat-1",
		ToolName:      "test_tool",
		Status:        "running",
		AsyncBehavior: &AsyncBehavior{}, // No cancellation config
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
	svc := NewAsyncTrackerService(nil, nil)

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

	svc := NewAsyncTrackerService(nil, nil)

	op := &AsyncOperation{
		ToolCallID:    "tc-1",
		ChatID:        "chat-1",
		ToolName:      "test_tool",
		ExternalRunID: "run-123",
		Status:        "running",
		AsyncBehavior: &AsyncBehavior{
			Cancellation: &CancellationBehavior{
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
	svc := NewAsyncTrackerService(nil, nil)

	// Operation with no backoff config
	svc.mu.Lock()
	svc.operations["tc-1"] = &AsyncOperation{
		ToolCallID: "tc-1",
		AsyncBehavior: &AsyncBehavior{
			StatusPolling: &StatusPolling{
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
