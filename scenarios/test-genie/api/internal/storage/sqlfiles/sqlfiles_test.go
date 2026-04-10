package sqlfiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestSplitStatementsSeparatesSemicolonDelimitedSQL(t *testing.T) {
	statements := SplitStatements(`
CREATE TABLE foo (id INTEGER);

INSERT INTO foo VALUES (1);
`)

	if len(statements) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(statements))
	}
	if statements[0] != "\nCREATE TABLE foo (id INTEGER);" {
		t.Fatalf("unexpected first statement: %q", statements[0])
	}
	if statements[1] != "\n\nINSERT INTO foo VALUES (1);" {
		t.Fatalf("unexpected second statement: %q", statements[1])
	}
}

func TestExecFileAppliesStatements(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sqlfiles.db")
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          "file:" + dbPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("database.Connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	scriptPath := filepath.Join(t.TempDir(), "schema.sql")
	if err := os.WriteFile(scriptPath, []byte(`
CREATE TABLE foo (id INTEGER);
INSERT INTO foo VALUES (1);
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	if err := ExecFile(db, scriptPath); err != nil {
		t.Fatalf("ExecFile returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM foo`).Scan(&count); err != nil {
		t.Fatalf("count foo rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}
