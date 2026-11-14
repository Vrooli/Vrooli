package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSupervisorRespawnsAfterCrash kills the fake sidecar after the
// handshake reply, observes the supervisor goroutine respawn a new
// child, and confirms a subsequent Extract succeeds against the
// replacement.
func TestSupervisorRespawnsAfterCrash(t *testing.T) {
	requireNode(t)
	// KILL_AFTER_N=1 means the fake exits immediately after replying
	// to the handshake.
	s := newTestSupervisor(t, "KILL_AFTER_N=1")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	// Reach into the supervisor and clear the kill-after env on the
	// next respawn so subsequent children stay alive.
	s.mu.Lock()
	s.extraEnv = nil
	s.mu.Unlock()

	// Wait until status returns to READY after a respawn (generation > 1).
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		gen := s.generation
		st := s.status
		s.mu.Unlock()
		if st == StatusReady && gen >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	s.mu.Lock()
	gen := s.generation
	st := s.status
	s.mu.Unlock()
	require.GreaterOrEqual(t, gen, 2, "supervisor should have respawned at least once")
	require.Equal(t, StatusReady, st)

	// New Extract on the replacement child should succeed.
	res, err := s.Extract(ctx, "/tmp/example")
	require.NoError(t, err)
	require.Empty(t, res.Graph.Nodes)
}

// TestInFlightRequestDrainsOnCrash starts an Extract against a fake
// sidecar configured to die before responding. The in-flight request
// must unblock with ErrSidecarUnavailable.
func TestInFlightRequestDrainsOnCrash(t *testing.T) {
	requireNode(t)

	s := newTestSupervisor(t, "SLOW_EXTRACT_MS=10000")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	// Launch the Extract on a goroutine so we can observe its error
	// after the heartbeat kills the child.
	errCh := make(chan error, 1)
	go func() {
		_, err := s.Extract(ctx, "/tmp/example")
		errCh <- err
	}()

	require.Eventually(t, func() bool {
		s.mu.Lock()
		active := s.activeRequests
		cmd := s.cmd
		s.mu.Unlock()
		if active == 0 || cmd == nil || cmd.Process == nil {
			return false
		}
		return cmd.Process.Kill() == nil
	}, 2*time.Second, 25*time.Millisecond)

	select {
	case err := <-errCh:
		require.True(t,
			errors.Is(err, ErrSidecarUnavailable),
			"expected ErrSidecarUnavailable, got %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("in-flight Extract did not drain within 10s")
	}
}
