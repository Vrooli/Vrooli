package graph_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"golang.org/x/tools/go/packages"

	intgraph "go-code-graph/internal/graph"
	graphmocks "go-code-graph/internal/graph/mocks"
)

// writeScenario materializes a minimal Go module under t.TempDir() and
// returns its absolute path. Used by tests that exercise the Service's
// preflight (which checks go.mod presence) but inject a FakeLoader so
// the actual packages.Load isn't invoked.
func writeScenario(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

func TestServiceExtractInvalidInput(t *testing.T) {
	t.Parallel()
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	_, _, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: "   "})
	if err == nil {
		t.Fatal("want error for empty module_path, got nil")
	}
	var ex intgraph.ExtractError
	if !errors.As(err, &ex) || ex.Kind != intgraph.ExtractErrorInvalidInput {
		t.Fatalf("want ExtractErrorInvalidInput, got %#v", err)
	}
	if code := intgraph.ErrorToConnectCode(err); code != connect.CodeInvalidArgument {
		t.Fatalf("invalid input → InvalidArgument, got %v", code)
	}
}

func TestServiceExtractPathUnreadable(t *testing.T) {
	t.Parallel()
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	_, _, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: "/definitely/does/not/exist/xyz"})
	if err == nil {
		t.Fatal("want error for missing path, got nil")
	}
	var ex intgraph.ExtractError
	if !errors.As(err, &ex) || ex.Kind != intgraph.ExtractErrorPathUnreadable {
		t.Fatalf("want ExtractErrorPathUnreadable, got %#v", err)
	}
	if code := intgraph.ErrorToConnectCode(err); code != connect.CodeNotFound {
		t.Fatalf("unreadable → NotFound, got %v", code)
	}
}

func TestServiceExtractNoGoMod(t *testing.T) {
	t.Parallel()
	root := writeScenario(t, map[string]string{
		"README.md": "no go.mod here",
	})
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	_, _, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: root})
	var ex intgraph.ExtractError
	if !errors.As(err, &ex) || ex.Kind != intgraph.ExtractErrorNoGoMod {
		t.Fatalf("want ExtractErrorNoGoMod, got %#v", err)
	}
	if code := intgraph.ErrorToConnectCode(err); code != connect.CodeInvalidArgument {
		t.Fatalf("no go.mod → InvalidArgument, got %v", code)
	}
}

func TestServiceExtractMultipleGoMod(t *testing.T) {
	t.Parallel()
	root := writeScenario(t, map[string]string{
		"go.mod":        "module a\n\ngo 1.25\n",
		"nested/go.mod": "module nested\n\ngo 1.25\n",
	})
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	_, _, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: root})
	var ex intgraph.ExtractError
	if !errors.As(err, &ex) || ex.Kind != intgraph.ExtractErrorMultipleGoMod {
		t.Fatalf("want ExtractErrorMultipleGoMod, got %#v", err)
	}
}

func TestServiceExtractWorkspaceUnsupported(t *testing.T) {
	t.Parallel()
	root := writeScenario(t, map[string]string{
		"go.work": "go 1.25\nuse ./mod1\n",
		"go.mod":  "module a\n\ngo 1.25\n",
	})
	svc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	_, _, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: root})
	var ex intgraph.ExtractError
	if !errors.As(err, &ex) || ex.Kind != intgraph.ExtractErrorWorkspaceUnsupported {
		t.Fatalf("want ExtractErrorWorkspaceUnsupported, got %#v", err)
	}
	if code := intgraph.ErrorToConnectCode(err); code != connect.CodeUnimplemented {
		t.Fatalf("workspace → Unimplemented, got %v", code)
	}
}

func TestServiceExtractLoaderError(t *testing.T) {
	t.Parallel()
	root := writeScenario(t, map[string]string{
		"go.mod":  "module a\n\ngo 1.25\n",
		"main.go": "package a\n",
	})
	boom := errors.New("loader exploded")
	loader := &graphmocks.FakeLoader{
		LoadFunc: func(_ context.Context, _ string, _ intgraph.LoadOptions) ([]*packages.Package, error) {
			return nil, boom
		},
	}
	svc := intgraph.NewService(loader, intgraph.NewPathMutex())
	_, _, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: root})
	var ex intgraph.ExtractError
	if !errors.As(err, &ex) || ex.Kind != intgraph.ExtractErrorInternal {
		t.Fatalf("want ExtractErrorInternal, got %#v", err)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("loader cause not wrapped: %v", err)
	}
	if code := intgraph.ErrorToConnectCode(err); code != connect.CodeInternal {
		t.Fatalf("loader failure → Internal, got %v", code)
	}
}

func TestServiceExtractHappyPath(t *testing.T) {
	t.Parallel()
	root := writeScenario(t, map[string]string{
		"go.mod":  "module example.com/m\n\ngo 1.25\n",
		"main.go": "package m\n",
	})
	loader := &graphmocks.FakeLoader{
		LoadFunc: func(_ context.Context, _ string, _ intgraph.LoadOptions) ([]*packages.Package, error) {
			return []*packages.Package{{PkgPath: "example.com/m", Name: "m"}}, nil
		},
	}
	svc := intgraph.NewService(loader, intgraph.NewPathMutex())
	g, warnings, err := svc.Extract(context.Background(), intgraph.ExtractInput{ModulePath: root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %+v", warnings)
	}
	found := false
	for _, n := range g.Nodes {
		if n.ID == "package:example.com/m" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected package node, got %+v", g.Nodes)
	}
}

// TestErrorToConnectCodeNil asserts the nil-in/zero-out contract.
func TestErrorToConnectCodeNil(t *testing.T) {
	t.Parallel()
	if got := intgraph.ErrorToConnectCode(nil); got != connect.Code(0) {
		t.Fatalf("nil err → code(0), got %v", got)
	}
	if got := intgraph.ToConnectError(nil); got != nil {
		t.Fatalf("ToConnectError(nil) should be nil, got %v", got)
	}
}

// TestErrorToConnectCodeUnknownErr asserts non-ExtractError values
// fall through to Internal.
func TestErrorToConnectCodeUnknownErr(t *testing.T) {
	t.Parallel()
	if got := intgraph.ErrorToConnectCode(errors.New("random")); got != connect.CodeInternal {
		t.Fatalf("unknown err → Internal, got %v", got)
	}
}
