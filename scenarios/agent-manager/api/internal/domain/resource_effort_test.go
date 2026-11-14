package domain

import (
	"encoding/json"
	"testing"
)

func TestDeliveryEffortConformanceRetainsExactRequestedEffort(t *testing.T) {
	t.Setenv(DeliveryEffortEnforceEnv, "1")
	for _, value := range []Effort{EffortMedium, EffortHigh} {
		c := &RunConfig{Model: "fixture-model", RoleRef: "code.economy.delivery", RunnerType: RunnerTypeCodex, Effort: value, PolicySnapshot: &ExecutionPolicySnapshot{SelectedCandidate: ExecutionCandidate{RunnerType: RunnerTypeCodex, Model: "fixture-model", ResourceRole: "code.delivery", DeclaredEffort: value}}}
		raw, _ := json.Marshal(c)
		if e := ValidateResourceEffort(c); e != nil {
			t.Fatal(e)
		}
		after, _ := json.Marshal(c)
		if string(raw) != string(after) {
			t.Fatal("conformance changed requested configuration")
		}
	}
	for _, declared := range []Effort{"", EffortMedium, "invalid"} {
		c := &RunConfig{Model: "fixture-model", RoleRef: "code.economy.delivery", RunnerType: RunnerTypeCodex, Effort: EffortHigh, PolicySnapshot: &ExecutionPolicySnapshot{SelectedCandidate: ExecutionCandidate{RunnerType: RunnerTypeCodex, Model: "fixture-model", ResourceRole: "code.delivery", DeclaredEffort: declared}}}
		raw, _ := json.Marshal(c)
		if e := ValidateResourceEffort(c); e == nil {
			t.Fatal("high delivery mismatch accepted")
		}
		after, _ := json.Marshal(c)
		if string(raw) != string(after) {
			t.Fatal("refusal coerced effort")
		}
	}
	if e := ValidateResourceEffort(&RunConfig{RoleRef: "code.economy.delivery", Effort: EffortMedium}); e == nil {
		t.Fatal("delivery missing evidence accepted")
	}
	if e := ValidateResourceEffort(&RunConfig{RoleRef: "code.default", Effort: EffortHigh}); e != nil {
		t.Fatal("ordinary high caller changed", e)
	}
}

func TestDeliveryEffortDifferentModelDoesNotInheritPrimaryEvidence(t *testing.T) {
	t.Setenv(DeliveryEffortEnforceEnv, "1")
	cfg := &RunConfig{RoleRef: "code.economy.delivery", Model: "fallback", Effort: EffortMedium}
	candidate := ExecutionCandidate{ResourceRole: "code.delivery", Model: "primary", DeclaredEffort: EffortMedium}
	before, _ := json.Marshal(cfg)
	if ValidateCandidateResourceEffort(cfg, candidate) == nil {
		t.Fatal("fallback inherited primary effort")
	}
	after, _ := json.Marshal(cfg)
	if string(before) != string(after) {
		t.Fatal("fallback refusal changed configuration")
	}
}

// P-18 posture: with the switch unset, a shared high-effort delivery profile
// and a parked run whose snapshot predates effort evidence stay admissible.
func TestDeliveryEffortGateIsOffByDefault(t *testing.T) {
	t.Setenv(DeliveryEffortEnforceEnv, "")
	high := &RunConfig{Model: "gpt-6-luna", RoleRef: "code.economy.delivery", RunnerType: RunnerTypeCodex, Effort: EffortHigh, PolicySnapshot: &ExecutionPolicySnapshot{SelectedCandidate: ExecutionCandidate{RunnerType: RunnerTypeCodex, Model: "gpt-6-luna", ResourceRole: "code.delivery", DeclaredEffort: EffortMedium}}}
	if e := ValidateResourceEffort(high); e != nil {
		t.Fatal("high delivery profile refused with enforcement off", e)
	}
	retained := &RunConfig{Model: "gpt-6-luna", RoleRef: "code.economy.delivery", RunnerType: RunnerTypeCodex, Effort: EffortHigh, PolicySnapshot: &ExecutionPolicySnapshot{SelectedCandidate: ExecutionCandidate{RunnerType: RunnerTypeCodex, Model: "gpt-6-luna", ResourceRole: "code.delivery"}}}
	if e := ValidateResourceEffort(retained); e != nil {
		t.Fatal("retained run without effort evidence refused with enforcement off", e)
	}
	t.Setenv(DeliveryEffortEnforceEnv, "on")
	if ValidateResourceEffort(high) == nil || ValidateResourceEffort(retained) == nil {
		t.Fatal("enforcement switch did not restore the exact effort gate")
	}
}
