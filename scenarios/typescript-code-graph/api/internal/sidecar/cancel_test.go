package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCancelPendingExtract verifies that cancelling the caller's
// context unblocks Extract immediately with ctx.Err(). The cancel IPC
// is best-effort and not asserted on the wire here (the fake just
// logs it).
func TestCancelPendingExtract(t *testing.T) {
	requireNode(t)

	// SLOW_EXTRACT_MS=5000 ensures the extract reply is well past our
	// ctx-cancel point. Heartbeat interval is bumped up so the
	// supervisor does not kill the child during the test.
	cfg := Config{
		DistPath:          fakeSidecarPath(t),
		HeartbeatInterval: 30 * time.Second,
		HeartbeatTimeout:  5 * time.Second,
		HandshakeTimeout:  2 * time.Second,
	}
	s := NewSupervisor(cfg)
	s.extraEnv = []string{"SLOW_EXTRACT_MS=5000"}

	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startCancel()
	require.NoError(t, s.Start(startCtx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	callCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := s.Extract(callCtx, "/tmp/example")
		errCh <- err
	}()

	// Let the request enqueue, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.True(t,
			errors.Is(err, context.Canceled),
			"expected context.Canceled, got %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Extract did not return after ctx cancel")
	}
}
