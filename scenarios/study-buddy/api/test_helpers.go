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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes logging for tests
func setupTestLogger() func() {
	// Disable gin debug logging during tests
	gin.SetMode(gin.TestMode)

	// Set up test logger
	oldLogger := log.Writer()
	log.SetOutput(io.Discard)

	return func() {
		log.SetOutput(oldLogger)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Router     *gin.Engine
	Cleanup    func()
	TestUserID string
}

// setupTestEnvironment creates an isolated test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Set lifecycle env for main.go check
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Set up mock database connection (in-memory or test database)
	// For now, we'll mock the DB
	var testDB *sql.DB

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())

	// Generate test user ID
	testUserID := uuid.New().String()

	env := &TestEnvironment{
		DB:         testDB,
		Router:     router,
		TestUserID: testUserID,
		Cleanup: func() {
			if testDB != nil {
				testDB.Close()
			}
		},
	}

	return env
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Check if test database is available
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
		dbName := "study_buddy_test"
		// Skip DB tests if no test database configured
		t.Skipf("Skipping database tests - no TEST_POSTGRES_DB configured (expected: %s)", dbName)
		return nil, func() {}
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping database tests - cannot connect: %v", err)
		return nil, func() {}
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping database tests - database not available: %v", err)
		return nil, func() {}
	}

	cleanup := func() {
		// Clean up test data
		testDB.Exec("TRUNCATE subjects, flashcards, study_sessions, spaced_repetition CASCADE")
		testDB.Close()
	}

	return testDB, cleanup
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
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
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default content type for POST/PUT
	if req.Method == "POST" || req.Method == "PUT" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields ...string) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' not found in response: %v", field, response)
		}
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v. Body: %s", err, w.Body.String())
	}

	if _, ok := response["error"]; !ok {
		t.Errorf("Error response missing 'error' field: %v", response)
	}
}

// TestSubject represents a test subject
type TestSubject struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Color       string
	Icon        string
}

// createTestSubject creates a subject for testing
func createTestSubject(t *testing.T, router *gin.Engine, userID string) *TestSubject {
	subject := &TestSubject{
		UserID:      userID,
		Name:        "Test Subject",
		Description: "A test subject for unit testing",
		Color:       "#8B5A96",
		Icon:        "📚",
	}

	w, err := makeHTTPRequest(router, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/subjects",
		Body: map[string]interface{}{
			"user_id":     subject.UserID,
			"name":        subject.Name,
			"description": subject.Description,
			"color":       subject.Color,
			"icon":        subject.Icon,
		},
	})

	if err != nil {
		t.Fatalf("Failed to create test subject: %v", err)
	}

	if w.Code != 201 {
		t.Fatalf("Failed to create test subject. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse subject creation response: %v", err)
	}

	subject.ID = response["id"].(string)
	return subject
}

// TestFlashcard represents a test flashcard
type TestFlashcard struct {
	ID         string
	UserID     string
	SubjectID  string
	Front      string
	Back       string
	Difficulty int
	Tags       []string
}

// createTestFlashcard creates a flashcard for testing
func createTestFlashcard(t *testing.T, router *gin.Engine, userID, subjectID string) *TestFlashcard {
	flashcard := &TestFlashcard{
		UserID:     userID,
		SubjectID:  subjectID,
		Front:      "What is the powerhouse of the cell?",
		Back:       "Mitochondria",
		Difficulty: 2,
		Tags:       []string{"biology", "cell"},
	}

	w, err := makeHTTPRequest(router, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/flashcards",
		Body: map[string]interface{}{
			"user_id":    flashcard.UserID,
			"subject_id": flashcard.SubjectID,
			"front":      flashcard.Front,
			"back":       flashcard.Back,
			"difficulty": flashcard.Difficulty,
			"tags":       flashcard.Tags,
		},
	})

	if err != nil {
		t.Fatalf("Failed to create test flashcard: %v", err)
	}

	if w.Code != 201 {
		t.Fatalf("Failed to create test flashcard. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse flashcard creation response: %v", err)
	}

	flashcard.ID = response["id"].(string)
	return flashcard
}

// TestSession represents a test study session
type TestSession struct {
	ID          string
	UserID      string
	SubjectID   string
	SessionType string
	StartedAt   time.Time
}

// createTestSession creates a study session for testing
func createTestSession(t *testing.T, router *gin.Engine, userID, subjectID string) *TestSession {
	session := &TestSession{
		UserID:      userID,
		SubjectID:   subjectID,
		SessionType: "review",
		StartedAt:   time.Now(),
	}

	w, err := makeHTTPRequest(router, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/study/session/start",
		Body: map[string]interface{}{
			"user_id":         session.UserID,
			"subject_id":      session.SubjectID,
			"session_type":    session.SessionType,
			"target_duration": 25,
		},
	})

	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	if w.Code != 201 {
		t.Fatalf("Failed to create test session. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse session creation response: %v", err)
	}

	session.ID = response["session_id"].(string)
	return session
}

// assertSpacedRepetitionData validates spaced repetition calculations
func assertSpacedRepetitionData(t *testing.T, data SpacedRepetitionData, expectedMinInterval, expectedMaxInterval int) {
	if data.Interval < expectedMinInterval || data.Interval > expectedMaxInterval {
		t.Errorf("Expected interval between %d and %d days, got %d", expectedMinInterval, expectedMaxInterval, data.Interval)
	}

	if data.EaseFactor < 1.3 {
		t.Errorf("Ease factor too low: %f (minimum should be 1.3)", data.EaseFactor)
	}

	if data.NextReview.Before(data.LastReviewed) {
		t.Errorf("Next review date (%v) is before last reviewed date (%v)", data.NextReview, data.LastReviewed)
	}
}

// assertXPCalculation validates XP calculation logic
func assertXPCalculation(t *testing.T, response, responseTime int, expectedMin, expectedMax int) {
	xp := calculateXP(mapResponseToString(response), responseTime)

	if xp < expectedMin || xp > expectedMax {
		t.Errorf("Expected XP between %d and %d, got %d", expectedMin, expectedMax, xp)
	}
}

// mapResponseToString converts quality int to response string
func mapResponseToString(quality int) string {
	switch quality {
	case 0:
		return "again"
	case 1:
		return "hard"
	case 3:
		return "good"
	case 5:
		return "easy"
	default:
		return "again"
	}
}

// compareFlashcards compares two flashcards for equality
func compareFlashcards(t *testing.T, expected, actual *TestFlashcard) {
	if expected.Front != actual.Front {
		t.Errorf("Front mismatch: expected '%s', got '%s'", expected.Front, actual.Front)
	}
	if expected.Back != actual.Back {
		t.Errorf("Back mismatch: expected '%s', got '%s'", expected.Back, actual.Back)
	}
	if expected.Difficulty != actual.Difficulty {
		t.Errorf("Difficulty mismatch: expected %d, got %d", expected.Difficulty, actual.Difficulty)
	}
}
