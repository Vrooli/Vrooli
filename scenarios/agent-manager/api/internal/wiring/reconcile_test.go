package wiring

import (
	"errors"
	"testing"

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
