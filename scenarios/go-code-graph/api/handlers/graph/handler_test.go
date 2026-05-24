package graph

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"golang.org/x/tools/go/packages"

	intgraph "go-code-graph/internal/graph"
	graphmocks "go-code-graph/internal/graph/mocks"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// newTestClient spins up an httptest server with the Connect handler
// mounted at the canonical path and returns a paired Connect client.
// The generated NewGoCodeGraphServiceHandler returns an http.Handler
// that already path-multiplexes the procedure names, so we mount it at
// the server root directly — the Connect client appends the procedure
// path itself.
func newTestClient(t *testing.T, svc *intgraph.Service) graph_v1connect.GoCodeGraphServiceClient {
	t.Helper()
	_, h := graph_v1connect.NewGoCodeGraphServiceHandler(NewConnectHandler(Deps{
		GraphService: svc,
		Logger:       log.New(io.Discard, "", 0),
	}))
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return graph_v1connect.NewGoCodeGraphServiceClient(server.Client(), server.URL)
}

func writeMinimalScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/m\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package m\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	return root
}

func TestHandlerExtractHappyPath(t *testing.T) {
	t.Parallel()
	root := writeMinimalScenario(t)
	loader := &graphmocks.FakeLoader{
		LoadFunc: func(_ context.Context, _ string, _ intgraph.LoadOptions) ([]*packages.Package, error) {
			return []*packages.Package{{PkgPath: "example.com/m", Name: "m"}}, nil
		},
	}
	svc := intgraph.NewService(loader, intgraph.NewPathMutex())
	client := newTestClient(t, svc)

	resp, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: root,
	}))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if resp.Msg.GetGraph() == nil {
		t.Fatalf("expected non-nil graph")
	}
	if resp.Msg.GetGraphHash() == "" {
		t.Errorf("expected non-empty graph hash")
	}
	if resp.Msg.GetExtractionMs() < 0 {
		t.Errorf("extraction_ms must be ≥ 0, got %d", resp.Msg.GetExtractionMs())
	}
}

func TestHandlerExtractInvalidArgument(t *testing.T) {
	t.Parallel()
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	client := newTestClient(t, svc)

	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: "",
	}))
	if err == nil {
		t.Fatal("expected error for empty scenario_path")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestHandlerExtractNotFound(t *testing.T) {
	t.Parallel()
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	client := newTestClient(t, svc)

	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: "/no/such/path/xyz",
	}))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound, got %v", connect.CodeOf(err))
	}
}

func TestHandlerExtractLoaderInternal(t *testing.T) {
	t.Parallel()
	root := writeMinimalScenario(t)
	boom := errors.New("loader boom")
	loader := &graphmocks.FakeLoader{
		LoadFunc: func(_ context.Context, _ string, _ intgraph.LoadOptions) ([]*packages.Package, error) {
			return nil, boom
		},
	}
	svc := intgraph.NewService(loader, intgraph.NewPathMutex())
	client := newTestClient(t, svc)

	_, err := client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: root,
	}))
	if err == nil {
		t.Fatal("expected error from loader failure")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("want Internal, got %v", connect.CodeOf(err))
	}
}

// (Rewrite RPC handler tests live in rewrite_handler_test.go.)
