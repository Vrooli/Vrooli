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

// =============================================================================
// buildToolSet Tests
// =============================================================================

func TestToolRegistry_BuildToolSet_Empty(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{}

	toolSet := registry.buildToolSet(manifests)

	if len(toolSet.Scenarios) != 0 {
		t.Errorf("expected 0 scenarios, got %d", len(toolSet.Scenarios))
	}
	if len(toolSet.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(toolSet.Tools))
	}
	if len(toolSet.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(toolSet.Categories))
	}
	if toolSet.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestToolRegistry_BuildToolSet_NilMetadata(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{
		"test-scenario": {
			ProtocolVersion: "1.0",
			Scenario: &toolspb.ScenarioInfo{
				Name:    "test-scenario",
				Version: "1.0.0",
			},
			Tools: []*toolspb.ToolDefinition{
				{
					Name:        "tool_without_metadata",
					Description: "A tool without metadata",
					// Metadata is nil
				},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	if len(toolSet.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolSet.Tools))
	}

	// Tool without metadata should be disabled by default
	if toolSet.Tools[0].Enabled {
		t.Error("tool without metadata should be disabled by default")
	}
	if toolSet.Tools[0].RequiresApproval {
		t.Error("tool without metadata should not require approval by default")
	}
}

func TestToolRegistry_BuildToolSet_RequiresApproval(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{
		"test-scenario": {
			ProtocolVersion: "1.0",
			Scenario: &toolspb.ScenarioInfo{
				Name: "test-scenario",
			},
			Tools: []*toolspb.ToolDefinition{
				{
					Name:        "safe_tool",
					Description: "A safe tool",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: true,
						RequiresApproval: false,
					},
				},
				{
					Name:        "dangerous_tool",
					Description: "A dangerous tool requiring approval",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: true,
						RequiresApproval: true,
					},
				},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	if len(toolSet.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(toolSet.Tools))
	}

	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == "safe_tool" && tool.RequiresApproval {
			t.Error("safe_tool should not require approval")
		}
		if tool.Tool.Name == "dangerous_tool" && !tool.RequiresApproval {
			t.Error("dangerous_tool should require approval")
		}
	}
}

func TestToolRegistry_BuildToolSet_CategoryDeduplication(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	// Two scenarios with overlapping categories
	manifests := map[string]*toolspb.ToolManifest{
		"scenario-a": {
			ProtocolVersion: "1.0",
			Scenario:        &toolspb.ScenarioInfo{Name: "scenario-a"},
			Tools:           []*toolspb.ToolDefinition{{Name: "tool_a", Category: "shared"}},
			Categories: []*toolspb.ToolCategory{
				{Id: "shared", Name: "Shared Category"},
				{Id: "unique-a", Name: "Unique A"},
			},
		},
		"scenario-b": {
			ProtocolVersion: "1.0",
			Scenario:        &toolspb.ScenarioInfo{Name: "scenario-b"},
			Tools:           []*toolspb.ToolDefinition{{Name: "tool_b", Category: "shared"}},
			Categories: []*toolspb.ToolCategory{
				{Id: "shared", Name: "Shared Category"}, // Same ID as scenario-a
				{Id: "unique-b", Name: "Unique B"},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	// Should have exactly 3 categories (shared is deduplicated)
	if len(toolSet.Categories) != 3 {
		t.Errorf("expected 3 categories (deduplicated), got %d", len(toolSet.Categories))
	}

	// Verify all expected categories exist
	catIDs := make(map[string]bool)
	for _, cat := range toolSet.Categories {
		catIDs[cat.Id] = true
	}
	if !catIDs["shared"] {
		t.Error("missing 'shared' category")
	}
	if !catIDs["unique-a"] {
		t.Error("missing 'unique-a' category")
	}
	if !catIDs["unique-b"] {
		t.Error("missing 'unique-b' category")
	}
}

func TestToolRegistry_BuildToolSet_ScenarioAssignment(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{
		"scenario-alpha": {
			ProtocolVersion: "1.0",
			Scenario:        &toolspb.ScenarioInfo{Name: "scenario-alpha"},
			Tools:           []*toolspb.ToolDefinition{{Name: "tool_from_alpha"}},
		},
		"scenario-beta": {
			ProtocolVersion: "1.0",
			Scenario:        &toolspb.ScenarioInfo{Name: "scenario-beta"},
			Tools:           []*toolspb.ToolDefinition{{Name: "tool_from_beta"}},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	// Verify each tool has correct scenario assignment
	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == "tool_from_alpha" && tool.Scenario != "scenario-alpha" {
			t.Errorf("expected scenario 'scenario-alpha' for tool_from_alpha, got %q", tool.Scenario)
		}
		if tool.Tool.Name == "tool_from_beta" && tool.Scenario != "scenario-beta" {
			t.Errorf("expected scenario 'scenario-beta' for tool_from_beta, got %q", tool.Scenario)
		}
	}
}

// =============================================================================
// difference Helper Tests
// =============================================================================

func TestDifference_Empty(t *testing.T) {
	result := difference([]string{}, []string{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d elements", len(result))
	}
}

func TestDifference_AEmpty(t *testing.T) {
	result := difference([]string{}, []string{"a", "b"})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d elements", len(result))
	}
}

func TestDifference_BEmpty(t *testing.T) {
	result := difference([]string{"a", "b", "c"}, []string{})
	if len(result) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result))
	}
}

func TestDifference_NoOverlap(t *testing.T) {
	result := difference([]string{"a", "b"}, []string{"c", "d"})
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
}

func TestDifference_PartialOverlap(t *testing.T) {
	result := difference([]string{"a", "b", "c"}, []string{"b", "d"})
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	// Should contain 'a' and 'c'
	resultSet := make(map[string]bool)
	for _, s := range result {
		resultSet[s] = true
	}
	if !resultSet["a"] {
		t.Error("expected 'a' in result")
	}
	if !resultSet["c"] {
		t.Error("expected 'c' in result")
	}
	if resultSet["b"] {
		t.Error("'b' should not be in result (it's in both)")
	}
}

func TestDifference_CompleteOverlap(t *testing.T) {
	result := difference([]string{"a", "b"}, []string{"a", "b", "c"})
	if len(result) != 0 {
		t.Errorf("expected 0 elements (all in b), got %d", len(result))
	}
}

// =============================================================================
// getScenarioNames Tests
// =============================================================================

func TestGetScenarioNames_NilCache(t *testing.T) {
	registry := &ToolRegistry{
		cachedTools: nil,
	}

	names := registry.getScenarioNames()
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}

func TestGetScenarioNames_EmptyCache(t *testing.T) {
	registry := &ToolRegistry{
		cachedTools: &domain.ToolSet{
			Scenarios: []*toolspb.ScenarioInfo{},
		},
	}

	names := registry.getScenarioNames()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}
}

func TestGetScenarioNames_WithScenarios(t *testing.T) {
	registry := &ToolRegistry{
		cachedTools: &domain.ToolSet{
			Scenarios: []*toolspb.ScenarioInfo{
				{Name: "alpha"},
				{Name: "beta"},
				{Name: "gamma"},
			},
		},
	}

	names := registry.getScenarioNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["alpha"] || !nameSet["beta"] || !nameSet["gamma"] {
		t.Errorf("missing expected scenario names: %v", names)
	}
}

func TestGetScenarioNames_WithNilScenario(t *testing.T) {
	registry := &ToolRegistry{
		cachedTools: &domain.ToolSet{
			Scenarios: []*toolspb.ScenarioInfo{
				{Name: "valid"},
				nil, // Should be skipped
				{Name: "another"},
			},
		},
	}

	names := registry.getScenarioNames()
	if len(names) != 2 {
		t.Errorf("expected 2 names (skipping nil), got %d", len(names))
	}
}

// =============================================================================
// InternalOnly Filtering Tests (Real Registry)
// =============================================================================

func TestToolRegistry_BuildToolSet_InternalOnlyFlag(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{
		"agent-manager": {
			ProtocolVersion: "1.0",
			Scenario:        &toolspb.ScenarioInfo{Name: "agent-manager"},
			Tools: []*toolspb.ToolDefinition{
				{
					Name:        "spawn_agent",
					Description: "Spawn an agent (public)",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: true,
						InternalOnly:     false,
					},
				},
				{
					Name:        "check_status",
					Description: "Check agent status (internal)",
					Metadata: &toolspb.ToolMetadata{
						EnabledByDefault: true,
						InternalOnly:     true,
					},
				},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	// Both tools should be in the tool set (filtering happens in GetToolsForOpenAI)
	if len(toolSet.Tools) != 2 {
		t.Errorf("expected 2 tools in toolset, got %d", len(toolSet.Tools))
	}

	// Verify internal flag is preserved
	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == "check_status" {
			if tool.Tool.Metadata == nil || !tool.Tool.Metadata.InternalOnly {
				t.Error("check_status should have InternalOnly=true")
			}
		}
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestGetScenarioNames_Concurrent(t *testing.T) {
	registry := &ToolRegistry{
		cachedTools: &domain.ToolSet{
			Scenarios: []*toolspb.ScenarioInfo{
				{Name: "scenario-1"},
				{Name: "scenario-2"},
			},
		},
	}

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			names := registry.getScenarioNames()
			if len(names) != 2 {
				t.Errorf("expected 2 names, got %d", len(names))
			}
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestToolRegistry_BuildToolSet_Concurrent(t *testing.T) {
	registry := &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		cfg:            config.Default(),
	}

	manifests := map[string]*toolspb.ToolManifest{
		"scenario": {
			ProtocolVersion: "1.0",
			Scenario:        &toolspb.ScenarioInfo{Name: "scenario"},
			Tools: []*toolspb.ToolDefinition{
				{Name: "tool_1", Description: "Tool 1"},
				{Name: "tool_2", Description: "Tool 2"},
			},
		},
	}

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() {
			toolSet := registry.buildToolSet(manifests)
			if len(toolSet.Tools) != 2 {
				t.Errorf("expected 2 tools, got %d", len(toolSet.Tools))
			}
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}
