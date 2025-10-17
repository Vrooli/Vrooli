package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes controlled logging for testing
func setupTestLogger() func() {
	// Capture original log settings
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	// Set test logger with clear prefix
	log.SetPrefix("[TEST] ")
	log.SetFlags(log.Ltime | log.Lshortfile)

	return func() {
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDB      *sql.DB
	TempRedis   *redis.Client
	OriginalDB  *sql.DB
	OriginalRDB *redis.Client
	Ctx         context.Context
	Cleanup     func()
}

// setupTestDatabase creates isolated test database connections
func setupTestDatabase(t *testing.T) *TestEnvironment {
	// Store original connections
	origDB := db
	origRDB := rdb
	testCtx := context.Background()

	// Get database configuration from environment
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			t.Skip("Skipping test: Database configuration not available")
			return nil
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Connect to test database
	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Failed to ping test database: %v", err)
		return nil
	}

	// Connect to Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	testRedis := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   1, // Use DB 1 for tests
	})

	// Test Redis connection
	if _, err := testRedis.Ping(testCtx).Result(); err != nil {
		testDB.Close()
		testRedis.Close()
		t.Skipf("Failed to connect to Redis: %v", err)
		return nil
	}

	// Set global test connections
	db = testDB
	rdb = testRedis
	ctx = testCtx

	return &TestEnvironment{
		TempDB:      testDB,
		TempRedis:   testRedis,
		OriginalDB:  origDB,
		OriginalRDB: origRDB,
		Ctx:         testCtx,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM visitor_events")
			testDB.Exec("DELETE FROM visitor_sessions")
			testDB.Exec("DELETE FROM visitors")

			// Flush Redis test database
			testRedis.FlushDB(testCtx)

			// Close connections
			testDB.Close()
			testRedis.Close()

			// Restore original connections
			db = origDB
			rdb = origRDB
		},
	}
}

// TestVisitor provides a pre-configured visitor for testing
type TestVisitor struct {
	Visitor  *Visitor
	Sessions []VisitorSession
	Events   []VisitorEvent
	Cleanup  func()
}

// setupTestVisitor creates a test visitor with sample data
func setupTestVisitor(t *testing.T, fingerprint string) *TestVisitor {
	now := time.Now()
	userAgent := "Mozilla/5.0 Test Browser"
	ipAddress := "192.168.1.100"

	visitor := &Visitor{
		ID:                    uuid.New().String(),
		Fingerprint:           fingerprint,
		FirstSeen:             now.Add(-24 * time.Hour),
		LastSeen:              now,
		SessionCount:          3,
		Identified:            false,
		UserAgent:             &userAgent,
		IPAddress:             &ipAddress,
		TotalPageViews:        15,
		TotalSessionDuration:  1800,
		Tags:                  []string{"test", "visitor"},
	}

	// Insert visitor into database
	insertQuery := `
		INSERT INTO visitors (id, fingerprint, first_seen, last_seen, session_count,
							  identified, user_agent, ip_address, total_page_views,
							  total_session_duration, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := db.Exec(insertQuery, visitor.ID, visitor.Fingerprint, visitor.FirstSeen,
		visitor.LastSeen, visitor.SessionCount, visitor.Identified, visitor.UserAgent,
		visitor.IPAddress, visitor.TotalPageViews, visitor.TotalSessionDuration, visitor.Tags)

	if err != nil {
		t.Fatalf("Failed to create test visitor: %v", err)
	}

	// Create sample sessions
	sessions := []VisitorSession{
		{
			ID:        uuid.New().String(),
			VisitorID: visitor.ID,
			Scenario:  "test-scenario",
			StartTime: now.Add(-2 * time.Hour),
			PageViews: 5,
		},
		{
			ID:        uuid.New().String(),
			VisitorID: visitor.ID,
			Scenario:  "test-scenario-2",
			StartTime: now.Add(-1 * time.Hour),
			PageViews: 8,
		},
	}

	return &TestVisitor{
		Visitor:  visitor,
		Sessions: sessions,
		Events:   []VisitorEvent{},
		Cleanup: func() {
			db.Exec("DELETE FROM visitor_events WHERE visitor_id IN (SELECT id FROM visitors WHERE fingerprint = $1)", fingerprint)
			db.Exec("DELETE FROM visitor_sessions WHERE visitor_id = $1", visitor.ID)
			db.Exec("DELETE FROM visitors WHERE id = $1", visitor.ID)
			rdb.Del(ctx, "visitor:"+fingerprint)
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

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
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

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContainMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// For simple text error responses
	body := w.Body.String()
	if shouldContainMessage != "" && body != "" {
		// Try to parse as JSON first
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
			// Check for error field
			if errMsg, ok := response["message"].(string); ok {
				body = errMsg
			}
		}
	}
}

// assertHealthResponse validates health check responses
func assertHealthResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) *HealthStatus {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	var health HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
		return nil
	}

	// Validate required fields
	if health.Status == "" {
		t.Error("Health status should not be empty")
	}
	if health.Services == nil {
		t.Error("Health services should not be nil")
	}

	return &health
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// TrackEventRequest creates a test tracking event request
func (g *TestDataGenerator) TrackEventRequest(fingerprint, scenario, eventType, pageURL string) VisitorEvent {
	return VisitorEvent{
		Fingerprint: fingerprint,
		Scenario:    scenario,
		EventType:   eventType,
		PageURL:     pageURL,
		Timestamp:   time.Now(),
		Properties:  map[string]interface{}{"test": "data"},
	}
}

// VisitorEventWithSession creates event with session ID
func (g *TestDataGenerator) VisitorEventWithSession(fingerprint, sessionID, scenario, eventType string) VisitorEvent {
	return VisitorEvent{
		Fingerprint: fingerprint,
		SessionID:   sessionID,
		Scenario:    scenario,
		EventType:   eventType,
		PageURL:     "/test/page",
		Timestamp:   time.Now(),
		Properties:  map[string]interface{}{},
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
