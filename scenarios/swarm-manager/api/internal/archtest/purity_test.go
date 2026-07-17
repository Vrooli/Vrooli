package archtest

import (
	"os"
	"path/filepath"
	"testing"
)

// modesRoot is the scenario's authored modes/ directory relative to this
// package (internal/archtest).
const modesRoot = "../../../modes"

// TestAuthoredModeIDsIncludeSentinelAndShippedModes grounds the derived
// forbidden vocabulary: the sentinel is always present, and the shipped
// catalog contributes every real mode folder (15 at cutover; the count
// assertion is >= so authoring a new mode extends rather than breaks this).
func TestAuthoredModeIDsIncludeSentinelAndShippedModes(t *testing.T) {
	ids, err := AuthoredModeIDs(modesRoot)
	if err != nil {
		t.Fatalf("AuthoredModeIDs: %v", err)
	}
	want := map[string]bool{"item-level": false, "holistic-loop": false, "phased-plan-drain": false, "execution-drain": false}
	for _, id := range ids {
		if _, ok := want[id]; ok {
			want[id] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("AuthoredModeIDs is missing %q", id)
		}
	}
	if len(ids) < 16 { // 15 shipped modes + the sentinel
		t.Errorf("AuthoredModeIDs returned %d ids, want at least 16 (15 shipped modes + sentinel)", len(ids))
	}
}

// TestModeNamePurityScannerFiresOnViolation red-proofs the shared purity
// scanner without committing a violation to any production package: a
// synthetic package directory containing a mode-name reference must be
// flagged, and a clean one must not. This is the same primitive
// RequireNoModeNameBranches runs against opsrunner/opsbridge/opscatalog, so a
// real regression there would fail exactly like the synthetic one here.
func TestModeNamePurityScannerFiresOnViolation(t *testing.T) {
	dir := t.TempDir()
	violating := "package sample\n\n// route holistic-loop specially\nvar mode = \"item-level\"\n"
	if err := os.WriteFile(filepath.Join(dir, "violating.go"), []byte(violating), 0o644); err != nil {
		t.Fatal(err)
	}
	// Test files are exempt (they may name modes as fixtures).
	if err := os.WriteFile(filepath.Join(dir, "exempt_test.go"), []byte("package sample\nvar x = \"holistic-loop\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := ScanPackageDirForTerms(dir, []string{"holistic-loop", "item-level", "phased-plan-drain"})
	if err != nil {
		t.Fatalf("scan synthetic dir: %v", err)
	}
	if len(hits["holistic-loop"]) != 1 || len(hits["item-level"]) != 1 {
		t.Fatalf("scanner must flag the synthetic mode references exactly once each, got %v", hits)
	}
	if len(hits["phased-plan-drain"]) != 0 {
		t.Fatalf("scanner flagged a term the synthetic source does not contain: %v", hits)
	}
}
