package runindex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingIndexIsEmptyNotError(t *testing.T) {
	records, err := Load(filepath.Join(t.TempDir(), "missing-index.json"))
	if err != nil {
		t.Fatalf("missing index: %v", err)
	}
	if records != nil {
		t.Fatalf("missing index should yield nil records, got %v", records)
	}
}

func BenchmarkLoad(b *testing.B) {
	dir := b.TempDir()
	indexPath := filepath.Join(dir, "runs.index.json")
	records := make([]RunRecord, 250)
	for i := range records {
		records[i] = RunRecord{
			RunID:     fmt.Sprintf("20260827-%06d-benchmark", i),
			Scenario:  "demo",
			StartedAt: time.Unix(int64(i), 0).UTC(),
			Status:    StatusPassed,
			Phases:    []PhaseRecord{{Name: "unit", Status: "passed"}, {Name: "docs", Status: "passed"}},
		}
	}
	payload, err := json.Marshal(records)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(indexPath, payload, 0o644); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(indexPath); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLoadSortsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "coverage"), 0o755); err != nil {
		t.Fatal(err)
	}
	const index = `[
  {"run_id": "20260601-000000-aaaaaaaa", "scenario": "demo", "started_at": "2026-06-01T00:00:00Z", "status": "passed", "diagnostics": {}},
  {"run_id": "20260603-000000-cccccccc", "scenario": "demo", "started_at": "2026-06-03T00:00:00Z", "status": "passed", "diagnostics": {}},
  {"run_id": "20260602-000000-bbbbbbbb", "scenario": "demo", "started_at": "2026-06-02T00:00:00Z", "status": "failed", "diagnostics": {}}
]`
	indexPath := filepath.Join(dir, "coverage", "runs.index.json")
	if err := os.WriteFile(indexPath, []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := Load(indexPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	wantOrder := []string{"20260603-000000-cccccccc", "20260602-000000-bbbbbbbb", "20260601-000000-aaaaaaaa"}
	for i, want := range wantOrder {
		if records[i].RunID != want {
			t.Fatalf("records[%d] = %q, want %q (newest-first)", i, records[i].RunID, want)
		}
	}
}

func TestLoadMalformedIndexErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "coverage"), 0o755); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(dir, "coverage", "runs.index.json")
	if err := os.WriteFile(indexPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(indexPath); err == nil {
		t.Fatal("malformed index must error, not silently yield zero records")
	}
}

func TestSortNewestFirstTiebreaksOnRunID(t *testing.T) {
	at := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	records := []RunRecord{
		{RunID: "a", StartedAt: at},
		{RunID: "b", StartedAt: at},
	}
	SortNewestFirst(records)
	if records[0].RunID != "b" {
		t.Fatalf("equal StartedAt must tiebreak by RunID desc, got %q first", records[0].RunID)
	}
}

func TestPhaseStatus(t *testing.T) {
	r := RunRecord{Phases: []PhaseRecord{{Name: "unit", Status: "passed"}, {Name: "docs", Status: "failed"}}}
	m := r.PhaseStatus()
	if m["unit"] != "passed" || m["docs"] != "failed" {
		t.Fatalf("PhaseStatus = %v", m)
	}
}
