package services

import (
	"testing"

	"agent-inbox/config"
	"agent-inbox/integrations"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

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
				},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	if len(toolSet.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolSet.Tools))
	}

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
				{Id: "shared", Name: "Shared Category"},
				{Id: "unique-b", Name: "Unique B"},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	if len(toolSet.Categories) != 3 {
		t.Errorf("expected 3 categories (deduplicated), got %d", len(toolSet.Categories))
	}

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

	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == "tool_from_alpha" && tool.Scenario != "scenario-alpha" {
			t.Errorf("expected scenario 'scenario-alpha' for tool_from_alpha, got %q", tool.Scenario)
		}
		if tool.Tool.Name == "tool_from_beta" && tool.Scenario != "scenario-beta" {
			t.Errorf("expected scenario 'scenario-beta' for tool_from_beta, got %q", tool.Scenario)
		}
	}
}

// difference helper tests

func TestDifference_Empty(t *testing.T) {
	result := difference([]string{}, []string{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d elements", len(result))
	}
}

func TestDifference_PartialOverlap(t *testing.T) {
	result := difference([]string{"a", "b", "c"}, []string{"b", "d"})
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
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
}

func TestDifference_CompleteOverlap(t *testing.T) {
	result := difference([]string{"a", "b"}, []string{"a", "b", "c"})
	if len(result) != 0 {
		t.Errorf("expected 0 elements (all in b), got %d", len(result))
	}
}

// InternalOnly Filtering Tests

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
					Metadata:    &toolspb.ToolMetadata{EnabledByDefault: true, InternalOnly: false},
				},
				{
					Name:        "check_status",
					Description: "Check agent status (internal)",
					Metadata:    &toolspb.ToolMetadata{EnabledByDefault: true, InternalOnly: true},
				},
			},
		},
	}

	toolSet := registry.buildToolSet(manifests)

	if len(toolSet.Tools) != 2 {
		t.Errorf("expected 2 tools in toolset, got %d", len(toolSet.Tools))
	}

	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == "check_status" {
			if tool.Tool.Metadata == nil || !tool.Tool.Metadata.InternalOnly {
				t.Error("check_status should have InternalOnly=true")
			}
		}
	}
}

// getScenarioNames Tests

func TestGetScenarioNames_NilCache(t *testing.T) {
	registry := &ToolRegistry{cachedTools: nil}
	names := registry.getScenarioNames()
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}
