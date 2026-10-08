package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	graphdomain "tech-tree-designer/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph/graph_v1connect"
)

type fakeSource struct{}

func (fakeSource) Graph(context.Context, graphdomain.SourceRequest) (*graphv1.TechTreeGraph, error) {
	return &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{{Scenario: "api"}, {Scenario: "ui"}},
		Edges: []*graphv1.TechEdge{{FromScenario: "ui", ToScenario: "api"}},
	}, nil
}

func TestModuleRegistersGraphConnectHandler(t *testing.T) {
	mod := Module(graphdomain.NewService(fakeSource{}))
	require.Equal(t, "graph", mod.Name)
	require.Len(t, mod.Endpoints, 5)

	router := http.NewServeMux()
	path, handler := graphconnect.NewGraphServiceHandler(NewHandler(graphdomain.NewService(fakeSource{})))
	router.Handle(path, handler)

	client := graphconnect.NewGraphServiceClient(&http.Client{
		Transport: localRoundTripper{handler: router},
		Timeout:   5 * time.Second,
	}, "http://tech-tree-designer.test")
	resp, err := client.DescribeTechTree(context.Background(), connect.NewRequest(&graphv1.DescribeTechTreeRequest{}))

	require.NoError(t, err)
	require.Len(t, resp.Msg.GetGraph().GetNodes(), 2)
	require.Len(t, resp.Msg.GetGraph().GetEdges(), 1)
}

type localRoundTripper struct {
	handler http.Handler
}

func (rt localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rt.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}
