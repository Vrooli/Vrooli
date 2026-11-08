package orchestration

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
	coredb "github.com/vrooli/api-core/database"
)

func TestWorkflowNudgerKeepsOriginalLeaseAfterProducerCancellationAndReplacement(t *testing.T) {
	db, err := coredb.Open(t.Context(), coredb.Config{Driver: coredb.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(t.Context(), "CREATE TABLE effects(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	install := func(name string) {
		t.Helper()
		if err := db.InstallTestPool(t.Context(), "file:"+filepath.Join(t.TempDir(), name), "lease", time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	install("original.db")
	request, err := db.PinTestPool(t.Context(), "lease")
	if err != nil {
		t.Fatal(err)
	}
	request, cancel := context.WithCancel(request)
	defer cancel()
	completed := make(chan error, 2)
	n := NewWorkflowNudger(func(ctx context.Context, _ uuid.UUID) error {
		_, err := db.ExecContext(ctx, "INSERT INTO effects VALUES ('queued')")
		completed <- err
		return err
	}, 1, time.Second)
	defer n.Stop()
	id := uuid.New()
	n.Enqueue(request, id)
	cancel()
	n.Start()
	await := func() error {
		t.Helper()
		select {
		case err := <-completed:
			return err
		case <-time.After(2 * time.Second):
			t.Fatal("queued drive did not return")
			return nil
		}
	}
	if err := await(); err != nil {
		t.Fatalf("producer cancellation discarded admitted work: %v", err)
	}
	var count int
	if err := db.QueryRowContext(context.WithoutCancel(request), "SELECT COUNT(*) FROM effects").Scan(&count); err != nil || count != 1 {
		t.Fatalf("original leased effect missing: count=%d err=%v", count, err)
	}
	if err := db.ClearTestPool("lease"); err != nil {
		t.Fatal(err)
	}
	install("replacement.db")
	n.Enqueue(request, id)
	if err := await(); err == nil {
		t.Fatal("old queued context selected a replacement lease or primary pool")
	}
	for _, ctx := range []context.Context{t.Context(), coredb.WithTestMode(t.Context())} {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM effects").Scan(&count); err != nil || count != 0 {
			t.Fatalf("old context changed another storage owner: count=%d err=%v", count, err)
		}
	}
	replacement, err := db.PinTestPool(t.Context(), "lease")
	if err != nil {
		t.Fatal(err)
	}
	// Identical execution IDs in separate storage generations cannot suppress
	// one another's queued work. Repeated notifications within one owner dedupe.
	queued := NewWorkflowNudger(func(context.Context, uuid.UUID) error { return nil }, 1, time.Second)
	defer queued.Stop()
	for _, ctx := range []context.Context{t.Context(), request, replacement} {
		queued.Enqueue(ctx, id)
		queued.Enqueue(ctx, id)
	}
	if got := queued.PendingCount(); got != 3 {
		t.Fatalf("cross-owner deduplication lost queued work: pending=%d want3", got)
	}
}

// TestWorkflowNudgerDedupesPendingEnqueues pins that a second enqueue of an
// execution already queued does not queue it twice — the run-terminal path may
// fire repeatedly but the drive is scheduled once.
func TestWorkflowNudgerDedupesPendingEnqueues(t *testing.T) {
	n := NewWorkflowNudger(func(context.Context, uuid.UUID) error { return nil }, 1, time.Second)
	id := uuid.New()
	n.Enqueue(t.Context(), id)
	n.Enqueue(t.Context(), id)
	n.Enqueue(t.Context(), id)
	if got := n.PendingCount(); got != 1 {
		t.Fatalf("PendingCount after duplicate enqueues = %d, want 1", got)
	}
	n.Enqueue(t.Context(), uuid.Nil) // ignored
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
			go func(id uuid.UUID) { defer wg.Done(); n.Enqueue(t.Context(), id) }(id)
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
	n.Enqueue(t.Context(), first)
	second := uuid.New()
	n.Enqueue(t.Context(), second)
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
	n.Enqueue(t.Context(), uuid.New())
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
