package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestBackendSmoke(t *testing.T) {
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "bas-smoke.db")

	t.Setenv("BAS_SQLITE_PATH", sqlitePath)
	t.Setenv("BAS_SKIP_DEMO_SEED", "true")

	log := logrus.New()
	db, err := NewConnection(log)
	if err != nil {
		t.Fatalf("failed to connect sqlite backend: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Basic lifecycle: insert and fetch a project.
	projectID := uuid.New()
	_, err = db.Exec(`INSERT INTO projects (id, name, folder_path, created_at, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		projectID.String(), "Smoke Project", "/tmp/smoke")
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM projects`); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 project, got %d", count)
	}

	// Ensure DB file exists on disk.
	if _, err := os.Stat(sqlitePath); err != nil {
		t.Fatalf("expected sqlite file on disk: %v", err)
	}
}

func TestSQLiteDSNIgnoresLegacyFallbackEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("BAS_SQLITE_PATH", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SQLITE_DATABASE_PATH", filepath.Join(home, "legacy-sqlite-root"))
	t.Setenv("VROOLI_DATA", filepath.Join(home, "legacy-vrooli-data"))

	dsn, err := sqliteDSN(nil)
	if err != nil {
		t.Fatalf("sqliteDSN() error = %v", err)
	}
	wantPath := filepath.Join(home, ".local", "share", "vrooli", "browser-automation-studio", "browser-automation-studio.db")
	if !strings.Contains(dsn, wantPath) {
		t.Fatalf("sqliteDSN() = %q, want path containing %q", dsn, wantPath)
	}
	if _, err := os.Stat(filepath.Dir(wantPath)); err != nil {
		t.Fatalf("expected canonical sqlite dir at %s: %v", filepath.Dir(wantPath), err)
	}
}

func TestInitSchemaAddsMissingWorkflowFilePathColumn(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "legacy-bas.db")

	t.Setenv("BAS_SQLITE_PATH", dbPath)
	t.Setenv("BAS_SKIP_DEMO_SEED", "true")

	log := logrus.New()
	db, err := connectSQLite(log)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE workflows (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT NOT NULL,
			folder_path TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create legacy workflows table: %v", err)
	}

	wrapped := &DB{DB: db, log: log}
	if err := wrapped.ensureWorkflowSchemaCompatibility(context.Background()); err != nil {
		t.Fatalf("ensureWorkflowSchemaCompatibility() error = %v", err)
	}

	hasColumn, err := wrapped.columnExists(context.Background(), "workflows", "file_path")
	if err != nil {
		t.Fatalf("columnExists() error = %v", err)
	}
	if !hasColumn {
		t.Fatal("expected workflows.file_path column to be added")
	}
}

func TestInitSchemaAddsMissingExecutionCompatibilityColumns(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "legacy-executions.db")

	t.Setenv("BAS_SQLITE_PATH", dbPath)
	t.Setenv("BAS_SKIP_DEMO_SEED", "true")

	log := logrus.New()
	db, err := connectSQLite(log)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE executions (
			id TEXT PRIMARY KEY,
			workflow_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create legacy executions table: %v", err)
	}

	wrapped := &DB{DB: db, log: log}
	if err := wrapped.ensureExecutionSchemaCompatibility(context.Background()); err != nil {
		t.Fatalf("ensureExecutionSchemaCompatibility() error = %v", err)
	}

	for _, columnName := range []string{"error_message", "result_path", "resumed_from_id"} {
		hasColumn, err := wrapped.columnExists(context.Background(), "executions", columnName)
		if err != nil {
			t.Fatalf("columnExists(%q) error = %v", columnName, err)
		}
		if !hasColumn {
			t.Fatalf("expected executions.%s column to be added", columnName)
		}
	}
}
