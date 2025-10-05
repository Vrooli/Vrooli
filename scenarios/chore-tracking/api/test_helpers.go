package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestEnvironment manages isolated test environment with database
type TestEnvironment struct {
	DB         *sql.DB
	Cleanup    func()
	OriginalDB *sql.DB
}

// setupTestDB creates an isolated test database connection
func setupTestDB(t *testing.T) *TestEnvironment {
	// Store original DB
	originalDB := db

	// Create test database connection
	testDBURL := os.Getenv("POSTGRES_URL")
	if testDBURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			t.Skip("Database configuration not available for testing")
		}

		testDBURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to open test database connection: %v", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctx); err != nil {
		testDB.Close()
		t.Skip("Test database not available")
	}

	// Set as global db for handlers
	db = testDB

	return &TestEnvironment{
		DB:         testDB,
		OriginalDB: originalDB,
		Cleanup: func() {
			// Clean up test data
			cleanupTestData(testDB)
			// Restore original DB
			db = originalDB
			testDB.Close()
		},
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(testDB *sql.DB) {
	// Delete in correct order to respect foreign keys
	testDB.Exec("DELETE FROM user_achievements WHERE user_id >= 9000")
	testDB.Exec("DELETE FROM user_rewards WHERE user_id >= 9000")
	testDB.Exec("DELETE FROM reward_redemptions WHERE user_id >= 9000")
	testDB.Exec("DELETE FROM chore_assignments WHERE user_id >= 9000 OR chore_id >= 9000")
	testDB.Exec("DELETE FROM chores WHERE id >= 9000")
	testDB.Exec("DELETE FROM users WHERE id >= 9000")
	testDB.Exec("DELETE FROM achievements WHERE id >= 9000")
	testDB.Exec("DELETE FROM rewards WHERE id >= 9000")
}

// TestChore provides a pre-configured chore for testing
type TestChore struct {
	Chore   *Chore
	Cleanup func()
}

// setupTestChore creates a test chore in the database
func setupTestChore(t *testing.T, env *TestEnvironment, title string, points int) *TestChore {
	query := `INSERT INTO chores (id, title, description, points, difficulty, category,
	          frequency, status, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at`

	choreID := 9000 + int(time.Now().Unix()%1000)
	description := fmt.Sprintf("Test chore: %s", title)
	chore := &Chore{
		ID:          choreID,
		Title:       title,
		Description: description,
		Points:      points,
		Difficulty:  "medium",
		Category:    "cleaning",
		Frequency:   "daily",
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	err := env.DB.QueryRow(query, chore.ID, chore.Title, chore.Description, chore.Points,
		chore.Difficulty, chore.Category, chore.Frequency, chore.Status, chore.CreatedAt).
		Scan(&chore.ID, &chore.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test chore: %v", err)
	}

	return &TestChore{
		Chore: chore,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM chores WHERE id = $1", chore.ID)
		},
	}
}

// TestUser provides a pre-configured user for testing
type TestUser struct {
	User    *User
	Cleanup func()
}

// setupTestUser creates a test user in the database
func setupTestUser(t *testing.T, env *TestEnvironment, name string) *TestUser {
	userID := 9000 + int(time.Now().Unix()%1000)
	user := &User{
		ID:            userID,
		Name:          name,
		Avatar:        "test-avatar.png",
		Level:         1,
		TotalPoints:   0,
		CurrentStreak: 0,
		LongestStreak: 0,
		CreatedAt:     time.Now(),
	}

	query := `INSERT INTO users (id, name, avatar, level, total_points, current_streak,
	          longest_streak, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := env.DB.Exec(query, user.ID, user.Name, user.Avatar, user.Level,
		user.TotalPoints, user.CurrentStreak, user.LongestStreak, user.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return &TestUser{
		User: user,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM users WHERE id = $1", user.ID)
		},
	}
}

// TestReward provides a pre-configured reward for testing
type TestReward struct {
	Reward  *Reward
	Cleanup func()
}

// setupTestReward creates a test reward in the database
func setupTestReward(t *testing.T, env *TestEnvironment, name string, cost int) *TestReward {
	rewardID := 9000 + int(time.Now().Unix()%1000)
	reward := &Reward{
		ID:          rewardID,
		Name:        name,
		Description: fmt.Sprintf("Test reward: %s", name),
		Cost:        cost,
		Icon:        "test-icon.png",
		Available:   true,
	}

	query := `INSERT INTO rewards (id, name, description, cost, icon, available)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := env.DB.Exec(query, reward.ID, reward.Name, reward.Description,
		reward.Cost, reward.Icon, reward.Available)

	if err != nil {
		t.Fatalf("Failed to create test reward: %v", err)
	}

	return &TestReward{
		Reward: reward,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM rewards WHERE id = $1", reward.ID)
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

	// Validate expected fields if provided
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
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var array []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &array); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// testHandlerWithRequest is a helper for testing handlers with specific requests
func testHandlerWithRequest(t *testing.T, handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	handler(w, httpReq)
	return w
}

// silenceTestOutput redirects stdout/stderr to null during tests
func silenceTestOutput() func() {
	// Save original stdout/stderr
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Redirect to null
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	os.Stderr = devNull

	return func() {
		devNull.Close()
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}
}

// waitForCondition polls a condition function until it returns true or timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatalf("Timeout waiting for condition: %s", message)
			}
		}
	}
}
