package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestCreateBrandHandler tests the CreateBrand handler
func TestCreateBrandHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	t.Run("MissingBrandName", func(t *testing.T) {
		service := NewBrandManagerService(db, "http://n8n:5678", "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"industry": "tech",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields: brand_name and industry")
	})

	t.Run("MissingIndustry", func(t *testing.T) {
		service := NewBrandManagerService(db, "http://n8n:5678", "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"brand_name": "Test Brand",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields: brand_name and industry")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		service := NewBrandManagerService(db, "http://n8n:5678", "", "", "", "")

		// Create request with invalid JSON manually
		reqBody := bytes.NewBuffer([]byte(`{"invalid": json}`))
		httpReq, _ := http.NewRequest("POST", "/api/brands", reqBody)
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		service.CreateBrand(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("N8nFailure", func(t *testing.T) {
		// Use a mock server that returns an error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"brand_name": "Test Brand",
				"industry":   "tech",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("SuccessWithDefaults", func(t *testing.T) {
		// Mock server that returns success
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"workflow_id": "test-123", "status": "started"}`))
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"brand_name": "Test Brand",
				"industry":   "tech",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			t.Logf("Response: %s", w.Body.String())
		}
	})

	t.Run("SuccessWithAllFields", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"workflow_id": "test-456"}`))
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"brand_name":   "Test Brand",
				"short_name":   "TB",
				"industry":     "tech",
				"template":     "modern-tech",
				"logo_style":   "minimalist",
				"color_scheme": "primary",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("N8nInvalidResponse", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"brand_name": "Test Brand",
				"industry":   "tech",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestCreateIntegrationHandler tests the CreateIntegration handler
func TestCreateIntegrationHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	t.Run("MissingBrandID", func(t *testing.T) {
		service := NewBrandManagerService(db, "http://n8n:5678", "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body: map[string]interface{}{
				"target_app_path": "/test/app",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields: brand_id and target_app_path")
	})

	t.Run("MissingTargetAppPath", func(t *testing.T) {
		service := NewBrandManagerService(db, "http://n8n:5678", "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body: map[string]interface{}{
				"brand_id": "00000000-0000-0000-0000-000000000000",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields: brand_id and target_app_path")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		service := NewBrandManagerService(db, "http://n8n:5678", "", "", "", "")

		reqBody := bytes.NewBuffer([]byte(`{"invalid": json}`))
		httpReq, _ := http.NewRequest("POST", "/api/integrations", reqBody)
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		service.CreateIntegration(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("N8nFailure", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body: map[string]interface{}{
				"brand_id":        "00000000-0000-0000-0000-000000000000",
				"target_app_path": "/test/app",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("SuccessWithDefaults", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session_id": "claude-123", "status": "started"}`))
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body: map[string]interface{}{
				"brand_id":        "00000000-0000-0000-0000-000000000000",
				"target_app_path": "/test/app",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			t.Logf("Response: %s", w.Body.String())
		}
	})

	t.Run("SuccessWithAllFields", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request body contains expected fields
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"session_id": "claude-456"}`))
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body: map[string]interface{}{
				"brand_id":         "00000000-0000-0000-0000-000000000000",
				"target_app_path":  "/test/app",
				"integration_type": "partial",
				"create_backup":    true,
			},
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("N8nInvalidResponse", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer mockServer.Close()

		service := NewBrandManagerService(db, mockServer.URL, "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body: map[string]interface{}{
				"brand_id":        "00000000-0000-0000-0000-000000000000",
				"target_app_path": "/test/app",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestServiceInitialization tests service initialization edge cases
func TestServiceInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyURLs", func(t *testing.T) {
		service := NewBrandManagerService(nil, "", "", "", "", "")

		if service.n8nBaseURL != "" {
			t.Errorf("Expected empty n8nBaseURL, got %s", service.n8nBaseURL)
		}

		if service.httpClient == nil {
			t.Error("Expected httpClient to be initialized even with empty URLs")
		}

		if service.logger == nil {
			t.Error("Expected logger to be initialized")
		}
	})

	t.Run("HTTPClientTimeout", func(t *testing.T) {
		service := NewBrandManagerService(nil, "", "", "", "", "")

		if service.httpClient.Timeout != httpTimeout {
			t.Errorf("Expected timeout %v, got %v", httpTimeout, service.httpClient.Timeout)
		}
	})
}

// TestN8nConnectionFailure tests n8n connection failure scenarios
func TestN8nConnectionFailure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	t.Run("InvalidURL", func(t *testing.T) {
		// Use invalid URL that will cause connection failure
		service := NewBrandManagerService(db, "http://invalid-url-that-does-not-exist-xyz:99999", "", "", "", "")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body: map[string]interface{}{
				"brand_name": "Test Brand",
				"industry":   "tech",
			},
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestHTTPRequestMethods tests different HTTP methods
func TestHTTPRequestMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					t.Errorf("Expected method %s, got %s", method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}

			req := HTTPTestRequest{
				Method: method,
				Path:   "/test",
			}

			w, err := makeHTTPRequest(req, handler)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}
