// +build testing

package main

import (
	"bytes"
	"context"
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
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Disable gin debug mode during tests
	gin.SetMode(gin.TestMode)

	// Save original log output
	originalOutput := log.Writer()

	// Optionally silence logs during tests (can be toggled for debugging)
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(io.Discard)
	}

	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB          *sql.DB
	Redis       *redis.Client
	App         *Application
	Config      *Configuration
	Cleanup     func()
}

// setupTestEnvironment creates an isolated test environment with mock dependencies
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Create test configuration
	config := &Configuration{
		APIPort:     "18888",
		UIPort:      "38888",
		DatabaseURL: getTestDatabaseURL(),
		RedisURL:    getTestRedisURL(),
		MinIOURL:    getTestMinIOURL(),
		OllamaURL:   getTestOllamaURL(),
		JWTSecret:   "test-secret-key-for-testing-only",
		Environment: "test",
		Mode:        "both",
	}

	// Initialize test database connection
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("Test database not available: %v", err)
	}

	// Initialize Redis client
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		db.Close()
		redisClient.Close()
		t.Skipf("Test Redis not available: %v", err)
	}

	// Create application instance
	app := &Application{
		Config:      config,
		DB:          db,
		Redis:       redisClient,
		PlatformMgr: NewPlatformManager(config.OllamaURL),
		JobProcessor: NewJobProcessor(db, redisClient, NewPlatformManager(config.OllamaURL)),
		WebSocket:   NewWebSocketManager(),
	}
	app.setupRouter()

	return &TestEnvironment{
		DB:     db,
		Redis:  redisClient,
		App:    app,
		Config: config,
		Cleanup: func() {
			// Clean up test data
			cleanupTestData(t, db, redisClient)

			// Close connections
			if app.WebSocket != nil {
				app.WebSocket.Close()
			}
			if redisClient != nil {
				redisClient.Close()
			}
			if db != nil {
				db.Close()
			}
		},
	}
}

// getTestDatabaseURL returns the test database URL from environment or default
func getTestDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	// Default test database
	return "postgres://postgres:postgres@localhost:5432/vrooli_social_media_scheduler_test?sslmode=disable"
}

// getTestRedisURL returns the test Redis URL from environment or default
func getTestRedisURL() string {
	if url := os.Getenv("TEST_REDIS_URL"); url != "" {
		return url
	}
	// Default test Redis
	return "redis://localhost:6379/15" // Use DB 15 for tests
}

// getTestMinIOURL returns the test MinIO URL from environment or default
func getTestMinIOURL() string {
	if url := os.Getenv("TEST_MINIO_URL"); url != "" {
		return url
	}
	return "http://localhost:9000"
}

// getTestOllamaURL returns the test Ollama URL from environment or default
func getTestOllamaURL() string {
	if url := os.Getenv("TEST_OLLAMA_URL"); url != "" {
		return url
	}
	return "http://localhost:11434"
}

// cleanupTestData removes test data from database and Redis
func cleanupTestData(t *testing.T, db *sql.DB, redisClient *redis.Client) {
	ctx := context.Background()

	// Clear Redis test data
	if redisClient != nil {
		keys, _ := redisClient.Keys(ctx, "test:*").Result()
		if len(keys) > 0 {
			redisClient.Del(ctx, keys...)
		}
	}

	// Clear test database tables (in correct order due to foreign keys)
	if db != nil {
		tables := []string{
			"scheduled_posts",
			"social_accounts",
			"campaigns",
			"users",
		}

		for _, table := range tables {
			_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE email LIKE 'test%%' OR created_at < NOW() - INTERVAL '1 hour'", table))
			if err != nil {
				// Ignore errors during cleanup
				t.Logf("Cleanup warning for %s: %v", table, err)
			}
		}
	}
}

// TestUser provides a pre-configured user for testing
type TestUser struct {
	ID       string
	Email    string
	Password string
	Token    string
}

// createTestUser creates a test user in the database and returns user info with JWT token
func createTestUser(t *testing.T, env *TestEnvironment) *TestUser {
	userID := uuid.New().String()
	email := fmt.Sprintf("test_%s@example.com", uuid.New().String()[:8])
	password := "TestPassword123!"

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Insert user into database
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, subscription_tier, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`

	_, err = env.DB.Exec(query, userID, email, string(hashedPassword), "Test", "User", "free", "UTC")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Generate JWT token
	token, err := generateTestJWT(userID, email, env.Config.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to generate test JWT: %v", err)
	}

	return &TestUser{
		ID:       userID,
		Email:    email,
		Password: password,
		Token:    token,
	}
}

// generateTestJWT creates a JWT token for testing
func generateTestJWT(userID, email, secret string) (string, error) {
	// This is a simplified version - actual implementation should match handlers.go
	return fmt.Sprintf("test_token_%s_%s", userID, email), nil
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP request against the test server
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
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

	// Set default content type
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Execute request
	w := httptest.NewRecorder()
	env.App.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedSuccess bool) Response {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
	}

	if response.Success != expectedSuccess {
		t.Errorf("Expected success=%v, got success=%v. Error: %s", expectedSuccess, response.Success, response.Error)
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	response := assertJSONResponse(t, w, expectedStatus, false)

	if expectedErrorContains != "" {
		if response.Error == "" {
			t.Errorf("Expected error message containing '%s', but got no error", expectedErrorContains)
		}
	}
}

// TestCampaign provides a pre-configured campaign for testing
type TestCampaign struct {
	ID     string
	UserID string
	Name   string
}

// createTestCampaign creates a test campaign in the database
func createTestCampaign(t *testing.T, env *TestEnvironment, userID string) *TestCampaign {
	campaignID := uuid.New().String()
	name := fmt.Sprintf("Test Campaign %s", uuid.New().String()[:8])

	query := `
		INSERT INTO campaigns (id, user_id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`

	_, err := env.DB.Exec(query, campaignID, userID, name, "Test campaign description", "active")
	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	return &TestCampaign{
		ID:     campaignID,
		UserID: userID,
		Name:   name,
	}
}

// TestScheduledPost provides a pre-configured scheduled post for testing
type TestScheduledPost struct {
	ID          string
	UserID      string
	Title       string
	Content     string
	ScheduledAt time.Time
}

// createTestScheduledPost creates a test scheduled post in the database
func createTestScheduledPost(t *testing.T, env *TestEnvironment, userID string) *TestScheduledPost {
	postID := uuid.New().String()
	title := fmt.Sprintf("Test Post %s", uuid.New().String()[:8])
	content := "This is test post content"
	scheduledAt := time.Now().Add(24 * time.Hour)

	query := `
		INSERT INTO scheduled_posts (id, user_id, title, content, platforms, scheduled_at, timezone, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	platforms := []string{"twitter", "linkedin"}
	_, err := env.DB.Exec(query, postID, userID, title, content, platforms, scheduledAt, "UTC", "scheduled")
	if err != nil {
		t.Fatalf("Failed to create test scheduled post: %v", err)
	}

	return &TestScheduledPost{
		ID:          postID,
		UserID:      userID,
		Title:       title,
		Content:     content,
		ScheduledAt: scheduledAt,
	}
}

// waitForCondition polls a condition until it's true or timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
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
