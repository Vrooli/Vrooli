package sidecar

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSupervisorRecoversWhenChildStopsReadingStdin reproduces a wedged
// sidecar: its event loop is stuck in synchronous work, so it stops reading
// stdin and the pipe fills. Callers must still return at their deadline and
// the heartbeat watchdog must still replace the child. Before frame writes
// honoured deadlines, a write into the full pipe blocked forever, pinned
// activeRequests above zero, and silenced the heartbeat for days.
func TestSupervisorRecoversWhenChildStopsReadingStdin(t *testing.T) {
	requireNode(t)

	s := newTestSupervisor(t, "HANG_ON_EXTRACT=1")
	s.cfg.HeartbeatInterval = 100 * time.Millisecond
	s.cfg.HeartbeatTimeout = 300 * time.Millisecond
	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startCancel()
	require.NoError(t, s.Start(startCtx))
	t.Cleanup(func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		_ = s.Shutdown(shutdownCtx)
	})

	// The first extract wedges the child's event loop.
	wedgeCtx, wedgeCancel := context.WithTimeout(context.Background(), time.Second)
	defer wedgeCancel()
	wedged := make(chan error, 1)
	go func() {
		_, err := s.Extract(wedgeCtx, "/tmp/wedge")
		wedged <- err
	}()
	time.Sleep(150 * time.Millisecond)

	// A frame larger than any pipe buffer cannot be delivered to a child
	// that no longer reads stdin.
	callCtx, callCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer callCancel()
	returned := make(chan error, 1)
	go func() {
		_, err := s.Extract(callCtx, "/tmp/"+strings.Repeat("x", 4<<20))
		returned <- err
	}()

	select {
	case err := <-returned:
		require.True(t, errors.Is(err, context.DeadlineExceeded), "expected deadline exceeded, got %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("Extract blocked past its deadline writing to a child that stopped reading stdin")
	}
	select {
	case err := <-wedged:
		require.True(t, errors.Is(err, context.DeadlineExceeded), "expected deadline exceeded, got %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("wedged Extract did not return at its deadline")
	}

	// With no caller waiting, the heartbeat must notice the wedged child
	// and replace it.
	s.mu.Lock()
	s.extraEnv = nil
	s.mu.Unlock()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		gen, st := s.generation, s.status
		s.mu.Unlock()
		if gen >= 2 && st == StatusReady {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("heartbeat did not replace a child that stopped reading stdin")
}
