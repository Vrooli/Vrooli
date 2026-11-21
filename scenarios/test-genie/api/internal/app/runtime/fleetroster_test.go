package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// The fleet ledger's coverage denominator is only meaningful if the roster
// actually names every scenario on disk. A roster that silently returns the
// scenarios that happened to run makes coverage read 100% by construction.

func TestFleetRosterNamesEveryScenarioDirectory(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Hidden entries and files are not scenarios.
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := fleetRosterFromScenariosRoot(root)(context.Background())
	if err != nil {
		t.Fatalf("roster: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("roster = %v, want the three scenario directories", names)
	}
	for i, want := range []string{"alpha", "beta", "gamma"} {
		if names[i] != want {
			t.Fatalf("roster[%d] = %q, want %q (order must be deterministic)", i, names[i], want)
		}
	}
}

// TestFleetRosterReportsAnErrorRatherThanAnEmptyFleet is the honesty property.
// An unreadable root must not read as "no scenarios exist": the ledger treats a
// roster error as unknown coverage, whereas an empty roster would let coverage
// report 100% of nothing.
func TestFleetRosterReportsAnErrorRatherThanAnEmptyFleet(t *testing.T) {
	names, err := fleetRosterFromScenariosRoot(filepath.Join(t.TempDir(), "does-not-exist"))(context.Background())
	if err == nil {
		t.Fatal("an unreadable scenarios root must report an error, not an empty fleet")
	}
	if len(names) != 0 {
		t.Fatalf("expected no names alongside the error, got %v", names)
	}
}

// TestFleetRosterCoversTheRealFleet checks the wiring against the actual
// repository, because the defect this guards was a denominator of 23 against a
// fleet of 121 — a roster that was resolving to the wrong place, not a roster
// that was computing the wrong answer.
func TestFleetRosterCoversTheRealFleet(t *testing.T) {
	root := "/home/matthalloran8/Vrooli/scenarios"
	if _, err := os.Stat(root); err != nil {
		t.Skip("probe requires the repo checkout")
	}
	names, err := fleetRosterFromScenariosRoot(root)(context.Background())
	if err != nil {
		t.Fatalf("roster: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	dirs := 0
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "" && entry.Name()[0] != '.' {
			dirs++
		}
	}
	// Ephemeral timestamped scenarios age out of the roster deliberately, so
	// the roster may be smaller — but not by much, and never a tiny fraction.
	if len(names) < dirs/2 {
		t.Fatalf("roster named %d of %d scenario directories; the denominator is not the fleet", len(names), dirs)
	}
	t.Logf("roster names %d of %d scenario directories", len(names), dirs)
}
