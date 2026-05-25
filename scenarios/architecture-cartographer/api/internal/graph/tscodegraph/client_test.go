package tscodegraph_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/tscodegraph"

	"github.com/vrooli/api-core/discovery"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// passthroughProject is a ProjectPathFn stub that treats the scenario
// name as the project path (always found). It keeps the Connect-hop
// tests focused on transport behavior; the real fs probing lives in the
// scenariopath package's own tests.
func passthroughProject(scenario string) (string, bool, error) {
	return scenario, true, nil
}

// newClient builds a tscodegraph client pointed at baseURL via a static
// discovery resolver, with the passthrough project resolver.
func newClient(baseURL string) *tscodegraph.Client {
	return tscodegraph.New(tscodegraph.Config{
		URLResolver: discovery.NewStaticResolver(baseURL),
		ProjectPath: passthroughProject,
	})
}

// fakeService is a programmable TypeScriptCodeGraphServiceHandler used
// by the in-process Connect server tests.
type fakeService struct {
	graph_v1connect.UnimplementedTypeScriptCodeGraphServiceHandler

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
	path, handler := graph_v1connect.NewTypeScriptCodeGraphServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestClient_NameAndLanguages(t *testing.T) {
	c := newClient("http://localhost:0")
	if got := c.Name(); got != "typescript" {
		t.Fatalf("Name=%q want %q", got, "typescript")
	}
	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != graph.LanguageTypeScript {
		t.Fatalf("SupportedLanguages=%v", langs)
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
							Id:   "file:src/a.ts",
							Kind: commonv1.NodeKind_NODE_KIND_FILE,
							Name: "a.ts",
							Path: "src/a.ts",
						},
						{
							Id:   "module:src/a",
							Kind: commonv1.NodeKind_NODE_KIND_MODULE,
							Name: "src/a",
							Path: "src/a",
						},
						{
							Id:   "ts_component:Foo",
							Kind: commonv1.NodeKind(201), // TS_NODE_KIND_COMPONENT, in the 200..299 reserved range
							Name: "Foo",
							Path: "src/a.ts",
						},
					},
					Edges: []*commonv1.CodeGraphEdge{
						{
							Id:         "edge:1",
							Kind:       commonv1.EdgeKind_EDGE_KIND_IMPORT,
							FromNodeId: "module:src/a",
							ToNodeId:   "module:src/b",
						},
					},
				},
				ExtractionMs: 42,
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
	if raw.Files[0].Path != "src/a.ts" {
		t.Fatalf("Files[0].Path=%q", raw.Files[0].Path)
	}
	if got, want := len(raw.Packages), 1; got != want {
		t.Fatalf("Packages len=%d want %d", got, want)
	}
	if raw.Packages[0].ImportPath != "src/a" {
		t.Fatalf("Packages[0].ImportPath=%q", raw.Packages[0].ImportPath)
	}
	if got, want := len(raw.Symbols), 1; got != want {
		t.Fatalf("Symbols len=%d want %d", got, want)
	}
	if raw.Symbols[0].Name != "Foo" {
		t.Fatalf("Symbols[0].Name=%q", raw.Symbols[0].Name)
	}
	if got, want := len(raw.Imports), 1; got != want {
		t.Fatalf("Imports len=%d want %d", got, want)
	}
	if raw.Imports[0].From != "module:src/a" || raw.Imports[0].ToPackageID != "module:src/b" {
		t.Fatalf("Imports[0]=%+v", raw.Imports[0])
	}
	if raw.ExtractionMS != 42 {
		t.Fatalf("ExtractionMS=%d want 42", raw.ExtractionMS)
	}
	if len(raw.Languages) != 1 || raw.Languages[0] != graph.LanguageTypeScript {
		t.Fatalf("Languages=%v", raw.Languages)
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
			if ie.Scenario != tscodegraph.ScenarioName {
				t.Fatalf("Scenario=%q", ie.Scenario)
			}
			if ie.Cause == nil {
				t.Fatal("Cause is nil; want underlying connect error")
			}
		})
	}
}

func TestClient_Extract_ContextCancellation(t *testing.T) {
	// The handler blocks until its context is cancelled; cancelling on
	// the client side must propagate through Connect rather than hang.
	// Connect surfaces cancellation as CodeCanceled which falls to the
	// default "internal" bucket. The point of the test is to confirm
	// cancellation produces a typed IntegrationError and does not deadlock.
	svc := &fakeService{
		extractFn: func(ctx context.Context, _ *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
			<-ctx.Done()
			return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
		},
	}
	c := newClient(startServer(t, svc))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := c.Extract(ctx, "demo")
	if err == nil {
		t.Fatal("want error from cancelled context")
	}
	var ie graph.IntegrationError
	if !errors.As(err, &ie) {
		t.Fatalf("want IntegrationError, got %T: %v", err, err)
	}
	if ie.Scenario != tscodegraph.ScenarioName {
		t.Fatalf("Scenario=%q", ie.Scenario)
	}
}

// failingResolver returns a fixed discovery error so tests can exercise
// the scenario_unreachable classification without a live producer.
type failingResolver struct{ err error }

func (f failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", f.err
}

func TestClient_Extract_NoTSProject_EmptyGraph(t *testing.T) {
	c := tscodegraph.New(tscodegraph.Config{
		URLResolver: discovery.NewStaticResolver("http://localhost:0"),
		ProjectPath: func(string) (string, bool, error) { return "", false, nil },
	})
	raw, err := c.Extract(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if n := len(raw.Files) + len(raw.Packages) + len(raw.Symbols) + len(raw.Imports); n != 0 {
		t.Fatalf("want empty graph for scenario with no TS project, got %d elements", n)
	}
}

func TestClient_Extract_ProjectResolveError_Internal(t *testing.T) {
	c := tscodegraph.New(tscodegraph.Config{
		URLResolver: discovery.NewStaticResolver("http://localhost:0"),
		ProjectPath: func(string) (string, bool, error) { return "", false, errors.New("boom") },
	})
	_, err := c.Extract(context.Background(), "demo")
	var ie graph.IntegrationError
	if !errors.As(err, &ie) || ie.Kind != "internal" {
		t.Fatalf("want internal IntegrationError, got %v", err)
	}
}

func TestClient_Extract_DiscoveryUnreachable(t *testing.T) {
	c := tscodegraph.New(tscodegraph.Config{
		URLResolver: failingResolver{err: &discovery.Error{Kind: discovery.ErrScenarioNotRunning, Scenario: tscodegraph.ScenarioName}},
		ProjectPath: passthroughProject,
	})
	_, err := c.Extract(context.Background(), "demo")
	var ie graph.IntegrationError
	if !errors.As(err, &ie) || ie.Kind != "scenario_unreachable" {
		t.Fatalf("want scenario_unreachable IntegrationError, got %v", err)
	}
}
