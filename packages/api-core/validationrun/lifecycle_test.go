package validationrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTransitionMatrix(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		state State
		event Event
		want  State
		err   ErrorCode
	}{
		{"claim queued", StateQueued, EventClaim, StateRunning, ""},
		{"succeed running", StateRunning, EventSucceed, StateSucceeded, ""},
		{"fail queued", StateQueued, EventFail, StateFailed, ""},
		{"cancel queued", StateQueued, EventCancel, StateCanceled, ""},
		{"recovery failure running", StateRunning, EventRecoveryFailed, StateRecoveryFailed, ""},
		{"reject succeed queued", StateQueued, EventSucceed, "", ErrorInvalidTransition},
		{"reject claim running", StateRunning, EventClaim, "", ErrorInvalidTransition},
		{"reject terminal", StateSucceeded, EventFail, "", ErrorInvalidTransition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Transition(testRun(tc.state), tc.event, now)
			if tc.err != "" {
				if !IsCode(err, tc.err) {
					t.Fatalf("Transition error = %v, want %s", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Transition: %v", err)
			}
			if got.State != tc.want {
				t.Fatalf("state = %s, want %s", got.State, tc.want)
			}
			if got.State.Terminal() && !got.CompletedAt.Equal(now) {
				t.Fatalf("completed at = %v, want %v", got.CompletedAt, now)
			}
		})
	}
}

func TestStartOrGetIsReplaySafeAndConcurrent(t *testing.T) {
	repo := newMemoryRepository()
	exec := &recordingExecutor{}
	coord := Coordinator{Repository: repo, Executor: exec, IDs: sequenceIDs{}, Clock: fixedClock{}}
	target := Target{Scenario: "demo"}
	first, created, err := coord.StartOrGet(context.Background(), target, "request-1", "parent-1")
	if err != nil || !created {
		t.Fatalf("first start = (%+v, %v, %v), want created", first, created, err)
	}
	second, created, err := coord.StartOrGet(context.Background(), target, "request-1", "parent-1")
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("replay = (%+v, %v, %v), want same existing run", second, created, err)
	}
	if _, _, err := coord.StartOrGet(context.Background(), Target{Scenario: "other"}, "request-1", "parent-1"); !IsCode(err, ErrorIdempotencyConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = coord.StartOrGet(context.Background(), target, "request-2", "parent-2")
		}()
	}
	wg.Wait()
	if got := repo.count(); got != 2 {
		t.Fatalf("persisted run count = %d, want 2", got)
	}
	if got := exec.calls(); got != 2 {
		t.Fatalf("executor calls = %d, want 2", got)
	}
}

func TestStartSurvivesCallerCancellation(t *testing.T) {
	repo := newMemoryRepository()
	exec := &recordingExecutor{started: make(chan struct{})}
	coord := Coordinator{Repository: repo, Executor: exec, IDs: sequenceIDs{}, Clock: fixedClock{}}
	ctx, cancel := context.WithCancel(context.Background())
	_, created, err := coord.StartOrGet(ctx, Target{Scenario: "demo"}, "request", "")
	cancel()
	if err != nil || !created {
		t.Fatalf("StartOrGet = (%v, %v), want created", created, err)
	}
	select {
	case <-exec.started:
	case <-time.After(time.Second):
		t.Fatal("executor did not outlive canceled caller")
	}
}

func TestAbortAndStaleCompletion(t *testing.T) {
	repo := newMemoryRepository()
	run := testRun(StateRunning)
	run.Version = 4
	repo.put(run)
	exec := &recordingExecutor{}
	coord := Coordinator{Repository: repo, Executor: exec, IDs: sequenceIDs{}, Clock: fixedClock{}}
	aborting, err := coord.Abort(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if !aborting.CancellationRequested || aborting.State != StateRunning {
		t.Fatalf("abort record = %+v", aborting)
	}
	if exec.aborts() != 1 {
		t.Fatalf("abort calls = %d, want 1", exec.aborts())
	}
	if _, err := coord.CommitTerminal(context.Background(), run.ID, EventCancel); err != nil {
		t.Fatalf("commit cancel: %v", err)
	}
	if _, err := coord.CommitTerminal(context.Background(), run.ID, EventSucceed); !IsCode(err, ErrorInvalidTransition) {
		t.Fatalf("stale success = %v, want invalid transition", err)
	}
}

func TestWaitBlocksForNotificationAndDoesNotAbortWork(t *testing.T) {
	repo := newMemoryRepository()
	run := testRun(StateRunning)
	repo.put(run)
	notifier := newMemoryNotifier()
	coord := Coordinator{Repository: repo, Notifier: notifier, IDs: sequenceIDs{}, Clock: fixedClock{}}
	go func() {
		time.Sleep(10 * time.Millisecond)
		current, _ := repo.Get(context.Background(), run.ID)
		next, _ := Transition(current, EventSucceed, fixedClock{}.Now())
		next.Version = current.Version + 1
		_ = repo.Update(context.Background(), next, current.Version)
		notifier.notify()
	}()
	got, err := coord.Wait(context.Background(), run.ID, time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got.State != StateSucceeded {
		t.Fatalf("wait state = %s, want succeeded", got.State)
	}
	if _, err := coord.Wait(context.Background(), "missing", time.Millisecond); !IsCode(err, ErrorNotFound) {
		t.Fatalf("missing wait error = %v", err)
	}
}

func TestWaitTimeoutIsTyped(t *testing.T) {
	repo := newMemoryRepository()
	repo.put(testRun(StateRunning))
	coord := Coordinator{Repository: repo, Notifier: newMemoryNotifier(), IDs: sequenceIDs{}, Clock: fixedClock{}}
	_, err := coord.Wait(context.Background(), "run-1", time.Millisecond)
	if !IsCode(err, ErrorWaitTimeout) {
		t.Fatalf("Wait error = %v, want typed timeout", err)
	}
}

func TestProviderPolicyDoesNotLeakIntoLifecyclePackage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	for _, name := range []string{"types.go", "lifecycle.go"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, forbidden := range []string{"workflow-health", "test-genie", "sqlite", "browser-automation", "cli-core"} {
			if strings.Contains(strings.ToLower(string(data)), forbidden) {
				t.Fatalf("%s must not contain provider policy reference %q", name, forbidden)
			}
		}
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC) }

type sequenceIDs struct{}

func (sequenceIDs) NewID() string { return "run-" + time.Now().UTC().Format("150405.000000000") }

func testRun(state State) Run {
	return Run{ID: "run-1", Target: Target{Scenario: "demo"}, IdempotencyKey: "key", State: state, CreatedAt: fixedClock{}.Now(), Version: 1}
}

type memoryRepository struct {
	mu    sync.Mutex
	byID  map[string]Run
	byKey map[string]string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{byID: map[string]Run{}, byKey: map[string]string{}}
}

func (r *memoryRepository) FindByIdempotency(_ context.Context, key string) (Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byKey[key]
	if !ok {
		return Run{}, &LifecycleError{Code: ErrorNotFound, Operation: "find"}
	}
	return r.byID[id], nil
}

func (r *memoryRepository) Get(_ context.Context, id string) (Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.byID[id]
	if !ok {
		return Run{}, &LifecycleError{Code: ErrorNotFound, Operation: "get"}
	}
	return run, nil
}

func (r *memoryRepository) Create(_ context.Context, run Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byKey[run.IdempotencyKey]; exists {
		return errors.New("unique key")
	}
	r.byID[run.ID] = run
	r.byKey[run.IdempotencyKey] = run.ID
	return nil
}

func (r *memoryRepository) Update(_ context.Context, run Run, expected int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.byID[run.ID]
	if !ok {
		return &LifecycleError{Code: ErrorNotFound, Operation: "update"}
	}
	if current.Version != expected {
		return errors.New("stale version")
	}
	r.byID[run.ID] = run
	return nil
}

func (r *memoryRepository) put(run Run) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[run.ID] = run
	r.byKey[run.IdempotencyKey] = run.ID
}
func (r *memoryRepository) count() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.byID) }

type recordingExecutor struct {
	mu                   sync.Mutex
	runCalls, abortCalls int
	started              chan struct{}
}

func (e *recordingExecutor) Run(_ context.Context, _ string) {
	e.mu.Lock()
	e.runCalls++
	e.mu.Unlock()
	if e.started != nil {
		select {
		case e.started <- struct{}{}:
		default:
		}
	}
}

func (e *recordingExecutor) Abort(_ context.Context, _ string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.abortCalls++
	return nil
}
func (e *recordingExecutor) calls() int  { e.mu.Lock(); defer e.mu.Unlock(); return e.runCalls }
func (e *recordingExecutor) aborts() int { e.mu.Lock(); defer e.mu.Unlock(); return e.abortCalls }

type memoryNotifier struct{ ch chan struct{} }

func newMemoryNotifier() *memoryNotifier { return &memoryNotifier{ch: make(chan struct{}, 1)} }
func (n *memoryNotifier) WaitForChange(ctx context.Context, _ string, _ int64) error {
	select {
	case <-n.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (n *memoryNotifier) notify() { n.ch <- struct{}{} }
