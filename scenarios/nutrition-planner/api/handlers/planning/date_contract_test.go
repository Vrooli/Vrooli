package planning

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"
	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning"
	_ "modernc.org/sqlite"
	internal "nutrition-planner/internal/planning"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

func TestGeneratePlanRejectsImpossibleCalendarDate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	owner, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(workspaces, recipe.NewService(recipe.NewSQLiteRepository(db, clock)), nil, internal.NewSQLiteRepository(db, clock), nil, nil, nil)
	generated, err := handler.GeneratePlan(ctx, connect.NewRequest(&planningv1.GeneratePlanRequest{WorkspaceId: owner.ID, Dates: []string{"2026-02-30"}}))
	if connect.CodeOf(err) == connect.CodeInvalidArgument {
		return
	}
	if err == nil {
		_, applyErr := handler.ApplyPlan(ctx, connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: owner.ID, ExpectedRevision: generated.Msg.GetCurrentRevision(), DraftJson: generated.Msg.GetDraftJson()}))
		if applyErr != nil {
			t.Fatalf("impossible-date draft could not be applied: %v", applyErr)
		}
		revision, persisted, readErr := internal.NewSQLiteRepository(db, clock).Get(ctx, owner.ID)
		if readErr != nil || revision != 1 {
			t.Fatalf("impossible-date draft did not persist: revision=%d err=%v", revision, readErr)
		}
		var draft internal.Draft
		if decodeErr := json.Unmarshal([]byte(persisted), &draft); decodeErr != nil || len(draft.Occurrences) != 0 {
			// With no eligible recipes the generator records this date as unresolved.
			if decodeErr != nil || len(draft.Unresolved) != 1 || draft.Unresolved[0].Date != "2026-02-30" {
				t.Fatalf("impossible date was not retained in persisted draft: draft=%+v err=%v", draft, decodeErr)
			}
		}
	}
	t.Fatalf("GeneratePlan accepted impossible date 2026-02-30: err=%v, want invalid argument", err)
}
