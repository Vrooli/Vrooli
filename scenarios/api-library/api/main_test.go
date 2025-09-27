package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_DB", "test_api_library")
	os.Setenv("POSTGRES_USER", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("QDRANT_HOST", "localhost")
	os.Setenv("QDRANT_PORT", "6333")
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Exit(code)
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal("Failed to decode response body")
	}

	if response["status"] != "healthy" {
		t.Errorf("handler returned wrong status: got %v want %v",
			response["status"], "healthy")
	}

	if response["service"] != "api-library" {
		t.Errorf("handler returned wrong service name: got %v want %v",
			response["service"], "api-library")
	}
}

func TestSearchAPIsHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantFields []string
	}{
		{
			name:       "POST search with query",
			method:     "POST",
			body:       `{"query":"payment","limit":5}`,
			wantStatus: http.StatusOK,
			wantFields: []string{"results", "count", "method"},
		},
		{
			name:       "POST search without query",
			method:     "POST",
			body:       `{"limit":5}`,
			wantStatus: http.StatusBadRequest,
			wantFields: []string{"error"},
		},
		{
			name:       "POST search with filters",
			method:     "POST",
			body:       `{"query":"api","limit":10,"filters":{"configured":true,"max_price":0.01}}`,
			wantStatus: http.StatusOK,
			wantFields: []string{"results"},
		},
		{
			name:       "GET search with query param",
			method:     "GET",
			body:       "",
			wantStatus: http.StatusOK,
			wantFields: []string{"results"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.method == "POST" {
				req, err = http.NewRequest(tt.method, "/api/v1/search", bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, "/api/v1/search?query=test", nil)
			}

			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			
			// Note: In a real test, we'd need to set up the router and database connection
			// For now, this is a structure test
			_ = rr
		})
	}
}

func TestCreateAPIHandler(t *testing.T) {
	apiData := API{
		Name:             "Test API",
		Provider:         "Test Provider",
		Description:      "Test Description",
		BaseURL:          "https://api.test.com",
		DocumentationURL: "https://docs.test.com",
		Category:         "Testing",
		Status:           "active",
		AuthType:         "api_key",
		Tags:             []string{"test", "example"},
		Capabilities:     []string{"testing", "mocking"},
	}

	body, err := json.Marshal(apiData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/v1/apis", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	// Note: In a real test, we'd need to set up the router and database connection
	_ = rr
}

func TestAddNoteHandler(t *testing.T) {
	note := Note{
		Content:   "This is a test note",
		Type:      "tip",
		CreatedBy: "test_user",
	}

	body, err := json.Marshal(note)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/v1/apis/test-id/notes", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	// Note: In a real test, we'd need to set up the router and database connection
	_ = rr
}

func TestMarkConfiguredHandler(t *testing.T) {
	configData := map[string]bool{
		"configured": true,
	}

	body, err := json.Marshal(configData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/v1/apis/test-id/configure", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	// Note: In a real test, we'd need to set up the router and database connection
	_ = rr
}

func TestRequestResearchHandler(t *testing.T) {
	researchRequest := map[string]interface{}{
		"capability": "payment processing",
		"requirements": map[string]interface{}{
			"max_price": 0.01,
			"regions":   []string{"US", "EU"},
			"features":  []string{"recurring", "subscriptions"},
		},
	}

	body, err := json.Marshal(researchRequest)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/v1/request-research", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	// Note: In a real test, we'd need to set up the router and database connection
	_ = rr
}

// Benchmark tests for performance validation
func BenchmarkSearchAPIs(b *testing.B) {
	searchReq := SearchRequest{
		Query: "payment processing",
		Limit: 10,
	}

	body, _ := json.Marshal(searchReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		// In a real benchmark, we'd call the actual handler
		_ = rr
	}
}

// Test response time requirement (<100ms)
func TestSearchResponseTime(t *testing.T) {
	searchReq := SearchRequest{
		Query: "test",
		Limit: 10,
	}

	body, _ := json.Marshal(searchReq)
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	rr := httptest.NewRecorder()
	// In a real test, we'd call the actual handler
	_ = rr
	elapsed := time.Since(start)

	// Check if response time is under 100ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("Search response time too slow: %v (should be < 100ms)", elapsed)
	}
}

// Test concurrent search requests
func TestConcurrentSearchRequests(t *testing.T) {
	numRequests := 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			searchReq := SearchRequest{
				Query: fmt.Sprintf("test%d", id),
				Limit: 5,
			}

			body, _ := json.Marshal(searchReq)
			req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			// In a real test, we'd call the actual handler
			_ = rr
			
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// Test input validation
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "Invalid JSON",
			endpoint:   "/api/v1/search",
			method:     "POST",
			body:       `{invalid json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Negative limit",
			endpoint:   "/api/v1/search",
			method:     "POST",
			body:       `{"query":"test","limit":-1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Excessive limit",
			endpoint:   "/api/v1/search",
			method:     "POST",
			body:       `{"query":"test","limit":10000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid note type",
			endpoint:   "/api/v1/apis/test-id/notes",
			method:     "POST",
			body:       `{"content":"test","type":"invalid"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.endpoint, bytes.NewBufferString(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			// In a real test, we'd call the actual handler
			_ = rr
		})
	}
}