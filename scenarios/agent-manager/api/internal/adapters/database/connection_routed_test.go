package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestDBRoutesTestModeOperationsToLeasedPool(t *testing.T) {
	ctx := context.Background()
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	routed, err := coredb.Open(ctx, coredb.Config{
		Driver:       coredb.DriverSQLite,
		DSN:          "file:" + primaryPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open primary: %v", err)
	}
	t.Cleanup(func() { _ = routed.Close() })

	db := NewRoutedDB(routed, nil)
	if _, err := db.ExecContext(ctx, `CREATE TABLE route_marker (value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create primary marker: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO route_marker(value) VALUES ('primary')`); err != nil {
		t.Fatalf("insert primary marker: %v", err)
	}

	testPath := filepath.Join(t.TempDir(), "test.db")
	if err := routed.InstallTestPool(ctx, "file:"+testPath, "lease-test", time.Minute); err != nil {
		t.Fatalf("install test pool: %v", err)
	}
	t.Cleanup(func() { _ = routed.ClearTestPool("lease-test") })
	testCtx := coredb.WithTestMode(ctx)
	if _, err := db.ExecContext(testCtx, `CREATE TABLE route_marker (value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create test marker: %v", err)
	}
	if _, err := db.ExecContext(testCtx, `INSERT INTO route_marker(value) VALUES ('test')`); err != nil {
		t.Fatalf("insert test marker: %v", err)
	}

	var primary, test string
	if err := db.GetContext(ctx, &primary, `SELECT value FROM route_marker`); err != nil {
		t.Fatalf("read primary marker: %v", err)
	}
	if err := db.GetContext(testCtx, &test, `SELECT value FROM route_marker`); err != nil {
		t.Fatalf("read test marker: %v", err)
	}
	if primary != "primary" || test != "test" {
		t.Fatalf("routed markers = primary %q, test %q; want isolated pools", primary, test)
	}
	stats := routed.LeaseStats()
	if stats.TestPoolRequests == 0 || stats.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("lease stats = %+v; want test requests and no primary fallthrough", stats)
	}
}
