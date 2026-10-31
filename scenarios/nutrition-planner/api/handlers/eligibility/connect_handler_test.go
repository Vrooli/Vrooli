package eligibility

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/eligibility"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/workspace"
)

func TestEvaluateReportsViableMethodsWithoutAccumulatingRecipeWideAppliances(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(workspace.Schema()); err != nil {
		t.Fatal(err)
	}
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, schedule.System()))
	ws, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Personal", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(workspaces, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	response, err := h.Evaluate(ctx, connect.NewRequest(&v1.EvaluateRequest{
		WorkspaceId: ws.ID, ProfileRevision: 9, RecipeRevision: 12,
		Appliances: []string{"oven"}, DeviceFunctions: []string{"oven"},
		RequiredAppliances: []string{"oven", "stove"},
		Methods: []*v1.EligibilityMethod{
			{Id: "stovetop", Name: "Stovetop", PreparationMode: "cooking", RequiredAppliances: []string{"stove"}, RequiredDeviceFunctions: []string{"cooktop"}},
			{Id: "roast", Name: "Roast", PreparationMode: "cooking", RequiredAppliances: []string{"oven"}, RequiredDeviceFunctions: []string{"oven"}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Status != "eligible" || response.Msg.ProfileRevision != 9 || response.Msg.RecipeRevision != 12 {
		t.Fatalf("decision status/revisions=%s/%d/%d", response.Msg.Status, response.Msg.ProfileRevision, response.Msg.RecipeRevision)
	}
	if len(response.Msg.ViableMethodIds) != 1 || response.Msg.ViableMethodIds[0] != "roast" {
		t.Fatalf("viable methods=%v", response.Msg.ViableMethodIds)
	}
	if len(response.Msg.MethodAssessments) != 2 || response.Msg.MethodAssessments[0].Status != "ineligible" || response.Msg.MethodAssessments[1].Status != "eligible" {
		t.Fatalf("method assessments=%#v", response.Msg.MethodAssessments)
	}
}
