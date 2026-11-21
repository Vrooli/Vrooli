package scenarioruntime

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go"
)

func TestRegistryStatePathMatchesRepoContract(t *testing.T) {
	want, err := repocontract.RuntimeHomeEntryPathAtRoot("/home", repocontract.HomeKeyRuntimeDB)
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Join("/home", registryStatePath); got != want {
		t.Fatalf("registryStatePath = %q, repo contract says %q", got, want)
	}
}

func TestSnapshotDatabaseIsConsistentAndRefusesToOverwrite(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	source := filepath.Join(dir, "runtime.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateInstance(ctx, Instance{InstanceID: "snap-fixture", Scenario: "demo"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "snapshot.db")
	if err := SnapshotDatabase(ctx, source, dest); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	restored, err := NewSQLiteStore(ctx, Config{DBPath: dest, ReadOnly: true})
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer restored.Close()
	var count int
	if err := restored.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_instances WHERE instance_id = 'snap-fixture'`).Scan(&count); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if count != 1 {
		t.Fatalf("snapshot lost the row: count = %d", count)
	}
	if err := SnapshotDatabase(ctx, source, dest); err == nil {
		t.Fatal("snapshot overwrote an existing destination")
	}
	if err := SnapshotDatabase(ctx, filepath.Join(dir, "missing.db"), filepath.Join(dir, "other.db")); err == nil {
		t.Fatal("snapshot accepted a missing source")
	}
	_ = store.Close()
}
