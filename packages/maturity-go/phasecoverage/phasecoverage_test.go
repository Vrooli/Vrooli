package phasecoverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/dimensions"
)

func TestLoadRepositoryDescriptorCoverage(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	coverage, err := Load(repoRoot)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if dim, ok := coverage.FirstDimensionForPhase("unit"); !ok || dim != dimensions.Dimension("tests") {
		t.Fatalf("unit first dimension = %q, %v; want tests, true", dim, ok)
	}
	phases := coverage.PhasesForDimensions(dimensions.Dimension("coverage"))
	found := false
	for _, phase := range phases {
		if phase == "unit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("coverage phases = %v, want unit", phases)
	}
	freshness := coverage.FreshnessRequiredPhases()
	if len(freshness) == 0 {
		t.Fatal("freshness required phases are empty")
	}
}

func TestDescriptorMetadataDrivesCoverageProjection(t *testing.T) {
	repoRoot := t.TempDir()
	descriptorPath := filepath.Join(repoRoot, "scenarios", "demo-health", ".vrooli", "test-genie.json")
	if err := os.MkdirAll(filepath.Dir(descriptorPath), 0o755); err != nil {
		t.Fatalf("mkdir descriptor dir: %v", err)
	}
	writeDescriptor := func(t *testing.T, dim, freshness string) {
		t.Helper()
		raw, err := json.Marshal(descriptor{
			Phase:                "demo",
			OrderHint:            1,
			Dimensions:           []string{dim},
			FreshnessRequirement: freshness,
		})
		if err != nil {
			t.Fatalf("marshal descriptor: %v", err)
		}
		if err := os.WriteFile(descriptorPath, raw, 0o644); err != nil {
			t.Fatalf("write descriptor: %v", err)
		}
	}

	writeDescriptor(t, "tests", "never")
	coverage, err := Load(repoRoot)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if phases := coverage.PhasesForDimensions(dimensions.Dimension("tests")); len(phases) != 1 || phases[0] != "demo" {
		t.Fatalf("tests phases = %v, want [demo]", phases)
	}
	if phases := coverage.PhasesForDimensions(dimensions.Dimension("docs")); len(phases) != 0 {
		t.Fatalf("docs phases = %v, want none before descriptor update", phases)
	}
	if freshness := coverage.FreshnessRequiredPhases(); len(freshness) != 0 {
		t.Fatalf("freshness phases = %v, want none", freshness)
	}

	writeDescriptor(t, "docs", "when_applicable")
	coverage, err = Load(repoRoot)
	if err != nil {
		t.Fatalf("Load after descriptor update failed: %v", err)
	}
	if phases := coverage.PhasesForDimensions(dimensions.Dimension("tests")); len(phases) != 0 {
		t.Fatalf("tests phases = %v, want none after descriptor update", phases)
	}
	if phases := coverage.PhasesForDimensions(dimensions.Dimension("docs")); len(phases) != 1 || phases[0] != "demo" {
		t.Fatalf("docs phases = %v, want [demo] after descriptor update", phases)
	}
	if freshness := coverage.FreshnessRequiredPhases(); len(freshness) != 1 || freshness[0] != "demo" {
		t.Fatalf("freshness phases = %v, want [demo]", freshness)
	}
}
