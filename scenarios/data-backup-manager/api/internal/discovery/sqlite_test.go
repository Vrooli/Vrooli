package discovery_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/discovery"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

// newDismissalDB returns a fresh sqlite handle with the discovery schema applied.
func newDismissalDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(discovery.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

func TestSQLiteDismissalStore_RoundTripAndIdempotency(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := discovery.NewSQLiteDismissalStore(newDismissalDB(t), clk)

	dismissed, err := store.IsDismissed(ctx, "abc")
	if err != nil {
		t.Fatalf("IsDismissed: %v", err)
	}
	if dismissed {
		t.Fatal("expected fresh id to be not dismissed")
	}

	if err := store.Dismiss(ctx, "abc", "suggestion"); err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	// Re-dismissing the same id is idempotent (ON CONFLICT DO NOTHING).
	if err := store.Dismiss(ctx, "abc", "suggestion"); err != nil {
		t.Fatalf("re-Dismiss: %v", err)
	}

	dismissed, err = store.IsDismissed(ctx, "abc")
	if err != nil {
		t.Fatalf("IsDismissed after dismiss: %v", err)
	}
	if !dismissed {
		t.Fatal("expected id to be dismissed after Dismiss")
	}

	// An unrelated id is unaffected.
	other, err := store.IsDismissed(ctx, "def")
	if err != nil {
		t.Fatalf("IsDismissed other: %v", err)
	}
	if other {
		t.Fatal("unrelated id should not be dismissed")
	}
}
