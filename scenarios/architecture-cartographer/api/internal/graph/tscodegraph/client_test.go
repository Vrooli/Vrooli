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
			if got := req.Msg.GetProjectPath(); got != "demo" {
				t.Fatalf("ProjectPath=%q want demo", got)
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

// TestClient_Extract_PackageNodeDiscrimination is the regression guard
// for the import-cluster DoS: the typescript-code-graph producer rides
// symbol-level nodes (calls, references, components, …) under
// NODE_KIND_PACKAGE with the real kind in attributes["kind"]. Only nodes
// whose kind attribute is empty or "package" are true packages. Without
// the gate, a single UI extraction injects thousands of fake packages
// into the import-cluster graph and explodes its O(N²) modularity pass.
func TestClient_Extract_PackageNodeDiscrimination(t *testing.T) {
	svc := &fakeService{
		extractFn: func(_ context.Context, _ *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
			return connect.NewResponse(&graphv1.ExtractResponse{
				Graph: &commonv1.CodeGraph{
					Nodes: []*commonv1.CodeGraphNode{
						// A real module → package.
						{Id: "ts_module:src/a", Kind: commonv1.NodeKind_NODE_KIND_MODULE, Name: "src/a"},
						// A real package node (empty kind) → package.
						{
							Id:         "pkg:src",
							Kind:       commonv1.NodeKind_NODE_KIND_PACKAGE,
							Name:       "src",
							Attributes: map[string]string{"kind": "package"},
						},
						// A call site ridden under NODE_KIND_PACKAGE → symbol, NOT package.
						{
							Id:         "ts_call:src/a.ts:19:1:describe",
							Kind:       commonv1.NodeKind_NODE_KIND_PACKAGE,
							Name:       "describe",
							Path:       "src/a.ts",
							Attributes: map[string]string{"kind": "TS_NODE_KIND_CALL", "file_id": "file:src/a.ts"},
						},
						// A reference ridden under NODE_KIND_PACKAGE → symbol.
						{
							Id:         "ts_reference:src/a.ts:20:5:foo",
							Kind:       commonv1.NodeKind_NODE_KIND_PACKAGE,
							Name:       "foo",
							Attributes: map[string]string{"kind": "TS_NODE_KIND_REFERENCE", "exported": "true"},
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
	if got, want := len(raw.Packages), 2; got != want {
		t.Fatalf("Packages len=%d want %d (only the module and the true package): %+v", got, want, raw.Packages)
	}
	if got, want := len(raw.Symbols), 2; got != want {
		t.Fatalf("Symbols len=%d want %d (the call + the reference): %+v", got, want, raw.Symbols)
	}
	for _, p := range raw.Packages {
		if p.ID == "ts_call:src/a.ts:19:1:describe" || p.ID == "ts_reference:src/a.ts:20:5:foo" {
			t.Fatalf("symbol-kind node leaked into Packages: %q", p.ID)
		}
	}
	// The call-site symbol must preserve its TS kind and file linkage.
	var foundCall bool
	for _, s := range raw.Symbols {
		if s.ID == "ts_call:src/a.ts:19:1:describe" {
			foundCall = true
			if s.Kind != "TS_NODE_KIND_CALL" {
				t.Fatalf("call symbol Kind=%q want TS_NODE_KIND_CALL", s.Kind)
			}
			if s.FileID != "file:src/a.ts" {
				t.Fatalf("call symbol FileID=%q want file:src/a.ts", s.FileID)
			}
		}
	}
	if !foundCall {
		t.Fatal("call-site node not found among symbols")
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
