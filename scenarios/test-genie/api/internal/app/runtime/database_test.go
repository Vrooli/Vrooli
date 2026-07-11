package runtime

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/storage/sqlfiles"
	"test-genie/internal/storage/sqlitedb"

	_ "modernc.org/sqlite"
)

func TestResolveInitializationFileSuccess(t *testing.T) {
	got, err := resolveInitializationFile("schema.sql")
	if err != nil {
		t.Fatalf("resolveInitializationFile returned error: %v", err)
	}

	if filepath.Base(got) != "schema.sql" {
		t.Fatalf("expected schema.sql path, got %s", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("expected resolved schema path to exist: %v", err)
	}
	if filepath.Base(filepath.Dir(got)) != initializationDialectDir {
		t.Fatalf("expected sqlite initialization directory, got %s", filepath.Dir(got))
	}
	if filepath.Base(filepath.Dir(filepath.Dir(got))) != "initialization" {
		t.Fatalf("expected initialization root in resolved path, got %s", got)
	}
}

func TestResolveInitializationFileErrorWhenMissing(t *testing.T) {
	_, err := resolveInitializationFile("missing.sql")
	if err == nil {
		t.Fatal("expected error when initialization file is missing")
	}
}

func TestExecSQLFileRunsStatements(t *testing.T) {
	db := openSQLite(t)

	tmp := t.TempDir()
	sqlPath := filepath.Join(tmp, "script.sql")
	sqlContent := `
-- comment
CREATE TABLE foo (
    id INT
);

INSERT INTO foo VALUES (1);
`
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0o644); err != nil {
		t.Fatalf("failed to write sql script: %v", err)
	}

	if err := sqlfiles.ExecFile(db, sqlPath); err != nil {
		t.Fatalf("ExecFile returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM foo`).Scan(&count); err != nil {
		t.Fatalf("count foo rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 inserted row, got %d", count)
	}
}

func TestExecSQLFilePropagatesExecErrors(t *testing.T) {
	db := openSQLite(t)

	tmp := t.TempDir()
	sqlPath := filepath.Join(tmp, "script.sql")
	if err := os.WriteFile(sqlPath, []byte("INSERT INTO missing_table VALUES (1);"), 0o644); err != nil {
		t.Fatalf("failed to write sql script: %v", err)
	}

	err := sqlfiles.ExecFile(db, sqlPath)
	if err == nil {
		t.Fatal("expected ExecFile to return error")
	}
}

func TestApplySchemaCreatesDomainTables(t *testing.T) {
	db := openSQLite(t)

	if err := ApplySchema(db, true); err != nil {
		t.Fatalf("ApplySchema returned error: %v", err)
	}

	assertTableExists(t, db, "suite_executions")
}

func TestApplySchemaWithoutSeedStillCreatesDomainTables(t *testing.T) {
	db := openSQLite(t)

	if err := ApplySchema(db, false); err != nil {
		t.Fatalf("ApplySchema returned error: %v", err)
	}

	assertTableExists(t, db, "suite_executions")
}

func TestEnsureDatabaseSchemaExecutesSchema(t *testing.T) {
	db := openSQLite(t)

	if err := ensureDatabaseSchema(db); err != nil {
		t.Fatalf("ensureDatabaseSchema returned error: %v", err)
	}

	assertTableExists(t, db, "suite_executions")
}

func openSQLite(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "runtime-test.db")
	db, err := sql.Open("sqlite", sqlitedb.BuildDSN(dbPath))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Ping(); err != nil {
		t.Fatalf("ping sqlite: %v", err)
	}
	return db
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected table %s to exist", table)
	}
	if err != nil {
		t.Fatalf("query sqlite_master for %s: %v", table, err)
	}
}
