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
	_ "github.com/lib/pq"
)

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Server      *Server
	DB          *sql.DB
	Redis       *redis.Client
	Cleanup     func()
	TestProfile *Profile
	TestAPIKey  string
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard) // Suppress logs during tests
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// setupTestEnvironment creates a complete test environment with database and Redis
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Set environment variables for testing
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "0") // Use random available port

	// Setup test database connection
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		dbHost := getEnvOrDefault("TEST_POSTGRES_HOST", "localhost")
		dbPort := getEnvOrDefault("TEST_POSTGRES_PORT", "5432")
		dbUser := getEnvOrDefault("TEST_POSTGRES_USER", "postgres")
		dbPassword := getEnvOrDefault("TEST_POSTGRES_PASSWORD", "postgres")
		dbName := getEnvOrDefault("TEST_POSTGRES_DB", "notification_hub_test")

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("Test database not reachable: %v", err)
	}

	// Setup test Redis connection
	redisURL := getEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/15")
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		db.Close()
		t.Skipf("Invalid Redis URL: %v", err)
	}

	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		db.Close()
		t.Skipf("Test Redis not available: %v", err)
	}

	// Create test server
	processor := NewNotificationProcessor(db, rdb)
	server := &Server{
		db:        db,
		redis:     rdb,
		port:      "0",
		processor: processor,
	}
	server.setupRoutes()

	// Create test profile with API key
	profile, apiKey, err := createTestProfile(db)
	if err != nil {
		rdb.Close()
		db.Close()
		t.Fatalf("Failed to create test profile: %v", err)
	}

	env := &TestEnvironment{
		Server:      server,
		DB:          db,
		Redis:       rdb,
		TestProfile: profile,
		TestAPIKey:  apiKey,
		Cleanup: func() {
			// Clean up test data
			cleanupTestData(db, rdb, profile.ID)
			rdb.Close()
			db.Close()
		},
	}

	return env
}

// createTestProfile creates a test profile and returns it with the API key
func createTestProfile(db *sql.DB) (*Profile, string, error) {
	profileID := uuid.New()
	slug := "test-profile-" + profileID.String()[:8]

	// Generate API key
	apiKey := slug + "_" + uuid.New().String()

	// For testing, we'll use a simple hash (in production, use bcrypt)
	// This is just for test convenience
	apiKeyHash := apiKey // In real implementation, this would be bcrypt.GenerateFromPassword

	query := `
		INSERT INTO profiles (id, name, slug, api_key_hash, api_key_prefix, settings, plan, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', NOW(), NOW())
		RETURNING created_at, updated_at
	`

	settings := map[string]interface{}{"test": true}
	var createdAt, updatedAt time.Time

	err := db.QueryRow(query, profileID, "Test Profile", slug, apiKeyHash, slug+"_",
		settings, "test").Scan(&createdAt, &updatedAt)
	if err != nil {
		return nil, "", err
	}

	profile := &Profile{
		ID:           profileID,
		Name:         "Test Profile",
		Slug:         slug,
		APIKeyPrefix: slug + "_",
		Settings:     settings,
		Plan:         "test",
		Status:       "active",
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	return profile, apiKey, nil
}

// createTestContact creates a test contact for the given profile
func createTestContact(db *sql.DB, profileID uuid.UUID) (*Contact, error) {
	contactID := uuid.New()
	identifier := fmt.Sprintf("test-%s@example.com", contactID.String()[:8])
	externalID := "ext-" + contactID.String()[:8]
	firstName := "Test"
	lastName := "User"

	query := `
		INSERT INTO contacts (
			id, profile_id, external_id, identifier, first_name, last_name,
			timezone, locale, preferences, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', NOW(), NOW())
		RETURNING created_at, updated_at
	`

	preferences := map[string]interface{}{"email": true, "sms": false}
	var createdAt, updatedAt time.Time

	err := db.QueryRow(query, contactID, profileID, externalID, identifier,
		firstName, lastName, "UTC", "en", preferences).Scan(&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	contact := &Contact{
		ID:          contactID,
		ProfileID:   profileID,
		ExternalID:  &externalID,
		Identifier:  identifier,
		FirstName:   &firstName,
		LastName:    &lastName,
		Timezone:    "UTC",
		Locale:      "en",
		Preferences: preferences,
		Status:      "active",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	return contact, nil
}

// cleanupTestData removes all test data for a given profile
func cleanupTestData(db *sql.DB, rdb *redis.Client, profileID uuid.UUID) {
	ctx := context.Background()

	// Delete notifications and related data
	db.Exec("DELETE FROM delivery_logs WHERE notification_id IN (SELECT id FROM notifications WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM notifications WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM unsubscribes WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM contacts WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM profiles WHERE id = $1", profileID)

	// Clear Redis test data
	pattern := fmt.Sprintf("*%s*", profileID.String())
	keys, _ := rdb.Keys(ctx, pattern).Result()
	if len(keys) > 0 {
		rdb.Del(ctx, keys...)
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP request, returning the response
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonData)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	env.Server.router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields []string) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field '%s' not found in response: %v", field, response)
		}
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorFragment string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Errorf("Expected 'error' field in response, got: %v", response)
		return
	}

	if expectedErrorFragment != "" {
		errorStr := fmt.Sprintf("%v", errorMsg)
		if errorStr != expectedErrorFragment {
			t.Errorf("Expected error containing '%s', got: '%s'", expectedErrorFragment, errorStr)
		}
	}
}

// Helper function to get environment variable or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// waitForCondition polls a condition function until it returns true or times out
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

// createTestTemplate creates a test template for the given profile
func createTestTemplate(db *sql.DB, profileID uuid.UUID) (*Template, error) {
	templateID := uuid.New()

	query := `
		INSERT INTO templates (
			id, profile_id, name, slug, channels, category,
			subject, content, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', NOW(), NOW())
	`

	name := "Test Template " + templateID.String()[:8]
	slug := "test-template-" + templateID.String()[:8]
	channels := []string{"email", "sms"}
	category := "transactional"
	subject := "Test Subject: {{name}}"
	content := map[string]interface{}{
		"text": "Hello {{name}}, this is a test message: {{message}}",
		"html": "<p>Hello {{name}}, this is a test message: {{message}}</p>",
	}

	_, err := db.Exec(query, templateID, profileID, name, slug, channels,
		category, subject, content)
	if err != nil {
		return nil, err
	}

	template := &Template{
		ID:       templateID,
		Name:     name,
		Slug:     slug,
		Channels: channels,
		Category: category,
		Subject:  &subject,
		Content:  content,
		Status:   "active",
	}

	return template, nil
}

// createTestNotification creates a test notification for testing
func createTestNotification(db *sql.DB, profileID, contactID uuid.UUID) (*Notification, error) {
	notificationID := uuid.New()

	query := `
		INSERT INTO notifications (
			id, profile_id, contact_id, subject, content, variables,
			channels_requested, channels_attempted, priority,
			scheduled_at, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', NOW(), NOW())
		RETURNING created_at
	`

	subject := "Test Notification"
	content := map[string]interface{}{
		"text": "Test notification content",
		"html": "<p>Test notification content</p>",
	}
	variables := map[string]interface{}{
		"name": "Test User",
	}
	channelsRequested := []string{"email"}
	channelsAttempted := []string{}
	priority := "normal"
	scheduledAt := time.Now()

	var createdAt time.Time
	err := db.QueryRow(query, notificationID, profileID, contactID, subject,
		content, variables, channelsRequested, channelsAttempted, priority,
		scheduledAt).Scan(&createdAt)
	if err != nil {
		return nil, err
	}

	notification := &Notification{
		ID:                notificationID,
		ProfileID:         profileID,
		ContactID:         contactID,
		Subject:           &subject,
		Content:           content,
		Variables:         variables,
		ChannelsRequested: channelsRequested,
		ChannelsAttempted: channelsAttempted,
		Priority:          priority,
		ScheduledAt:       scheduledAt,
		Status:            "pending",
		CreatedAt:         createdAt,
	}

	return notification, nil
}

// createUnsubscribe creates an unsubscribe record for testing
func createUnsubscribe(db *sql.DB, profileID, contactID uuid.UUID, channel string) error {
	query := `
		INSERT INTO unsubscribes (
			id, profile_id, contact_id, channel, reason, active, created_at
		) VALUES ($1, $2, $3, $4, $5, true, NOW())
	`

	unsubID := uuid.New()
	reason := "Testing unsubscribe functionality"

	_, err := db.Exec(query, unsubID, profileID, contactID, channel, reason)
	return err
}

// getNotificationCount returns the count of notifications for a profile
func getNotificationCount(db *sql.DB, profileID uuid.UUID, status string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE profile_id = $1 AND status = $2`
	err := db.QueryRow(query, profileID, status).Scan(&count)
	return count, err
}

// getDeliveryLogCount returns the count of delivery logs for a notification
func getDeliveryLogCount(db *sql.DB, notificationID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM delivery_logs WHERE notification_id = $1`
	err := db.QueryRow(query, notificationID).Scan(&count)
	return count, err
}

// assertNotificationStatus verifies notification status
func assertNotificationStatus(t *testing.T, db *sql.DB, notificationID uuid.UUID, expectedStatus string) {
	t.Helper()

	var status string
	query := `SELECT status FROM notifications WHERE id = $1`
	err := db.QueryRow(query, notificationID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to get notification status: %v", err)
	}

	if status != expectedStatus {
		t.Errorf("Expected notification status '%s', got '%s'", expectedStatus, status)
	}
}

// assertContactExists verifies contact exists in database
func assertContactExists(t *testing.T, db *sql.DB, contactID uuid.UUID) {
	t.Helper()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1)`
	err := db.QueryRow(query, contactID).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check contact existence: %v", err)
	}

	if !exists {
		t.Errorf("Expected contact %s to exist", contactID)
	}
}

// assertProfileExists verifies profile exists in database
func assertProfileExists(t *testing.T, db *sql.DB, profileID uuid.UUID) {
	t.Helper()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM profiles WHERE id = $1)`
	err := db.QueryRow(query, profileID).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check profile existence: %v", err)
	}

	if !exists {
		t.Errorf("Expected profile %s to exist", profileID)
	}
}
