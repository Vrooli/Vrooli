// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestSearchPatternsComprehensive provides comprehensive coverage for search endpoint
func TestSearchPatternsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		validateFunc   func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "EmptyQuery",
			queryParams:    map[string]string{},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				if _, ok := response["patterns"]; !ok {
					t.Error("Expected patterns field")
				}
				if _, ok := response["total"]; !ok {
					t.Error("Expected total field")
				}
			},
		},
		{
			name:           "QueryWithTags",
			queryParams:    map[string]string{"tags": "testing"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				if _, ok := response["patterns"]; !ok {
					t.Error("Expected patterns field")
				}
			},
		},
		{
			name:           "QueryWithSection",
			queryParams:    map[string]string{"section": "Unit Testing"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				if _, ok := response["patterns"]; !ok {
					t.Error("Expected patterns field")
				}
			},
		},
		{
			name:           "InvalidLimitHandling",
			queryParams:    map[string]string{"limit": "not-a-number"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				// Should use default limit
				if limit, ok := response["limit"].(float64); ok && limit != 50 {
					t.Errorf("Expected default limit 50, got %v", limit)
				}
			},
		},
		{
			name:           "ZeroLimit",
			queryParams:    map[string]string{"limit": "0"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				// Should use default
				if _, ok := response["patterns"]; !ok {
					t.Error("Expected patterns field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/patterns/search",
				Query:  tt.queryParams,
			}

			w := makeHTTPRequest(searchPatternsHandler, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.validateFunc != nil && w.Code == http.StatusOK {
				response := assertJSONResponse(t, w, tt.expectedStatus)
				tt.validateFunc(t, response)
			}
		})
	}
}

// TestGetPatternComprehensive provides comprehensive pattern retrieval coverage
func TestGetPatternComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	tests := []struct {
		name           string
		patternID      string
		expectedStatus int
		checkTitle     string
	}{
		{
			name:           "ExistingPatternByID",
			patternID:      "test-pattern-1",
			expectedStatus: http.StatusOK,
			checkTitle:     "Test Pattern",
		},
		{
			name:           "NonExistentPattern",
			patternID:      "does-not-exist",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "EmptyID",
			patternID:      "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "SpecialCharactersInID",
			patternID:      "test-pattern-!@#",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/patterns/" + tt.patternID,
				URLVars: map[string]string{"id": tt.patternID},
			}

			w := makeHTTPRequest(getPatternHandler, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == http.StatusOK {
				response := assertJSONResponse(t, w, tt.expectedStatus)
				if title, ok := response["title"].(string); ok && tt.checkTitle != "" {
					if title != tt.checkTitle {
						t.Errorf("Expected title %s, got %s", tt.checkTitle, title)
					}
				}
			}
		})
	}
}

// TestRecipeHandlerComprehensive provides comprehensive recipe endpoint coverage
func TestRecipeHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	tests := []struct {
		name           string
		recipeID       string
		expectedStatus int
		checkField     string
		checkValue     string
	}{
		{
			name:           "ExistingRecipe",
			recipeID:       "test-recipe-1",
			expectedStatus: http.StatusOK,
			checkField:     "type",
			checkValue:     "greenfield",
		},
		{
			name:           "NonExistentRecipe",
			recipeID:       "no-such-recipe",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "EmptyRecipeID",
			recipeID:       "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/recipes/" + tt.recipeID,
				URLVars: map[string]string{"id": tt.recipeID},
			}

			w := makeHTTPRequest(getRecipeHandler, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == http.StatusOK && tt.checkField != "" {
				response := assertJSONResponse(t, w, tt.expectedStatus)
				if value, ok := response[tt.checkField].(string); ok {
					if value != tt.checkValue {
						t.Errorf("Expected %s = %s, got %s", tt.checkField, tt.checkValue, value)
					}
				}
			}
		})
	}
}

// TestCodeGenerationComprehensive tests all code generation scenarios
func TestCodeGenerationComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "ValidGoGeneration",
			requestBody: GenerationRequest{
				RecipeID:       "test-recipe-1",
				Language:       "go",
				Parameters:     map[string]interface{}{"key": "value"},
				TargetPlatform: "linux",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "MissingRecipeID",
			requestBody: GenerationRequest{
				RecipeID: "",
				Language: "go",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "MissingLanguage",
			requestBody: GenerationRequest{
				RecipeID: "test-recipe-1",
				Language: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "MalformedJSON",
			requestBody:    "not valid json {{{",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "NonExistentLanguageImpl",
			requestBody: GenerationRequest{
				RecipeID: "test-recipe-1",
				Language: "nonexistent",
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/recipes/generate",
				Body:   tt.requestBody,
			}

			w := makeHTTPRequest(generateCodeHandler, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestImplementationsComprehensive tests implementation filtering
func TestImplementationsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		validateFunc   func(t *testing.T, response []interface{})
	}{
		{
			name:           "WithValidRecipeID",
			queryParams:    map[string]string{"recipe_id": "test-recipe-1"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []interface{}) {
				if len(response) == 0 {
					t.Error("Expected at least one implementation")
				}
			},
		},
		{
			name:           "WithLanguageFilter",
			queryParams:    map[string]string{"recipe_id": "test-recipe-1", "language": "go"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []interface{}) {
				for _, item := range response {
					impl := item.(map[string]interface{})
					if lang, ok := impl["language"].(string); ok && lang != "go" {
						t.Errorf("Expected language go, got %s", lang)
					}
				}
			},
		},
		{
			name:           "MissingRecipeID",
			queryParams:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NonExistentRecipeID",
			queryParams:    map[string]string{"recipe_id": "does-not-exist"},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []interface{}) {
				if len(response) != 0 {
					t.Error("Expected empty array for non-existent recipe")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/implementations",
				Query:  tt.queryParams,
			}

			w := makeHTTPRequest(getImplementationsHandler, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == http.StatusOK && tt.validateFunc != nil {
				response := assertArrayResponse(t, w, tt.expectedStatus)
				tt.validateFunc(t, response)
			}
		})
	}
}

// TestChaptersAndStatsComprehensive tests aggregate endpoints
func TestChaptersAndStatsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("GetChapters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/chapters",
		}

		w := makeHTTPRequest(getChaptersHandler, req)
		response := assertArrayResponse(t, w, http.StatusOK)

		if len(response) > 0 {
			chapter := response[0].(map[string]interface{})
			if _, ok := chapter["name"]; !ok {
				t.Error("Expected name field in chapter")
			}
			if _, ok := chapter["pattern_count"]; !ok {
				t.Error("Expected pattern_count field in chapter")
			}
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/stats",
		}

		w := makeHTTPRequest(getStatsHandler, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["statistics"]; !ok {
			t.Error("Expected statistics field")
		}
		if _, ok := response["maturity_levels"]; !ok {
			t.Error("Expected maturity_levels field")
		}
		if _, ok := response["languages"]; !ok {
			t.Error("Expected languages field")
		}
	})
}

// TestGetFacetsComprehensive tests facet retrieval
func TestGetFacetsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("FacetsStructure", func(t *testing.T) {
		facets := getFacets()

		if _, ok := facets["chapters"]; !ok {
			t.Error("Expected chapters facet")
		}
		if _, ok := facets["maturity_levels"]; !ok {
			t.Error("Expected maturity_levels facet")
		}
		if _, ok := facets["tags"]; !ok {
			t.Error("Expected tags facet")
		}

		// Check that facets contain arrays
		if chapters, ok := facets["chapters"].([]string); !ok {
			t.Error("Expected chapters to be []string")
		} else {
			t.Logf("Found %d chapters", len(chapters))
		}

		if levels, ok := facets["maturity_levels"].([]string); !ok {
			t.Error("Expected maturity_levels to be []string")
		} else {
			t.Logf("Found %d maturity levels", len(levels))
		}

		if tags, ok := facets["tags"].([]string); !ok {
			t.Error("Expected tags to be []string")
		} else {
			t.Logf("Found %d tags", len(tags))
		}
	})
}
