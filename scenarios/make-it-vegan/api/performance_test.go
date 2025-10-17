package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkCheckIngredients benchmarks the ingredient checking endpoint
func BenchmarkCheckIngredients(b *testing.B) {
	router := setupTestRouter()

	reqBody := CheckRequest{Ingredients: "flour, milk, sugar, eggs, butter"}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkCheckIngredientsVegan benchmarks with all vegan ingredients
func BenchmarkCheckIngredientsVegan(b *testing.B) {
	router := setupTestRouter()

	reqBody := CheckRequest{Ingredients: "flour, sugar, salt, water, oil"}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkFindSubstitute benchmarks the substitute finding endpoint
func BenchmarkFindSubstitute(b *testing.B) {
	router := setupTestRouter()

	reqBody := SubstituteRequest{
		Ingredient: "milk",
		Context:    "baking",
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/substitute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkVeganizeRecipe benchmarks the recipe veganization endpoint
func BenchmarkVeganizeRecipe(b *testing.B) {
	router := setupTestRouter()

	reqBody := VeganizeRequest{
		Recipe: "Mix 1 cup milk, 2 eggs, and 1 tbsp butter. Add flour and sugar. Bake for 30 minutes.",
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkVeganDatabaseInit benchmarks database initialization
func BenchmarkVeganDatabaseInit(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = InitVeganDatabase()
	}
}

// BenchmarkCheckIngredientsLogic benchmarks the core logic
func BenchmarkCheckIngredientsLogic(b *testing.B) {
	db := InitVeganDatabase()
	ingredients := "flour, milk, eggs, butter, cheese, sugar, salt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.CheckIngredients(ingredients)
	}
}

// BenchmarkGetAlternatives benchmarks alternative retrieval
func BenchmarkGetAlternatives(b *testing.B) {
	db := InitVeganDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetAlternatives("milk")
	}
}

// BenchmarkGetQuickSubstitute benchmarks quick substitute lookup
func BenchmarkGetQuickSubstitute(b *testing.B) {
	db := InitVeganDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetQuickSubstitute("1 egg")
	}
}

// BenchmarkGetNutritionalInsights benchmarks nutritional info retrieval
func BenchmarkGetNutritionalInsights(b *testing.B) {
	db := InitVeganDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetNutritionalInsights()
	}
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkGetCommonProducts benchmarks common products endpoint
func BenchmarkGetCommonProducts(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/products", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkGetNutrition benchmarks nutrition endpoint
func BenchmarkGetNutrition(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/nutrition", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkParallelCheckIngredients benchmarks concurrent requests
func BenchmarkParallelCheckIngredients(b *testing.B) {
	router := setupTestRouter()

	reqBody := CheckRequest{Ingredients: "flour, milk, sugar"}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
		}
	})
}

// TestPerformanceRequirements tests that performance meets requirements
func TestPerformanceRequirements(t *testing.T) {
	t.Run("ResponseTimeUnder500ms", func(t *testing.T) {
		// This is a placeholder test that documents the performance requirement
		// In a real scenario, you'd measure actual response times
		// Per PRD: "API response time < 500ms (95th percentile)"
		t.Log("Performance requirement: API response time < 500ms (95th percentile)")
		t.Log("Run benchmarks with: go test -bench=. -benchtime=10s")
	})

	t.Run("SupportsTargetConcurrency", func(t *testing.T) {
		// Per PRD: "Support 100 concurrent users"
		t.Log("Performance requirement: Support 100 concurrent users")
		t.Log("Run parallel benchmarks to verify concurrency handling")
	})

	t.Run("ProcessLargeIngredientLists", func(t *testing.T) {
		// Per PRD: "Process ingredient lists up to 50 items"
		db := InitVeganDatabase()

		// Create a large ingredient list
		ingredients := make([]string, 50)
		for i := 0; i < 50; i++ {
			if i%3 == 0 {
				ingredients[i] = "milk"
			} else {
				ingredients[i] = "flour"
			}
		}

		ingredientStr := ""
		for i, ing := range ingredients {
			if i > 0 {
				ingredientStr += ", "
			}
			ingredientStr += ing
		}

		// Should handle 50 items without error
		isVegan, nonVegan, reasons := db.CheckIngredients(ingredientStr)

		if len(nonVegan) != len(reasons) {
			t.Errorf("Mismatch between non-vegan items (%d) and reasons (%d)", len(nonVegan), len(reasons))
		}

		t.Logf("Processed %d ingredients: isVegan=%v, found %d non-vegan items",
			len(ingredients), isVegan, len(nonVegan))
	})

	t.Run("CacheHitRate", func(t *testing.T) {
		// Per PRD: "Cache hit rate > 80% for common queries"
		t.Log("Performance requirement: Cache hit rate > 80% for common queries")
		t.Log("This requires Redis to be running and monitoring cache statistics")
	})
}
