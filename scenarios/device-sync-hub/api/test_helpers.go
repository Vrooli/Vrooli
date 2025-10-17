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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// Global test logger
var testLogger *log.Logger

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	testLogger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() {
		testLogger = nil
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir     string
	OriginalWD  string
	StoragePath string
	Cleanup     func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "device-sync-hub-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create storage directories
	storagePath := filepath.Join(tempDir, "files")
	thumbnailPath := filepath.Join(storagePath, "thumbnails")

	if err := os.MkdirAll(thumbnailPath, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create storage directories: %v", err)
	}

	return &TestEnvironment{
		TempDir:     tempDir,
		OriginalWD:  originalWD,
		StoragePath: storagePath,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// TestServer wraps Server for testing
type TestServer struct {
	Server   *Server
	DB       *sql.DB
	Router   *mux.Router
	Env      *TestEnvironment
	Cleanup  func()
}

// setupTestServer creates a test server with in-memory database
func setupTestServer(t *testing.T) *TestServer {
	cleanup := setupTestLogger()
	env := setupTestDirectory(t)

	// Use test database if available, otherwise skip
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		t.Skip("POSTGRES_URL not set, skipping database-dependent test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		env.Cleanup()
		cleanup()
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		env.Cleanup()
		cleanup()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	config := &Config{
		Port:               "0",
		UIPort:             "0",
		AuthServiceURL:     "", // Empty for test mode
		StoragePath:        env.StoragePath,
		MaxFileSize:        10485760, // 10MB
		DefaultExpiryHours: 24,
		ThumbnailSize:      200,
	}

	server := &Server{
		config:      config,
		db:          db,
		connections: make(map[string]*WebSocketConnection),
		broadcast:   make(chan BroadcastMessage, 100),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}

	// Try to connect to Redis if available
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		server.redis = connectRedis()
	}

	// Setup routes (this creates and assigns the router)
	server.setupRoutes()

	testServer := &TestServer{
		Server:  server,
		DB:      db,
		Router:  server.router, // Use the router created by setupRoutes
		Env:     env,
		Cleanup: func() {
			// Clean up test data
			db.Exec("DELETE FROM sync_items")
			db.Exec("DELETE FROM device_sessions")
			db.Close()
			env.Cleanup()
			cleanup()
		},
	}

	return testServer
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	URLVars     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func (ts *TestServer) makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			jsonBody, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if _, ok := req.Headers["Content-Type"]; !ok && req.Body != nil {
		req.Headers["Content-Type"] = "application/json"
	}

	// Add default test authorization if not present and path requires auth
	if _, ok := req.Headers["Authorization"]; !ok && strings.HasPrefix(req.Path, "/api/v1") {
		req.Headers["Authorization"] = "Bearer test-token-" + uuid.New().String()
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set URL vars for mux
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Fatalf("Response missing 'error' field or not a string: %+v", response)
	}

	if expectedErrorContains != "" && !strings.Contains(errorMsg, expectedErrorContains) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorContains, errorMsg)
	}
}

// createTestUser creates a test user for authentication
func createTestUser() *User {
	return &User{
		ID:    uuid.New().String(),
		Email: "test@example.com",
		Roles: []string{"user"},
	}
}

// createTestDevice creates a test device
func (ts *TestServer) createTestDevice(t *testing.T, userID string) *Device {
	t.Helper()

	deviceID := uuid.New().String()
	device := &Device{
		ID:           deviceID,
		UserID:       userID,
		Name:         "Test Device",
		Type:         "desktop",
		Platform:     "linux",
		LastSeen:     time.Now(),
		Capabilities: []string{"clipboard", "files", "notifications"},
		IsOnline:     true,
	}

	query := `
		INSERT INTO device_sessions (id, user_id, device_info, websocket_id, last_seen, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	deviceInfo := map[string]interface{}{
		"name":         device.Name,
		"type":         device.Type,
		"platform":     device.Platform,
		"capabilities": device.Capabilities,
	}

	deviceInfoJSON, _ := json.Marshal(deviceInfo)
	websocketID := fmt.Sprintf("ws-%s", deviceID)

	_, err := ts.DB.Exec(query, device.ID, device.UserID, deviceInfoJSON, websocketID, device.LastSeen, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}

	return device
}

// createTestSyncItem creates a test sync item
func (ts *TestServer) createTestSyncItem(t *testing.T, userID, itemType string) *SyncItem {
	t.Helper()

	item := &SyncItem{
		ID:            uuid.New().String(),
		UserID:        userID,
		Type:          itemType,
		Content:       map[string]interface{}{"data": "test content"},
		SourceDevice:  uuid.New().String(),
		TargetDevices: []string{},
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Status:        "active",
	}

	query := `
		INSERT INTO sync_items (id, user_id, type, content, source_device, target_devices, created_at, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	contentJSON, _ := json.Marshal(item.Content)
	targetDevicesJSON, _ := json.Marshal(item.TargetDevices)

	_, err := ts.DB.Exec(query, item.ID, item.UserID, item.Type, contentJSON,
		item.SourceDevice, targetDevicesJSON, item.CreatedAt, item.ExpiresAt, item.Status)
	if err != nil {
		t.Fatalf("Failed to create test sync item: %v", err)
	}

	return item
}

// createMultipartFileRequest creates a multipart file upload request
func createMultipartFileRequest(t *testing.T, fieldName, fileName string, content []byte, extraFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add extra fields
	for key, value := range extraFields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("Failed to write field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", description)
}

// cleanupExpiredItems manually triggers cleanup for testing
func (ts *TestServer) cleanupExpiredItems(t *testing.T) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ts.DB.ExecContext(ctx, `
		DELETE FROM sync_items
		WHERE expires_at < NOW() AND status = 'active'
	`)

	if err != nil {
		t.Fatalf("Failed to cleanup expired items: %v", err)
	}

	affected, _ := result.RowsAffected()
	return int(affected)
}

// getSyncItemCount returns the number of sync items for a user
func (ts *TestServer) getSyncItemCount(t *testing.T, userID string) int {
	t.Helper()

	var count int
	err := ts.DB.QueryRow(`
		SELECT COUNT(*) FROM sync_items
		WHERE user_id = $1 AND status = 'active'
	`, userID).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to get sync item count: %v", err)
	}

	return count
}

// getDeviceCount returns the number of devices for a user
func (ts *TestServer) getDeviceCount(t *testing.T, userID string) int {
	t.Helper()

	var count int
	err := ts.DB.QueryRow(`
		SELECT COUNT(*) FROM device_sessions
		WHERE user_id = $1
	`, userID).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to get device count: %v", err)
	}

	return count
}
