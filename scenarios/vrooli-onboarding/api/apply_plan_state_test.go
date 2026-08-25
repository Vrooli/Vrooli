package main

import "testing"

// TestApplyStateFromReadinessNeverGuesses pins the mapping from a readiness
// verdict to the state shown to the operator before consent. "deferred" and
// "unsupported" both mean this process could not decide, and presenting either
// as "satisfied" would tell the operator an item is already in place when
// nothing verified that.
func TestApplyStateFromReadinessNeverGuesses(t *testing.T) {
	cases := map[string]string{
		"ready":       applyStateSatisfied,
		"missing":     applyStatePending,
		"deferred":    applyStateUnknown,
		"unsupported": applyStateUnknown,
		"":            applyStateUnknown,
		"something":   applyStateUnknown,
	}
	for status, want := range cases {
		if got := applyStateFromReadiness(status); got != want {
			t.Fatalf("applyStateFromReadiness(%q) = %q, want %q", status, got, want)
		}
	}
}

// TestObservedStateDefaultsToUnknown covers the kinds that are not measured.
// Resources and scenarios each need a control-plane round trip, so they are
// reported as unknown rather than assumed satisfied.
func TestObservedStateDefaultsToUnknown(t *testing.T) {
	observed := map[string]string{"tool:git": applyStateSatisfied, "tool:blank": ""}
	if got := observedState(observed, "tool:git"); got != applyStateSatisfied {
		t.Fatalf("measured item = %q, want %q", got, applyStateSatisfied)
	}
	for _, id := range []string{"resource:postgres", "scenario:web-console", "tool:blank"} {
		if got := observedState(observed, id); got != applyStateUnknown {
			t.Fatalf("unmeasured item %q = %q, want %q", id, got, applyStateUnknown)
		}
	}
}

// TestApplyPlanStateDoesNotChangeSelectionDigest is the safety property. State
// is observed from the host, so if it fed the digest, an unrelated host change
// between planning and applying would invalidate consent the operator had
// already given, or silently mark a stale selection as fresh.
func TestApplyPlanStateDoesNotChangeSelectionDigest(t *testing.T) {
	base := []applyItem{
		{ID: "tool:git", Kind: "tool", Name: "git", Required: true},
		{ID: "safeguard:clock", Kind: "safeguard", Name: "clock", Required: true, Privileged: true},
	}
	withState := []applyItem{
		{ID: "tool:git", Kind: "tool", Name: "git", Required: true, State: applyStateSatisfied},
		{ID: "safeguard:clock", Kind: "safeguard", Name: "clock", Required: true, Privileged: true, State: applyStatePending},
	}
	if selectionDigest(base) != selectionDigest(withState) {
		t.Fatal("observed state must not participate in the selection digest")
	}
}

// TestApplyPlanEmitsOnlyTheKindsBothSurfacesRender is the parity guard between
// the wizard CLI and the onboarding UI. Both renderers enumerate these four
// kinds to explain what apply does and to label each section. A fifth kind
// added to buildApplyPlan without updating them would reach an operator as an
// unlabelled item whose host effect is never disclosed, so this test fails
// until both surfaces are taught about it.
//
// CLI: cli/domains/wizard/register.go — pluralKind and the per-kind effect list.
// UI:  ui/src/lib/applyPlan.ts — KIND_LABELS, KIND_ORDER, APPLY_KIND_ACTIONS.
func TestApplyPlanEmitsOnlyTheKindsBothSurfacesRender(t *testing.T) {
	items := buildApplyPlan(applyPlanInput{
		Requirements: hostRequirementsResponse{
			Tools:      []hostItem{{hostRequirement: hostRequirement{Name: "git"}, Status: "required"}},
			Safeguards: []hostItem{{hostRequirement: hostRequirement{Name: "clock"}, Status: "required"}},
		},
		Closure: closureResult{
			Resources: []closureMember{{Name: "postgres"}},
			Scenarios: []closureMember{{Name: "web-console"}},
		},
	})

	rendered := map[string]bool{"tool": true, "safeguard": true, "resource": true, "scenario": true}
	seen := map[string]bool{}
	for _, item := range items {
		if !rendered[item.Kind] {
			t.Fatalf("buildApplyPlan emits kind %q, which neither the wizard CLI nor the onboarding UI can label or explain", item.Kind)
		}
		seen[item.Kind] = true
	}
	for kind := range rendered {
		if !seen[kind] {
			t.Fatalf("kind %q is rendered by both surfaces but this test no longer exercises it; the parity guard would miss a regression", kind)
		}
	}
}
