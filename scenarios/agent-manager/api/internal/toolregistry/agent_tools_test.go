package toolregistry

import (
	"context"
	"testing"
)

func TestAgentToolProvider_Name(t *testing.T) {
	provider := NewAgentToolProvider()

	if provider.Name() != "agent-manager-core" {
		t.Errorf("expected name 'agent-manager-core', got %s", provider.Name())
	}
}

func TestAgentToolProvider_Categories(t *testing.T) {
	provider := NewAgentToolProvider()
	categories := provider.Categories(context.Background())

	if len(categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(categories))
	}

	expectedCategories := map[string]bool{
		"agent_lifecycle": false,
		"agent_status":    false,
		"agent_results":   false,
	}

	for _, cat := range categories {
		if _, ok := expectedCategories[cat.ID]; !ok {
			t.Errorf("unexpected category ID: %s", cat.ID)
		}
		expectedCategories[cat.ID] = true

		// Verify required fields
		if cat.Name == "" {
			t.Errorf("category %s has empty name", cat.ID)
		}
		if cat.Description == "" {
			t.Errorf("category %s has empty description", cat.ID)
		}
	}

	for id, found := range expectedCategories {
		if !found {
			t.Errorf("missing expected category: %s", id)
		}
	}
}

func TestAgentToolProvider_Tools(t *testing.T) {
	provider := NewAgentToolProvider()
	tools := provider.Tools(context.Background())

	if len(tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(tools))
	}

	expectedTools := map[string]struct {
		category string
		required []string
	}{
		"spawn_coding_agent":   {category: "agent_lifecycle", required: []string{"task"}},
		"check_agent_status":   {category: "agent_status", required: []string{"run_id"}},
		"stop_agent":           {category: "agent_lifecycle", required: []string{"run_id"}},
		"list_active_agents":   {category: "agent_status", required: []string{}},
		"get_agent_diff":       {category: "agent_results", required: []string{"run_id"}},
		"approve_agent_changes": {category: "agent_results", required: []string{"run_id"}},
	}

	for _, tool := range tools {
		expected, ok := expectedTools[tool.Name]
		if !ok {
			t.Errorf("unexpected tool: %s", tool.Name)
			continue
		}

		// Verify category
		if tool.Category != expected.category {
			t.Errorf("tool %s: expected category %s, got %s", tool.Name, expected.category, tool.Category)
		}

		// Verify description is set
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}

		// Verify parameters
		if tool.Parameters.Type != "object" {
			t.Errorf("tool %s: expected parameters type 'object', got %s", tool.Name, tool.Parameters.Type)
		}

		// Verify required parameters
		requiredSet := make(map[string]bool)
		for _, r := range tool.Parameters.Required {
			requiredSet[r] = true
		}
		for _, r := range expected.required {
			if !requiredSet[r] {
				t.Errorf("tool %s: missing required parameter %s", tool.Name, r)
			}
		}

		// Verify metadata
		if tool.Metadata.TimeoutSeconds <= 0 && tool.Name != "list_active_agents" {
			t.Errorf("tool %s: expected positive timeout", tool.Name)
		}
	}
}

func TestAgentToolProvider_SpawnCodingAgent_Schema(t *testing.T) {
	provider := NewAgentToolProvider()
	tools := provider.Tools(context.Background())

	var spawnTool *struct {
		name   string
		params map[string]interface{}
	}
	for _, tool := range tools {
		if tool.Name == "spawn_coding_agent" {
			spawnTool = &struct {
				name   string
				params map[string]interface{}
			}{
				name: tool.Name,
			}
			break
		}
	}

	if spawnTool == nil {
		t.Fatal("spawn_coding_agent tool not found")
	}
}

func TestAgentToolProvider_ToolMetadata(t *testing.T) {
	provider := NewAgentToolProvider()
	tools := provider.Tools(context.Background())

	for _, tool := range tools {
		// Check that each tool has appropriate metadata
		meta := tool.Metadata

		// spawn_coding_agent should be long-running
		if tool.Name == "spawn_coding_agent" {
			if !meta.LongRunning {
				t.Errorf("spawn_coding_agent should be marked as long-running")
			}
			if meta.CostEstimate != "high" {
				t.Errorf("spawn_coding_agent should have high cost estimate")
			}
		}

		// Status tools should be idempotent and low cost
		if tool.Name == "check_agent_status" || tool.Name == "list_active_agents" {
			if !meta.Idempotent {
				t.Errorf("%s should be idempotent", tool.Name)
			}
			if meta.CostEstimate != "low" {
				t.Errorf("%s should have low cost estimate", tool.Name)
			}
		}

		// approve_agent_changes requires approval
		if tool.Name == "approve_agent_changes" {
			if !meta.RequiresApproval {
				t.Errorf("approve_agent_changes should require approval")
			}
		}

		// All tools should be enabled by default
		if !meta.EnabledByDefault {
			t.Errorf("tool %s should be enabled by default", tool.Name)
		}
	}
}

func TestAgentToolProvider_ToolExamples(t *testing.T) {
	provider := NewAgentToolProvider()
	tools := provider.Tools(context.Background())

	for _, tool := range tools {
		examples := tool.Metadata.Examples

		if len(examples) == 0 {
			t.Errorf("tool %s should have at least one example", tool.Name)
			continue
		}

		for i, example := range examples {
			if example.Description == "" {
				t.Errorf("tool %s example %d has empty description", tool.Name, i)
			}
			if example.Input == nil {
				t.Errorf("tool %s example %d has nil input", tool.Name, i)
			}
		}
	}
}
