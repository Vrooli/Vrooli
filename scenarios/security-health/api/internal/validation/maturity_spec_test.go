package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversSecurityHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}

	if spec.Provider != "security-health" {
		t.Fatalf("provider = %q, want security-health", spec.Provider)
	}
	if spec.Phase != "security" {
		t.Fatalf("phase = %q, want security", spec.Phase)
	}
	for _, code := range []string{
		"security-health.scenario-not-found",
		"security-health.scanner-absent",
		"security-health.scanner-degraded",
		"security-health.substrate-unsupported",
	} {
		if _, ok := spec.Findings[code]; !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
	}
	if spec.Fallback.GlobalImpact != assessment.ImpactSafetyBlocker {
		t.Fatalf("fallback impact = %q, want safety_blocker", spec.Fallback.GlobalImpact)
	}
}
