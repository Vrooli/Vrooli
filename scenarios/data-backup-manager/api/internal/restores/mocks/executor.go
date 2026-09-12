package mocks

import (
	"context"

	"data-backup-manager/internal/restores"
)

// SyncExecutor is the test executor: it runs each submitted restore/verify job
// inline and to completion on the caller's goroutine, so a service test can
// request a restore and then immediately observe the terminal record without
// polling or sleeping. It mirrors the production async contract (Bind once,
// then Submit) but collapses the asynchrony for determinism.
type SyncExecutor struct {
	baseCtx context.Context
	run     restores.RestoreFunc
}

// NewSyncExecutor constructs an inline executor for tests.
func NewSyncExecutor() *SyncExecutor { return &SyncExecutor{} }

func (e *SyncExecutor) Bind(baseCtx context.Context, run restores.RestoreFunc) {
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	e.baseCtx = baseCtx
	e.run = run
}

func (e *SyncExecutor) Submit(job restores.RestoreJob) {
	if e.run != nil {
		e.run(e.baseCtx, job)
	}
}

func (e *SyncExecutor) Shutdown(context.Context) error { return nil }

// Compile-time guarantee.
var _ restores.Executor = (*SyncExecutor)(nil)
