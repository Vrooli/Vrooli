package graph

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// fakeService implements graph_v1connect.GoCodeGraphServiceHandler so the
// CLI tests can exercise the real Connect-RPC client transport against an
// httptest server. Only Extract is wired here; the rewrite handlers
// return Unimplemented (inherited from the embedded base) because this
// file is the graph-domain test surface.
type fakeService struct {
	graphconnect.UnimplementedGoCodeGraphServiceHandler

	mu          sync.Mutex
	extractResp *graphv1.ExtractResponse
	extractErr  error
	extractReqs []*graphv1.ExtractRequest
}

func (s *fakeService) Extract(_ context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
	s.mu.Lock()
	s.extractReqs = append(s.extractReqs, req.Msg)
	s.mu.Unlock()
	if s.extractErr != nil {
		return nil, s.extractErr
	}
	resp := s.extractResp
	if resp == nil {
		resp = &graphv1.ExtractResponse{}
	}
	return connect.NewResponse(resp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := graphconnect.NewGoCodeGraphServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleGraph() *commonv1.CodeGraph {
	return &commonv1.CodeGraph{
		Nodes: []*commonv1.CodeGraphNode{
			{Id: "package:foo", Kind: commonv1.NodeKind_NODE_KIND_PACKAGE, Name: "foo", Path: "foo"},
			{Id: "package:bar", Kind: commonv1.NodeKind_NODE_KIND_PACKAGE, Name: "bar", Path: "bar"},
			{Id: "file:foo/main.go", Kind: commonv1.NodeKind_NODE_KIND_FILE, Name: "main.go", Path: "foo/main.go"},
		},
		Edges: []*commonv1.CodeGraphEdge{},
	}
}

func TestExtract_RendersSummary(t *testing.T) {
	svc := &fakeService{extractResp: &graphv1.ExtractResponse{
		Graph:        sampleGraph(),
		ExtractionMs: 42,
		GraphHash:    "deadbeef",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "path", Required: true}},
		Flags:       []cliapp.Flag{{Name: "include-vendor"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"path": "/tmp/mod"},
	})

	require.NoError(t, h.extract(ctx))
	require.Len(t, svc.extractReqs, 1)
	require.Equal(t, "/tmp/mod", svc.extractReqs[0].GetModulePath())
	require.False(t, svc.extractReqs[0].GetIncludeVendor())

	body := out.String()
	require.Contains(t, body, "Extracted 3 node(s), 0 edge(s) in 42ms")
	require.Contains(t, body, "hash=deadbeef")
	require.Contains(t, body, "Top-level packages")
	require.Contains(t, body, "foo")
	require.Contains(t, body, "bar")
}

func TestExtract_PassesIncludeVendorFlag(t *testing.T) {
	svc := &fakeService{extractResp: &graphv1.ExtractResponse{Graph: &commonv1.CodeGraph{}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "path", Required: true}},
		Flags:       []cliapp.Flag{{Name: "include-vendor"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"path": "/tmp/mod"},
		BoolFlags:   map[string]bool{"include-vendor": true},
	})

	require.NoError(t, h.extract(ctx))
	require.True(t, svc.extractReqs[0].GetIncludeVendor())
}

func TestExtract_SurfacesConnectErrors(t *testing.T) {
	svc := &fakeService{extractErr: connect.NewError(connect.CodeInvalidArgument, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "path", Required: true}},
		Flags:       []cliapp.Flag{{Name: "include-vendor"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"path": "/tmp/mod"},
	})

	err := h.extract(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_argument")
}
