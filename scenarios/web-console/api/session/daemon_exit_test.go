package session

import (
	"context"
	"sync"
	"testing"

	"web-console/internal/backend"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// A session whose terminal process exits leaves nothing that can reach its
// agent runtime. The manager must hand that runtime to the installed stopper,
// or the daemon outlives the session for as long as the machine stays up.
func TestRecordProcessExitStopsAgentDaemon(t *testing.T) {
	useIsolatedSessionState(t)
	sm := NewManagerWithFactory(func(pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	var mu sync.Mutex
	var stopped []string
	sm.SetAgentDaemonStopper(func(_ context.Context, id string) {
		mu.Lock()
		defer mu.Unlock()
		stopped = append(stopped, id)
	})

	sm.recordProcessExit("sess-exit", backend.Standard)

	mu.Lock()
	defer mu.Unlock()
	if len(stopped) != 1 || stopped[0] != "sess-exit" {
		t.Fatalf("stopped = %v, want [sess-exit]", stopped)
	}
}

// A shutdown is not an exit: a persistent session's tmux pane and agent keep
// running across a restart, so its daemon must survive with them.
func TestShutdownExitKeepsPersistentAgentDaemon(t *testing.T) {
	useIsolatedSessionState(t)
	sm := NewManagerWithFactory(func(pty.LaunchSpec) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	})

	stopped := 0
	sm.SetAgentDaemonStopper(func(context.Context, string) { stopped++ })
	sm.mu.Lock()
	sm.shuttingDown = true
	sm.mu.Unlock()

	sm.recordProcessExit("sess-shutdown", backend.Persistent)

	if stopped != 0 {
		t.Fatalf("daemon stopped during shutdown: %d calls", stopped)
	}
}
