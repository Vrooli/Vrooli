package services

import (
	"context"
	"testing"
	"time"

	"agent-inbox/config"
	"agent-inbox/domain"
	"agent-inbox/integrations"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// mockScenarioClient is a test double for integrations.ScenarioClient
type mockScenarioClient struct {
	manifests map[string]*toolspb.ToolManifest
	errors    map[string]error
}

func (m *mockScenarioClient) FetchToolManifest(ctx context.Context, scenarioName string) (*toolspb.ToolManifest, error) {
	if err, ok := m.errors[scenarioName]; ok {
		return nil, err
	}
	if manifest, ok := m.manifests[scenarioName]; ok {
		return manifest, nil
	}
	return nil, nil
}

func (m *mockScenarioClient) FetchMultiple(ctx context.Context, scenarioNames []string) (map[string]*toolspb.ToolManifest, map[string]error) {
	results := make(map[string]*toolspb.ToolManifest)
	errors := make(map[string]error)

	for _, name := range scenarioNames {
		if err, ok := m.errors[name]; ok {
			errors[name] = err
		} else if manifest, ok := m.manifests[name]; ok {
			results[name] = manifest
		}
	}

	return results, errors
}

func (m *mockScenarioClient) CheckScenarioStatus(ctx context.Context, scenarioName string) *domain.ScenarioStatus {
	status := &domain.ScenarioStatus{
		Scenario:    scenarioName,
		LastChecked: time.Now(),
	}

	if _, ok := m.errors[scenarioName]; ok {
		status.Available = false
		status.Error = "mock error"
	} else if manifest, ok := m.manifests[scenarioName]; ok {
		status.Available = true
		status.ToolCount = len(manifest.Tools)
	}

	return status
}

func (m *mockScenarioClient) InvalidateCache(scenarioName string) {}
func (m *mockScenarioClient) InvalidateAllCache()                 {}

func TestToolRegistry_BuildToolSet(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{
		"agent-manager": {
			ProtocolVersion: "1.0",
			Scenario: &toolspb.ScenarioInfo{
				Name:    "agent-manager",
				Version: "1.0.0",
			},
			Tools: []*toolspb.ToolDefinition{
				{
					Name:        "spawn_coding_agent",
					Description: "Spawn a coding agent",
					Category:    "lifecycle",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: true,
					},
				},
				{
					Name:        "check_agent_status",
					Description: "Check agent status",
					Category:    "status",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: true,
					},
				},
			},
			Categories: []*toolspb.ToolCategory{
				{Id: "lifecycle", Name: "Lifecycle"},
				{Id: "status", Name: "Status"},
			},
		},
		"research-agent": {
			ProtocolVersion: "1.0",
			Scenario: &toolspb.ScenarioInfo{
				Name:    "research-agent",
				Version: "1.0.0",
			},
			Tools: []*toolspb.ToolDefinition{
				{
					Name:        "web_search",
					Description: "Search the web",
					Category:    "search",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: false,
					},
				},
			},
			Categories: []*toolspb.ToolCategory{
				{Id: "search", Name: "Search"},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	// Check scenarios
	if len(toolSet.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(toolSet.Scenarios))
	}

	// Check tools
	if len(toolSet.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(toolSet.Tools))
	}

	// Check categories (should be deduplicated)
	if len(toolSet.Categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(toolSet.Categories))
	}

	// Verify enabled states are set from metadata
	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == "web_search" && tool.Enabled {
			t.Error("web_search should be disabled by default")
		}
		if tool.Tool.Name == "spawn_coding_agent" && !tool.Enabled {
			t.Error("spawn_coding_agent should be enabled by default")
		}
	}
}

func TestToolDefinition_ToOpenAIFunction(t *testing.T) {
	tool := &toolspb.ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"param1": {
					Type:        "string",
					Description: "A parameter",
				},
			},
			Required: []string{"param1"},
		},
	}

	result := domain.ToOpenAIFunction(tool)

	// Check type
	if result["type"] != "function" {
		t.Errorf("expected type 'function', got %v", result["type"])
	}

	// Check function object
	fn, ok := result["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function to be a map")
	}

	if fn["name"] != "test_tool" {
		t.Errorf("expected name 'test_tool', got %v", fn["name"])
	}

	if fn["description"] != "A test tool" {
		t.Errorf("expected description 'A test tool', got %v", fn["description"])
	}

	params, ok := fn["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("expected parameters to be a map")
	}

	if params["type"] != "object" {
		t.Errorf("expected parameters type 'object', got %v", params["type"])
	}
}

func TestEffectiveTool_Source(t *testing.T) {
	tests := []struct {
		name     string
		source   domain.ToolConfigurationScope
		expected string
	}{
		{"global", domain.ScopeGlobal, "global"},
		{"chat", domain.ScopeChat, "chat"},
		{"empty (default)", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := domain.EffectiveTool{Source: tt.source}
			if string(tool.Source) != tt.expected {
				t.Errorf("expected source %q, got %q", tt.expected, tool.Source)
			}
		})
	}
}

func TestToolConfigurationScope_Values(t *testing.T) {
	if domain.ScopeGlobal != "global" {
		t.Errorf("expected ScopeGlobal to be 'global', got %q", domain.ScopeGlobal)
	}
	if domain.ScopeChat != "chat" {
		t.Errorf("expected ScopeChat to be 'chat', got %q", domain.ScopeChat)
	}
}
