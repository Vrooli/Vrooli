package sidecar

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeSidecarPath returns the absolute path to testdata/fake_sidecar.js.
// runtime.Caller(0) anchors us to this test file so we don't depend on
// the test working directory.
func fakeSidecarPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "testdata", "fake_sidecar.js")
}

// requireNode skips the test when the `node` binary is not on PATH.
// Process-based tests in this package depend on node being present;
// pure-Go tests do not.
func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node binary not on PATH; skipping process-based test")
	}
}

// newTestSupervisor builds a supervisor that points at the fake
// sidecar and uses test-friendly short intervals.
func newTestSupervisor(t *testing.T, env ...string) *Supervisor {
	t.Helper()
	cfg := Config{
		DistPath:          fakeSidecarPath(t),
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  200 * time.Millisecond,
		HandshakeTimeout:  2 * time.Second,
		StderrSink:        io.Discard,
	}
	s := NewSupervisor(cfg)
	s.extraEnv = env
	return s
}

func TestSupervisorHappyPath(t *testing.T) {
	requireNode(t)
	s := newTestSupervisor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	require.Equal(t, StatusReady, s.Status())

	res, err := s.Extract(ctx, "/tmp/example")
	require.NoError(t, err)
	require.Empty(t, res.Graph.Nodes)
	require.Empty(t, res.Graph.Edges)
	require.Empty(t, res.Warnings)
	require.NotEmpty(t, res.RequestID, "Extract must surface the IPC request id")

	results, err := s.RewriteApply(ctx, "/tmp/example", []Operation{
		{FileMove: &FileMove{From: "a.ts", To: "b.ts"}},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "OPERATION_STATUS_OK", results[0].Status)
}

func TestSupervisorShutdownStopsChild(t *testing.T) {
	requireNode(t)
	s := newTestSupervisor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, s.Start(ctx))

	shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
	defer c()
	require.NoError(t, s.Shutdown(shutdownCtx))

	// After shutdown, the supervisor goroutine has returned.
	select {
	case <-s.supDone:
	case <-time.After(time.Second):
		t.Fatal("supDone did not close after Shutdown")
	}
}

// TestSupervisorStartCtxCancelDoesNotKillChild pins the D5 contract: the
// ctx passed to Start scopes only the startup/handshake window. Once
// Start returns, cancelling that ctx must NOT tear the child down —
// teardown happens exclusively through Shutdown.
func TestSupervisorStartCtxCancelDoesNotKillChild(t *testing.T) {
	requireNode(t)
	s := newTestSupervisor(t)

	startCtx, cancelStart := context.WithCancel(context.Background())
	require.NoError(t, s.Start(startCtx))
	require.Equal(t, StatusReady, s.Status())

	// Cancel the caller's startup ctx and give any erroneous SIGKILL
	// plenty of time to propagate.
	cancelStart()
	time.Sleep(300 * time.Millisecond)

	// The child must still be alive and serving requests.
	require.Equal(t, StatusReady, s.Status(),
		"child must survive cancellation of the Start ctx")
	res, err := s.Extract(context.Background(), "/tmp/example")
	require.NoError(t, err, "Extract must still succeed after Start ctx cancel")
	require.NotEmpty(t, res.RequestID)

	// Graceful teardown still works via Shutdown.
	shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
	defer c()
	require.NoError(t, s.Shutdown(shutdownCtx))
	select {
	case <-s.supDone:
	case <-time.After(time.Second):
		t.Fatal("supDone did not close after Shutdown")
	}
}
