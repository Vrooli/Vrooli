package recipe

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe"
	_ "modernc.org/sqlite"
	internal "nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

func TestRecipeConnectHandlerPreservesStepEquipmentAndSettingsByRevision(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ws, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(internal.NewService(internal.NewSQLiteRepository(db, clock)), workspaces, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})

	created, err := h.CreateRecipe(ctx, connect.NewRequest(&v1.CreateRecipeRequest{
		WorkspaceId:        ws.ID,
		Name:               "Roasted soup",
		RequiredAppliances: []string{"oven", "stove"},
		Methods: []*v1.RecipeMethod{{Id: "oven", Name: "Roast then simmer", Steps: []*v1.RecipeStep{
			{Id: "roast", Instruction: "Roast vegetables", Equipment: []string{"sheet pan"}, Settings: []string{"220 C", "25 minutes"}},
			{Id: "blend", Instruction: "Blend until smooth", DependsOn: []string{"roast"}, Equipment: []string{"blender"}},
			{Id: "serve", Instruction: "Serve", DependsOn: []string{"blend"}},
		}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	recipeID := created.Msg.Recipe.Id
	assertMethodFields := func(label string, got *v1.Recipe, roastEquipment, roastSettings []string) {
		t.Helper()
		if !reflect.DeepEqual(got.RequiredAppliances, []string{"oven", "stove"}) {
			t.Errorf("%s recipe-wide appliances=%v", label, got.RequiredAppliances)
		}
		steps := got.Methods[0].Steps
		if !reflect.DeepEqual(steps[0].Equipment, roastEquipment) || !reflect.DeepEqual(steps[0].Settings, roastSettings) {
			t.Errorf("%s roast step equipment/settings=%v/%v", label, steps[0].Equipment, steps[0].Settings)
		}
		if !reflect.DeepEqual(steps[1].Equipment, []string{"blender"}) || len(steps[1].Settings) != 0 {
			t.Errorf("%s blend step equipment/settings=%v/%v", label, steps[1].Equipment, steps[1].Settings)
		}
		if len(steps[2].Equipment) != 0 || len(steps[2].Settings) != 0 {
			t.Errorf("%s empty step values should remain absent: %v/%v", label, steps[2].Equipment, steps[2].Settings)
		}
	}
	assertMethodFields("create response", created.Msg.Recipe, []string{"sheet pan"}, []string{"220 C", "25 minutes"})

	readV1, err := h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: ws.ID, Id: recipeID, Revision: 1}))
	if err != nil {
		t.Fatal(err)
	}
	assertMethodFields("revision 1 read", readV1.Msg.Recipe, []string{"sheet pan"}, []string{"220 C", "25 minutes"})

	updated, err := h.UpdateRecipe(ctx, connect.NewRequest(&v1.UpdateRecipeRequest{
		WorkspaceId: ws.ID, Id: recipeID, ExpectedRevision: 1, Name: "Roasted soup",
		RequiredAppliances: []string{"oven", "stove"},
		Methods: []*v1.RecipeMethod{{Id: "oven", Name: "Roast then simmer", Steps: []*v1.RecipeStep{
			{Id: "roast", Instruction: "Roast vegetables", Equipment: []string{"cast iron pan"}, Settings: []string{"200 C", "30 minutes"}},
			{Id: "blend", Instruction: "Blend until smooth", DependsOn: []string{"roast"}, Equipment: []string{"blender"}},
			{Id: "serve", Instruction: "Serve", DependsOn: []string{"blend"}},
		}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.Recipe.Revision != 2 {
		t.Fatalf("update revision=%d, want 2", updated.Msg.Recipe.Revision)
	}
	assertMethodFields("revision 2 response", updated.Msg.Recipe, []string{"cast iron pan"}, []string{"200 C", "30 minutes"})
	readV1, err = h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: ws.ID, Id: recipeID, Revision: 1}))
	if err != nil {
		t.Fatal(err)
	}
	assertMethodFields("immutable revision 1 read", readV1.Msg.Recipe, []string{"sheet pan"}, []string{"220 C", "25 minutes"})
	readV2, err := h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: ws.ID, Id: recipeID, Revision: 2}))
	if err != nil {
		t.Fatal(err)
	}
	assertMethodFields("revision 2 read", readV2.Msg.Recipe, []string{"cast iron pan"}, []string{"200 C", "30 minutes"})
}
