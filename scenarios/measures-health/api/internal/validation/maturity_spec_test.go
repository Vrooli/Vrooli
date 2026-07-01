package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversEmittedFindingCodes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(ResolveRepoRoot(), "scenarios", "measures-health", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Capabilities) != 5 {
		t.Fatalf("capabilities = %d, want 5", len(spec.Capabilities))
	}
	for _, code := range []string{
		"measures.architecture-fallback",
		"measures.hollow-declaration",
		"measures.illegal-domain-declaration",
		"measures.malformed-declaration",
		"measures.stale-waiver",
		"measures.tier-fallback",
		"measures.tier-partial",
		"measures.uncovered-domain",
		"measures.undeclared-substrate",
	} {
		mapping, ok := spec.Findings[code]
		if !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("maturity spec finding %q must declare capability_id", code)
		}
	}
	if spec.Fallback.CapabilityID == "" {
		t.Fatal("maturity spec fallback must declare capability_id")
	}
}
