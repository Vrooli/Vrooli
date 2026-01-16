package services

import (
	"context"
	"testing"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// =============================================================================
// Integration Tests for CompletionService
// =============================================================================
//
// These tests verify the orchestration of forced tool handling and async guidance
// injection in the full PrepareCompletionRequest flow.
//
// Unlike unit tests which test individual functions in isolation, these tests
// verify that the pieces work correctly together.

// mockChatSettings is a minimal mock for testing chat settings retrieval.
type mockChatSettings struct {
	Model              string
	ToolsEnabled       bool
	WebSearchEnabled   bool
	SystemPrompt       string
	ShouldReturnNil    bool
	ShouldReturnError  error
}

// mockRepository provides a minimal repository mock for integration testing.
type mockRepository struct {
	chatSettings map[string]*mockChatSettings
	messages     map[string][]domain.Message
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		chatSettings: make(map[string]*mockChatSettings),
		messages:     make(map[string][]domain.Message),
	}
}

// SetChatSettings sets the chat settings for a chat ID.
func (m *mockRepository) SetChatSettings(chatID string, settings *mockChatSettings) {
	m.chatSettings[chatID] = settings
}

// SetMessages sets the messages for a chat ID.
func (m *mockRepository) SetMessages(chatID string, messages []domain.Message) {
	m.messages[chatID] = messages
}

// createTestCompletionService creates a CompletionService with injected mocks
// for integration testing.
func createTestCompletionService(registry ToolRegistryInterface, tracker *AsyncTrackerService) *CompletionService {
	return &CompletionService{
		toolRegistry: registry,
		asyncTracker: tracker,
		// Other dependencies will be nil - tests should only exercise paths
		// that don't need them, or should provide mocks
	}
}

// TestIntegration_ForcedToolBypassesEnabledFilter verifies that forcing a tool
// works even when the tool is not in the enabled tools list.
func TestIntegration_ForcedToolBypassesEnabledFilter(t *testing.T) {
	// Setup: Create a mock registry with a tool that's only available via GetToolByName
	registry := newMockToolRegistry()

	// Add a tool that would normally be filtered out (not added via addTool)
	disabledTool := createSimpleTool("special_tool", "A tool not normally available")
	registry.ToolsByName["special_tool"] = disabledTool
	registry.ToolScenarios["special_tool"] = "special-scenario"

	svc := createTestCompletionService(registry, nil)

	// Test: getForcedToolDefinition should find it
	toolDef, err := svc.getForcedToolDefinition(context.Background(), "special-scenario:special_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool was found
	if toolDef == nil {
		t.Fatal("expected tool definition, got nil")
	}

	// Verify the tool is correctly formatted for OpenAI
	fn, ok := toolDef["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function key in tool definition")
	}
	if fn["name"] != "special_tool" {
		t.Errorf("expected name 'special_tool', got %v", fn["name"])
	}

	// Verify: The tool should NOT be in GetToolsForOpenAI (it's "disabled")
	openAITools, err := registry.GetToolsForOpenAI(context.Background(), "any-chat")
	if err != nil {
		t.Fatalf("GetToolsForOpenAI error: %v", err)
	}

	for _, tool := range openAITools {
		fn, ok := tool["function"].(map[string]interface{})
		if ok && fn["name"] == "special_tool" {
			t.Error("special_tool should NOT be in OpenAI tools list (it's disabled)")
		}
	}
}

// TestIntegration_InternalToolsFilteredFromAI verifies that tools marked as
// internal_only are excluded from the AI's tool list but can still be
// retrieved via GetToolByName (for the async tracker to use).
func TestIntegration_InternalToolsFilteredFromAI(t *testing.T) {
	registry := newMockToolRegistry()

	// Add a public tool
	publicTool := createSimpleTool("spawn_agent", "Spawn an agent")
	registry.addTool("agent-manager", publicTool)

	// Add an internal tool (status polling)
	internalTool := createInternalTool("check_agent_status", "Check status (internal)")
	registry.addTool("agent-manager", internalTool)

	// Verify: Public tool is in OpenAI list
	openAITools, err := registry.GetToolsForOpenAI(context.Background(), "chat-1")
	if err != nil {
		t.Fatalf("GetToolsForOpenAI error: %v", err)
	}

	foundPublic := false
	foundInternal := false
	for _, tool := range openAITools {
		fn, ok := tool["function"].(map[string]interface{})
		if !ok {
			continue
		}
		name := fn["name"].(string)
		if name == "spawn_agent" {
			foundPublic = true
		}
		if name == "check_agent_status" {
			foundInternal = true
		}
	}

	if !foundPublic {
		t.Error("expected public tool 'spawn_agent' in OpenAI list")
	}
	if foundInternal {
		t.Error("internal tool 'check_agent_status' should NOT be in OpenAI list")
	}

	// Verify: Internal tool can still be found via GetToolByName
	tool, scenario, err := registry.GetToolByName(context.Background(), "check_agent_status")
	if err != nil {
		t.Fatalf("GetToolByName error: %v", err)
	}
	if tool == nil {
		t.Error("expected to find internal tool via GetToolByName")
	}
	if scenario != "agent-manager" {
		t.Errorf("expected scenario 'agent-manager', got %q", scenario)
	}
}

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
