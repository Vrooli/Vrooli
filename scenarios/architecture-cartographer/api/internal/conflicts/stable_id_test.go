package conflicts_test

import (
	"strings"
	"testing"

	"architecture-cartographer/internal/conflicts"
)

func TestStableID_DeterministicAcrossRuns(t *testing.T) {
	c1 := conflicts.Conflict{
		Scenario:  "demo",
		Detector:  "cycle",
		Type:      "cycle",
		Subtype:   "cross-domain",
		Locations: []string{"a", "b"},
		Domains:   []string{"x", "y"},
	}
	c2 := c1
	// Reorder locations + domains to confirm sort-canonicalization.
	c2.Locations = []string{"b", "a"}
	c2.Domains = []string{"y", "x"}
	if conflicts.StableID(c1) != conflicts.StableID(c2) {
		t.Fatalf("stable_id should be order-invariant: %s vs %s", conflicts.StableID(c1), conflicts.StableID(c2))
	}
	if !strings.HasPrefix(conflicts.StableID(c1), "csid:") {
		t.Fatalf("stable_id should be csid: prefixed, got %q", conflicts.StableID(c1))
	}
}

func TestStableID_ChangesWithType(t *testing.T) {
	base := conflicts.Conflict{Scenario: "demo", Detector: "cycle", Type: "cycle", Locations: []string{"a"}}
	other := base
	other.Type = "mislocated_file"
	if conflicts.StableID(base) == conflicts.StableID(other) {
		t.Fatal("stable_id must differ when type differs")
	}
}

func TestStableID_IgnoresVolatileFields(t *testing.T) {
	base := conflicts.Conflict{Scenario: "demo", Detector: "cycle", Type: "cycle", Locations: []string{"a"}}
	noisy := base
	noisy.Severity = conflicts.SeverityBlocker
	noisy.ResolutionNote = "note"
	noisy.SuggestedFixes = []conflicts.Fix{{ID: "x"}}
	if conflicts.StableID(base) != conflicts.StableID(noisy) {
		t.Fatal("stable_id should ignore severity/notes/fixes")
	}
}
