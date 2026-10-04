package planning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func planningDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(Schema()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestApplyPersistsOccurrencesAndRetriesIdempotently(t *testing.T) {
	db := planningDB(t)
	repo := NewSQLiteRepository(db, schedule.System())
	ctx := context.Background()
	draft := Draft{RunID: "run-1", InputReferences: []string{"recipe:r1:4"}, Occurrences: []Occurrence{{Date: "2026-10-03", SlotName: "dinner", Mode: "fixed", Quantity: "1", RecipeID: "r1", RecipeRevision: 4, RecipeName: "Soup", Reason: "known fit", Locked: true}}}
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := repo.Apply(ctx, "w1", 0, string(raw))
	if err != nil || revision != 1 {
		t.Fatalf("first apply: revision=%d err=%v", revision, err)
	}
	if retried, err := repo.Apply(ctx, "w1", 0, string(raw)); err != nil || retried != 1 {
		t.Fatalf("identical retry: revision=%d err=%v", retried, err)
	}
	changed := draft
	changed.Occurrences = append([]Occurrence(nil), draft.Occurrences...)
	changed.Occurrences[0].RecipeName = "Different"
	changedRaw, _ := json.Marshal(changed)
	if _, err := repo.Apply(ctx, "w1", 0, string(changedRaw)); err == nil {
		t.Fatal("stale different draft was accepted")
	} else {
		var stale ErrStaleInputs
		if !errors.As(err, &stale) {
			t.Fatalf("expected stale conflict, got %v", err)
		}
	}
	actual, saved, err := repo.Get(ctx, "w1")
	if err != nil || actual != 1 {
		t.Fatalf("get: revision=%d err=%v", actual, err)
	}
	var got Draft
	if err := json.Unmarshal([]byte(saved), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Occurrences) != 1 || got.Occurrences[0].RecipeRevision != 4 || !got.Occurrences[0].Locked {
		t.Fatalf("persisted occurrence=%+v", got.Occurrences)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM plan_occurrences WHERE workspace_id='w1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("occurrence rows=%d err=%v", count, err)
	}
}

func TestApplyRangePreservesOtherDatesAndRecognizesLateRetry(t *testing.T) {
	db := planningDB(t)
	repo := NewSQLiteRepository(db, schedule.System())
	ctx := context.Background()
	first := Draft{RunID: "today", Occurrences: []Occurrence{{Date: "2026-10-03", SlotName: "dinner", RecipeID: "r1", RecipeName: "Soup"}}}
	firstRaw, _ := json.Marshal(first)
	if revision, err := repo.Apply(ctx, "w1", 0, string(firstRaw)); err != nil || revision != 1 {
		t.Fatalf("first range revision=%d err=%v", revision, err)
	}
	week := Draft{RunID: "week", Occurrences: []Occurrence{{Date: "2026-10-05", SlotName: "dinner", RecipeID: "r2", RecipeName: "Rice"}}}
	weekRaw, _ := json.Marshal(week)
	if revision, err := repo.Apply(ctx, "w1", 1, string(weekRaw)); err != nil || revision != 2 {
		t.Fatalf("second range revision=%d err=%v", revision, err)
	}
	if revision, err := repo.Apply(ctx, "w1", 0, string(firstRaw)); err != nil || revision != 1 {
		t.Fatalf("late identical retry revision=%d err=%v", revision, err)
	}
	_, raw, err := repo.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	var got Draft
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Occurrences) != 2 || got.Occurrences[0].Date != "2026-10-03" || got.Occurrences[1].Date != "2026-10-05" {
		t.Fatalf("range write did not preserve both dates: %+v", got.Occurrences)
	}
}

func TestMigrateLegacyPlanPreservesOccurrencesAndIsIdempotent(t *testing.T) {
	db := planningDB(t)
	_, err := db.Exec(`CREATE TABLE plans(workspace_id TEXT PRIMARY KEY,revision INTEGER NOT NULL,plan_json TEXT NOT NULL,updated_at TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	legacy := `{"occurrences":[{"date":"2026-10-03","slotName":"dinner","mode":"fixed","quantity":"1","recipeId":"r1","recipeName":"Soup","reason":"legacy","locked":true}],"inputReferences":["recipe:r1:9"],"runId":"legacy-run","seed":19}`
	if _, err := db.Exec(`INSERT INTO plans VALUES(?,?,?,?)`, "w1", 7, legacy, "2026-10-03T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyPlans(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyPlans(context.Background(), db); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	revision, raw, err := repo.Get(context.Background(), "w1")
	if err != nil || revision != 7 {
		t.Fatalf("get: revision=%d err=%v", revision, err)
	}
	var got Draft
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Occurrences) != 1 || got.Occurrences[0].RecipeRevision != 9 || !got.Occurrences[0].Locked || got.RunID != "legacy-run" {
		t.Fatalf("migrated draft=%+v", got)
	}
	var legacyCount, migratedCount int
	if err := db.QueryRow(`SELECT count(*) FROM plans`).Scan(&legacyCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM plan_occurrences`).Scan(&migratedCount); err != nil {
		t.Fatal(err)
	}
	if legacyCount != 1 || migratedCount != 1 {
		t.Fatalf("legacy rows=%d migrated rows=%d", legacyCount, migratedCount)
	}
}

func TestMigrateLegacyPlansRollsBackAllRowsOnMalformedData(t *testing.T) {
	db := planningDB(t)
	if _, err := db.Exec(`CREATE TABLE plans(workspace_id TEXT PRIMARY KEY,revision INTEGER NOT NULL,plan_json TEXT NOT NULL,updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	valid := `{"occurrences":[{"date":"2026-10-03","slotName":"dinner","recipeId":"r1"}],"runId":"good"}`
	if _, err := db.Exec(`INSERT INTO plans VALUES(?,?,?,?),(?,?,?,?)`, "a", 1, valid, "now", "z", 1, "not-json", "now"); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyPlans(context.Background(), db); err == nil {
		t.Fatal("malformed legacy row did not fail migration")
	}
	for _, table := range []string{"plan_state", "plan_occurrences", "plan_migrations"} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s rows=%d err=%v; migration was partially committed", table, count, err)
		}
	}
}
