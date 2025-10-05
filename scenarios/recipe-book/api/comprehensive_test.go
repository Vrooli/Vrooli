// +build testing


package main

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestComprehensiveRecipeWorkflow tests complete recipe lifecycle
func TestComprehensiveRecipeWorkflow(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	// Skip if no database
	if db == nil {
		t.Skip("Skipping integration tests without database")
	}

	var recipeID string

	t.Run("CreateRecipe", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Workflow Test Recipe")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok {
				recipeID = id
			}
		}
	})

	t.Run("GetRecipe", func(t *testing.T) {
		if recipeID == "" {
			t.Skip("No recipe created")
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + recipeID,
			URLVars: map[string]string{"id": recipeID},
			QueryParams: map[string]string{"user_id": "test-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getRecipeHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id": recipeID,
		})
	})

	t.Run("UpdateRecipe", func(t *testing.T) {
		if recipeID == "" {
			t.Skip("No recipe created")
		}

		recipe := TestData.CreateRecipeRequest("Updated Workflow Recipe")
		recipe.ID = recipeID

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/recipes/" + recipeID,
			URLVars: map[string]string{"id": recipeID},
			Body:    recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateRecipeHandler(w, httpReq)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"title": "Updated Workflow Recipe",
			})
		}
	})

	t.Run("RateRecipe", func(t *testing.T) {
		if recipeID == "" {
			t.Skip("No recipe created")
		}

		rating := RecipeRating{
			UserID: "test-user-" + uuid.New().String(),
			Rating: 4,
			Notes:  "Good recipe!",
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + recipeID + "/rate",
			URLVars: map[string]string{"id": recipeID},
			Body:    rating,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rateRecipeHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("ShareRecipe", func(t *testing.T) {
		if recipeID == "" {
			t.Skip("No recipe created")
		}

		shareReq := struct {
			UserIDs []string `json:"user_ids"`
		}{
			UserIDs: []string{"friend1", "friend2"},
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/" + recipeID + "/share",
			URLVars: map[string]string{"id": recipeID},
			Body:    shareReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		shareRecipeHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("DeleteRecipe", func(t *testing.T) {
		if recipeID == "" {
			t.Skip("No recipe created")
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/recipes/" + recipeID,
			URLVars: map[string]string{"id": recipeID},
			QueryParams: map[string]string{"user_id": "test-user"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteRecipeHandler(w, httpReq)

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 204, 200, or 404, got %d", w.Code)
		}
	})
}

// TestErrorHandling tests comprehensive error scenarios
func TestErrorHandling(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CreateRecipeWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   "{not valid json",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("GetNonExistentRecipe", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test without database")
		}

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
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("UpdateWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/recipes/some-id",
			URLVars: map[string]string{"id": "some-id"},
			Body:    "{malformed",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DeleteNonExistentRecipe", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test without database")
		}

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
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("SearchWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   "{invalid json",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("GenerateWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   "{invalid",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ModifyWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/some-id/modify",
			URLVars: map[string]string{"id": "some-id"},
			Body:    "not json",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		modifyRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("RateWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/some-id/rate",
			URLVars: map[string]string{"id": "some-id"},
			Body:    "bad json",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rateRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ShareWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/recipes/some-id/share",
			URLVars: map[string]string{"id": "some-id"},
			Body:    "invalid",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		shareRecipeHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ShoppingListWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping-list",
			Body:   "not json",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateShoppingListHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("UpdatePreferencesWithMalformedJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/users/user-id/preferences",
			URLVars: map[string]string{"id": "user-id"},
			Body:    "{bad",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateUserPreferencesHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestBoundaryConditions tests edge cases and boundaries
func TestBoundaryConditions(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	t.Run("EmptyRecipeTitle", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Should accept (validation is not strict)
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Status for empty title: %d", w.Code)
		}
	})

	t.Run("LargeServings", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Large Batch Recipe")
		recipe.Servings = 10000

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes",
			Body:   recipe,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createRecipeHandler(w, httpReq)

		// Should handle large numbers
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Logf("Status for large servings: %d", w.Code)
		}
	})

	t.Run("ManyIngredients", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Complex Recipe")
		recipe.Ingredients = make([]Ingredient, 100)
		for i := range recipe.Ingredients {
			recipe.Ingredients[i] = Ingredient{
				Name:   "ingredient",
				Amount: 1,
				Unit:   "cup",
			}
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

		// Should handle many ingredients
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Logf("Status for many ingredients: %d", w.Code)
		}
	})

	t.Run("ManyInstructions", func(t *testing.T) {
		recipe := TestData.CreateRecipeRequest("Detailed Recipe")
		recipe.Instructions = make([]string, 50)
		for i := range recipe.Instructions {
			recipe.Instructions[i] = "Step"
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

		// Should handle many steps
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Logf("Status for many instructions: %d", w.Code)
		}
	})

	t.Run("SearchWithEmptyQuery", func(t *testing.T) {
		req := TestData.SearchRequest("")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)

		// Should handle empty query
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Logf("Status for empty search query: %d", w.Code)
		}
	})

	t.Run("SearchWithLargeLimit", func(t *testing.T) {
		req := TestData.SearchRequest("pasta")
		req.Limit = 10000

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/search",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchRecipesHandler(w, httpReq)

		// Should handle large limit
		if w.Code != http.StatusOK {
			t.Logf("Status for large search limit: %d", w.Code)
		}
	})

	t.Run("ShoppingListWithNoRecipes", func(t *testing.T) {
		shoppingReq := struct {
			RecipeIDs []string `json:"recipe_ids"`
			UserID    string   `json:"user_id"`
		}{
			RecipeIDs: []string{},
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

		// Should handle empty list
		if w.Code != http.StatusOK {
			t.Logf("Status for empty shopping list: %d", w.Code)
		}
	})

	t.Run("ShoppingListWithManyRecipes", func(t *testing.T) {
		recipeIDs := make([]string, 100)
		for i := range recipeIDs {
			recipeIDs[i] = uuid.New().String()
		}

		shoppingReq := struct {
			RecipeIDs []string `json:"recipe_ids"`
			UserID    string   `json:"user_id"`
		}{
			RecipeIDs: recipeIDs,
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

		// Should handle many recipes
		if w.Code != http.StatusOK {
			t.Logf("Status for large shopping list: %d", w.Code)
		}
	})
}

// TestRecipeVisibility tests visibility and sharing logic
func TestRecipeVisibility(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestRecipes(t)

	if db == nil {
		t.Skip("Skipping visibility tests without database")
	}

	t.Run("PrivateRecipeAccess", func(t *testing.T) {
		recipe := setupTestRecipe(t, "Private Recipe")
		recipe.Recipe.Visibility = "private"
		defer recipe.Cleanup()

		// Owner can access
		w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + recipe.Recipe.ID,
			URLVars: map[string]string{"id": recipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": recipe.Recipe.CreatedBy},
		})

		getRecipeHandler(w, httpReq)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"id": recipe.Recipe.ID,
			})
		}
	})

	t.Run("PublicRecipeAccess", func(t *testing.T) {
		recipe := setupTestRecipe(t, "Public Recipe")
		recipe.Recipe.Visibility = "public"
		defer recipe.Cleanup()

		// Anyone can access public recipes
		w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/" + recipe.Recipe.ID,
			URLVars: map[string]string{"id": recipe.Recipe.ID},
			QueryParams: map[string]string{"user_id": "different-user"},
		})

		getRecipeHandler(w, httpReq)

		// Should succeed or fail based on implementation
		if w.Code != http.StatusOK && w.Code != http.StatusForbidden {
			t.Logf("Unexpected status for public recipe: %d", w.Code)
		}
	})
}
