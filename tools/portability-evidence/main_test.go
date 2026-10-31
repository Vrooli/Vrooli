package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitClaimsSortsAndDeduplicates(t *testing.T) {
	got := splitClaims("system-monitor-memory, system-monitor-cpu,system-monitor-memory,,system-monitor-cpu")
	want := []string{"system-monitor-cpu", "system-monitor-memory"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitClaims() = %#v, want %#v", got, want)
	}
}

func TestWriteRecordPublishesValidJSONAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vrooli", "evidence", "native-platform", "minimouse.json")
	want := record{
		SchemaVersion: 1, Kind: "hardware-persistence", HostOS: "darwin", Architecture: "amd64",
		GeneratedAt: "2026-08-25T15:00:00Z", Passed: false, Source: "test", RunID: "run-1",
		Host: "minimouse", Surface: "lifecycle", ArtifactURI: "artifact://run-1",
		Capabilities: []string{"system-monitor-cpu"},
	}
	if err := writeRecord(path, want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got record
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("record = %#v, want %#v", got, want)
	}
	if matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".native-evidence-*.tmp")); err != nil || len(matches) != 0 {
		t.Fatalf("temporary evidence files remain: %v (err=%v)", matches, err)
	}
}

func TestRunRequiresExplicitPassVerdict(t *testing.T) {
	err := run([]string{"--output", filepath.Join(t.TempDir(), "record.json"), "--kind", "hardware-persistence", "--os", "darwin", "--arch", "amd64", "--run-id", "run", "--host", "host", "--surface", "surface", "--artifact-uri", "artifact://run"})
	if err == nil {
		t.Fatal("run accepted a record without an explicit --passed verdict")
	}
}
