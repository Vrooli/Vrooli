package gocodegraph_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/gocodegraph"

	"github.com/vrooli/api-core/discovery"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// passthroughProject treats the scenario name as the project path
// (always found), keeping the Connect-hop tests focused on transport.
func passthroughProject(scenario string) (string, bool, error) {
	return scenario, true, nil
}

// newClient builds a gocodegraph client pointed at baseURL via a static
// discovery resolver, with the passthrough project resolver.
func newClient(baseURL string) *gocodegraph.Client {
	return gocodegraph.New(gocodegraph.Config{
		URLResolver: discovery.NewStaticResolver(baseURL),
		ProjectPath: passthroughProject,
	})
}

// failingResolver returns a fixed discovery error.
type failingResolver struct{ err error }

func (f failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", f.err
}

// fakeService is a programmable GoCodeGraphServiceHandler used by the
// in-process Connect server tests.
type fakeService struct {
	graph_v1connect.UnimplementedGoCodeGraphServiceHandler

	extractFn func(ctx context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error)
}

func (f *fakeService) Extract(ctx context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
	if f.extractFn != nil {
		return f.extractFn(ctx, req)
	}
	return connect.NewResponse(&graphv1.ExtractResponse{}), nil
}

// startServer spins up an httptest server mounting the fakeService on
// the generated Connect handler path. Returns the base URL; cleanup
// is registered via t.Cleanup.
func startServer(t *testing.T, svc *fakeService) string {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := graph_v1connect.NewGoCodeGraphServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestClient_NameAndLanguages(t *testing.T) {
	c := newClient("http://localhost:0")
	if got := c.Name(); got != "go" {
		t.Fatalf("Name=%q want %q", got, "go")
	}
	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != graph.LanguageGo {
		t.Fatalf("SupportedLanguages=%v", langs)
	}
}

func TestClient_DiscoveryUnreachableReturnsScenarioUnreachable(t *testing.T) {
	c := gocodegraph.New(gocodegraph.Config{
		URLResolver: failingResolver{err: &discovery.Error{Kind: discovery.ErrScenarioNotRunning, Scenario: gocodegraph.ScenarioName}},
		ProjectPath: passthroughProject,
	})
	_, err := c.Extract(context.Background(), "demo")
	var ie graph.IntegrationError
	if !errors.As(err, &ie) {
		t.Fatalf("want IntegrationError, got %v", err)
	}
	if ie.Kind != "scenario_unreachable" {
		t.Fatalf("Kind=%q want scenario_unreachable", ie.Kind)
	}
	if ie.Scenario != gocodegraph.ScenarioName {
		t.Fatalf("Scenario=%q", ie.Scenario)
	}
}

func TestClient_NoGoProject_EmptyGraph(t *testing.T) {
	c := gocodegraph.New(gocodegraph.Config{
		URLResolver: discovery.NewStaticResolver("http://localhost:0"),
		ProjectPath: func(string) (string, bool, error) { return "", false, nil },
	})
	raw, err := c.Extract(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if n := len(raw.Files) + len(raw.Packages) + len(raw.Symbols) + len(raw.Imports); n != 0 {
		t.Fatalf("want empty graph for scenario with no Go project, got %d elements", n)
	}
}

func TestClient_Extract_HappyPath(t *testing.T) {
	svc := &fakeService{
		extractFn: func(_ context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
			if got := req.Msg.GetScenarioPath(); got != "demo" {
				t.Fatalf("ScenarioPath=%q want demo", got)
			}
			return connect.NewResponse(&graphv1.ExtractResponse{
				Graph: &commonv1.CodeGraph{
					Nodes: []*commonv1.CodeGraphNode{
						{
							Id:   "package:example.com/foo",
							Kind: commonv1.NodeKind_NODE_KIND_PACKAGE,
							Name: "foo",
							Path: "example.com/foo",
							Attributes: map[string]string{
								"language":    "go",
								"import_path": "example.com/foo",
							},
						},
						{
							Id:   "file:foo/a.go",
							Kind: commonv1.NodeKind_NODE_KIND_FILE,
							Name: "a.go",
							Path: "foo/a.go",
							Attributes: map[string]string{
								"language":   "go",
								"package_id": "package:example.com/foo",
							},
						},
						{
							Id:   "go_func:package:example.com/foo:Bar",
							Kind: commonv1.NodeKind_NODE_KIND_PACKAGE,
							Name: "Bar",
							Path: "foo/a.go",
							Attributes: map[string]string{
								"language":   "go",
								"package_id": "package:example.com/foo",
								"file_id":    "file:foo/a.go",
								"kind":       "go_func",
								"exported":   "true",
							},
						},
					},
					Edges: []*commonv1.CodeGraphEdge{
						{
							Id:         "import:package:example.com/foo->package:example.com/bar",
							Kind:       commonv1.EdgeKind_EDGE_KIND_IMPORT,
							FromNodeId: "package:example.com/foo",
							ToNodeId:   "package:example.com/bar",
						},
					},
				},
				ExtractionMs: 17,
			}), nil
		},
	}
	c := newClient(startServer(t, svc))

	raw, err := c.Extract(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got, want := len(raw.Files), 1; got != want {
		t.Fatalf("Files len=%d want %d", got, want)
	}
	if raw.Files[0].Path != "foo/a.go" {
		t.Fatalf("Files[0].Path=%q", raw.Files[0].Path)
	}
	if raw.Files[0].PackageID != "package:example.com/foo" {
		t.Fatalf("Files[0].PackageID=%q", raw.Files[0].PackageID)
	}
	if raw.Files[0].Language != graph.LanguageGo {
		t.Fatalf("Files[0].Language=%q", raw.Files[0].Language)
	}
	if got, want := len(raw.Packages), 1; got != want {
		t.Fatalf("Packages len=%d want %d", got, want)
	}
	if raw.Packages[0].ImportPath != "example.com/foo" {
		t.Fatalf("Packages[0].ImportPath=%q", raw.Packages[0].ImportPath)
	}
	if got, want := len(raw.Symbols), 1; got != want {
		t.Fatalf("Symbols len=%d want %d", got, want)
	}
	sym := raw.Symbols[0]
	if sym.Name != "Bar" || sym.Kind != "go_func" || !sym.Exported {
		t.Fatalf("Symbols[0]=%+v", sym)
	}
	if sym.PackageID != "package:example.com/foo" || sym.FileID != "file:foo/a.go" {
		t.Fatalf("Symbols[0]=%+v", sym)
	}
	if got, want := len(raw.Imports), 1; got != want {
		t.Fatalf("Imports len=%d want %d", got, want)
	}
	imp := raw.Imports[0]
	if imp.From != "package:example.com/foo" || imp.ToPackageID != "package:example.com/bar" {
		t.Fatalf("Imports[0]=%+v", imp)
	}
	if raw.ExtractionMS != 17 {
		t.Fatalf("ExtractionMS=%d want 17", raw.ExtractionMS)
	}
	if len(raw.Languages) != 1 || raw.Languages[0] != graph.LanguageGo {
		t.Fatalf("Languages=%v", raw.Languages)
	}
}

func TestClient_Extract_ModulePromotesToPackage(t *testing.T) {
	// go-code-graph may emit NODE_KIND_MODULE for the top-level module
	// node; cartographer collapses modules onto PackageNode.
	svc := &fakeService{
		extractFn: func(_ context.Context, _ *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
			return connect.NewResponse(&graphv1.ExtractResponse{
				Graph: &commonv1.CodeGraph{
					Nodes: []*commonv1.CodeGraphNode{
						{
							Id:   "module:example.com",
							Kind: commonv1.NodeKind_NODE_KIND_MODULE,
							Name: "example.com",
							Path: ".",
						},
					},
				},
			}), nil
		},
	}
	c := newClient(startServer(t, svc))
	raw, err := c.Extract(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := len(raw.Packages); got != 1 {
		t.Fatalf("Packages len=%d want 1", got)
	}
	if raw.Packages[0].ImportPath != "example.com" {
		t.Fatalf("Packages[0].ImportPath=%q", raw.Packages[0].ImportPath)
	}
}

func TestClient_Extract_ErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		code     connect.Code
		wantKind string
	}{
		{"unavailable", connect.CodeUnavailable, "scenario_unreachable"},
		{"deadline", connect.CodeDeadlineExceeded, "scenario_timeout"},
		{"invalid_argument", connect.CodeInvalidArgument, "invalid_argument"},
		{"not_found", connect.CodeNotFound, "not_found"},
		{"unimplemented", connect.CodeUnimplemented, "unimplemented"},
		{"internal", connect.CodeInternal, "internal"},
		{"unknown_default", connect.CodeFailedPrecondition, "internal"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeService{
				extractFn: func(_ context.Context, _ *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
					return nil, connect.NewError(tc.code, errors.New("simulated"))
				},
			}
			c := newClient(startServer(t, svc))
			_, err := c.Extract(context.Background(), "demo")
			var ie graph.IntegrationError
			if !errors.As(err, &ie) {
				t.Fatalf("want IntegrationError, got %v", err)
			}
			if ie.Kind != tc.wantKind {
				t.Fatalf("Kind=%q want %q", ie.Kind, tc.wantKind)
			}
			if ie.Scenario != gocodegraph.ScenarioName {
				t.Fatalf("Scenario=%q", ie.Scenario)
			}
			if ie.Cause == nil {
				t.Fatal("Cause is nil; want underlying connect error")
			}
		})
	}
}

func TestClient_Extract_ContextCancellation(t *testing.T) {
	svc := &fakeService{
		extractFn: func(ctx context.Context, _ *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
			<-ctx.Done()
			return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
		},
	}
	c := newClient(startServer(t, svc))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Extract(ctx, "demo")
	if err == nil {
		t.Fatal("want error from cancelled context")
	}
	var ie graph.IntegrationError
	if !errors.As(err, &ie) {
		t.Fatalf("want IntegrationError, got %T: %v", err, err)
	}
	if ie.Scenario != gocodegraph.ScenarioName {
		t.Fatalf("Scenario=%q", ie.Scenario)
	}
}
