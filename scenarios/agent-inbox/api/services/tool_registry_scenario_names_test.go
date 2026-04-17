package services

import (
	"agent-inbox/domain"
	"testing"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

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
}
