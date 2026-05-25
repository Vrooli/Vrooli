package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/sidecar"
)

// thisFileDir returns the directory of this test file so fixture paths
// resolve regardless of the test working directory.
func thisFileDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Dir(file)
}

// fixtureAbsPath returns the absolute path to bas/fixtures/<name>.
func fixtureAbsPath(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(thisFileDir(t),
		"..", "..", "..", "bas", "fixtures", name)
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return abs
}

// sidecarDistPath returns the absolute path to the bundled sidecar entry.
func sidecarDistPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(thisFileDir(t),
		"..", "..", "..", "sidecar", "dist", "index.js")
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return abs
}

// requireRealSidecar skips the test when prerequisites are missing
// (node binary, built sidecar dist). On CI both should be present.
func requireRealSidecar(t *testing.T) (nodeBin, distPath string) {
	t.Helper()
	bin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node binary not on PATH; skipping integration test")
	}
	dist := sidecarDistPath(t)
	if _, err := os.Stat(dist); err != nil {
		t.Skipf("sidecar dist not built at %s; run pnpm build in sidecar/", dist)
	}
	return bin, dist
}

// safeBuf wraps bytes.Buffer with a mutex so the supervisor's stderr
// io.Copy goroutine can write concurrently with the test reading on
// cleanup without -race tripping.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// startRealSupervisor spawns the real Node sidecar and returns a ready
// Supervisor. The caller does not need to call Shutdown — Cleanup is
// registered automatically.
func startRealSupervisor(t *testing.T) *sidecar.Supervisor {
	t.Helper()
	nodeBin, dist := requireRealSidecar(t)
	stderr := &safeBuf{}
	t.Cleanup(func() {
		s := stderr.String()
		if s != "" {
			t.Logf("sidecar stderr:\n%s", s)
		}
	})
	sup := sidecar.NewSupervisor(sidecar.Config{
		DistPath:          dist,
		NodeBin:           nodeBin,
		HeartbeatInterval: 30 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
		HandshakeTimeout:  10 * time.Second,
		StderrSink:        stderr,
	})
	// Start's ctx scopes only the handshake window (the child's lifetime
	// is owned by the supervisor's internal context), so any ctx is safe
	// here; Shutdown in the t.Cleanup below performs the orderly
	// tear-down.
	if err := sup.Start(context.Background()); err != nil {
		t.Fatalf("sup.Start: %v\nsidecar stderr:\n%s", err, stderr.String())
	}
	st := sup.Status()
	if st != sidecar.StatusReady {
		t.Fatalf("sup.Status after Start = %s (want STATUS_READY)\nsidecar stderr:\n%s", st, stderr.String())
	}
	t.Cleanup(func() {
		shCtx, sc := context.WithTimeout(context.Background(), 3*time.Second)
		defer sc()
		_ = sup.Shutdown(shCtx)
	})
	return sup
}

// nodeKindToWire reverses graph.NodeKind back to the int32 wire enum
// values. Mirrors normalize.decodeNodeKind so the integration test can
// re-encode a domain Graph into the sidecar wire shape and compare
// byte-for-byte vs the committed expected-graph.json (which is captured
// from the sidecar wire output via `jq -S .graph`).
func nodeKindToWire(k graph.NodeKind) int32 {
	switch k {
	case graph.NodeKindFile:
		return 1
	case graph.NodeKindModule:
		return 200
	case graph.NodeKindComponent:
		return 201
	case graph.NodeKindHook:
		return 202
	case graph.NodeKindClass:
		return 203
	case graph.NodeKindInterface:
		return 204
	case graph.NodeKindType:
		return 205
	case graph.NodeKindFunction:
		return 206
	case graph.NodeKindVar:
		return 207
	case graph.NodeKindConst:
		return 208
	case graph.NodeKindReExport:
		return 209
	default:
		return 0
	}
}

func edgeKindToWire(k graph.EdgeKind) int32 {
	switch k {
	case graph.EdgeKindReExport:
		return 3
	case graph.EdgeKindImport:
		return 1
	default:
		return 0
	}
}

// wireNode is the canonical JSON shape we marshal for comparison
// against expected-graph.json. Field order matches `jq -S` (alphabetic).
type wireNode struct {
	Attributes      map[string]string `json:"attributes,omitempty"`
	ID              string            `json:"id"`
	Kind            int32             `json:"kind"`
	LeadingComments []string          `json:"leading_comments"`
	Name            string            `json:"name"`
	Path            string            `json:"path"`
}

type wireEdge struct {
	Attributes map[string]string `json:"attributes,omitempty"`
	FromNodeID string            `json:"from_node_id"`
	ID         string            `json:"id"`
	Kind       int32             `json:"kind"`
	ToNodeID   string            `json:"to_node_id"`
}

type wireGraph struct {
	Edges []wireEdge `json:"edges"`
	Nodes []wireNode `json:"nodes"`
}

// canonicalGraphJSON marshals a domain Graph to the canonical wire-shape
// JSON used by the committed expected-graph.json fixtures. Output is
// pretty-printed with two-space indent to match `jq -S`.
func canonicalGraphJSON(t *testing.T, g graph.Graph) []byte {
	t.Helper()
	wg := wireGraph{
		Nodes: make([]wireNode, 0, len(g.Nodes)),
		Edges: make([]wireEdge, 0, len(g.Edges)),
	}
	for _, n := range g.Nodes {
		lc := n.LeadingComments
		if lc == nil {
			lc = []string{}
		}
		wg.Nodes = append(wg.Nodes, wireNode{
			ID:              n.ID,
			Kind:            nodeKindToWire(n.Kind),
			Name:            n.Name,
			Path:            n.Path,
			Attributes:      sortedAttrs(n.Attributes),
			LeadingComments: lc,
		})
	}
	for _, e := range g.Edges {
		wg.Edges = append(wg.Edges, wireEdge{
			ID:         e.ID,
			Kind:       edgeKindToWire(e.Kind),
			FromNodeID: e.From,
			ToNodeID:   e.To,
			Attributes: sortedAttrs(e.Attributes),
		})
	}
	// re-sort to mirror what the sidecar emits (already sorted by
	// Normalize, but be explicit).
	sort.Slice(wg.Nodes, func(i, j int) bool { return wg.Nodes[i].ID < wg.Nodes[j].ID })
	sort.Slice(wg.Edges, func(i, j int) bool {
		if wg.Edges[i].FromNodeID != wg.Edges[j].FromNodeID {
			return wg.Edges[i].FromNodeID < wg.Edges[j].FromNodeID
		}
		return wg.Edges[i].ToNodeID < wg.Edges[j].ToNodeID
	})

	// encoding/json sorts map keys alphabetically (matching `jq -S`).
	// We must disable HTMLEscape so `>` survives as `>` rather than
	// `>` — jq emits the raw character.
	var raw bytes.Buffer
	enc := json.NewEncoder(&raw)
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(wg))
	// Drop the trailing newline json.Encoder appends so json.Indent
	// produces a clean buffer, then re-indent.
	raw0 := bytes.TrimRight(raw.Bytes(), "\n")
	var pretty bytes.Buffer
	require.NoError(t, json.Indent(&pretty, raw0, "", "  "))
	pretty.WriteByte('\n')
	return pretty.Bytes()
}

// sortedAttrs returns a copy of in, or nil if empty. encoding/json
// already sorts map keys when marshaling so we just need to nil-out
// empties.
func sortedAttrs(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// TestExtractIntegration runs the real sidecar against each fixture and
// asserts the produced graph matches the committed expected-graph.json
// byte-for-byte. UPDATE_FIXTURES=1 rewrites the committed files in dev
// mode.
func TestExtractIntegration(t *testing.T) {
	for _, name := range []string{"ts-junk-drawer", "ts-jsdoc-tags"} {
		name := name
		t.Run(name, func(t *testing.T) {
			sup := startRealSupervisor(t)
			svc := graph.NewService(sup, graph.NewPathMutex())

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			fixDir := fixtureAbsPath(t, name)
			out, err := svc.Extract(ctx, graph.ExtractInput{ScenarioPath: fixDir})
			require.NoError(t, err)

			got := canonicalGraphJSON(t, out.Graph)
			expectPath := filepath.Join(fixDir, "expected-graph.json")

			if os.Getenv("UPDATE_FIXTURES") == "1" {
				require.NoError(t, os.WriteFile(expectPath, got, 0o644))
				return
			}

			want, err := os.ReadFile(expectPath)
			require.NoError(t, err, "missing %s — run with UPDATE_FIXTURES=1", expectPath)
			require.Equal(t, string(want), string(got),
				"canonical graph diverged from committed fixture; "+
					"if intentional, re-run with UPDATE_FIXTURES=1")
		})
	}
}
