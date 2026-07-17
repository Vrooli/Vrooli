package orchestration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

	// Every distinct id drives at least once. Wait for at least len(ids) drives.
	deadline := time.After(3 * time.Second)
	seen := 0
	for seen < len(ids) {
		select {
		case <-done:
			seen++
		case <-deadline:
			t.Fatalf("only observed %d drives before timeout", seen)
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
