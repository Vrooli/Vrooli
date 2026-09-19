package assessment

import (
	"reflect"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestPresentationViewPreservesProviderCapabilityOrder(t *testing.T) {
	presentation := &commonv1.PhasePresentation{
		ContractVersion: PhasePresentationContractVersion,
		Provider:        "brand-manager",
		Phase:           "branding",
		CurrentLevel:    "L1",
		Capabilities: []*commonv1.PhaseCapabilityPresentation{
			{Id: "second", Label: "Second", CurrentLevel: "L1"},
			{Id: "first", Label: "First", CurrentLevel: "L0"},
		},
	}
	view := PresentationView(presentation)
	if view.Summary != "brand-manager/branding: L1" {
		t.Fatalf("summary = %q", view.Summary)
	}
	want := []string{"Second: L1", "First: L0"}
	if !reflect.DeepEqual(view.Lines, want) {
		t.Fatalf("lines = %v, want %v", view.Lines, want)
	}
}

func TestPresentationViewDoesNotReconstructMissingOrHistoricalPresentation(t *testing.T) {
	for name, presentation := range map[string]*commonv1.PhasePresentation{
		"missing":    nil,
		"historical": {ContractVersion: "legacy"},
	} {
		t.Run(name, func(t *testing.T) {
			view := PresentationView(presentation)
			if view.Summary == "" || len(view.Lines) == 0 {
				t.Fatalf("view = %+v, want explicit degraded rendering", view)
			}
		})
	}
}
