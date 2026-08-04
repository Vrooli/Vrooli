package wiring

import (
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/modelpolicydrift"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
)

// Shutdown stops durable background workers before closing storage. Parked
// runs and nudge work are recovered on the next bootstrap, so shutdown is
// safe even when a worker is interrupted mid-operation.
func Shutdown(db *database.DB, reconciler *orchestration.Reconciler, awaitRegistry *orchestration.AwaitRegistry, workflowNudger *orchestration.WorkflowNudger, drift ...*modelpolicydrift.Scheduler) {
	shutdownLog := obs.Component("shutdown")
	if reconciler != nil {
		if err := reconciler.Stop(); err != nil {
			shutdownLog.Warn("reconciler shutdown failed", obs.KeyError, err.Error())
		}
	}
	if awaitRegistry != nil {
		awaitRegistry.Stop()
	}
	if workflowNudger != nil {
		workflowNudger.Stop()
	}
	for _, scheduler := range drift {
		if scheduler != nil {
			scheduler.Stop()
		}
	}
	if db != nil {
		_ = db.Close()
	}
	shutdownLog.Info("server stopped")
}
