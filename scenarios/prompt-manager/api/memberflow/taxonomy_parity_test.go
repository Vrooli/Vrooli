package memberflow

import (
	"path/filepath"
	"testing"
)

// TestTaxonomyParity_BugAndFriction verifies that the two universal observation
// flow taxonomies (bug-report and friction-report) share the same structural
// shape. They have different domain content but the same top-level keys, an
// `unknown` escape-valve signalType, populated evidenceRules / actionSelection /
// schemas / honestyFlags, and a porPath pointing at a real markdown file.
//
// This codifies the universal-observation-flow primitive: when a third
// universal-source intake lands, its taxonomy must satisfy the same parity, or
// this test should be updated alongside the architectural decision.
func TestTaxonomyParity_BugAndFriction(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	bugTx, err := LoadTaxonomy(repoRoot, "bug-report")
	if err != nil {
		t.Fatalf("LoadTaxonomy(bug-report): %v", err)
	}
	frictionTx, err := LoadTaxonomy(repoRoot, "friction-report")
	if err != nil {
		t.Fatalf("LoadTaxonomy(friction-report): %v", err)
	}

	for _, c := range []struct {
		name string
		tx   *Taxonomy
	}{
		{"bug-report", bugTx},
		{"friction-report", frictionTx},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.tx.SchemaVersion != 1 {
				t.Errorf("schemaVersion: got %d, want 1", c.tx.SchemaVersion)
			}
			if c.tx.OwnerTeam == "" {
				t.Error("owner_team: empty")
			}
			if c.tx.PoRPath == "" {
				t.Error("porPath: empty")
			}
			if len(c.tx.SignalTypes) == 0 {
				t.Error("signalTypes: empty")
			}
			if len(c.tx.EvidenceRules) == 0 {
				t.Error("evidenceRules: empty")
			}
			if len(c.tx.ActionSelect) == 0 {
				t.Error("actionSelection: empty")
			}
			if len(c.tx.Schemas) == 0 {
				t.Error("schemas: empty")
			}
			if len(c.tx.HonestyFlags) == 0 {
				t.Error("honestyFlags: empty")
			}

			// Universal observation flows must include an `unknown` escape
			// valve in their signalTypes list, so the producer always has a
			// honest fallback when classification is uncertain at file time.
			hasUnknown := false
			for _, st := range c.tx.SignalTypes {
				if st.ID == "unknown" {
					hasUnknown = true
					break
				}
			}
			if !hasUnknown {
				t.Error("signalTypes: missing required `unknown` escape-valve entry")
			}
		})
	}
}
