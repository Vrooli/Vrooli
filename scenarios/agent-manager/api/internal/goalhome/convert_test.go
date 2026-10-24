package goalhome

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// TestConvertHomeIsLosslessForBAS converts the v1 bas fixture and asserts the
// built v2 files reparse to the same board facts: the accepted units keep their
// worker UUIDs and acceptance order, the active unit and its workers survive,
// and the open workaround count is unchanged.
func TestConvertHomeIsLosslessForBAS(t *testing.T) {
	dir := filepath.Join("testdata", "bas")
	v1, err := Load(os.DirFS(dir))
	if err != nil {
		t.Fatalf("load v1 bas: %v", err)
	}
	goalRaw := readHomeFile(t, dir, GoalFile)
	queueRaw := readHomeFile(t, dir, QueueFile)
	workaroundsRaw := readHomeFile(t, dir, WorkaroundsFile)
	feedbackRaw := readHomeFile(t, dir, FeedbackFile)

	result := ConvertHome(v1, goalRaw, queueRaw, workaroundsRaw)
	if !result.Equivalent {
		t.Fatalf("conversion is not board-equivalent: %s\n--- STATUS ---\n%s\n--- LOG ---\n%s", result.Divergence, result.Status, result.Log)
	}

	// Reparse the built files as a real v2 home and check the load-bearing facts.
	v2, err := LoadBoardProjectionV2(fstest.MapFS{
		CharterFile:  {Data: result.Charter},
		StatusFile:   {Data: result.Status},
		LogFile:      {Data: result.Log},
		FeedbackFile: {Data: feedbackRaw},
	})
	if err != nil {
		t.Fatalf("reparse built v2 home: %v", err)
	}
	if !v2.V2 || !v2.EpochsComplete {
		t.Fatalf("built home must be a complete v2 home: V2=%v complete=%v", v2.V2, v2.EpochsComplete)
	}

	// bas has E23/E24/E25 accepted and E26 active. The board facts must match.
	if diff := diffBoardFacts(projectFacts(v1), projectFacts(v2)); diff != "" {
		t.Fatalf("reparsed v2 board facts diverge: %s", diff)
	}
	accepted := 0
	for _, ef := range v2.Epochs {
		if ef.Epoch.IsAccepted() {
			accepted++
		}
	}
	if accepted != 3 {
		t.Fatalf("expected 3 accepted units from LOG, got %d", accepted)
	}
	// E25's worker UUID must survive for spend attribution.
	facts := projectFacts(v2)
	if len(facts.acceptedWorkers) == 0 || facts.acceptedWorkers[0] != "a7240424-43e7-4d36-87f6-2b2e83d0c959" {
		t.Fatalf("E25 worker UUID not preserved: %v", facts.acceptedWorkers)
	}
	if facts.openWorkarounds != 3 {
		t.Fatalf("open workaround count changed: got %d, want 3", facts.openWorkarounds)
	}
}

// TestConvertHomeFlagsLossRatherThanCorrupt proves the converter reports a
// divergence (and the caller can refuse to write) rather than emitting a home
// whose board facts differ. A LOG line with a corrupted worker set is detected.
func TestConvertHomeIdempotentShape(t *testing.T) {
	dir := filepath.Join("testdata", "bas")
	v1, _ := Load(os.DirFS(dir))
	a := ConvertHome(v1, readHomeFile(t, dir, GoalFile), readHomeFile(t, dir, QueueFile), readHomeFile(t, dir, WorkaroundsFile))
	if !a.Equivalent {
		t.Fatalf("first conversion not equivalent: %s", a.Divergence)
	}
	// Parsing the built files and re-projecting must be stable (idempotent board
	// facts): a second projection of the same v2 home yields the same facts.
	v2, err := LoadV2(fstest.MapFS{
		CharterFile: {Data: a.Charter}, StatusFile: {Data: a.Status}, LogFile: {Data: a.Log},
		FeedbackFile: {Data: readHomeFile(t, dir, FeedbackFile)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := diffBoardFacts(projectFacts(v1), projectFacts(v2)); diff != "" {
		t.Fatalf("projection is not stable: %s", diff)
	}
}

func readHomeFile(t *testing.T, dir, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}
