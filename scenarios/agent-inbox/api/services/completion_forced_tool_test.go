package services

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// TestGetForcedToolDefinition_ValidFormat tests the helper that retrieves forced tools.
func TestGetForcedToolDefinition_ValidFormat(t *testing.T) {
	registry := newMockToolRegistry()
	registry.addTool("test-scenario", createSimpleTool("test_tool", "A test tool"))

	svc := &CompletionService{toolRegistry: registry}

	toolDef, err := svc.getForcedToolDefinition(context.Background(), "test-scenario:test_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if toolDef == nil {
		t.Fatal("expected tool definition, got nil")
	}

	fn, ok := toolDef["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function to be a map")
	}
	if fn["name"] != "test_tool" {
		t.Errorf("expected name 'test_tool', got %v", fn["name"])
	}
}

// TestGetForcedToolDefinition_InvalidFormat tests handling of malformed forced tool strings.
func TestGetForcedToolDefinition_InvalidFormat(t *testing.T) {
	registry := newMockToolRegistry()
	svc := &CompletionService{toolRegistry: registry}

	testCases := []struct {
		name       string
		forcedTool string
	}{
		{"no colon", "invalid"},
		{"empty string", ""},
		{"only colon", ":"},
		{"no tool name", "scenario:"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.getForcedToolDefinition(context.Background(), tc.forcedTool)
			if err == nil {
				t.Error("expected error for invalid format")
			}
		})
	}
}

// TestGetForcedToolDefinition_ToolNotFound tests handling when tool doesn't exist.
func TestGetForcedToolDefinition_ToolNotFound(t *testing.T) {
	registry := newMockToolRegistry()
	svc := &CompletionService{toolRegistry: registry}

	_, err := svc.getForcedToolDefinition(context.Background(), "scenario:nonexistent_tool")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

// TestGetForcedToolDefinition_DisabledTool tests that forced tools bypass enabled filters.
func TestGetForcedToolDefinition_DisabledTool(t *testing.T) {
	registry := newMockToolRegistry()
	tool := createSimpleTool("disabled_tool", "A disabled tool")
	registry.ToolsByName["disabled_tool"] = tool
	registry.ToolScenarios["disabled_tool"] = "test-scenario"

	svc := &CompletionService{toolRegistry: registry}

	toolDef, err := svc.getForcedToolDefinition(context.Background(), "test-scenario:disabled_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if toolDef == nil {
		t.Fatal("expected tool definition even for disabled tool")
	}
}

// TestGetForcedToolDefinition_WithMetadata tests tools with metadata.
func TestGetForcedToolDefinition_WithMetadata(t *testing.T) {
	registry := newMockToolRegistry()

	metadata := &toolspb.ToolMetadata{EnabledByDefault: true}
	tool := createToolWithMetadata("internal_tool", "An internal tool", metadata)
	registry.addTool("test-scenario", tool)

	svc := &CompletionService{toolRegistry: registry}

	toolDef, err := svc.getForcedToolDefinition(context.Background(), "test-scenario:internal_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if toolDef == nil {
		t.Fatal("expected tool definition")
	}
}

// TestMockToolRegistry_GetToolByName tests the mock implementation.
func TestMockToolRegistry_GetToolByName(t *testing.T) {
	registry := newMockToolRegistry()
	registry.addTool("scenario-a", createSimpleTool("tool_a", "Tool A"))
	registry.addTool("scenario-b", createSimpleTool("tool_b", "Tool B"))

	ctx := context.Background()

	tool, scenario, err := registry.GetToolByName(ctx, "tool_a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool.Name != "tool_a" {
		t.Errorf("expected tool name 'tool_a', got %q", tool.Name)
	}
	if scenario != "scenario-a" {
		t.Errorf("expected scenario 'scenario-a', got %q", scenario)
	}

	_, _, err = registry.GetToolByName(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

// TestMockToolRegistry_GetToolsForOpenAI tests the mock returns OpenAI formatted tools.
func TestMockToolRegistry_GetToolsForOpenAI(t *testing.T) {
	registry := newMockToolRegistry()
	registry.addTool("scenario", createSimpleTool("tool_1", "Tool 1"))
	registry.addTool("scenario", createSimpleTool("tool_2", "Tool 2"))

	ctx := context.Background()
	tools, err := registry.GetToolsForOpenAI(ctx, "chat-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

// TestMockToolRegistry_GetToolApprovalRequired tests the mock approval behavior.
func TestMockToolRegistry_GetToolApprovalRequired(t *testing.T) {
	registry := newMockToolRegistry()
	registry.ApprovalRequirements["dangerous_tool"] = true
	registry.ApprovalRequirements["safe_tool"] = false

	ctx := context.Background()

	required, _, err := registry.GetToolApprovalRequired(ctx, "chat-123", "dangerous_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !required {
		t.Error("expected approval required for dangerous_tool")
	}

	required, _, err = registry.GetToolApprovalRequired(ctx, "chat-123", "safe_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if required {
		t.Error("expected no approval required for safe_tool")
	}
}
