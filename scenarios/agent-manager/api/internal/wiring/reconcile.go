package wiring

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
)

// ScheduleDeclarationReconcile retries declaration reconciliation after
// bootstrap without delaying server readiness.
func ScheduleDeclarationReconcile(orch *orchestration.Orchestrator, repoRoot string) {
	if orch == nil || strings.TrimSpace(repoRoot) == "" {
		return
	}
	go func() {
		defer obs.RecoverToFailure("deferred declaration reconcile", nil)
		for _, delay := range []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute} {
			time.Sleep(delay)
			summary := orch.ReconcileDeclaringScenarios(context.Background(), repoRoot)
			if _, err := orch.ReconcileSelfDeclarations(context.Background(), repoRoot); err != nil {
				obs.Logger().Warn("deferred self-declaration reconciliation failed", obs.KeyError, err.Error())
			}
			if summary.Failed == 0 {
				obs.Logger().Info("deferred declaration reconciliation complete", "scanned", summary.Scanned, "reconciled", summary.Reconciled)
				return
			}
			obs.Logger().Warn("deferred declaration reconciliation still has failures", "attemptDelay", delay.String(), "failed", summary.Failed)
		}
	}()
}
