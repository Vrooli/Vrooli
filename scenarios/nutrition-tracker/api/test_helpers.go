// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Suppress logs during tests
	return func() { log.SetOutput(originalOutput) }
}

// TestDatabase manages test database connection
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates an isolated test database
func setupTestDB(t *testing.T) *TestDatabase {
	// Use environment variable or in-memory database for testing
	testDBURL := os.Getenv("TEST_POSTGRES_URL")
	if testDBURL == "" {
		// Skip database tests if no test database available
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
		return nil
	}

	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Ping to ensure connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Create test schema
	setupTestSchema(t, testDB)

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			cleanupTestSchema(testDB)
			testDB.Close()
		},
	}
}

// setupTestSchema creates necessary tables for testing
func setupTestSchema(t *testing.T, db *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS meals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			meal_type VARCHAR(50) NOT NULL,
			meal_date DATE NOT NULL,
			meal_time TIME,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS foods (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			brand VARCHAR(255),
			serving_size DECIMAL(10, 2),
			serving_unit VARCHAR(50),
			calories DECIMAL(10, 2),
			protein DECIMAL(10, 2),
			carbs DECIMAL(10, 2),
			fat DECIMAL(10, 2),
			fiber DECIMAL(10, 2),
			sugar DECIMAL(10, 2),
			sodium DECIMAL(10, 2),
			food_category VARCHAR(100),
			source VARCHAR(50) DEFAULT 'user_created',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS meal_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			meal_id UUID REFERENCES meals(id) ON DELETE CASCADE,
			food_id UUID REFERENCES foods(id) ON DELETE SET NULL,
			custom_food_name VARCHAR(255),
			quantity DECIMAL(10, 2) NOT NULL,
			unit VARCHAR(50),
			calories DECIMAL(10, 2),
			protein DECIMAL(10, 2),
			carbs DECIMAL(10, 2),
			fat DECIMAL(10, 2),
			fiber DECIMAL(10, 2),
			sugar DECIMAL(10, 2),
			sodium DECIMAL(10, 2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS user_goals (
			user_id VARCHAR(255) PRIMARY KEY,
			daily_calories INTEGER,
			protein_grams INTEGER,
			carbs_grams INTEGER,
			fat_grams INTEGER,
			fiber_grams INTEGER,
			sugar_limit INTEGER,
			sodium_limit INTEGER,
			weight_goal VARCHAR(50),
			activity_level VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

// cleanupTestSchema drops test tables
func cleanupTestSchema(db *sql.DB) {
	db.Exec("DROP TABLE IF EXISTS meal_items CASCADE")
	db.Exec("DROP TABLE IF EXISTS meals CASCADE")
	db.Exec("DROP TABLE IF EXISTS foods CASCADE")
	db.Exec("DROP TABLE IF EXISTS user_goals CASCADE")
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Query   map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	if req.Query != nil {
		q := httpReq.URL.Query()
		for key, value := range req.Query {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httpReq)

	return recorder, nil
}

// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	if target != nil {
		if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
			t.Errorf("Failed to unmarshal response: %v. Body: %s", err, recorder.Body.String())
		}
	}
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, recorder.Code)
	}

	if expectedMessage != "" {
		body := recorder.Body.String()
		if body != expectedMessage && !contains(body, expectedMessage) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedMessage, body)
		}
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr)))))
}

// createTestMeal creates a test meal in the database
func createTestMeal(t *testing.T, db *sql.DB, userID, mealType, mealDate string) string {
	var mealID string
	err := db.QueryRow(`
		INSERT INTO meals (user_id, meal_type, meal_date, meal_time)
		VALUES ($1, $2, $3, '12:00:00')
		RETURNING id
	`, userID, mealType, mealDate).Scan(&mealID)

	if err != nil {
		t.Fatalf("Failed to create test meal: %v", err)
	}

	return mealID
}

// createTestFood creates a test food in the database
func createTestFood(t *testing.T, db *sql.DB, name string, calories float64) string {
	var foodID string
	err := db.QueryRow(`
		INSERT INTO foods (name, brand, serving_size, serving_unit, calories, protein, carbs, fat, food_category)
		VALUES ($1, 'Test Brand', 100, 'g', $2, 10, 20, 5, 'test')
		RETURNING id
	`, name, calories).Scan(&foodID)

	if err != nil {
		t.Fatalf("Failed to create test food: %v", err)
	}

	return foodID
}

// createTestMealItem creates a test meal item in the database
func createTestMealItem(t *testing.T, db *sql.DB, mealID, foodName string, calories float64) {
	_, err := db.Exec(`
		INSERT INTO meal_items (meal_id, custom_food_name, quantity, unit, calories, protein, carbs, fat, fiber, sugar, sodium)
		VALUES ($1, $2, 1, 'serving', $3, 10, 20, 5, 3, 8, 200)
	`, mealID, foodName, calories)

	if err != nil {
		t.Fatalf("Failed to create test meal item: %v", err)
	}
}

// createTestUserGoals creates test user goals in the database
func createTestUserGoals(t *testing.T, db *sql.DB, userID string) {
	_, err := db.Exec(`
		INSERT INTO user_goals (user_id, daily_calories, protein_grams, carbs_grams, fat_grams, fiber_grams, sugar_limit, sodium_limit, weight_goal, activity_level)
		VALUES ($1, 2000, 75, 225, 65, 25, 50, 2300, 'maintain', 'moderately_active')
		ON CONFLICT (user_id) DO NOTHING
	`, userID)

	if err != nil {
		t.Fatalf("Failed to create test user goals: %v", err)
	}
}
