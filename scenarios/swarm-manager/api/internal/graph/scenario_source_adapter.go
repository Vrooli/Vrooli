package graph

import (
	"context"
	"strings"
	"swarm-manager/internal/scenarios"
)

// ScenarioInventorySource loads scenario inventory from the scenarios package.
type ScenarioInventorySource interface {
	List(ctx context.Context) ([]scenarios.ScenarioSource, error)
}

// ScenarioSourceAdapter normalizes scenario inventory into the graph contract.
type ScenarioSourceAdapter struct {
	source ScenarioInventorySource
}

// NewScenarioSourceAdapter creates a graph-facing scenario lister that
// normalizes inventory status into the graph's expected status vocabulary.
func NewScenarioSourceAdapter(source ScenarioInventorySource) *ScenarioSourceAdapter {
	return &ScenarioSourceAdapter{source: source}
}

func normalizeGraphScenarioStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return "running"
	case "stopped", "available":
		return "stopped"
	case "error":
		return "error"
	default:
		return "unknown"
	}
}

// List adapts scenario inventory entries into graph scenario nodes.
func (a *ScenarioSourceAdapter) List(ctx context.Context) ([]ScenarioEntry, error) {
	if a == nil || a.source == nil {
		return nil, nil
	}

	sources, err := a.source.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]ScenarioEntry, 0, len(sources))
	for _, source := range sources {
		result = append(result, ScenarioEntry{
			Name:   source.Name,
			Status: normalizeGraphScenarioStatus(source.Status),
		})
	}
	return result, nil
}
