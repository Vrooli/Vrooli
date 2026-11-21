package database_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
)

func TestContextPartitionRetainsExactLeaseGeneration(t *testing.T) {
	primary := database.ContextPartition(t.Context())
	if primary != database.ContextPartition(nil) || primary == database.ContextPartition(database.WithTestMode(t.Context())) {
		t.Fatal("primary and unpinned test traffic share an event partition")
	}
	var prior database.RoutingPartition
	for owner := range 2 {
		db, err := openSQLitePool(t, filepath.Join(t.TempDir(), "primary.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		for generation := range 2 {
			if err := db.InstallTestPool(t.Context(), "file:"+filepath.Join(t.TempDir(), "leased.db"), "same-lease", time.Minute); err != nil {
				t.Fatal(err)
			}
			ctx, err := db.PinTestPool(t.Context(), "same-lease")
			if err != nil {
				t.Fatal(err)
			}
			key := database.ContextPartition(ctx)
			if key == primary || key == prior {
				t.Fatalf("owner/generation alias: owner=%d generation=%d", owner, generation)
			}
			ctx, cancel := context.WithCancel(ctx)
			cancel()
			if key != database.ContextPartition(context.WithoutCancel(ctx)) {
				t.Fatal("detaching producer cancellation changed storage ownership")
			}
			if err := db.ClearTestPool("same-lease"); err != nil {
				t.Fatal(err)
			}
			if key != database.ContextPartition(ctx) {
				t.Fatal("cleared context was reclassified as another storage owner")
			}
			prior = key
		}
	}
}
