package validate

import (
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestPresentationRendererUsesCanonicalCapabilityOrderAndFixState(t *testing.T) {
	presentation := &commonv1.PhasePresentation{
		ContractVersion:      "v1",
		CurrentLevel:         "L1",
		NextLevel:            "L2",
		NextAction:           "Add the missing manifest slot.",
		FocusCapabilityLabel: "Manifest Contract",
		DocumentationTopics:  []string{"ui-health missing-slot canonical fix"},
		Capabilities: []*commonv1.PhaseCapabilityPresentation{{
			Id:           "manifest_contract",
			Label:        "Manifest Contract",
			CurrentLevel: "L1",
			NextLevel:    "L2",
			Findings: []*commonv1.PhasePresentationFinding{{
				Code:          "missing-slot",
				Count:         2,
				FixAffordance: commonv1.FixAffordance_FIX_AFFORDANCE_MANUAL,
			}},
		}},
	}
	summary := strings.Join(formatPresentationSummary(presentation), "\n")
	results := strings.Join(formatPresentationResults(presentation), "\n")
	for _, want := range []string{"Presentation v1: L1", "Next: Add the missing manifest slot. [→ Manifest Contract]", "Manifest Contract: L1 → L2", "missing-slot ×2 [MANUAL]", "ui-health missing-slot canonical fix"} {
		if !strings.Contains(summary+"\n"+results, want) {
			t.Fatalf("canonical presentation output missing %q:\n%s\n%s", want, summary, results)
		}
	}
}
