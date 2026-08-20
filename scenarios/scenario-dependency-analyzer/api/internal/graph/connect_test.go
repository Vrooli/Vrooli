package graph

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
)

func TestInterfaceGraphToProto(t *testing.T) {
	actual := interfaceGraphToProto(interfacegraph.Graph{
		Nodes: []interfacegraph.Node{
			{Scenario: "producer"},
			{Scenario: "consumer"},
		},
		Edges: []interfacegraph.Edge{
			{
				FromScenario:   "consumer",
				ToScenario:     "producer",
				TransportWorld: "go",
				Stability:      []string{"stable"},
				Evidence: []interfacegraph.Evidence{
					{
						Source:     interfacegraph.EvidenceProtoImport,
						ImportPath: "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/v1/graph/graph.proto",
						FromFile:   "consumer/api/main.go",
						ToFile:     "packages/proto/schemas/scenario-dependency-analyzer/v1/graph/graph.proto",
						Path:       "consumer -> producer",
						Analyzer:   "proto-health",
					},
				},
			},
		},
		Errors: []interfacegraph.Error{
			{Source: "code-facts", Scenario: "consumer", Message: "partial evidence"},
		},
	})

	if len(actual.GetNodes()) != 2 {
		t.Fatalf("nodes = %d, want 2", len(actual.GetNodes()))
	}
	if actual.GetNodes()[0].GetScenario() != "producer" {
		t.Fatalf("first node scenario = %q, want producer", actual.GetNodes()[0].GetScenario())
	}

	edges := actual.GetEdges()
	if len(edges) != 1 {
		t.Fatalf("edges = %d, want 1", len(edges))
	}
	edge := edges[0]
	if edge.GetFromScenario() != "consumer" || edge.GetToScenario() != "producer" {
		t.Fatalf("edge = %s -> %s, want consumer -> producer", edge.GetFromScenario(), edge.GetToScenario())
	}
	if got := edge.GetEvidence()[0].GetSource(); got != graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT {
		t.Fatalf("evidence source = %v, want proto import", got)
	}
	if got := len(actual.GetErrors()); got != 1 {
		t.Fatalf("errors = %d, want 1", got)
	}
}
