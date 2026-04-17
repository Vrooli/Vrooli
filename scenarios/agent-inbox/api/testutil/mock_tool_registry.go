// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"agent-inbox/domain"
	"context"
	"fmt"
	"sync"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// MockToolRegistry provides a controllable tool registry for testing.
// It implements the methods used by CompletionService.
type MockToolRegistry struct {
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
	GetToolApprovalCalls   []GetToolApprovalCall
}

// GetToolApprovalCall records a call to GetToolApprovalRequired.
type GetToolApprovalCall struct {
	ChatID   string
	ToolName string
}

// NewMockToolRegistry creates a new mock tool registry for testing.
func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		ToolsByName:          make(map[string]*toolspb.ToolDefinition),
		ToolScenarios:        make(map[string]string),
		ApprovalRequirements: make(map[string]bool),
	}
}

// AddTool adds a tool to the registry for testing.
func (m *MockToolRegistry) AddTool(scenario string, tool *toolspb.ToolDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ToolsByName[tool.Name] = tool
	m.ToolScenarios[tool.Name] = scenario
	// Also add to OpenAI tools list
	m.OpenAITools = append(m.OpenAITools, domain.ToOpenAIFunction(tool))
}

// GetToolByName returns a tool by name, bypassing enabled filters.
func (m *MockToolRegistry) GetToolByName(ctx context.Context, toolName string) (*toolspb.ToolDefinition, string, error) {
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

// GetToolsForOpenAI returns enabled tools in OpenAI format.
func (m *MockToolRegistry) GetToolsForOpenAI(ctx context.Context, chatID string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolsForOpenAICalls = append(m.GetToolsForOpenAICalls, chatID)

	if m.GetToolsForOpenAIError != nil {
		return nil, m.GetToolsForOpenAIError
	}

	return m.OpenAITools, nil
}

// GetToolApprovalRequired checks if a tool requires approval.
func (m *MockToolRegistry) GetToolApprovalRequired(ctx context.Context, chatID, toolName string) (bool, domain.ToolConfigurationScope, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetToolApprovalCalls = append(m.GetToolApprovalCalls, GetToolApprovalCall{
		ChatID:   chatID,
		ToolName: toolName,
	})

	if m.GetToolApprovalError != nil {
		return false, "", m.GetToolApprovalError
	}

	required := m.ApprovalRequirements[toolName]
	return required, "", nil
}

// Reset clears all recorded calls and state.
func (m *MockToolRegistry) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetToolByNameCalls = nil
	m.GetToolsForOpenAICalls = nil
	m.GetToolApprovalCalls = nil
}

// CreateSimpleTool creates a simple tool definition for testing.
func CreateSimpleTool(name, description string) *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        name,
		Description: description,
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: make(map[string]*toolspb.ParameterSchema),
		},
	}
}

// CreateToolWithMetadata creates a tool definition with metadata for testing.
func CreateToolWithMetadata(name, description string, metadata *toolspb.ToolMetadata) *toolspb.ToolDefinition {
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
