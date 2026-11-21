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
		retryDeclarationReconcile(context.Background(), waitDeclarationRetry, func(ctx context.Context) (orchestration.SweepSummary, *orchestration.ReconcileScenarioDeclarationsResult, error) {
			summary := orch.ReconcileDeclaringScenarios(ctx, repoRoot)
			selfResult, selfErr := orch.ReconcileSelfDeclarations(ctx, repoRoot)
			return summary, selfResult, selfErr
		}, report)
	}()
}

func waitDeclarationRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Keep retrying at a capped cadence: a long dependency outage must not leave
// startup readiness permanently degraded after its owner becomes available.
func retryDeclarationReconcile(ctx context.Context, wait func(context.Context, time.Duration) bool, reconcile func(context.Context) (orchestration.SweepSummary, *orchestration.ReconcileScenarioDeclarationsResult, error), report func(string, error) bool) {
	delays := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}
	for attempt := 0; ; attempt++ {
		index := attempt
		if index >= len(delays) {
			index = len(delays) - 1
		}
		delay := delays[index]
		if !wait(ctx, delay) || ctx.Err() != nil {
			return
		}
		summary, selfResult, selfErr := reconcile(ctx)
		scenarioErr, selfErr := declarationRetryFailures(summary, selfResult, selfErr)
		reported := true
		if report != nil {
			scenarioReported := report("scenario_declarations", scenarioErr)
			selfReported := report("self_declarations", selfErr)
			reported = scenarioReported && selfReported
		}
		if scenarioErr == nil && selfErr == nil && reported {
			obs.Logger().Info("deferred declaration reconciliation complete", "scanned", summary.Scanned, "reconciled", summary.Reconciled)
			return
		}
		obs.Logger().Warn("deferred declaration reconciliation still has failures", "attemptDelay", delay.String(), "failed", summary.Failed, "self_error", selfErr)
	}
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
