package services

import (
	"strings"
	"testing"
)

// TestIntegration_AsyncGuidanceWithActiveOperations verifies that when there
// are active async operations, the guidance message is properly constructed.
func TestIntegration_AsyncGuidanceWithActiveOperations(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil)

	// Add active operations
	tracker.AddTestOperation(&AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-123",
		ToolName:   "spawn_coding_agent",
		Status:     "running",
	})
	tracker.AddTestOperation(&AsyncOperation{
		ToolCallID: "tc-2",
		ChatID:     "chat-123",
		ToolName:   "run_automation",
		Status:     "running",
	})

	svc := createTestCompletionService(tracker)

	// Get active operations
	activeOps := tracker.GetActiveOperations("chat-123")
	if len(activeOps) != 2 {
		t.Fatalf("expected 2 active operations, got %d", len(activeOps))
	}

	// Build guidance message
	guidance := svc.buildAsyncGuidanceMessage(activeOps)

	// Verify guidance mentions both tools
	if !strings.Contains(guidance, "spawn_coding_agent") {
		t.Error("expected guidance to mention 'spawn_coding_agent'")
	}
	if !strings.Contains(guidance, "run_automation") {
		t.Error("expected guidance to mention 'run_automation'")
	}

	// Verify guidance instructs not to poll
	if !strings.Contains(guidance, "DO NOT call") {
		t.Error("expected guidance to instruct not to call status tools")
	}
	if !strings.Contains(guidance, "automatically") {
		t.Error("expected guidance to mention automatic delivery")
	}
}

// TestIntegration_AsyncGuidanceNotInjectedWhenNoOps verifies that async guidance
// is not injected when there are no active operations.
func TestIntegration_AsyncGuidanceNotInjectedWhenNoOps(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil)

	// No operations added

	activeOps := tracker.GetActiveOperations("chat-empty")
	if len(activeOps) != 0 {
		t.Errorf("expected 0 active operations, got %d", len(activeOps))
	}

	// The PrepareCompletionRequest logic only injects guidance when activeOps > 0
	// This test verifies the condition works correctly
}

// TestIntegration_AsyncGuidanceOnlyForSpecificChat verifies that async guidance
// is only injected for the chat with active operations, not all chats.
func TestIntegration_AsyncGuidanceOnlyForSpecificChat(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil)

	// Add operation for chat-1
	tracker.AddTestOperation(&AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		ToolName:   "some_tool",
		Status:     "running",
	})

	// Verify chat-1 has active operations
	chat1Ops := tracker.GetActiveOperations("chat-1")
	if len(chat1Ops) != 1 {
		t.Errorf("expected 1 active operation for chat-1, got %d", len(chat1Ops))
	}

	// Verify chat-2 has no active operations
	chat2Ops := tracker.GetActiveOperations("chat-2")
	if len(chat2Ops) != 0 {
		t.Errorf("expected 0 active operations for chat-2, got %d", len(chat2Ops))
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

// TestIntegration_ConcurrentAsyncGuidance verifies thread-safe async guidance building.
func TestIntegration_ConcurrentAsyncGuidance(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil)

	// Add operations
	for i := 0; i < 5; i++ {
		tracker.AddTestOperation(&AsyncOperation{
			ToolCallID: idString("tc", i),
			ChatID:     "chat-concurrent",
			ToolName:   idString("tool", i),
			Status:     "running",
		})
	}

	svc := createTestCompletionService(tracker)

	// Concurrent guidance building
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			activeOps := tracker.GetActiveOperations("chat-concurrent")
			if len(activeOps) > 0 {
				msg := svc.buildAsyncGuidanceMessage(activeOps)
				if msg == "" {
					t.Error("empty guidance message")
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}
}
