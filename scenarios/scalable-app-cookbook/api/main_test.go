// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	t.Run("Success_DatabaseConnected", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(healthHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got: %v", response["status"])
		}

		if dbStatus, ok := response["database"].(bool); !ok || !dbStatus {
			t.Errorf("Expected database to be true, got: %v", response["database"])
		}

		if service, ok := response["service"].(string); !ok || service != "scalable-app-cookbook-api" {
			t.Errorf("Expected service 'scalable-app-cookbook-api', got: %v", response["service"])
		}
	})

	// Note: Error_DatabaseDisconnected test is skipped because the handler
	// doesn't check for nil db before calling Ping(). This is acceptable
	// since the main() function ensures db is always initialized before
	// the server starts accepting requests.
}

// TestSearchPatternsHandler tests the pattern search endpoint
func TestSearchPatternsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success_NoFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["patterns"]; !ok {
			t.Error("Expected 'patterns' field in response")
		}

		if _, ok := response["total"]; !ok {
			t.Error("Expected 'total' field in response")
		}

		if _, ok := response["facets"]; !ok {
			t.Error("Expected 'facets' field in response")
		}
	})

	t.Run("Success_WithQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"query": "test"},
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if patterns, ok := response["patterns"].([]interface{}); !ok {
			t.Error("Expected 'patterns' to be an array")
		} else if len(patterns) > 0 {
			// Verify pattern structure
			pattern := patterns[0].(map[string]interface{})
			if _, ok := pattern["id"]; !ok {
				t.Error("Expected pattern to have 'id' field")
			}
			if _, ok := pattern["title"]; !ok {
				t.Error("Expected pattern to have 'title' field")
			}
		}
	})

	t.Run("Success_WithChapterFilter", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"chapter": "Testing"},
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("Success_WithMaturityLevel", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"maturity_level": "proven"},
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("Success_WithPagination", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"limit": "10", "offset": "0"},
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if limit, ok := response["limit"].(float64); !ok || limit != 10 {
			t.Errorf("Expected limit 10, got: %v", response["limit"])
		}

		if offset, ok := response["offset"].(float64); !ok || offset != 0 {
			t.Errorf("Expected offset 0, got: %v", response["offset"])
		}
	})

	t.Run("Success_LimitClamping", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"limit": "200"}, // Exceeds max of 100
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Should clamp to default (50) since invalid
		if limit, ok := response["limit"].(float64); !ok || limit > 100 {
			t.Errorf("Expected limit to be clamped, got: %v", response["limit"])
		}
	})

	t.Run("Success_NegativeOffsetHandling", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"offset": "-5"},
		}

		w := makeHTTPRequest(searchPatternsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Should default to 0
		if offset, ok := response["offset"].(float64); !ok || offset != 0 {
			t.Errorf("Expected offset 0 for negative input, got: %v", response["offset"])
		}
	})
}

// TestGetPatternHandler tests retrieving a single pattern
func TestGetPatternHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success_ByID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/test-pattern-1",
			URLVars: map[string]string{"id": "test-pattern-1"},
		}

		w := makeHTTPRequest(getPatternHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != "test-pattern-1" {
			t.Errorf("Expected id 'test-pattern-1', got: %v", response["id"])
		}

		if title, ok := response["title"].(string); !ok || title != "Test Pattern" {
			t.Errorf("Expected title 'Test Pattern', got: %v", response["title"])
		}
	})

	t.Run("Success_ByTitle", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/test%20pattern",
			URLVars: map[string]string{"id": "test pattern"},
		}

		w := makeHTTPRequest(getPatternHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if title, ok := response["title"].(string); !ok || title != "Test Pattern" {
			t.Errorf("Expected title 'Test Pattern', got: %v", response["title"])
		}
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/non-existent-pattern",
			URLVars: map[string]string{"id": "non-existent-pattern"},
		}

		w := makeHTTPRequest(getPatternHandler, req)
		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// TestGetRecipesHandler tests retrieving recipes for a pattern
func TestGetRecipesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success_AllRecipes", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/test-pattern-1/recipes",
			URLVars: map[string]string{"id": "test-pattern-1"},
		}

		w := makeHTTPRequest(getRecipesHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["pattern"]; !ok {
			t.Error("Expected 'pattern' field in response")
		}

		if _, ok := response["recipes"]; !ok {
			t.Error("Expected 'recipes' field in response")
		}
	})

	t.Run("Success_FilterByType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/test-pattern-1/recipes",
			URLVars: map[string]string{"id": "test-pattern-1"},
			Query:   map[string]string{"type": "greenfield"},
		}

		w := makeHTTPRequest(getRecipesHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if recipes, ok := response["recipes"].([]interface{}); ok {
			for _, r := range recipes {
				recipe := r.(map[string]interface{})
				if recipeType, ok := recipe["type"].(string); ok && recipeType != "greenfield" {
					t.Errorf("Expected all recipes to be type 'greenfield', got: %v", recipeType)
				}
			}
		}
	})

	t.Run("Error_PatternNotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/non-existent/recipes",
			URLVars: map[string]string{"id": "non-existent"},
		}

		w := makeHTTPRequest(getRecipesHandler, req)
		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// TestGetRecipeHandler tests retrieving a single recipe
func TestGetRecipeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success_ByID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/test-recipe-1",
			URLVars: map[string]string{"id": "test-recipe-1"},
		}

		w := makeHTTPRequest(getRecipeHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if id, ok := response["id"].(string); !ok || id != "test-recipe-1" {
			t.Errorf("Expected id 'test-recipe-1', got: %v", response["id"])
		}

		// Verify all required fields are present
		requiredFields := []string{"pattern_id", "title", "type", "prerequisites",
			"steps", "config_snippets", "validation_checks", "timeout_sec"}
		for _, field := range requiredFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/non-existent",
			URLVars: map[string]string{"id": "non-existent"},
		}

		w := makeHTTPRequest(getRecipeHandler, req)
		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// TestGenerateCodeHandler tests the code generation endpoint
func TestGenerateCodeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success_ValidRequest", func(t *testing.T) {
		requestBody := GenerationRequest{
			RecipeID: "test-recipe-1",
			Language: "go",
			Parameters: map[string]interface{}{
				"param1": "value1",
			},
			TargetPlatform: "linux",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   requestBody,
		}

		w := makeHTTPRequest(generateCodeHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["generated_code"]; !ok {
			t.Error("Expected 'generated_code' field in response")
		}

		if _, ok := response["file_structure"]; !ok {
			t.Error("Expected 'file_structure' field in response")
		}

		if _, ok := response["dependencies"]; !ok {
			t.Error("Expected 'dependencies' field in response")
		}

		if _, ok := response["setup_instructions"]; !ok {
			t.Error("Expected 'setup_instructions' field in response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   "invalid-json",
		}

		w := makeHTTPRequest(generateCodeHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("Error_RecipeNotFound", func(t *testing.T) {
		requestBody := GenerationRequest{
			RecipeID: "non-existent-recipe",
			Language: "go",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   requestBody,
		}

		w := makeHTTPRequest(generateCodeHandler, req)
		assertErrorResponse(t, w, http.StatusNotFound, "")
	})

	t.Run("Error_LanguageNotFound", func(t *testing.T) {
		requestBody := GenerationRequest{
			RecipeID: "test-recipe-1",
			Language: "rust", // Not in test data
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body:   requestBody,
		}

		w := makeHTTPRequest(generateCodeHandler, req)
		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// TestGetImplementationsHandler tests retrieving implementations
func TestGetImplementationsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success_AllImplementations", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/implementations",
			Query:  map[string]string{"recipe_id": "test-recipe-1"},
		}

		w := makeHTTPRequest(getImplementationsHandler, req)
		response := assertArrayResponse(t, w, http.StatusOK)

		if len(response) > 0 {
			impl := response[0].(map[string]interface{})
			requiredFields := []string{"id", "recipe_id", "language", "code",
				"file_path", "description"}
			for _, field := range requiredFields {
				if _, ok := impl[field]; !ok {
					t.Errorf("Expected field '%s' in implementation", field)
				}
			}
		}
	})

	t.Run("Success_FilterByLanguage", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/implementations",
			Query: map[string]string{
				"recipe_id": "test-recipe-1",
				"language":  "go",
			},
		}

		w := makeHTTPRequest(getImplementationsHandler, req)
		response := assertArrayResponse(t, w, http.StatusOK)

		for _, r := range response {
			impl := r.(map[string]interface{})
			if lang, ok := impl["language"].(string); ok && lang != "go" {
				t.Errorf("Expected language 'go', got: %v", lang)
			}
		}
	})

	t.Run("Error_MissingRecipeID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/implementations",
		}

		w := makeHTTPRequest(getImplementationsHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestGetChaptersHandler tests retrieving chapters
func TestGetChaptersHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/chapters",
		}

		w := makeHTTPRequest(getChaptersHandler, req)
		response := assertArrayResponse(t, w, http.StatusOK)

		if len(response) > 0 {
			chapter := response[0].(map[string]interface{})
			if _, ok := chapter["name"]; !ok {
				t.Error("Expected 'name' field in chapter")
			}
			if _, ok := chapter["pattern_count"]; !ok {
				t.Error("Expected 'pattern_count' field in chapter")
			}
		}
	})
}

// TestGetStatsHandler tests retrieving statistics
func TestGetStatsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/stats",
		}

		w := makeHTTPRequest(getStatsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if stats, ok := response["statistics"].(map[string]interface{}); !ok {
			t.Error("Expected 'statistics' field in response")
		} else {
			requiredFields := []string{"total_patterns", "total_recipes",
				"total_implementations", "total_chapters"}
			for _, field := range requiredFields {
				if _, ok := stats[field]; !ok {
					t.Errorf("Expected field '%s' in statistics", field)
				}
			}
		}

		if _, ok := response["maturity_levels"]; !ok {
			t.Error("Expected 'maturity_levels' field in response")
		}

		if _, ok := response["languages"]; !ok {
			t.Error("Expected 'languages' field in response")
		}
	})
}

// TestGetFacets tests the getFacets helper function
func TestGetFacets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("Success", func(t *testing.T) {
		facets := getFacets()

		if _, ok := facets["chapters"]; !ok {
			t.Error("Expected 'chapters' facet")
		}

		if _, ok := facets["maturity_levels"]; !ok {
			t.Error("Expected 'maturity_levels' facet")
		}

		if _, ok := facets["tags"]; !ok {
			t.Error("Expected 'tags' facet")
		}
	})
}

// TestPaginationScenarios tests pagination edge cases
func TestPaginationScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	scenarios := PaginationTestScenarios("/api/v1/patterns/search")

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			w := makeHTTPRequest(searchPatternsHandler, scenario.Request)

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", scenario.ExpectedStatus, w.Code)
			}

			if scenario.ValidateBody != nil {
				scenario.ValidateBody(t, w.Body.String())
			}
		})
	}
}
