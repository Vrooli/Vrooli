package orchestration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil/mocks"
)

// Compile-time proof the mocks fake satisfies the production Waiter seam.
// (Lives here, not in mocks, because white-box orchestration tests import
// mocks — so mocks must not import orchestration.)
var _ orchestration.Waiter = (*mocks.FakeWaiter)(nil)

// recordingWaker is a minimal awaitWaker for registry tests. It records every
// wake and, like the production *Orchestrator, cancels the registry watcher on
// wake (so the test mirrors real double-resolve safety).
type recordingWaker struct {
	mu     sync.Mutex
	wakes  []orchestration.WakeRunInput
	parked []*domain.Run

	reg    *orchestration.AwaitRegistry
	wakeCh chan orchestration.WakeRunInput
}

func newRecordingWaker() *recordingWaker {
	return &recordingWaker{wakeCh: make(chan orchestration.WakeRunInput, 8)}
}

func (w *recordingWaker) WakeRun(_ context.Context, in orchestration.WakeRunInput) (*domain.Run, error) {
	w.mu.Lock()
	w.wakes = append(w.wakes, in)
	reg := w.reg
	w.mu.Unlock()
	if reg != nil {
		reg.Cancel(in.RunID)
	}
	select {
	case w.wakeCh <- in:
	default:
	}
	return nil, nil
}

func (w *recordingWaker) ListParkedRuns(context.Context) ([]*domain.Run, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.parked, nil
}

func (w *recordingWaker) wakeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.wakes)
}

func (w *recordingWaker) awaitWake(t *testing.T) orchestration.WakeRunInput {
	t.Helper()
	select {
	case in := <-w.wakeCh:
		return in
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for wake")
		return orchestration.WakeRunInput{}
	}
}

func handle(producer, key string, deadline *time.Time) *domain.AwaitHandle {
	return &domain.AwaitHandle{
		Producer:     producer,
		Key:          key,
		Deadline:     deadline,
		RegisteredAt: time.Now(),
	}
}

// eventually polls cond until true or the deadline, failing otherwise.
func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestAwaitRegistry_ResolveWakesRun(t *testing.T) {
	waker := newRecordingWaker()
	waiter := mocks.NewFakeWaiter(orchestration.ProducerTestGenie)
	reg := orchestration.NewAwaitRegistry(waker, waiter)
	waker.reg = reg
	defer reg.Stop()

	runID := uuid.New()
	reg.Register(runID, handle(orchestration.ProducerTestGenie, "scn/run-1", nil))
	eventually(t, func() bool { return reg.Watching(runID) }, "watcher not registered")

	waiter.Resolve(`{"status":"passed"}`)

	in := waker.awaitWake(t)
	if in.RunID != runID {
		t.Fatalf("woke wrong run: got %s want %s", in.RunID, runID)
	}
	if in.TimedOut {
		t.Fatal("resolve should not be a timeout wake")
	}
	if in.Result != `{"status":"passed"}` {
		t.Fatalf("unexpected result %q", in.Result)
	}
	if got := waiter.Calls(); len(got) != 1 || got[0] != "scn/run-1" {
		t.Fatalf("unexpected Wait calls: %v", got)
	}
	eventually(t, func() bool { return !reg.Watching(runID) }, "watcher not deregistered after wake")
}

func TestAwaitRegistry_ProducerErrorWakesWithErrorResult(t *testing.T) {
	waker := newRecordingWaker()
	waiter := mocks.NewFakeWaiter(orchestration.ProducerGCT)
	reg := orchestration.NewAwaitRegistry(waker, waiter)
	waker.reg = reg
	defer reg.Stop()

	runID := uuid.New()
	reg.Register(runID, handle(orchestration.ProducerGCT, "scn/baseline", nil))
	eventually(t, func() bool { return reg.Watching(runID) }, "watcher not registered")

	waiter.Fail(errors.New("diff backend unavailable"))

	in := waker.awaitWake(t)
	if in.TimedOut {
		t.Fatal("producer error should not be a timeout wake")
	}
	if want := "[wait error]"; len(in.Result) == 0 || in.Result[:len(want)] != want {
		t.Fatalf("error wake result should start with %q, got %q", want, in.Result)
	}
}

func TestAwaitRegistry_DeadlineWakesTimedOut(t *testing.T) {
	waker := newRecordingWaker()
	// FakeWaiter blocks until ctx is done; with a short deadline that means the
	// wait ctx hits DeadlineExceeded and the registry issues a timeout-wake.
	waiter := mocks.NewFakeWaiter(orchestration.ProducerTestGenie)
	reg := orchestration.NewAwaitRegistry(waker, waiter)
	waker.reg = reg
	defer reg.Stop()

	runID := uuid.New()
	deadline := time.Now().Add(80 * time.Millisecond)
	reg.Register(runID, handle(orchestration.ProducerTestGenie, "scn/run-x", &deadline))

	in := waker.awaitWake(t)
	if !in.TimedOut {
		t.Fatalf("expected timeout wake, got %+v", in)
	}
	if in.RunID != runID {
		t.Fatalf("woke wrong run")
	}
}

func TestAwaitRegistry_CancelDoesNotWake(t *testing.T) {
	waker := newRecordingWaker()
	waiter := mocks.NewFakeWaiter(orchestration.ProducerTestGenie)
	reg := orchestration.NewAwaitRegistry(waker, waiter)
	defer reg.Stop()

	runID := uuid.New()
	reg.Register(runID, handle(orchestration.ProducerTestGenie, "scn/run-c", nil))
	eventually(t, func() bool { return reg.Watching(runID) }, "watcher not registered")

	reg.Cancel(runID)

	eventually(t, func() bool { return !reg.Watching(runID) }, "watcher not deregistered after cancel")
	// Give any errant wake a chance to land, then assert none happened.
	time.Sleep(30 * time.Millisecond)
	if n := waker.wakeCount(); n != 0 {
		t.Fatalf("cancel should not wake; got %d wakes", n)
	}
}

func TestAwaitRegistry_UnknownProducerWakesImmediately(t *testing.T) {
	waker := newRecordingWaker()
	reg := orchestration.NewAwaitRegistry(waker) // no waiters registered
	waker.reg = reg
	defer reg.Stop()

	runID := uuid.New()
	reg.Register(runID, handle("mystery", "scn/thing", nil))

	in := waker.awaitWake(t)
	if in.TimedOut {
		t.Fatal("unknown producer should be an immediate error wake, not a timeout")
	}
	if want := "[wait error] no waiter"; len(in.Result) < len(want) || in.Result[:len(want)] != want {
		t.Fatalf("expected no-waiter error, got %q", in.Result)
	}
}

func TestAwaitRegistry_RegisterIsIdempotentPerRun(t *testing.T) {
	waker := newRecordingWaker()
	waiter := mocks.NewFakeWaiter(orchestration.ProducerTestGenie)
	reg := orchestration.NewAwaitRegistry(waker, waiter)
	defer reg.Stop()

	runID := uuid.New()
	h := handle(orchestration.ProducerTestGenie, "scn/run-d", nil)
	reg.Register(runID, h)
	reg.Register(runID, h) // second register is a no-op (one handle per run)
	eventually(t, func() bool { return reg.Watching(runID) }, "watcher not registered")
	if c := reg.ActiveCount(); c != 1 {
		t.Fatalf("expected exactly one watcher, got %d", c)
	}
}

func TestAwaitRegistry_RecoverParkedRunsRespawnsWaiters(t *testing.T) {
	waker := newRecordingWaker()
	deadlineless := (*time.Time)(nil)
	parked1 := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked, AwaitHandle: handle(orchestration.ProducerTestGenie, "scn/r1", deadlineless)}
	parked2 := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked, AwaitHandle: handle(orchestration.ProducerGCT, "scn/r2", deadlineless)}
	noHandle := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked} // skipped
	waker.parked = []*domain.Run{parked1, parked2, noHandle}

	reg := orchestration.NewAwaitRegistry(
		waker,
		mocks.NewFakeWaiter(orchestration.ProducerTestGenie),
		mocks.NewFakeWaiter(orchestration.ProducerGCT),
	)
	defer reg.Stop()

	n, err := reg.RecoverParkedRuns(context.Background())
	if err != nil {
		t.Fatalf("RecoverParkedRuns: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 waiters re-spawned (handle-less run skipped), got %d", n)
	}
	eventually(t, func() bool { return reg.Watching(parked1.ID) && reg.Watching(parked2.ID) }, "recovered watchers not registered")
	if reg.Watching(noHandle.ID) {
		t.Fatal("handle-less parked run should not have a watcher")
	}
}

func TestAwaitRegistry_StopCancelsAllWaiters(t *testing.T) {
	waker := newRecordingWaker()
	reg := orchestration.NewAwaitRegistry(waker, mocks.NewFakeWaiter(orchestration.ProducerTestGenie))

	for i := 0; i < 3; i++ {
		reg.Register(uuid.New(), handle(orchestration.ProducerTestGenie, "scn/run", nil))
	}
	eventually(t, func() bool { return reg.ActiveCount() == 3 }, "watchers not all registered")

	reg.Stop()

	if c := reg.ActiveCount(); c != 0 {
		t.Fatalf("Stop should drain all watchers, got %d", c)
	}
	if n := waker.wakeCount(); n != 0 {
		t.Fatalf("Stop should not wake runs; got %d wakes", n)
	}
	// Register after Stop is a no-op.
	reg.Register(uuid.New(), handle(orchestration.ProducerTestGenie, "scn/late", nil))
	if c := reg.ActiveCount(); c != 0 {
		t.Fatalf("Register after Stop should be a no-op, got %d", c)
	}
}
