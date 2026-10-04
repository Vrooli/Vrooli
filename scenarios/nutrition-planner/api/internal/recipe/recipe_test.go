package recipe

import (
	"context"
	"database/sql"
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
