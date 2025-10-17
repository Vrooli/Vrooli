package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestIntegrationWorkflow tests a complete user workflow
func TestIntegrationWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CompleteVeganCheckWorkflow", func(t *testing.T) {
		// Step 1: Check ingredients
		checkReq := CheckRequest{Ingredients: "milk, eggs, butter"}
		body, _ := json.Marshal(checkReq)

		req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Check ingredients failed: %v", rr.Code)
		}

		var checkResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &checkResp)

		isVegan, _ := checkResp["isVegan"].(bool)
		if isVegan {
			t.Error("Expected non-vegan result")
		}

		// Step 2: Find substitutes for each non-vegan ingredient
		nonVeganItems, ok := checkResp["nonVeganItems"].([]interface{})
		if !ok || len(nonVeganItems) == 0 {
			t.Fatal("Expected non-vegan items")
		}

		for _, item := range nonVeganItems {
			itemStr, _ := item.(string)

			subReq := SubstituteRequest{
				Ingredient: itemStr,
				Context:    "baking",
			}
			body, _ := json.Marshal(subReq)

			req, _ := http.NewRequest("POST", "/api/substitute", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Find substitute failed for %s: %v", itemStr, rr.Code)
			}

			var subResp map[string]interface{}
			json.Unmarshal(rr.Body.Bytes(), &subResp)

			if _, exists := subResp["alternatives"]; !exists {
				t.Errorf("Expected alternatives for %s", itemStr)
			}
		}

		// Step 3: Get nutrition information
		req, _ = http.NewRequest("GET", "/api/nutrition", nil)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Get nutrition failed: %v", rr.Code)
		}

		var nutritionResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &nutritionResp)

		if _, exists := nutritionResp["nutritionalInfo"]; !exists {
			t.Error("Expected nutritional info")
		}
	})

	t.Run("RecipeVeganizationWorkflow", func(t *testing.T) {
		// User wants to veganize a recipe
		recipe := "Mix 2 cups milk, 3 eggs, 1/2 cup butter with 2 cups flour. Add 1 cup cheese. Bake at 350F for 30 minutes."

		veganizeReq := VeganizeRequest{Recipe: recipe}
		body, _ := json.Marshal(veganizeReq)

		req, _ := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Veganize recipe failed: %v", rr.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		veganRecipe, ok := resp["veganRecipe"].(string)
		if !ok {
			t.Fatal("Expected vegan recipe string")
		}

		if veganRecipe == recipe {
			t.Error("Vegan recipe should be different from original")
		}

		substitutions, ok := resp["substitutions"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected substitutions map")
		}

		if len(substitutions) == 0 {
			t.Error("Expected at least one substitution")
		}

		// Verify cooking tips are provided
		if _, exists := resp["cookingTips"]; !exists {
			t.Error("Expected cooking tips")
		}
	})

	t.Run("ExploratoryWorkflow", func(t *testing.T) {
		// User explores common non-vegan products
		req, _ := http.NewRequest("GET", "/api/products", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Get products failed: %v", rr.Code)
		}

		var products map[string][]string
		json.Unmarshal(rr.Body.Bytes(), &products)

		if len(products) == 0 {
			t.Fatal("Expected product categories")
		}

		// User checks if each product in dairy category is vegan
		if dairyProducts, exists := products["dairy"]; exists {
			for _, product := range dairyProducts {
				checkReq := CheckRequest{Ingredients: product}
				body, _ := json.Marshal(checkReq)

				req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("Check failed for %s: %v", product, rr.Code)
					continue
				}

				var checkResp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &checkResp)

				isVegan, _ := checkResp["isVegan"].(bool)
				if isVegan {
					t.Errorf("Dairy product %s should not be vegan", product)
				}
			}
		}
	})
}

// TestEndToEndScenarios tests complete scenarios
func TestEndToEndScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	scenarios := []struct {
		name        string
		description string
		steps       []func(t *testing.T, router *httptest.Server)
	}{
		{
			name:        "NewVeganConvertingRecipe",
			description: "A new vegan wants to convert their favorite cookie recipe",
			steps: []func(t *testing.T, router *httptest.Server){
				func(t *testing.T, ts *httptest.Server) {
					// Check if current recipe is vegan
					checkReq := CheckRequest{Ingredients: "flour, butter, eggs, sugar, chocolate chips"}
					body, _ := json.Marshal(checkReq)

					req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
					req.Header.Set("Content-Type", "application/json")

					rr := httptest.NewRecorder()
					router.ServeHTTP(rr, req)

					var resp map[string]interface{}
					json.Unmarshal(rr.Body.Bytes(), &resp)

					if resp["isVegan"].(bool) {
						t.Error("Expected recipe to not be vegan")
					}
				},
				func(t *testing.T, ts *httptest.Server) {
					// Get vegan version of recipe
					recipe := "Mix 1 cup butter, 2 eggs, 2 cups flour, 1 cup sugar, and chocolate chips. Bake at 375F."

					veganizeReq := VeganizeRequest{Recipe: recipe}
					body, _ := json.Marshal(veganizeReq)

					req, _ := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
					req.Header.Set("Content-Type", "application/json")

					rr := httptest.NewRecorder()
					router.ServeHTTP(rr, req)

					var resp map[string]interface{}
					json.Unmarshal(rr.Body.Bytes(), &resp)

					if _, exists := resp["substitutions"]; !exists {
						t.Error("Expected substitutions")
					}
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			ts := httptest.NewServer(router)
			defer ts.Close()

			for i, step := range scenario.steps {
				t.Logf("Step %d of %s", i+1, scenario.description)
				step(t, ts)
			}
		})
	}
}

// TestAPIContractCompliance tests that API responses match expected contracts
func TestAPIContractCompliance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CheckIngredientsContract", func(t *testing.T) {
		reqBody := CheckRequest{Ingredients: "milk"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Verify required fields
		requiredFields := []string{"isVegan", "ingredients", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := resp[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}

		// When not vegan, additional fields required
		if !resp["isVegan"].(bool) {
			requiredNonVeganFields := []string{"nonVeganItems", "reasons", "analysis"}
			for _, field := range requiredNonVeganFields {
				if _, exists := resp[field]; !exists {
					t.Errorf("Missing required field for non-vegan result: %s", field)
				}
			}
		}
	})

	t.Run("SubstituteContract", func(t *testing.T) {
		reqBody := SubstituteRequest{Ingredient: "milk", Context: "baking"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/substitute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Verify required fields
		requiredFields := []string{"ingredient", "context", "alternatives", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := resp[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})

	t.Run("VeganizeContract", func(t *testing.T) {
		reqBody := VeganizeRequest{Recipe: "Mix milk and eggs"}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Verify required fields
		requiredFields := []string{"veganRecipe", "substitutions", "cookingTips", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := resp[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})

	t.Run("ProductsContract", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/products", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var resp map[string][]string
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Should have product categories
		if len(resp) == 0 {
			t.Error("Expected product categories")
		}

		// All values should be string arrays
		for category, items := range resp {
			if len(items) == 0 {
				t.Errorf("Category %s has no items", category)
			}
		}
	})

	t.Run("NutritionContract", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/nutrition", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Verify nutritionalInfo structure
		nutritionalInfo, exists := resp["nutritionalInfo"]
		if !exists {
			t.Fatal("Missing nutritionalInfo field")
		}

		info, ok := nutritionalInfo.(map[string]interface{})
		if !ok {
			t.Fatal("nutritionalInfo should be an object")
		}

		requiredFields := []string{"protein", "b12", "iron", "calcium", "omega3", "considerations", "goodSources"}
		for _, field := range requiredFields {
			if _, exists := info[field]; !exists {
				t.Errorf("Missing required nutrition field: %s", field)
			}
		}
	})
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("ConcurrentCheckIngredients", func(t *testing.T) {
		const concurrency = 10

		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()

				ingredients := "flour, milk, sugar"
				if id%2 == 0 {
					ingredients = "flour, sugar, salt"
				}

				reqBody := CheckRequest{Ingredients: ingredients}
				body, _ := json.Marshal(reqBody)

				req, _ := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("Request %d failed with status %d", id, rr.Code)
				}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}
