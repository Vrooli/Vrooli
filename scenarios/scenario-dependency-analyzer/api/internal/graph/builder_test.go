package graph

import (
	"errors"
	"testing"
	"time"

	"scenario-dependency-analyzer/internal/seams"
	types "scenario-dependency-analyzer/internal/types"
)

type fakeDependencyStore struct {
	deps []types.ScenarioDependency
	err  error
}

func (s fakeDependencyStore) LoadAllDependencies() ([]types.ScenarioDependency, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.deps, nil
}

type fakeScenarioCatalog map[string]bool

func (c fakeScenarioCatalog) KnownScenario(name string) bool {
	return c[name]
}

func TestBuilderGenerateFiltersByGraphTypeAndKnownScenarios(t *testing.T) {
	builder := NewBuilder(
		fakeDependencyStore{deps: []types.ScenarioDependency{
			{ScenarioName: "consumer", DependencyName: "postgres", DependencyType: "resource", Required: true},
			{ScenarioName: "consumer", DependencyName: "core-a", DependencyType: "scenario", Required: true},
			{ScenarioName: "consumer", DependencyName: "stale-scenario", DependencyType: "scenario", Required: true},
		}},
		fakeScenarioCatalog{"core-a": true},
		&seams.Dependencies{
			Clock: seams.NewTestClock(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)),
			IDs:   seams.NewSequentialIDGenerator("graph"),
		},
	)

	graph, err := builder.Generate("combined")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if graph.ID != "graph-1" {
		t.Fatalf("graph ID = %q, want graph-1", graph.ID)
	}
	if graph.Type != "combined" {
		t.Fatalf("graph type = %q, want combined", graph.Type)
	}
	if containsNode(graph.Nodes, "stale-scenario") {
		t.Fatalf("stale scenario reference should be filtered: %#v", graph.Nodes)
	}
	if !containsNode(graph.Nodes, "postgres") || !containsNode(graph.Nodes, "core-a") {
		t.Fatalf("expected resource and known scenario nodes: %#v", graph.Nodes)
	}
	if got, want := len(graph.Edges), 2; got != want {
		t.Fatalf("edges = %d, want %d", got, want)
	}
	if graph.Metadata["total_edges"] != 2 {
		t.Fatalf("metadata total_edges = %#v, want 2", graph.Metadata["total_edges"])
	}
}

func TestBuilderGenerateResourceGraphOnlyIncludesResources(t *testing.T) {
	builder := NewBuilder(
		fakeDependencyStore{deps: []types.ScenarioDependency{
			{ScenarioName: "consumer", DependencyName: "postgres", DependencyType: "resource"},
			{ScenarioName: "consumer", DependencyName: "core-a", DependencyType: "scenario"},
		}},
		nil,
		&seams.Dependencies{
			Clock: seams.NewTestClock(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)),
			IDs:   seams.NewSequentialIDGenerator("graph"),
		},
	)

	graph, err := builder.Generate("resource")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if containsNode(graph.Nodes, "core-a") {
		t.Fatalf("scenario node should be excluded from resource graph: %#v", graph.Nodes)
	}
	if !containsNode(graph.Nodes, "postgres") {
		t.Fatalf("resource node missing from resource graph: %#v", graph.Nodes)
	}
}

func TestBuilderGeneratePropagatesStoreErrors(t *testing.T) {
	builder := NewBuilder(fakeDependencyStore{err: errors.New("store down")}, nil, nil)

	_, err := builder.Generate("combined")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCalculateComplexityScore(t *testing.T) {
	if score := CalculateComplexityScore(nil, nil); score != 0 {
		t.Fatalf("empty graph score = %f, want 0", score)
	}

	nodes := []types.GraphNode{{ID: "a"}, {ID: "b"}}
	edges := []types.GraphEdge{{Source: "a", Target: "b"}}
	if score := CalculateComplexityScore(nodes, edges); score <= 0 || score > 1 {
		t.Fatalf("score = %f, want normalized score in (0, 1]", score)
	}
}

func containsNode(nodes []types.GraphNode, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}
