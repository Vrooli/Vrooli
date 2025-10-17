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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes logger for testing
func setupTestLogger() func() {
	// Discard logs during tests unless verbose
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(io.Discard)
	}
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestDB manages test database connections
type TestDB struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *TestDB {
	// Use test database configuration
	dbURL := os.Getenv("TEST_POSTGRES_URL")
	if dbURL == "" {
		// Build from components
		dbHost := getEnv("TEST_POSTGRES_HOST", "localhost")
		dbPort := getEnv("TEST_POSTGRES_PORT", "5432")
		dbUser := getEnv("TEST_POSTGRES_USER", "postgres")
		dbPassword := getEnv("TEST_POSTGRES_PASSWORD", "postgres")
		dbName := getEnv("TEST_POSTGRES_DB", "stream_analyzer_test")

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Database not available for testing: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Cannot connect to test database: %v", err)
		return nil
	}

	// Set global db for handlers
	originalDB := db
	db = testDB

	return &TestDB{
		DB: testDB,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM insights")
			testDB.Exec("DELETE FROM organized_notes")
			testDB.Exec("DELETE FROM stream_entries")
			testDB.Exec("DELETE FROM campaigns")

			// Restore original db
			db = originalDB
			testDB.Close()
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		jsonData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, val := range req.QueryParams {
			q.Add(key, val)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set headers
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add URL vars (for mux router)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertErrorResponse validates error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	body := w.Body.String()
	if expectedError != "" && body != expectedError+"\n" {
		// HTTP error messages end with newline
		if !contains(body, expectedError) {
			t.Errorf("Expected error containing '%s', got '%s'", expectedError, body)
		}
	}
}

// createTestCampaign creates a campaign for testing
func createTestCampaign(t *testing.T, testDB *TestDB, name string) *Campaign {
	t.Helper()

	campaign := &Campaign{
		Name:          name,
		Description:   "Test campaign: " + name,
		ContextPrompt: "Test context for " + name,
		Color:         "#3B82F6",
		Icon:          "📝",
	}

	err := testDB.DB.QueryRow(`
		INSERT INTO campaigns (name, description, context_prompt, color, icon)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, campaign.Name, campaign.Description, campaign.ContextPrompt,
		campaign.Color, campaign.Icon).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	campaign.Active = true
	return campaign
}

// createTestStreamEntry creates a stream entry for testing
func createTestStreamEntry(t *testing.T, testDB *TestDB, campaignID string, content string) *StreamEntry {
	t.Helper()

	entry := &StreamEntry{
		CampaignID: campaignID,
		Content:    content,
		Type:       "text",
		Source:     "test",
		Metadata:   json.RawMessage(`{}`),
	}

	err := testDB.DB.QueryRow(`
		INSERT INTO stream_entries (campaign_id, content, type, source, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, entry.CampaignID, entry.Content, entry.Type, entry.Source, entry.Metadata).Scan(&entry.ID, &entry.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test stream entry: %v", err)
	}

	return entry
}

// createTestNote creates an organized note for testing
func createTestNote(t *testing.T, testDB *TestDB, campaignID string, title string) *OrganizedNote {
	t.Helper()

	note := &OrganizedNote{
		CampaignID: campaignID,
		Title:      title,
		Content:    "Test content for " + title,
		Summary:    "Test summary",
		Category:   "test-category",
		Tags:       []string{"test", "automation"},
		Priority:   5,
		Metadata:   json.RawMessage(`{}`),
	}

	var tags []byte
	tags, _ = json.Marshal(note.Tags)

	err := testDB.DB.QueryRow(`
		INSERT INTO organized_notes (campaign_id, title, content, summary, category, tags, priority, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, note.CampaignID, note.Title, note.Content, note.Summary,
		note.Category, tags, note.Priority, note.Metadata).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create test note: %v", err)
	}

	return note
}

// createTestInsight creates an insight for testing
func createTestInsight(t *testing.T, testDB *TestDB, campaignID string, noteIDs []string) *Insight {
	t.Helper()

	insight := &Insight{
		CampaignID: campaignID,
		NoteIDs:    noteIDs,
		Type:       "pattern",
		Content:    "Test insight content",
		Confidence: 0.85,
		Metadata:   json.RawMessage(`{}`),
	}

	var noteIDsJSON []byte
	noteIDsJSON, _ = json.Marshal(insight.NoteIDs)

	err := testDB.DB.QueryRow(`
		INSERT INTO insights (campaign_id, note_ids, insight_type, content, confidence, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, insight.CampaignID, noteIDsJSON, insight.Type, insight.Content,
		insight.Confidence, insight.Metadata).Scan(&insight.ID, &insight.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test insight: %v", err)
	}

	return insight
}

// generateUUID generates a random UUID string
func generateUUID() string {
	return uuid.New().String()
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// assertJSONArrayResponse validates that response contains an array
func assertJSONArrayResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertResponseField validates a specific field in JSON response
func assertResponseField(t *testing.T, response map[string]interface{}, field string, expected interface{}) {
	t.Helper()

	actual, exists := response[field]
	if !exists {
		t.Errorf("Expected field '%s' not found in response", field)
		return
	}

	if expected != nil && actual != expected {
		t.Errorf("Expected field '%s' to be %v, got %v", field, expected, actual)
	}
}

// assertResponseContains validates response contains a substring
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, substr string) {
	t.Helper()

	body := w.Body.String()
	if !contains(body, substr) {
		t.Errorf("Expected response to contain '%s', got: %s", substr, body)
	}
}

// createTestCampaignWithDefaults creates a campaign with default values for testing
func createTestCampaignWithDefaults(t *testing.T, testDB *TestDB) *Campaign {
	t.Helper()
	return createTestCampaign(t, testDB, "Default Test Campaign")
}

// createMultipleTestCampaigns creates multiple campaigns for bulk testing
func createMultipleTestCampaigns(t *testing.T, testDB *TestDB, count int, namePrefix string) []*Campaign {
	t.Helper()

	campaigns := make([]*Campaign, count)
	for i := 0; i < count; i++ {
		campaigns[i] = createTestCampaign(t, testDB, fmt.Sprintf("%s %d", namePrefix, i+1))
	}
	return campaigns
}

// createMultipleTestNotes creates multiple notes for bulk testing
func createMultipleTestNotes(t *testing.T, testDB *TestDB, campaignID string, count int, titlePrefix string) []*OrganizedNote {
	t.Helper()

	notes := make([]*OrganizedNote, count)
	for i := 0; i < count; i++ {
		notes[i] = createTestNote(t, testDB, campaignID, fmt.Sprintf("%s %d", titlePrefix, i+1))
	}
	return notes
}

// createTestNoteWithCustomTags creates a note with custom tags for testing
func createTestNoteWithCustomTags(t *testing.T, testDB *TestDB, campaignID string, title string, tags []string) *OrganizedNote {
	t.Helper()

	note := &OrganizedNote{
		CampaignID: campaignID,
		Title:      title,
		Content:    "Test content for " + title,
		Summary:    "Test summary",
		Category:   "test-category",
		Tags:       tags,
		Priority:   5,
		Metadata:   json.RawMessage(`{}`),
	}

	var tagsJSON []byte
	tagsJSON, _ = json.Marshal(note.Tags)

	err := testDB.DB.QueryRow(`
		INSERT INTO organized_notes (campaign_id, title, content, summary, category, tags, priority, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, note.CampaignID, note.Title, note.Content, note.Summary,
		note.Category, tagsJSON, note.Priority, note.Metadata).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create test note with custom tags: %v", err)
	}

	return note
}
