// +build testing

package main

import (
	"bytes"
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
	"github.com/microcosm-cc/bluemonday"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalMode string
	cleanup      func()
}

// setupTestLogger initializes the logger for testing
func setupTestLogger() func() {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard) // Suppress logs during tests
	return func() {
		gin.SetMode(gin.ReleaseMode)
		log.SetOutput(os.Stdout)
	}
}

// TestDatabase manages test database setup and teardown
type TestDatabase struct {
	DB      *Database
	Config  *Config
	Cleanup func()
}

// setupTestDatabase creates an isolated test database
func setupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	// Use test database configuration
	config := &Config{
		DBHost: getEnv("POSTGRES_HOST", "localhost"),
		DBPort: getEnvInt("POSTGRES_PORT", 5432),
		DBUser: getEnv("POSTGRES_USER", "postgres"),
		DBPass: getEnv("POSTGRES_PASSWORD", "postgres"),
		DBName: getEnv("POSTGRES_DB", "vrooli"),
		Port:   8080,
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
		return nil
	}

	// Create test schema
	testSchema := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			parent_id UUID,
			author_id UUID,
			author_name VARCHAR(255),
			content TEXT NOT NULL,
			content_type VARCHAR(50) DEFAULT 'markdown',
			metadata JSONB DEFAULT '{}',
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			version INT DEFAULT 1,
			thread_path TEXT,
			depth INT DEFAULT 0,
			reply_count INT DEFAULT 0,
			FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS scenario_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) UNIQUE NOT NULL,
			auth_required BOOLEAN DEFAULT true,
			allow_anonymous BOOLEAN DEFAULT false,
			allow_rich_media BOOLEAN DEFAULT false,
			moderation_level VARCHAR(50) DEFAULT 'manual',
			theme_config JSONB DEFAULT '{}',
			notification_settings JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_comments_scenario ON comments(scenario_name);
		CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
		CREATE INDEX IF NOT EXISTS idx_comments_thread ON comments(thread_path);
	`

	if _, err := db.conn.Exec(testSchema); err != nil {
		db.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return &TestDatabase{
		DB:     db,
		Config: config,
		Cleanup: func() {
			// Clean up test data
			db.conn.Exec("TRUNCATE TABLE comments CASCADE")
			db.conn.Exec("TRUNCATE TABLE scenario_configs CASCADE")
			db.Close()
		},
	}
}

// TestComment provides a pre-configured comment for testing
type TestComment struct {
	Comment *Comment
	Cleanup func()
}

// createTestComment creates a test comment in the database
func createTestComment(t *testing.T, db *Database, scenarioName string, content string) *TestComment {
	t.Helper()

	comment := &Comment{
		ScenarioName: scenarioName,
		Content:      content,
		ContentType:  "markdown",
		Status:       "active",
		Metadata:     make(map[string]interface{}),
	}

	if err := db.CreateComment(comment); err != nil {
		t.Fatalf("Failed to create test comment: %v", err)
	}

	return &TestComment{
		Comment: comment,
		Cleanup: func() {
			// Comments will be cleaned up with database cleanup
		},
	}
}

// TestScenarioConfig provides a pre-configured scenario config for testing
type TestScenarioConfig struct {
	Config  *ScenarioConfig
	Cleanup func()
}

// createTestScenarioConfig creates a test scenario config
func createTestScenarioConfig(t *testing.T, db *Database, scenarioName string) *TestScenarioConfig {
	t.Helper()

	config, err := db.CreateDefaultConfig(scenarioName)
	if err != nil {
		t.Fatalf("Failed to create test scenario config: %v", err)
	}

	return &TestScenarioConfig{
		Config: config,
		Cleanup: func() {
			// Configs will be cleaned up with database cleanup
		},
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	Params  map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(app *App, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Execute request
	router := app.setupRouter()
	router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); ok {
		if expectedError != "" && errorMsg != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, errorMsg)
		}
	} else {
		t.Error("Response does not contain error field")
	}
}

// assertCommentFields validates comment structure
func assertCommentFields(t *testing.T, comment map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"id", "scenario_name", "content", "content_type", "status", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := comment[field]; !ok {
			t.Errorf("Comment missing required field: %s", field)
		}
	}
}

// assertConfigFields validates config structure
func assertConfigFields(t *testing.T, config map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"id", "scenario_name", "auth_required", "allow_anonymous", "moderation_level"}
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			t.Errorf("Config missing required field: %s", field)
		}
	}
}

// MockSessionAuthService provides a mock session auth service for testing
type MockSessionAuthService struct {
	ValidateFunc func(token string) (*UserInfo, error)
}

func (m *MockSessionAuthService) ValidateToken(token string) (*UserInfo, error) {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(token)
	}
	// Default mock behavior
	return &UserInfo{
		ID:       uuid.New(),
		Username: "testuser",
		Name:     "Test User",
	}, nil
}

// MockNotificationService provides a mock notification service for testing
type MockNotificationService struct {
	SendFunc func(comment *Comment, config *ScenarioConfig) error
}

// setupTestApp creates a test app with mock services
func setupTestApp(t *testing.T, testDB *TestDatabase) *App {
	t.Helper()

	sessionAuth := &SessionAuthService{
		baseURL: "http://localhost:8001",
		client:  &http.Client{Timeout: 1 * time.Second},
	}

	notifications := &NotificationService{
		baseURL: "http://localhost:8002",
		client:  &http.Client{Timeout: 1 * time.Second},
	}

	// Initialize sanitizer for markdown
	sanitizer := bluemonday.UGCPolicy()

	app := &App{
		config:        testDB.Config,
		db:            testDB.DB,
		sessionAuth:   sessionAuth,
		notifications: notifications,
		sanitizer:     sanitizer,
	}

	return app
}

// waitForCondition polls a condition with timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}
