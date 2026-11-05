package baselinefloor

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeScripts drops the given name→SQL files into a fresh migrations dir and
// returns the dir.
func writeScripts(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, sqlText := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(sqlText), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// tableExists reports whether a table is present in the SQLite db at path.
func tableExists(t *testing.T, dbPath, table string) bool {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open %s: %v", dbPath, err)
	}
	defer func() { _ = db.Close() }()
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&name)
	switch err {
	case nil:
		return true
	case sql.ErrNoRows:
		return false
	default:
		t.Fatalf("query sqlite_master: %v", err)
		return false
	}
}

func TestLoadScriptsMissingDirIsFastPath(t *testing.T) {
	scripts, err := LoadScripts(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if scripts != nil {
		t.Fatalf("missing dir should yield nil scripts (fast path), got %v", scripts)
	}
}

func TestLoadScriptsLexicalOrderAndChecksum(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"002-second.sql": "SELECT 2;",
		"001-first.sql":  "SELECT 1;",
		"notes.txt":      "ignored",
	})
	// A subdirectory must be ignored too.
	if err := os.Mkdir(filepath.Join(dir, "sub.sql"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	scripts, err := LoadScripts(dir)
	if err != nil {
		t.Fatalf("LoadScripts: %v", err)
	}
	if len(scripts) != 2 {
		t.Fatalf("want 2 scripts (only *.sql files), got %d: %+v", len(scripts), scripts)
	}
	if scripts[0].Name != "001-first.sql" || scripts[1].Name != "002-second.sql" {
		t.Fatalf("scripts not in lexical order: %+v", scripts)
	}
	if scripts[0].Checksum == "" || scripts[0].Checksum == scripts[1].Checksum {
		t.Fatalf("checksums should be present and content-distinct: %+v", scripts)
	}
}

func TestRunMigrationsFastPathOpensNothing(t *testing.T) {
	// No scripts + a deliberately non-existent db path: the fast path must never
	// touch the database.
	res, err := RunMigrations(EngineSQLite, "/nonexistent/should-not-be-opened.db", nil, MigrateOptions{})
	if err != nil {
		t.Fatalf("fast path should not error: %v", err)
	}
	if !res.FastPath || res.ScriptsSeen != 0 || len(res.Applied) != 0 {
		t.Fatalf("unexpected fast-path result: %+v", res)
	}
}

func TestRunMigrationsUnsupportedEngine(t *testing.T) {
	dir := writeScripts(t, map[string]string{"001.sql": "SELECT 1;"})
	scripts, _ := LoadScripts(dir)
	_, err := RunMigrations(Engine("postgres"), filepath.Join(t.TempDir(), "x.db"), scripts, MigrateOptions{})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("postgres should be a surfaced not-supported error, got %v", err)
	}
}

func TestRunMigrationsScriptsButNoDBPath(t *testing.T) {
	dir := writeScripts(t, map[string]string{"001.sql": "SELECT 1;"})
	scripts, _ := LoadScripts(dir)
	if _, err := RunMigrations(EngineSQLite, "  ", scripts, MigrateOptions{}); err == nil {
		t.Fatal("scripts present with no db path must error")
	}
}

func TestRunMigrationsApplyCreatesSchemaAndTracks(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"001-create.sql": "CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT);",
		"002-seed.sql":   "INSERT INTO widgets (name) VALUES ('a');\nINSERT INTO widgets (name) VALUES ('b');",
	})
	scripts, _ := LoadScripts(dir)
	dbPath := filepath.Join(t.TempDir(), "live.db")

	res, err := RunMigrations(EngineSQLite, dbPath, scripts, MigrateOptions{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(res.Applied) != 2 || len(res.Skipped) != 0 {
		t.Fatalf("first apply should run both scripts: %+v", res)
	}
	if !tableExists(t, dbPath, "widgets") {
		t.Fatal("widgets table should exist after apply")
	}
	// Multi-statement script executed fully (2 rows).
	db, _ := sql.Open("sqlite", dbPath)
	defer func() { _ = db.Close() }()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM widgets").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("multi-statement seed should insert 2 rows, got %d", count)
	}
}

func TestRunMigrationsReRunIsNoOp(t *testing.T) {
	dir := writeScripts(t, map[string]string{"001-create.sql": "CREATE TABLE t (x INTEGER);"})
	scripts, _ := LoadScripts(dir)
	dbPath := filepath.Join(t.TempDir(), "live.db")

	if _, err := RunMigrations(EngineSQLite, dbPath, scripts, MigrateOptions{}); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	res, err := RunMigrations(EngineSQLite, dbPath, scripts, MigrateOptions{})
	if err != nil {
		t.Fatalf("re-run: %v", err)
	}
	if len(res.Applied) != 0 || len(res.Skipped) != 1 {
		t.Fatalf("re-run should skip the already-applied script: %+v", res)
	}
}

func TestRunMigrationsChecksumDriftRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live.db")
	dir1 := writeScripts(t, map[string]string{"001.sql": "CREATE TABLE a (x INTEGER);"})
	s1, _ := LoadScripts(dir1)
	if _, err := RunMigrations(EngineSQLite, path, s1, MigrateOptions{}); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	// Same filename, different contents → edited after apply → hard error. The
	// dry-run-against-a-copy carries the live tracking table, so the drift is
	// caught before the live apply even starts.
	dir2 := writeScripts(t, map[string]string{"001.sql": "CREATE TABLE a (x INTEGER, y INTEGER);"})
	s2, _ := LoadScripts(dir2)
	_, err := RunMigrations(EngineSQLite, path, s2, MigrateOptions{})
	if err == nil || !strings.Contains(err.Error(), "different checksum") {
		t.Fatalf("checksum drift should be rejected, got %v", err)
	}
}

func TestApplySQLiteIsAtomicOnFailure(t *testing.T) {
	// applySQLite is the live-apply half; a bad later script must roll back the
	// good earlier one AND the tracking table (all-or-nothing).
	path := filepath.Join(t.TempDir(), "live.db")
	scripts := []Script{
		{Name: "001-good.sql", SQL: "CREATE TABLE good (x INTEGER);", Checksum: "aaa"},
		{Name: "002-bad.sql", SQL: "THIS IS NOT VALID SQL;", Checksum: "bbb"},
	}
	if _, _, err := applySQLite(path, scripts); err == nil {
		t.Fatal("a bad script must fail the batch")
	}
	if tableExists(t, path, "good") {
		t.Fatal("the good script must have rolled back with the bad one")
	}
	if tableExists(t, path, migrationTrackingTable) {
		t.Fatal("the tracking table must have rolled back too (nothing committed)")
	}
}

func TestRunMigrationsDryRunDoesNotMutateLive(t *testing.T) {
	dir := writeScripts(t, map[string]string{"001.sql": "CREATE TABLE shadowonly (x INTEGER);"})
	scripts, _ := LoadScripts(dir)
	path := filepath.Join(t.TempDir(), "live.db")

	res, err := RunMigrations(EngineSQLite, path, scripts, MigrateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !res.DryRun || len(res.Applied) != 1 {
		t.Fatalf("dry-run should report the planned apply: %+v", res)
	}
	// Live must be untouched: the candidate ran on a throwaway copy.
	if _, statErr := os.Stat(path); statErr == nil && tableExists(t, path, "shadowonly") {
		t.Fatal("dry-run must not create the table in the live database")
	}
}

func TestRunMigrationsBounceLeavesLiveUntouched(t *testing.T) {
	// A bad script must be caught by the dry-run-against-a-copy BEFORE live is
	// touched: the good first script must NOT land in live.
	dir := writeScripts(t, map[string]string{
		"001-good.sql": "CREATE TABLE keep (x INTEGER);",
		"002-bad.sql":  "DEFINITELY NOT SQL;",
	})
	scripts, _ := LoadScripts(dir)
	path := filepath.Join(t.TempDir(), "live.db")

	_, err := RunMigrations(EngineSQLite, path, scripts, MigrateOptions{})
	if err == nil || !strings.Contains(err.Error(), "dry-run failed") {
		t.Fatalf("a bad script should bounce via the dry-run, got %v", err)
	}
	if _, statErr := os.Stat(path); statErr == nil && tableExists(t, path, "keep") {
		t.Fatal("a bounced migration must leave live untouched (no partial apply)")
	}
}

func TestRunMigrationsAgainstPopulatedLiveDatabase(t *testing.T) {
	// Seed a live DB, then migrate it — exercises the copy-with-data path.
	path := filepath.Join(t.TempDir(), "live.db")
	seed, _ := sql.Open("sqlite", path)
	if _, err := seed.Exec("CREATE TABLE existing (id INTEGER PRIMARY KEY); INSERT INTO existing (id) VALUES (1);"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = seed.Close()

	dir := writeScripts(t, map[string]string{"001-add.sql": "ALTER TABLE existing ADD COLUMN label TEXT;"})
	scripts, _ := LoadScripts(dir)
	res, err := RunMigrations(EngineSQLite, path, scripts, MigrateOptions{})
	if err != nil {
		t.Fatalf("migrate populated db: %v", err)
	}
	if len(res.Applied) != 1 {
		t.Fatalf("want one applied script, got %+v", res)
	}
	db, _ := sql.Open("sqlite", path)
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("UPDATE existing SET label = 'x' WHERE id = 1"); err != nil {
		t.Fatalf("new column should exist after migrate: %v", err)
	}
}
