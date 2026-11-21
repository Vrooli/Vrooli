package recovery

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
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

// writeEngagementScript authors one ordered migration in the engagement folder.
func writeEngagementScript(t *testing.T, svc Service, scenario, slug, name, sql string) {
	t.Helper()
	migDir := svc.Store.MigrationsPath(scenario, slug)
	if err := os.MkdirAll(migDir, 0o750); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migDir, name), []byte(sql), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

// withHome points a service's namespace resolution at a temp home so the
// variant-aware data dir resolves to a writable location, and returns the
// resolved live data dir for the scenario.
func withHome(t *testing.T, svc *Service, scenario string) string {
	t.Helper()
	home := t.TempDir()
	svc.HomeDir = func() (string, error) { return home, nil }
	ns, err := svc.Namespace(NamespaceRequest{Scenario: scenario})
	if err != nil {
		t.Fatalf("resolve namespace: %v", err)
	}
	if ns.DataDir == "" {
		t.Fatal("expected a resolvable data dir for the live variant")
	}
	if err := os.MkdirAll(ns.DataDir, 0o750); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	return ns.DataDir
}

func TestMigrateAutoResolvesCanonicalSQLiteDB(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	dataDir := withHome(t, &svc, "demo")

	// The live database follows the react-vite convention "<data-dir>/<scenario>.db".
	canonical := filepath.Join(dataDir, "demo.db")
	if err := os.WriteFile(canonical, nil, 0o600); err != nil {
		t.Fatalf("seed live db: %v", err)
	}
	writeEngagementScript(t, svc, "demo", "wip", "001-init.sql", "CREATE TABLE t (x INTEGER);")

	// No --db-path: the runner must resolve the canonical live database itself.
	out, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip"})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !out.DBPathAutoResolved {
		t.Fatalf("expected DBPathAutoResolved=true, got %+v", out)
	}
	if out.Database != canonical {
		t.Fatalf("resolved database = %q, want %q", out.Database, canonical)
	}
	if len(out.Applied) != 1 {
		t.Fatalf("expected one applied migration, got %+v", out)
	}
	db, _ := sql.Open("sqlite", canonical)
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("INSERT INTO t (x) VALUES (1)"); err != nil {
		t.Fatalf("migrated table should exist in the auto-resolved db: %v", err)
	}
}

func TestMigrateAutoResolvesSoleDBFile(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	dataDir := withHome(t, &svc, "demo")

	// A scenario whose db file is NOT the canonical name still resolves when it is
	// the only *.db in the data dir.
	sole := filepath.Join(dataDir, "store.db")
	if err := os.WriteFile(sole, nil, 0o600); err != nil {
		t.Fatalf("seed db: %v", err)
	}
	writeEngagementScript(t, svc, "demo", "wip", "001-init.sql", "CREATE TABLE t (x INTEGER);")

	out, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip"})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if !out.DBPathAutoResolved || out.Database != sole {
		t.Fatalf("expected auto-resolution to the sole db %q, got %+v", sole, out)
	}
}

func TestMigrateExplicitDBPathSkipsAutoResolution(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	dataDir := withHome(t, &svc, "demo")
	// Seed a canonical db that auto-resolution WOULD pick, to prove the explicit
	// path wins over it.
	if err := os.WriteFile(filepath.Join(dataDir, "demo.db"), nil, 0o600); err != nil {
		t.Fatalf("seed canonical db: %v", err)
	}
	writeEngagementScript(t, svc, "demo", "wip", "001-init.sql", "CREATE TABLE t (x INTEGER);")

	explicit := filepath.Join(t.TempDir(), "explicit.db")
	out, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip", DBPath: explicit})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if out.DBPathAutoResolved {
		t.Fatalf("explicit --db-path must not be reported as auto-resolved: %+v", out)
	}
	if out.Database != explicit {
		t.Fatalf("explicit db path ignored: got %q want %q", out.Database, explicit)
	}
}

func TestMigrateAutoResolveErrorsWhenNoDBFile(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	withHome(t, &svc, "demo") // empty data dir — no *.db present
	writeEngagementScript(t, svc, "demo", "wip", "001-init.sql", "CREATE TABLE t (x INTEGER);")

	_, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip"})
	if err == nil {
		t.Fatal("expected an actionable error when no SQLite db can be resolved")
	}
	if !strings.Contains(err.Error(), "--db-path") {
		t.Fatalf("error should point at --db-path, got: %v", err)
	}
}

func TestMigrateAutoResolveErrorsWhenAmbiguous(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	dataDir := withHome(t, &svc, "demo")
	// Two non-canonical db files ⇒ ambiguous ⇒ refuse to guess.
	for _, name := range []string{"a.db", "b.db"} {
		if err := os.WriteFile(filepath.Join(dataDir, name), nil, 0o600); err != nil {
			t.Fatalf("seed db %s: %v", name, err)
		}
	}
	writeEngagementScript(t, svc, "demo", "wip", "001-init.sql", "CREATE TABLE t (x INTEGER);")

	_, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip"})
	if err == nil {
		t.Fatal("expected an ambiguity error for multiple .db files")
	}
	if !strings.Contains(err.Error(), "--db-path") {
		t.Fatalf("ambiguity error should point at --db-path, got: %v", err)
	}
}

func TestMigrateFastPathNeedsNoDBResolution(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	// No HomeDir seam and no scripts: the fast path must not attempt DB resolution.
	out, err := svc.Migrate(MigrateRequest{Scenario: "demo", Slug: "wip"})
	if err != nil {
		t.Fatalf("fast path must not require a database: %v", err)
	}
	if !out.FastPath || out.DBPathAutoResolved {
		t.Fatalf("no-scripts fast path should skip db resolution: %+v", out)
	}
}
