package investigationpolicy

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func testPolicy(mode Mode) Policy {
	return Policy{SchemaVersion: SchemaVersion, Version: "recommended-read-only.v1", Mode: mode, Rules: []Rule{{Kind: "no_material_progress", AfterSeconds: 900, IgnoreKnownOwnerWaits: true, Enabled: true}, {Kind: "repeated_failure", MinDistinctAttempts: 3, WindowSeconds: 1800, Enabled: true}, {Kind: "phase_elapsed", AfterSeconds: 3600, MinimumNewEvidence: 1, Enabled: true}, {Kind: "budget_fraction", Fraction: .8, RequireObservedBudget: true, Enabled: true}, {Kind: "terminal_mismatch", Enabled: true}}, Limits: Limits{CooldownSeconds: 1800, MaxInvestigationsPerExecution: 3, MaxConcurrentPerExecution: 1, MaxConcurrentGlobal: 4, MaxInvestigatorDepth: 1}, Recommendations: RecommendationPolicy{AllowedKinds: []string{"observe", "request_evidence", "nudge", "escalate"}}, Exclusions: Exclusions{InvestigationWorkloads: true, DisabledExecutions: true, TerminalExecutions: true}}
}

func TestPolicyIgnoresKnownOwnerWaitAndMatchesRepeatedFailure(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	p := testPolicy(ModeShadow)
	o := Observation{ExecutionID: "exec-1", PhaseID: "phase-1", PhaseGeneration: "7", RuleVersion: p.Version, Now: now, LastMaterialProgress: now.Add(-time.Hour), KnownOwnerWait: true, Attempts: []Attempt{{ID: "a", Failed: true, OccurredAt: now.Add(-10 * time.Minute)}, {ID: "b", Failed: true, OccurredAt: now.Add(-9 * time.Minute)}, {ID: "c", Failed: true, OccurredAt: now.Add(-8 * time.Minute)}}}
	decision, err := p.Evaluate(o)
	if err != nil || !decision.Eligible || len(decision.Matches) != 1 || decision.Matches[0].Kind != "repeated_failure" || decision.IncidentFingerprint == "" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
}

func TestPolicyExcludesInvestigationRecursionAndShadowDoesNotDispatch(t *testing.T) {
	p := testPolicy(ModeShadow)
	o := Observation{ExecutionID: "exec-1", PhaseID: "phase-1", Now: time.Now().UTC(), TerminalMismatch: true, InvestigationWorkload: true, InvestigatorDepth: 1}
	decision, err := p.Evaluate(o)
	if err != nil || decision.Eligible || len(decision.Reasons) < 2 {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if decision.Mode != ModeShadow {
		t.Fatalf("mode=%s", decision.Mode)
	}
}

func TestPolicyCooldownAndBudgetRuleAreExplicit(t *testing.T) {
	p := testPolicy(ModeAutomatic)
	now := time.Now().UTC()
	last := now.Add(-time.Minute)
	o := Observation{ExecutionID: "exec-1", PhaseID: "phase-1", Now: now, BudgetUsed: 80, BudgetTotal: 100, LastIncidentAt: &last}
	decision, err := p.Evaluate(o)
	if err != nil || decision.Eligible || len(decision.Matches) != 1 || decision.Matches[0].Kind != "budget_fraction" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if len(decision.Reasons) == 0 {
		t.Fatal("cooldown reason missing")
	}
}

func TestPolicyQueuesEligibleWorkWhenAdmissionCapacityIsFull(t *testing.T) {
	p := testPolicy(ModeAutomatic)
	o := Observation{ExecutionID: "exec-queue", PhaseID: "phase-1", Now: time.Now().UTC(), TerminalMismatch: true, ConcurrentExecution: 1, ConcurrentGlobal: 4}
	decision, err := p.Evaluate(o)
	if err != nil || !decision.Eligible || !decision.Queued {
		t.Fatalf("decision=%+v err=%v, want eligible queued work", decision, err)
	}
	if len(decision.Reasons) == 0 || decision.Reasons[len(decision.Reasons)-1] == "" {
		t.Fatalf("decision=%+v, want queue explanation", decision)
	}
}

func TestPolicyRulesAreEnabledWhenEnabledFieldIsOmitted(t *testing.T) {
	p := testPolicy(ModeAutomatic)
	for i := range p.Rules {
		p.Rules[i].Enabled = false
	}
	now := time.Now().UTC()
	decision, err := p.Evaluate(Observation{ExecutionID: "exec-1", PhaseID: "phase-1", Now: now, TerminalMismatch: true})
	if err != nil || !decision.Eligible || len(decision.Matches) != 1 || decision.Matches[0].Kind != "terminal_mismatch" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	p.Rules[len(p.Rules)-1].Disabled = true
	decision, err = p.Evaluate(Observation{ExecutionID: "exec-1", PhaseID: "phase-1", Now: now, TerminalMismatch: true})
	if err != nil || decision.Eligible || len(decision.Matches) != 0 {
		t.Fatalf("disabled decision=%+v err=%v", decision, err)
	}
}

func TestRecommendedPolicyExampleIsValid(t *testing.T) {
	raw, err := os.ReadFile("../../../.vrooli/plan-investigation-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	var policy Policy
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	if err := policy.Validate(); err != nil {
		t.Fatalf("example policy invalid: %v", err)
	}
}
