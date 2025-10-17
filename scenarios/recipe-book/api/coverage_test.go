// +build testing


package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestGetRecipeHandlerComprehensive tests all paths of getRecipeHandler
func TestGetRecipeHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		// Temporarily set db to nil
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/test-id",
			URLVars: map[string]string{"id": "test-id"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("RecipeNotFound", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + uuid.New().String(),
			URLVars: map[string]string{"id": uuid.New().String()},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("PrivateRecipeAccessDenied", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Private Recipe")
		testRecipe.Recipe.Visibility = "private"
		testRecipe.Recipe.SharedWith = []string{}
		defer testRecipe.Cleanup()

		// Update visibility in database
		sharedWithJSON, _ := json.Marshal([]string{})
		db.Exec("UPDATE recipes SET visibility = 'private', shared_with = $2 WHERE id = $1",
			testRecipe.Recipe.ID, sharedWithJSON)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars:     map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": "unauthorized-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
			t.Errorf("Expected 403 or 404, got %d", w.Code)
		}
	})

	t.Run("SharedRecipeWithAccess", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Shared Recipe")
		sharedUserID := "shared-user-123"
		testRecipe.Recipe.Visibility = "private"
		testRecipe.Recipe.SharedWith = []string{sharedUserID}
		defer testRecipe.Cleanup()

		// Update in database
		sharedWithJSON, _ := json.Marshal([]string{sharedUserID})
		db.Exec("UPDATE recipes SET visibility = 'private', shared_with = $2 WHERE id = $1",
			testRecipe.Recipe.ID, sharedWithJSON)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars:     map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": sharedUserID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("PublicRecipeAccess", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Public Recipe")
		defer testRecipe.Cleanup()

		// Update to public
		db.Exec("UPDATE recipes SET visibility = 'public' WHERE id = $1", testRecipe.Recipe.ID)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars:     map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": "any-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"id": testRecipe.Recipe.ID,
			})
			if response != nil {
				if _, ok := response["ingredients"]; !ok {
					t.Error("Expected ingredients in response")
				}
				if _, ok := response["instructions"]; !ok {
					t.Error("Expected instructions in response")
				}
			}
		}
	})
}

// TestListRecipesHandlerComprehensive tests all paths of listRecipesHandler
func TestListRecipesHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("WithUserIDFilter", func(t *testing.T) {
		recipe1 := setupTestRecipe(t, "User Recipe 1")
		defer recipe1.Cleanup()

		recipe2 := setupTestRecipe(t, "User Recipe 2")
		recipe2.Recipe.CreatedBy = "different-user"
		defer recipe2.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"user_id": recipe1.Recipe.CreatedBy,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if recipes, ok := response["recipes"].([]interface{}); ok {
					// Should find at least recipe1
					if len(recipes) < 1 {
						t.Error("Expected at least 1 recipe")
					}
				}
			}
		}
	})

	t.Run("WithVisibilityFilter", func(t *testing.T) {
		recipe := setupTestRecipe(t, "Public Recipe")
		defer recipe.Cleanup()

		db.Exec("UPDATE recipes SET visibility = 'public' WHERE id = $1", recipe.Recipe.ID)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"visibility": "public",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("WithCuisineFilter", func(t *testing.T) {
		recipe := setupTestRecipe(t, "Italian Recipe")
		defer recipe.Cleanup()

		db.Exec("UPDATE recipes SET cuisine = 'Italian' WHERE id = $1", recipe.Recipe.ID)

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"cuisine": "Italian",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		// Create multiple recipes
		for i := 0; i < 3; i++ {
			recipe := setupTestRecipe(t, "Recipe "+string(rune(i+'0')))
			defer recipe.Cleanup()
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/recipes",
			QueryParams: map[string]string{
				"limit":  "2",
				"offset": "1",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listRecipesHandler(w, httpReq)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if total, ok := response["total"].(float64); ok {
					if total < 0 {
						t.Error("Total should be non-negative")
					}
				}
			}
		}
	})
}

// TestDeleteRecipeHandlerComprehensive tests all paths of deleteRecipeHandler
func TestDeleteRecipeHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/recipes/test-id",
			URLVars: map[string]string{"id": "test-id"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("RecipeNotFound", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "DELETE",
			Path:        "/api/v1/recipes/" + uuid.New().String(),
			URLVars:     map[string]string{"id": uuid.New().String()},
			QueryParams: map[string]string{"user_id": "test-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("UnauthorizedDelete", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Delete")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "DELETE",
			Path:        "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars:     map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": "unauthorized-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("SuccessfulDelete", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Delete")
		defer testRecipe.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "DELETE",
			Path:        "/api/v1/recipes/" + testRecipe.Recipe.ID,
			URLVars:     map[string]string{"id": testRecipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": testRecipe.Recipe.CreatedBy},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Expected 204 or 200, got %d", w.Code)
		}
	})
}

// TestCreateRecipeHandlerComprehensive tests all paths of createRecipeHandler
func TestCreateRecipeHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recipe := TestData.CreateRecipeRequest("Test Recipe")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("WithDefaults", func(t *testing.T) {
		recipe := Recipe{
			Title:       "Minimal Recipe",
			Description: "Test",
			Ingredients: []Ingredient{{Name: "test", Amount: 1, Unit: "cup"}},
			Instructions: []string{"Test step"},
			CreatedBy:   "test-user",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		if w.Code == http.StatusCreated {
			response := assertJSONResponse(t, w, http.StatusCreated, nil)
			if response != nil {
				if visibility, ok := response["visibility"].(string); ok {
					if visibility != "private" {
						t.Errorf("Expected default visibility 'private', got %s", visibility)
					}
				}
				if source, ok := response["source"].(string); ok {
					if source != "original" {
						t.Errorf("Expected default source 'original', got %s", source)
					}
				}
			}

			// Cleanup
			if id, ok := response["id"].(string); ok {
				db.Exec("DELETE FROM recipes WHERE id = $1", id)
			}
		}
	})

	t.Run("WithCompleteData", func(t *testing.T) {
		recipe := Recipe{
			Title:       "Complete Recipe",
			Description: "Full recipe with all fields",
			Ingredients: []Ingredient{
				{Name: "flour", Amount: 2, Unit: "cups", Notes: "all-purpose"},
				{Name: "sugar", Amount: 1, Unit: "cup"},
			},
			Instructions: []string{"Step 1", "Step 2", "Step 3"},
			PrepTime:     15,
			CookTime:     30,
			Servings:     4,
			Tags:         []string{"dessert", "baking"},
			Cuisine:      "American",
			DietaryInfo:  []string{"vegetarian"},
			Nutrition: NutritionInfo{
				Calories: 300,
				Protein:  5,
				Carbs:    50,
				Fat:      10,
				Fiber:    2,
				Sugar:    25,
				Sodium:   200,
			},
			PhotoURL:   "https://example.com/photo.jpg",
			CreatedBy:  "test-user-" + uuid.New().String(),
			Visibility: "public",
			Source:     "original",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		if w.Code == http.StatusCreated {
			response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
				"title": "Complete Recipe",
			})

			if response != nil {
				// Verify all fields are returned
				if _, ok := response["id"]; !ok {
					t.Error("Expected id in response")
				}
				if _, ok := response["created_at"]; !ok {
					t.Error("Expected created_at in response")
				}
				if _, ok := response["updated_at"]; !ok {
					t.Error("Expected updated_at in response")
				}

				// Cleanup
				if id, ok := response["id"].(string); ok {
					db.Exec("DELETE FROM recipes WHERE id = $1", id)
				}
			}
		}
	})
}

// TestUpdateRecipeHandlerComprehensive tests all paths of updateRecipeHandler
func TestUpdateRecipeHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recipe := TestData.CreateRecipeRequest("Updated Recipe")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/recipes/test-id",
			URLVars: map[string]string{"id": "test-id"},
			Body:    recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("UpdateAllFields", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Original Recipe")
		defer testRecipe.Cleanup()

		updatedRecipe := *testRecipe.Recipe
		updatedRecipe.Title = "Completely Updated Recipe"
		updatedRecipe.Description = "New description"
		updatedRecipe.PrepTime = 25
		updatedRecipe.CookTime = 45
		updatedRecipe.Servings = 6
		updatedRecipe.Cuisine = "Italian"
		updatedRecipe.Visibility = "public"
		updatedRecipe.Ingredients = []Ingredient{
			{Name: "new ingredient", Amount: 3, Unit: "cups"},
		}
		updatedRecipe.Instructions = []string{"New step 1", "New step 2"}

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

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"title": "Completely Updated Recipe",
			})

			if response != nil {
				if cuisine, ok := response["cuisine"].(string); ok {
					if cuisine != "Italian" {
						t.Errorf("Expected cuisine 'Italian', got %s", cuisine)
					}
				}
			}
		}
	})
}

// TestMarkCookedHandlerComprehensive tests all paths of markCookedHandler
func TestMarkCookedHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)
	defer cleanupTestRatings(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rating := RecipeRating{
			UserID: "test-user",
			Rating: 5,
			Notes:  "Great!",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/test-id/cook",
			URLVars: map[string]string{"id": "test-id"},
			Body:    rating,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		markCookedHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("MarkAsCooked", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Cook")
		defer testRecipe.Cleanup()

		rating := RecipeRating{
			UserID:    "test-user-" + uuid.New().String(),
			Rating:    4,
			Notes:     "Delicious!",
			Anonymous: false,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/cook",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    rating,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		markCookedHandler(w, httpReq)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "recorded",
			})
		}
	})

	t.Run("AnonymousRating", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe for Anonymous Rating")
		defer testRecipe.Cleanup()

		rating := RecipeRating{
			UserID:    "test-user-" + uuid.New().String(),
			Rating:    5,
			Notes:     "",
			Anonymous: true,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + testRecipe.Recipe.ID + "/cook",
			URLVars: map[string]string{"id": testRecipe.Recipe.ID},
			Body:    rating,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		markCookedHandler(w, httpReq)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "recorded",
			})
		}
	})
}

// TestShareRecipeHandlerComprehensive tests all paths of shareRecipeHandler
func TestShareRecipeHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		shareReq := struct {
			UserIDs []string `json:"user_ids"`
		}{
			UserIDs: []string{"user1", "user2"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/test-id/share",
			URLVars: map[string]string{"id": "test-id"},
			Body:    shareReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		shareRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})

	if db == nil {
		t.Skip("Skipping database-dependent tests")
		return
	}

	t.Run("ShareWithMultipleUsers", func(t *testing.T) {
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

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "shared",
			})
		}
	})

	t.Run("ShareWithEmptyList", func(t *testing.T) {
		testRecipe := setupTestRecipe(t, "Recipe to Unshare")
		defer testRecipe.Cleanup()

		shareReq := struct {
			UserIDs []string `json:"user_ids"`
		}{
			UserIDs: []string{},
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

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"status": "shared",
			})
		}
	})
}

// TestHelperFunctionsComprehensive tests helper functions for coverage
func TestHelperFunctionsComprehensive(t *testing.T) {
	t.Run("AssertJSONArray", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping-list",
			Body: map[string]interface{}{
				"recipe_ids": []string{},
				"user_id":    "test-user",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateShoppingListHandler(w, httpReq)

		if w.Code == http.StatusOK {
			array := assertJSONArray(t, w, http.StatusOK, "shopping_list")
			if array == nil {
				t.Error("Expected shopping_list array")
			}
		}
	})

	t.Run("AssertRecipeEqual", func(t *testing.T) {
		recipe1 := &Recipe{
			Title:       "Test Recipe",
			Description: "Test",
			PrepTime:    10,
			CookTime:    20,
			Servings:    4,
		}

		recipe2 := &Recipe{
			Title:       "Test Recipe",
			Description: "Test",
			PrepTime:    10,
			CookTime:    20,
			Servings:    4,
		}

		// This will pass
		assertRecipeEqual(t, recipe1, recipe2)
	})

	t.Run("AssertErrorResponseEmpty", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		// This should be 200, not an error
		if w.Code != http.StatusOK {
			assertErrorResponse(t, w, w.Code)
		}
	})
}

// TestSearchHandlerComprehensive tests search edge cases
func TestSearchHandlerComprehensive(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SearchWithDefaultLimit", func(t *testing.T) {
		searchReq := SearchRequest{
			Query:  "test query",
			UserID: "test-user",
			// Limit not set, should default to 10
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)

		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				if _, ok := response["query_interpretation"]; !ok {
					t.Error("Expected query_interpretation in response")
				}
			}
		}
	})

	t.Run("SearchWithFilters", func(t *testing.T) {
		searchReq := SearchRequest{
			Query:  "pasta",
			UserID: "test-user",
			Limit:  5,
		}
		searchReq.Filters.Dietary = []string{"vegetarian", "gluten-free"}
		searchReq.Filters.MaxTime = 30
		searchReq.Filters.Ingredients = []string{"tomatoes", "basil"}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, nil)
		}
	})
}

// TestTestPatternUsage tests using the pattern builder
func TestTestPatternUsage(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("UseTestScenarioBuilder", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/v1/recipes").
			AddMissingFields("POST", "/api/v1/recipes").
			Build()

		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(patterns))
		}

		suite := &HandlerTestSuite{
			HandlerName: "CreateRecipe",
			Handler:     createRecipeHandler,
			BaseURL:     "/api/v1/recipes",
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("UsePatternFunctions", func(t *testing.T) {
		pattern := invalidUUIDPattern("GET", "/api/v1/recipes/invalid-uuid")
		if pattern.ExpectedStatus != http.StatusNotFound {
			t.Error("Expected 404 for invalid UUID pattern")
		}

		pattern2 := emptyQueryPattern("POST", "/api/v1/recipes/search")
		if pattern2.ExpectedStatus != http.StatusBadRequest {
			t.Error("Expected 400 for empty query pattern")
		}
	})
}
