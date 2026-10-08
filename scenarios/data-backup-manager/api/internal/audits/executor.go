package audits

import (
	"context"

	"data-backup-manager/internal/asyncpool"
)

// AuditJob is the unit of background audit work: drive an already-created
// audit record to its terminal state. The resolved target and repo name travel
// with it so the worker does not re-resolve anything.
type AuditJob struct {
	Audit    Audit
	Target   TargetForAudit
	DestName string
}

// AuditFunc executes one audit job to a terminal state.
type AuditFunc = asyncpool.JobFunc[AuditJob]

// Executor schedules audit jobs onto background workers.
type Executor interface {
	Bind(baseCtx context.Context, run AuditFunc)
	Submit(job AuditJob)
	Shutdown(ctx context.Context) error
}

// AsyncExecutor is the production audit worker pool.
type AsyncExecutor = asyncpool.Pool[AuditJob]

// NewAsyncExecutor constructs the production executor.
func NewAsyncExecutor(workers int) *AsyncExecutor {
	return asyncpool.New[AuditJob](workers, 256)
}

var _ Executor = (*AsyncExecutor)(nil)
