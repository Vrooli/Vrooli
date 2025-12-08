package execute

import (
	"reflect"
	"testing"

	"test-genie/cli/internal/phases"
)

func TestPlanPhaseOrderPrefersCatalogAndSkip(t *testing.T) {
	descriptors := []phases.Descriptor{
		{Name: "structure"},
		{Name: "dependencies"},
		{Name: "lint"},
		{Name: "unit"},
	}
	got := planPhaseOrder(nil, []string{"lint"}, descriptors)
	want := []string{"structure", "dependencies", "unit"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected catalog-driven phases %v, got %v", want, got)
	}
}

func TestPlanPhaseOrderRespectsRequestedPhasesAndAliases(t *testing.T) {
	requested := []string{"unit", "e2e", "performance"}
	got := planPhaseOrder(requested, []string{"performance"}, nil)
	want := []string{"unit", "playbooks"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected requested phases %v, got %v", want, got)
	}
}
