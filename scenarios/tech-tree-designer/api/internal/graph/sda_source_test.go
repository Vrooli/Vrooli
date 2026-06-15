package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sdagraphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type fakeSDAClient struct {
	req  SDAInterfaceGraphRequest
	resp *SDAInterfaceGraphResponse
	err  error
}

func (f *fakeSDAClient) DescribeInterfaceGraph(_ context.Context, req SDAInterfaceGraphRequest) (*SDAInterfaceGraphResponse, error) {
	f.req = req
	return f.resp, f.err
}

func TestSDASourceGraphMapsInterfaceGraph(t *testing.T) {
	client := &fakeSDAClient{resp: &SDAInterfaceGraphResponse{Graph: &sdagraphv1.InterfaceGraph{
		Nodes: []*sdagraphv1.GraphNode{
			{Scenario: "connect-app"},
			{Scenario: "proto-health"},
		},
		Edges: []*sdagraphv1.GraphEdge{
			{
				FromScenario:   "connect-app",
				ToScenario:     "proto-health",
				TransportWorld: "connect",
				Stability:      []string{"stable"},
				Evidence: []*sdagraphv1.GraphEvidence{
					{
						Source:   sdagraphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT,
						FromFile: "connect-app/v1/api/api.proto",
						ToFile:   "proto-health/v1/shared/surface.proto",
					},
					{
						Source:     sdagraphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT,
						ImportPath: "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared",
						Path:       "scenarios/connect-app/api/internal/protohealth/client.go",
						Analyzer:   "go-code-graph",
					},
				},
			},
		},
		Errors: []*sdagraphv1.GraphError{
			{Source: "code-facts", Scenario: "broken-app", Message: "analyzer unavailable"},
		},
	}}}

	graph, err := NewSDASource(client).Graph(context.Background(), SourceRequest{
		ScenarioFilter:  []string{"connect-app"},
		Limit:           25,
		StabilityFilter: "stable",
		MaxScenarioHops: 2,
	})

	require.NoError(t, err)
	require.Equal(t, SDAInterfaceGraphRequest{
		Scenarios:       []string{"connect-app"},
		Limit:           25,
		StabilityFilter: "stable",
		MaxScenarioHops: 2,
	}, client.req)
	require.Equal(t, []string{"connect-app", "proto-health"}, nodeScenarios(graph.Nodes))

	connectNode := findNode(graph.Nodes, "connect-app")
	require.NotNil(t, connectNode)
	require.Equal(t, graphv1.NodeKind_NODE_KIND_LIVE, connectNode.GetKind())
	require.Equal(t, "Connect App", connectNode.GetDisplayName())

	require.Len(t, graph.Edges, 1)
	edge := graph.Edges[0]
	require.Equal(t, "connect-app", edge.GetFromScenario())
	require.Equal(t, "proto-health", edge.GetToScenario())
	require.Equal(t, "connect", edge.GetTransportWorld())
	require.Equal(t, []string{"stable"}, edge.GetStability())
	require.Len(t, edge.GetEvidence(), 2)
	require.Equal(t, graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT, edge.GetEvidence()[0].GetSource())
	require.Equal(t, graphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT, edge.GetEvidence()[1].GetSource())
	require.Equal(t, "go-code-graph", edge.GetEvidence()[1].GetAnalyzer())

	require.Len(t, graph.Errors, 1)
	require.Equal(t, "code-facts", graph.Errors[0].GetSource())
	require.Equal(t, "broken-app", graph.Errors[0].GetScenario())
	require.Equal(t, "analyzer unavailable", graph.Errors[0].GetMessage())
}

func TestSDASourceGraphReturnsClientErrors(t *testing.T) {
	client := &fakeSDAClient{err: errors.New("offline")}

	graph, err := NewSDASource(client).Graph(context.Background(), SourceRequest{})

	require.Nil(t, graph)
	require.ErrorContains(t, err, "describe SDA interface graph")
	require.ErrorContains(t, err, "offline")
}

func TestSDASourceGraphRequiresClient(t *testing.T) {
	graph, err := NewSDASource(nil).Graph(context.Background(), SourceRequest{})

	require.Nil(t, graph)
	require.ErrorContains(t, err, "not configured")
}

func nodeScenarios(nodes []*graphv1.TechNode) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.GetScenario())
	}
	return out
}

func findNode(nodes []*graphv1.TechNode, scenario string) *graphv1.TechNode {
	for _, node := range nodes {
		if node.GetScenario() == scenario {
			return node
		}
	}
	return nil
}
