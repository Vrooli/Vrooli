package cooking

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	cookingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cooking"
	_ "modernc.org/sqlite"
	internal "nutrition-planner/internal/cooking"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

type testWorkspaceService struct{ workspace.Service }

func (testWorkspaceService) Get(_ context.Context, id, owner string) (workspace.Workspace, error) {
	if id != "w1" {
		return workspace.Workspace{}, workspace.ErrNotFound{ID: id}
	}
	if owner != "owner" {
		return workspace.Workspace{}, workspace.ErrForbidden{ID: id}
	}
	return workspace.Workspace{ID: "w1", OwnerSubject: "owner"}, nil
}

type testRecipeService struct{ revision recipe.Recipe }

func (s testRecipeService) Create(context.Context, recipe.CreateInput) (recipe.Recipe, error) {
	return recipe.Recipe{}, nil
}
func (s testRecipeService) List(context.Context, string) ([]recipe.Recipe, error) { return nil, nil }
func (s testRecipeService) Get(context.Context, string, string) (recipe.Recipe, error) {
	return recipe.Recipe{}, nil
}
func (s testRecipeService) GetRevision(_ context.Context, id, workspaceID string, revision int64) (recipe.Recipe, error) {
	if id != s.revision.ID || workspaceID != s.revision.WorkspaceID || revision != s.revision.Revision {
		return recipe.Recipe{}, recipe.ErrNotFound{ID: id}
	}
	return s.revision, nil
}
func (s testRecipeService) Update(context.Context, recipe.UpdateInput) (recipe.Recipe, error) {
	return recipe.Recipe{}, nil
}

func TestCookingConnectHandlerPinsWorkspaceRevisionAndRetriesFinish(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(internal.Schema()); err != nil {
		t.Fatal(err)
	}
	method := recipe.Method{ID: "stove", Name: "Stovetop", Steps: []recipe.MethodStep{{ID: "prep", Instruction: "Chop"}, {ID: "simmer", Instruction: "Simmer"}}}
	pinned := recipe.Recipe{ID: "r1", WorkspaceID: "w1", Revision: 3, Name: "Soup", Methods: []recipe.Method{method}}
	h := NewConnectHandler(internal.NewSQLiteRepository(db), testRecipeService{revision: pinned}, testWorkspaceService{}, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	started, err := h.StartSession(ctx, connect.NewRequest(&cookingv1.StartSessionRequest{WorkspaceId: "w1", SessionId: "s1", RecipeId: "r1", RecipeRevision: 3, MethodId: "stove", Scale: "4"}))
	if err != nil {
		t.Fatal(err)
	}
	if started.Msg.Session.RecipeRevision != 3 || started.Msg.Session.MethodId != "stove" || started.Msg.Session.Scale != "4" {
		t.Fatalf("session pin changed: %#v", started.Msg.Session)
	}
	badMethod, err := h.StartSession(ctx, connect.NewRequest(&cookingv1.StartSessionRequest{WorkspaceId: "w1", SessionId: "s2", RecipeId: "r1", RecipeRevision: 3, MethodId: "microwave", Scale: "4"}))
	if err == nil || badMethod != nil {
		t.Fatal("started a method absent from the pinned revision")
	}
	actor := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "other", Verified: true})
	if _, err = h.GetSession(actor, connect.NewRequest(&cookingv1.GetSessionRequest{WorkspaceId: "w1", SessionId: "s1"})); err == nil {
		t.Fatal("another actor read the cooking session")
	}
	timer := &cookingv1.Timer{Id: "timer-1", StepId: "simmer", DurationSeconds: 600, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	progressReq := &cookingv1.SaveSessionRequest{WorkspaceId: "w1", SessionId: "s1", EventId: "progress-1", ExpectedVersion: 1, CurrentStepIndex: 1, CompletedSteps: []string{"prep"}, Timers: []*cookingv1.Timer{timer}}
	progress, err := h.SaveSession(ctx, connect.NewRequest(progressReq))
	if err != nil {
		t.Fatal(err)
	}
	if progress.Msg.Session.CurrentStepIndex != 1 || len(progress.Msg.Session.CompletedSteps) != 1 || len(progress.Msg.Session.Timers) != 1 {
		t.Fatalf("progress not saved: %#v", progress.Msg.Session)
	}
	finishReq := &cookingv1.SaveSessionRequest{WorkspaceId: "w1", SessionId: "s1", EventId: "finish-1", ExpectedVersion: progress.Msg.Session.Version, CurrentStepIndex: 1, CompletedSteps: []string{"prep"}, Timers: []*cookingv1.Timer{timer}, Finish: true, ActualYield: "3.5", YieldUnit: "bowl"}
	finished, err := h.SaveSession(ctx, connect.NewRequest(finishReq))
	if err != nil {
		t.Fatal(err)
	}
	retry, err := h.SaveSession(ctx, connect.NewRequest(finishReq))
	if err != nil {
		t.Fatal(err)
	}
	if finished.Msg.Session.Status != "finished" || finished.Msg.Session.ActualYield != "3.5" || retry.Msg.Session.Version != finished.Msg.Session.Version {
		t.Fatalf("finish=%#v retry=%#v", finished.Msg.Session, retry.Msg.Session)
	}
}
