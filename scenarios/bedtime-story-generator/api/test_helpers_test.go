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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Redirect logs to avoid cluttering test output
	log.SetOutput(ioutil.Discard)

	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "bedtime-story-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// TestDatabase provides mock database for testing
type TestDatabase struct {
	DB      *sql.DB
	Mock    sqlmock.Sqlmock
	Cleanup func()
}

// setupTestDatabase creates a mock database for testing
func setupTestDatabase(t *testing.T) *TestDatabase {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	oldDB := db
	db = mockDB

	return &TestDatabase{
		DB:   mockDB,
		Mock: mock,
		Cleanup: func() {
			db = oldDB
			mockDB.Close()
		},
	}
}

// TestStory provides a pre-configured story for testing
type TestStory struct {
	Story   *Story
	Cleanup func()
}

// setupTestStory creates a test story with sample data
func setupTestStory(t *testing.T, title string, ageGroup string) *TestStory {
	now := time.Now()
	lastRead := now.Add(-1 * time.Hour)

	story := &Story{
		ID:             uuid.New().String(),
		Title:          title,
		Content:        "Once upon a time in a magical forest...",
		AgeGroup:       ageGroup,
		Theme:          "adventure",
		StoryLength:    "medium",
		ReadingTime:    10,
		CharacterNames: []string{"Alice", "Bob"},
		PageCount:      5,
		CreatedAt:      now,
		TimesRead:      3,
		LastRead:       &lastRead,
		IsFavorite:     false,
		Illustrations:  map[string]string{"1": "illustration_1.png"},
	}

	return &TestStory{
		Story: story,
		Cleanup: func() {
			// Clean up any cached story
			storyCache.Delete(story.ID)
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
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var array []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &array); err != nil {
		t.Fatalf("Failed to parse JSON array: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// Error responses may be plain text or JSON
	responseBody := w.Body.String()

	if expectedErrorMessage != "" {
		if w.Header().Get("Content-Type") == "application/json" {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				if errorMsg, exists := response["error"]; exists {
					if errorStr, ok := errorMsg.(string); ok {
						responseBody = errorStr
					}
				}
			}
		}

		if responseBody != expectedErrorMessage && !contains(responseBody, expectedErrorMessage) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, responseBody)
		}
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// GenerateStoryRequest creates a test story generation request
func (g *TestDataGenerator) GenerateStoryRequest(ageGroup, theme, length string, characters []string) GenerateStoryRequest {
	return GenerateStoryRequest{
		AgeGroup:       ageGroup,
		Theme:          theme,
		Length:         length,
		CharacterNames: characters,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// mockStoryRows creates mock rows for story queries
func mockStoryRows(mock sqlmock.Sqlmock, stories ...*Story) {
	rows := sqlmock.NewRows([]string{
		"id", "title", "content", "age_group", "theme",
		"story_length", "reading_time_minutes", "page_count",
		"character_names", "created_at", "times_read", "last_read",
		"is_favorite", "illustrations",
	})

	for _, story := range stories {
		charactersJSON, _ := json.Marshal(story.CharacterNames)
		illustrationsJSON, _ := json.Marshal(story.Illustrations)

		rows.AddRow(
			story.ID,
			story.Title,
			story.Content,
			story.AgeGroup,
			story.Theme,
			story.StoryLength,
			story.ReadingTime,
			story.PageCount,
			string(charactersJSON),
			story.CreatedAt,
			story.TimesRead,
			story.LastRead,
			story.IsFavorite,
			string(illustrationsJSON),
		)
	}

	mock.ExpectQuery("SELECT (.+) FROM stories").WillReturnRows(rows)
}

// mockThemeRows creates mock rows for theme queries
func mockThemeRows(mock sqlmock.Sqlmock, themes ...map[string]string) {
	rows := sqlmock.NewRows([]string{"name", "description", "emoji", "color"})

	for _, theme := range themes {
		rows.AddRow(
			theme["name"],
			theme["description"],
			theme["emoji"],
			theme["color"],
		)
	}

	mock.ExpectQuery("SELECT name, description, emoji, color FROM story_themes").
		WillReturnRows(rows)
}
