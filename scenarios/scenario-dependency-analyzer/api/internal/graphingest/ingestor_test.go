package graphingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"scenario-dependency-analyzer/internal/interfacegraph"
	types "scenario-dependency-analyzer/internal/types"
)

type fakeGraphSource struct {
	graph interfacegraph.Graph
	stats interfacegraph.BuildStats
	err   error
}

func (f fakeGraphSource) BuildWithStats(ctx context.Context, req interfacegraph.BuildRequest) (interfacegraph.Graph, interfacegraph.BuildStats, error) {
	return f.graph, f.stats, f.err
}

type fakeDeclared struct {
	all     []types.ScenarioDependency
	buckets map[string]map[string][]types.ScenarioDependency
}

func (f fakeDeclared) LoadAllDependencies() ([]types.ScenarioDependency, error) { return f.all, nil }
func (f fakeDeclared) LoadStoredDependencies(scenario string) (map[string][]types.ScenarioDependency, error) {
	return f.buckets[scenario], nil
}

type fakeAnalyze struct {
	scenarioCalls int
	allCalls      int
}

func (f *fakeAnalyze) AnalyzeScenario(string) (*types.DependencyAnalysisResponse, error) {
	f.scenarioCalls++
	return &types.DependencyAnalysisResponse{}, nil
}

func (f *fakeAnalyze) AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	f.allCalls++
	return map[string]*types.DependencyAnalysisResponse{"consumer": {}}, nil
}

type fakePersist struct {
	replaced    []types.UnifiedGraphEdge
	upserted    map[string][]types.UnifiedGraphEdge
	markedStale []string
}

func (f *fakePersist) ReplaceGraphEdges(edges []types.UnifiedGraphEdge) error {
	f.replaced = edges
	return nil
}

func (f *fakePersist) UpsertGraphEdgesForScenario(scenario string, edges []types.UnifiedGraphEdge) error {
	if f.upserted == nil {
		f.upserted = map[string][]types.UnifiedGraphEdge{}
	}
	f.upserted[scenario] = edges
	return nil
}

func (f *fakePersist) MarkScenarioEdgesStale(scenario string) error {
	f.markedStale = append(f.markedStale, scenario)
	return nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestRebuildFleetMergesAndPersistsWhenApplied(t *testing.T) {
	graph := interfacegraph.Graph{Edges: []interfacegraph.Edge{
		{FromScenario: "consumer", ToScenario: "core-a", Evidence: []interfacegraph.Evidence{{Source: interfacegraph.EvidenceProtoImport, FromFile: "x.proto"}}},
	}}
	declared := fakeDeclared{all: []types.ScenarioDependency{
		{ScenarioName: "consumer", DependencyName: "postgres", DependencyType: "resource", Required: true},
		{ScenarioName: "consumer", DependencyName: "core-a", DependencyType: "scenario", AccessMethod: "declared", Required: true},
	}}
	analyze := &fakeAnalyze{}
	persist := &fakePersist{}
	ing := NewIngestor(fakeGraphSource{graph: graph}, declared, analyze, persist, WithClock(fixedClock{time.Now()}))

	report, err := ing.RebuildFleet(context.Background(), "/repo", true)
	if err != nil {
		t.Fatalf("RebuildFleet: %v", err)
	}
	if analyze.allCalls != 1 {
		t.Fatalf("expected AnalyzeAllScenarios called once, got %d", analyze.allCalls)
	}
	if report.EdgesPersisted != 2 {
		t.Fatalf("edges persisted = %d, want 2", report.EdgesPersisted)
	}
	if report.ScenarioEdges != 1 || report.ResourceEdges != 1 {
		t.Fatalf("edge kinds = scenario %d / resource %d, want 1/1", report.ScenarioEdges, report.ResourceEdges)
	}
	if len(persist.replaced) != 2 {
		t.Fatalf("persisted %d edges, want 2", len(persist.replaced))
	}
	// core-a edge should carry proto_import (highest confidence) over declared.
	for _, e := range persist.replaced {
		if e.To == "core-a" && e.Source != SourceProtoImport {
			t.Fatalf("core-a edge source = %q, want proto_import", e.Source)
		}
	}
	if report.Metrics == nil || report.Metrics.GetWallClockMs() < 0 {
		t.Fatalf("expected execution metrics on report")
	}
}

func TestRebuildFleetDryRunDoesNotPersist(t *testing.T) {
	persist := &fakePersist{}
	ing := NewIngestor(fakeGraphSource{}, fakeDeclared{}, &fakeAnalyze{}, persist)
	if _, err := ing.RebuildFleet(context.Background(), "/repo", false); err != nil {
		t.Fatalf("RebuildFleet dry-run: %v", err)
	}
	if persist.replaced != nil {
		t.Fatalf("dry-run must not persist, got %d edges", len(persist.replaced))
	}
}

func TestIngestScenarioRetainsLastGoodOnSourceOutage(t *testing.T) {
	persist := &fakePersist{}
	ing := NewIngestor(fakeGraphSource{err: errors.New("proto-health down")}, fakeDeclared{}, &fakeAnalyze{}, persist)

	result, err := ing.IngestScenario(context.Background(), "/repo", "consumer", true)
	if err == nil {
		t.Fatalf("expected error on source outage")
	}
	if !result.Degraded {
		t.Fatalf("expected degraded result")
	}
	if len(persist.markedStale) != 1 || persist.markedStale[0] != "consumer" {
		t.Fatalf("expected consumer marked stale, got %v", persist.markedStale)
	}
	if persist.upserted["consumer"] != nil {
		t.Fatalf("must not upsert during outage")
	}
}

func TestIngestScenarioUpsertsOnlyFromScenarioEdges(t *testing.T) {
	graph := interfacegraph.Graph{Edges: []interfacegraph.Edge{
		{FromScenario: "consumer", ToScenario: "core-a", Evidence: []interfacegraph.Evidence{{Source: interfacegraph.EvidenceGoImport, ImportPath: "x/core-a"}}},
	}}
	declared := fakeDeclared{buckets: map[string]map[string][]types.ScenarioDependency{
		"consumer": {
			"resources": {{ScenarioName: "consumer", DependencyName: "redis", DependencyType: "resource"}},
		},
	}}
	persist := &fakePersist{}
	ing := NewIngestor(fakeGraphSource{graph: graph}, declared, &fakeAnalyze{}, persist)

	result, err := ing.IngestScenario(context.Background(), "/repo", "consumer", true)
	if err != nil {
		t.Fatalf("IngestScenario: %v", err)
	}
	edges := persist.upserted["consumer"]
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges for consumer, got %d", len(edges))
	}
	for _, e := range edges {
		if e.From != "consumer" {
			t.Fatalf("upsert window must contain only consumer-origin edges, got %q", e.From)
		}
	}
	if result.EdgesPersisted != 2 {
		t.Fatalf("edges persisted = %d, want 2", result.EdgesPersisted)
	}
}
