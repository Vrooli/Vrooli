package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

func TestActivationOutcomeObservationsBindTerminalResults(t *testing.T) {
	home := t.TempDir()
	id := strings.Repeat("a", 32)
	observeActivationOutcome(home, coreactivation.Record{ID: id, Phase: coreactivation.Verified}, nil)
	observeActivationOutcome(home, coreactivation.Record{ID: id, Phase: coreactivation.Verified}, nil)
	observeActivationOutcome(home, coreactivation.Record{ID: id, Phase: coreactivation.FailedRestored}, nil)
	observeActivationOutcome(home, coreactivation.Record{ID: id, Phase: coreactivation.Publishing}, nil)
	observeActivationOutcome(home, coreactivation.Record{ID: id, Phase: coreactivation.Verified}, errors.New("commit uncertain"))
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(state, "root-rebaseline", "attempts")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 3 {
		t.Fatalf("observations: %v %v", entries, err)
	}
	counts := map[string]int{}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			OperationID string                           `json:"operation_id"`
			Events      []struct{ Stage, Reason string } `json:"events"`
		}
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if got.OperationID != id || len(got.Events) != 2 || got.Events[1].Stage != "terminal" {
			t.Fatalf("bad outcome: %s", data)
		}
		counts[got.Events[1].Reason]++
	}
	if counts["verified"] != 2 || counts["failed_restored"] != 1 {
		t.Fatalf("unexpected observations: %v", counts)
	}
}
