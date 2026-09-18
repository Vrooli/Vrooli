package portability

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

func restoreDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), planning.Schema(), Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w1','Daily','actor',3,'now','now')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestApplyWorkspaceReplacesSupportedDomainsWithCheckpoint(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	content, err := Workspace(WorkspaceSnapshot{
		ID: "w-export", Name: "Imported", Revision: 9,
		Recipes:      []recipe.Recipe{{ID: "r1", Revision: 2, Name: "Soup", Status: "draft"}},
		PlanRevision: 4, PlanJSON: `{"occurrences":[{"recipeId":"r1"}]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	result, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "restore-1", envelope)
	if err != nil || result.WorkspaceRevision != 4 || result.RecipesApplied != 1 || result.CheckpointID == "" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM recipes WHERE workspace_id='w1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("recipe count=%d err=%v", count, err)
	}
	var plan string
	if err := db.QueryRow(`SELECT plan_json FROM plans WHERE workspace_id='w1'`).Scan(&plan); err != nil || plan == "" {
		t.Fatalf("plan=%q err=%v", plan, err)
	}
	var checkpoints int
	if err := db.QueryRow(`SELECT COUNT(*) FROM portability_restore_checkpoints WHERE id=?`, result.CheckpointID).Scan(&checkpoints); err != nil || checkpoints != 1 {
		t.Fatalf("checkpoint count=%d err=%v", checkpoints, err)
	}
	replay, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "restore-1", envelope)
	if err != nil || replay.CheckpointID != result.CheckpointID {
		t.Fatalf("idempotent replay=%#v err=%v", replay, err)
	}
}

func TestApplyWorkspaceRollsBackWhenImportedPlanIsInvalid(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	content, err := Workspace(WorkspaceSnapshot{ID: "w-export", Recipes: []recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup", Status: "draft"}}})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	badPlan, _ := json.Marshal(map[string]any{"revision": 1, "planJson": "not-json"})
	envelope.Records = append(envelope.Records, Record{Kind: "plan", ID: "w-export", Revision: 1, Data: badPlan})
	envelope.Manifest.RecordCount = len(envelope.Records)
	if _, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "restore-bad", envelope); err == nil {
		t.Fatal("expected invalid plan rejection")
	}
	var revision int64
	if err := db.QueryRow(`SELECT revision FROM workspaces WHERE id='w1'`).Scan(&revision); err != nil || revision != 3 {
		t.Fatalf("workspace revision=%d err=%v", revision, err)
	}
	var checkpoints int
	if err := db.QueryRow(`SELECT COUNT(*) FROM portability_restore_checkpoints`).Scan(&checkpoints); err != nil || checkpoints != 0 {
		t.Fatalf("rollback left checkpoint count=%d err=%v", checkpoints, err)
	}
}
