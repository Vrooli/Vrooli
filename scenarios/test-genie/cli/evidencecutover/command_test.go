package evidencecutover

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/executionevidence"
	"test-genie/internal/persistence"
	"test-genie/internal/storage/sqlitedb"

	_ "modernc.org/sqlite"
)

func TestPlanIsReadOnlyAndApplyRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "coverage", "runs", "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "coverage", "runs", "run", "findings.json"), []byte("detail"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "archive")
	database := filepath.Join(dir, "test-genie.db")
	databaseArchive := filepath.Join(dir, "archive.db")
	dsn, err := sqlitedb.BuildDSN(database)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistence.ApplySchema(db, false); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"plan", "--scenario-dir", dir, "--archive-dir", archive, "--database-path", database, "--database-archive", databaseArchive}, &out); err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !strings.Contains(out.String(), "files=1") {
		t.Fatalf("plan output = %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "coverage", "runs", "run", "findings.json")); err != nil {
		t.Fatalf("plan mutated evidence: %v", err)
	}
	if err := run([]string{"apply", "--scenario-dir", dir, "--archive-dir", archive, "--database-path", database, "--database-archive", databaseArchive}, &out); err == nil {
		t.Fatal("apply without confirmation must fail")
	}
	if err := run([]string{"apply", "--scenario-dir", dir, "--archive-dir", archive, "--database-path", database, "--database-archive", databaseArchive, "--confirm", executionevidence.CutoverConfirmation}, &out); err != nil {
		t.Fatalf("confirmed apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archive, executionevidence.CutoverReceiptFile)); err != nil {
		t.Fatalf("cutover receipt missing: %v", err)
	}
	if _, err := os.Stat(databaseArchive + ".cutover-receipt.json"); err != nil {
		t.Fatalf("database receipt missing: %v", err)
	}
}
