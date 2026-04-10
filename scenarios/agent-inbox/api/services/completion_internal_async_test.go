package services

import (
	"context"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// createInternalTool creates a tool marked as internal_only for testing.
func createInternalTool(name, description string) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault: true,
			InternalOnly:     true,
		},
	}
}

// TestGetToolsForOpenAI_FiltersInternalTools verifies that internal tools
// are filtered out from the list returned to the AI.
func TestGetToolsForOpenAI_FiltersInternalTools(t *testing.T) {
	registry := newMockToolRegistry()

	// Add a public tool
	registry.addTool("agent-manager", createSimpleTool("spawn_coding_agent", "Spawn a coding agent"))

	// Add an internal status tool (should be filtered)
	registry.addTool("agent-manager", createInternalTool("check_agent_status", "Check agent status (internal)"))

	// Add an internal cancellation tool (should be filtered)
	registry.addTool("agent-manager", createInternalTool("cancel_agent", "Cancel agent (internal)"))

	ctx := context.Background()
	tools, err := registry.GetToolsForOpenAI(ctx, "chat-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return 1 tool (the public one)
	if len(tools) != 1 {
		t.Errorf("expected 1 tool (public only), got %d", len(tools))
	}

	// Verify it's the public tool
	if len(tools) > 0 {
		fn, ok := tools[0]["function"].(map[string]interface{})
		if !ok {
			t.Fatal("expected function to be a map")
		}
		if fn["name"] != "spawn_coding_agent" {
			t.Errorf("expected 'spawn_coding_agent', got %v", fn["name"])
		}
	}
}

// TestGetToolsForOpenAI_IncludesPublicAsyncTools verifies that async tools
// without internal_only flag are still included in the AI's tool list.
func TestGetToolsForOpenAI_IncludesPublicAsyncTools(t *testing.T) {
	registry := newMockToolRegistry()

	asyncTool := &toolspb.ToolDefinition{
		Name:        "spawn_coding_agent",
		Description: "Spawn a coding agent that runs asynchronously",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault: true,
			LongRunning:      true,
			InternalOnly:     false,
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:          "check_agent_status",
					PollIntervalSeconds: 5,
				},
			},
		},
	}
	registry.addTool("agent-manager", asyncTool)

	ctx := context.Background()
	tools, err := registry.GetToolsForOpenAI(ctx, "chat-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

// TestGetToolsForOpenAI_AllInternalReturnsEmpty verifies behavior when
// all enabled tools are internal.
func TestGetToolsForOpenAI_AllInternalReturnsEmpty(t *testing.T) {
	registry := newMockToolRegistry()

	registry.addTool("agent-manager", createInternalTool("check_status", "Check status"))
	registry.addTool("agent-manager", createInternalTool("cancel_op", "Cancel operation"))

	ctx := context.Background()
	tools, err := registry.GetToolsForOpenAI(ctx, "chat-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tools) != 0 {
		t.Errorf("expected 0 tools (all internal), got %d", len(tools))
	}
}

// TestGetToolByName_FindsInternalTool verifies that GetToolByName can find
// internal tools (used by async tracker to force tool calls).
func TestGetToolByName_FindsInternalTool(t *testing.T) {
	registry := newMockToolRegistry()

	registry.addTool("agent-manager", createInternalTool("check_agent_status", "Check status"))

	ctx := context.Background()
	tool, scenario, err := registry.GetToolByName(ctx, "check_agent_status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tool == nil {
		t.Fatal("expected to find internal tool")
	}
	if tool.Name != "check_agent_status" {
		t.Errorf("expected 'check_agent_status', got %q", tool.Name)
	}
	if scenario != "agent-manager" {
		t.Errorf("expected scenario 'agent-manager', got %q", scenario)
	}
}

// strContains is a simple helper to check if a string contains a substring.
func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBuildAsyncGuidanceMessage_SingleOp verifies guidance for one active operation.
func TestBuildAsyncGuidanceMessage_SingleOp(t *testing.T) {
	svc := &CompletionService{}

	ops := []*AsyncOperation{
		{ToolName: "spawn_coding_agent"},
	}

	msg := svc.buildAsyncGuidanceMessage(ops)

	if !strContains(msg, "spawn_coding_agent") {
		t.Error("expected message to contain tool name 'spawn_coding_agent'")
	}
	if !strContains(msg, "DO NOT call") {
		t.Error("expected message to instruct not to call status tools")
	}
	if !strContains(msg, "automatically") {
		t.Error("expected message to mention automatic delivery")
	}
}

// TestBuildAsyncGuidanceMessage_MultipleOps verifies guidance for multiple active operations.
func TestBuildAsyncGuidanceMessage_MultipleOps(t *testing.T) {
	svc := &CompletionService{}

	ops := []*AsyncOperation{
		{ToolName: "spawn_coding_agent"},
		{ToolName: "run_browser_automation"},
		{ToolName: "execute_workflow"},
	}

	msg := svc.buildAsyncGuidanceMessage(ops)

	if !strContains(msg, "spawn_coding_agent") {
		t.Error("expected message to contain 'spawn_coding_agent'")
	}
	if !strContains(msg, "run_browser_automation") {
		t.Error("expected message to contain 'run_browser_automation'")
	}
	if !strContains(msg, "execute_workflow") {
		t.Error("expected message to contain 'execute_workflow'")
	}
}

// TestBuildAsyncGuidanceMessage_EmptyOps verifies behavior with no operations.
func TestBuildAsyncGuidanceMessage_EmptyOps(t *testing.T) {
	svc := &CompletionService{}

	var ops []*AsyncOperation

	msg := svc.buildAsyncGuidanceMessage(ops)

	if msg == "" {
		t.Error("expected non-empty message even with no ops")
	}
}
