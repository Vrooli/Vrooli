package checks

import (
	"context"
	"testing"
	"time"
)

func TestScenarioAgentRecoveryWaitsForContinuousGracePeriod(t *testing.T) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(100, 0)}
	reg.SetClock(clock)
	reg.SetRecoveryGracePeriod(10 * time.Minute)
	reg.SetConfigProvider(&mockConfigProvider{
		enabledChecks:  map[string]bool{"scenario-agent-manager": true},
		autoHealChecks: map[string]bool{"scenario-agent-manager": true},
	})
	reg.Register(&mockHealableCheck{
		id:            "scenario-agent-manager",
		actions:       []RecoveryAction{{ID: "restart", Available: true, Dangerous: true}},
		executeResult: ActionResult{Success: false, Error: "restart failed"},
	})
	requests := 0
	reg.SetRecoveryRequester(func(_ context.Context, _, _ string) (RecoveryAdmission, error) {
		requests++
		return RecoveryAdmission{RequestID: "request-1", Outcome: "failed"}, nil
	})
	result := Result{CheckID: "scenario-agent-manager", Status: StatusCritical, Message: "unhealthy"}

	for _, elapsed := range []time.Duration{0, 5 * time.Minute, 10 * time.Minute} {
		clock.Set(time.Unix(100, 0).Add(elapsed))
		reg.RunAutoHeal(context.Background(), []Result{result})
	}
	if requests != 0 {
		t.Fatalf("recovery requests before ten minutes = %d, want 0", requests)
	}
	clock.Set(time.Unix(100, 0).Add(11 * time.Minute))
	reg.RunAutoHeal(context.Background(), []Result{result})
	if requests != 1 {
		t.Fatalf("recovery requests after grace = %d, want 1", requests)
	}
}

func TestComputedSupervisionMembersNeverBecomePermanentlySuspended(t *testing.T) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(100, 0)}
	reg.SetClock(clock)
	reg.SetSupervisedChecks(map[string]string{"scenario-derived-recovery-dependency": "try_start"})
	reg.SetConfigProvider(&mockConfigProvider{
		enabledChecks:  map[string]bool{"scenario-derived-recovery-dependency": true},
		autoHealChecks: map[string]bool{"scenario-derived-recovery-dependency": true},
	})
	reg.Register(&mockHealableCheck{
		id:            "scenario-derived-recovery-dependency",
		actions:       []RecoveryAction{{ID: "restart", Available: true, Dangerous: true}},
		executeResult: ActionResult{Success: false, Error: "restart failed"},
	})
	result := Result{CheckID: "scenario-derived-recovery-dependency", Status: StatusCritical}
	for i := 0; i < 3; i++ {
		reg.RunAutoHeal(context.Background(), []Result{result})
		clock.Advance(10 * time.Minute)
	}
	tracker, ok := reg.GetHealTracker(result.CheckID)
	if !ok || tracker.IsSuspended() {
		t.Fatalf("computed supervision tracker = %+v, want active retry state", tracker)
	}
}

func TestRunAllPrioritizesSupervisedChecks(t *testing.T) {
	reg := newTestRegistry()
	reg.SetSupervisedChecks(map[string]string{"scenario-agent-manager": "try_start"})
	reg.Register(&mockCheck{
		id:       "optional-slow-check",
		interval: 0,
		result:   Result{CheckID: "optional-slow-check", Status: StatusOK},
	})
	reg.Register(&mockCheck{
		id:       "scenario-agent-manager",
		interval: 0,
		result:   Result{CheckID: "scenario-agent-manager", Status: StatusOK},
	})

	results := reg.RunAll(context.Background(), true)
	if len(results) != 2 {
		t.Fatalf("RunAll returned %d results, want 2", len(results))
	}
	if results[0].CheckID != "scenario-agent-manager" {
		t.Fatalf("first check = %q, want supervised scenario first", results[0].CheckID)
	}
}

func TestScenarioAgentEscalationHonorsTriggerAndRecoveryOwner(t *testing.T) {
	for _, name := range []string{"eligible", "pressure-or-maintenance-gated", "ineligible-observation", "soft-warning"} {
		t.Run(name, func(t *testing.T) {
			reg := newTestRegistry()
			reg.SetRecoveryGracePeriod(0)
			id := "scenario-agent-manager"
			reg.Register(&mockHealableCheck{id: id})
			reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{id: true}})
			reg.healTrackers[id] = &HealTracker{ConsecutiveFailures: 100, UnhealthySince: time.Now().Add(-time.Hour)}
			gate := &sequenceRecoveryGate{allowed: []bool{name != "pressure-or-maintenance-gated"}}
			reg.SetRecoveryOwnershipGate(gate)
			requests := 0
			reg.SetRecoveryRequester(func(context.Context, string, string) (RecoveryAdmission, error) {
				requests++
				return RecoveryAdmission{RequestID: "receipt", Outcome: "failed"}, nil
			})
			result := Result{CheckID: id, Status: StatusCritical}
			if name == "ineligible-observation" {
				result.Details = map[string]interface{}{"autoHealEligible": false}
			}
			if name == "soft-warning" {
				result.Status = StatusWarning
			}
			got := reg.requestScenarioRecoveryIfReady(t.Context(), result)
			want := 0
			if name == "eligible" {
				want = 1
			}
			if requests != want || (got.RequestID != "") != (want == 1) {
				t.Fatalf("escalation receipt=%+v calls=%d want=%d", got, requests, want)
			}
			if name == "pressure-or-maintenance-gated" && gate.calls != 1 {
				t.Fatal("escalation did not consult owner")
			}
		})
	}
}
