package recipe

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func repo(t *testing.T) Repository {
	t.Helper()
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if _, e = db.Exec(Schema()); e != nil {
		t.Fatal(e)
	}
	return NewSQLiteRepository(db, schedule.System())
}

func TestRevisionHistoryAndWorkspaceIsolation(t *testing.T) {
	r := repo(t)
	s := NewService(r)
	ctx := context.Background()
	v, e := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Bowl"})
	if e != nil {
		t.Fatal(e)
	}
	v2, e := s.Update(ctx, UpdateInput{WorkspaceID: "w1", ID: v.ID, ExpectedRevision: 1, Name: "Better Bowl"})
	if e != nil || v2.Revision != 2 {
		t.Fatalf("update %#v %v", v2, e)
	}
	if _, e := s.Update(ctx, UpdateInput{WorkspaceID: "w1", ID: v.ID, ExpectedRevision: 1, Name: "Conflict"}); e == nil {
		t.Fatal("stale update accepted")
	}
	historical, e := s.GetRevision(ctx, v.ID, "w1", 1)
	if e != nil || historical.Name != "Bowl" || historical.Revision != 1 {
		t.Fatalf("historical revision %#v %v", historical, e)
	}
	old, e := r.Get(ctx, v.ID, "w1")
	if e != nil || old.Name != "Better Bowl" || old.Revision != 2 {
		t.Fatalf("current %#v %v", old, e)
	}
	if _, e := s.Get(ctx, v.ID, "w2"); e == nil {
		t.Fatal("foreign recipe readable")
	}
	items, e := s.List(ctx, "w2")
	if e != nil || len(items) != 0 {
		t.Fatalf("foreign list %#v %v", items, e)
	}
}

func TestCreateIdempotencyReplaysAndRejectsChangedPayload(t *testing.T) {
	s := NewService(repo(t))
	ctx := context.Background()
	first, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Soup", IdempotencyKey: "capture-1"})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Soup", IdempotencyKey: "capture-1"})
	if err != nil || replay.ID != first.ID {
		t.Fatalf("replay %#v %v", replay, err)
	}
	if _, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Salad", IdempotencyKey: "capture-1"}); err == nil {
		t.Fatal("changed idempotent payload accepted")
	}
}

func TestRecipeYieldAndIngredientsPersistAcrossRevisions(t *testing.T) {
	s := NewService(repo(t))
	ctx := context.Background()
	first, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Bowl", CanonicalYield: "2", ServingUnit: "servings", Ingredients: []Ingredient{{ID: "rice", Name: "Rice", Amount: "40", Unit: "g"}}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.Update(ctx, UpdateInput{WorkspaceID: "w1", ID: first.ID, ExpectedRevision: first.Revision, Name: "Bowl", CanonicalYield: "4", ServingUnit: "servings", Ingredients: []Ingredient{{ID: "rice", Name: "Rice", Amount: "80", Unit: "g"}}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.CanonicalYield != "4" || len(updated.Ingredients) != 1 || updated.Ingredients[0].Amount != "80" {
		t.Fatalf("updated=%#v", updated)
	}
	if first.CanonicalYield != "2" || first.Ingredients[0].Amount != "40" {
		t.Fatalf("create snapshot mutated=%#v", first)
	}
}

func TestStructuredRecipeFieldsPersistAcrossNameAndNotesEdit(t *testing.T) {
	s := NewService(repo(t))
	ctx := context.Background()
	method := Method{ID: "stovetop", Name: "Stovetop", Steps: []MethodStep{{
		ID: "simmer", Instruction: "Simmer the rice", Inputs: []string{"rice"}, Outputs: []string{"bowl"},
	}}}
	ingredients := []Ingredient{{ID: "rice", Name: "Rice", Amount: "180", Unit: "g", Preparation: "rinsed"}}
	groups := []string{"grain-bowl", "weeknight"}
	appliances := []string{"saucepan", "stove"}
	allergenEvidence := map[string]string{"sesame": "not-present", "soy": "check-label"}
	created, err := s.Create(ctx, CreateInput{
		WorkspaceID: "w1", Name: "Original Bowl", Notes: "Original notes",
		Methods: []Method{method}, Groups: groups, RequiredAppliances: appliances,
		AllergenEvidence: allergenEvidence, CanonicalYield: "2", ServingUnit: "bowls",
		Ingredients: ingredients, OriginalText: "Cook rice, then serve.",
		SourceURL: "https://example.test/recipe", SourceType: "web",
	})
	if err != nil {
		t.Fatalf("create structured recipe: %v", err)
	}

	updated, err := s.Update(ctx, UpdateInput{
		WorkspaceID: "w1", ID: created.ID, ExpectedRevision: created.Revision,
		Name: "Renamed Bowl", Notes: "Updated notes",
		Methods: created.Methods, Groups: created.Groups, RequiredAppliances: created.RequiredAppliances,
		AllergenEvidence: created.AllergenEvidence, CanonicalYield: created.CanonicalYield,
		ServingUnit: created.ServingUnit, Ingredients: created.Ingredients,
		OriginalText: created.OriginalText, SourceURL: created.SourceURL, SourceType: created.SourceType,
	})
	if err != nil {
		t.Fatalf("update structured recipe: %v", err)
	}
	if updated.Revision != 2 || updated.Name != "Renamed Bowl" || updated.Notes != "Updated notes" {
		t.Fatalf("updated recipe = %#v", updated)
	}

	reopened, err := s.GetRevision(ctx, created.ID, "w1", 2)
	if err != nil {
		t.Fatalf("retrieve immutable revision 2: %v", err)
	}
	if reopened.Revision != 2 || reopened.Name != "Renamed Bowl" || reopened.Notes != "Updated notes" {
		t.Fatalf("revision 2 identity = %#v", reopened)
	}
	if !reflect.DeepEqual(reopened.Methods, created.Methods) ||
		!reflect.DeepEqual(reopened.Groups, created.Groups) ||
		!reflect.DeepEqual(reopened.RequiredAppliances, created.RequiredAppliances) ||
		!reflect.DeepEqual(reopened.AllergenEvidence, created.AllergenEvidence) ||
		reopened.CanonicalYield != created.CanonicalYield || reopened.ServingUnit != created.ServingUnit ||
		!reflect.DeepEqual(reopened.Ingredients, created.Ingredients) ||
		reopened.OriginalText != created.OriginalText || reopened.SourceURL != created.SourceURL || reopened.SourceType != created.SourceType {
		t.Fatalf("revision 2 lost preserved fields: got %#v, want structured/provenance fields from %#v", reopened, created)
	}
}

func TestMethodsCanReferenceRecipeIngredientsAndEarlierOutputs(t *testing.T) {
	s := NewService(repo(t))
	ctx := context.Background()
	method := Method{ID: "stovetop", Name: "Stovetop", Steps: []MethodStep{
		{ID: "prepare", Instruction: "Prepare the rice", Inputs: []string{"rice"}, Outputs: []string{"base"}},
		{ID: "finish", Instruction: "Finish the bowl", DependsOn: []string{"prepare"}, Inputs: []string{"base"}, Outputs: []string{"bowl"}},
	}}
	created, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Bowl", Ingredients: []Ingredient{{ID: "rice", Name: "Rice"}}, Methods: []Method{method}})
	if err != nil {
		t.Fatalf("create method graph: %v", err)
	}
	updated, err := s.Update(ctx, UpdateInput{WorkspaceID: "w1", ID: created.ID, ExpectedRevision: created.Revision, Name: "Bowl", Ingredients: []Ingredient{{ID: "rice", Name: "Rice"}}, Methods: []Method{method}})
	if err != nil || updated.Revision != 2 || updated.Methods[0].Steps[1].Inputs[0] != "base" {
		t.Fatalf("update method graph = %#v, %v", updated, err)
	}
	if _, err := s.Create(ctx, CreateInput{WorkspaceID: "w1", Name: "Broken", Ingredients: []Ingredient{{ID: "rice"}}, Methods: []Method{{ID: "invalid", Steps: []MethodStep{{ID: "mix", Inputs: []string{"missing"}}}}}}); err == nil {
		t.Fatal("method graph with a dangling ingredient reference was accepted")
	}
}
