package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// The retired review columns are dropped and only the publisher's generic rows
// go; an investigation's finding survives; running again changes nothing.
func TestRetireRunFindingReviewStateDropsColumnsAndGenericRowsOnly(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	raw, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "findings.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	db := &DB{DB: raw, log: log}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE run_findings (id TEXT PRIMARY KEY, severity TEXT NOT NULL, recommendation_text TEXT NOT NULL,
		proposed_workaround TEXT NOT NULL DEFAULT '', acceptance_impact TEXT NOT NULL DEFAULT '', owner_hint TEXT NOT NULL DEFAULT '', fingerprint TEXT NOT NULL,
		operator_decision TEXT NOT NULL DEFAULT '', target_measure TEXT NOT NULL DEFAULT '', before_value REAL, after_value REAL, effectiveness TEXT NOT NULL DEFAULT '',
		friction_topic TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ id, severity, text, topic string }{
		{"generic", "recurring", genericFindingText, ""},
		{"generic-routed", "recurring", genericFindingText, "friction-inbox/toolchain/x"},
		{"investigation", "major", "Add a regression test", ""},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO run_findings (id, severity, recommendation_text, fingerprint, friction_topic) VALUES (?, ?, ?, ?, ?)`, row.id, row.severity, row.text, row.id, row.topic); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := db.retireRunFindingReviewState(ctx); err != nil {
			t.Fatal(err)
		}
	}
	columns, err := db.tableColumns(ctx, "run_findings")
	if err != nil {
		t.Fatal(err)
	}
	for _, retired := range retiredRunFindingColumns {
		if _, ok := columns[retired]; ok {
			t.Errorf("column %s was not dropped", retired)
		}
	}
	var ids []string
	if err := db.SelectContext(ctx, &ids, `SELECT id FROM run_findings ORDER BY id`); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "generic-routed" || ids[1] != "investigation" {
		t.Fatalf("rows kept = %v, want the routed row and the investigation finding", ids)
	}
}
