package cliinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/coreactivation"
)

func TestDirectRootInstallRecordsEnrollmentReadFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.Directory(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Directory(), "enrollment.json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicInstall("missing", filepath.Join(bin, "vrooli")); err == nil {
		t.Fatal("corrupt enrollment must fail closed")
	}
	dir := filepath.Join(home, ".vrooli", "state", "root-rebaseline", "attempts")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected early failure observation: %v %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Events []struct{ Stage, Reason string }
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	last := record.Events[len(record.Events)-1]
	if last.Stage != "terminal" || last.Reason != "failed" {
		t.Fatalf("unexpected events: %+v", record.Events)
	}
}

func TestDirectRootInstallRecordsFailureWithoutChangingOutcome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := AtomicInstall(filepath.Join(t.TempDir(), "missing"), filepath.Join(bin, "vrooli")); err == nil {
		t.Fatal("missing source should fail")
	}
	entries, err := os.ReadDir(filepath.Join(home, ".vrooli", "state", "root-rebaseline", "attempts"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one attempt: %v %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".vrooli", "state", "root-rebaseline", "attempts", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Events []struct {
			Stage  string `json:"stage"`
			Reason string `json:"reason"`
		} `json:"events"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	last := record.Events[len(record.Events)-1]
	if last.Stage != "terminal" || last.Reason != "failed" {
		t.Fatalf("unexpected events: %+v", record.Events)
	}
}
