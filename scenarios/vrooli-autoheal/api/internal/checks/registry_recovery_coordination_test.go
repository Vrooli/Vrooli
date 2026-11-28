package checks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func recoveryCoordinationFixture(t *testing.T, exhausted bool) (*Registry, *mockHealableCheck, Result) {
	t.Helper()
	id := "scenario-agent-manager"
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(10_000, 0)}
	reg.SetClock(clock)
	reg.SetConfigProvider(&mockConfigProvider{enabledChecks: map[string]bool{id: true}, autoHealChecks: map[string]bool{id: true}})
	result := Result{CheckID: id, Status: StatusCritical, Message: "actual target is unhealthy"}
	check := &mockHealableCheck{id: id, result: result, actions: []RecoveryAction{{ID: "restart", Available: true, Dangerous: true}}, executeResult: ActionResult{Success: false}}
	reg.Register(check)
	if exhausted {
		reg.healTrackers[id] = &HealTracker{ConsecutiveFailures: 3, UnhealthySince: clock.Now().Add(-time.Hour)}
	}
	return reg, check, result
}

func TestAdmittedNativeRepairExcludesSameTickMechanicalRestart(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(map[bool]string{false: "acknowledged", true: "ambiguous-dispatch"}[uncertain], func(t *testing.T) {
			reg, check, result := recoveryCoordinationFixture(t, true)
			requests := 0
			reg.SetRecoveryRequester(func(context.Context, string, string) (RecoveryAdmission, error) {
				requests++
				var err error
				if uncertain {
					err = errors.New("dispatch acknowledgement lost after durable admission")
				}
				return RecoveryAdmission{RequestID: "admitted-native-repair", Outcome: "scheduled", NativePending: true}, err
			})
			reg.RunAutoHeal(t.Context(), []Result{result})
			reg.RunAutoHeal(t.Context(), []Result{result})
			if requests != 1 || len(check.executedActions) != 0 {
				t.Fatalf("native repair raced or replayed: requests=%d mechanical=%v", requests, check.executedActions)
			}
		})
	}
}

func TestNativeRepairOwnershipSurvivesHealthyObservationAndReleasesAfterExit(t *testing.T) {
	reg, check, result := recoveryCoordinationFixture(t, true)
	pending := true
	reg.SetRecoveryObserver(func(context.Context) (map[string]RecoveryAdmission, error) {
		return map[string]RecoveryAdmission{result.CheckID: {RequestID: "native", Outcome: "running", NativePending: pending}}, nil
	})
	healthy := result
	healthy.Status = StatusOK
	reg.RunAutoHeal(t.Context(), []Result{healthy})
	reg.RunAutoHeal(t.Context(), []Result{result})
	if len(check.executedActions) != 0 {
		t.Fatal("healthy observation reset the independent native execution owner")
	}
	pending = false
	reg.RunAutoHeal(t.Context(), []Result{result})
	if len(check.executedActions) != 1 {
		t.Fatalf("exact native exit did not return mechanical recovery: %v", check.executedActions)
	}
}

func TestNativeRecoveryOwnershipRefreshesImmediatelyBeforeMechanicalEffect(t *testing.T) {
	reg, check, result := recoveryCoordinationFixture(t, false)
	reads := 0
	reg.SetRecoveryObserver(func(context.Context) (map[string]RecoveryAdmission, error) {
		reads++
		return map[string]RecoveryAdmission{result.CheckID: {RequestID: "externally-admitted", NativePending: reads > 1}}, nil
	})
	reg.RunAutoHeal(t.Context(), []Result{result})
	if reads < 2 || len(check.executedActions) != 0 {
		t.Fatalf("collection snapshot authorized a later conflicting effect: reads=%d actions=%v", reads, check.executedActions)
	}
}

func TestRetainedTerminalAndLegacyReceiptsPreserveMechanicalRecovery(t *testing.T) {
	for _, outcome := range []string{"succeeded", "failed", "unconfirmed"} {
		t.Run(outcome, func(t *testing.T) {
			reg, check, result := recoveryCoordinationFixture(t, true)
			reg.SetRecoveryRequester(func(context.Context, string, string) (RecoveryAdmission, error) {
				return RecoveryAdmission{RequestID: "retained-budget-receipt", Outcome: outcome}, nil
			})
			results := reg.RunAutoHeal(t.Context(), []Result{result})
			if len(check.executedActions) != 1 {
				t.Fatalf("receipt without native ownership disabled mechanical repair: %v", results)
			}
			for _, result := range results {
				if strings.Contains(result.Reason, "scheduled") {
					t.Fatalf("retained receipt was mislabeled new work: %+v", result)
				}
			}
		})
	}
}

func TestRecoveryObserverFailurePreservesOnlyKnownNativeOwnership(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(map[bool]string{false: "optional-broker-unavailable", true: "known-native-writer"}[known], func(t *testing.T) {
			reg, check, result := recoveryCoordinationFixture(t, false)
			if known {
				reg.retainRecoveryAdmission(result.CheckID, RecoveryAdmission{RequestID: "known", NativePending: true})
			}
			reg.SetRecoveryObserver(func(context.Context) (map[string]RecoveryAdmission, error) {
				return nil, errors.New("owner read unavailable")
			})
			reg.RunAutoHeal(t.Context(), []Result{result})
			if (len(check.executedActions) == 0) != known {
				t.Fatalf("optional observation failure invented or erased native ownership: known=%v actions=%v", known, check.executedActions)
			}
		})
	}
}
