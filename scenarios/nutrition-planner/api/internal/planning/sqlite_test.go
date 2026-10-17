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

func TestMigratePopulatedLegacyPlansAfterSchemaPreservesWorkspaces(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE plans(workspace_id TEXT PRIMARY KEY,revision INTEGER NOT NULL,plan_json TEXT NOT NULL,updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	legacy := []struct {
		workspace, raw, updated string
		revision                int
	}{
		{"alpha", `{"occurrences":[{"date":"2026-10-03","slotName":"dinner","mode":"fixed","quantity":"2","recipeId":"soup","recipeName":"Soup","reason":"family favorite","locked":true},{"date":"2026-10-04","slotName":"lunch","mode":"flexible","quantity":"1","recipeId":"salad","recipeName":"Salad","reason":"quick meal"}],"inputReferences":["recipe:soup:12","recipe:salad:3"],"runId":"alpha-run","seed":41}`, "2026-10-03T01:00:00Z", 7},
		{"beta", `{"occurrences":[{"date":"2026-10-05","slotName":"breakfast","mode":"fixed","quantity":"1","recipeId":"oats","recipeName":"Oats","reason":"planned","locked":true}],"inputReferences":["recipe:oats:5"],"runId":"beta-run","seed":9}`, "2026-10-05T02:00:00Z", 11},
	}
	for _, row := range legacy {
		if _, err := db.Exec(`INSERT INTO plans VALUES(?,?,?,?)`, row.workspace, row.revision, row.raw, row.updated); err != nil {
			t.Fatal(err)
		}
	}
	// Match startup: populated legacy storage predates creation of planning tables.
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := MigrateLegacyPlans(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	for _, row := range legacy {
		revision, raw, err := repo.Get(ctx, row.workspace)
		if err != nil || revision != int64(row.revision) {
			t.Fatalf("%s revision=%d err=%v", row.workspace, revision, err)
		}
		var got Draft
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatal(err)
		}
		var want Draft
		if err := json.Unmarshal([]byte(row.raw), &want); err != nil {
			t.Fatal(err)
		}
		if got.RunID != want.RunID || got.Seed != want.Seed || len(got.Occurrences) != len(want.Occurrences) {
			t.Fatalf("%s metadata/occurrences changed: %+v", row.workspace, got)
		}
		for i, occurrence := range want.Occurrences {
			if occurrence.RecipeRevision == 0 {
				occurrence.RecipeRevision = map[string]int64{"soup": 12, "salad": 3, "oats": 5}[occurrence.RecipeID]
			}
			if got.Occurrences[i] != occurrence {
				t.Fatalf("%s occurrence %d = %+v, want %+v", row.workspace, i, got.Occurrences[i], occurrence)
			}
		}
		var archived string
		if err := db.QueryRow(`SELECT plan_json FROM plans WHERE workspace_id=?`, row.workspace).Scan(&archived); err != nil || archived != row.raw {
			t.Fatalf("%s legacy row changed: raw=%q err=%v", row.workspace, archived, err)
		}
	}
	// A later startup must leave normalized state authoritative even if archival JSON differs.
	if _, err := db.Exec(`UPDATE plan_state SET revision=99 WHERE workspace_id='alpha'`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyPlans(ctx, db); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if revision, _, err := repo.Get(ctx, "alpha"); err != nil || revision != 99 {
		t.Fatalf("repeat migration overwrote normalized state: revision=%d err=%v", revision, err)
	}
	var sourceRows, occurrenceRows int
	if err := db.QueryRow(`SELECT count(*) FROM plans`).Scan(&sourceRows); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM plan_occurrences`).Scan(&occurrenceRows); err != nil {
		t.Fatal(err)
	}
	if sourceRows != 2 || occurrenceRows != 3 {
		t.Fatalf("source rows=%d normalized occurrences=%d", sourceRows, occurrenceRows)
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
