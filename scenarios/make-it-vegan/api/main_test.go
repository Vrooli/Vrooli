package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Initialize test database
	veganDB = InitVeganDatabase()
	// Initialize disabled cache for predictable testing
	cache = &CacheClient{
		redis:  nil,
		enable: false,
	}
	os.Exit(m.Run())
}

// Helper to create test router
func setupTestRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/api/check", checkIngredients).Methods("POST")
	router.HandleFunc("/api/substitute", findSubstitute).Methods("POST")
	router.HandleFunc("/api/veganize", veganizeRecipe).Methods("POST")
	router.HandleFunc("/api/products", getCommonProducts).Methods("GET")
	router.HandleFunc("/api/nutrition", getNutrition).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")
	return router
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("HealthCheckReturnsOK", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expected := "OK"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
		}
	})
}

// TestCheckIngredients tests the ingredient checking endpoint
func TestCheckIngredients(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	tests := []struct {
		name           string
		ingredients    string
		expectedVegan  bool
		expectNonVegan bool
	}{
		{
			name:           "AllVeganIngredients",
			ingredients:    "flour, sugar, salt, water",
			expectedVegan:  true,
			expectNonVegan: false,
		},
		{
			name:           "NonVeganIngredients",
			ingredients:    "milk, eggs, butter",
			expectedVegan:  false,
			expectNonVegan: true,
		},
		{
			name:           "MixedIngredients",
			ingredients:    "flour, milk, sugar",
			expectedVegan:  false,
			expectNonVegan: true,
		},
		{
			name:           "VeganMilkException",
			ingredients:    "soy milk, almond milk, oat milk",
			expectedVegan:  true,
			expectNonVegan: false,
		},
		{
			name:           "VeganButterException",
			ingredients:    "peanut butter, almond butter",
			expectedVegan:  true,
			expectNonVegan: false,
		},
		{
			name:           "EmptyIngredients",
			ingredients:    "",
			expectedVegan:  true,
			expectNonVegan: false,
		},
		{
			name:           "SingleNonVeganItem",
			ingredients:    "cheese",
			expectedVegan:  false,
			expectNonVegan: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := CheckRequest{Ingredients: tt.ingredients}
			body, _ := json.Marshal(reqBody)

			req, err := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			isVegan, ok := response["isVegan"].(bool)
			if !ok {
				t.Fatal("isVegan field missing or wrong type")
			}

			if isVegan != tt.expectedVegan {
				t.Errorf("Expected isVegan=%v, got %v. Response: %v", tt.expectedVegan, isVegan, response)
			}

			if tt.expectNonVegan {
				if _, exists := response["nonVeganItems"]; !exists {
					t.Error("Expected nonVeganItems field for non-vegan ingredients")
				}
				if _, exists := response["reasons"]; !exists {
					t.Error("Expected reasons field for non-vegan ingredients")
				}
			}

			// Verify timestamp exists
			if _, exists := response["timestamp"]; !exists {
				t.Error("Expected timestamp field in response")
			}
		})
	}
}

// TestCheckIngredientsErrorCases tests error handling
func TestCheckIngredientsErrorCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/check", bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/check", bytes.NewBufferString(""))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

// TestFindSubstitute tests the substitute finding endpoint
func TestFindSubstitute(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	tests := []struct {
		name               string
		ingredient         string
		context            string
		expectAlternatives bool
	}{
		{
			name:               "MilkSubstitute",
			ingredient:         "milk",
			context:            "baking",
			expectAlternatives: true,
		},
		{
			name:               "EggsSubstitute",
			ingredient:         "eggs",
			context:            "baking",
			expectAlternatives: true,
		},
		{
			name:               "ButterSubstitute",
			ingredient:         "butter",
			context:            "cooking",
			expectAlternatives: true,
		},
		{
			name:               "CheeseSubstitute",
			ingredient:         "cheese",
			context:            "spreading",
			expectAlternatives: true,
		},
		{
			name:               "HoneySubstitute",
			ingredient:         "honey",
			context:            "sweetening",
			expectAlternatives: true,
		},
		{
			name:               "UnknownIngredient",
			ingredient:         "unknown-ingredient",
			context:            "cooking",
			expectAlternatives: false,
		},
		{
			name:               "EmptyContext",
			ingredient:         "milk",
			context:            "",
			expectAlternatives: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := SubstituteRequest{
				Ingredient: tt.ingredient,
				Context:    tt.context,
			}
			body, _ := json.Marshal(reqBody)

			req, err := http.NewRequest("POST", "/api/substitute", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Verify required fields
			if _, exists := response["ingredient"]; !exists {
				t.Error("Expected ingredient field in response")
			}
			if _, exists := response["alternatives"]; !exists {
				t.Error("Expected alternatives field in response")
			}

			alternatives, _ := response["alternatives"].([]interface{})
			if tt.expectAlternatives && len(alternatives) == 0 {
				t.Errorf("Expected alternatives for %s, got none", tt.ingredient)
			}
		})
	}
}

// TestFindSubstituteErrorCases tests error handling for substitute endpoint
func TestFindSubstituteErrorCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/substitute", bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

// TestVeganizeRecipe tests the recipe veganization endpoint
func TestVeganizeRecipe(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	tests := []struct {
		name                 string
		recipe               string
		expectSubstitutions  bool
		expectedSubstituions int
	}{
		{
			name:                "RecipeWithNonVeganIngredients",
			recipe:              "Mix 1 cup milk, 2 eggs, and 1 tbsp butter to make pancakes",
			expectSubstitutions: true,
		},
		{
			name:                "AlreadyVeganRecipe",
			recipe:              "Mix flour, water, and oil to make bread",
			expectSubstitutions: false,
		},
		{
			name:                "EmptyRecipe",
			recipe:              "",
			expectSubstitutions: false,
		},
		{
			name:                "ComplexRecipe",
			recipe:              "Preheat oven. Mix milk, eggs, cheese, and butter. Bake for 30 minutes.",
			expectSubstitutions: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := VeganizeRequest{Recipe: tt.recipe}
			body, _ := json.Marshal(reqBody)

			req, err := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Verify required fields
			if _, exists := response["veganRecipe"]; !exists {
				t.Error("Expected veganRecipe field in response")
			}
			if _, exists := response["substitutions"]; !exists {
				t.Error("Expected substitutions field in response")
			}
			if _, exists := response["cookingTips"]; !exists {
				t.Error("Expected cookingTips field in response")
			}

			substitutions, _ := response["substitutions"].(map[string]interface{})
			if tt.expectSubstitutions && len(substitutions) == 0 {
				t.Error("Expected substitutions but got none")
			}
			if !tt.expectSubstitutions && len(substitutions) > 0 {
				t.Errorf("Expected no substitutions but got %d", len(substitutions))
			}
		})
	}
}

// TestVeganizeRecipeErrorCases tests error handling
func TestVeganizeRecipeErrorCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("InvalidJSON", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/veganize", bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

// TestGetCommonProducts tests the common products endpoint
func TestGetCommonProducts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("GetCommonProductsSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/products", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string][]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify expected categories
		expectedCategories := []string{"dairy", "eggs", "meat", "other"}
		for _, category := range expectedCategories {
			if _, exists := response[category]; !exists {
				t.Errorf("Expected category '%s' not found in response", category)
			}
		}

		// Verify categories have items
		for category, items := range response {
			if len(items) == 0 {
				t.Errorf("Category '%s' has no items", category)
			}
		}
	})
}

// TestGetNutrition tests the nutrition endpoint
func TestGetNutrition(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("GetNutritionSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/nutrition", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify nutritionalInfo exists
		nutritionalInfo, exists := response["nutritionalInfo"]
		if !exists {
			t.Fatal("Expected nutritionalInfo field in response")
		}

		info, ok := nutritionalInfo.(map[string]interface{})
		if !ok {
			t.Fatal("nutritionalInfo is not a map")
		}

		// Verify key nutrient fields
		expectedFields := []string{"protein", "b12", "iron", "calcium", "omega3", "considerations", "goodSources"}
		for _, field := range expectedFields {
			if _, exists := info[field]; !exists {
				t.Errorf("Expected field '%s' not found in nutritionalInfo", field)
			}
		}
	})
}

// TestCORSHeaders tests that CORS is properly configured
func TestCORSHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(router)

	t.Run("CORSPreflightRequest", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", "/api/check", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// CORS should allow the request
		if rr.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("Expected Access-Control-Allow-Origin header")
		}
	})
}

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "EnvVarSet",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "EnvVarNotSet",
			key:          "TEST_VAR_NOTSET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestFindSubstituteContextHandling tests context-specific recommendations
func TestFindSubstituteContextHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	tests := []struct {
		name        string
		ingredient  string
		context     string
		expectTip   bool
		tipContains string
	}{
		{
			name:        "BakingContext",
			ingredient:  "milk",
			context:     "baking",
			expectTip:   true,
			tipContains: "baking",
		},
		{
			name:        "CookingContext",
			ingredient:  "eggs",
			context:     "cooking",
			expectTip:   true,
			tipContains: "cooking",
		},
		{
			name:        "SpreadingContext",
			ingredient:  "butter",
			context:     "spreading",
			expectTip:   true,
			tipContains: "spreading",
		},
		{
			name:       "DefaultContext",
			ingredient: "cheese",
			context:    "general cooking",
			expectTip:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := SubstituteRequest{
				Ingredient: tt.ingredient,
				Context:    tt.context,
			}
			body, _ := json.Marshal(reqBody)

			req, err := http.NewRequest("POST", "/api/substitute", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectTip {
				if _, exists := response["quickTip"]; !exists {
					t.Error("Expected quickTip in response")
				} else if tt.tipContains != "" {
					tip := response["quickTip"].(string)
					if !strings.Contains(strings.ToLower(tip), tt.tipContains) {
						t.Errorf("Expected tip to contain '%s', got '%s'", tt.tipContains, tip)
					}
				}
			}
		})
	}
}

// TestVeganizeRecipeTitleCase tests recipe veganization handles title case
func TestVeganizeRecipeTitleCase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("TitleCaseReplacement", func(t *testing.T) {
		reqBody := VeganizeRequest{
			Recipe: "Add Milk and Butter to the mixture",
		}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		veganRecipe, exists := response["veganRecipe"]
		if !exists {
			t.Fatal("Expected veganRecipe in response")
		}

		// Verify substitutions were made
		substitutions := response["substitutions"].(map[string]interface{})
		if len(substitutions) == 0 {
			t.Error("Expected substitutions to be made for non-vegan recipe")
		}

		_ = veganRecipe // Use the variable
	})
}

// TestVeganizeRecipeMultipleSubstitutions tests handling many substitutions
func TestVeganizeRecipeMultipleSubstitutions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("ComplexRecipeWithManyIngredients", func(t *testing.T) {
		reqBody := VeganizeRequest{
			Recipe: "Beat eggs with milk, add melted butter and honey, then mix in cheese",
		}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", "/api/veganize", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have made multiple substitutions
		substitutions, exists := response["substitutions"]
		if !exists {
			t.Fatal("Expected substitutions in response")
		}

		subsMap := substitutions.(map[string]interface{})
		if len(subsMap) < 3 {
			t.Errorf("Expected at least 3 substitutions, got %d", len(subsMap))
		}

		// Verify message reflects multiple substitutions
		message, exists := response["message"]
		if !exists {
			t.Error("Expected message in response")
		} else {
			msgStr := message.(string)
			if !strings.Contains(msgStr, "substitution") {
				t.Error("Expected message to mention substitutions")
			}
		}
	})
}

// TestCheckIngredientsWithSuggestions tests that suggestions are provided
func TestCheckIngredientsWithSuggestions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NonVeganIngredientsIncludeSuggestions", func(t *testing.T) {
		reqBody := CheckRequest{Ingredients: "milk, butter"}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have suggestions
		suggestions, exists := response["suggestions"]
		if !exists {
			t.Error("Expected suggestions for non-vegan ingredients")
		} else {
			sugMap := suggestions.(map[string]interface{})
			if len(sugMap) == 0 {
				t.Error("Expected non-empty suggestions map")
			}
		}

		// Should have explanations
		if _, exists := response["explanations"]; !exists {
			t.Error("Expected explanations field in response")
		}
	})
}

// TestCheckIngredientsCacheIntegration tests caching behavior
func TestCheckIngredientsCacheIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CachedFieldInResponse", func(t *testing.T) {
		reqBody := CheckRequest{Ingredients: "flour, sugar"}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", "/api/check", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have cached field (will be false since cache is disabled in tests)
		if _, exists := response["cached"]; !exists {
			t.Error("Expected cached field in response")
		}
	})
}
