package orchestration_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
)

type immediateShutdownWaiter struct{}

func (immediateShutdownWaiter) Producer() string { return "shutdown-test" }

func (immediateShutdownWaiter) Wait(context.Context, string) (string, error) {
	return "ready", nil
}

type blockedWakeWaker struct {
	started   chan context.Context
	release   chan struct{}
	completed chan error
	calls     atomic.Int32
	returnErr error
}

func newBlockedWakeWaker() *blockedWakeWaker {
	return &blockedWakeWaker{
		started:   make(chan context.Context, 1),
		release:   make(chan struct{}),
		completed: make(chan error, 1),
	}
}

func (w *blockedWakeWaker) WakeRun(ctx context.Context, _ orchestration.WakeRunInput) (*domain.Run, error) {
	w.calls.Add(1)
	w.started <- ctx
	select {
	case <-ctx.Done():
		w.completed <- ctx.Err()
		return nil, ctx.Err()
	case <-w.release:
		w.completed <- w.returnErr
		return nil, w.returnErr
	}
}

func (*blockedWakeWaker) ListParkedRuns(context.Context) ([]*domain.Run, error) {
	return nil, nil
}

func awaitWakeContext(t *testing.T, waker *blockedWakeWaker) context.Context {
	t.Helper()
	select {
	case ctx := <-waker.started:
		return ctx
	case <-time.After(time.Second):
		t.Fatal("registry did not attempt the resolved wake")
		return nil
	}
}

func TestAwaitRegistryStopCancelsAndJoinsInFlightWake(t *testing.T) {
	waker := newBlockedWakeWaker()
	registry := orchestration.NewAwaitRegistry(waker, immediateShutdownWaiter{})
	registry.Register(uuid.New(), &domain.AwaitHandle{Producer: "shutdown-test", Key: "done"})
	ctx := awaitWakeContext(t, waker)

	stopped := make(chan struct{})
	go func() {
		registry.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		close(waker.release)
		<-stopped
		t.Fatal("Stop did not cancel the in-flight wake and join its watcher")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("wake context error = %v, want context.Canceled from registry shutdown", ctx.Err())
	}
	if err := <-waker.completed; !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight WakeRun error = %v, want context.Canceled", err)
	}
	if got := waker.calls.Load(); got != 1 {
		t.Fatalf("Stop triggered %d wake attempts, want 1", got)
	}
}

func TestAwaitRegistryCancelDuringWakeDoesNotCancelWakeContext(t *testing.T) {
	waker := newBlockedWakeWaker()
	registry := orchestration.NewAwaitRegistry(waker, immediateShutdownWaiter{})
	defer registry.Stop()

	runID := uuid.New()
	registry.Register(runID, &domain.AwaitHandle{Producer: "shutdown-test", Key: "done"})
	ctx := awaitWakeContext(t, waker)

	registry.Cancel(runID)
	if err := ctx.Err(); err != nil {
		t.Fatalf("per-run Cancel cancelled an in-flight WakeRun context: %v", err)
	}
	close(waker.release)
	select {
	case err := <-waker.completed:
		if err != nil {
			t.Fatalf("WakeRun after per-run Cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight WakeRun did not finish after release")
	}
	if got := waker.calls.Load(); got != 1 {
		t.Fatalf("Cancel caused %d wake attempts, want 1", got)
	}
}

func TestAwaitRegistryCancelsWatcherContextAfterWakeError(t *testing.T) {
	waker := newBlockedWakeWaker()
	waker.returnErr = errors.New("wake failed")
	registry := orchestration.NewAwaitRegistry(waker, immediateShutdownWaiter{})
	defer registry.Stop()

	runID := uuid.New()
	registry.Register(runID, &domain.AwaitHandle{Producer: "shutdown-test", Key: "done"})
	ctx := awaitWakeContext(t, waker)
	close(waker.release)

	if err := <-waker.completed; !errors.Is(err, waker.returnErr) {
		t.Fatalf("WakeRun error = %v, want %v", err, waker.returnErr)
	}
	eventually(t, func() bool { return ctx.Err() == context.Canceled }, "watcher exit did not release its child context")
	if got := waker.calls.Load(); got != 1 {
		t.Fatalf("wake failure caused %d attempts, want one visible attempt", got)
	}
}
