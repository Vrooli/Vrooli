package sidecar

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestHeartbeatKillsUnresponsiveChild configures the fake sidecar to
// swallow heartbeats; the supervisor must kill it and respawn.
func TestHeartbeatKillsUnresponsiveChild(t *testing.T) {
	requireNode(t)

	s := newTestSupervisor(t, "IGNORE_HEARTBEAT=1")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	// Clear IGNORE_HEARTBEAT so the respawned child answers heartbeats.
	s.mu.Lock()
	s.extraEnv = nil
	s.mu.Unlock()

	// Wait for generation to advance (proof of respawn).
	deadline := time.Now().Add(10 * time.Second)
	respawned := false
	for time.Now().Before(deadline) {
		s.mu.Lock()
		gen := s.generation
		st := s.status
		s.mu.Unlock()
		if gen >= 2 && st == StatusReady {
			respawned = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.True(t, respawned, "supervisor did not respawn unresponsive child")
}

func TestHeartbeatSkipsWhileExtractIsActive(t *testing.T) {
	requireNode(t)

	s := newTestSupervisor(t, "BLOCK_EXTRACT_MS=500")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	res, err := s.Extract(ctx, "/tmp/example")
	require.NoError(t, err)
	require.NotEmpty(t, res.RequestID)
	require.Equal(t, StatusReady, s.Status())

	s.mu.Lock()
	gen := s.generation
	s.mu.Unlock()
	require.Equal(t, 1, gen, "heartbeat must not kill and respawn a busy sidecar")
}
