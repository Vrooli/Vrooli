// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Suppress logs during tests unless explicitly needed
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	DB         *sql.DB
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "recipe-book-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Setup test database connection (using in-memory or test DB)
	testDB := setupTestDB(t)

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		DB:         testDB,
		Cleanup: func() {
			if testDB != nil {
				testDB.Close()
			}
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use environment variables or fallback to test defaults
	dbHost := os.Getenv("TEST_POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("TEST_POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("TEST_POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("TEST_POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("TEST_POSTGRES_DB")
	if dbName == "" {
		dbName = "recipe_book_test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Logf("Warning: Failed to connect to test database: %v", err)
		return nil
	}

	if err = testDB.Ping(); err != nil {
		t.Logf("Warning: Failed to ping test database: %v", err)
		testDB.Close()
		return nil
	}

	// Set the global db variable for testing
	db = testDB

	return testDB
}

// TestRecipe provides a pre-configured recipe for testing
type TestRecipe struct {
	Recipe  *Recipe
	Cleanup func()
}

// setupTestRecipe creates a test recipe with sample data
func setupTestRecipe(t *testing.T, title string) *TestRecipe {
	recipe := &Recipe{
		ID:          uuid.New().String(),
		Title:       title,
		Description: "Test recipe: " + title,
		Ingredients: []Ingredient{
			{Name: "flour", Amount: 2, Unit: "cups"},
			{Name: "eggs", Amount: 3, Unit: "whole"},
			{Name: "milk", Amount: 1, Unit: "cup"},
		},
		Instructions: []string{
			"Mix dry ingredients",
			"Add wet ingredients",
			"Bake at 350F for 30 minutes",
		},
		PrepTime:   15,
		CookTime:   30,
		Servings:   4,
		Tags:       []string{"breakfast", "baking"},
		Cuisine:    "American",
		DietaryInfo: []string{"vegetarian"},
		Nutrition: NutritionInfo{
			Calories: 250,
			Protein:  8,
			Carbs:    35,
			Fat:      10,
		},
		CreatedBy:  "test-user-" + uuid.New().String(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Visibility: "private",
		Source:     "original",
		SharedWith: []string{},
	}

	// Insert into database if available
	if db != nil {
		ingredientsJSON, _ := json.Marshal(recipe.Ingredients)
		instructionsJSON, _ := json.Marshal(recipe.Instructions)
		tagsJSON, _ := json.Marshal(recipe.Tags)
		dietaryJSON, _ := json.Marshal(recipe.DietaryInfo)
		nutritionJSON, _ := json.Marshal(recipe.Nutrition)
		sharedWithJSON, _ := json.Marshal(recipe.SharedWith)

		_, err := db.Exec(`
			INSERT INTO recipes (id, title, description, ingredients, instructions,
			                    prep_time, cook_time, servings, tags, cuisine,
			                    dietary_info, nutrition, photo_url, created_by,
			                    created_at, updated_at, visibility, shared_with,
			                    source, parent_recipe_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
			recipe.ID, recipe.Title, recipe.Description, ingredientsJSON, instructionsJSON,
			recipe.PrepTime, recipe.CookTime, recipe.Servings, tagsJSON, recipe.Cuisine,
			dietaryJSON, nutritionJSON, recipe.PhotoURL, recipe.CreatedBy,
			recipe.CreatedAt, recipe.UpdatedAt, recipe.Visibility, sharedWithJSON,
			recipe.Source, recipe.ParentID)

		if err != nil {
			t.Logf("Warning: Failed to insert test recipe: %v", err)
		}
	}

	return &TestRecipe{
		Recipe: recipe,
		Cleanup: func() {
			if db != nil {
				db.Exec("DELETE FROM recipes WHERE id = $1", recipe.ID)
			}
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected error response body, got empty response")
	}
}

// assertRecipeEqual compares two recipes for equality
func assertRecipeEqual(t *testing.T, expected, actual *Recipe) {
	if expected.Title != actual.Title {
		t.Errorf("Expected title %s, got %s", expected.Title, actual.Title)
	}
	if expected.Description != actual.Description {
		t.Errorf("Expected description %s, got %s", expected.Description, actual.Description)
	}
	if expected.PrepTime != actual.PrepTime {
		t.Errorf("Expected prep_time %d, got %d", expected.PrepTime, actual.PrepTime)
	}
	if expected.CookTime != actual.CookTime {
		t.Errorf("Expected cook_time %d, got %d", expected.CookTime, actual.CookTime)
	}
	if expected.Servings != actual.Servings {
		t.Errorf("Expected servings %d, got %d", expected.Servings, actual.Servings)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateRecipeRequest creates a test recipe creation request
func (g *TestDataGenerator) CreateRecipeRequest(title string) Recipe {
	return Recipe{
		Title:       title,
		Description: "Test recipe: " + title,
		Ingredients: []Ingredient{
			{Name: "ingredient1", Amount: 1, Unit: "cup"},
		},
		Instructions: []string{"Step 1", "Step 2"},
		PrepTime:     10,
		CookTime:     20,
		Servings:     4,
		Tags:         []string{"test"},
		Cuisine:      "Test",
		DietaryInfo:  []string{},
		Nutrition:    NutritionInfo{Calories: 100},
		CreatedBy:    "test-user",
		Visibility:   "private",
		Source:       "original",
	}
}

// SearchRequest creates a test search request
func (g *TestDataGenerator) SearchRequest(query string) SearchRequest {
	return SearchRequest{
		Query:  query,
		UserID: "test-user",
		Limit:  10,
	}
}

// GenerateRequest creates a test generate request
func (g *TestDataGenerator) GenerateRequest(prompt string) GenerateRequest {
	return GenerateRequest{
		Prompt: prompt,
		UserID: "test-user",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// cleanupTestRecipes removes all test recipes from database
func cleanupTestRecipes(t *testing.T) {
	if db != nil {
		_, err := db.Exec("DELETE FROM recipes WHERE created_by LIKE 'test-user%'")
		if err != nil {
			t.Logf("Warning: Failed to cleanup test recipes: %v", err)
		}
	}
}

// cleanupTestRatings removes all test ratings from database
func cleanupTestRatings(t *testing.T) {
	if db != nil {
		_, err := db.Exec("DELETE FROM recipe_ratings WHERE user_id LIKE 'test-user%'")
		if err != nil {
			t.Logf("Warning: Failed to cleanup test ratings: %v", err)
		}
	}
}
