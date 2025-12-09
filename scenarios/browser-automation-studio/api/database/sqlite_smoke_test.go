package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestSQLiteBackendSmoke(t *testing.T) {
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "bas-smoke.db")

	// Allow CI toggle to skip sqlite smoke when not desired.
	if skip := os.Getenv("BAS_SKIP_SQLITE_TESTS"); strings.EqualFold(skip, "true") {
		t.Skip("BAS_SKIP_SQLITE_TESTS is set; skipping sqlite smoke test")
	}

	t.Setenv("BAS_DB_BACKEND", "sqlite")
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
