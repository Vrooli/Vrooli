package services

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// TestIntegration_AsyncGuidanceWithActiveOperations verifies that when there
// are active async operations, the guidance message is properly constructed.
func TestIntegration_AsyncGuidanceWithActiveOperations(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil, nil)

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

	svc := createTestCompletionService(nil, tracker)

	// Get active operations
	activeOps := tracker.GetActiveOperations("chat-123")
	if len(activeOps) != 2 {
		t.Fatalf("expected 2 active operations, got %d", len(activeOps))
	}

	// Build guidance message
	guidance := svc.buildAsyncGuidanceMessage(activeOps)

	// Verify guidance mentions both tools
	if !strContains(guidance, "spawn_coding_agent") {
		t.Error("expected guidance to mention 'spawn_coding_agent'")
	}
	if !strContains(guidance, "run_automation") {
		t.Error("expected guidance to mention 'run_automation'")
	}

	// Verify guidance instructs not to poll
	if !strContains(guidance, "DO NOT call") {
		t.Error("expected guidance to instruct not to call status tools")
	}
	if !strContains(guidance, "automatically") {
		t.Error("expected guidance to mention automatic delivery")
	}
}

// TestIntegration_AsyncGuidanceNotInjectedWhenNoOps verifies that async guidance
// is not injected when there are no active operations.
func TestIntegration_AsyncGuidanceNotInjectedWhenNoOps(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil, nil)

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
	tracker := NewAsyncTrackerService(nil, nil, nil)

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

// TestIntegration_ForcedToolInternalCanBeForced verifies that even internal
// tools can be forced via the force_tool parameter (for async tracker use).
func TestIntegration_ForcedToolInternalCanBeForced(t *testing.T) {
	registry := newMockToolRegistry()

	// Add an internal tool
	internalTool := createToolWithMetadata("check_status", "Internal status tool", &toolspb.ToolMetadata{
		InternalOnly: true,
	})
	registry.addTool("agent-manager", internalTool)

	svc := createTestCompletionService(registry, nil)

	// Force the internal tool
	toolDef, err := svc.getForcedToolDefinition(context.Background(), "agent-manager:check_status")
	if err != nil {
		t.Fatalf("unexpected error forcing internal tool: %v", err)
	}

	if toolDef == nil {
		t.Fatal("expected to be able to force an internal tool")
	}

	fn, ok := toolDef["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function key")
	}
	if fn["name"] != "check_status" {
		t.Errorf("expected name 'check_status', got %v", fn["name"])
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

// TestIntegration_ConcurrentForcedToolLookup verifies thread-safe forced tool lookup.
func TestIntegration_ConcurrentForcedToolLookup(t *testing.T) {
	registry := newMockToolRegistry()

	// Add multiple tools
	for i := 0; i < 10; i++ {
		tool := createSimpleTool(idString("tool", i), "Test tool")
		registry.addTool("scenario", tool)
	}

	svc := createTestCompletionService(registry, nil)

	// Concurrent lookups
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			toolID := id % 10
			_, err := svc.getForcedToolDefinition(context.Background(), "scenario:"+idString("tool", toolID))
			if err != nil {
				t.Errorf("concurrent lookup %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestIntegration_ConcurrentAsyncGuidance verifies thread-safe async guidance building.
func TestIntegration_ConcurrentAsyncGuidance(t *testing.T) {
	tracker := NewAsyncTrackerService(nil, nil, nil)

	// Add operations
	for i := 0; i < 5; i++ {
		tracker.AddTestOperation(&AsyncOperation{
			ToolCallID: idString("tc", i),
			ChatID:     "chat-concurrent",
			ToolName:   idString("tool", i),
			Status:     "running",
		})
	}

	svc := createTestCompletionService(nil, tracker)

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
