package interactive

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

type delayedScreenSessions struct {
	*fakeSessions
	entered chan struct{}
	release chan struct{}
}

func (s *delayedScreenSessions) Screen(context.Context, string, bool) (string, error) {
	close(s.entered)
	<-s.release
	return "selected model is at capacity", nil
}

func TestSessionWatcherDiscardsScreenReturnedAfterCancellation(t *testing.T) {
	sessions := &delayedScreenSessions{fakeSessions: newFakeSessions("session"), entered: make(chan struct{}), release: make(chan struct{})}
	c := NewCoordinator(CoordinatorDeps{Sessions: sessions, SessionPoll: time.Millisecond})
	run := newInteractiveRun(domain.RunnerTypeCodex, "fixture.jsonl")
	run.WebConsoleSessionID = "session"
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, done := c.watchSession(ctx, ctx, cancel, run)
	select {
	case <-sessions.entered:
	case <-time.After(time.Second):
		t.Fatal("watcher did not reach the delayed screen")
	}
	cancel()
	close(sessions.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not exit after cancellation")
	}
	if c.recoverableInterruption.Load() || c.recoveryScheduled.Load() {
		t.Fatal("late screen from a canceled watcher scheduled a recovery")
	}
}
