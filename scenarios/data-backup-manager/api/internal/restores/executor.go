package restores

import (
	"context"

	"data-backup-manager/internal/asyncpool"
)

// RestoreJob is the unit of background restore/verify work. It carries the
// stable destination ID so the worker can re-resolve a removable repository
// immediately before use, even if the volume changed while queued.
type RestoreJob struct {
	Restore       Restore
	Target        TargetForRestore
	DestinationID string
	DestName      string
}

// RestoreFunc executes one restore/verify job to a terminal state.
type RestoreFunc = asyncpool.JobFunc[RestoreJob]

// Executor schedules restore/verify jobs onto background workers.
type Executor interface {
	Bind(baseCtx context.Context, run RestoreFunc)
	Submit(job RestoreJob)
	Shutdown(ctx context.Context) error
}

// AsyncExecutor is the production restore worker pool.
type AsyncExecutor = asyncpool.Pool[RestoreJob]

// NewAsyncExecutor constructs the production executor.
func NewAsyncExecutor(workers int) *AsyncExecutor {
	return asyncpool.New[RestoreJob](workers, 256)
}

var _ Executor = (*AsyncExecutor)(nil)
