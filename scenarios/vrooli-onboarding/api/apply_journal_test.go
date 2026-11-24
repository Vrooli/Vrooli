package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyJournalIDIsBoundToSafeFilename(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	for _, id := range []string{"../outside", `..\outside`, "", " padded"} {
		if _, err := applyRunPath(id); err == nil {
			t.Errorf("applyRunPath(%q) unexpectedly accepted an unsafe ID", id)
		}
	}

	path, err := applyRunPath("different-file")
	if err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Dir(path)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	misfiled := `{"id":"actual-record","status":"indeterminate","items":[]}`
	writeFixtureFile(t, filepath.Join(runDir, "different-file.json"), misfiled)
	if _, err := loadPersistedApplyRun("different-file"); err == nil || !strings.Contains(err.Error(), "mismatched id") {
		t.Fatalf("mismatched filename/body ID error = %v, want identity rejection", err)
	}
	if _, err := readApplyRunJournals(); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("misfiled recovery journal error = %v, want fail-closed rejection", err)
	}
}

func TestApplyRecoveryRejectsUnknownJournalLifecycleValues(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	for _, testCase := range []struct {
		id   string
		json string
	}{
		{id: "unknown-status", json: `{"run_id":"unknown-status","status":"future_status","items":[{"outcome":"applying"}]}`},
		{id: "unknown-outcome", json: `{"run_id":"unknown-outcome","status":"indeterminate","items":[{"outcome":"future_outcome"}]}`},
	} {
		t.Run(testCase.id, func(t *testing.T) {
			path, err := applyRunPath(testCase.id)
			if err != nil {
				t.Fatal(err)
			}
			writeFixtureFile(t, path, testCase.json)
			if _, err := loadPersistedApplyRun(testCase.id); err == nil {
				t.Fatal("loaded unknown durable lifecycle value")
			}
			if err := checkApplyReplayEligibility("local", nil); err == nil || !strings.Contains(err.Error(), "unreadable") {
				t.Fatalf("replay eligibility error = %v, want fail-closed unknown journal", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestApplyJournalControlsPreserveLatestProgress(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	run := applyRun{
		ID: "apply-journal-merge", Target: "local", Status: applyRunApplying,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Items: []applyItemResult{{
			applyItem: applyItem{ID: "resource:alpha", Kind: "resource", Name: "alpha"},
			Outcome:   applyItemPending,
		}},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatal(err)
	}

	// Model a runner snapshot held before a control-plane cancellation.
	stale, err := loadPersistedApplyRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	stale.Items[0].Outcome = applyItemApplied
	if err := updateApplyRun(stale); err != nil {
		t.Fatal(err)
	}
	if _, err := mutatePersistedApplyRun(run.ID, func(latest *applyRun) error {
		latest.CancelRequested = true
		latest.RunnerPID = 123
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// A stale progress write may carry newer item state but must not clear the
	// monotonic cancellation intent written after that snapshot was read.
	stale.Items[0].Outcome = applyItemApplied
	if err := updateApplyRun(stale); err != nil {
		t.Fatal(err)
	}
	if err := restampApplyRunHeartbeat(run.ID); err != nil {
		t.Fatal(err)
	}

	persisted, err := loadPersistedApplyRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Items[0].Outcome != applyItemApplied {
		t.Fatalf("latest item result was overwritten: %q", persisted.Items[0].Outcome)
	}
	if !persisted.CancelRequested {
		t.Fatal("cancellation intent was lost during heartbeat")
	}
	if persisted.Heartbeat == "" {
		t.Fatal("heartbeat was not written")
	}
	if persisted.Status != applyRunApplying {
		t.Fatalf("status changed during control updates: %q", persisted.Status)
	}
}
