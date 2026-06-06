package recovery

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/baselinefloor"

	_ "modernc.org/sqlite"
)

func TestMigrateFastPathWhenNoScripts(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)

	out, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip"})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !out.FastPath {
		t.Fatalf("an engagement with no migrations folder is the fast path, got %+v", out)
	}
	if out.MigrationsDir == "" {
		t.Fatalf("the resolved migrations dir should be reported: %+v", out)
	}
}

func TestMigrateAppliesEngagementScripts(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)

	// Author an ordered script in the engagement's managed migrations folder.
	migDir := svc.Store.MigrationsPath("demo", "wip")
	if err := os.MkdirAll(migDir, 0o750); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "001-init.sql"), []byte("CREATE TABLE t (x INTEGER);"), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	dbPath := filepath.Join(t.TempDir(), "live.db")

	out, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip", DBPath: dbPath})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if out.FastPath || len(out.Applied) != 1 || out.Engine != baselinefloor.EngineSQLite {
		t.Fatalf("expected one applied sqlite migration, got %+v", out)
	}
	db, _ := sql.Open("sqlite", dbPath)
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("INSERT INTO t (x) VALUES (1)"); err != nil {
		t.Fatalf("migrated table should exist: %v", err)
	}
}

func TestMigrateRequiresRef(t *testing.T) {
	svc, _ := newTestService(t, time.Now())
	if _, err := svc.Migrate(MigrateRequest{Scenario: "", Slug: "wip"}); err == nil {
		t.Fatal("missing scenario must error")
	}
}
