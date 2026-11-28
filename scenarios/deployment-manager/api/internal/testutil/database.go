// Package testutil contains test-only infrastructure for deployment-manager.
// Production packages must not import it; no_prod_import_test.go enforces that
// boundary.
package testutil

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func OpenSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:testutil?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type Clock struct{ NowValue time.Time }

func (c Clock) Now() time.Time { return c.NowValue }
