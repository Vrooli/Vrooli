package spawn

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// =============================================================================
// HELPERS
// =============================================================================

// captureSink records every event emitted via obs.Sink.Emit so tests
// can assert on the lifecycle taxonomy without spinning up the event
// store.
type captureSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (c *captureSink) Emit(evt *domain.RunEvent) error {
	c.mu.Lock()
	c.events = append(c.events, evt)
	c.mu.Unlock()
	return nil
}

func (c *captureSink) snapshot() []*domain.RunEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]*domain.RunEvent, len(c.events))
	copy(cp, c.events)
	return cp
}

// makeJob builds a minimally-valid Job for testing. Passing a nil
// *captureSink yields a Job with a nil-interface Sink (not a typed
// nil, which would defeat obs's nil-check and panic).
func makeJob(t *testing.T, sink *captureSink, fn ExecuteFn) *Job {
	t.Helper()
	job := &Job{
		RunID:      uuid.New(),
		RunMode:    domain.RunModeSandboxed,
		RunnerType: domain.RunnerType("codex"),
		Fn:         fn,
	}
	if sink != nil {
		job.Sink = sink
	}
	return job
}

// awaitTrue polls cond up to timeout. Fails the test if cond never
// returns true. Used in lieu of time.Sleep so tests are responsive.
func awaitTrue(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting: %s", msg)
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

func TestNew_PanicsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  Config
	}{
		{"zero MaxStartingConcurrency", Config{MaxStartingConcurrency: 0, QueueCapacity: 4}},
		{"negative MaxStartingConcurrency", Config{MaxStartingConcurrency: -1, QueueCapacity: 4}},
		{"QueueCapacity below MaxStartingConcurrency", Config{MaxStartingConcurrency: 4, QueueCapacity: 2}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic for %+v", tc.cfg)
				}
			}()
			_ = New(tc.cfg)
		})
	}
}

// =============================================================================
// CORE BEHAVIOUR
// =============================================================================

func TestDispatcher_RunsSingleJob(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 4})
	defer d.Close()

	sink := &captureSink{}
	done := make(chan struct{})
	job := makeJob(t, sink, func(started StartedFn) {
		started()
		close(done)
	})

	if err := d.Enqueue(job); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ExecuteFn did not run within 2s")
	}

	awaitTrue(t, time.Second, "Stats() to drain", func() bool {
		s := d.Stats()
		return s.QueueDepth == 0 && s.ActiveCount == 0 && s.StartingCount == 0
	})

	// Lifecycle taxonomy: exactly spawn-enqueued + spawn-started.
	events := sink.snapshot()
	if len(events) != 2 {
		t.Fatalf("expected 2 lifecycle events, got %d", len(events))
	}
	if events[0].EventType != domain.EventTypeLifecycle {
		t.Errorf("event[0] type = %q, want lifecycle", events[0].EventType)
	}
	if events[1].EventType != domain.EventTypeLifecycle {
		t.Errorf("event[1] type = %q, want lifecycle", events[1].EventType)
	}
}

func TestDispatcher_SerializesStartupSlot(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 16})
	defer d.Close()

	const jobs = 5

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	release := make(chan struct{}) // gate: jobs hold their slot until released
	completed := make(chan struct{}, jobs)

	for i := 0; i < jobs; i++ {
		fn := func(started StartedFn) {
			now := concurrent.Add(1)
			for {
				cur := maxConcurrent.Load()
				if now <= cur {
					break
				}
				if maxConcurrent.CompareAndSwap(cur, now) {
					break
				}
			}
			<-release
			concurrent.Add(-1)
			started()
			completed <- struct{}{}
		}
		if err := d.Enqueue(makeJob(t, nil, fn)); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}

	// Give the worker a moment to start the first job.
	awaitTrue(t, time.Second, "first job to start", func() bool {
		return concurrent.Load() == 1
	})

	// Release one job at a time and verify the next picks up only after
	// the previous releases the slot.
	for i := 0; i < jobs; i++ {
		release <- struct{}{}
		<-completed
	}

	if got := maxConcurrent.Load(); got != 1 {
		t.Errorf("max concurrent startups = %d, want 1 (MaxStartingConcurrency=1)", got)
	}

	awaitTrue(t, time.Second, "Stats() to drain", func() bool {
		s := d.Stats()
		return s.QueueDepth == 0 && s.ActiveCount == 0 && s.StartingCount == 0
	})
}

func TestDispatcher_ParallelStartupWithCapacity(t *testing.T) {
	t.Parallel()

	const cap = 3
	d := New(Config{MaxStartingConcurrency: cap, QueueCapacity: 16})
	defer d.Close()

	const jobs = cap * 2

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	release := make(chan struct{})
	completed := make(chan struct{}, jobs)

	for i := 0; i < jobs; i++ {
		fn := func(started StartedFn) {
			now := concurrent.Add(1)
			for {
				cur := maxConcurrent.Load()
				if now <= cur {
					break
				}
				if maxConcurrent.CompareAndSwap(cur, now) {
					break
				}
			}
			<-release
			concurrent.Add(-1)
			started()
			completed <- struct{}{}
		}
		if err := d.Enqueue(makeJob(t, nil, fn)); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}

	// Wait until the dispatcher has filled the slot ceiling.
	awaitTrue(t, 2*time.Second, "concurrent to reach cap", func() bool {
		return concurrent.Load() == cap
	})

	for i := 0; i < jobs; i++ {
		release <- struct{}{}
		<-completed
	}

	if got := maxConcurrent.Load(); got != cap {
		t.Errorf("max concurrent startups = %d, want %d", got, cap)
	}
}

// =============================================================================
// SLOT-RELEASE SAFETY
// =============================================================================

func TestDispatcher_DefaultReleaseOnEarlyReturn(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 4})
	defer d.Close()

	first := make(chan struct{})
	second := make(chan struct{})

	// First job returns WITHOUT calling started — the slot must still
	// be released so the second job can run.
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		close(first)
	})); err != nil {
		t.Fatal(err)
	}

	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		started()
		close(second)
	})); err != nil {
		t.Fatal(err)
	}

	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("first job did not run")
	}
	select {
	case <-second:
	case <-time.After(2 * time.Second):
		t.Fatal("second job blocked — defer-release safety net failed")
	}
}

func TestDispatcher_DefaultReleaseOnPanic(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 4})
	defer d.Close()

	first := make(chan struct{})
	second := make(chan struct{})

	// Panic inside ExecuteFn must not leak the starting slot. We catch
	// it inside the test ExecuteFn (the dispatcher does not recover —
	// matching production where the runJob goroutine panic crashes the
	// process; the Close defer here keeps the test process alive).
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		defer func() {
			_ = recover()
			close(first)
		}()
		panic("simulated launcher failure")
	})); err != nil {
		t.Fatal(err)
	}

	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		started()
		close(second)
	})); err != nil {
		t.Fatal(err)
	}

	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("first job did not run")
	}
	select {
	case <-second:
	case <-time.After(2 * time.Second):
		t.Fatal("second job blocked after panic — defer-release safety net failed")
	}
}

func TestDispatcher_StartedIsIdempotent(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 4})
	defer d.Close()

	done := make(chan struct{})
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		started()
		started()
		started()
		close(done)
	})); err != nil {
		t.Fatal(err)
	}

	<-done
	awaitTrue(t, time.Second, "Stats() to drain", func() bool {
		s := d.Stats()
		return s.StartingCount == 0 && s.ActiveCount == 0
	})
}

// =============================================================================
// QUEUE / CAPACITY
// =============================================================================

func TestDispatcher_RejectsWhenQueueFull(t *testing.T) {
	t.Parallel()

	const queueCap = 2
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: queueCap})
	defer d.Close()

	// Block the worker on one job so queued ones accumulate.
	gate := make(chan struct{})
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		<-gate
		started()
	})); err != nil {
		t.Fatal(err)
	}

	awaitTrue(t, time.Second, "first job to occupy slot", func() bool {
		return d.Stats().StartingCount == 1
	})

	// Fill the queue.
	noop := func(started StartedFn) { started() }
	for i := 0; i < queueCap; i++ {
		if err := d.Enqueue(makeJob(t, nil, noop)); err != nil {
			t.Fatalf("queued enqueue %d failed: %v", i, err)
		}
	}

	// Next enqueue must reject with the canonical capacity error.
	err := d.Enqueue(makeJob(t, nil, noop))
	if err == nil {
		t.Fatal("expected CapacityExceededError when queue full")
	}
	var capErr *domain.CapacityExceededError
	if !errors.As(err, &capErr) {
		t.Fatalf("expected *CapacityExceededError, got %T: %v", err, err)
	}
	if capErr.Resource != "spawn_queue" {
		t.Errorf("Resource = %q, want spawn_queue", capErr.Resource)
	}

	// Counters must NOT have moved on the rejection.
	if got := d.Stats().QueueDepth; got != queueCap {
		t.Errorf("QueueDepth after rejection = %d, want %d", got, queueCap)
	}

	// Drain.
	close(gate)
	awaitTrue(t, 2*time.Second, "queue to drain", func() bool {
		s := d.Stats()
		return s.QueueDepth == 0 && s.ActiveCount == 0
	})
}

// =============================================================================
// SPACING
// =============================================================================

func TestDispatcher_AppliesMinSpacing(t *testing.T) {
	t.Parallel()

	const spacing = 80 * time.Millisecond
	d := New(Config{MaxStartingConcurrency: 1, MinSpacing: spacing, QueueCapacity: 8})
	defer d.Close()

	const jobs = 3
	starts := make(chan time.Time, jobs)

	for i := 0; i < jobs; i++ {
		if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
			starts <- time.Now()
			started()
		})); err != nil {
			t.Fatal(err)
		}
	}

	collected := make([]time.Time, 0, jobs)
	for i := 0; i < jobs; i++ {
		select {
		case ts := <-starts:
			collected = append(collected, ts)
		case <-time.After(2 * time.Second):
			t.Fatalf("job %d did not start", i)
		}
	}

	// Allow a small scheduler-jitter slack on the lower bound.
	const slack = 15 * time.Millisecond
	for i := 1; i < len(collected); i++ {
		gap := collected[i].Sub(collected[i-1])
		if gap+slack < spacing {
			t.Errorf("gap between job %d and %d = %v, want >= %v", i-1, i, gap, spacing)
		}
	}
}

// =============================================================================
// SHUTDOWN
// =============================================================================

func TestDispatcher_CloseRejectsFurtherEnqueues(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 4})
	d.Close()

	err := d.Enqueue(makeJob(t, nil, func(started StartedFn) { started() }))
	if !errors.Is(err, ErrDispatcherClosed) {
		t.Fatalf("expected ErrDispatcherClosed, got %v", err)
	}

	// Stats after Close must be zero.
	s := d.Stats()
	if s.QueueDepth != 0 || s.ActiveCount != 0 || s.StartingCount != 0 {
		t.Errorf("Stats after Close = %+v, want zero", s)
	}
}

func TestDispatcher_CloseDrainsPendingCounters(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 8})

	// Hold the worker on one job.
	gate := make(chan struct{})
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		<-gate
		started()
	})); err != nil {
		t.Fatal(err)
	}

	awaitTrue(t, time.Second, "first job to occupy slot", func() bool {
		return d.Stats().StartingCount == 1
	})

	// Stack pending jobs in the queue.
	noop := func(started StartedFn) { started() }
	for i := 0; i < 3; i++ {
		if err := d.Enqueue(makeJob(t, nil, noop)); err != nil {
			t.Fatal(err)
		}
	}

	// Release the running job and Close in parallel.
	close(gate)
	d.Close()

	s := d.Stats()
	if s.QueueDepth != 0 {
		t.Errorf("QueueDepth after Close = %d, want 0", s.QueueDepth)
	}
}

func TestAdmissionCancellationBeforeDispatchPreventsExecutorStart(t *testing.T) {
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 2})
	defer d.Close()
	blockerEntered := make(chan struct{})
	releaseBlocker := make(chan struct{})
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		started()
		close(blockerEntered)
		<-releaseBlocker
	})); err != nil {
		t.Fatalf("enqueue blocker: %v", err)
	}
	<-blockerEntered

	runID := uuid.New()
	admission, err := d.ReserveAdmission(runID)
	if err != nil {
		t.Fatalf("reserve admission: %v", err)
	}
	executed := make(chan struct{}, 1)
	if err := d.EnqueueAdmitted(&Job{RunID: runID, Fn: func(StartedFn) { executed <- struct{}{} }}, admission); err != nil {
		t.Fatalf("enqueue admitted: %v", err)
	}
	if admitted, mayHaveStarted := d.PrepareCancelAdmission(runID); admitted != admission || mayHaveStarted {
		t.Fatalf("queued admission prepare cancel = (%p, %t), want (%p, false)", admitted, mayHaveStarted, admission)
	}
	if mayHaveStarted := admission.CommitCancel(); mayHaveStarted {
		t.Fatal("queued admission commit cancel reported an effect already started")
	}
	close(releaseBlocker)

	awaitTrue(t, time.Second, "dispatcher counters drain", func() bool {
		s := d.Stats()
		return s.QueueDepth == 0 && s.ActiveCount == 0 && s.StartingCount == 0
	})
	select {
	case <-executed:
		t.Fatal("executor ran after cancellation won the admission gate")
	default:
	}
	// The worker discarded this job without entering its Fn, so no executor
	// defer could release the reservation. The same RunID can be reserved only
	// after that discard cleanup has removed the old admission.
	reused, err := d.ReserveAdmission(runID)
	if err != nil {
		t.Fatalf("canceled queued admission leaked its reservation: %v", err)
	}
	d.ReleaseAdmission(runID, reused)
}

func TestAdmissionCancellationAfterExecutorClaimCancelsContext(t *testing.T) {
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	defer d.Close()
	runID := uuid.New()
	admission, err := d.ReserveAdmission(runID)
	if err != nil {
		t.Fatalf("reserve admission: %v", err)
	}
	defer d.ReleaseAdmission(runID, admission)
	entered := make(chan struct{})
	finish := make(chan struct{})
	if err := d.EnqueueAdmitted(&Job{RunID: runID, Fn: func(started StartedFn) {
		started()
		close(entered)
		<-finish
	}}, admission); err != nil {
		t.Fatalf("enqueue admitted: %v", err)
	}
	<-entered
	if !d.CancelAdmission(runID) {
		t.Fatal("claimed executor was not reported as possibly active")
	}
	select {
	case <-admission.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("cancellation did not reach the admitted context")
	}
	close(finish)
	awaitTrue(t, time.Second, "dispatcher counters drain", func() bool {
		s := d.Stats()
		return s.ActiveCount == 0 && s.StartingCount == 0
	})
}

func TestDispatcherCloseReleasesQueuedAdmission(t *testing.T) {
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 2})
	gate := make(chan struct{})
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		<-gate
		started()
	})); err != nil {
		t.Fatalf("enqueue blocker: %v", err)
	}
	awaitTrue(t, time.Second, "first job to occupy slot", func() bool {
		return d.Stats().StartingCount == 1
	})
	runID := uuid.New()
	admission, err := d.ReserveAdmission(runID)
	if err != nil {
		t.Fatalf("reserve admission: %v", err)
	}
	if err := d.EnqueueAdmitted(&Job{RunID: runID, Fn: func(StartedFn) {
		t.Error("queued admitted job executed after dispatcher shutdown")
	}}, admission); err != nil {
		t.Fatalf("enqueue admitted: %v", err)
	}
	closed := make(chan struct{})
	go func() {
		d.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not close while an executor was in flight")
	}
	select {
	case <-admission.Context().Done():
	default:
		t.Fatal("shutdown discard did not cancel the queued admission context")
	}
	close(gate)
	awaitTrue(t, time.Second, "in-flight job to finish after close", func() bool {
		return d.Stats().ActiveCount == 0
	})
}

func TestDispatcherConcurrentCloseAndEnqueueDoesNotStrandAdmissions(t *testing.T) {
	const attempts = 32
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: attempts + 1})
	defer d.Close()
	blockerGate := make(chan struct{})
	var releaseBlocker sync.Once
	defer releaseBlocker.Do(func() { close(blockerGate) })
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		<-blockerGate
		started()
	})); err != nil {
		t.Fatalf("enqueue blocker: %v", err)
	}
	awaitTrue(t, time.Second, "blocker to hold the only starting slot", func() bool {
		return d.Stats().StartingCount == 1
	})

	type result struct {
		runID     uuid.UUID
		admission *Admission
		err       error
	}
	preRunID := uuid.New()
	preAdmission, err := d.ReserveAdmission(preRunID)
	if err != nil {
		t.Fatalf("reserve pre-close admission: %v", err)
	}
	if err := d.EnqueueAdmitted(&Job{RunID: preRunID, Fn: func(StartedFn) {
		t.Error("queued job executed after concurrent shutdown")
	}}, preAdmission); err != nil {
		t.Fatalf("enqueue pre-close admission: %v", err)
	}
	results := make(chan result, attempts)
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		runID := uuid.New()
		admission, err := d.ReserveAdmission(runID)
		if err != nil {
			t.Fatalf("reserve admission %d: %v", i, err)
		}
		go func(admission *Admission, runID uuid.UUID) {
			<-start
			err := d.EnqueueAdmitted(&Job{RunID: runID, Fn: func(StartedFn) {
				t.Error("queued job executed after concurrent shutdown")
			}}, admission)
			results <- result{runID: runID, admission: admission, err: err}
		}(admission, runID)
	}
	closed := make(chan struct{})
	go func() {
		<-start
		d.Close()
		close(closed)
	}()
	close(start)

	accepted := 1 // The pre-close item guarantees shutdown has queued work to drain.
	collected := []result{{runID: preRunID, admission: preAdmission}}
	for i := 0; i < attempts; i++ {
		select {
		case got := <-results:
			collected = append(collected, got)
			if got.err == nil {
				accepted++
			} else if errors.Is(got.err, ErrDispatcherClosed) {
				d.ReleaseAdmission(got.runID, got.admission)
			} else {
				t.Fatalf("enqueue returned unexpected error: %v", got.err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("concurrent enqueue did not return")
		}
	}
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("dispatcher did not close")
	}
	releaseBlocker.Do(func() { close(blockerGate) })
	awaitTrue(t, time.Second, "dispatcher counters and admissions to drain", func() bool {
		s := d.Stats()
		return s.QueueDepth == 0 && s.ActiveCount == 0 && s.StartingCount == 0
	})
	if accepted == 0 {
		t.Fatal("concurrent enqueue race did not exercise an accepted job")
	}
	for _, got := range collected {
		select {
		case <-got.admission.Context().Done():
		default:
			t.Fatalf("admission %s remained live after enqueue result %v", got.runID, got.err)
		}
	}
	d.admissionMu.Lock()
	defer d.admissionMu.Unlock()
	if len(d.admissions) != 0 {
		t.Fatalf("%d admission reservations remained after close", len(d.admissions))
	}
}

func TestDispatcherCloseSerializesWithEnqueueBeforeQueueSend(t *testing.T) {
	clockArmed := atomic.Bool{}
	clockEntered := make(chan struct{})
	releaseClock := make(chan struct{})
	var releaseClockOnce sync.Once
	clock := func() time.Time {
		if clockArmed.CompareAndSwap(true, false) {
			close(clockEntered)
			<-releaseClock
		}
		return time.Now()
	}
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 2, Clock: clock})
	defer d.Close()
	defer releaseClockOnce.Do(func() { close(releaseClock) })

	blockerEntered := make(chan struct{})
	blockerRelease := make(chan struct{})
	var releaseBlockerOnce sync.Once
	defer releaseBlockerOnce.Do(func() { close(blockerRelease) })
	if err := d.Enqueue(makeJob(t, nil, func(started StartedFn) {
		close(blockerEntered)
		<-blockerRelease
		started()
	})); err != nil {
		t.Fatalf("enqueue blocker: %v", err)
	}
	select {
	case <-blockerEntered:
	case <-time.After(time.Second):
		t.Fatal("blocker did not enter")
	}

	runID := uuid.New()
	admission, err := d.ReserveAdmission(runID)
	if err != nil {
		t.Fatalf("reserve admission: %v", err)
	}
	clockArmed.Store(true)
	enqueueResult := make(chan error, 1)
	go func() {
		enqueueResult <- d.EnqueueAdmitted(&Job{RunID: runID, Fn: func(StartedFn) {
			t.Error("accepted job ran after Close")
		}}, admission)
	}()
	select {
	case <-clockEntered:
	case <-time.After(time.Second):
		t.Fatal("enqueue did not reach the controlled pre-send clock boundary")
	}
	if d.admissionMu.TryLock() {
		d.admissionMu.Unlock()
		t.Fatal("enqueue did not hold the dispatcher acceptance lock across its pre-send boundary")
	}

	closeStarted := make(chan struct{})
	closeDone := make(chan struct{})
	go func() {
		close(closeStarted)
		d.Close()
		close(closeDone)
	}()
	<-closeStarted
	select {
	case <-closeDone:
		t.Fatal("Close returned while enqueue was inside its acceptance critical section")
	default:
	}
	releaseClockOnce.Do(func() { close(releaseClock) })
	select {
	case err := <-enqueueResult:
		if err != nil {
			t.Fatalf("enqueue that acquired the acceptance lock was rejected: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("enqueue did not finish after releasing the controlled clock")
	}
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("Close did not finish after enqueue released the acceptance lock")
	}
	releaseBlockerOnce.Do(func() { close(blockerRelease) })
	awaitTrue(t, time.Second, "dispatcher counters to drain", func() bool {
		stats := d.Stats()
		return stats.QueueDepth == 0 && stats.ActiveCount == 0 && stats.StartingCount == 0
	})
	select {
	case <-admission.Context().Done():
	default:
		t.Fatal("accepted queued admission was not released during shutdown")
	}
	d.admissionMu.Lock()
	defer d.admissionMu.Unlock()
	if len(d.admissions) != 0 {
		t.Fatalf("%d admission reservations remained after close", len(d.admissions))
	}
}

func TestDispatcher_RecoversPanickingJob(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	defer d.Close()

	panics := make(chan string, 1)
	job := makeJob(t, nil, func(StartedFn) { panic("boom") })
	job.OnPanic = func(failure obs.PanicFailure) {
		panics <- failure.Error()
	}
	if err := d.Enqueue(job); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-panics:
		if got != "panic in spawn dispatcher job: boom" {
			t.Fatalf("panic failure = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not report recovered panic")
	}

	awaitTrue(t, time.Second, "dispatcher counters to release", func() bool {
		stats := d.Stats()
		return stats.ActiveCount == 0 && stats.StartingCount == 0
	})
}

// panicOnSecondEmitSink lets the request-path EmitSpawnEnqueued succeed, then
// panics on the worker-path EmitSpawnStarted.
type panicOnSecondEmitSink struct{ calls atomic.Int32 }

func (s *panicOnSecondEmitSink) Emit(*domain.RunEvent) error {
	if s.calls.Add(1) == 2 {
		panic("simulated sink defect")
	}
	return nil
}

// A sink panic between slot acquisition and runJob must be contained without
// leaking the starting slot: the panicking job still executes and subsequent
// jobs still flow through the (sole) spawn path.
func TestDispatcher_WorkerSurvivesSinkPanic(t *testing.T) {
	t.Parallel()

	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 2})
	defer d.Close()

	badExecuted := make(chan struct{})
	bad := makeJob(t, nil, func(started StartedFn) { started(); close(badExecuted) })
	bad.Sink = &panicOnSecondEmitSink{}
	if err := d.Enqueue(bad); err != nil {
		t.Fatal(err)
	}

	goodExecuted := make(chan struct{})
	good := makeJob(t, nil, func(started StartedFn) { started(); close(goodExecuted) })
	if err := d.Enqueue(good); err != nil {
		t.Fatal(err)
	}

	select {
	case <-badExecuted:
	case <-time.After(2 * time.Second):
		t.Fatal("panicking-sink job never executed")
	}
	select {
	case <-goodExecuted:
	case <-time.After(2 * time.Second):
		t.Fatal("subsequent job never executed — spawn path wedged after contained sink panic")
	}
}
