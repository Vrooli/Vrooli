package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// mockToolRegistry implements ToolRegistryInterface for testing.
type mockToolRegistry struct {
	mu sync.Mutex

	// ToolsByName maps tool names to their definitions
	ToolsByName map[string]*toolspb.ToolDefinition

	// ToolScenarios maps tool names to their scenario names
	ToolScenarios map[string]string

	// OpenAITools is the list returned by GetToolsForOpenAI
	OpenAITools []map[string]interface{}

	// ApprovalRequirements maps tool names to approval requirements
	ApprovalRequirements map[string]bool

	// Error responses for testing error paths
	GetToolByNameError     error
	GetToolsForOpenAIError error
	GetToolApprovalError   error

	// Call tracking
	GetToolByNameCalls     []string
	GetToolsForOpenAICalls []string
}

func newMockToolRegistry() *mockToolRegistry {
	return &mockToolRegistry{
		ToolsByName:          make(map[string]*toolspb.ToolDefinition),
		ToolScenarios:        make(map[string]string),
		ApprovalRequirements: make(map[string]bool),
	}
}

func (m *mockToolRegistry) addTool(scenario string, tool *toolspb.ToolDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ToolsByName[tool.Name] = tool
	m.ToolScenarios[tool.Name] = scenario
	m.OpenAITools = append(m.OpenAITools, domain.ToOpenAIFunction(tool))
}

func (m *mockToolRegistry) GetToolByName(ctx context.Context, toolName string) (*toolspb.ToolDefinition, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolByNameCalls = append(m.GetToolByNameCalls, toolName)

	if m.GetToolByNameError != nil {
		return nil, "", m.GetToolByNameError
	}

	tool, ok := m.ToolsByName[toolName]
	if !ok {
		return nil, "", fmt.Errorf("tool not found: %s", toolName)
	}

	scenario := m.ToolScenarios[toolName]
	return tool, scenario, nil
}

func (m *mockToolRegistry) GetToolsForOpenAI(ctx context.Context, chatID string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolsForOpenAICalls = append(m.GetToolsForOpenAICalls, chatID)

	if m.GetToolsForOpenAIError != nil {
		return nil, m.GetToolsForOpenAIError
	}

	// Filter internal tools, matching real implementation behavior
	var result []map[string]interface{}
	for _, tool := range m.OpenAITools {
		fn, ok := tool["function"].(map[string]interface{})
		if !ok {
			continue
		}
		toolName, _ := fn["name"].(string)
		if toolDef, exists := m.ToolsByName[toolName]; exists {
			if toolDef.Metadata != nil && toolDef.Metadata.InternalOnly {
				continue
			}
		}
		result = append(result, tool)
	}

	return result, nil
}

func (m *mockToolRegistry) GetToolApprovalRequired(ctx context.Context, chatID, toolName string) (bool, domain.ToolConfigurationScope, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetToolApprovalError != nil {
		return false, "", m.GetToolApprovalError
	}

	required := m.ApprovalRequirements[toolName]
	return required, "", nil
}

// createSimpleTool creates a simple tool definition for testing.
func createSimpleTool(name, description string) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
	}
}

// createToolWithMetadata creates a tool definition with metadata for testing.
func createToolWithMetadata(name, description string, metadata *toolspb.ToolMetadata) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
		Metadata: metadata,
	}
}

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

	// Check the function name
	fn, ok := toolDef["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function to be a map")
	}
	if fn["name"] != "test_tool" {
		t.Errorf("expected name 'test_tool', got %v", fn["name"])
	}

	// Verify GetToolByName was called
	if len(registry.GetToolByNameCalls) != 1 {
		t.Errorf("expected 1 GetToolByName call, got %d", len(registry.GetToolByNameCalls))
	}
	if registry.GetToolByNameCalls[0] != "test_tool" {
		t.Errorf("expected call with 'test_tool', got %q", registry.GetToolByNameCalls[0])
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
	// Add a tool but don't include it in OpenAITools (simulating disabled)
	tool := createSimpleTool("disabled_tool", "A disabled tool")
	registry.ToolsByName["disabled_tool"] = tool
	registry.ToolScenarios["disabled_tool"] = "test-scenario"
	// Note: NOT calling registry.addTool() so it's not in OpenAITools list

	svc := &CompletionService{toolRegistry: registry}

	// getForcedToolDefinition should still find it because it uses GetToolByName
	// which bypasses the enabled filter
	toolDef, err := svc.getForcedToolDefinition(context.Background(), "test-scenario:disabled_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if toolDef == nil {
		t.Fatal("expected tool definition even for disabled tool")
	}

	fn, ok := toolDef["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function map")
	}
	if fn["name"] != "disabled_tool" {
		t.Errorf("expected 'disabled_tool', got %v", fn["name"])
	}
}

// TestGetForcedToolDefinition_WithMetadata tests tools with metadata.
func TestGetForcedToolDefinition_WithMetadata(t *testing.T) {
	registry := newMockToolRegistry()

	// Add a tool with metadata
	metadata := &toolspb.ToolMetadata{
		EnabledByDefault: true,
	}
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

	// Test finding existing tool
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

	// Test tool not found
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

	// Verify call was tracked
	if len(registry.GetToolsForOpenAICalls) != 1 {
		t.Errorf("expected 1 call, got %d", len(registry.GetToolsForOpenAICalls))
	}
	if registry.GetToolsForOpenAICalls[0] != "chat-123" {
		t.Errorf("expected chatID 'chat-123', got %q", registry.GetToolsForOpenAICalls[0])
	}
}

// TestMockToolRegistry_GetToolApprovalRequired tests the mock approval behavior.
func TestMockToolRegistry_GetToolApprovalRequired(t *testing.T) {
	registry := newMockToolRegistry()
	registry.ApprovalRequirements["dangerous_tool"] = true
	registry.ApprovalRequirements["safe_tool"] = false

	ctx := context.Background()

	// Test tool requiring approval
	required, _, err := registry.GetToolApprovalRequired(ctx, "chat-123", "dangerous_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !required {
		t.Error("expected approval required for dangerous_tool")
	}

	// Test tool not requiring approval
	required, _, err = registry.GetToolApprovalRequired(ctx, "chat-123", "safe_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if required {
		t.Error("expected no approval required for safe_tool")
	}
}

// =============================================================================
// Internal Tool Filtering Tests
// =============================================================================

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

	// Add an async tool that is NOT marked as internal (should be included)
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
			InternalOnly:     false, // Explicitly NOT internal
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

	// Should include the async tool
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	// Verify it's the async tool
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

// TestGetToolsForOpenAI_AllInternalReturnsEmpty verifies behavior when
// all enabled tools are internal.
func TestGetToolsForOpenAI_AllInternalReturnsEmpty(t *testing.T) {
	registry := newMockToolRegistry()

	// Add only internal tools
	registry.addTool("agent-manager", createInternalTool("check_status", "Check status"))
	registry.addTool("agent-manager", createInternalTool("cancel_op", "Cancel operation"))

	ctx := context.Background()
	tools, err := registry.GetToolsForOpenAI(ctx, "chat-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty slice
	if len(tools) != 0 {
		t.Errorf("expected 0 tools (all internal), got %d", len(tools))
	}
}

// TestGetToolByName_FindsInternalTool verifies that GetToolByName can find
// internal tools (used by async tracker to force tool calls).
func TestGetToolByName_FindsInternalTool(t *testing.T) {
	registry := newMockToolRegistry()

	// Add an internal tool
	registry.addTool("agent-manager", createInternalTool("check_agent_status", "Check status"))

	ctx := context.Background()
	tool, scenario, err := registry.GetToolByName(ctx, "check_agent_status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find the internal tool
	if tool == nil {
		t.Fatal("expected to find internal tool")
	}
	if tool.Name != "check_agent_status" {
		t.Errorf("expected 'check_agent_status', got %q", tool.Name)
	}
	if scenario != "agent-manager" {
		t.Errorf("expected scenario 'agent-manager', got %q", scenario)
	}

	// Verify metadata
	if tool.Metadata == nil || !tool.Metadata.InternalOnly {
		t.Error("expected tool to be marked as internal")
	}
}

// =============================================================================
// Async Guidance Message Tests
// =============================================================================

// TestBuildAsyncGuidanceMessage_SingleOp verifies guidance for one active operation.
func TestBuildAsyncGuidanceMessage_SingleOp(t *testing.T) {
	svc := &CompletionService{}

	ops := []*AsyncOperation{
		{ToolName: "spawn_coding_agent"},
	}

	msg := svc.buildAsyncGuidanceMessage(ops)

	// Should mention the tool name
	if !strContains(msg, "spawn_coding_agent") {
		t.Error("expected message to contain tool name 'spawn_coding_agent'")
	}

	// Should instruct not to poll
	if !strContains(msg, "DO NOT call") {
		t.Error("expected message to instruct not to call status tools")
	}

	// Should mention automatic delivery
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

	// Should mention all tool names
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

	var ops []*AsyncOperation // nil/empty

	msg := svc.buildAsyncGuidanceMessage(ops)

	// Should still produce a valid message (empty tool list)
	if msg == "" {
		t.Error("expected non-empty message even with no ops")
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
