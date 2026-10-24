package safetyadmission

import "testing"

func TestClassifyConservatively(t *testing.T) {
	tests := []struct {
		name   string
		argv   []string
		effect Effect
	}{
		{"status", []string{"git", "-C", "/repo", "status", "--porcelain"}, EffectRead},
		{"diff", []string{"git", "diff", "--cached"}, EffectRead},
		{"init", []string{"git", "init"}, EffectRepoMutation},
		{"commit", []string{"git", "commit", "-m", "seed"}, EffectRepoMutation},
		{"remote push", []string{"git", "push", "origin", "main"}, EffectExternalWrite},
		{"unknown", []string{"git", "custom-command"}, EffectUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := Classify(tt.argv)
			if got != tt.effect {
				t.Fatalf("Classify(%v) = %q, want %q", tt.argv, got, tt.effect)
			}
		})
	}
}

func TestNewReportRejectsAnyNonReadEffect(t *testing.T) {
	report := NewReport([]Command{{Owner: "fixture", Argv: []string{"git", "init"}, Effect: EffectRepoMutation, Reason: "mutation"}})
	if report.Admitted {
		t.Fatal("repository mutation must not be admitted")
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %#v, want one finding", report.Findings)
	}
}

func TestNewReportAdmitsReadOnlyInventory(t *testing.T) {
	report := NewReport([]Command{{Owner: "reader", Argv: []string{"git", "status"}, Effect: EffectRead, Reason: "inspection"}})
	if !report.Admitted || len(report.Findings) != 0 {
		t.Fatalf("read-only report = %#v, want admitted without findings", report)
	}
}
