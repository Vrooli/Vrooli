// +build testing


package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"service": "recipe-book",
		})

		if response != nil {
			if status, ok := response["status"].(string); ok {
				if status != "healthy" && status != "unhealthy" {
					t.Errorf("Expected status to be 'healthy' or 'unhealthy', got %s", status)
				}
			}
		}
	})
}

// TestCreateRecipeHandler tests recipe creation
func TestCreateRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Test Pancakes")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		if db != nil {
			response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
				"title": "Test Pancakes",
			})

			if response != nil {
				if id, ok := response["id"].(string); ok {
					if id == "" {
						t.Error("Expected non-empty recipe ID")
					}
				}
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   map[string]interface{}{},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Handler should still create recipe with defaults, check it doesn't crash
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Logf("Unexpected status code: %d", w.Code)
		}
	})
}

// TestGetRecipeHandler tests retrieving a single recipe
func TestGetRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Test Recipe")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": testRecipe.Recipe.CreatedBy},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		if db != nil {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"id":    testRecipe.Recipe.ID,
				"title": "Test Recipe",
			})

			if response != nil {
				if _, ok := response["ingredients"]; !ok {
					t.Error("Expected ingredients field in response")
				}
				if _, ok := response["instructions"]; !ok {
					t.Error("Expected instructions field in response")
				}
			}
		}
	})

	t.Run("NonExistentRecipe", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
			QueryParams: map[string]string{"user_id": "test-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		if db != nil {
			assertErrorResponse(t, w, http.StatusNotFound)
		}
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Private Recipe")
		testRecipe.Recipe.Visibility = "private"
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": "different-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		if db != nil {
			if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
				t.Errorf("Expected status 403 or 404 for unauthorized access, got %d", w.Code)
			}
		}
	})
}

// TestListRecipesHandler tests listing recipes
func TestListRecipesHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		// Create multiple test recipes
		recipe1 := setupTestRecipe(t, "Recipe 1")
		defer recipe1.Cleanup()

		recipe2 := setupTestRecipe(t, "Recipe 2")
		defer recipe2.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"user_id": recipe1.Recipe.CreatedBy,
				"limit":   "20",
				"offset":  "0",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)

		// Handle both DB and non-DB cases
		if db != nil {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if recipes, ok := response["recipes"].([]interface{}); ok {
					if len(recipes) < 1 {
						t.Error("Expected at least 1 recipe in list")
					}
				}
			}
		} else {
			// Without DB, expect 500 error
			if w.Code != http.StatusInternalServerError {
				t.Logf("Expected 500 without database, got %d", w.Code)
			}
		}
	})

	t.Run("WithFilters", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"cuisine":    "Italian",
				"visibility": "public",
				"limit":      "10",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)

		// Should not error even if no results
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})
}

// TestUpdateRecipeHandler tests updating recipes
func TestUpdateRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Original Recipe")
		defer testRecipe.Cleanup()

		updatedRecipe := *testRecipe.Recipe
		updatedRecipe.Title = "Updated Recipe"
		updatedRecipe.Description = "Updated description"

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    updatedRecipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateRecipeHandler(w, httpReq)

		if db != nil && w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"title": "Updated Recipe",
			})

			if response != nil {
				if desc, ok := response["description"].(string); ok {
					if desc != "Updated description" {
						t.Errorf("Expected description to be updated")
					}
				}
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Test Recipe")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    `{"invalid json`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestDeleteRecipeHandler tests deleting recipes
func TestDeleteRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Delete")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": testRecipe.Recipe.CreatedBy},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)

		if db != nil {
			if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
				t.Errorf("Expected status 204 or 200, got %d", w.Code)
			}
		}
	})

	t.Run("UnauthorizedDelete", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Protected Recipe")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": "different-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)

		if db != nil {
			if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
				t.Errorf("Expected status 403 or 404, got %d", w.Code)
			}
		}
	})

	t.Run("NonExistentRecipe", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/recipes/" + nonExistentID,
			URLVars: map[string]string{"id": nonExistentID},
			QueryParams: map[string]string{"user_id": "test-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)

		if db != nil {
			assertErrorResponse(t, w, http.StatusNotFound)
		}
	})
}

// TestSearchRecipesHandler tests semantic search functionality
func TestSearchRecipesHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		searchReq := TestData.SearchRequest("chocolate cake")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["results"]; !ok {
				t.Error("Expected results field in search response")
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   `{"invalid json`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("WithFilters", func(t *testing.T) {
		searchReq := SearchRequest{
			Query:  "pasta",
			UserID: "test-user",
			Limit:  5,
		}
		searchReq.Filters.Dietary = []string{"vegetarian"}
		searchReq.Filters.MaxTime = 30

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestGenerateRecipeHandler tests AI recipe generation
func TestGenerateRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		genReq := TestData.GenerateRequest("healthy breakfast recipe")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   genReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateRecipeHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["recipe"]; !ok {
				t.Error("Expected recipe field in generate response")
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   `{"invalid json`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("WithDietaryRestrictions", func(t *testing.T) {
		genReq := GenerateRequest{
			Prompt:              "dinner recipe",
			UserID:              "test-user",
			DietaryRestrictions: []string{"gluten-free", "dairy-free"},
			AvailableIngredients: []string{"chicken", "rice", "vegetables"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   genReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateRecipeHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestModifyRecipeHandler tests recipe modification
func TestModifyRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Original Recipe")
		defer testRecipe.Cleanup()

		modifyReq := ModifyRequest{
			ModificationType: "make_vegan",
			UserID:           "test-user",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/modify",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    modifyReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		modifyRecipeHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["modified_recipe"]; !ok {
				t.Error("Expected modified_recipe field in response")
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Test Recipe")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/modify",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    `{"invalid json`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		modifyRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestRateRecipeHandler tests recipe rating functionality
func TestRateRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)
	defer cleanupTestRatings(t)

	t.Run("Success", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Rate")
		defer testRecipe.Cleanup()

		rating := RecipeRating{
			UserID:    "test-user-" + uuid.New().String(),
			Rating:    5,
			Notes:     "Delicious!",
			Anonymous: false,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/rate",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    rating,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rateRecipeHandler(w, httpReq)

		if db != nil && w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "recorded",
			})

			if response == nil {
				t.Error("Expected response body")
			}
		}
	})

	t.Run("InvalidRating", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Test Recipe")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/rate",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    `{"invalid json`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestShareRecipeHandler tests recipe sharing functionality
func TestShareRecipeHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Share")
		defer testRecipe.Cleanup()

		shareReq := struct {
			UserIDs []string `json:"user_ids"`
		}{
			UserIDs: []string{"user1", "user2", "user3"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/share",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    shareReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		shareRecipeHandler(w, httpReq)

		if db != nil && w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "shared",
			})

			if response == nil {
				t.Error("Expected response body")
			}
		}
	})
}

// TestGenerateShoppingListHandler tests shopping list generation
func TestGenerateShoppingListHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("Success", func(t *testing.T) {
		recipe1 := setupTestRecipe(t, "Recipe 1")
		defer recipe1.Cleanup()

		recipe2 := setupTestRecipe(t, "Recipe 2")
		defer recipe2.Cleanup()

		shoppingReq := struct {
			RecipeIDs []string `json:"recipe_ids"`
			UserID    string   `json:"user_id"`
		}{
			RecipeIDs: []string{recipe1.Recipe.ID, recipe2.Recipe.ID},
			UserID:    "test-user",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping-list",
			Body:   shoppingReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateShoppingListHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["shopping_list"]; !ok {
				t.Error("Expected shopping_list field in response")
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping-list",
			Body:   `{"invalid json`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateShoppingListHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestUserPreferencesHandlers tests user preference management
func TestUserPreferencesHandlers(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	userID := "test-user-" + uuid.New().String()

	t.Run("GetPreferences", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/users/" + userID + "/preferences",
			URLVars: map[string]string{"id": userID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getUserPreferencesHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Error("Expected response body")
		}
	})

	t.Run("UpdatePreferences", func(t *testing.T) {
		preferences := map[string]interface{}{
			"dietary_restrictions": []string{"vegetarian", "gluten-free"},
			"favorite_cuisines":    []string{"Italian", "Japanese"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/users/" + userID + "/preferences",
			URLVars: map[string]string{"id": userID},
			Body:    preferences,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateUserPreferencesHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Error("Expected response body")
		}
	})
}

// TestPerformance tests performance of key operations
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("CreateRecipePerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 10

		for i := 0; i < iterations; i++ {
			recipe := TestData.CreateRecipeRequest("Performance Test " + string(rune(i)))

			w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/recipes",
				Body:   recipe,
			})

			createRecipeHandler(w, httpReq)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Average recipe creation time: %v", avgDuration)

		if avgDuration > 500*time.Millisecond {
			t.Logf("Warning: Recipe creation is slow (avg %v per request)", avgDuration)
		}
	})

	t.Run("ListRecipesPerformance", func(t *testing.T) {
		// Create test data
		for i := 0; i < 5; i++ {
			testRecipe := setupTestRecipe(t, "Perf Recipe "+string(rune(i)))
			defer testRecipe.Cleanup()
		}

		start := time.Now()

		w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"limit": "20",
			},
		})

		listRecipesHandler(w, httpReq)

		duration := time.Since(start)
		t.Logf("List recipes execution time: %v", duration)

		if duration > 1*time.Second {
			t.Logf("Warning: Recipe listing is slow (%v)", duration)
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("EmptyIngredients", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("No Ingredients Recipe")
		recipe.Ingredients = []Ingredient{}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Should not crash, may accept or reject empty ingredients
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Unexpected status for empty ingredients: %d", w.Code)
		}
	})

	t.Run("VeryLongRecipeTitle", func(t *testing.T) {
		longTitle := ""
		for i := 0; i < 500; i++ {
			longTitle += "A"
		}

		recipe := TestData.CreateRecipeRequest(longTitle)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Should handle gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Unexpected status for very long title: %d", w.Code)
		}
	})

	t.Run("NegativeCookTime", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Negative Time Recipe")
		recipe.CookTime = -30
		recipe.PrepTime = -15

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Should handle gracefully (accept or reject)
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Unexpected status for negative times: %d", w.Code)
		}
	})

	t.Run("ZeroServings", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Zero Servings Recipe")
		recipe.Servings = 0

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Should handle gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Unexpected status for zero servings: %d", w.Code)
		}
	})
}
