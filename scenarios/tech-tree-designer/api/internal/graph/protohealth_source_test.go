package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type fakeProtoSurfaceClient struct {
	req  ProtoSurfaceRequest
	resp *ProtoSurfaceResponse
	err  error
}

func (f *fakeProtoSurfaceClient) DescribeScenariosProtos(_ context.Context, req ProtoSurfaceRequest) (*ProtoSurfaceResponse, error) {
	f.req = req
	return f.resp, f.err
}

func TestProtoHealthSourceGraphMapsSurfacesToScenarioGraph(t *testing.T) {
	client := &fakeProtoSurfaceClient{resp: &ProtoSurfaceResponse{
		Results: []ProtoSurfaceResult{
			{
				Scenario: "connect-app",
				Surface: ProtoSurface{
					Scenario:       "connect-app",
					TransportWorld: "connect",
					Files: []ProtoFile{
						{Path: "connect-app/v1/api/api.proto", Stability: "stable"},
						{Path: "connect-app/v1/worker/worker.proto", Stability: "beta"},
					},
					CrossScenarioImports: []ProtoImport{
						{
							FromFile:     "connect-app/v1/api/api.proto",
							ToFile:       "proto-health/v1/shared/surface.proto",
							FromScenario: "connect-app",
							ToScenario:   "proto-health",
						},
					},
				},
			},
			{
				Scenario: "hand-rolled-app",
				Surface: ProtoSurface{
					Scenario:       "hand-rolled-app",
					TransportWorld: "hand_rolled",
					Files:          []ProtoFile{{Path: "hand-rolled-app/v1/api/api.proto", Stability: "experimental"}},
				},
			},
			{
				Scenario: "broken-app",
				Error:    "descriptor unavailable",
			},
		},
	}}

	graph, err := NewProtoHealthSource(client).Graph(context.Background(), SourceRequest{
		ScenarioFilter:  []string{"connect-app", "hand-rolled-app"},
		Limit:           25,
		StabilityFilter: "stable",
	})

	require.NoError(t, err)
	require.Equal(t, ProtoSurfaceRequest{
		Scenarios:       []string{"connect-app", "hand-rolled-app"},
		Limit:           25,
		StabilityFilter: "stable",
	}, client.req)
	require.Equal(t, []string{"broken-app", "connect-app", "hand-rolled-app", "proto-health"}, nodeScenarios(graph.Nodes))

	connectNode := findNode(graph.Nodes, "connect-app")
	require.NotNil(t, connectNode)
	require.Equal(t, graphv1.NodeKind_NODE_KIND_LIVE, connectNode.GetKind())
	require.Equal(t, "connect", connectNode.GetTransportWorld())
	require.Equal(t, []string{"beta", "stable"}, connectNode.GetStability())

	handRolledNode := findNode(graph.Nodes, "hand-rolled-app")
	require.NotNil(t, handRolledNode)
	require.Equal(t, "hand_rolled", handRolledNode.GetTransportWorld())
	require.Equal(t, []string{"experimental"}, handRolledNode.GetStability())

	require.Len(t, graph.Edges, 1)
	edge := graph.Edges[0]
	require.Equal(t, "connect-app", edge.GetFromScenario())
	require.Equal(t, "proto-health", edge.GetToScenario())
	require.Equal(t, "connect", edge.GetTransportWorld())
	require.Equal(t, []string{"stable"}, edge.GetStability())
	require.Len(t, edge.GetEvidence(), 1)
	require.Equal(t, graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT, edge.GetEvidence()[0].GetSource())
	require.Equal(t, "connect-app/v1/api/api.proto", edge.GetEvidence()[0].GetFromFile())
	require.Equal(t, "proto-health/v1/shared/surface.proto", edge.GetEvidence()[0].GetToFile())

	require.Len(t, graph.Errors, 1)
	require.Equal(t, SourceProtoHealth, graph.Errors[0].GetSource())
	require.Equal(t, "broken-app", graph.Errors[0].GetScenario())
	require.Equal(t, "descriptor unavailable", graph.Errors[0].GetMessage())
}

func TestProtoHealthSourceGraphReturnsClientErrors(t *testing.T) {
	client := &fakeProtoSurfaceClient{err: errors.New("offline")}

	graph, err := NewProtoHealthSource(client).Graph(context.Background(), SourceRequest{})

	require.Nil(t, graph)
	require.ErrorContains(t, err, "describe proto surfaces")
	require.ErrorContains(t, err, "offline")
}

func TestProtoHealthSourceGraphRequiresClient(t *testing.T) {
	graph, err := NewProtoHealthSource(nil).Graph(context.Background(), SourceRequest{})

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
