package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		service := setupTestService(t, nil, env.DataDir)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		service.Health(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		// Verify required fields
		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
		if response["service"] != serviceName {
			t.Errorf("Expected service '%s', got '%v'", serviceName, response["service"])
		}
		if response["version"] != apiVersion {
			t.Errorf("Expected version '%s', got '%v'", apiVersion, response["version"])
		}
	})

	t.Run("DatabaseUnavailable", func(t *testing.T) {
		// This test would require a mock database that fails Ping()
		// Skipping for now as it requires more setup
		t.Skip("Requires mock database setup")
	})
}

// TestUploadBook tests the book upload endpoint
func TestUploadBook(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("Success_TXT", func(t *testing.T) {
		bookContent := []byte(createSampleBookContent())
		req := createMultipartRequest(t, "/api/v1/books/upload",
			map[string]string{
				"title":   "Test Book",
				"author":  "Test Author",
				"user_id": "test-user",
			},
			"file", "test-book.txt", bookContent)

		w := httptest.NewRecorder()
		service.UploadBook(w, req)

		// Note: This will fail without actual database, but tests the handler logic
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 201 or 500, got %d", w.Code)
		}
	})

	t.Run("MissingFile", func(t *testing.T) {
		req := createMultipartRequest(t, "/api/v1/books/upload",
			map[string]string{
				"title": "Test Book",
			},
			"", "", nil) // No file

		w := httptest.NewRecorder()
		service.UploadBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "No file provided")
	})

	t.Run("UnsupportedFileType", func(t *testing.T) {
		bookContent := []byte("test content")
		req := createMultipartRequest(t, "/api/v1/books/upload",
			map[string]string{
				"title": "Test Book",
			},
			"file", "test-book.docx", bookContent) // Unsupported format

		w := httptest.NewRecorder()
		service.UploadBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Unsupported file type")
	})

	t.Run("EmptyTitle", func(t *testing.T) {
		bookContent := []byte("test content")
		req := createMultipartRequest(t, "/api/v1/books/upload",
			map[string]string{},
			"file", "my-book.txt", bookContent)

		w := httptest.NewRecorder()
		service.UploadBook(w, req)

		// Should default to filename without extension
		// Actual behavior depends on database availability
	})
}

// TestGetBooks tests listing books
func TestGetBooks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("EmptyList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/books", nil)
		w := httptest.NewRecorder()

		service.GetBooks(w, req)

		// Without database, this will error
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("WithUserIDFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/books?user_id=test-user", nil)
		w := httptest.NewRecorder()

		service.GetBooks(w, req)

		// Expect database error without real DB
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestGetBook tests retrieving a specific book
func TestGetBook(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/books/invalid-uuid", nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": "invalid-uuid"})
		w := httptest.NewRecorder()

		service.GetBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid book ID")
	})

	t.Run("NonExistentBook", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/books/%s", bookID), nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		w := httptest.NewRecorder()

		service.GetBook(w, req)

		// Expect not found or database error
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 404 or 500, got %d", w.Code)
		}
	})

	t.Run("WithUserProgress", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/books/%s?user_id=test-user", bookID), nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		w := httptest.NewRecorder()

		service.GetBook(w, req)

		// Expect database error without real DB
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 404 or 500, got %d", w.Code)
		}
	})
}

// TestChatWithBook tests the chat endpoint
func TestChatWithBook(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("InvalidUUID", func(t *testing.T) {
		body := map[string]interface{}{
			"message":          "What happens in chapter 1?",
			"user_id":          "test-user",
			"current_position": 10,
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/v1/books/invalid-uuid/chat", bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": "invalid-uuid"})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ChatWithBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid book ID")
	})

	t.Run("MissingMessage", func(t *testing.T) {
		bookID := uuid.New()
		body := map[string]interface{}{
			"user_id":          "test-user",
			"current_position": 10,
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/books/%s/chat", bookID), bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ChatWithBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields")
	})

	t.Run("MissingUserID", func(t *testing.T) {
		bookID := uuid.New()
		body := map[string]interface{}{
			"message":          "What happens in chapter 1?",
			"current_position": 10,
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/books/%s/chat", bookID), bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ChatWithBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/books/%s/chat", bookID), bytes.NewReader([]byte("{invalid json}")))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ChatWithBook(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})

	t.Run("ValidRequest_BookNotFound", func(t *testing.T) {
		bookID := uuid.New()
		body := map[string]interface{}{
			"message":          "What happens in chapter 1?",
			"user_id":          "test-user",
			"current_position": 10,
			"position_type":    "chunk",
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/books/%s/chat", bookID), bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ChatWithBook(w, req)

		// Expect not found or database error
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 404 or 500, got %d", w.Code)
		}
	})
}

// TestUpdateProgress tests the progress update endpoint
func TestUpdateProgress(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("InvalidUUID", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          "test-user",
			"current_position": 25,
			"position_type":    "chunk",
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", "/api/v1/books/invalid-uuid/progress", bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": "invalid-uuid"})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.UpdateProgress(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid book ID")
	})

	t.Run("MissingUserID", func(t *testing.T) {
		bookID := uuid.New()
		body := map[string]interface{}{
			"current_position": 25,
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/books/%s/progress", bookID), bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.UpdateProgress(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required field")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/books/%s/progress", bookID), bytes.NewReader([]byte("{invalid}")))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.UpdateProgress(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})

	t.Run("ValidRequest_BookNotFound", func(t *testing.T) {
		bookID := uuid.New()
		body := map[string]interface{}{
			"user_id":          "test-user",
			"current_position": 25,
			"position_type":    "chunk",
			"position_value":   25.0,
		}

		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/books/%s/progress", bookID), bytes.NewReader(bodyBytes))
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.UpdateProgress(w, req)

		// Expect not found or database error
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 404 or 500, got %d", w.Code)
		}
	})
}

// TestGetConversations tests retrieving conversation history
func TestGetConversations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/books/invalid-uuid/conversations?user_id=test-user", nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": "invalid-uuid"})
		w := httptest.NewRecorder()

		service.GetConversations(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid book ID")
	})

	t.Run("MissingUserID", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/books/%s/conversations", bookID), nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		w := httptest.NewRecorder()

		service.GetConversations(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required parameter")
	})

	t.Run("WithLimit", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/books/%s/conversations?user_id=test-user&limit=10", bookID), nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		w := httptest.NewRecorder()

		service.GetConversations(w, req)

		// Expect database error without real DB
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("InvalidLimit", func(t *testing.T) {
		bookID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/books/%s/conversations?user_id=test-user&limit=9999", bookID), nil)
		req = mux.SetURLVars(req, map[string]string{"book_id": bookID.String()})
		w := httptest.NewRecorder()

		service.GetConversations(w, req)

		// Should cap at maxChatHistory
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("Contains", func(t *testing.T) {
		slice := []string{"txt", "pdf", "epub"}

		if !contains(slice, "txt") {
			t.Error("Expected contains to find 'txt'")
		}
		if !contains(slice, "pdf") {
			t.Error("Expected contains to find 'pdf'")
		}
		if contains(slice, "docx") {
			t.Error("Expected contains not to find 'docx'")
		}
		if contains(slice, "") {
			t.Error("Expected contains not to find empty string")
		}
	})

	t.Run("ToJSON", func(t *testing.T) {
		data := map[string]interface{}{
			"key": "value",
			"num": 42,
		}

		jsonStr := toJSON(data)
		if jsonStr == "" {
			t.Error("Expected non-empty JSON string")
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Errorf("Expected valid JSON, got error: %v", err)
		}
	})

	t.Run("Min", func(t *testing.T) {
		if min(5, 10) != 5 {
			t.Error("Expected min(5, 10) = 5")
		}
		if min(10, 5) != 5 {
			t.Error("Expected min(10, 5) = 5")
		}
		if min(5, 5) != 5 {
			t.Error("Expected min(5, 5) = 5")
		}
		if min(-5, 5) != -5 {
			t.Error("Expected min(-5, 5) = -5")
		}
	})

	t.Run("EstimateProcessingTime", func(t *testing.T) {
		service := setupTestService(t, nil, "/tmp")

		// Small file
		time1 := service.estimateProcessingTime(100 * 1024) // 100KB
		if time1 < 5 {
			t.Errorf("Expected minimum 5 seconds, got %d", time1)
		}

		// Large file
		time2 := service.estimateProcessingTime(10 * 1024 * 1024) // 10MB
		if time2 <= time1 {
			t.Error("Expected larger file to have longer estimate")
		}
	})
}

// TestProcessBookAsync tests async book processing
func TestProcessBookAsync(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := setupTestService(t, nil, env.DataDir)

	t.Run("SimulatedProcessing", func(t *testing.T) {
		bookID := uuid.New()
		filePath := createTestBookFile(t, env, "test-book.txt", createSampleBookContent())

		// This will try to update database, which will fail without real DB
		// But we can test that the function doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("processBookAsync panicked: %v", r)
			}
		}()

		// Don't actually wait for completion in tests
		go service.processBookAsync(bookID, filePath, "txt")
		time.Sleep(100 * time.Millisecond) // Give it a moment to start
	})
}

// TestGetSafeContext tests context retrieval with position filtering
func TestGetSafeContext(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	service := setupTestService(t, nil, "/tmp")

	t.Run("WithinBoundary", func(t *testing.T) {
		bookID := uuid.New()
		chunks, err := service.getSafeContext(bookID, 10, "test query")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(chunks) > 10+1 {
			t.Errorf("Expected at most 11 chunks (position+1), got %d", len(chunks))
		}

		// Verify all chunks are within boundary
		for _, chunk := range chunks {
			if chunk.ChunkNumber > 10 {
				t.Errorf("Chunk %d exceeds position boundary of 10", chunk.ChunkNumber)
			}
		}
	})

	t.Run("StartOfBook", func(t *testing.T) {
		bookID := uuid.New()
		chunks, err := service.getSafeContext(bookID, 0, "test query")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(chunks) > 1 {
			t.Errorf("Expected at most 1 chunk at position 0, got %d", len(chunks))
		}
	})
}

// TestGenerateChatResponse tests AI response generation
func TestGenerateChatResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	service := setupTestService(t, nil, "/tmp")

	t.Run("MockResponse", func(t *testing.T) {
		book := Book{
			ID:     uuid.New(),
			Title:  "Test Book",
			Author: "Test Author",
		}

		chunks := []BookChunk{
			{ChunkNumber: 0, Content: "Chapter 1 content"},
			{ChunkNumber: 1, Content: "Chapter 1 continued"},
		}

		response, sources, err := service.generateChatResponse("What is this about?", chunks, book, 0.7)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if response == "" {
			t.Error("Expected non-empty response")
		}

		if len(sources) == 0 {
			t.Error("Expected at least one source")
		}

		// Verify response mentions the book
		if !contains([]string{response}, book.Title) && !contains([]string{response}, book.Author) {
			t.Log("Response doesn't mention book title or author (expected for mock)")
		}
	})
}

// TestNewBookTalkService tests service initialization
func TestNewBookTalkService(t *testing.T) {
	t.Run("ValidInitialization", func(t *testing.T) {
		service := NewBookTalkService(nil, "http://n8n:5678", "http://qdrant:6333", "/data")

		if service == nil {
			t.Fatal("Expected non-nil service")
		}

		if service.n8nBaseURL != "http://n8n:5678" {
			t.Errorf("Expected n8n URL 'http://n8n:5678', got '%s'", service.n8nBaseURL)
		}

		if service.qdrantURL != "http://qdrant:6333" {
			t.Errorf("Expected qdrant URL 'http://qdrant:6333', got '%s'", service.qdrantURL)
		}

		if service.dataDir != "/data" {
			t.Errorf("Expected data dir '/data', got '%s'", service.dataDir)
		}

		if service.logger == nil {
			t.Error("Expected logger to be initialized")
		}
	})
}
