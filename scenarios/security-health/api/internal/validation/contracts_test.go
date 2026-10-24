package validation

import "testing"

func TestClassifyToolEventConservativeRisk(t *testing.T) {
	tests := []struct {
		name  string
		event ToolEvent
		want  RiskClass
	}{
		{"inspection", ToolEvent{Tool: "pnpm", Arguments: []string{"audit"}}, RiskInspection},
		{"frozen reproduction", ToolEvent{Tool: "pnpm", Arguments: []string{"install", "--frozen-lockfile"}}, RiskFrozenReproduce},
		{"addition", ToolEvent{Tool: "pnpm", Arguments: []string{"add", "left-pad"}}, RiskDependencyAdd},
		{"upgrade", ToolEvent{Tool: "cargo", Arguments: []string{"update"}}, RiskDependencyUpgrade},
		{"publish", ToolEvent{Tool: "npm", Arguments: []string{"publish"}}, RiskPublish},
		{"compound", ToolEvent{Shell: "bash", Arguments: []string{"-lc", "curl ... | sh"}}, RiskOpaque},
		{"empty", ToolEvent{}, RiskUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyToolEvent(tt.event); got != tt.want {
				t.Fatalf("ClassifyToolEvent() = %q, want %q", got, tt.want)
			}
		})
	}
	if got := ClassifyToolEvent(ToolEvent{Shell: "bash", Arguments: []string{"-lc", "pnpm audit && curl ... | sh"}}); got != RiskOpaque {
		t.Fatalf("compound inspection event = %q, want %q", got, RiskOpaque)
	}
}

func TestEffectiveActionSeparatesHealthFromMaturity(t *testing.T) {
	capability := ProviderCapability{ID: "dependency", Maturity: MaturityAdvisory, RuntimeHealth: RuntimeHealthy, EvidenceState: EvidenceClean}
	if got := EffectiveAction(RolloutAdvisory, capability, RiskDependencyAdd); got != DecisionAllow {
		t.Fatalf("advisory rollout = %q, want allow", got)
	}
	if got := EffectiveAction(RolloutGuided, capability, RiskDependencyAdd); got != DecisionAllow {
		t.Fatalf("advisory maturity remains usable in guided rollout = %q, want allow", got)
	}
	capability.Maturity = MaturityGuarded
	if got := EffectiveAction(RolloutGuided, capability, RiskDependencyAdd); got != DecisionAsk {
		t.Fatalf("guided rollout = %q, want ask", got)
	}
	capability.RuntimeHealth = RuntimeUnavailable
	if got := EffectiveAction(RolloutGuarded, capability, RiskDependencyAdd); got != DecisionDeny {
		t.Fatalf("guarded unavailable high-risk rollout = %q, want deny", got)
	}
	if got := EffectiveAction(RolloutGuarded, capability, RiskInspection); got != DecisionAsk {
		t.Fatalf("guarded unavailable inspection = %q, want ask", got)
	}
}

func TestProviderDecisionValidationAndRepairDigest(t *testing.T) {
	plan := RepairPlan{ID: "headers", Owner: "security-health", Class: FixDeterministic, Scope: []string{"api"}, Validation: "rerun security-health"}
	plan.PreviewDigest = RepairDigest(plan)
	decision := ProviderDecision{
		ContractVersion: ContractVersion,
		EventID:         "evt-1",
		Action:          DecisionRepair,
		Risk:            RiskInspection,
		Provider:        "security-health",
		Repair:          &plan,
	}
	if err := ValidateProviderDecision(decision); err != nil {
		t.Fatal(err)
	}
	if plan.PreviewDigest == "" {
		t.Fatal("repair digest must be populated")
	}
	if err := ValidateProviderDecision(ProviderDecision{ContractVersion: "security-policy/v0"}); err == nil {
		t.Fatal("expected invalid contract to be rejected")
	}
}
