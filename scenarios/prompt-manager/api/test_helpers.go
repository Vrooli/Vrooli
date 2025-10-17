package main

import (
	"bytes"
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
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Discard logs during tests unless explicitly needed
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB           *sql.DB
	Server       *APIServer
	Router       *mux.Router
	TestCampaign *Campaign
	TestPrompt   *Prompt
	Cleanup      func()
}

// setupTestEnvironment creates a test API server with mock database
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Create in-memory SQLite database for testing
	// Note: In production, this would use a test PostgreSQL instance
	// For now, we'll create a minimal environment for unit tests

	env := &TestEnvironment{
		Cleanup: func() {
			// Cleanup resources
		},
	}

	return env
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Get test database URL from environment
	dbURL := os.Getenv("TEST_POSTGRES_URL")
	if dbURL == "" {
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Set search path to test schema
	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS prompt_mgr_test")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	_, err = db.Exec("SET search_path TO prompt_mgr_test, public")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to set search path: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DROP SCHEMA IF EXISTS prompt_mgr_test CASCADE")
		db.Close()
	}

	return db, cleanup
}

// setupTestTables creates test database tables
func setupTestTables(t *testing.T, db *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS campaigns (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			description TEXT,
			color TEXT DEFAULT '#6366f1',
			parent_id UUID REFERENCES campaigns(id),
			sort_order INTEGER DEFAULT 0,
			is_favorite BOOLEAN DEFAULT FALSE,
			prompt_count INTEGER DEFAULT 0,
			last_used TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS prompts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			campaign_id UUID NOT NULL REFERENCES campaigns(id),
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			description TEXT,
			variables JSONB DEFAULT '[]'::jsonb,
			usage_count INTEGER DEFAULT 0,
			last_used TIMESTAMP,
			is_favorite BOOLEAN DEFAULT FALSE,
			is_archived BOOLEAN DEFAULT FALSE,
			quick_access_key TEXT UNIQUE,
			version INTEGER DEFAULT 1,
			parent_version_id UUID,
			word_count INTEGER,
			estimated_tokens INTEGER,
			effectiveness_rating INTEGER CHECK (effectiveness_rating >= 1 AND effectiveness_rating <= 5),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS tags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE,
			color TEXT,
			description TEXT
		);

		CREATE TABLE IF NOT EXISTS prompt_tags (
			prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
			tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			PRIMARY KEY (prompt_id, tag_id)
		);

		CREATE TABLE IF NOT EXISTS test_results (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
			model TEXT NOT NULL,
			input_variables TEXT,
			response TEXT,
			response_time FLOAT,
			token_count INTEGER,
			rating INTEGER CHECK (rating >= 1 AND rating <= 5),
			notes TEXT,
			tested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}
}

// createTestCampaign creates a test campaign in the database
func createTestCampaign(t *testing.T, db *sql.DB, name string) *Campaign {
	campaign := &Campaign{
		ID:          uuid.New().String(),
		Name:        name,
		Description: ptrString("Test campaign: " + name),
		Color:       "#6366f1",
		Icon:        "folder",
		SortOrder:   0,
		IsFavorite:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO campaigns (id, name, description, color, sort_order, is_favorite, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING prompt_count, last_used`

	err := db.QueryRow(query,
		campaign.ID, campaign.Name, campaign.Description, campaign.Color,
		campaign.SortOrder, campaign.IsFavorite, campaign.CreatedAt, campaign.UpdatedAt,
	).Scan(&campaign.PromptCount, &campaign.LastUsed)

	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	return campaign
}

// createTestPrompt creates a test prompt in the database
func createTestPrompt(t *testing.T, db *sql.DB, campaignID string, title string, content string) *Prompt {
	prompt := &Prompt{
		ID:         uuid.New().String(),
		CampaignID: campaignID,
		Title:      title,
		Content:    content,
		Variables:  []string{},
		Version:    1,
		WordCount:  calculateWordCount(&content),
		EstimatedTokens: calculateTokenCount(&content),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	variablesJSON, _ := json.Marshal(prompt.Variables)

	query := `
		INSERT INTO prompts (id, campaign_id, title, content, variables, word_count,
		                    estimated_tokens, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING usage_count, last_used, is_favorite, is_archived, version, parent_version_id`

	err := db.QueryRow(query,
		prompt.ID, prompt.CampaignID, prompt.Title, prompt.Content, variablesJSON,
		prompt.WordCount, prompt.EstimatedTokens, prompt.CreatedAt, prompt.UpdatedAt,
	).Scan(&prompt.UsageCount, &prompt.LastUsed, &prompt.IsFavorite,
		&prompt.IsArchived, &prompt.Version, &prompt.ParentVersionID)

	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	return prompt
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(server *APIServer, router *mux.Router, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonData, _ := json.Marshal(v)
			bodyReader = bytes.NewBuffer(jsonData)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Set URL vars for mux
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	if expectedStatus >= 200 && expectedStatus < 300 {
		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
		}
		return result
	}

	return nil
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	body := w.Body.String()
	if expectedMessage != "" && body != "" {
		// For HTTP errors, the body is plain text
		if expectedMessage != body && !contains(body, expectedMessage) {
			t.Errorf("Expected error message containing '%s', got '%s'", expectedMessage, body)
		}
	}
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func ptrBool(b bool) *bool {
	return &b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
