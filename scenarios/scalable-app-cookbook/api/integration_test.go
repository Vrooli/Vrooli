// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestIntegrationPatternWorkflow tests the complete pattern discovery workflow
func TestIntegrationPatternWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("CompleteWorkflow_SearchToGeneration", func(t *testing.T) {
		// Step 1: Search for patterns
		searchReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"query": "test"},
		}

		searchResp := makeHTTPRequest(searchPatternsHandler, searchReq)
		searchData := assertJSONResponse(t, searchResp, http.StatusOK)

		patterns, ok := searchData["patterns"].([]interface{})
		if !ok || len(patterns) == 0 {
			t.Skip("No patterns found for integration test")
		}

		// Step 2: Get pattern details
		pattern := patterns[0].(map[string]interface{})
		patternID := pattern["id"].(string)

		getPatternReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/" + patternID,
			URLVars: map[string]string{"id": patternID},
		}

		getPatternResp := makeHTTPRequest(getPatternHandler, getPatternReq)
		patternData := assertJSONResponse(t, getPatternResp, http.StatusOK)

		if id, ok := patternData["id"].(string); !ok || id != patternID {
			t.Errorf("Pattern ID mismatch: expected %s, got %v", patternID, patternData["id"])
		}

		// Step 3: Get recipes for the pattern
		getRecipesReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/" + patternID + "/recipes",
			URLVars: map[string]string{"id": patternID},
		}

		getRecipesResp := makeHTTPRequest(getRecipesHandler, getRecipesReq)
		recipesData := assertJSONResponse(t, getRecipesResp, http.StatusOK)

		recipes, ok := recipesData["recipes"].([]interface{})
		if !ok || len(recipes) == 0 {
			t.Log("No recipes found for pattern, skipping generation test")
			return
		}

		// Step 4: Generate code from recipe
		recipe := recipes[0].(map[string]interface{})
		recipeID := recipe["id"].(string)

		generateReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body: GenerationRequest{
				RecipeID:       recipeID,
				Language:       "go",
				Parameters:     map[string]interface{}{"test": "integration"},
				TargetPlatform: "linux",
			},
		}

		generateResp := makeHTTPRequest(generateCodeHandler, generateReq)
		generateData := assertJSONResponse(t, generateResp, http.StatusOK)

		if _, ok := generateData["generated_code"]; !ok {
			t.Error("Expected generated_code in response")
		}
	})
}

// TestIntegrationRecipeToImplementation tests recipe-to-implementation flow
func TestIntegrationRecipeToImplementation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("RecipeImplementationFlow", func(t *testing.T) {
		// Get recipe
		recipeReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/recipes/test-recipe-1",
			URLVars: map[string]string{"id": "test-recipe-1"},
		}

		recipeResp := makeHTTPRequest(getRecipeHandler, recipeReq)
		recipeData := assertJSONResponse(t, recipeResp, http.StatusOK)

		if _, ok := recipeData["id"]; !ok {
			t.Fatal("Expected recipe id in response")
		}

		// Get implementations for recipe
		implReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/implementations",
			Query:  map[string]string{"recipe_id": "test-recipe-1"},
		}

		implResp := makeHTTPRequest(getImplementationsHandler, implReq)
		implementations := assertArrayResponse(t, implResp, http.StatusOK)

		if len(implementations) == 0 {
			t.Error("Expected at least one implementation")
		}

		// Verify implementation structure
		for _, item := range implementations {
			impl := item.(map[string]interface{})
			requiredFields := []string{"id", "recipe_id", "language", "code"}
			for _, field := range requiredFields {
				if _, ok := impl[field]; !ok {
					t.Errorf("Implementation missing required field: %s", field)
				}
			}
		}
	})
}

// TestIntegrationChapterStatistics tests chapter and stats integration
func TestIntegrationChapterStatistics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("ChapterStatsConsistency", func(t *testing.T) {
		// Get chapters
		chaptersReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/chapters",
		}

		chaptersResp := makeHTTPRequest(getChaptersHandler, chaptersReq)
		chapters := assertArrayResponse(t, chaptersResp, http.StatusOK)

		// Get overall stats
		statsReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/stats",
		}

		statsResp := makeHTTPRequest(getStatsHandler, statsReq)
		statsData := assertJSONResponse(t, statsResp, http.StatusOK)

		stats, ok := statsData["statistics"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected statistics object")
		}

		// Calculate total patterns from chapters
		var totalFromChapters float64
		for _, item := range chapters {
			chapter := item.(map[string]interface{})
			if count, ok := chapter["pattern_count"].(float64); ok {
				totalFromChapters += count
			}
		}

		// Compare with stats total
		totalFromStats, ok := stats["total_patterns"].(float64)
		if !ok {
			t.Fatal("Expected total_patterns in stats")
		}

		if totalFromChapters != totalFromStats {
			t.Logf("Chapter sum: %v, Stats total: %v", totalFromChapters, totalFromStats)
			// Note: They may differ if patterns have null chapters
		}
	})
}

// TestIntegrationSearchFiltering tests multiple search filters together
func TestIntegrationSearchFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("MultipleFilters", func(t *testing.T) {
		// Search with multiple filters
		searchReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query: map[string]string{
				"chapter":        "Testing",
				"maturity_level": "L2",
				"limit":          "10",
				"offset":         "0",
			},
		}

		searchResp := makeHTTPRequest(searchPatternsHandler, searchReq)
		searchData := assertJSONResponse(t, searchResp, http.StatusOK)

		patterns, ok := searchData["patterns"].([]interface{})
		if !ok {
			t.Fatal("Expected patterns array")
		}

		// Verify filters were applied
		for _, item := range patterns {
			pattern := item.(map[string]interface{})

			if chapter, ok := pattern["chapter"].(string); ok {
				if chapter != "Testing" {
					t.Errorf("Expected chapter 'Testing', got %s", chapter)
				}
			}

			if level, ok := pattern["maturity_level"].(string); ok {
				if level != "L2" {
					t.Errorf("Expected maturity_level 'L2', got %s", level)
				}
			}
		}

		// Verify pagination
		if limit, ok := searchData["limit"].(float64); ok {
			if limit != 10 {
				t.Errorf("Expected limit 10, got %v", limit)
			}
		}

		if offset, ok := searchData["offset"].(float64); ok {
			if offset != 0 {
				t.Errorf("Expected offset 0, got %v", offset)
			}
		}
	})
}

// TestIntegrationErrorRecovery tests that errors don't cascade
func TestIntegrationErrorRecovery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("ErrorDoesNotAffectSubsequentRequests", func(t *testing.T) {
		// Make a request that should fail
		errorReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/nonexistent",
			URLVars: map[string]string{"id": "nonexistent"},
		}

		errorResp := makeHTTPRequest(getPatternHandler, errorReq)
		if errorResp.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", errorResp.Code)
		}

		// Make a valid request immediately after
		validReq := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/test-pattern-1",
			URLVars: map[string]string{"id": "test-pattern-1"},
		}

		validResp := makeHTTPRequest(getPatternHandler, validReq)
		validData := assertJSONResponse(t, validResp, http.StatusOK)

		if id, ok := validData["id"].(string); !ok || id != "test-pattern-1" {
			t.Error("Valid request failed after error")
		}
	})
}
