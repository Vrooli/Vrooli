package testsqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"test-genie/internal/playbooksclaims"
	"test-genie/internal/storage/sqlfiles"
	"test-genie/internal/storage/sqlitedb"

	"github.com/vrooli/api-core/database"
	// Register modernc.org/sqlite as the pure-Go "sqlite" driver.
	_ "modernc.org/sqlite"
)

// Open returns a temporary SQLite database initialized with Test Genie's schema.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	return open(t, false)
}

// OpenWithSeed returns a temporary SQLite database initialized with schema and seed data.
func OpenWithSeed(t *testing.T) *sql.DB {
	t.Helper()
	return open(t, true)
}

func open(t *testing.T, includeSeed bool) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test-genie.db")
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          sqlitedb.BuildDSN(path),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := sqlfiles.ExecFile(db, filepath.Join(scenarioRoot(), "initialization", "sqlite", "schema.sql")); err != nil {
		t.Fatalf("apply sqlite schema: %v", err)
	}
	if _, err := db.Exec(playbooksclaims.Schema()); err != nil {
		t.Fatalf("apply playbooksclaims schema: %v", err)
	}
	if includeSeed {
		if err := sqlfiles.ExecFile(db, filepath.Join(scenarioRoot(), "initialization", "sqlite", "seed.sql")); err != nil {
			t.Fatalf("apply sqlite seed: %v", err)
		}
	}
	return db
}

func scenarioRoot() string {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
