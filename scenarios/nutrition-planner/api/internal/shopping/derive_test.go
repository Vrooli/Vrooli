package shopping

import (
	"context"
	"testing"
	"time"

	"nutrition-planner/internal/cost"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/inventory"
	"nutrition-planner/internal/money"
	planning "nutrition-planner/internal/planning"
	recipe "nutrition-planner/internal/recipe"
)

type testInventory []inventory.Event

func (r testInventory) Append(context.Context, string, inventory.Event) error { return nil }
func (r testInventory) List(context.Context, string) ([]inventory.Event, error) {
	return []inventory.Event(r), nil
}

type testCosts []cost.Observation

func (r testCosts) Create(_ context.Context, v cost.Observation) (cost.Observation, error) {
	return v, nil
}
func (r testCosts) List(context.Context, string, string) ([]cost.Observation, error) {
	return []cost.Observation(r), nil
}

func TestEnrichUsesOnlyCurrentUnconditionalCompatibleEvidence(t *testing.T) {
	amount, _ := decimalx.Parse("500")
	price, _ := money.New(250, "USD", 2)
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	lines := []Line{{Key: "ingredient:rice", Label: "Rice", ItemID: "rice", Unit: "g", Amount: amount, Need: "500 g", Stock: "unknown", Missing: "unknown", PackageCount: "unknown", Price: "unknown", PortionCost: "unknown", CheckoutTotal: "unknown", ActualSpend: "unknown"}}
	events := testInventory{{ID: "p1", Kind: inventory.Purchase, ItemID: "rice", Amount: mustShoppingDecimal(t, "100"), Unit: "g"}}
	observations := testCosts{{ID: "current", ItemID: "rice", PackageAmount: mustShoppingDecimal(t, "400"), PackageUnit: "g", Price: price, ObservedAt: now.Add(-time.Hour), Available: true}, {ID: "coupon", ItemID: "rice", PackageAmount: mustShoppingDecimal(t, "400"), PackageUnit: "g", Price: price, ObservedAt: now.Add(-time.Hour), Available: true, CouponRequired: true}}
	got, err := Enrich(context.Background(), "w1", lines, events, observations, now)
	if err != nil {
		t.Fatal(err)
	}
	line := got[0]
	if line.Stock != "100 g" || line.Missing != "400 g" || line.PackageCount != "1" || line.Price != "2.50 USD / 400 g" || line.PortionCost != "3.125 USD" || line.CheckoutTotal != "2.5 USD" || line.ActualSpend != "unknown" {
		t.Fatalf("enriched line=%+v", line)
	}
}

func mustShoppingDecimal(t *testing.T, value string) decimalx.Decimal {
	t.Helper()
	v, err := decimalx.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestDerivePreservesUnknownsAndChecklistOnlyState(t *testing.T) {
	lines := Derive(planning.Draft{Occurrences: []planning.Occurrence{{RecipeID: "r1", RecipeName: "Bowl"}}}, []recipe.Recipe{{ID: "r1", Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"rice"}}}}}}}, map[string]bool{"ingredient:rice": true})
	if len(lines) != 1 || !lines[0].Checked || lines[0].Need != "unknown" || lines[0].Price != "unknown" {
		t.Fatalf("lines=%+v", lines)
	}
}

func TestDeriveScalesKnownIngredientDemandByPlannedQuantity(t *testing.T) {
	draft := planning.Draft{Occurrences: []planning.Occurrence{{RecipeID: "r1", RecipeRevision: 1, RecipeName: "Bowl", Quantity: "3"}}}
	item := recipe.Recipe{ID: "r1", Revision: 1, CanonicalYield: "2", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "100", Unit: "g"}}, Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"rice"}}}}}}
	lines := Derive(draft, []recipe.Recipe{item}, nil)
	if len(lines) != 1 || lines[0].Need != "150 g" || lines[0].ItemID != "rice" || lines[0].Amount.String() != "150" {
		t.Fatalf("scaled demand=%+v", lines)
	}
}

func TestDeriveUsesPinnedRecipeRevision(t *testing.T) {
	draft := planning.Draft{Occurrences: []planning.Occurrence{
		{RecipeID: "r1", RecipeRevision: 1, RecipeName: "Bowl"},
		{RecipeID: "r1", RecipeRevision: 2, RecipeName: "Bowl"},
	}}
	revisions := []recipe.Recipe{
		{ID: "r1", Revision: 1, Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"rice"}}}}}},
		{ID: "r1", Revision: 2, Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"quinoa"}}}}}},
	}
	lines := Derive(draft, revisions, nil)
	if len(lines) != 2 || lines[0].Key != "ingredient:quinoa" || lines[1].Key != "ingredient:rice" {
		t.Fatalf("pinned revisions produced lines=%+v", lines)
	}
}

func TestDeriveIgnoresOpenSkippedSlotsWithoutInventingDemand(t *testing.T) {
	draft := planning.Draft{Occurrences: []planning.Occurrence{
		{Date: "2026-10-03", SlotName: "dinner", Mode: "open", RecipeName: "Open dinner"},
		{Date: "2026-10-04", SlotName: "dinner", Mode: "open", RecipeName: "Skipped dinner"},
		{Date: "2026-10-05", SlotName: "dinner", RecipeID: "r1", RecipeRevision: 1, RecipeName: "Bowl"},
	}}
	recipes := []recipe.Recipe{{ID: "r1", Revision: 1, Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"rice"}}}}}}}
	lines := Derive(draft, recipes, nil)
	if len(lines) != 1 || lines[0].Key != "ingredient:rice" || lines[0].Need != "unknown" || lines[0].Stock != "unknown" || lines[0].Missing != "unknown" || lines[0].PackageCount != "unknown" || lines[0].Price != "unknown" {
		t.Fatalf("open slots must not create demand, and unsupported quantity/stock/price must remain unknown: %+v", lines)
	}
}

func TestDiffShowsOnlyIngredientIdentitiesAddedOrRemovedAndRetainsUnknowns(t *testing.T) {
	before := Derive(planning.Draft{Occurrences: []planning.Occurrence{{RecipeID: "old", RecipeRevision: 1}}}, []recipe.Recipe{{ID: "old", Revision: 1, Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"beans", "rice"}}}}}}}, nil)
	after := Derive(planning.Draft{Occurrences: []planning.Occurrence{{RecipeID: "new", RecipeRevision: 2}}}, []recipe.Recipe{{ID: "new", Revision: 2, Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"beans", "greens"}}}}}}}, nil)
	changes := Diff(before, after)
	if len(changes) != 2 || changes[0].Key != "ingredient:greens" || changes[0].After == nil || changes[1].Key != "ingredient:rice" || changes[1].Before == nil {
		t.Fatalf("unexpected shopping diff: %+v", changes)
	}
	if changes[0].After.Need != "unknown" || changes[0].After.Stock != "unknown" || changes[0].After.PackageCount != "unknown" || changes[0].After.Price != "unknown" {
		t.Fatalf("diff invented shopping evidence: %+v", changes[0])
	}
}

func TestDiffIncludesChangedNeedForAnExistingIngredient(t *testing.T) {
	before := []Line{{Key: "ingredient:rice", Need: "100 g", Stock: "20 g", Missing: "80 g", PackageCount: "1", Price: "2 USD / 500 g"}}
	after := []Line{{Key: "ingredient:rice", Need: "200 g", Stock: "20 g", Missing: "180 g", PackageCount: "1", Price: "2 USD / 500 g"}}
	changes := Diff(before, after)
	if len(changes) != 1 || changes[0].Before == nil || changes[0].After == nil || changes[0].Before.Need != "100 g" || changes[0].After.Need != "200 g" {
		t.Fatalf("quantity impact should be reviewable: %+v", changes)
	}
}
