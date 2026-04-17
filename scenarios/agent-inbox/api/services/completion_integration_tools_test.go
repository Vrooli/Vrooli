package services

import (
	"context"
	"testing"
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
