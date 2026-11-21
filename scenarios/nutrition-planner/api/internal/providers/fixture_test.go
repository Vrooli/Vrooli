package providers

import (
	"context"
	"errors"
	"testing"
)

func TestFixtureNutritionPreservesSourceAndPreparationIdentity(t *testing.T) {
	provider := FixtureNutrition{Records: []NutritionFixtureRecord{{
		Search: NutritionSearchResult{SourceID: "fdc-1", SourceVersion: "fixture-v1", Name: "Beans", Preparation: "cooked", DataType: "foundation"},
		Detail: NutritionResult{SourceID: "fdc-1", SourceVersion: "fixture-v1", ObservedAt: "2026-09-18", Values: map[string]string{"protein": "9"}, Evidence: []string{"fixture"}},
	}}}
	results, err := provider.Search(context.Background(), NutritionSearchQuery{Query: "beans", Preparation: "cooked", DataType: "foundation"})
	if err != nil || len(results) != 1 || results[0].SourceVersion != "fixture-v1" {
		t.Fatalf("results=%#v err=%v", results, err)
	}
	detail, err := provider.Detail(context.Background(), NutritionQuery{FoodID: "fdc-1", Preparation: "cooked", DataType: "foundation"})
	if err != nil || detail.Values["protein"] != "9" {
		t.Fatalf("detail=%#v err=%v", detail, err)
	}
	if _, err := provider.Detail(context.Background(), NutritionQuery{FoodID: "missing"}); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("missing source err=%v", err)
	}
}
