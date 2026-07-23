package orchestration

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// Completion-driven advance (Phase 4).
//
// The workflow engine is a crash-safe *pull* loop: driveWorkflowExecution reads
// durable state, calls Advance, and exits at the no-progress version fixpoint
// when a dispatched run is still non-terminal. Nothing re-drives the execution
// on its own — before this nudge, consumers polled Advance on a ticker.
//
// The nudge is a *trigger* for that existing pull loop, never a new scheduler.
// When a run belonging to a workflow attempt reaches a terminal status, the
// orchestrator enqueues an idempotent drive of the parent execution. The drive
// re-reads durable state and is guarded by the engine's optimistic-version CAS,
// so a nudge racing an explicit Advance (or a second nudge) is safe. The
// reconciler recovery sweep (RecoverWorkflowExecutions, run at boot and every
// reconcile cycle) is the durable backstop for a crash between run-terminal and
// enqueue.

// WorkflowNudger is a small deduplicating work queue: at most one drive per
// execution is pending at a time, distinct executions drive concurrently, and
// each drive runs under its own bounded, request-detached context.
type WorkflowNudger struct {
	drive   func(ctx context.Context, id uuid.UUID) error
	onPanic func(context.Context, uuid.UUID, obs.PanicFailure)
	workers int
	timeout time.Duration

	mu      sync.Mutex
	cond    *sync.Cond
	queue   []uuid.UUID
	pending map[uuid.UUID]struct{}
	closed  bool
	wg      sync.WaitGroup
}

// NewWorkflowNudger builds a nudger over the given drive function. workers and
// timeout come from config.Levers.Workflow. The returned nudger is inert until
// Start spawns its worker goroutines.
func NewWorkflowNudger(drive func(ctx context.Context, id uuid.UUID) error, workers int, timeout time.Duration, onPanic ...func(context.Context, uuid.UUID, obs.PanicFailure)) *WorkflowNudger {
	if workers < 1 {
		workers = 1
	}
	n := &WorkflowNudger{
		drive:   drive,
		workers: workers,
		timeout: timeout,
		pending: make(map[uuid.UUID]struct{}),
	}
	if len(onPanic) > 0 {
		n.onPanic = onPanic[0]
	}
	n.cond = sync.NewCond(&n.mu)
	return n
}

func (n *WorkflowNudger) log() *slog.Logger { return obs.Component("workflow-nudge") }

// Start spawns the worker goroutines that drain the queue. Idempotent-safe to
// call once at wiring time.
func (n *WorkflowNudger) Start() {
	if n == nil {
		return
	}
	for range n.workers {
		n.wg.Add(1)
		go n.worker()
	}
}

// Enqueue schedules an idempotent drive of the parent execution. It never
// blocks the caller (the run-terminal path must not stall on it) and dedupes:
// an execution already queued is not queued twice. No-op after Stop or when the
// nudger is nil.
func (n *WorkflowNudger) Enqueue(id uuid.UUID) {
	if n == nil || id == uuid.Nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return
	}
	if _, exists := n.pending[id]; exists {
		return
	}
	n.pending[id] = struct{}{}
	n.queue = append(n.queue, id)
	n.cond.Signal()
}

// Stop drains no further work, wakes all workers, and blocks until they exit.
// Executions still needing progress are re-driven by the reconciler backstop on
// the next cycle, so a drop on shutdown is durable-safe.
func (n *WorkflowNudger) Stop() {
	if n == nil {
		return
	}
	n.mu.Lock()
	n.closed = true
	n.mu.Unlock()
	n.cond.Broadcast()
	n.wg.Wait()
}

func (n *WorkflowNudger) worker() {
	defer n.wg.Done()
	for {
		n.mu.Lock()
		for len(n.queue) == 0 && !n.closed {
			n.cond.Wait()
		}
		if len(n.queue) == 0 && n.closed {
			n.mu.Unlock()
			return
		}
		id := n.queue[0]
		n.queue = n.queue[1:]
		// Clear the pending mark BEFORE driving so a completion that lands
		// during this drive re-queues and triggers another drive rather than
		// being deduped away.
		delete(n.pending, id)
		n.mu.Unlock()

		n.runDrive(id)
	}
}

func (n *WorkflowNudger) runDrive(id uuid.UUID) {
	defer obs.RecoverToFailure("workflow nudger drive", func(failure obs.PanicFailure) {
		n.log().Error("workflow nudge panic recovered", "executionId", id.String(), obs.KeyError, failure.Error(), "stack", failure.Stack)
		if n.onPanic != nil {
			n.onPanic(context.Background(), id, failure)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()
	if err := n.drive(ctx, id); err != nil {
		n.log().Warn("nudge drive failed",
			"executionId", id.String(),
			obs.KeyError, err.Error())
	}
}

// PendingCount reports how many executions are queued for a drive. Exposed for
// observability and tests.
func (n *WorkflowNudger) PendingCount() int {
	if n == nil {
		return 0
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.pending)
}

// workflowWaitRegistry is the event-driven notifier backing WaitWorkflowExecution.
// A waiter subscribes, re-reads durable state to close the register/settle gap,
// then blocks on its channel; the drive path closes every subscriber's channel
// when an execution settles terminal. There is no ticker — a waiter never polls.
type workflowWaitRegistry struct {
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan struct{}]struct{}
}

func newWorkflowWaitRegistry() *workflowWaitRegistry {
	return &workflowWaitRegistry{subs: make(map[uuid.UUID]map[chan struct{}]struct{})}
}

// subscribe registers a fresh wake channel for an execution and returns it. The
// caller must re-read durable state after subscribing (the terminal transition
// may have committed just before this call) and must unsubscribe when done.
func (r *workflowWaitRegistry) subscribe(id uuid.UUID) chan struct{} {
	ch := make(chan struct{})
	r.mu.Lock()
	if r.subs[id] == nil {
		r.subs[id] = make(map[chan struct{}]struct{})
	}
	r.subs[id][ch] = struct{}{}
	r.mu.Unlock()
	return ch
}

// unsubscribe removes a subscriber. Safe if notify already closed and removed
// it (the channel is closed exactly once, only by notify).
func (r *workflowWaitRegistry) unsubscribe(id uuid.UUID, ch chan struct{}) {
	r.mu.Lock()
	if subs := r.subs[id]; subs != nil {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(r.subs, id)
		}
	}
	r.mu.Unlock()
}

// notify wakes every current waiter for an execution by closing its channel.
// Idempotent across repeated terminal observations: the second call finds no
// subscribers.
func (r *workflowWaitRegistry) notify(id uuid.UUID) {
	r.mu.Lock()
	subs := r.subs[id]
	delete(r.subs, id)
	r.mu.Unlock()
	for ch := range subs {
		close(ch)
	}
}

// count returns the current subscriber count for one execution. It is used by
// deterministic waiter tests to synchronize on registration rather than sleep.
func (r *workflowWaitRegistry) count(id uuid.UUID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.subs[id])
}
