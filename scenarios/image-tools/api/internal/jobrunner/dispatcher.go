// Package jobrunner is the seam between the generic durable-job Manager
// (internal/jobs) and the per-operation execution logic that lands with each
// image-operation domain. The Manager runs ONE Runner; this Dispatcher fans
// that single entry point out to per-operation handlers keyed by job.Operation.
//
// Operation domains (generate/enhance/analyze, …) register their handler here at
// boot; until they do, a submitted job for an unregistered operation fails
// cleanly with ErrNoRunner rather than hanging. In Phase 1 no operations are
// registered yet — the job system is durable and observable, but nothing can be
// executed, which matches "boots healthy with zero image ops".
package jobrunner

import (
	"context"
	"errors"
	"fmt"
	"sync"

	internaljobs "image-tools/internal/jobs"
)

// ErrNoRunner is returned when a job is submitted for an operation that has no
// registered handler.
var ErrNoRunner = errors.New("jobrunner: no handler registered for operation")

// OpRunner executes one operation's work. It matches the internal/jobs.Runner
// shape so a handler can be the body of a job directly.
type OpRunner func(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error)

// Dispatcher routes a job to the handler registered for its operation.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]OpRunner
}

// New returns an empty Dispatcher.
func New() *Dispatcher {
	return &Dispatcher{handlers: make(map[string]OpRunner)}
}

// Register binds an operation to its handler. A duplicate registration for the
// same operation is a wiring bug and panics at boot.
func (d *Dispatcher) Register(operation string, run OpRunner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, dup := d.handlers[operation]; dup {
		panic(fmt.Sprintf("jobrunner: duplicate handler for operation %q", operation))
	}
	d.handlers[operation] = run
}

// Operations returns the registered operation names (unordered).
func (d *Dispatcher) Operations() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ops := make([]string, 0, len(d.handlers))
	for op := range d.handlers {
		ops = append(ops, op)
	}
	return ops
}

// Run implements internal/jobs.Runner: it looks up the handler for the job's
// operation and executes it, or returns ErrNoRunner.
func (d *Dispatcher) Run(ctx context.Context, job internaljobs.Job, emit func(progress int, message string)) (internaljobs.Result, error) {
	d.mu.RLock()
	run, ok := d.handlers[job.Operation]
	d.mu.RUnlock()
	if !ok {
		return internaljobs.Result{}, fmt.Errorf("%w: %q", ErrNoRunner, job.Operation)
	}
	return run(ctx, job, emit)
}
