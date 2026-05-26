package graph_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// The canonical wire-shape JSON used by the committed expected-graph.json
// fixtures is produced by the production graph.CanonicalJSON (see
// internal/graph/canonical.go). TestExtractIntegration calls it directly so
// the determinism check validates the exact code path ValidateFixture uses —
// there is intentionally no parallel canonicalizer in this test.

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

			got, err := graph.CanonicalJSON(out.Graph)
			require.NoError(t, err)
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
