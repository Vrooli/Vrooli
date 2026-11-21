package sqlfiles

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"test-genie/internal/dbexec"
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

type cancelingSQLExecutor struct {
	dbexec.Executor
	expected context.Context
	cancel   context.CancelFunc
	calls    int
}

func (e *cancelingSQLExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if ctx != e.expected {
		return nil, errors.New("statement lost its caller context")
	}
	e.calls++
	e.cancel()
	return nil, nil
}
func TestExecFileContextStopsBeforeFollowingStatement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.sql")
	if err := os.WriteFile(path, []byte("SELECT 1; SELECT 2;"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	executor := &cancelingSQLExecutor{expected: ctx, cancel: cancel}
	if err := ExecFileContext(ctx, executor, path); !errors.Is(err, context.Canceled) || executor.calls != 1 {
		t.Fatal("canceled migration dispatched another statement", executor.calls, err)
	}
	if err := ExecFileContext(ctx, executor, "missing.sql"); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled migration performed file work", err)
	}
}
