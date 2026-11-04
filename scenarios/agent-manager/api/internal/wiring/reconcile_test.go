package wiring

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/orchestration"
)

func TestDeclarationRetryFailuresRequireBothOwnersToSucceed(t *testing.T) {
	scenarioErr, selfErr := declarationRetryFailures(orchestration.SweepSummary{}, &orchestration.ReconcileScenarioDeclarationsResult{}, nil)
	if scenarioErr != nil || selfErr != nil {
		t.Fatalf("successful retry reported failure: scenario=%v self=%v", scenarioErr, selfErr)
	}

	scenarioErr, selfErr = declarationRetryFailures(
		orchestration.SweepSummary{Failed: 1},
		&orchestration.ReconcileScenarioDeclarationsResult{ProfilesFailed: 1},
		nil,
	)
	if scenarioErr == nil || selfErr == nil {
		t.Fatalf("failed retry was reported complete: scenario=%v self=%v", scenarioErr, selfErr)
	}

	callErr := errors.New("self declaration source unavailable")
	scenarioErr, selfErr = declarationRetryFailures(orchestration.SweepSummary{}, nil, callErr)
	if scenarioErr != nil || !errors.Is(selfErr, callErr) {
		t.Fatalf("top-level self retry error lost: scenario=%v self=%v", scenarioErr, selfErr)
	}
}

func TestDeclarationReconcileRetriesPastInitialWindowAndWaitsForHealthOwner(t *testing.T) {
	attempts, reports := 0, 0
	var delays []time.Duration
	retryDeclarationReconcile(context.Background(), func(_ context.Context, delay time.Duration) bool {
		delays = append(delays, delay)
		return len(delays) <= 6
	}, func(context.Context) (orchestration.SweepSummary, *orchestration.ReconcileScenarioDeclarationsResult, error) {
		attempts++
		if attempts <= 3 {
			return orchestration.SweepSummary{Failed: 1}, &orchestration.ReconcileScenarioDeclarationsResult{ProfilesFailed: 1}, nil
		}
		return orchestration.SweepSummary{}, &orchestration.ReconcileScenarioDeclarationsResult{}, nil
	}, func(_ string, _ error) bool {
		reports++
		// The first successful retry arrives before startup has finished.
		return attempts > 4
	})
	if attempts != 5 || reports != 10 {
		t.Fatalf("recovery must continue until both owners accept the result: attempts=%d reports=%d", attempts, reports)
	}
	if len(delays) != 5 || delays[0] != 5*time.Second || delays[1] != 30*time.Second || delays[2] != 2*time.Minute || delays[4] != 2*time.Minute {
		t.Fatalf("retry cadence was not capped: %v", delays)
	}
}

func TestDeclarationReconcileStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	retryDeclarationReconcile(ctx, waitDeclarationRetry, func(context.Context) (orchestration.SweepSummary, *orchestration.ReconcileScenarioDeclarationsResult, error) {
		called = true
		return orchestration.SweepSummary{}, nil, nil
	}, nil)
	if called {
		t.Fatal("cancelled owner must not start reconciliation")
	}
}
