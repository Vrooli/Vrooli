package providers

import (
	"context"
	"errors"
	"strings"
)

var ErrSourceNotFound = errors.New("provider source record not found")

type FixtureNutrition struct {
	Records []NutritionFixtureRecord
}

type NutritionFixtureRecord struct {
	Search NutritionSearchResult
	Detail NutritionResult
}

func (f FixtureNutrition) Search(_ context.Context, query NutritionSearchQuery) ([]NutritionSearchResult, error) {
	needle := strings.ToLower(strings.TrimSpace(query.Query))
	var out []NutritionSearchResult
	for _, record := range f.Records {
		if needle != "" && !strings.Contains(strings.ToLower(record.Search.Name), needle) && !strings.Contains(strings.ToLower(record.Search.ProductName), needle) {
			continue
		}
		if query.Preparation != "" && query.Preparation != record.Search.Preparation {
			continue
		}
		if query.DataType != "" && query.DataType != record.Search.DataType {
			continue
		}
		out = append(out, record.Search)
	}
	return out, nil
}

func (f FixtureNutrition) Detail(_ context.Context, query NutritionQuery) (NutritionResult, error) {
	for _, record := range f.Records {
		if record.Search.SourceID == query.FoodID {
			return record.Detail, nil
		}
	}
	return NutritionResult{}, ErrSourceNotFound
}

var _ NutritionCatalog = FixtureNutrition{}
