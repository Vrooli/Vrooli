package eligibility

import "testing"

func TestRequiredRulesCannotBeOverriddenByPreferences(t *testing.T) {
	profile := Profile{Revision: 3, ExcludedGroups: []string{"soy"}, Allergies: []string{"peanut"}, Appliances: []string{"oven"}}
	decision := Evaluate(profile, Candidate{Revision: 7, Groups: []string{"soy"}, RequiredAppliances: []string{"stove"}, AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent}, MethodIDs: []string{"m1"}})
	if decision.Status != Ineligible || len(decision.Reasons) < 2 {
		t.Fatalf("decision %#v", decision)
	}
	if decision.ProfileRevision != 3 || decision.RecipeRevision != 7 || decision.EvaluatorVersion != EvaluatorVersion {
		t.Fatalf("missing traceability %#v", decision)
	}
}

func TestMissingEvidenceIsNotAbsence(t *testing.T) {
	decision := Evaluate(Profile{Allergies: []string{"soy"}}, Candidate{AllergenEvidence: map[string]EvidenceState{}})
	if decision.Status != NeedsInformation {
		t.Fatalf("decision %#v", decision)
	}
	decision = Evaluate(Profile{Allergies: []string{"soy"}}, Candidate{AllergenEvidence: map[string]EvidenceState{"soy": DeclaredAbsent}})
	if decision.Status != Eligible {
		t.Fatalf("absence should be decisive: %#v", decision)
	}
}
