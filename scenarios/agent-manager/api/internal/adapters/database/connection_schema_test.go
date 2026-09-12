package database

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestMigrateWatchActionDecisionNullabilityPreservesRows(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE cohort_watches (watch_id TEXT PRIMARY KEY);
        CREATE TABLE cohort_watch_decisions (decision_id TEXT PRIMARY KEY);`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cohort_watches (watch_id) VALUES ('watch-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE cohort_watch_actions (
        action_id TEXT PRIMARY KEY, watch_id TEXT NOT NULL, decision_id TEXT NOT NULL,
        idempotency_key TEXT NOT NULL UNIQUE, kind INTEGER NOT NULL,
        target_run_id TEXT NOT NULL DEFAULT '', state INTEGER NOT NULL,
        action_json TEXT NOT NULL, cooldown_until TEXT, created_at TEXT NOT NULL,
        acknowledged_at TEXT
    )`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cohort_watch_actions
        (action_id,watch_id,decision_id,idempotency_key,kind,target_run_id,state,action_json,created_at)
        VALUES ('action-1','watch-1','decision-1','key-1',1,'run-1',1,'{}','2026-09-05T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	wrapper := NewDB(db, nil)
	if err := wrapper.migrateWatchActionDecisionNullability(context.Background()); err != nil {
		t.Fatal(err)
	}
	var notNull int
	if err := db.Get(&notNull, `SELECT "notnull" FROM pragma_table_info('cohort_watch_actions') WHERE name='decision_id'`); err != nil {
		t.Fatal(err)
	}
	if notNull != 0 {
		t.Fatalf("decision_id remains NOT NULL: %d", notNull)
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM cohort_watch_actions WHERE action_id='action-1' AND decision_id='decision-1'`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("preserved actions=%d", count)
	}
	if _, err := db.Exec(`INSERT INTO cohort_watch_actions
        (action_id,watch_id,decision_id,idempotency_key,kind,state,action_json,created_at)
        VALUES ('action-2','watch-1',NULL,'key-2',1,1,'{}','2026-09-05T00:00:00Z')`); err != nil {
		t.Fatalf("nullable action insert: %v", err)
	}
}
