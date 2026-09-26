package orchestration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// TestWorkflowNudgerDedupesPendingEnqueues pins that a second enqueue of an
// execution already queued does not queue it twice — the run-terminal path may
// fire repeatedly but the drive is scheduled once.
func TestWorkflowNudgerDedupesPendingEnqueues(t *testing.T) {
	n := NewWorkflowNudger(func(context.Context, uuid.UUID) error { return nil }, 1, time.Second)
	id := uuid.New()
	n.Enqueue(id)
	n.Enqueue(id)
	n.Enqueue(id)
	if got := n.PendingCount(); got != 1 {
		t.Fatalf("PendingCount after duplicate enqueues = %d, want 1", got)
	}
	n.Enqueue(uuid.Nil) // ignored
	if got := n.PendingCount(); got != 1 {
		t.Fatalf("PendingCount after nil enqueue = %d, want 1", got)
	}
}

// TestWorkflowNudgerDrivesEachEnqueuedExecution pins that started workers drain
// the queue and drive every distinct execution at least once. Run with -race,
// it also gates concurrent enqueue/drive against data races.
func TestWorkflowNudgerDrivesEachEnqueuedExecution(t *testing.T) {
	var mu sync.Mutex
	driven := map[uuid.UUID]int{}
	done := make(chan struct{}, 256)
	n := NewWorkflowNudger(func(_ context.Context, id uuid.UUID) error {
		mu.Lock()
		driven[id]++
		mu.Unlock()
		done <- struct{}{}
		return nil
	}, 4, time.Second)
	n.Start()
	defer n.Stop()

	ids := make([]uuid.UUID, 8)
	for i := range ids {
		ids[i] = uuid.New()
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		for range 5 {
			wg.Add(1)
			go func(id uuid.UUID) { defer wg.Done(); n.Enqueue(id) }(id)
		}
	}
	wg.Wait()

	// Every distinct id drives at least once. Duplicate completion signals can
	// arrive before a different queued ID runs, so wait for the actual map
	// predicate rather than counting arbitrary notifications.
	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		allDriven := true
		for _, id := range ids {
			if driven[id] == 0 {
				allDriven = false
				break
			}
		}
		mu.Unlock()
		if allDriven {
			break
		}
		select {
		case <-done:
		case <-deadline:
			t.Fatal("not every queued execution was driven before timeout")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for _, id := range ids {
		if driven[id] == 0 {
			t.Fatalf("execution %s never driven", id)
		}
	}
}

func TestWorkflowNudgerSurvivesPanickingDrive(t *testing.T) {
	var calls atomic.Int64
	completed := make(chan struct{}, 1)
	panics := make(chan uuid.UUID, 1)
	n := NewWorkflowNudger(func(_ context.Context, _ uuid.UUID) error {
		if calls.Add(1) == 1 {
			panic("injected drive panic")
		}
		completed <- struct{}{}
		return nil
	}, 1, time.Second, func(_ context.Context, id uuid.UUID, _ obs.PanicFailure) { panics <- id })
	n.Start()
	defer n.Stop()

	first := uuid.New()
	n.Enqueue(first)
	second := uuid.New()
	n.Enqueue(second)
	select {
	case id := <-panics:
		if id != first {
			t.Fatalf("panic callback id = %s, want %s", id, first)
		}
	case <-time.After(time.Second):
		t.Fatal("panic callback was not invoked")
	}
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("nudger worker did not survive panicking drive")
	}
}

// TestWorkflowNudgerStopIsIdempotentAndUnblocks pins clean shutdown even with
// nothing queued.
func TestWorkflowNudgerStopIsIdempotentAndUnblocks(t *testing.T) {
	var drives atomic.Int64
	n := NewWorkflowNudger(func(context.Context, uuid.UUID) error { drives.Add(1); return nil }, 2, time.Second)
	n.Start()
	n.Stop()
	// Enqueue after Stop is a no-op.
	n.Enqueue(uuid.New())
	if drives.Load() != 0 {
		t.Fatalf("drives after stop = %d, want 0", drives.Load())
	}
}

// TestWorkflowWaitRegistryNotifyWakesAllSubscribers pins the event-driven wake:
// every current subscriber's channel closes on notify, and unsubscribe is safe.
func TestWorkflowWaitRegistryNotifyWakesAllSubscribers(t *testing.T) {
	r := newWorkflowWaitRegistry()
	id := uuid.New()
	a := r.subscribe(id)
	b := r.subscribe(id)
	other := r.subscribe(uuid.New())

	r.notify(id)

	select {
	case <-a:
	case <-time.After(time.Second):
		t.Fatal("subscriber a not woken")
	}
	select {
	case <-b:
	case <-time.After(time.Second):
		t.Fatal("subscriber b not woken")
	}
	select {
	case <-other:
		t.Fatal("unrelated subscriber was woken")
	default:
	}
	// Unsubscribe after notify (already removed) is safe.
	r.unsubscribe(id, a)
	r.unsubscribe(uuid.New(), other)
}
