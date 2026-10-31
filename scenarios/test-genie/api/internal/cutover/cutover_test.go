package cutover

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/persistence"
	sharedruns "test-genie/internal/shared/runs"
	"test-genie/internal/storage/sqlitedb"

	_ "modernc.org/sqlite"
)

func TestApplyOfflineArchivesBothStores(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "coverage", "runs", "old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "coverage", "runs", "old", "findings.json"), []byte("detail"), 0o644); err != nil {
		t.Fatal(err)
	}
	liveDB, archiveDB := filepath.Join(dir, "live.db"), filepath.Join(dir, "archive.db")
	liveDSN, err := sqlitedb.BuildDSN(liveDB)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", liveDSN)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistence.ApplySchema(db, false); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanOffline(dir, filepath.Join(dir, "evidence-archive"), liveDB, archiveDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyOffline(plan, Confirmation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "evidence-archive", "runs", "old", "findings.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archiveDB + ".cutover-receipt.json"); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", liveDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_executions`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("replacement rows=%d err=%v", rows, err)
	}
}

func TestPlanOfflineRejectsActiveRun(t *testing.T) {
	dir := t.TempDir()
	if err := sharedruns.NewIndex(dir).Append(sharedruns.RunRecord{RunID: "running", Status: sharedruns.StatusInProgress}); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanOffline(dir, filepath.Join(dir, "evidence-archive"), filepath.Join(dir, "live.db"), filepath.Join(dir, "archive.db")); err == nil {
		t.Fatal("active run must reject cutover preflight")
	}
}
