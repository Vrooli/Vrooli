package domain

import "testing"

func TestLiveDependentDelegationAdmitsNarrowerEffortOnly(t *testing.T) {
	parent := &RunAdmission{RuntimeVersion: "codex 0.156", PassedControlArgs: []string{"-m", "gpt-6-luna"},
		EffectiveRunner: "codex", EffectiveModel: "gpt-6-luna", EffectiveEffort: "high"}
	for _, tc := range []struct {
		effort string
		admit  bool
	}{
		{"high", true},
		{"medium", true},
		{"low", true},
		{"xhigh", false},
		{"", false},
	} {
		err := AdmitLiveDependentDelegation(parent, DependentDelegationRequest{Runner: "codex", Model: "gpt-6-luna", Effort: tc.effort})
		if (err == nil) != tc.admit {
			t.Fatalf("effort %q: err=%v, want admit=%v", tc.effort, err, tc.admit)
		}
	}
	if err := AdmitLiveDependentDelegation(parent, DependentDelegationRequest{Runner: "codex", Model: "gpt-6-sol", Effort: "medium"}); err == nil {
		t.Fatal("a different model is not a narrowing and must be refused")
	}
}
