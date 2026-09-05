package scenarioruntime

import (
	"context"
	"fmt"
	"testing"
)

func newBatchTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(context.Background(), Config{HomeDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// TestBatchReadsChunkPastSQLiteVariableLimit seeds more instances than one
// IN(...) chunk holds and exercises all three batch reads, including that
// missing IDs are absent from the maps rather than zero-valued.
func TestBatchReadsChunkPastSQLiteVariableLimit(t *testing.T) {
	store := newBatchTestStore(t)
	ctx := context.Background()

	const total = batchIDChunkSize + 25
	ids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("inst-%04d", i)
		ids = append(ids, id)
		if _, err := store.CreateInstance(ctx, Instance{
			InstanceID: id,
			Scenario:   "alpha",
			Status:     StatusRunning,
		}); err != nil {
			t.Fatalf("CreateInstance %s: %v", id, err)
		}
	}
	pidA, pidB := 4100, 4200
	if _, err := store.AddProcessRef(ctx, ProcessRef{RefID: "ref-a", InstanceID: ids[0], PID: &pidA, Step: "api"}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
	if _, err := store.AddProcessRef(ctx, ProcessRef{RefID: "ref-b", InstanceID: ids[0], PID: &pidB, Step: "ui"}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
	if _, err := store.UpsertHealthSnapshot(ctx, HealthSnapshot{InstanceID: ids[1], Scenario: "alpha", Status: HealthStatusHealthy}); err != nil {
		t.Fatalf("UpsertHealthSnapshot: %v", err)
	}

	// Query with duplicates, an empty ID, and a missing ID mixed in.
	queryIDs := append(append([]string{}, ids...), ids[0], "", "inst-missing")

	instances, err := store.GetInstances(ctx, queryIDs)
	if err != nil {
		t.Fatalf("GetInstances: %v", err)
	}
	if len(instances) != total {
		t.Fatalf("instances = %d, want %d", len(instances), total)
	}
	if _, ok := instances["inst-missing"]; ok {
		t.Fatal("missing ID must be absent from the map")
	}

	refs, err := store.ListProcessRefsForInstances(ctx, queryIDs)
	if err != nil {
		t.Fatalf("ListProcessRefsForInstances: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("refs map size = %d, want 1 (only ids[0] has refs)", len(refs))
	}
	got := refs[ids[0]]
	want, err := store.ListProcessRefs(ctx, ids[0])
	if err != nil {
		t.Fatalf("ListProcessRefs: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("batched refs = %d, single-instance refs = %d", len(got), len(want))
	}
	for i := range want {
		if got[i].RefID != want[i].RefID {
			t.Fatalf("ref order diverged at %d: batched %q vs single %q", i, got[i].RefID, want[i].RefID)
		}
	}

	health, err := store.GetHealthSnapshots(ctx, queryIDs)
	if err != nil {
		t.Fatalf("GetHealthSnapshots: %v", err)
	}
	if len(health) != 1 || health[ids[1]].Status != HealthStatusHealthy {
		t.Fatalf("health = %#v, want only %s healthy", health, ids[1])
	}
}

func TestBatchReadsEmptyInputReturnEmptyMaps(t *testing.T) {
	store := newBatchTestStore(t)
	ctx := context.Background()
	instances, err := store.GetInstances(ctx, nil)
	if err != nil || len(instances) != 0 {
		t.Fatalf("GetInstances(nil) = %v, %v", instances, err)
	}
	refs, err := store.ListProcessRefsForInstances(ctx, []string{""})
	if err != nil || len(refs) != 0 {
		t.Fatalf("ListProcessRefsForInstances([\"\"]) = %v, %v", refs, err)
	}
	health, err := store.GetHealthSnapshots(ctx, []string{})
	if err != nil || len(health) != 0 {
		t.Fatalf("GetHealthSnapshots([]) = %v, %v", health, err)
	}
}
