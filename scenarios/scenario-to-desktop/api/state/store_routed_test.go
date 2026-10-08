package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestRoutedStoreWritesTestModeStateOnlyToLeasedRoot(t *testing.T) {
	primaryRoot := t.TempDir()
	testRoot := t.TempDir()
	roots := filerouting.New(storage.Paths{StateDir: primaryRoot})
	if err := roots.InstallTestRoots(storage.Paths{StateDir: testRoot}, "lease-1", time.Minute); err != nil {
		t.Fatalf("install test roots: %v", err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("lease-1") })

	store, err := NewRoutedStore(roots)
	if err != nil {
		t.Fatalf("new routed store: %v", err)
	}
	ctx := database.WithTestMode(context.Background())
	if err := store.Save(ctx, &ScenarioState{ScenarioName: "test-scenario"}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if _, err := os.Stat(filepath.Join(testRoot, "test-scenario.json")); err != nil {
		t.Fatalf("test root state missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(primaryRoot, "test-scenario.json")); !os.IsNotExist(err) {
		t.Fatalf("primary root was written during test mode, stat error = %v", err)
	}
	if stats := roots.LeaseStats(); stats.TestRootWrites != 1 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("lease stats = %+v, want one test-root write and no primary leak", stats)
	}
}
