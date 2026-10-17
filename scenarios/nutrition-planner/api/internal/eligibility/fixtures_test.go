package eligibility

import "testing"

// FIX-06 candidates intentionally keep hard exclusions independent from any
// preference score: the evaluator must never trade them away.
func TestFIX06RulesBeatPreferencesAcrossCandidates(t *testing.T) {
	profile := Profile{Revision: 4, ExcludedGroups: []string{"soy", "meat"}, Allergies: []string{"peanut", "soy"}, Appliances: []string{"oven"}}
	cases := []struct {
		name      string
		candidate Candidate
		want      Status
	}{
		{"soy tofu bowl", Candidate{Groups: []string{"soy"}, AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent}}, Ineligible},
		{"chicken salad", Candidate{Groups: []string{"meat"}, AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent}}, Ineligible},
		{"stove-only soup", Candidate{RequiredAppliances: []string{"stove"}, AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent}}, Ineligible},
		{"lentil salad unresolved soy", Candidate{AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent, "soy": Unknown}}, NeedsInformation},
		{"ready lentil salad", Candidate{AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent, "soy": DeclaredAbsent}}, Eligible},
		{"bean bowl", Candidate{Groups: []string{"beans"}, AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent, "soy": DeclaredAbsent}}, Eligible},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(profile, tc.candidate)
			if got.Status != tc.want {
				t.Fatalf("got %s want %s reasons=%#v", got.Status, tc.want, got.Reasons)
			}
		})
	}
}
