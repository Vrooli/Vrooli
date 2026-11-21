package cooking

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLiteSessionPersistsPinnedProgressTimerAndIdempotentFinish(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	r := NewSQLiteRepository(db)
	ctx := context.Background()
	started := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	s, err := r.Create(ctx, Session{ID: "cook-1", WorkspaceID: "w1", RecipeID: "r1", RecipeRevision: 3, MethodID: "oven", Scale: "2", StartedAt: started})
	if err != nil {
		t.Fatal(err)
	}
	s.CurrentStepIndex = 1
	s.CompletedSteps = []string{"prep"}
	s.Timers = []Timer{{ID: "timer-1", StepID: "bake", DurationSeconds: 600, StartedAt: started}}
	s, err = r.Save(ctx, "w1", "advance-1", "hash-advance-1", s, s.Version)
	if err != nil {
		t.Fatal(err)
	}
	// Reopening the repository returns the same pinned revision and running timer.
	reopened := NewSQLiteRepository(db)
	got, err := reopened.Get(ctx, "w1", "cook-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RecipeRevision != 3 || got.MethodID != "oven" || got.Scale != "2" || got.CurrentStepIndex != 1 || len(got.CompletedSteps) != 1 || got.Timers[0].StartedAt != started {
		t.Fatalf("progress was not durably restored: %#v", got)
	}
	got.Status = "finished"
	got.ActualYield = "5"
	got.YieldUnit = "serving"
	finished, err := r.Save(ctx, "w1", "finish-1", "hash-finish-1", got, got.Version)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := r.Save(ctx, "w1", "finish-1", "hash-finish-1", got, got.Version)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Version != finished.Version || retry.ActualYield != "5" || retry.FinishedAt.IsZero() {
		t.Fatalf("duplicate finish was not idempotent: %#v", retry)
	}
	if _, err = r.Save(ctx, "w1", "finish-1", "different-payload", got, got.Version); err == nil {
		t.Fatal("same event id accepted a different payload hash")
	}
	if _, err = r.Get(ctx, "w2", "cook-1"); err == nil {
		t.Fatal("session crossed workspace boundary")
	}
}

func TestSQLiteSessionUpgradeKeepsExistingRecipeAndInventoryRows(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE recipes(id TEXT PRIMARY KEY,workspace_id TEXT,current_revision INTEGER,created_at TEXT,updated_at TEXT);
CREATE TABLE recipe_revisions(recipe_id TEXT,revision INTEGER,workspace_id TEXT,name TEXT,notes TEXT DEFAULT '',source_url TEXT DEFAULT '',source_type TEXT DEFAULT '',original_text TEXT DEFAULT '',status TEXT,methods_json TEXT DEFAULT '[]',groups_json TEXT DEFAULT '[]',required_appliances_json TEXT DEFAULT '[]',allergen_evidence_json TEXT DEFAULT '{}',canonical_yield TEXT DEFAULT '',serving_unit TEXT DEFAULT 'serving',ingredients_json TEXT DEFAULT '[]',created_at TEXT,PRIMARY KEY(recipe_id,revision));
CREATE TABLE inventory_events(workspace_id TEXT,event_id TEXT,kind TEXT,item_id TEXT DEFAULT '',batch_id TEXT DEFAULT '',amount TEXT,unit TEXT,recipe_id TEXT DEFAULT '',created_at TEXT,PRIMARY KEY(workspace_id,event_id));
CREATE TABLE inventory_batches(workspace_id TEXT,batch_id TEXT,recipe_id TEXT DEFAULT '',recipe_revision INTEGER DEFAULT 0,yield_amount TEXT,available_amount TEXT,unit TEXT,PRIMARY KEY(workspace_id,batch_id));
INSERT INTO recipes VALUES('r','w',1,'now','now'); INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,status,created_at) VALUES('r',1,'w','Soup','draft','now'); INSERT INTO inventory_events(workspace_id,event_id,kind,item_id,amount,unit,created_at) VALUES('w','raw','purchase','beans','2','cup','now'); INSERT INTO inventory_batches(workspace_id,batch_id,recipe_id,recipe_revision,yield_amount,available_amount,unit) VALUES('w','b','r',1,'4','3','serving');`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	var recipes, revisions, events, batches int
	if err = db.QueryRow(`SELECT (SELECT count(*) FROM recipes),(SELECT count(*) FROM recipe_revisions),(SELECT count(*) FROM inventory_events),(SELECT count(*) FROM inventory_batches)`).Scan(&recipes, &revisions, &events, &batches); err != nil {
		t.Fatal(err)
	}
	if recipes != 1 || revisions != 1 || events != 1 || batches != 1 {
		t.Fatalf("pre-existing rows changed: %d %d %d %d", recipes, revisions, events, batches)
	}
}
