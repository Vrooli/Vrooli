package runmodel

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSchemaIsAvailable(t *testing.T) {
	if Schema() == "" {
		t.Fatal("run-model schema must be embedded")
	}
}

func TestSchemaRetiresLegacyRunTimestampTriggerWithoutChangingRows(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	const revision = "2026-11-06T20:00:00.000000011Z"
	if _, err := db.Exec(`INSERT INTO runs (id, status, updated_at) VALUES ('retained', 'running', ?)`, revision); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER update_runs_updated_at AFTER UPDATE ON runs FOR EACH ROW BEGIN UPDATE runs SET updated_at = datetime('now') WHERE id = NEW.id; END;`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := db.Exec(Schema()); err != nil {
			t.Fatal(err)
		}
		var status, updatedAt string
		if err := db.QueryRow(`SELECT status, updated_at FROM runs WHERE id='retained'`).Scan(&status, &updatedAt); err != nil {
			t.Fatal(err)
		}
		if status != "running" || updatedAt != revision {
			t.Fatalf("schema application changed retained row: %q %q", status, updatedAt)
		}
	}
	const next = "2026-11-06T20:00:00.000000012Z"
	if _, err := db.Exec(`UPDATE runs SET status='complete', updated_at=? WHERE id='retained'`, next); err != nil {
		t.Fatal(err)
	}
	var actual string
	if err := db.QueryRow(`SELECT updated_at FROM runs WHERE id='retained'`).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != next {
		t.Fatalf("repository revision overwritten: %q", actual)
	}
}
