package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixtureRun struct {
	target, run, asset string
	age                time.Duration
	bytes              int
}

func writeFixture(t *testing.T, root string, now time.Time, r fixtureRun) string {
	t.Helper()
	dir := filepath.Join(root, r.target, "runs", r.run, r.asset)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stamp := now.Add(-r.age)
	timeline := filepath.Join(dir, timelineFile)
	if err := os.WriteFile(timeline, []byte(strings.Repeat("x", r.bytes)), 0o644); err != nil {
		t.Fatal(err)
	}
	latest := filepath.Join(dir, latestFile)
	if err := os.WriteFile(latest, []byte(`{"run_id":"`+r.run+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{timeline, latest} {
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	return timeline
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func TestPruneTimelinesRemovesOnlyOldUncitedTimelinesAndKeepsSummaries(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	policy := RetentionPolicy{MinAge: 3 * day, KeepPerWorkflow: 1}

	newest := writeFixture(t, root, now, fixtureRun{"hub", "run-new", "case-a", 10 * day, 10})
	old := writeFixture(t, root, now, fixtureRun{"hub", "run-old", "case-a", 20 * day, 10})
	young := writeFixture(t, root, now, fixtureRun{"hub", "run-young", "case-b", 1 * day, 10})
	youngNewest := writeFixture(t, root, now, fixtureRun{"hub", "run-young2", "case-b", 0, 10})
	cited := writeFixture(t, root, now, fixtureRun{"hub", "run-cited", "case-a", 30 * day, 10})
	active := writeFixture(t, root, now, fixtureRun{"hub", "run-active", "case-a", 30 * day, 10})
	pinned := writeFixture(t, root, now, fixtureRun{"hub", "run-pinned", "case-a", 30 * day, 10})
	if err := os.WriteFile(filepath.Join(root, "hub", "runs", "run-pinned", RetainMarker), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(root, "hub", "timeline.json") // outside the run layout
	if err := os.WriteFile(stray, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(stray, now.Add(-90*day), now.Add(-90*day)); err != nil {
		t.Fatal(err)
	}

	res, err := PruneTimelines(root, policy, RetentionInputs{
		Now:        now,
		ActiveRuns: map[string]bool{"run-active": true},
		CitedRuns:  map[string]bool{"run-cited": true},
	})
	if err != nil {
		t.Fatalf("PruneTimelines: %v", err)
	}
	if exists(old) {
		t.Error("old uncited timeline beyond the newest-per-workflow window must be pruned")
	}
	for name, path := range map[string]string{"newest": newest, "young": young, "youngNewest": youngNewest, "cited": cited, "active": active, "pinned": pinned, "stray": stray} {
		if !exists(path) {
			t.Errorf("%s timeline was removed; it must be kept", name)
		}
	}
	if !exists(filepath.Join(filepath.Dir(old), latestFile)) {
		t.Error("latest.json (the evidence reference) must survive timeline retention")
	}
	if len(res.Pruned) != 1 || res.Pruned[0].RunID != "run-old" || res.Pruned[0].Reason != "age" {
		t.Errorf("Pruned = %+v, want only run-old by age", res.Pruned)
	}
	for _, reason := range []string{"cited_evidence", "active_run", "retain_marker", "newest_per_workflow"} {
		if res.Protected[reason] == 0 {
			t.Errorf("Protected[%q] = 0; want it counted (%v)", reason, res.Protected)
		}
	}
}

func TestPruneTimelinesByteBudgetRespectsPressureFloor(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	hour := time.Hour
	policy := RetentionPolicy{MinAge: 72 * hour, KeepPerWorkflow: 0, MaxBytes: 120, PressureMinAge: 24 * hour}

	older := writeFixture(t, root, now, fixtureRun{"hub", "r1", "a", 48 * hour, 100})
	newer := writeFixture(t, root, now, fixtureRun{"hub", "r2", "a", 30 * hour, 100})
	today := writeFixture(t, root, now, fixtureRun{"hub", "r3", "a", 2 * hour, 100})

	res, err := PruneTimelines(root, policy, RetentionInputs{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if exists(older) || exists(newer) {
		t.Error("byte pass must remove timelines past the pressure floor oldest-first while over budget")
	}
	if !exists(today) {
		t.Error("byte pass must never remove a timeline younger than PressureMinAge")
	}
	if !res.OverBudget {
		t.Error("store still over budget after the floor must report OverBudget")
	}
}

func TestCitedRunIDsFindsUUIDsInEvidenceText(t *testing.T) {
	dir := t.TempDir()
	id := "db061437-aa6a-407e-bf39-622654162b6e"
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte("see run-artifacts/hub/runs/"+id+"/case/latest.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	cited, err := CitedRunIDs(dir, filepath.Join(dir, "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if !cited[id] {
		t.Errorf("CitedRunIDs missed %s: %v", id, cited)
	}
}

func TestWriteRetentionReceiptPersistsPrunedPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retention", "run-artifacts-receipt.json")
	in := RetentionResult{Pruned: []PrunedTimeline{{Path: "/x/timeline.json", RunID: "r", Bytes: 5, Reason: "age"}}}
	if err := WriteRetentionReceipt(path, in); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out RetentionResult
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Pruned) != 1 || out.Pruned[0].Path != "/x/timeline.json" {
		t.Errorf("receipt = %+v", out)
	}
}
