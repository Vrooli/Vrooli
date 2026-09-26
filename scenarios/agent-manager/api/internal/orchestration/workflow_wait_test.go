package orchestration

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// waitFakeExecRepo is a minimal WorkflowExecutionRepository that only serves
// Get (the sole method WaitWorkflowExecution touches). Every mutating method is
// left to the embedded nil interface so a call the wait path must NOT make
// panics loudly — proving the waiter never mutates execution state.
type waitFakeExecRepo struct {
	repository.WorkflowExecutionRepository
	mu      sync.Mutex
	exec    *domain.WorkflowExecution
	getHits int
}

func (r *waitFakeExecRepo) set(status domain.WorkflowExecutionStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exec.Status = status
}

func (r *waitFakeExecRepo) status() domain.WorkflowExecutionStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.exec.Status
}

func (r *waitFakeExecRepo) Get(_ context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getHits++
	if r.exec == nil || r.exec.ID != id {
		return nil, nil
	}
	snapshot := *r.exec
	return &snapshot, nil
}

func newWaitHarness(status domain.WorkflowExecutionStatus) (*Orchestrator, *waitFakeExecRepo, uuid.UUID) {
	id := uuid.New()
	repo := &waitFakeExecRepo{exec: &domain.WorkflowExecution{ID: id, Status: status}}
	o := New(nil, nil, nil, WithWorkflowExecutionRepository(repo))
	return o, repo, id
}

func TestWaitWorkflowExecutionReturnsImmediatelyForTerminal(t *testing.T) {
	o, _, id := newWaitHarness(domain.WorkflowExecutionSucceeded)
	res, err := o.WaitWorkflowExecution(context.Background(), id, 0)
	if err != nil || res == nil || res.TimedOut || res.Execution.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("terminal wait = %+v err=%v", res, err)
	}
}

func TestWaitWorkflowExecutionBlocksUntilTerminal(t *testing.T) {
	o, repo, id := newWaitHarness(domain.WorkflowExecutionRunning)
	type outcome struct {
		res *WaitWorkflowExecutionResult
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		res, err := o.WaitWorkflowExecution(context.Background(), id, 0)
		ch <- outcome{res, err}
	}()

	awaitWorkflowWaiters(t, o, id, 1)
	select {
	case got := <-ch:
		t.Fatalf("wait returned before terminal: %+v", got)
	default:
	}

	repo.set(domain.WorkflowExecutionSucceeded)
	o.onWorkflowExecutionSettled(&domain.WorkflowExecution{ID: id, Status: domain.WorkflowExecutionSucceeded})

	select {
	case got := <-ch:
		if got.err != nil || got.res.TimedOut || got.res.Execution.Status != domain.WorkflowExecutionSucceeded {
			t.Fatalf("woken wait = %+v err=%v", got.res, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not wake after terminal notify")
	}
}

func TestWaitWorkflowExecutionRespectsDeadline(t *testing.T) {
	o, repo, id := newWaitHarness(domain.WorkflowExecutionRunning)
	start := time.Now()
	res, err := o.WaitWorkflowExecution(context.Background(), id, 150*time.Millisecond)
	if err != nil {
		t.Fatalf("deadline wait err = %v", err)
	}
	if !res.TimedOut {
		t.Fatalf("expected timed out, got %+v", res)
	}
	if time.Since(start) < 150*time.Millisecond {
		t.Fatalf("returned before deadline")
	}
	// The execution is untouched by the timed-out waiter.
	if repo.status() != domain.WorkflowExecutionRunning {
		t.Fatalf("execution status mutated by timed-out wait: %s", repo.status())
	}
}

func TestWaitWorkflowExecutionSurvivesConcurrentWaiters(t *testing.T) {
	o, repo, id := newWaitHarness(domain.WorkflowExecutionRunning)
	const waiters = 12
	var wg sync.WaitGroup
	errs := make(chan error, waiters)
	for range waiters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := o.WaitWorkflowExecution(context.Background(), id, 0)
			if err != nil {
				errs <- err
				return
			}
			if res.Execution.Status != domain.WorkflowExecutionFailed {
				errs <- errors.New("waiter observed non-terminal status")
			}
		}()
	}
	awaitWorkflowWaiters(t, o, id, waiters)
	repo.set(domain.WorkflowExecutionFailed)
	o.onWorkflowExecutionSettled(&domain.WorkflowExecution{ID: id, Status: domain.WorkflowExecutionFailed})

	waitDone := make(chan struct{})
	go func() { wg.Wait(); close(waitDone) }()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent waiters did not all wake")
	}
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent waiter error: %v", err)
	}
}

// TestWaitWorkflowExecutionCancelDoesNotCancelExecution pins the core contract
// invariant: cancelling the waiter's context returns promptly with the ctx
// error and never touches the execution — the fake repo panics on any mutating
// call, so a clean return proves no cancellation was propagated.
func TestWaitWorkflowExecutionCancelDoesNotCancelExecution(t *testing.T) {
	o, repo, id := newWaitHarness(domain.WorkflowExecutionRunning)
	ctx, cancel := context.WithCancel(context.Background())
	type outcome struct {
		res *WaitWorkflowExecutionResult
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		res, err := o.WaitWorkflowExecution(ctx, id, 0)
		ch <- outcome{res, err}
	}()
	awaitWorkflowWaiters(t, o, id, 1)
	cancel()

	select {
	case got := <-ch:
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("cancelled wait err = %v, want context.Canceled", got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled wait did not return")
	}
	if repo.status() != domain.WorkflowExecutionRunning {
		t.Fatalf("execution status changed after waiter cancel: %s", repo.status())
	}
}

func awaitWorkflowWaiters(t *testing.T, o *Orchestrator, id uuid.UUID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if o.workflowWaiters.count(id) == want {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("waiter subscriptions = %d, want %d", o.workflowWaiters.count(id), want)
}
