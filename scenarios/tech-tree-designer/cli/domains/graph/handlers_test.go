package graph

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "tech-tree-designer/cli/internal/testutil"
)

type graphService struct {
	describeReq  *graphv1.DescribeTechTreeRequest
	neighborsReq *graphv1.GetNeighborhoodRequest
	exportReq    *graphv1.ExportTechTreeRequest
	pathReq      *graphv1.FindPathRequest
	ancestorsReq *graphv1.ListAncestorsRequest
}

func (s *graphService) DescribeTechTree(_ context.Context, req *connect.Request[graphv1.DescribeTechTreeRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	s.describeReq = req.Msg
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: testGraph()}), nil
}

func (s *graphService) GetNeighborhood(_ context.Context, req *connect.Request[graphv1.GetNeighborhoodRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	s.neighborsReq = req.Msg
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: testGraph()}), nil
}

func (s *graphService) FindPath(_ context.Context, req *connect.Request[graphv1.FindPathRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	s.pathReq = req.Msg
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: testGraph()}), nil
}

func (s *graphService) ListAncestors(_ context.Context, req *connect.Request[graphv1.ListAncestorsRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	s.ancestorsReq = req.Msg
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: testGraph()}), nil
}

func (s *graphService) ExportTechTree(_ context.Context, req *connect.Request[graphv1.ExportTechTreeRequest]) (*connect.Response[graphv1.ExportTechTreeResponse], error) {
	s.exportReq = req.Msg
	return connect.NewResponse(&graphv1.ExportTechTreeResponse{
		Format:    req.Msg.GetFormat(),
		Content:   "digraph tech_tree {}\n",
		MediaType: "text/vnd.graphviz",
	}), nil
}

func connectAPI(t *testing.T, svc *graphService) http.Handler {
	t.Helper()
	path, handler := graphconnect.NewGraphServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestDescribeCallsConnectAndRendersGraph(t *testing.T) {
	svc := &graphService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "scenarios"}, {Name: "stability"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"scenarios": "ui, api", "stability": "stable"},
	})

	require.NoError(t, h.describe(ctx))
	require.Equal(t, []string{"ui", "api"}, svc.describeReq.GetScenarioFilter())
	require.Equal(t, "stable", svc.describeReq.GetStabilityFilter())
	require.Contains(t, out.String(), "Scenario interface graph: 2 node(s), 1 edge(s).")
	require.Contains(t, out.String(), "ui -> api")
}

func TestDescribeJSONIsProtoWireShape(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &graphService{}))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "scenarios"}, {Name: "stability"}},
	}, cliapptest.TestRunContextOptions{JSON: true})

	require.NoError(t, h.describe(ctx))
	var got graphv1.DescribeTechTreeResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.GetGraph().GetNodes(), 2)
}

func TestNeighborsParsesDepth(t *testing.T) {
	svc := &graphService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "depth"}, {Name: "scenarios"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "ui"},
		Flags:       map[string]string{"depth": "2"},
	})

	require.NoError(t, h.neighbors(ctx))
	require.Equal(t, "ui", svc.neighborsReq.GetScenario())
	require.EqualValues(t, 2, svc.neighborsReq.GetDepth())
}

func TestExportPrintsContentForHumanOutput(t *testing.T) {
	svc := &graphService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "format", Default: "text"}, {Name: "scenarios"}, {Name: "stability"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"format": "dot"},
	})

	require.NoError(t, h.export(ctx))
	require.Equal(t, graphv1.ExportFormat_EXPORT_FORMAT_DOT, svc.exportReq.GetFormat())
	require.Equal(t, "digraph tech_tree {}\n", out.String())
}

func TestPathPassesEndpointsAndRendersGraph(t *testing.T) {
	svc := &graphService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "from", Required: true}, {Name: "to", Required: true}},
		Flags:       []cliapp.Flag{{Name: "scenarios"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"from": "ui", "to": "api"},
		Flags:       map[string]string{"scenarios": "ui, api"},
	})

	require.NoError(t, h.path(ctx))
	require.Equal(t, "ui", svc.pathReq.GetFromScenario())
	require.Equal(t, "api", svc.pathReq.GetToScenario())
	require.Equal(t, []string{"ui", "api"}, svc.pathReq.GetScenarioFilter())
	require.Contains(t, out.String(), "Path from ui to api: 2 node(s), 1 edge(s).")
}

func TestAncestorsPassesScenarioAndRendersGraph(t *testing.T) {
	svc := &graphService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "scenarios"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "api"},
	})

	require.NoError(t, h.ancestors(ctx))
	require.Equal(t, "api", svc.ancestorsReq.GetScenario())
	require.Contains(t, out.String(), "Ancestors for api: 2 node(s), 1 edge(s).")
}

func testGraph() *graphv1.TechTreeGraph {
	return &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{
			{Scenario: "api"},
			{Scenario: "ui"},
		},
		Edges: []*graphv1.TechEdge{{FromScenario: "ui", ToScenario: "api"}},
	}
}
