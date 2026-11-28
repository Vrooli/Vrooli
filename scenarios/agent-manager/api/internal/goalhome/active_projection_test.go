package goalhome

import (
	"strings"
	"testing"
)

func TestBoardProjectionWithDuplicateActiveUnitIsUnknown(t *testing.T) {
	files := fixtureFiles(t, "v2-bas", map[string]func(string) string{
		StatusFile: func(status string) string {
			first := "- unit: E26  status: acceptance review  workers: e8c322ca-72e7-434c-9d37-1d2cf853961f"
			duplicate := "- unit: E26  status: implementation  workers: 11111111-1111-4111-8111-111111111111"
			return strings.Replace(status,
				"- unit: E26  status: acceptance review (BAS-FB-064 gate amendment)  workers: e8c322ca-72e7-434c-9d37-1d2cf853961f",
				first+"\n"+duplicate, 1)
		},
	})

	home, err := LoadBoardProjectionV2(files)
	if err != nil {
		t.Fatalf("LoadBoardProjectionV2: %v", err)
	}
	if home.EpochsComplete {
		t.Fatal("duplicate active identity must make the board projection partial")
	}

	statusMarkedIncomplete := false
	for _, file := range home.IncompleteEpochFiles {
		if file == StatusFile {
			statusMarkedIncomplete = true
			break
		}
	}
	if !statusMarkedIncomplete {
		t.Fatalf("partial projection must name %s as incomplete; got %v", StatusFile, home.IncompleteEpochFiles)
	}

	// LOG remains an independent source of accepted facts. A bad active roster
	// must not erase those facts or let any STATUS active candidate through.
	if len(home.Epochs) != 3 {
		t.Fatalf("projected units = %d, want only 3 accepted LOG units", len(home.Epochs))
	}
	wantAccepted := []string{"E23", "E24", "E25"}
	for i, unit := range home.Epochs {
		if !unit.Epoch.IsAccepted() || unit.ID() != wantAccepted[i] {
			t.Fatalf("projected unit %d = %s (accepted=%t), want accepted %s", i, unit.ID(), unit.Epoch.IsAccepted(), wantAccepted[i])
		}
	}

	found := false
	for _, finding := range Lint(home).Findings {
		if finding.Blocking && finding.File == StatusFile && strings.Contains(finding.Detail, "E26") && strings.Contains(strings.ToLower(finding.Detail), "duplicate") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("lint must report a blocking STATUS.md finding that names duplicate unit E26")
	}
}
