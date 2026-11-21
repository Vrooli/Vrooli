package mocks

import (
	"context"

	"data-backup-manager/internal/runs"
)

// SyncExecutor is the test executor: it runs each submitted job inline and to
// completion on the caller's goroutine, so a service test can TriggerRun and
// then immediately GetRun the terminal run without polling or sleeping. It
// mirrors the production async contract (Bind once, then Submit) but collapses
// the asynchrony for determinism.
type SyncExecutor struct {
	baseCtx context.Context
	run     runs.RunFunc
}

// NewSyncExecutor constructs an inline executor for tests.
func NewSyncExecutor() *SyncExecutor { return &SyncExecutor{} }

func (e *SyncExecutor) Bind(baseCtx context.Context, run runs.RunFunc) {
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	e.baseCtx = baseCtx
	e.run = run
}

func (e *SyncExecutor) Submit(job runs.RunJob) {
	if e.run != nil {
		e.run(e.baseCtx, job)
	}
}

func (e *SyncExecutor) Shutdown(context.Context) error { return nil }

// Compile-time guarantee.
var _ runs.Executor = (*SyncExecutor)(nil)
