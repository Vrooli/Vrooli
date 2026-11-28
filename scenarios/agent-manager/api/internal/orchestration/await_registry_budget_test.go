package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
)

type wakeDeadlineWaker struct {
	deadlines chan observedWakeDeadline
}

type observedWakeDeadline struct {
	deadline   time.Time
	observedAt time.Time
}

func (w *wakeDeadlineWaker) WakeRun(ctx context.Context, _ WakeRunInput) (*domain.Run, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, context.DeadlineExceeded
	}
	w.deadlines <- observedWakeDeadline{deadline: deadline, observedAt: time.Now()}
	return nil, nil
}

func (*wakeDeadlineWaker) ListParkedRuns(context.Context) ([]*domain.Run, error) {
	return nil, nil
}

type immediateAwaitWaiter struct{}

func (immediateAwaitWaiter) Producer() string { return "immediate" }

func (immediateAwaitWaiter) Wait(context.Context, string) (string, error) {
	return "ready", nil
}

func TestAwaitRegistryWakeBudgetIncludesCompactionAndContinuation(t *testing.T) {
	waker := &wakeDeadlineWaker{deadlines: make(chan observedWakeDeadline, 1)}
	registry := NewAwaitRegistry(waker, immediateAwaitWaiter{})
	defer registry.Stop()

	registry.Register(uuid.New(), &domain.AwaitHandle{Producer: "immediate", Key: "done"})
	var observed observedWakeDeadline
	select {
	case observed = <-waker.deadlines:
	case <-time.After(time.Second):
		t.Fatal("registry did not attempt to deliver the resolved await")
	}

	// WakeRun waits for any bounded parked-session compaction before it starts
	// the continuation. The original 30s registry deadline could expire during
	// that wait, causing the one-shot watcher to disappear while the run stayed
	// parked. Keep the existing 30s continuation allowance after compaction.
	minimum := parkCompactionTimeout + 30*time.Second
	if remaining := observed.deadline.Sub(observed.observedAt); remaining < minimum-time.Second {
		t.Fatalf("wake context budget = %s, want at least compaction %s plus continuation allowance", remaining, minimum)
	}
}
