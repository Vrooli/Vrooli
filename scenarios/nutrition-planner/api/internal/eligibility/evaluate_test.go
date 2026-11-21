package eligibility

import "testing"

func TestRequiredRulesCannotBeOverriddenByPreferences(t *testing.T) {
	profile := Profile{Revision: 3, ExcludedGroups: []string{"soy"}, Allergies: []string{"peanut"}, Appliances: []string{"oven"}}
	decision := Evaluate(profile, Candidate{Revision: 7, Groups: []string{"soy"}, AllergenEvidence: map[string]EvidenceState{"peanut": DeclaredAbsent}, Methods: []Method{{ID: "m1", PreparationMode: PreparationCooking, RequiredAppliances: []string{"stove"}, RequiredDeviceFunctions: []string{"cooktop"}}}})
	if decision.Status != Ineligible || len(decision.Reasons) < 2 {
		t.Fatalf("decision %#v", decision)
	}
	if decision.ProfileRevision != 3 || decision.RecipeRevision != 7 || decision.EvaluatorVersion != EvaluatorVersion {
		t.Fatalf("missing traceability %#v", decision)
	}
}

func TestMissingEvidenceIsNotAbsence(t *testing.T) {
	ready := []Method{{ID: "ready", PreparationMode: PreparationReadyToEat}}
	decision := Evaluate(Profile{Allergies: []string{"soy"}}, Candidate{AllergenEvidence: map[string]EvidenceState{}, Methods: ready})
	if decision.Status != NeedsInformation {
		t.Fatalf("decision %#v", decision)
	}
	decision = Evaluate(Profile{Allergies: []string{"soy"}}, Candidate{AllergenEvidence: map[string]EvidenceState{"soy": DeclaredAbsent}, Methods: ready})
	if decision.Status != Eligible {
		t.Fatalf("absence should be decisive: %#v", decision)
	}
}

func TestEachCompleteMethodIsAssessedAsAnAlternative(t *testing.T) {
	profile := Profile{Revision: 8, Appliances: []string{"oven"}, DeviceFunctions: []string{"oven"}}
	decision := Evaluate(profile, Candidate{Revision: 13, Methods: []Method{
		{ID: "stovetop", Name: "Stovetop", PreparationMode: PreparationCooking, RequiredAppliances: []string{"stove"}, RequiredDeviceFunctions: []string{"cooktop"}},
		{ID: "roast", Name: "Roast", PreparationMode: PreparationCooking, RequiredAppliances: []string{"oven"}, RequiredDeviceFunctions: []string{"oven"}},
	}})
	if decision.Status != Eligible || len(decision.ViableMethodIDs) != 1 || decision.ViableMethodIDs[0] != "roast" {
		t.Fatalf("alternatives were combined or lost: %#v", decision)
	}
	if len(decision.Methods) != 2 || decision.Methods[0].Status != Ineligible || decision.Methods[1].Status != Eligible {
		t.Fatalf("per-method status not reported: %#v", decision.Methods)
	}
	if decision.ProfileRevision != 8 || decision.RecipeRevision != 13 {
		t.Fatalf("revisions not traceable: %#v", decision)
	}
}

func TestDeviceFunctionsAreExplicitAndIndependentFromDeviceNames(t *testing.T) {
	method := Method{ID: "slow-cook", PreparationMode: PreparationCooking, RequiredAppliances: []string{"slow_cooker"}, RequiredDeviceFunctions: []string{"slow_cook"}}
	profile := Profile{Appliances: []string{"slow_cooker"}, DeviceFunctions: []string{"pressure_cook"}}
	decision := Evaluate(profile, Candidate{Methods: []Method{method}})
	if decision.Status != Ineligible || len(decision.ViableMethodIDs) != 0 {
		t.Fatalf("a device name or category was treated as its enabled function: %#v", decision)
	}
	profile.DeviceFunctions = []string{"pressure_cook", "slow_cook"}
	decision = Evaluate(profile, Candidate{Methods: []Method{method}})
	if decision.Status != Eligible || len(decision.ViableMethodIDs) != 1 || decision.ViableMethodIDs[0] != "slow-cook" {
		t.Fatalf("explicit device function did not satisfy method: %#v", decision)
	}
}

func TestZeroApplianceProfilesRequireExplicitAssemblyOrReadyToEatMode(t *testing.T) {
	profile := Profile{Cookware: []string{"saucepan"}, Tools: []string{"chef_knife"}}
	cases := []struct {
		name   string
		method Method
		want   Status
	}{
		{"cooking is not inferred as appliance-free", Method{ID: "cook", PreparationMode: PreparationCooking, RequiredDeviceFunctions: []string{"cooktop"}}, Ineligible},
		{"explicit assembly uses selected tools", Method{ID: "assembly", PreparationMode: PreparationAssembly, RequiredTools: []string{"chef_knife"}}, Eligible},
		{"explicit ready-to-eat needs no appliance", Method{ID: "ready", PreparationMode: PreparationReadyToEat}, Eligible},
		{"unspecified mode is not ready-to-eat", Method{ID: "unknown"}, NeedsInformation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(profile, Candidate{Methods: []Method{tc.method}})
			if got.Status != tc.want {
				t.Fatalf("status=%s want=%s decision=%#v", got.Status, tc.want, got)
			}
		})
	}
}

func TestMethodsRequireEveryCapabilityAndRecipeNeedsAMethod(t *testing.T) {
	profile := Profile{Appliances: []string{"oven"}, Cookware: []string{"baking_sheet"}, DeviceFunctions: []string{"oven"}}
	decision := Evaluate(profile, Candidate{Methods: []Method{
		{ID: "partial", PreparationMode: PreparationCooking, RequiredAppliances: []string{"oven"}, RequiredCookware: []string{"casserole_dish"}, RequiredDeviceFunctions: []string{"oven"}},
	}})
	if decision.Status != Ineligible || len(decision.ViableMethodIDs) != 0 {
		t.Fatalf("method qualified without every requirement: %#v", decision)
	}
	decision = Evaluate(profile, Candidate{})
	if decision.Status != NeedsInformation || len(decision.Methods) != 0 {
		t.Fatalf("methodless legacy recipe was treated as a complete method: %#v", decision)
	}
}
