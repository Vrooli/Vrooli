package models

import (
	"strings"
	"testing"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// A license gate must read as a monetization/redistribution restriction, never
// as a normal-use exclusion; only an exclusion says the model is kept out.
func TestFormatGateSeparatesLicenseGatesFromExclusions(t *testing.T) {
	gate := formatGate(&modelsv1.BlocklistEntry{
		Id: "codeformer", License: "S-Lab License 1.0", Reason: "non-commercial",
		Gate: "license", GatedUses: []string{"monetized_use", "hosted_inference"},
	})
	if !strings.Contains(gate, "license gate, refuses monetized_use, hosted_inference only") || strings.Contains(gate, "excluded") {
		t.Fatalf("license gate rendered as %q", gate)
	}
	exclusion := formatGate(&modelsv1.BlocklistEntry{
		Id: "flux-dev-and-fill", License: "FLUX [dev] non-commercial", Reason: "does not fit",
		Gate: "exclusion", ExclusionKind: "hardware_fit",
	})
	if !strings.Contains(exclusion, "excluded (hardware_fit)") {
		t.Fatalf("exclusion rendered as %q", exclusion)
	}
	if got := findingSeverityLabel(modelsv1.CatalogFindingSeverity_CATALOG_FINDING_SEVERITY_INFO); got != "info" {
		t.Fatalf("info severity label = %q", got)
	}
}
