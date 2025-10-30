//go:build testing
// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Disable logging during tests to reduce noise
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Router     *mux.Router
	TestUserID string
	Cleanup    func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		// Try to build from components - use same database as scenario for testing
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")

		if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" {
			// Use the notes database directly for testing
			// In production, you might want a separate test database
			dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/notes?sslmode=disable",
				dbUser, dbPassword, dbHost, dbPort)
		} else {
			t.Skip("Database configuration not available for testing - set POSTGRES_* environment variables")
		}
	}

	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip(fmt.Sprintf("Failed to connect to test database: %v - tests require PostgreSQL", err))
	}

	// Test connection with timeout
	testDB.SetConnMaxLifetime(time.Second * 5)
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skip(fmt.Sprintf("Failed to ping test database: %v - ensure PostgreSQL is running", err))
	}

	return testDB
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	testDB := setupTestDB(t)

	// Set the global db variable for handlers to use
	db = testDB

	// Verify schema exists
	var tableExists bool
	err := testDB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'users'
		)
	`).Scan(&tableExists)

	if err != nil || !tableExists {
		testDB.Close()
		t.Skip("Database schema not initialized - run scenario setup first")
	}

	// Create a test user
	var testUserID string
	err = testDB.QueryRow(`
		INSERT INTO users (username, email, password_hash, preferences)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
		RETURNING id
	`, "test_user", "test@notes.local", "test_hash", `{"theme": "dark"}`).Scan(&testUserID)

	if err != nil {
		testDB.Close()
		t.Skipf("Failed to create test user: %v - database schema may not be initialized", err)
	}

	// Set default user ID
	defaultUserID = testUserID

	// Setup router with all routes
	router := mux.NewRouter()
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/notes", getNotesHandler).Methods("GET")
	router.HandleFunc("/api/notes", createNoteHandler).Methods("POST")
	router.HandleFunc("/api/notes/{id}", getNoteHandler).Methods("GET")
	router.HandleFunc("/api/notes/{id}", updateNoteHandler).Methods("PUT")
	router.HandleFunc("/api/notes/{id}", deleteNoteHandler).Methods("DELETE")
	router.HandleFunc("/api/folders", getFoldersHandler).Methods("GET")
	router.HandleFunc("/api/folders", createFolderHandler).Methods("POST")
	router.HandleFunc("/api/folders/{id}", updateFolderHandler).Methods("PUT")
	router.HandleFunc("/api/folders/{id}", deleteFolderHandler).Methods("DELETE")
	router.HandleFunc("/api/tags", getTagsHandler).Methods("GET")
	router.HandleFunc("/api/tags", createTagHandler).Methods("POST")
	router.HandleFunc("/api/templates", getTemplatesHandler).Methods("GET")
	router.HandleFunc("/api/templates", createTemplateHandler).Methods("POST")
	router.HandleFunc("/api/search", searchHandler).Methods("POST")
	router.HandleFunc("/api/search/semantic", semanticSearchHandler).Methods("POST")

	return &TestEnvironment{
		DB:         testDB,
		Router:     router,
		TestUserID: testUserID,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM note_tags WHERE note_id IN (SELECT id FROM notes WHERE user_id = $1)", testUserID)
			testDB.Exec("DELETE FROM notes WHERE user_id = $1", testUserID)
			testDB.Exec("DELETE FROM folders WHERE user_id = $1", testUserID)
			testDB.Exec("DELETE FROM templates WHERE user_id = $1", testUserID)
			testDB.Exec("DELETE FROM users WHERE id = $1", testUserID)
			testDB.Close()
		},
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates and executes an HTTP request
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) *httptest.ResponseRecorder {
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

	// Set default headers
	if req.Headers != nil {
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}
	}
	if bodyReader != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Record response
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// TestNote provides a pre-configured note for testing
type TestNote struct {
	Note    *Note
	Cleanup func()
}

// createTestNote creates a note in the database for testing
func createTestNote(t *testing.T, env *TestEnvironment, title, content string) *TestNote {
	note := &Note{
		ID:           uuid.New().String(),
		UserID:       env.TestUserID,
		Title:        title,
		Content:      content,
		ContentType:  "markdown",
		IsPinned:     false,
		IsArchived:   false,
		IsFavorite:   false,
		WordCount:    len(content) / 5, // Rough estimate
		ReadingTime:  len(content) / 1000,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastAccessed: time.Now(),
	}

	_, err := env.DB.Exec(`
		INSERT INTO notes (id, user_id, title, content, content_type,
		                   is_pinned, is_archived, is_favorite,
		                   word_count, reading_time_minutes,
		                   created_at, updated_at, last_accessed)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, note.ID, note.UserID, note.Title, note.Content, note.ContentType,
		note.IsPinned, note.IsArchived, note.IsFavorite,
		note.WordCount, note.ReadingTime,
		note.CreatedAt, note.UpdatedAt, note.LastAccessed)

	if err != nil {
		t.Fatalf("Failed to create test note: %v", err)
	}

	return &TestNote{
		Note: note,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM note_tags WHERE note_id = $1", note.ID)
			env.DB.Exec("DELETE FROM notes WHERE id = $1", note.ID)
		},
	}
}

// TestFolder provides a pre-configured folder for testing
type TestFolder struct {
	Folder  *Folder
	Cleanup func()
}

// createTestFolder creates a folder in the database for testing
func createTestFolder(t *testing.T, env *TestEnvironment, name string) *TestFolder {
	folder := &Folder{
		ID:        uuid.New().String(),
		UserID:    env.TestUserID,
		Name:      name,
		Icon:      "📁",
		Color:     "#6366f1",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := env.DB.Exec(`
		INSERT INTO folders (id, user_id, name, icon, color, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, folder.ID, folder.UserID, folder.Name, folder.Icon, folder.Color,
		folder.Position, folder.CreatedAt, folder.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}

	return &TestFolder{
		Folder: folder,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM folders WHERE id = $1", folder.ID)
		},
	}
}

// TestTag provides a pre-configured tag for testing
type TestTag struct {
	Tag     *Tag
	Cleanup func()
}

// createTestTag creates a tag in the database for testing
func createTestTag(t *testing.T, env *TestEnvironment, name string) *TestTag {
	tag := &Tag{
		ID:         uuid.New().String(),
		Name:       name,
		Color:      "#10b981",
		UsageCount: 0,
	}

	_, err := env.DB.Exec(`
		INSERT INTO tags (id, name, color, usage_count)
		VALUES ($1, $2, $3, $4)
	`, tag.ID, tag.Name, tag.Color, tag.UsageCount)

	if err != nil {
		t.Fatalf("Failed to create test tag: %v", err)
	}

	return &TestTag{
		Tag: tag,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM note_tags WHERE tag_id = $1", tag.ID)
			env.DB.Exec("DELETE FROM tags WHERE id = $1", tag.ID)
		},
	}
}

// waitForCondition polls a condition until it's true or timeout
func waitForCondition(timeout time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
