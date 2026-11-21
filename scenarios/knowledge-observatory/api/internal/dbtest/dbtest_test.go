package dbtest_test

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/modules"
)

// TestNewAppliesEverySchema is the guard on the test helper itself: if New
// silently failed to apply schemas, every repository test would pass against an
// empty database and prove nothing.
func TestNewAppliesEverySchema(t *testing.T) {
	db := dbtest.New(t, modules.AllSchemas()...)

	var tables int
	err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).
		Scan(&tables)
	if err != nil {
		t.Fatalf("count tables: %v", err)
	}
	// One per table owned by a domain; see docs/internal/STORAGE_AUDIT.md §2.
	if tables != 13 {
		t.Errorf("created %d tables, want the 13 owned across all domains", tables)
	}

	var views int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'view'`).Scan(&views); err != nil {
		t.Fatalf("count views: %v", err)
	}
	if views != 1 {
		t.Errorf("created %d views, want dashboard_metrics", views)
	}
}

// TestNewIsIsolatedPerCall matters because the repository tests assume a clean
// database; a shared file would make them order-dependent.
func TestNewIsIsolatedPerCall(t *testing.T) {
	first := dbtest.New(t, modules.AllSchemas()...)
	second := dbtest.New(t, modules.AllSchemas()...)
	ctx := context.Background()

	if _, err := first.ExecContext(ctx,
		`INSERT INTO quality_metrics (id, collection_name) VALUES ('x', 'c')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var n int
	if err := second.QueryRowContext(ctx, `SELECT COUNT(*) FROM quality_metrics`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("second database saw %d rows from the first; databases must be isolated", n)
	}
}

// TestNewIsIdempotentAcrossReapply proves the schemas are safe to apply on every
// boot, which is the contract EnsureSchemas relies on.
func TestNewIsIdempotentAcrossReapply(t *testing.T) {
	db := dbtest.New(t, modules.AllSchemas()...)
	if err := apidb.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		t.Fatalf("re-applying every schema must be a no-op: %v", err)
	}
}
