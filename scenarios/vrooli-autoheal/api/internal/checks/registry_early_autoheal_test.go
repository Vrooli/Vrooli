package checks

import (
	"context"
	"testing"
)

func TestRunAutoHeal_SelectsFirstSafeAction(t *testing.T) {
	reg := newTestRegistry()

	healableCheck := &mockHealableCheck{
		id:     "healable-check",
		result: Result{CheckID: "healable-check", Status: StatusCritical},
		actions: []RecoveryAction{
			{ID: "first-safe", Available: true, Dangerous: false},
			{ID: "second-safe", Available: true, Dangerous: false},
		},
		executeResult: ActionResult{Success: true},
	}
	reg.Register(healableCheck)

	config := &mockConfigProvider{
		autoHealChecks: map[string]bool{
			"healable-check": true,
		},
	}
	reg.SetConfigProvider(config)

	results := []Result{
		{CheckID: "healable-check", Status: StatusCritical},
	}

	reg.RunAutoHeal(context.Background(), results)

	if len(healableCheck.executedActions) != 1 || healableCheck.executedActions[0] != "first-safe" {
		t.Errorf("expected 'first-safe' to be selected, got %v", healableCheck.executedActions)
	}
}
