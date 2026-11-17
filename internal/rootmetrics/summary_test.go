package rootmetrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarySeparatesIncompleteUnknownAndDeduplicatesOutcomes(t *testing.T) {
	home := t.TempDir()
	admission := Begin(home, "install")
	id := strings.Repeat("a", 32)
	admission.Operation(id)
	admission.Event("admission_approved", "")
	admission.Event("terminal", "dispatched")
	observer := Begin(home, "activation_observation")
	observer.Operation(id)
	observer.Event("terminal", "verified")
	duplicate := Begin(home, "activation_observation")
	duplicate.Operation(id)
	duplicate.Event("terminal", "verified")
	Begin(home, "rebaseline") // an interrupted or still-running attempt
	dir := filepath.Dir(admission.path)
	unsupported := Begin(home, "install")
	data, err := os.ReadFile(unsupported.path)
	if err != nil {
		t.Fatal(err)
	}
	var r rootAttemptRecord
	if err = json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	r.Schema = 99
	data, _ = json.Marshal(r)
	if err = os.WriteFile(unsupported.path, data, 0600); err != nil {
		t.Fatal(err)
	}
	malformed := Begin(home, "install")
	if err = os.WriteFile(malformed.path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Summarize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Files != 6 || s.Valid != 4 || s.Unsupported != 1 || s.Malformed != 1 || s.Incomplete != 1 || s.Operations != 1 || s.OperationOutcomes["verified"] != 1 || s.Terminals["activation_observation/verified"] != 2 {
		t.Fatalf("unexpected summary: %+v", s)
	}
	conflict := Begin(home, "activation_observation")
	conflict.Operation(id)
	conflict.Event("terminal", "restored")
	s, err = Summarize(dir)
	if err != nil || s.ConflictingOperations != 1 || s.OperationOutcomes["verified"] != 0 {
		t.Fatalf("conflicting observations hidden: %+v %v", s, err)
	}
}

func TestSummaryReportsAttributionAndChangedOwners(t *testing.T) {
	home := t.TempDir()
	a := Begin(home, "install")
	a.record.Attribution = "compared"
	a.record.Owners = []string{"alpha", "beta"}
	a.Event("admission_refused", "state_contract_changed")
	b := Begin(home, "rebaseline")
	b.record.Attribution = "candidate_unavailable"
	b.Event("terminal", "failed")
	c := Begin(home, "install")
	c.record.Attribution = "compared"
	c.record.Owners = []string{"/private"}
	c.Event("terminal", "failed")
	s, err := Summarize(filepath.Dir(a.path))
	if err != nil || s.Attributions["compared"] != 1 || s.Attributions["candidate_unavailable"] != 1 || s.ChangedOwners["alpha"] != 1 || s.ChangedOwners["beta"] != 1 || s.InvalidAttribution != 1 {
		t.Fatalf("attribution: %+v %v", s, err)
	}
}
