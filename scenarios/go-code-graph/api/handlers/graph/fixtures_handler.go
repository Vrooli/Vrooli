package graph

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	intgraph "go-code-graph/internal/graph"

	"github.com/vrooli/api-core/graphfixtures"
	"github.com/vrooli/api-core/textdiff"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
)

// defaultFixturesDir is used when Deps.FixturesDir is empty. Resolved relative
// to the server's working directory, which the lifecycle sets to the scenario
// root.
const defaultFixturesDir = "bas/fixtures"

const (
	goModFile = "go.mod"
)

func (h *connectHandler) fixturesDir() string {
	if strings.TrimSpace(h.deps.FixturesDir) != "" {
		return h.deps.FixturesDir
	}
	if env := strings.TrimSpace(os.Getenv("GO_CODE_GRAPH_FIXTURES_DIR")); env != "" {
		return env
	}
	// The lifecycle launches the server from api/ (not the module root), so
	// resolve fixtures against the scenario directory it injects. Falls back to
	// a cwd-relative path for ad-hoc local runs / tests.
	for _, envVar := range []string{"VROOLI_SCENARIO_DIR", "SCENARIO_PATH"} {
		if root := strings.TrimSpace(os.Getenv(envVar)); root != "" {
			return filepath.Join(root, defaultFixturesDir)
		}
	}
	return defaultFixturesDir
}

// ListFixtures enumerates fixture directories (those containing a go.mod) under
// the fixtures root. has_expected reports whether an expected-graph.json
// baseline exists. A missing fixtures root is not an error — it yields an empty
// list so the UI can show its empty state.
func (h *connectHandler) ListFixtures(_ context.Context, _ *connect.Request[graphv1.ListFixturesRequest]) (*connect.Response[graphv1.ListFixturesResponse], error) {
	items, err := graphfixtures.List(h.fixturesDir(), goModFile)
	if err != nil {
		h.deps.Logger.Printf("graph.ListFixtures(%q): %v", h.fixturesDir(), err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read fixtures dir: %w", err))
	}
	fixtures := make([]*graphv1.FixtureInfo, 0, len(items))
	for _, item := range items {
		fixtures = append(fixtures, &graphv1.FixtureInfo{
			Name: item.Name, Path: item.Path, HasExpected: item.HasExpected,
		})
	}
	return connect.NewResponse(&graphv1.ListFixturesResponse{Fixtures: fixtures}), nil
}

// ValidateFixture re-runs Extract against a named fixture and byte-compares the
// canonical JSON against the fixture's expected-graph.json. This promotes the
// determinism integration test to a first-class RPC so the UI can run it
// without reading repo files in the browser.
func (h *connectHandler) ValidateFixture(ctx context.Context, req *connect.Request[graphv1.ValidateFixtureRequest]) (*connect.Response[graphv1.ValidateFixtureResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("fixture name is required"))
	}
	// Reject path-escape attempts; the name must be a single directory segment.
	if name != filepath.Base(name) || strings.Contains(name, "..") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid fixture name %q", name))
	}

	dir := filepath.Join(h.fixturesDir(), name)
	if _, err := os.Stat(filepath.Join(dir, goModFile)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("fixture %q not found", name))
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("resolve fixture path: %w", err))
	}

	g, _, err := h.deps.GraphService.Extract(ctx, intgraph.ExtractInput{ModulePath: abs})
	if err != nil {
		connectErr := intgraph.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("graph.ValidateFixture(%q): extract: %v", name, err)
		}
		return nil, connectErr
	}

	actual := intgraph.CanonicalJSON(g)
	resp := &graphv1.ValidateFixtureResponse{
		ActualBytes: int64(len(actual)),
		GraphHash:   intgraph.GraphHash(g),
	}

	expected, err := os.ReadFile(filepath.Join(dir, graphfixtures.ExpectedGraphFile))
	if err != nil {
		if os.IsNotExist(err) {
			// No baseline to compare against; surface as a failed precondition
			// so the UI can prompt to bootstrap it rather than reporting a pass.
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("fixture %q has no %s baseline", name, graphfixtures.ExpectedGraphFile))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read expected graph: %w", err))
	}
	resp.ExpectedBytes = int64(len(expected))

	if bytes.Equal(actual, expected) {
		resp.Passed = true
		return connect.NewResponse(resp), nil
	}
	resp.Passed = false
	resp.Diff = textdiff.LineDiff(string(expected), string(actual))
	return connect.NewResponse(resp), nil
}
