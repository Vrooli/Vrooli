package app

import (
	"testing"

	types "scenario-dependency-analyzer/internal/types"
)

func TestCalculateGraphCentrality(t *testing.T) {
	graph := &types.DependencyGraph{
		Type: "combined",
		Nodes: []types.GraphNode{
			{ID: "core", Type: "scenario", Group: "scenarios"},
			{ID: "api", Type: "scenario", Group: "scenarios"},
			{ID: "app", Type: "scenario", Group: "scenarios"},
			{ID: "cli", Type: "scenario", Group: "scenarios"},
			{ID: "worker", Type: "scenario", Group: "scenarios"},
			{ID: "postgres", Type: "resource", Group: "resources"},
		},
		Edges: []types.GraphEdge{
			{Source: "api", Target: "core", Type: "scenario", Required: true, Weight: 2},
			{Source: "app", Target: "api", Type: "scenario", Required: true, Weight: 2},
			{Source: "worker", Target: "api", Type: "scenario", Required: false, Weight: 1},
			{Source: "cli", Target: "app", Type: "scenario", Required: true, Weight: 2},
			{Source: "app", Target: "postgres", Type: "resource", Required: true, Weight: 2},
		},
	}

	report := calculateGraphCentrality(graph, []string{"core"}, "api")
	if got := len(report.Nodes); got != 1 {
		t.Fatalf("expected one filtered centrality row, got %d", got)
	}
	row := report.Nodes[0]
	if row.Scenario != "api" {
		t.Fatalf("expected api row, got %q", row.Scenario)
	}
	if row.DirectReverseDependencyCount != 2 {
		t.Fatalf("direct reverse dependency count = %d, want 2", row.DirectReverseDependencyCount)
	}
	if row.TransitiveReverseDependencyCount != 3 {
		t.Fatalf("transitive reverse dependency count = %d, want 3", row.TransitiveReverseDependencyCount)
	}
	if row.RequiredReverseDependencyCount != 2 {
		t.Fatalf("required reverse dependency count = %d, want 2", row.RequiredReverseDependencyCount)
	}
	if row.RequiredEdgeWeightedScore != 7 {
		t.Fatalf("required-edge weighted score = %v, want 7", row.RequiredEdgeWeightedScore)
	}
	if row.DistanceToCoreSeed != 1 || row.NearestCoreSeed != "core" {
		t.Fatalf("nearest core = (%d, %q), want (1, core)", row.DistanceToCoreSeed, row.NearestCoreSeed)
	}
}

func TestCalculateGraphCentralityCycleTolerance(t *testing.T) {
	graph := &types.DependencyGraph{
		Type: "combined",
		Nodes: []types.GraphNode{
			{ID: "core", Type: "scenario", Group: "scenarios"},
			{ID: "a", Type: "scenario", Group: "scenarios"},
			{ID: "b", Type: "scenario", Group: "scenarios"},
			{ID: "c", Type: "scenario", Group: "scenarios"},
		},
		Edges: []types.GraphEdge{
			{Source: "a", Target: "b", Type: "scenario", Required: true, Weight: 2},
			{Source: "b", Target: "c", Type: "scenario", Required: true, Weight: 2},
			{Source: "c", Target: "a", Type: "scenario", Required: false, Weight: 1},
			{Source: "a", Target: "core", Type: "scenario", Required: false, Weight: 1},
		},
	}

	report := calculateGraphCentrality(graph, []string{"core"}, "b")
	if got := len(report.Nodes); got != 1 {
		t.Fatalf("expected one filtered centrality row, got %d", got)
	}
	row := report.Nodes[0]
	if row.TransitiveReverseDependencyCount != 2 {
		t.Fatalf("transitive reverse dependency count = %d, want 2", row.TransitiveReverseDependencyCount)
	}
	if row.DistanceToCoreSeed != 2 || row.NearestCoreSeed != "core" {
		t.Fatalf("nearest core = (%d, %q), want (2, core)", row.DistanceToCoreSeed, row.NearestCoreSeed)
	}
}

func TestCalculateGraphCentralityUnknownScenario(t *testing.T) {
	graph := &types.DependencyGraph{
		Type:  "combined",
		Nodes: []types.GraphNode{{ID: "known", Type: "scenario", Group: "scenarios"}},
	}

	report := calculateGraphCentrality(graph, []string{"core"}, "missing")
	if got := len(report.Nodes); got != 0 {
		t.Fatalf("expected no rows for unknown scenario, got %d", got)
	}
}
