package execute

import (
	"reflect"
	"testing"

	execTypes "test-genie/cli/internal/execute"
	"test-genie/cli/internal/phases"
)

func TestPlanPhaseOrderPrefersCatalogAndSkip(t *testing.T) {
	descriptors := []phases.Descriptor{
		{Name: "structure"},
		{Name: "dependencies"},
		{Name: "lint"},
		{Name: "unit"},
	}
	got := planPhaseOrder(nil, []string{"lint"}, descriptors, nil)
	want := []string{"structure", "dependencies", "unit"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected catalog-driven phases %v, got %v", want, got)
	}
}

func TestPlanPhaseOrderRespectsRequestedPhasesAndAliases(t *testing.T) {
	requested := []string{"unit", "e2e", "performance"}
	got := planPhaseOrder(requested, []string{"performance"}, nil, nil)
	want := []string{"unit", "playbooks"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected requested phases %v, got %v", want, got)
	}
}

func TestPlanPhaseOrderSkipsDisabledByDefault(t *testing.T) {
	descriptors := []phases.Descriptor{
		{Name: "structure"},
		{Name: "playbooks"},
		{Name: "unit"},
	}
	toggles := map[string]execTypes.PhaseToggle{
		"playbooks": {Disabled: true},
	}
	got := planPhaseOrder(nil, nil, descriptors, toggles)
	want := []string{"structure", "unit"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected disabled phases to be removed by default, want %v got %v", want, got)
	}
}
