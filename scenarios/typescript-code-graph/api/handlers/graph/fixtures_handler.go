package graph

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"

	intgraph "typescript-code-graph/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
)

// defaultFixturesDir is used when Deps.FixturesDir is empty. Resolved relative
// to the server's working directory, which the lifecycle sets to the scenario
// root.
const defaultFixturesDir = "bas/fixtures"

const (
	expectedGraphFile = "expected-graph.json"
	tsConfigFile      = "tsconfig.json"
	// maxDiffLines caps the per-side diff payload so a wildly divergent graph
	// can't return a multi-megabyte response.
	maxDiffLines = 400
)

func (h *connectHandler) fixturesDir() string {
	if strings.TrimSpace(h.deps.FixturesDir) != "" {
		return h.deps.FixturesDir
	}
	if env := strings.TrimSpace(os.Getenv("TYPESCRIPT_CODE_GRAPH_FIXTURES_DIR")); env != "" {
		return env
	}
	// The lifecycle launches the server from api/ (not the scenario root), so
	// resolve fixtures against the scenario directory it injects. Falls back to
	// a cwd-relative path for ad-hoc local runs / tests.
	for _, envVar := range []string{"VROOLI_SCENARIO_DIR", "SCENARIO_PATH"} {
		if root := strings.TrimSpace(os.Getenv(envVar)); root != "" {
			return filepath.Join(root, defaultFixturesDir)
		}
	}
	return defaultFixturesDir
}

// ListFixtures enumerates fixture directories (those containing a tsconfig.json)
// under the fixtures root. has_expected reports whether an expected-graph.json
// baseline exists. A missing fixtures root is not an error — it yields an empty
// list so the UI can show its empty state.
func (h *connectHandler) ListFixtures(_ context.Context, _ *connect.Request[graphv1.ListFixturesRequest]) (*connect.Response[graphv1.ListFixturesResponse], error) {
	root := h.fixturesDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return connect.NewResponse(&graphv1.ListFixturesResponse{}), nil
		}
		h.deps.Logger.Printf("graph.ListFixtures(%q): %v", root, err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read fixtures dir: %w", err))
	}

	fixtures := make([]*graphv1.FixtureInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, tsConfigFile)); err != nil {
			continue // not a TS project fixture
		}
		_, expErr := os.Stat(filepath.Join(dir, expectedGraphFile))
		fixtures = append(fixtures, &graphv1.FixtureInfo{
			Name:        entry.Name(),
			Path:        filepath.ToSlash(dir),
			HasExpected: expErr == nil,
		})
	}
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].GetName() < fixtures[j].GetName() })

	return connect.NewResponse(&graphv1.ListFixturesResponse{Fixtures: fixtures}), nil
}

// ValidateFixture re-runs Extract against a named fixture (through the sidecar)
// and byte-compares the canonical JSON against the fixture's
// expected-graph.json. This promotes the determinism integration test to a
// first-class RPC so the UI can run it without reading repo files in the
// browser.
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
	if _, err := os.Stat(filepath.Join(dir, tsConfigFile)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("fixture %q not found", name))
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("resolve fixture path: %w", err))
	}

	out, err := h.deps.GraphService.Extract(ctx, intgraph.ExtractInput{ProjectPath: abs})
	if err != nil {
		connectErr := intgraph.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("graph.ValidateFixture(%q): extract: %v", name, err)
		}
		return nil, connectErr
	}

	actual, err := intgraph.CanonicalJSON(out.Graph)
	if err != nil {
		h.deps.Logger.Printf("graph.ValidateFixture(%q): canonicalize: %v", name, err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("canonicalize graph: %w", err))
	}
	resp := &graphv1.ValidateFixtureResponse{
		ActualBytes: int64(len(actual)),
		GraphHash:   out.GraphHash,
	}

	expected, err := os.ReadFile(filepath.Join(dir, expectedGraphFile))
	if err != nil {
		if os.IsNotExist(err) {
			// No baseline to compare against; surface as a failed precondition
			// so the UI can prompt to bootstrap it rather than reporting a pass.
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("fixture %q has no %s baseline", name, expectedGraphFile))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read expected graph: %w", err))
	}
	resp.ExpectedBytes = int64(len(expected))

	if bytes.Equal(actual, expected) {
		resp.Passed = true
		return connect.NewResponse(resp), nil
	}
	resp.Passed = false
	resp.Diff = lineDiff(string(expected), string(actual))
	return connect.NewResponse(resp), nil
}

// lineDiff produces a compact, focused line diff: it trims the common prefix
// and suffix lines and renders only the differing middle region with a couple
// of context lines, marking removed lines with "-" and added lines with "+".
// Deterministic and dependency-free.
func lineDiff(expected, actual string) string {
	e := strings.Split(expected, "\n")
	a := strings.Split(actual, "\n")

	prefix := 0
	for prefix < len(e) && prefix < len(a) && e[prefix] == a[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(e)-prefix && suffix < len(a)-prefix &&
		e[len(e)-1-suffix] == a[len(a)-1-suffix] {
		suffix++
	}

	const ctx = 2
	var b strings.Builder
	start := prefix - ctx
	if start < 0 {
		start = 0
	}
	for i := start; i < prefix; i++ {
		fmt.Fprintf(&b, "  %s\n", e[i])
	}

	emitted := 0
	for i := prefix; i < len(e)-suffix && emitted < maxDiffLines; i++ {
		fmt.Fprintf(&b, "- %s\n", e[i])
		emitted++
	}
	emitted = 0
	for i := prefix; i < len(a)-suffix && emitted < maxDiffLines; i++ {
		fmt.Fprintf(&b, "+ %s\n", a[i])
		emitted++
	}

	end := len(e) - suffix + ctx
	if end > len(e) {
		end = len(e)
	}
	for i := len(e) - suffix; i < end; i++ {
		fmt.Fprintf(&b, "  %s\n", e[i])
	}

	return b.String()
}
