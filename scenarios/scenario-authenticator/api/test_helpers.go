package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalFlags int
	originalPrefix string
	cleanup func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Suppress logs during tests unless verbose mode is enabled
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	RedisConnected bool
	TestUsers      []*models.User
	Cleanup        func()
}

// setupTestDatabase initializes a test database connection
func setupTestDatabase(t *testing.T) *TestEnvironment {
	// Use test database URL or in-memory if available
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	// Override to use test database
	testDBURL := os.Getenv("TEST_POSTGRES_URL")
	if testDBURL != "" {
		dbURL = testDBURL
	}

	if err := db.InitDB(dbURL); err != nil {
		t.Skipf("Failed to initialize test database: %v", err)
	}

	// Initialize Redis for session tests
	redisURL := os.Getenv("REDIS_URL")
	redisConnected := false
	if redisURL != "" {
		if err := db.InitRedis(redisURL); err == nil {
			redisConnected = true
		}
	}

	env := &TestEnvironment{
		RedisConnected: redisConnected,
		TestUsers:      []*models.User{},
		Cleanup: func() {
			// Clean up test data
			if db.DB != nil {
				// Delete test users created during tests
				db.DB.Exec("DELETE FROM users WHERE email LIKE '%@test.local'")
				db.DB.Exec("DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@test.local')")
				db.DB.Exec("DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@test.local')")
			}
		},
	}

	return env
}

// TestUser provides a pre-configured user for testing
type TestUser struct {
	User     *models.User
	Password string
	Token    string
	Cleanup  func()
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, email, password string, roles []string) *TestUser {
	if roles == nil {
		roles = []string{"user"}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Email:        email,
		Username:     email[:strings.IndexByte(email, '@')],
		PasswordHash: string(hashedPassword),
		Roles:        roles,
		CreatedAt:    time.Now(),
		EmailVerified: true,
	}

	// Insert into database
	rolesJSON, _ := json.Marshal(roles)
	_, err = db.DB.Exec(`
		INSERT INTO users (id, email, username, password_hash, roles, email_verified, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.ID, user.Email, user.Username, user.PasswordHash, string(rolesJSON), user.EmailVerified, "{}", user.CreatedAt, user.CreatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Generate token
	token, err := auth.GenerateToken(user)
	if err != nil {
		t.Logf("Warning: Failed to generate token for test user: %v", err)
		token = ""
	}

	return &TestUser{
		User:     user,
		Password: password,
		Token:    token,
		Cleanup: func() {
			db.DB.Exec("DELETE FROM users WHERE id = $1", user.ID)
			db.DB.Exec("DELETE FROM sessions WHERE user_id = $1", user.ID)
			db.DB.Exec("DELETE FROM api_keys WHERE user_id = $1", user.ID)
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and returns an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*http.Request, error) {
	var bodyReader io.Reader

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
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
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

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	return httpReq, nil
}

// executeRequest executes an HTTP request against a handler
func executeRequest(handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler(w, req)
	return w
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

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorSubstring != "" {
		errorStr, ok := errorMsg.(string)
		if !ok {
			t.Errorf("Expected error to be string, got %T", errorMsg)
			return
		}
		if !contains(errorStr, expectedErrorSubstring) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorStr)
		}
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// assertResponseHeader validates response headers
func assertResponseHeader(t *testing.T, w *httptest.ResponseRecorder, headerName, expectedValue string) {
	actualValue := w.Header().Get(headerName)
	if actualValue != expectedValue {
		t.Errorf("Expected header '%s' to be '%s', got '%s'", headerName, expectedValue, actualValue)
	}
}

// assertResponseContains checks if response body contains a substring
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, substring string) {
	body := w.Body.String()
	if !contains(body, substring) {
		t.Errorf("Expected response to contain '%s', got: %s", substring, body)
	}
}

// createAuthHeader creates an Authorization header with Bearer token
func createAuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}

// mockAuthContext adds authentication context to a request for testing middleware
func mockAuthContext(req *http.Request, user *models.User) *http.Request {
	claims := &models.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
	}
	ctx := context.WithValue(req.Context(), "claims", claims)
	return req.WithContext(ctx)
}

// SecurityTestPayload provides common security test payloads
type SecurityTestPayload struct {
	SQLInjection []string
	XSS          []string
	PathTraversal []string
	CommandInjection []string
}

// GetSecurityTestPayloads returns common security attack payloads
func GetSecurityTestPayloads() *SecurityTestPayload {
	return &SecurityTestPayload{
		SQLInjection: []string{
			"' OR '1'='1",
			"'; DROP TABLE users--",
			"' UNION SELECT * FROM users--",
			"admin'--",
			"1' OR '1' = '1' /*",
			"' OR 1=1--",
		},
		XSS: []string{
			"<script>alert('XSS')</script>",
			"<img src=x onerror=alert('XSS')>",
			"javascript:alert('XSS')",
			"<svg/onload=alert('XSS')>",
			"'\"><script>alert(String.fromCharCode(88,83,83))</script>",
		},
		PathTraversal: []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"....//....//....//etc/passwd",
		},
		CommandInjection: []string{
			"; ls -la",
			"| cat /etc/passwd",
			"`whoami`",
			"$(whoami)",
		},
	}
}

func indexSubstring(s, substr string) int {
	n := len(substr)
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
