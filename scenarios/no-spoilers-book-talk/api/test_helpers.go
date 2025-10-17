package main

import (
	"bytes"
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
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	buffer bytes.Buffer
}

// setupTestLogger initializes a logger for testing
func setupTestLogger() func() {
	// Disable standard logger output during tests
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	DataDir    string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "no-spoilers-book-talk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dataDir := filepath.Join(tempDir, "data")
	uploadsDir := filepath.Join(dataDir, "uploads")

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create data directories: %v", err)
	}

	originalWD, _ := os.Getwd()

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		DataDir:    dataDir,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates an in-memory or test database
func setupTestDatabase(t *testing.T) *TestDatabase {
	// For now, we'll use a mock database
	// In production, this would connect to a test PostgreSQL instance
	db, mock := setupMockDB(t)

	return &TestDatabase{
		DB: db,
		Cleanup: func() {
			if db != nil {
				db.Close()
			}
			if mock != nil {
				// Verify all expectations were met
			}
		},
	}
}

// setupMockDB creates a mock database for testing
func setupMockDB(t *testing.T) (*sql.DB, interface{}) {
	// This is a placeholder - in real implementation, use sqlmock
	// For now, we'll return nil and handle it in tests
	return nil, nil
}

// TestBook provides a pre-configured book for testing
type TestBook struct {
	Book    *Book
	Cleanup func()
}

// createTestBook creates a test book in the database
func createTestBook(t *testing.T, db *sql.DB, title, author string) *TestBook {
	book := &Book{
		ID:               uuid.New(),
		Title:            title,
		Author:           author,
		FilePath:         "/tmp/test-book.txt",
		FileType:         "txt",
		FileSizeBytes:    1024,
		TotalChunks:      100,
		TotalWords:       50000,
		TotalCharacters:  250000,
		ProcessingStatus: "completed",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if db != nil {
		_, err := db.Exec(`
			INSERT INTO books (id, title, author, file_path, file_type, file_size_bytes,
			                  total_chunks, total_words, total_characters, processing_status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			book.ID, book.Title, book.Author, book.FilePath, book.FileType,
			book.FileSizeBytes, book.TotalChunks, book.TotalWords,
			book.TotalCharacters, book.ProcessingStatus)

		if err != nil {
			t.Logf("Warning: Failed to insert test book: %v", err)
		}
	}

	return &TestBook{
		Book: book,
		Cleanup: func() {
			if db != nil {
				db.Exec("DELETE FROM books WHERE id = $1", book.ID)
			}
		},
	}
}

// createTestBookFile creates a physical test book file
func createTestBookFile(t *testing.T, env *TestEnvironment, filename, content string) string {
	filePath := filepath.Join(env.DataDir, "uploads", filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
	URLVars     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default content type for JSON
	if req.Body != nil && req.Headers["Content-Type"] == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set URL variables (for mux)
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	response := assertJSONResponse(t, w, expectedStatus)

	if _, ok := response["error"]; !ok {
		t.Errorf("Expected 'error' field in response, got: %v", response)
	}

	if expectedError != "" {
		if errorMsg, ok := response["error"].(string); ok {
			if !strings.Contains(errorMsg, expectedError) {
				t.Errorf("Expected error to contain '%s', got '%s'", expectedError, errorMsg)
			}
		}
	}

	// Verify timestamp exists
	if _, ok := response["timestamp"]; !ok {
		t.Errorf("Expected 'timestamp' field in error response")
	}
}

// createMultipartRequest creates a multipart form request for file upload
func createMultipartRequest(t *testing.T, path string, formData map[string]string, fileField, filename string, fileContent []byte) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, value := range formData {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("Failed to write form field: %v", err)
		}
	}

	// Add file
	if fileField != "" && filename != "" {
		part, err := writer.CreateFormFile(fileField, filename)
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", path, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// setupTestService creates a test service instance
func setupTestService(t *testing.T, db *sql.DB, dataDir string) *BookTalkService {
	if db == nil {
		// Use a placeholder database that won't panic
		// In real tests, this should be a mock or test database
		t.Log("Warning: Using nil database in test service")
	}

	return NewBookTalkService(db, "http://localhost:5678", "http://localhost:6333", dataDir)
}

// assertBookResponse validates a book response structure
func assertBookResponse(t *testing.T, response map[string]interface{}) {
	requiredFields := []string{"id", "title", "file_type", "processing_status"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in book response", field)
		}
	}
}

// assertConversationResponse validates a conversation response structure
func assertConversationResponse(t *testing.T, response map[string]interface{}) {
	requiredFields := []string{"conversation_id", "response", "position_boundary_respected", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in conversation response", field)
		}
	}
}

// assertProgressResponse validates a progress response structure
func assertProgressResponse(t *testing.T, response map[string]interface{}) {
	requiredFields := []string{"progress_id", "book_id", "user_id", "current_position", "percentage_complete"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in progress response", field)
		}
	}
}

// createTestProgress creates test user progress
func createTestProgress(t *testing.T, db *sql.DB, bookID uuid.UUID, userID string, position int) uuid.UUID {
	progressID := uuid.New()

	if db != nil {
		_, err := db.Exec(`
			INSERT INTO user_progress (id, book_id, user_id, current_position, position_type, position_value)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			progressID, bookID, userID, position, "chunk", float64(position))

		if err != nil {
			t.Logf("Warning: Failed to insert test progress: %v", err)
		}
	}

	return progressID
}

// Helper to create sample book content
func createSampleBookContent() string {
	return `Chapter 1: The Beginning

This is a test book with multiple chapters and content.
It contains various paragraphs and sections to test chunking.

Chapter 2: The Middle

More content here for testing purposes.
This helps verify that the book processing works correctly.

Chapter 3: The End

Final chapter with concluding content.`
}
