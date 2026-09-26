package interfacegraph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBuilderBuildMergesProtoAndGoImportEvidence(t *testing.T) {
	builder := NewBuilder(
		fakeProtoClient{resp: &ProtoSurfaceResponse{Results: []ProtoSurfaceResult{
			{
				Scenario: "alpha",
				Surface: ProtoSurface{
					Scenario:       "alpha",
					TransportWorld: "TRANSPORT_WORLD_CONNECT",
					Files: []ProtoFile{
						{Path: "packages/proto/schemas/bravo/v1/bravo.proto", Stability: "stable"},
					},
					CrossScenarioImports: []ProtoImport{
						{
							FromFile:     "packages/proto/schemas/alpha/v1/alpha.proto",
							ToFile:       "packages/proto/schemas/bravo/v1/bravo.proto",
							FromScenario: "alpha",
							ToScenario:   "bravo",
						},
					},
				},
			},
			{Scenario: "bravo", Surface: ProtoSurface{Scenario: "bravo"}},
		}}},
		fakeImportClient{resp: &ImportFactsResponse{Results: []ImportFactsResult{
			{
				Scenario: "alpha",
				Facts: []ImportFact{
					{
						ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/bravo/v1/bravo",
						Path:       "scenarios/alpha/api/internal/bravo/client.go",
						Analyzer:   "go-code-graph",
					},
				},
			},
			{Scenario: "bravo"},
		}}},
	)

	graph, err := builder.Build(context.Background(), BuildRequest{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(graph.Errors) != 0 {
		t.Fatalf("errors = %v, want none", graph.Errors)
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("edges = %d, want 1: %#v", len(graph.Edges), graph.Edges)
	}
	edge := graph.Edges[0]
	if edge.FromScenario != "alpha" || edge.ToScenario != "bravo" {
		t.Fatalf("edge = %s -> %s, want alpha -> bravo", edge.FromScenario, edge.ToScenario)
	}
	if edge.TransportWorld != "connect" {
		t.Fatalf("transport = %q, want connect", edge.TransportWorld)
	}
	if len(edge.Stability) != 1 || edge.Stability[0] != "stable" {
		t.Fatalf("stability = %v, want [stable]", edge.Stability)
	}
	gotEvidence := map[string]bool{}
	for _, ev := range edge.Evidence {
		gotEvidence[ev.Source] = true
	}
	if !gotEvidence[EvidenceProtoImport] || !gotEvidence[EvidenceGoImport] {
		t.Fatalf("evidence = %v, want proto and go import", edge.Evidence)
	}
}

func TestBuilderBuildAttributesFilteredImportsWithRepoScenarioCatalog(t *testing.T) {
	repoRoot := t.TempDir()
	writeScenarioManifest(t, repoRoot, "alpha")
	writeScenarioManifest(t, repoRoot, "bravo")

	builder := NewBuilder(
		fakeProtoClient{resp: &ProtoSurfaceResponse{Results: []ProtoSurfaceResult{
			{Scenario: "alpha", Surface: ProtoSurface{Scenario: "alpha"}},
		}}},
		fakeImportClient{resp: &ImportFactsResponse{Results: []ImportFactsResult{
			{
				Scenario: "alpha",
				Facts: []ImportFact{
					{
						ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/bravo/v1/bravo",
						Path:       "scenarios/alpha/api/internal/bravo/client.go",
						Analyzer:   "go-code-graph",
					},
				},
			},
		}}},
	)

	graph, err := builder.Build(context.Background(), BuildRequest{
		Scenarios: []string{"alpha"},
		RepoRoot:  repoRoot,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(graph.Errors) != 0 {
		t.Fatalf("errors = %v, want none", graph.Errors)
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("edges = %d, want 1: %#v", len(graph.Edges), graph.Edges)
	}
	edge := graph.Edges[0]
	if edge.FromScenario != "alpha" || edge.ToScenario != "bravo" {
		t.Fatalf("edge = %s -> %s, want alpha -> bravo", edge.FromScenario, edge.ToScenario)
	}
}

func TestBuilderBuildPreservesPerScenarioErrors(t *testing.T) {
	builder := NewBuilder(
		fakeProtoClient{resp: &ProtoSurfaceResponse{Results: []ProtoSurfaceResult{
			{Scenario: "alpha", Error: "descriptor failed"},
		}}},
		fakeImportClient{resp: &ImportFactsResponse{Results: []ImportFactsResult{
			{Scenario: "alpha", Error: "analyzer failed"},
		}}},
	)

	graph, err := builder.Build(context.Background(), BuildRequest{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(graph.Errors) != 2 {
		t.Fatalf("errors = %v, want 2", graph.Errors)
	}
}

type fakeProtoClient struct {
	resp *ProtoSurfaceResponse
	err  error
}

func (f fakeProtoClient) DescribeScenariosProtos(context.Context, ProtoSurfaceRequest) (*ProtoSurfaceResponse, error) {
	return f.resp, f.err
}

type fakeImportClient struct {
	resp *ImportFactsResponse
	err  error
}

func (f fakeImportClient) DescribeFleetImports(context.Context, ImportFactsRequest) (*ImportFactsResponse, error) {
	return f.resp, f.err
}

func writeScenarioManifest(t *testing.T, repoRoot, scenario string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir scenario manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(`{"service":{"name":"`+scenario+`"}}`), 0o644); err != nil {
		t.Fatalf("write service manifest: %v", err)
	}
}
