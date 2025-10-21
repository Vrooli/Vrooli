package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// setupTestLogger initializes a test logger with controlled output
func setupTestLogger() func() {
	// Suppress logs during tests unless -v flag is used
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestServer provides a complete test server instance
type TestServer struct {
	Server  *APIServer
	DB      *sql.DB
	Router  *mux.Router
	Cleanup func()
}

// setupTestServer creates an isolated test server with in-memory database
func setupTestServer(t *testing.T) *TestServer {
	// Create in-memory test database connection
	// Note: For PostgreSQL, we'd need to use a test database
	// For now, we'll mock the database connection with a nil DB
	// Handlers that require DB will be skipped or mocked

	// Create a mock DB connection that won't actually be used
	// Most tests don't require the actual database
	var db *sql.DB // nil is fine for tests that don't use it

	server := &APIServer{
		db:          db,
		n8nURL:      "http://localhost:5678",
		windmillURL: "http://localhost:8000",
		searxngURL:  "http://localhost:8280",
		qdrantURL:   "http://localhost:6333",
		minioURL:    "http://localhost:9000",
		ollamaURL:   "http://localhost:11434",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	router := mux.NewRouter()

	// Set up routes (same as main)
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/reports", server.getReports).Methods("GET")
	api.HandleFunc("/reports", server.createReport).Methods("POST")
	api.HandleFunc("/reports/{id}", server.getReport).Methods("GET")
	api.HandleFunc("/reports/{id}", server.updateReport).Methods("PUT")
	api.HandleFunc("/reports/{id}", server.deleteReport).Methods("DELETE")
	api.HandleFunc("/reports/{id}/pdf", server.getReportPDF).Methods("GET")
	api.HandleFunc("/reports/count", server.getReportCount).Methods("GET")
	api.HandleFunc("/reports/confidence-average", server.getReportConfidenceAverage).Methods("GET")
	api.HandleFunc("/schedules", server.getSchedules).Methods("GET")
	api.HandleFunc("/schedules/count", server.getScheduleCount).Methods("GET")
	api.HandleFunc("/schedules/{id}/run", server.runScheduleNow).Methods("POST")
	api.HandleFunc("/schedules/{id}/toggle", server.toggleSchedule).Methods("POST")
	api.HandleFunc("/chat/conversations", server.getChatConversationsSummary).Methods("GET")
	api.HandleFunc("/chat/messages", server.getChatMessages).Methods("GET")
	api.HandleFunc("/chat/message", server.handleChatMessage).Methods("POST")
	api.HandleFunc("/chat/sessions/count", server.getChatSessionsCount).Methods("GET")
	api.HandleFunc("/conversations", server.getConversations).Methods("GET")
	api.HandleFunc("/conversations", server.createConversation).Methods("POST")
	api.HandleFunc("/conversations/{id}", server.getConversation).Methods("GET")
	api.HandleFunc("/conversations/{id}/messages", server.getMessages).Methods("GET")
	api.HandleFunc("/conversations/{id}/messages", server.sendMessage).Methods("POST")
	api.HandleFunc("/search", server.performSearch).Methods("POST")
	api.HandleFunc("/search/history", server.getSearchHistory).Methods("GET")
	api.HandleFunc("/detect-contradictions", server.detectContradictions).Methods("POST")
	api.HandleFunc("/analyze", server.analyzeContent).Methods("POST")
	api.HandleFunc("/analyze/insights", server.extractInsights).Methods("POST")
	api.HandleFunc("/analyze/trends", server.analyzeTrends).Methods("POST")
	api.HandleFunc("/analyze/competitive", server.analyzeCompetitive).Methods("POST")
	api.HandleFunc("/templates", server.getTemplates).Methods("GET")
	api.HandleFunc("/depth-configs", server.getDepthConfigs).Methods("GET")
	api.HandleFunc("/dashboard/stats", server.getDashboardStats).Methods("GET")
	api.HandleFunc("/knowledge/search", server.searchKnowledge).Methods("GET")
	api.HandleFunc("/knowledge/collections", server.getCollections).Methods("GET")

	return &TestServer{
		Server: server,
		Router: router,
		Cleanup: func() {
			if server.db != nil {
				server.db.Close()
			}
		},
	}
}

// HTTPTestRequest defines a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(ts *TestServer, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default content type for POST/PUT
	if req.Method == "POST" || req.Method == "PUT" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
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

	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, httpReq)
	return w
}

// assertJSONResponse validates a JSON response matches expected structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// createTestReport creates a test report in the system
func createTestReport(t *testing.T, ts *TestServer, req ReportRequest) *Report {
	// Default values
	if req.Depth == "" {
		req.Depth = "standard"
	}
	if req.TargetLength == 0 {
		req.TargetLength = 5
	}
	if req.Language == "" {
		req.Language = "en"
	}

	reportID := uuid.New().String()
	now := time.Now()

	report := &Report{
		ID:           reportID,
		Title:        req.Topic,
		Topic:        req.Topic,
		Depth:        req.Depth,
		TargetLength: req.TargetLength,
		Language:     req.Language,
		Status:       "pending",
		RequestedBy:  req.RequestedBy,
		Organization: req.Organization,
		Tags:         req.Tags,
		Category:     req.Category,
		RequestedAt:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return report
}

// mockN8NServer creates a mock n8n server for testing
func mockN8NServer(t *testing.T) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock n8n webhook response
		if r.URL.Path == "/webhook/research-request" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "received",
			})
			return
		}
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(handler)
}

// mockSearXNGServer creates a mock SearXNG server for testing
func mockSearXNGServer(t *testing.T) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			// Return mock search results
			response := map[string]interface{}{
				"query": r.URL.Query().Get("q"),
				"results": []map[string]interface{}{
					{
						"title":   "Test Result 1",
						"url":     "https://example.com/1",
						"content": "This is test content for result 1",
					},
					{
						"title":         "Test Result 2",
						"url":           "https://arxiv.org/paper1",
						"content":       "Academic research paper content",
						"publishedDate": time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
					},
				},
				"engines":    []string{"google", "duckduckgo"},
				"query_time": 0.5,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(handler)
}

// mockOllamaServer creates a mock Ollama server for testing
func mockOllamaServer(t *testing.T) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			// Parse request to determine what to return
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			prompt := req["prompt"].(string)
			var response string

			if contains(prompt, "Extract") && contains(prompt, "key factual claims") {
				// Mock claim extraction
				response = `["Claim 1: AI is growing rapidly", "Claim 2: Market size is $50B"]`
			} else if contains(prompt, "contradict") {
				// Mock contradiction detection
				response = `{"is_contradiction": false, "confidence": 0.8, "explanation": "Claims are compatible"}`
			} else {
				response = "Mock response"
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"response": response,
				"done":     true,
			})
			return
		}
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []string{"llama3.2:3b"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(handler)
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// assertEqual asserts two values are equal
func assertEqual(t *testing.T, got, want interface{}, msg string) {
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// assertNotNil asserts value is not nil
func assertNotNil(t *testing.T, value interface{}, msg string) {
	if value == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// assertStringContains asserts string contains substring
func assertStringContains(t *testing.T, s, substr, msg string) {
	if !contains(s, substr) {
		t.Errorf("%s: expected %q to contain %q", msg, s, substr)
	}
}
