package profile

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/profile"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/profile"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

func TestApplyProfileCountsRecipesByOneCompleteMethodAndStoredDeviceFunctions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), profile.Schema(), recipe.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ws, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	if _, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ws.ID, Name: "Roast", RequiredAppliances: []string{"oven", "slow_cooker"}, Methods: []recipe.Method{
		{ID: "slow-cook", Name: "Slow cook", PreparationMode: "cooking", RequiredAppliances: []string{"slow_cooker"}, RequiredDeviceFunctions: []string{"slow_cook"}},
		{ID: "oven", Name: "Oven", PreparationMode: "cooking", RequiredAppliances: []string{"oven"}, RequiredDeviceFunctions: []string{"oven"}},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ws.ID, Name: "Pressure", Methods: []recipe.Method{{ID: "pressure", Name: "Pressure", PreparationMode: "cooking", RequiredAppliances: []string{"pressure_cooker"}, RequiredDeviceFunctions: []string{"pressure_cook"}}}}); err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(profile.NewService(profile.NewSQLiteRepository(db, clock)), workspaces, recipes, nil)
	ctx = identity.WithPrincipal(ctx, identity.Principal{Subject: "owner", Verified: true})
	response, err := h.ApplyProfile(ctx, connect.NewRequest(&v1.ApplyProfileRequest{
		WorkspaceId: ws.ID, Appliances: []string{"oven"}, ReplaceKitchenDevices: true, KitchenDevices: []*v1.KitchenDevice{{Id: "range-1", Type: "range", Count: 1, Capacity: "4 burners", Capabilities: []string{"oven"}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.MatchingMeals != 1 || response.Msg.ExcludedMeals != 1 || response.Msg.NeedsReviewMeals != 0 {
		t.Fatalf("method eligibility counts=%d matching, %d excluded, %d needs review", response.Msg.MatchingMeals, response.Msg.ExcludedMeals, response.Msg.NeedsReviewMeals)
	}
	if got := response.Msg.Profile.KitchenDevices; len(got) != 1 || got[0].Id != "range-1" || len(got[0].Capabilities) != 1 || got[0].Capabilities[0] != "oven" {
		t.Fatalf("stored device functions were not preserved: %#v", got)
	}
}
