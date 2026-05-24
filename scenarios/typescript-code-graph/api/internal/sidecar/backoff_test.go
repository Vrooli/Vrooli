package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSupervisorExhaustsRestartBudget configures the fake sidecar to
// die immediately after every handshake. After 5 crashes in under 60s,
// the supervisor should mark itself PERMANENTLY_UNHEALTHY and refuse
// further requests.
func TestSupervisorExhaustsRestartBudget(t *testing.T) {
	requireNode(t)

	s := newTestSupervisor(t, "KILL_AFTER_N=1")
	// Disable heartbeats; they would race with the crash loop and add
	// noise to the timing.
	s.cfg.HeartbeatInterval = time.Hour

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	// Wait for the supervisor to give up.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if s.Status() == StatusPermanentlyUnhealthy {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.Equal(t, StatusPermanentlyUnhealthy, s.Status(),
		"supervisor should be permanently unhealthy after exhausting restart budget")

	// Subsequent requests must surface the permanent error.
	_, _, err := s.Extract(context.Background(), "/tmp/example")
	require.True(t,
		errors.Is(err, ErrSidecarPermanentlyUnhealthy),
		"expected ErrSidecarPermanentlyUnhealthy, got %v", err)
}
