package services

import (
	"context"
	"fmt"
	"sync"

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
