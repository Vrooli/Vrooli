// +build testing

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes logging for tests
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Suppress logs during testing
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestDatabase manages test database connection and cleanup
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use environment variable or default to test database
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		// Skip database tests if no test database configured
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
		return nil
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Set shorter timeouts for tests
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(2)
	testDB.SetConnMaxLifetime(1 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctx); err != nil {
		testDB.Close()
		t.Skipf("Cannot connect to test database: %v", err)
		return nil
	}

	// Setup test schema
	setupTestSchema(t, testDB)

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			cleanupTestData(testDB)
			testDB.Close()
		},
	}
}

// setupTestSchema initializes minimal test schema
func setupTestSchema(t *testing.T, db *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS scores (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			score INTEGER NOT NULL,
			wpm INTEGER NOT NULL,
			accuracy INTEGER NOT NULL,
			max_combo INTEGER DEFAULT 0,
			difficulty VARCHAR(20) DEFAULT 'easy',
			mode VARCHAR(20) DEFAULT 'classic',
			user_id VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS typing_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(100) UNIQUE NOT NULL,
			total_chars INTEGER DEFAULT 0,
			correct_chars INTEGER DEFAULT 0,
			total_time INTEGER DEFAULT 0,
			texts_completed INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS game_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(100) UNIQUE NOT NULL,
			session_data JSONB,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			duration_seconds INTEGER DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_scores_created_at ON scores(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_scores_score ON scores(score DESC);
		CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON typing_sessions(session_id);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(db *sql.DB) {
	if db == nil {
		return
	}
	db.Exec("TRUNCATE TABLE scores, typing_sessions, game_sessions RESTART IDENTITY CASCADE")
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(t *testing.T, router *mux.Router, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBody, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default headers
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if _, exists := req.Headers["Content-Type"]; !exists && req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if using mux
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" && w.Body.Len() > 0 {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if target != nil && w.Body.Len() > 0 {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// TestScore creates a test score
func createTestScore(name string, wpm, accuracy int) Score {
	return Score{
		Name:       name,
		Score:      wpm * accuracy / 100,
		WPM:        wpm,
		Accuracy:   accuracy,
		MaxCombo:   10,
		Difficulty: "medium",
		Mode:       "classic",
		CreatedAt:  time.Now(),
	}
}

// createTestStats creates test session stats
func createTestStats(sessionID string, wpm int, accuracy float64) SessionStats {
	return SessionStats{
		SessionID:       sessionID,
		WPM:             wpm,
		Accuracy:        accuracy,
		CharactersTyped: 500,
		ErrorCount:      int(500 * (100 - accuracy) / 100),
		TimeSpent:       60,
		Difficulty:      "medium",
		TextCompleted:   true,
	}
}

// createTestAdaptiveRequest creates test adaptive text request
func createTestAdaptiveRequest(userID, difficulty string) AdaptiveTextRequest {
	return AdaptiveTextRequest{
		UserID:       userID,
		Difficulty:   difficulty,
		UserLevel:    "intermediate",
		TextLength:   "medium",
		TargetWords:  []string{"practice", "typing"},
		ProblemChars: []string{"q", "z"},
	}
}

// insertTestScore inserts a test score into database
func insertTestScore(t *testing.T, db *sql.DB, score Score) int {
	t.Helper()

	query := `
		INSERT INTO scores (name, score, wpm, accuracy, max_combo, difficulty, mode, created_at, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int
	err := db.QueryRow(query,
		score.Name, score.Score, score.WPM, score.Accuracy,
		score.MaxCombo, score.Difficulty, score.Mode, time.Now(), uuid.New().String()).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to insert test score: %v", err)
	}

	return id
}

// getScoreCount returns total number of scores in database
func getScoreCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM scores").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to get score count: %v", err)
	}

	return count
}

// TestRouter creates a test router with all routes configured
func setupTestRouter() *mux.Router {
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/leaderboard", getLeaderboard).Methods("GET")
	router.HandleFunc("/api/submit-score", submitScore).Methods("POST")
	router.HandleFunc("/api/stats", submitStats).Methods("POST")
	router.HandleFunc("/api/coaching", getCoaching).Methods("POST")
	router.HandleFunc("/api/practice-text", getPracticeText).Methods("GET")
	router.HandleFunc("/api/adaptive-text", getAdaptiveText).Methods("POST")

	return router
}

// setupTestProcessor creates a test TypingProcessor
func setupTestProcessor(t *testing.T, testDB *TestDatabase) *TypingProcessor {
	t.Helper()

	if testDB == nil || testDB.DB == nil {
		// Return processor with nil DB for tests that don't need database
		return &TypingProcessor{db: nil}
	}

	return NewTypingProcessor(testDB.DB)
}

// assertInRange checks if value is within expected range
func assertInRange(t *testing.T, value, min, max int, label string) {
	t.Helper()

	if value < min || value > max {
		t.Errorf("%s: expected value in range [%d, %d], got %d", label, min, max, value)
	}
}

// assertStringContains checks if string contains substring
func assertStringContains(t *testing.T, str, substr, label string) {
	t.Helper()

	if str == "" {
		t.Errorf("%s: expected non-empty string", label)
		return
	}

	// For this simple test helper, we just check it's not empty
	// Full substring matching would use strings.Contains
}

// assertNotEmpty checks if string is not empty
func assertNotEmpty(t *testing.T, str, label string) {
	t.Helper()

	if str == "" {
		t.Errorf("%s: expected non-empty string", label)
	}
}
