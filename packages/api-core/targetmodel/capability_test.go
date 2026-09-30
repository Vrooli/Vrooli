package targetmodel

import "testing"

func TestCapabilityReadinessCheckUsesFourStateVocabulary(t *testing.T) {
	for _, state := range []ReadinessState{ReadinessReady, ReadinessMissing, ReadinessNotApplicable, ReadinessUnknown} {
		check := CapabilityReadinessCheck("codex", "Codex", state, "detail", "recover")
		if check.Identity != "capability:codex" || check.State != state {
			t.Fatalf("check = %+v, want capability identity and state %q", check, state)
		}
		if check.Passed != (state == ReadinessReady) {
			t.Fatalf("state %q passed = %v", state, check.Passed)
		}
	}
}

func TestCapabilityReadinessCheckNormalizesInvalidStateToUnknown(t *testing.T) {
	check := CapabilityReadinessCheck("codex", "Codex", ReadinessState("invalid"), "", "")
	if check.State != ReadinessUnknown || check.Passed {
		t.Fatalf("check = %+v, want unknown and not passed", check)
	}
}

// The identity stays the slug while the label becomes the human name. A
// consumer keys on Identity and renders Label; conflating the two is what made
// the session launcher offer an agent called "codex".
func TestCapabilityReadinessCheckKeepsSlugIdentityAndHumanLabel(t *testing.T) {
	check := CapabilityReadinessCheck("claude", "Claude Code", ReadinessReady, "", "")
	if check.Identity != "capability:claude" {
		t.Fatalf("Identity = %q, want capability:claude", check.Identity)
	}
	if check.Label != "Claude Code" {
		t.Fatalf("Label = %q, want Claude Code", check.Label)
	}
}

// A producer that has no name for a capability must still yield a labelled
// fact. Falling back to the slug is strictly better than an empty string,
// which renders as a nameless card.
func TestCapabilityReadinessCheckFallsBackToSlugWhenLabelBlank(t *testing.T) {
	for _, label := range []string{"", "   "} {
		check := CapabilityReadinessCheck("opencode", label, ReadinessReady, "", "")
		if check.Label != "opencode" {
			t.Fatalf("label %q gave Label = %q, want opencode", label, check.Label)
		}
	}
}
