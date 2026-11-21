package process

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStaleRecordCleanupCannotEraseReplacementGeneration(t *testing.T) {
	home := t.TempDir()
	old := Record{PID: 42, Generation: "old", Incarnation: "boot:old", StartedAt: time.Now().UTC()}
	if err := WriteScenarioRecord(home, "fixture", "api", old); err != nil {
		t.Fatal(err)
	}
	observed, err := ReadScenarioRecords(home, "fixture")
	if err != nil || len(observed) != 1 {
		t.Fatalf("initial record: %v %v", observed, err)
	}
	newer := old
	newer.Generation, newer.Incarnation = "new", "boot:new"
	if err := WriteScenarioRecord(home, "fixture", "api", newer); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveScenarioRecordIfMatches(home, "fixture", "api", observed[0]); err != nil || removed {
		t.Fatalf("stale snapshot retired a newer generation: removed=%v err=%v", removed, err)
	}
	current, err := ReadScenarioRecords(home, "fixture")
	if err != nil || len(current) != 1 || current[0].Generation != "new" {
		t.Fatalf("replacement lost: %v %v", current, err)
	}
	dir, err := ScenarioProcessDir(home, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "api.pid")); err != nil || string(data) != "42\n" {
		t.Fatalf("replacement PID sidecar changed: %q %v", data, err)
	}
	if removed, err := RemoveScenarioRecordIfMatches(home, "fixture", "api", current[0]); err != nil || !removed {
		t.Fatalf("exact retirement refused: %v %v", removed, err)
	}
	for _, name := range []string{"api.json", "api.pid"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("exact retirement retained %s: %v", name, err)
		}
	}
}
