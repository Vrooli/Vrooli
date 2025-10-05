// +build testing

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(ioutil.Discard) // Suppress logs during tests
	return func() { log.SetOutput(originalOutput) }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Router     *gin.Engine
	EncKey     []byte
	UserID     string
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with database
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Setup logger
	loggerCleanup := setupTestLogger()

	// Create temp directory
	tempDir, err := ioutil.TempDir("", "period-tracker-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Setup encryption key
	salt := []byte("period-tracker-salt-2024")
	testEncryptionKey := deriveKey("test-encryption-key", salt)

	// Setup test database connection
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Build from individual components for testing
		dbHost := getEnvOrDefault("POSTGRES_HOST", "localhost")
		dbPort := getEnvOrDefault("POSTGRES_PORT", "5432")
		dbUser := getEnvOrDefault("POSTGRES_USER", "postgres")
		dbPassword := getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
		dbName := getEnvOrDefault("POSTGRES_DB", "period_tracker_test")

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test database connection (with timeout for CI environments)
	testDB.SetConnMaxLifetime(time.Second * 30)
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(2)

	// Try to ping database with timeout
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = testDB.PingContext(pingCtx)
	if err != nil {
		testDB.Close()
		os.RemoveAll(tempDir)
		t.Skipf("Skipping test - database not available: %v", err)
	}

	// Set globals for testing
	db = testDB
	encryptionKey = testEncryptionKey

	// Create test user ID
	testUserID := uuid.New().String()

	// Setup Gin router in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Setup routes
	setupRoutes(router)

	env := &TestEnvironment{
		DB:         testDB,
		Router:     router,
		EncKey:     testEncryptionKey,
		UserID:     testUserID,
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			testDB.Close()
			os.RemoveAll(tempDir)
			loggerCleanup()
		},
	}

	return env
}

// setupRoutes configures routes for testing
func setupRoutes(r *gin.Engine) {
	// Public routes
	r.GET("/health", healthCheck)

	// API routes with authentication
	api := r.Group("/api/v1")
	api.Use(authMiddleware())
	api.Use(auditMiddleware())
	{
		// Cycle management
		api.POST("/cycles", createCycle)
		api.GET("/cycles", getCycles)

		// Symptom logging
		api.POST("/symptoms", logSymptoms)
		api.GET("/symptoms", getSymptoms)

		// Predictions
		api.GET("/predictions", getPredictions)

		// Pattern detection
		api.GET("/patterns", getPatterns)

		// Health and encryption status
		api.GET("/health/encryption", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"encryption_enabled": true,
				"algorithm":          "AES-GCM",
				"key_derivation":     "PBKDF2",
				"status":             "active",
			})
		})

		api.GET("/auth/status", func(c *gin.Context) {
			userID := c.GetString("user_id")
			c.JSON(http.StatusOK, gin.H{
				"authenticated": true,
				"user_id":       userID,
				"multi_tenant":  true,
			})
		})
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	UserID  string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set user ID header for authentication
	userID := req.UserID
	if userID == "" {
		userID = env.UserID
	}
	httpReq.Header.Set("X-User-ID", userID)

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response structure and status
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedKeys []string) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	if expectedStatus >= 200 && expectedStatus < 300 {
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json; charset=utf-8" && contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	for _, key := range expectedKeys {
		if _, exists := response[key]; !exists {
			t.Errorf("Expected response key '%s' not found. Response: %v", key, response)
		}
	}

	return response
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Body: %s", err, w.Body.String())
	}

	if _, exists := response["error"]; !exists {
		t.Errorf("Expected 'error' key in error response. Response: %v", response)
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, env *TestEnvironment, userID string) {
	tables := []string{"audit_logs", "detected_patterns", "predictions", "daily_symptoms", "cycles"}
	for _, table := range tables {
		_, err := env.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE user_id = $1", table), userID)
		if err != nil {
			t.Logf("Warning: Failed to clean up %s: %v", table, err)
		}
	}
}

// Helper function for tests to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createTestCycle creates a test cycle in the database
func createTestCycle(t *testing.T, env *TestEnvironment, userID, startDate, flowIntensity, notes string) string {
	cycleID := uuid.New().String()
	notesEncrypted, err := encryptString(notes)
	if err != nil {
		t.Fatalf("Failed to encrypt notes: %v", err)
	}

	_, err = env.DB.Exec(`
		INSERT INTO cycles (id, user_id, start_date, flow_intensity, notes_encrypted)
		VALUES ($1, $2, $3, $4, $5)`,
		cycleID, userID, startDate, flowIntensity, notesEncrypted)

	if err != nil {
		t.Fatalf("Failed to create test cycle: %v", err)
	}

	return cycleID
}

// createTestSymptom creates a test symptom entry in the database
func createTestSymptom(t *testing.T, env *TestEnvironment, userID, date string, moodRating int) string {
	symptomID := uuid.New().String()
	symptomsJSON, _ := json.Marshal([]string{"headache", "cramps"})
	symptomsEncrypted, err := encryptString(string(symptomsJSON))
	if err != nil {
		t.Fatalf("Failed to encrypt symptoms: %v", err)
	}

	_, err = env.DB.Exec(`
		INSERT INTO daily_symptoms (id, user_id, symptom_date, physical_symptoms_encrypted, mood_rating)
		VALUES ($1, $2, $3, $4, $5)`,
		symptomID, userID, date, symptomsEncrypted, moodRating)

	if err != nil {
		t.Fatalf("Failed to create test symptom: %v", err)
	}

	return symptomID
}
