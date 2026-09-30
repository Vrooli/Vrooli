package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type fakeGraphSource struct {
	req   SourceRequest
	graph *graphv1.TechTreeGraph
	err   error
}

func (f *fakeGraphSource) Graph(_ context.Context, req SourceRequest) (*graphv1.TechTreeGraph, error) {
	f.req = req
	return f.graph, f.err
}

func TestServiceDescribePassesSourceRequest(t *testing.T) {
	source := &fakeGraphSource{graph: sampleGraph()}
	got, err := NewService(source).Describe(context.Background(), SourceRequest{
		ScenarioFilter:  []string{"api"},
		Limit:           10,
		StabilityFilter: "stable",
	})

	require.NoError(t, err)
	require.Equal(t, sampleGraph(), got)
	require.Equal(t, SourceRequest{
		ScenarioFilter:  []string{"api"},
		Limit:           10,
		StabilityFilter: "stable",
	}, source.req)
}

func TestServiceDescribeDegradesSourceErrorIntoGraphError(t *testing.T) {
	source := &fakeGraphSource{err: errors.New("proto-health offline")}
	got, err := NewService(source).Describe(context.Background(), SourceRequest{})

	require.NoError(t, err)
	require.Empty(t, got.Nodes)
	require.Empty(t, got.Edges)
	require.Len(t, got.Errors, 1)
	require.Contains(t, got.Errors[0].GetMessage(), "proto-health offline")
}

type fakePlannedSource struct {
	graph *graphv1.TechTreeGraph
	err   error
}

func (f fakePlannedSource) PlannedGraph(context.Context) (*graphv1.TechTreeGraph, error) {
	return f.graph, f.err
}

func TestServiceDescribeMergesPlannedGraph(t *testing.T) {
	source := &fakeGraphSource{graph: sampleGraph()}
	planned := fakePlannedSource{graph: &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{{
			Scenario:       "planned-demo",
			Kind:           graphv1.NodeKind_NODE_KIND_PLANNED,
			TransportWorld: "none",
			Stability:      []string{"experimental"},
		}},
		Edges: []*graphv1.TechEdge{{
			FromScenario:   "planned-demo",
			ToScenario:     "api",
			TransportWorld: "none",
			Stability:      []string{"experimental"},
			Evidence: []*graphv1.GraphEvidence{{
				Source:     graphv1.EvidenceSource_EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT,
				ImportPath: "api/v1/service.proto",
			}},
		}},
	}}

	got, err := NewServiceWithPlanned(source, planned).Describe(context.Background(), SourceRequest{})

	require.NoError(t, err)
	require.Contains(t, nodeScenarios(got.Nodes), "planned-demo")
	require.Contains(t, edgePairs(got.Edges), "planned-demo->api")
}

func TestServiceNeighborhoodIncludesIncomingAndOutgoingEdges(t *testing.T) {
	got, err := NewService(&fakeGraphSource{graph: sampleGraph()}).Neighborhood(context.Background(), "api", 1, nil)

	require.NoError(t, err)
	require.Equal(t, []string{"api", "auth", "ui"}, nodeScenarios(got.Nodes))
	require.Equal(t, []string{"api->auth", "ui->api"}, edgePairs(got.Edges))
}

func TestServicePathReturnsDirectedShortestPath(t *testing.T) {
	got, err := NewService(&fakeGraphSource{graph: sampleGraph()}).Path(context.Background(), "ui", "db", nil)

	require.NoError(t, err)
	require.Equal(t, []string{"api", "auth", "db", "ui"}, nodeScenarios(got.Nodes))
	require.Equal(t, []string{"api->auth", "auth->db", "ui->api"}, edgePairs(got.Edges))
}

func TestServiceAncestorsReturnsReachableDependencies(t *testing.T) {
	got, err := NewService(&fakeGraphSource{graph: sampleGraph()}).Ancestors(context.Background(), "api", nil)

	require.NoError(t, err)
	require.Equal(t, []string{"api", "auth", "db"}, nodeScenarios(got.Nodes))
	require.Equal(t, []string{"api->auth", "auth->db"}, edgePairs(got.Edges))
}

func TestServiceExportFormats(t *testing.T) {
	svc := NewService(&fakeGraphSource{graph: sampleGraph()})

	dot, err := svc.Export(context.Background(), SourceRequest{}, graphv1.ExportFormat_EXPORT_FORMAT_DOT)
	require.NoError(t, err)
	require.Equal(t, "text/vnd.graphviz", dot.GetMediaType())
	require.Contains(t, dot.GetContent(), `"api" -> "auth"`)

	text, err := svc.Export(context.Background(), SourceRequest{}, graphv1.ExportFormat_EXPORT_FORMAT_TEXT)
	require.NoError(t, err)
	require.Equal(t, "text/plain", text.GetMediaType())
	require.Contains(t, text.GetContent(), "Tech Tree Graph: 4 node(s), 3 edge(s)")

	js, err := svc.Export(context.Background(), SourceRequest{}, graphv1.ExportFormat_EXPORT_FORMAT_JSON)
	require.NoError(t, err)
	require.Equal(t, "application/json", js.GetMediaType())
	require.Contains(t, js.GetContent(), `"nodes"`)
}

func sampleGraph() *graphv1.TechTreeGraph {
	return &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{
			{Scenario: "api", DisplayName: "API", TransportWorld: "connect", Stability: []string{"stable"}},
			{Scenario: "auth", DisplayName: "Auth", TransportWorld: "connect", Stability: []string{"stable"}},
			{Scenario: "db", DisplayName: "DB", TransportWorld: "none", Stability: []string{"stable"}},
			{Scenario: "ui", DisplayName: "UI", TransportWorld: "connect", Stability: []string{"beta"}},
		},
		Edges: []*graphv1.TechEdge{
			{FromScenario: "api", ToScenario: "auth"},
			{FromScenario: "auth", ToScenario: "db"},
			{FromScenario: "ui", ToScenario: "api"},
		},
	}
}

func edgePairs(edges []*graphv1.TechEdge) []string {
	out := make([]string, 0, len(edges))
	for _, edge := range edges {
		out = append(out, edge.GetFromScenario()+"->"+edge.GetToScenario())
	}
	return out
}
