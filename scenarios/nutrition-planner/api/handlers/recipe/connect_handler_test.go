package recipe

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
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
	ws, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Personal", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(internal.NewService(internal.NewSQLiteRepository(db, clock)), workspaces, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})

	created, err := h.CreateRecipe(ctx, connect.NewRequest(&v1.CreateRecipeRequest{
		WorkspaceId:        ws.ID,
		Name:               "Roasted soup",
		RequiredAppliances: []string{"oven", "stove"},
		Methods: []*v1.RecipeMethod{{Id: "oven", Name: "Roast then simmer", PreparationMode: "cooking", RequiredAppliances: []string{"oven"}, RequiredCookware: []string{"baking_sheet"}, RequiredTools: []string{"chef_knife"}, RequiredDeviceFunctions: []string{"oven", "blend"}, Steps: []*v1.RecipeStep{
			{Id: "roast", Instruction: "Roast vegetables", Equipment: []string{"sheet pan"}, Settings: []string{"220 C", "25 minutes"}},
			{Id: "blend", Instruction: "Blend until smooth", DependsOn: []string{"roast"}, Equipment: []string{"blender"}},
			{Id: "serve", Instruction: "Serve", DependsOn: []string{"blend"}},
		}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	recipeID := created.Msg.Recipe.Id
	assertMethodFields := func(label string, got *v1.Recipe, roastEquipment, roastSettings, requiredCookware []string) {
		t.Helper()
		if !reflect.DeepEqual(got.RequiredAppliances, []string{"oven", "stove"}) {
			t.Errorf("%s recipe-wide appliances=%v", label, got.RequiredAppliances)
		}
		method := got.Methods[0]
		if method.PreparationMode != "cooking" || !reflect.DeepEqual(method.RequiredAppliances, []string{"oven"}) || !reflect.DeepEqual(method.RequiredCookware, requiredCookware) || !reflect.DeepEqual(method.RequiredTools, []string{"chef_knife"}) || !reflect.DeepEqual(method.RequiredDeviceFunctions, []string{"oven", "blend"}) {
			t.Errorf("%s method requirements=%q/%v/%v/%v/%v", label, method.PreparationMode, method.RequiredAppliances, method.RequiredCookware, method.RequiredTools, method.RequiredDeviceFunctions)
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
	assertMethodFields("create response", created.Msg.Recipe, []string{"sheet pan"}, []string{"220 C", "25 minutes"}, []string{"baking_sheet"})

	readV1, err := h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: ws.ID, Id: recipeID, Revision: 1}))
	if err != nil {
		t.Fatal(err)
	}
	assertMethodFields("revision 1 read", readV1.Msg.Recipe, []string{"sheet pan"}, []string{"220 C", "25 minutes"}, []string{"baking_sheet"})

	updated, err := h.UpdateRecipe(ctx, connect.NewRequest(&v1.UpdateRecipeRequest{
		WorkspaceId: ws.ID, Id: recipeID, ExpectedRevision: 1, Name: "Roasted soup",
		RequiredAppliances: []string{"oven", "stove"},
		Methods: []*v1.RecipeMethod{{Id: "oven", Name: "Roast then simmer", PreparationMode: "cooking", RequiredAppliances: []string{"oven"}, RequiredCookware: []string{"cast_iron_pan"}, RequiredTools: []string{"chef_knife"}, RequiredDeviceFunctions: []string{"oven", "blend"}, Steps: []*v1.RecipeStep{
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
	assertMethodFields("revision 2 response", updated.Msg.Recipe, []string{"cast iron pan"}, []string{"200 C", "30 minutes"}, []string{"cast_iron_pan"})
	readV1, err = h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: ws.ID, Id: recipeID, Revision: 1}))
	if err != nil {
		t.Fatal(err)
	}
	assertMethodFields("immutable revision 1 read", readV1.Msg.Recipe, []string{"sheet pan"}, []string{"220 C", "25 minutes"}, []string{"baking_sheet"})
	readV2, err := h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: ws.ID, Id: recipeID, Revision: 2}))
	if err != nil {
		t.Fatal(err)
	}
	assertMethodFields("revision 2 read", readV2.Msg.Recipe, []string{"cast iron pan"}, []string{"200 C", "30 minutes"}, []string{"cast_iron_pan"})
}

func TestRecipeConnectHandlerRejectsForeignReadsAndWrites(t *testing.T) {
	ctx := context.Background()
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
	local, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Local", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Foreign", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		t.Fatal(err)
	}
	repo := internal.NewSQLiteRepository(db, clock)
	h := NewConnectHandler(internal.NewService(repo), workspaces, nil)
	created, err := h.CreateRecipe(ctx, connect.NewRequest(&v1.CreateRecipeRequest{WorkspaceId: foreign.ID, Name: "FOREIGN SECRET RECIPE"}))
	if err != nil {
		t.Fatal(err)
	}
	foreignID := created.Msg.Recipe.Id
	beforeLocal, err := repo.List(ctx, local.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeForeign, err := repo.List(ctx, foreign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := h.GetRecipe(ctx, connect.NewRequest(&v1.GetRecipeRequest{WorkspaceId: local.ID, Id: foreignID})); err == nil || response != nil || strings.Contains(err.Error(), "FOREIGN SECRET RECIPE") {
		t.Fatalf("foreign recipe read response=%v err=%v, want content-free rejection", response, err)
	}
	if response, err := h.GetRecipeRevision(ctx, connect.NewRequest(&v1.GetRecipeRevisionRequest{WorkspaceId: local.ID, Id: foreignID, Revision: 1})); err == nil || response != nil || strings.Contains(err.Error(), "FOREIGN SECRET RECIPE") {
		t.Fatalf("foreign recipe revision read response=%v err=%v, want content-free rejection", response, err)
	}
	if response, err := h.UpdateRecipe(ctx, connect.NewRequest(&v1.UpdateRecipeRequest{WorkspaceId: local.ID, Id: foreignID, ExpectedRevision: 1, Name: "Attempted overwrite"})); err == nil || response != nil || strings.Contains(err.Error(), "FOREIGN SECRET RECIPE") {
		t.Fatalf("foreign recipe write response=%v err=%v, want content-free rejection", response, err)
	}
	afterLocal, err := repo.List(ctx, local.ID)
	if err != nil {
		t.Fatal(err)
	}
	afterForeign, err := repo.List(ctx, foreign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeLocal, afterLocal) || !reflect.DeepEqual(beforeForeign, afterForeign) {
		t.Fatalf("foreign recipe request mutated rows: local before=%+v after=%+v; foreign before=%+v after=%+v", beforeLocal, afterLocal, beforeForeign, afterForeign)
	}
	if response, err := h.GetRecipe(ctx, connect.NewRequest(&v1.GetRecipeRequest{WorkspaceId: foreign.ID, Id: foreignID})); err != nil || response.Msg.Recipe.Name != "FOREIGN SECRET RECIPE" {
		t.Fatalf("ownerless same-workspace recipe read response=%v err=%v", response, err)
	}
}
