package audits

import (
	"context"
	"sync"
)

// AuditJob is the unit of background audit work: drive an already-created
// (requested) audit record to its terminal state. The resolved target and repo
// name travel with it so the worker does not re-resolve anything — the
// synchronous request already validated them.
type AuditJob struct {
	Audit    Audit
	Target   TargetForAudit
	DestName string
}

// AuditFunc executes one audit job to a terminal state, bound to the executor's
// base (server-lifetime) context — NOT a request context, so a client
// disconnect cannot cancel an in-flight audit (mirrors restores).
type AuditFunc func(ctx context.Context, job AuditJob)

// Executor schedules audit jobs onto background workers so the request RPCs
// return immediately and the heavy restore+capture+walk runs decoupled from the
// request lifecycle.
//
// seam: production wires AsyncExecutor (a worker pool over a server-lifetime
// context); tests wire SyncExecutor (inline) so terminal state is observable
// without polling. There is no synchronous production path.
type Executor interface {
	// Bind registers the job function and base context and starts the workers.
	// The service calls it once during construction, before any Submit.
	Bind(baseCtx context.Context, run AuditFunc)
	// Submit schedules a job. It must not block on the job's completion.
	Submit(job AuditJob)
	// Shutdown waits for in-flight jobs to finish or for ctx to cancel.
	Shutdown(ctx context.Context) error
}

// AsyncExecutor is the production Executor: a bounded worker pool draining a
// buffered queue, each worker bound to the server-lifetime context.
type AsyncExecutor struct {
	workers int
	queue   chan AuditJob
	run     AuditFunc
	baseCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewAsyncExecutor constructs the production executor. workers < 1 clamps to 1.
func NewAsyncExecutor(workers int) *AsyncExecutor {
	if workers < 1 {
		workers = 1
	}
	return &AsyncExecutor{workers: workers, queue: make(chan AuditJob, 256)}
}

// Bind wires the job function and launches the workers, deriving its own
// cancellable context so Shutdown can stop them without relying on the caller
// cancelling the base context (mirrors restores.AsyncExecutor).
func (e *AsyncExecutor) Bind(baseCtx context.Context, run AuditFunc) {
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	e.baseCtx, e.cancel = context.WithCancel(baseCtx)
	e.run = run
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.loop()
	}
}

func (e *AsyncExecutor) loop() {
	defer e.wg.Done()
	for {
		select {
		case <-e.baseCtx.Done():
			return
		case job := <-e.queue:
			e.run(e.baseCtx, job)
		}
	}
}

// Submit enqueues a job without blocking on completion. If the server is
// shutting down (base context cancelled) the job is dropped — startup
// reconciliation closes its row on the next boot.
func (e *AsyncExecutor) Submit(job AuditJob) {
	select {
	case e.queue <- job:
	case <-e.baseCtx.Done():
	}
}

// Shutdown signals the workers to stop (after finishing the job in flight) and
// waits for them to drain, bounded by ctx.
func (e *AsyncExecutor) Shutdown(ctx context.Context) error {
	if e.cancel != nil {
		e.cancel()
	}
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Compile-time guarantee.
var _ Executor = (*AsyncExecutor)(nil)
