package testsqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"test-genie/internal/storage/sqlfiles"
	"test-genie/internal/storage/sqlitedb"
	"test-genie/internal/dbexec"

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

// OpenRouted returns a temporary SQLite database as a *database.RoutedDB —
// the production handle shape after the routed-test-db migration. Use it for
// tests that construct Server/Bootstrapped (which now hold *RoutedDB); the
// same handle still satisfies the dbexec.Executor seam every repository takes.
func OpenRouted(t *testing.T) *database.RoutedDB {
	t.Helper()
	return openRouted(t, false)
}

func openRouted(t *testing.T, includeSeed bool) *database.RoutedDB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test-genie.db")
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          sqlitedb.BuildDSN(path),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open routed sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := applyDomainSchemas(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	_ = includeSeed
	return db
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

	if err := applyDomainSchemas(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	_ = includeSeed
	return db
}

func applyDomainSchemas(db dbexec.Executor) error {
	root := filepath.Join(scenarioRoot(), "api", "internal")
	for _, domain := range []string{"execution", "playbooksclaims", "remediation", "selfhealthsnapshots"} {
		if err := sqlfiles.ExecFile(db, filepath.Join(root, domain, "schema.sql")); err != nil {
			return fmt.Errorf("apply %s schema: %w", domain, err)
		}
	}
	return nil
}

func scenarioRoot() string {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
