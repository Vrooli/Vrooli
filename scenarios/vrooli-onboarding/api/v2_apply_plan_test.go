package main

import "testing"

func TestBuildApplyPlanUsesConsentAndClosureAsOnePlan(t *testing.T) {
	falseValue := false
	plan := buildApplyPlan(applyPlanInput{
		Closure: closureResult{
			Resources: []closureMember{
				{Name: "required-db", Required: true, Provenance: []closureProvenance{{From: "writer"}}},
				{Name: "declined-ai", Required: false},
			},
			Scenarios: []closureMember{{Name: "writer", Direct: true, Provenance: []closureProvenance{{From: "writer"}}}},
		},
		Requirements: hostRequirementsResponse{
			Tools:      []hostItem{{hostRequirement: hostRequirement{Name: "git", Required: true}, Status: "required"}, {hostRequirement: hostRequirement{Name: "optional-tool"}, Status: "optional"}},
			Safeguards: []hostItem{{hostRequirement: hostRequirement{Name: "firewall"}, Status: "opted_in"}, {hostRequirement: hostRequirement{Name: "declined-safeguard"}, Status: "optional"}},
		},
		State: OperatorState{Resources: map[string]EnabledChoice{"declined-ai": {Enabled: &falseValue}}},
	})
	if got := len(plan); got != 4 {
		t.Fatalf("plan item count = %d, want 4: %#v", got, plan)
	}
	for _, want := range []string{"tool:git", "safeguard:firewall", "resource:required-db", "scenario:writer"} {
		if !hasApplyItem(plan, want) {
			t.Errorf("plan missing %s: %#v", want, plan)
		}
	}
	if hasApplyItem(plan, "resource:declined-ai") || hasApplyItem(plan, "tool:optional-tool") || hasApplyItem(plan, "safeguard:declined-safeguard") {
		t.Fatal("plan included an unconsented optional item")
	}
	if len(plan[3].Dependencies) != 1 || plan[3].Dependencies[0] != "resource:required-db" {
		t.Fatalf("scenario dependency = %#v", plan[3].Dependencies)
	}
}

func hasApplyItem(items []applyItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
