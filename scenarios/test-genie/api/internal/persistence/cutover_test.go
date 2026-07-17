package persistence

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/storage/sqlitedb"

	_ "modernc.org/sqlite"
)

func TestDatabaseCutoverArchivesAndRebuilds(t *testing.T) {
	live := filepath.Join(t.TempDir(), "live.db")
	archive := filepath.Join(t.TempDir(), "archive.db")
	db, err := sql.Open("sqlite", sqlitedb.BuildDSN(live))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplySchema(db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO suite_executions (id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, terminal_outcome, started_at, completed_at) VALUES ('11111111-1111-1111-1111-111111111111','demo','[]','[]','[]',0,1,'passed','2026-01-01T00:00:00Z','2026-01-01T00:01:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDatabaseCutover(live, archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyDatabaseCutover(plan, DatabaseCutoverConfirmation); err != nil {
		t.Fatal(err)
	}
	if err := verifySQLite(archive); err != nil {
		t.Fatal(err)
	}
	if err := verifySQLite(live); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archive + ".cutover-receipt.json"); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", sqlitedb.BuildDSN(live))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_executions`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("replacement rows=%d err=%v", rows, err)
	}
}
