package runs

import (
	"context"
	"sync"
)

// RunJob is the unit of background backup work: execute an already-created
// (pending) run to its terminal state. The plan id and trigger travel with it
// so the worker can resolve and execute the run without re-reading the row.
type RunJob struct {
	RunID   string
	PlanID  string
	Trigger TriggerSource
}

// RunFunc executes one run to a terminal state. It is bound to the executor by
// the service (it is the service's executeRun). The ctx is the executor's base
// (server-lifetime) context, NOT a request context — that is the whole point:
// a client disconnect must not cancel an in-flight backup.
type RunFunc func(ctx context.Context, job RunJob)

// Executor schedules backup runs onto background workers so TriggerRun returns
// immediately and the run executes decoupled from the request lifecycle.
//
// seam: production wires AsyncExecutor (a worker pool over a server-lifetime
// context); tests wire a synchronous inline executor so run completion is
// deterministic without polling. There is no synchronous production path — the
// async contract is the only contract.
type Executor interface {
	// Bind registers the run function and the base (server-lifetime) context,
	// and starts the workers. The service calls it once during construction,
	// before any Submit.
	Bind(baseCtx context.Context, run RunFunc)
	// Submit schedules a job. It must not block on the job's completion.
	Submit(job RunJob)
	// Shutdown waits for in-flight jobs to finish or for ctx to cancel.
	Shutdown(ctx context.Context) error
}

// AsyncExecutor is the production Executor: a bounded worker pool draining a
// buffered queue, each worker bound to the server-lifetime context.
type AsyncExecutor struct {
	workers int
	queue   chan RunJob
	run     RunFunc
	baseCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewAsyncExecutor constructs the production executor. workers < 1 is clamped to
// 1. The queue is generously buffered; for a single-user tool it never fills.
func NewAsyncExecutor(workers int) *AsyncExecutor {
	if workers < 1 {
		workers = 1
	}
	return &AsyncExecutor{workers: workers, queue: make(chan RunJob, 256)}
}

// Bind wires the run function and launches the workers. It derives its own
// cancellable context from baseCtx so Shutdown can stop the workers itself —
// the workers stop when EITHER baseCtx (server lifetime) is cancelled OR
// Shutdown is called, so Shutdown is self-sufficient and never deadlocks
// waiting on a context only the caller can cancel.
func (e *AsyncExecutor) Bind(baseCtx context.Context, run RunFunc) {
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
// reconciliation will close its run row on the next boot.
func (e *AsyncExecutor) Submit(job RunJob) {
	select {
	case e.queue <- job:
	case <-e.baseCtx.Done():
	}
}

// Shutdown signals the workers to stop (after finishing the job in flight) and
// waits for them to drain, bounded by ctx. It is self-sufficient — it does not
// rely on the caller cancelling the base context.
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
