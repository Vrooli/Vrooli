package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRecordQualificationObservationPersistsLatestHealthyStart(t *testing.T) {
	root := t.TempDir()
	if err := recordQualificationObservation(root, "whisper"); err != nil {
		t.Fatalf("recordQualificationObservation() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, qualificationObservationRelativePath))
	if err != nil {
		t.Fatalf("read observation ledger: %v", err)
	}
	var observations []resourceQualificationObservation
	if err := json.Unmarshal(data, &observations); err != nil {
		t.Fatalf("decode observation ledger: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observation count = %d, want 1", len(observations))
	}
	observed := observations[0]
	if observed.Resource != "whisper" || observed.HostOS != runtime.GOOS || observed.Architecture != runtime.GOARCH || !observed.HealthPassed || observed.Node == "" || observed.RunID == "" {
		t.Fatalf("unexpected observation: %+v", observed)
	}
	info, err := os.Stat(filepath.Join(root, qualificationObservationRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("observation permissions = %o; want 600", mode)
	}
}

func TestRecordQualificationObservationDoesNotDuplicateCell(t *testing.T) {
	root := t.TempDir()
	if err := recordQualificationObservation(root, "whisper"); err != nil {
		t.Fatal(err)
	}
	if err := recordQualificationObservation(root, "whisper"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, qualificationObservationRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	var observations []resourceQualificationObservation
	if err := json.Unmarshal(data, &observations); err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 {
		t.Fatalf("observation count = %d, want one cell", len(observations))
	}
}
