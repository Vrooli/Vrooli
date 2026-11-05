package runs

import (
	"context"

	"data-backup-manager/internal/asyncpool"
)

// RunJob is the unit of background backup work. The plan id and trigger travel
// with it so the worker can execute the run without re-reading the row.
type RunJob struct {
	RunID   string
	PlanID  string
	Trigger TriggerSource
}

// RunFunc executes one run to a terminal state.
type RunFunc = asyncpool.JobFunc[RunJob]

// Executor schedules backup runs onto background workers.
type Executor interface {
	Bind(baseCtx context.Context, run RunFunc)
	Submit(job RunJob)
	Shutdown(ctx context.Context) error
}

// AsyncExecutor is the production backup worker pool.
type AsyncExecutor = asyncpool.Pool[RunJob]

// NewAsyncExecutor constructs the production executor.
func NewAsyncExecutor(workers int) *AsyncExecutor {
	return asyncpool.New[RunJob](workers, 256)
}

var _ Executor = (*AsyncExecutor)(nil)
