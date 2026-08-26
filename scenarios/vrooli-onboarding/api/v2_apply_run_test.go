package main

import "testing"

func TestSelectionDigestChangesWhenConsentPlanChanges(t *testing.T) {
	base := []applyItem{{ID: "safeguard:firewall", Kind: "safeguard", Name: "firewall"}}
	withResource := append(append([]applyItem(nil), base...), applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"})
	if selectionDigest(base) == selectionDigest(withResource) {
		t.Fatal("selection digest ignored a plan item")
	}
}

func TestApplyRunPersistsPendingItems(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	run := applyRun{ID: "apply-test", Status: "pending", Items: []applyItemResult{{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: "pending"}}}
	storeApplyRun(run)
	loaded, ok := applyRunSnapshot(run.ID)
	if !ok || loaded.Items[0].Outcome != "pending" {
		t.Fatalf("persisted run = %#v, found=%v", loaded, ok)
	}
}

// The run is executed by a detached process, so this process's map is not a
// cache of the run -- it is a snapshot of how the run looked when this process
// last touched it, which for an accepted run is the moment before any work
// happened. Preferring it served "pending" for the entire life of a run that
// was progressing on disk the whole time, and the wizard polling for a terminal
// status waited forever for one that had already been written.
func TestApplyRunSnapshotPrefersTheExecutingProcessesRecord(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	run := applyRun{
		ID:     "apply-detached-runner",
		Status: "pending",
		Items:  []applyItemResult{{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: "pending"}},
	}
	storeApplyRun(run)

	// Stand in for the detached runner: it advances the persisted record and
	// has no way to reach this process's memory.
	advanced := run
	advanced.Status = "applied"
	advanced.RunnerPID = 4242
	advanced.Items = []applyItemResult{{applyItem: run.Items[0].applyItem, Outcome: "applied"}}
	if err := persistApplyRun(advanced); err != nil {
		t.Fatalf("persist the runner's record: %v", err)
	}

	observed, ok := applyRunSnapshot(run.ID)
	if !ok {
		t.Fatal("snapshot lost a run that exists on disk")
	}
	if observed.Status != "applied" {
		t.Fatalf("snapshot status = %q, want applied; the API is serving its own stale copy", observed.Status)
	}
	if observed.Items[0].Outcome != "applied" {
		t.Fatalf("snapshot item outcome = %q, want applied", observed.Items[0].Outcome)
	}
}

// Memory still answers before the first persisted write lands, and on a host
// where the state file cannot be read back.
func TestApplyRunSnapshotFallsBackToMemory(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	run := applyRun{ID: "apply-memory-only", Status: "pending"}
	applyRuns.Lock()
	applyRuns.items[run.ID] = run
	applyRuns.Unlock()

	observed, ok := applyRunSnapshot(run.ID)
	if !ok || observed.Status != "pending" {
		t.Fatalf("snapshot = %#v, found=%v; want the in-memory run", observed, ok)
	}
}
