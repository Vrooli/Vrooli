package services

import (
	"testing"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

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

func TestGetScenarioNames_WithNilScenario(t *testing.T) {
	registry := &ToolRegistry{
		cachedTools: &domain.ToolSet{
			Scenarios: []*toolspb.ScenarioInfo{
				{Name: "valid"},
				nil,
				{Name: "another"},
			},
		},
	}

	names := registry.getScenarioNames()
	if len(names) != 2 {
		t.Errorf("expected 2 names (skipping nil), got %d", len(names))
	}
}

// Concurrency Tests

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
