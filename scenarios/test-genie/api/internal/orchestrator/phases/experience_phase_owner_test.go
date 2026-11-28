package phases

import "testing"

// The unified experience command must return one served-UI verdict. Spec
// authoring remains available under a separate, explicitly scoped phase.
func TestExperiencePhaseHasOneCanonicalProvider(t *testing.T) {
	catalog := DefaultCatalog()

	served, ok := catalog.Lookup("experience")
	if !ok {
		t.Fatal("canonical experience phase is missing")
	}
	if served.Delegated == nil || served.Delegated.ProviderScenario != "ui-health" {
		t.Fatalf("experience owner = %+v, want ui-health", served.Delegated)
	}
	if !served.Delegated.IncludeExecution {
		t.Fatal("experience phase must request runtime evaluation")
	}
	if served.RuntimeClass != "execution" {
		t.Fatalf("experience runtime class = %q, want execution", served.RuntimeClass)
	}
	if served.PhaseClass != "runtime" {
		t.Fatalf("experience phase class = %q, want runtime", served.PhaseClass)
	}
	if served.Doc != "scenarios/test-genie/docs/phases/experience/README.md" {
		t.Fatalf("experience docs = %q", served.Doc)
	}

	if _, ok := catalog.Lookup("ui-health"); ok {
		t.Fatal("retired ui-health key must not register a competing UI verdict")
	}
	contract, ok := catalog.Lookup("experience-contract")
	if !ok {
		t.Fatal("experience-contract phase is missing")
	}
	if contract.Delegated == nil || contract.Delegated.ProviderScenario != "experience-manager" {
		t.Fatalf("experience-contract owner = %+v, want experience-manager", contract.Delegated)
	}
	if contract.Doc != "scenarios/test-genie/docs/phases/experience-contract/README.md" {
		t.Fatalf("experience-contract docs = %q", contract.Doc)
	}
}
