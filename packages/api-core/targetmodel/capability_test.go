package targetmodel

import "testing"

func TestCapabilityReadinessCheckUsesFourStateVocabulary(t *testing.T) {
	for _, state := range []ReadinessState{ReadinessReady, ReadinessMissing, ReadinessNotApplicable, ReadinessUnknown} {
		check := CapabilityReadinessCheck("codex", state, "detail", "recover")
		if check.Identity != "capability:codex" || check.State != state {
			t.Fatalf("check = %+v, want capability identity and state %q", check, state)
		}
		if check.Passed != (state == ReadinessReady) {
			t.Fatalf("state %q passed = %v", state, check.Passed)
		}
	}
}

func TestCapabilityReadinessCheckNormalizesInvalidStateToUnknown(t *testing.T) {
	check := CapabilityReadinessCheck("codex", ReadinessState("invalid"), "", "")
	if check.State != ReadinessUnknown || check.Passed {
		t.Fatalf("check = %+v, want unknown and not passed", check)
	}
}
