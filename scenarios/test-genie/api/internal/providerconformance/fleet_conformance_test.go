package providerconformance

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/phases"
)

// TestEveryCatalogPhaseResolvesConformantContract is the fleet anti-drift guard:
// every provider descriptor that backs a live catalog phase must satisfy the
// Phase Capability Contract — a first-class North Star, a next_unlock on every
// non-top rung, and a skeleton-conformant remediation doc. It runs the real
// validator (descriptor + doc checks; the live probe is nil in test) against
// each provider's committed descriptor, so a regression in any provider's
// ladder or remediation doc fails here before it can ship.
func TestEveryCatalogPhaseResolvesConformantContract(t *testing.T) {
	repoRoot := repoRootFromTest(t)

	// The set of phases the live catalog exposes.
	catalogPhases := map[string]bool{}
	for _, name := range phases.ValidPhaseNames() {
		catalogPhases[name] = true
	}

	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatalf("read scenarios dir: %v", err)
	}

	svc := New(repoRoot)
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		descPath := filepath.Join(scenariosDir, scenario, ".vrooli", "test-genie.json")
		raw, err := os.ReadFile(descPath)
		if err != nil {
			continue // not a phase provider
		}
		var head struct {
			Phase string `json:"phase"`
		}
		if err := json.Unmarshal(raw, &head); err != nil || head.Phase == "" {
			continue
		}
		if !catalogPhases[head.Phase] {
			continue // descriptor exists but its phase is not in the live catalog
		}
		checked++
		report, err := svc.ValidateScenario(context.Background(), scenario, "")
		if err != nil {
			t.Errorf("%s (%s): ValidateScenario error: %v", scenario, head.Phase, err)
			continue
		}
		for _, f := range report.Findings {
			switch f.Code {
			case CodeNorthStarMissing, CodeLadderIncomplete, CodeDocsSkeletonIncomplete, CodeDocsMissing:
				t.Errorf("%s (%s): contract gap %s — %s", scenario, head.Phase, f.Code, f.Message)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no catalog-backing provider descriptors were checked; the discovery changed")
	}
	t.Logf("verified Phase Capability Contract conformance for %d catalog-backing providers", checked)
}
