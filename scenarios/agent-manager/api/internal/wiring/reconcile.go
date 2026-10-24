package wiring

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
)

// ScheduleDeclarationReconcile retries declaration reconciliation after
// bootstrap without delaying server readiness.
func ScheduleDeclarationReconcile(orch *orchestration.Orchestrator, repoRoot string, report func(string, error) bool) {
	if orch == nil || strings.TrimSpace(repoRoot) == "" {
		return
	}
	go func() {
		defer obs.RecoverToFailure("deferred declaration reconcile", nil)
		for _, delay := range []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute} {
			time.Sleep(delay)
			summary := orch.ReconcileDeclaringScenarios(context.Background(), repoRoot)
			selfResult, selfErr := orch.ReconcileSelfDeclarations(context.Background(), repoRoot)
			scenarioErr, selfErr := declarationRetryFailures(summary, selfResult, selfErr)
			if report != nil {
				report("scenario_declarations", scenarioErr)
				report("self_declarations", selfErr)
			}
			if scenarioErr == nil && selfErr == nil {
				obs.Logger().Info("deferred declaration reconciliation complete", "scanned", summary.Scanned, "reconciled", summary.Reconciled)
				return
			}
			obs.Logger().Warn("deferred declaration reconciliation still has failures", "attemptDelay", delay.String(), "failed", summary.Failed, "self_error", selfErr)
		}
	}()
}

// declarationRetryFailures maps both retry owners back to the startup steps
// whose retained health results they can repair. Per-item self failures count
// as incomplete even when the orchestrator returned no top-level error.
func declarationRetryFailures(summary orchestration.SweepSummary, self *orchestration.ReconcileScenarioDeclarationsResult, selfErr error) (scenarioErr, selfFailure error) {
	if summary.Failed > 0 {
		scenarioErr = fmt.Errorf("%d scenario declarations failed", summary.Failed)
	}
	if selfErr != nil {
		selfFailure = selfErr
	} else if self == nil {
		selfFailure = fmt.Errorf("self-declaration reconciliation returned no result")
	} else if failures := self.ProfilesFailed + self.WorkflowsFailed; failures > 0 {
		selfFailure = fmt.Errorf("self-declaration registration incomplete: %d failures", failures)
	}
	return scenarioErr, selfFailure
}
