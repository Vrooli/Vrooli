package rules

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

// scenarioRoot resolves the structure-health scenario directory (where .vrooli
// lives) from this test file's location: api/internal/rules → ../../.. .
func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

// TestMaturityLadderIsWellFormed is the anti-drift gate for SH-BOUND-003: every
// finding code in .vrooli/maturity.json must carry a complete mapping (local
// level impact, dimension, default severity) and reference a defined ladder
// level (or the foundation_blocker sentinel). This keeps the emitted finding
// codes and the maturity ladder from drifting apart.
//
// [REQ:SH-BOUND-003]
func TestMaturityLadderIsWellFormed(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(scenarioRoot(t))
	if err != nil {
		t.Fatalf("load maturity spec: %v", err)
	}
	if spec.Provider != "structure-health" || spec.Phase != "structure" {
		t.Fatalf("unexpected provider/phase: %q/%q", spec.Provider, spec.Phase)
	}
	if len(spec.Findings) == 0 {
		t.Fatal("maturity spec has no finding mappings")
	}

	capabilityLevels := map[string]map[string]bool{}
	if len(spec.Capabilities) == 0 {
		capabilityLevels[""] = map[string]bool{}
		for _, lvl := range spec.Levels {
			capabilityLevels[""][lvl.ID] = true
		}
	} else {
		for _, capability := range spec.Capabilities {
			if capability.ID == "" {
				t.Fatal("maturity capability has empty id")
			}
			if capability.Label == "" {
				t.Fatalf("maturity capability %q has empty label", capability.ID)
			}
			capabilityLevels[capability.ID] = map[string]bool{}
			for _, lvl := range capability.Levels {
				capabilityLevels[capability.ID][lvl.ID] = true
			}
		}
	}

	for code, m := range spec.Findings {
		if len(spec.Capabilities) > 0 && m.CapabilityID == "" {
			t.Errorf("%s: empty capability_id", code)
		}
		if m.LocalLevelImpact == "" {
			t.Errorf("%s: empty local_level_impact", code)
		}
		if m.Dimension == "" {
			t.Errorf("%s: empty dimension", code)
		}
		if m.SeverityDefault == "" {
			t.Errorf("%s: empty severity_default", code)
		}
		if m.CleanRequirement == "" {
			t.Errorf("%s: empty clean_requirement", code)
		}
		levels := capabilityLevels[m.CapabilityID]
		// local_level_impact references a level defined on the owning capability
		// unless it is the foundation_blocker sentinel used for unresolvable
		// skeletons in legacy specs.
		if m.LocalLevelImpact != "" && !levels[m.LocalLevelImpact] &&
			string(m.GlobalImpact) != "foundation_blocker" {
			t.Errorf("%s: local_level_impact %q is not a defined level for capability %q", code, m.LocalLevelImpact, m.CapabilityID)
		}
	}
}
