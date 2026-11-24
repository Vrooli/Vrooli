package filedb

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestOpenClosesAfterLaterCleanupCallbacks(t *testing.T) {
	var dbOpenDuringLaterCleanup bool
	var db *database.RoutedDB

	t.Run("database-lifetime", func(t *testing.T) {
		opened := Open(t, "lifetime.db")
		db = opened
		t.Cleanup(func() {
			dbOpenDuringLaterCleanup = opened.Primary().PingContext(context.Background()) == nil
		})
	})

	if !dbOpenDuringLaterCleanup {
		t.Fatal("a cleanup registered after Open must run while the database is still open")
	}
	if err := db.Primary().PingContext(context.Background()); err == nil {
		t.Fatal("Open left the database open after its test cleanup")
	}
}
