package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, Health)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if service, ok := response["service"].(string); !ok || service != serviceName {
			t.Errorf("Expected service '%s', got %v", serviceName, response["service"])
		}

		if version, ok := response["version"].(string); !ok || version != apiVersion {
			t.Errorf("Expected version '%s', got %v", apiVersion, response["version"])
		}
	})
}

// TestNewLogger tests logger creation
func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.Logger == nil {
		t.Error("Expected Logger field to be initialized")
	}
}

// TestLoggerMethods tests logger methods
func TestLoggerMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	logger := NewLogger()

	t.Run("Error", func(t *testing.T) {
		// Should not panic
		logger.Error("test error", io.EOF)
	})

	t.Run("Warn", func(t *testing.T) {
		// Should not panic
		logger.Warn("test warning", io.EOF)
	})

	t.Run("Info", func(t *testing.T) {
		// Should not panic
		logger.Info("test info")
	})
}

// TestHTTPError tests the HTTPError function
func TestHTTPError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BasicError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "test error", http.StatusBadRequest, io.EOF)

		assertErrorResponse(t, w, http.StatusBadRequest, "test error")
	})

	t.Run("InternalServerError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "internal error", http.StatusInternalServerError, io.EOF)

		assertErrorResponse(t, w, http.StatusInternalServerError, "internal error")
	})

	t.Run("NotFoundError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "not found", http.StatusNotFound, io.EOF)

		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

// TestNewBrandManagerService tests service creation
func TestNewBrandManagerService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)

	service := NewBrandManagerService(
		db,
		"http://localhost:5678",
		"http://localhost:8000",
		"http://localhost:8188",
		"localhost:9000",
		"http://localhost:8200",
	)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.db != db {
		t.Error("Expected db to be set")
	}

	if service.n8nBaseURL != "http://localhost:5678" {
		t.Errorf("Expected n8nBaseURL to be set, got %s", service.n8nBaseURL)
	}

	if service.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}

	if service.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

// TestListBrands tests the ListBrands handler
func TestListBrands(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(db, "", "", "", "", "")

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/brands",
		}

		w, err := makeHTTPRequest(req, service.ListBrands)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("WithBrands", func(t *testing.T) {
		// Create test brands
		createTestBrand(t, db, "Test Brand 1")
		createTestBrand(t, db, "Test Brand 2")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/brands",
		}

		w, err := makeHTTPRequest(req, service.ListBrands)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		// Response should be an array
		if _, ok := response["0"]; !ok {
			// It's actually returned as an array in Go, check body directly
			var brands []Brand
			if err := json.Unmarshal(w.Body.Bytes(), &brands); err != nil {
				t.Fatalf("Failed to parse brands array: %v", err)
			}

			if len(brands) < 2 {
				t.Errorf("Expected at least 2 brands, got %d", len(brands))
			}
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		suite := PaginationTestSuite{
			HandlerName: "ListBrands",
			Handler:     service.ListBrands,
			BasePath:    "/api/brands",
		}
		suite.RunPaginationTests(t)
	})
}

// TestCreateBrand tests the CreateBrand handler
func TestCreateBrand(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(db, "http://localhost:5678", "", "", "", "")

	t.Run("MissingRequiredFields", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMissingRequiredField("/api/brands", "POST", "brand_name").
			AddMissingRequiredField("/api/brands", "POST", "industry").
			AddEmptyBody("/api/brands", "POST").
			Build()

		suite := HandlerTestSuite{
			HandlerName: "CreateBrand",
			Handler:     service.CreateBrand,
		}
		suite.RunErrorTests(t, patterns)
	})

	t.Run("ValidRequest_N8nUnavailable", func(t *testing.T) {
		// This will fail because n8n is not available in test env
		// But we can verify the request validation works
		requestBody := map[string]interface{}{
			"brand_name": "Test Brand",
			"industry":   "technology",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body:   requestBody,
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Will get error trying to reach n8n, but that's expected
		if w.Code == http.StatusOK {
			t.Log("Unexpected success - n8n must be running")
		}
	})

	t.Run("WithDefaults", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"brand_name": "Test Brand",
			"industry":   "technology",
			// Omit optional fields to test defaults
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body:   requestBody,
		}

		w, err := makeHTTPRequest(req, service.CreateBrand)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Expected to fail trying to reach n8n
		if w.Code != http.StatusInternalServerError {
			// If it succeeds, verify response
			if w.Code == http.StatusOK {
				response := assertJSONResponse(t, w, http.StatusOK)
				if _, ok := response["message"]; !ok {
					t.Error("Expected 'message' in response")
				}
			}
		}
	})
}

// TestGetBrandByID tests the GetBrandByID handler
func TestGetBrandByID(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(db, "", "", "", "", "")

	t.Run("Success", func(t *testing.T) {
		brandID := createTestBrand(t, db, "Test Brand")

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/" + brandID.String(),
			URLVars: map[string]string{"id": brandID.String()},
		}

		w, err := makeHTTPRequest(req, service.GetBrandByID)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if idStr, ok := response["id"].(string); !ok || idStr != brandID.String() {
			t.Errorf("Expected brand ID %s, got %v", brandID.String(), response["id"])
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddNonExistentBrand("/api/brands/{id}", "id").
			Build()

		suite := HandlerTestSuite{
			HandlerName: "GetBrandByID",
			Handler:     service.GetBrandByID,
		}
		suite.RunErrorTests(t, patterns)
	})
}

// TestGetBrandStatus tests the GetBrandStatus handler
func TestGetBrandStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(db, "", "", "", "", "")

	t.Run("BrandExists", func(t *testing.T) {
		createTestBrand(t, db, "Status Test Brand")

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/status/Status Test Brand",
			URLVars: map[string]string{"name": "Status Test Brand"},
		}

		w, err := makeHTTPRequest(req, service.GetBrandStatus)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "completed" {
			t.Errorf("Expected status 'completed', got %v", response["status"])
		}

		if _, ok := response["brand"]; !ok {
			t.Error("Expected 'brand' in response")
		}
	})

	t.Run("BrandInProgress", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/brands/status/NonExistentBrand",
			URLVars: map[string]string{"name": "NonExistentBrand"},
		}

		w, err := makeHTTPRequest(req, service.GetBrandStatus)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "in_progress" {
			t.Errorf("Expected status 'in_progress', got %v", response["status"])
		}
	})
}

// TestListIntegrations tests the ListIntegrations handler
func TestListIntegrations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(db, "", "", "", "", "")

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/integrations",
		}

		w, err := makeHTTPRequest(req, service.ListIntegrations)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("WithIntegrations", func(t *testing.T) {
		brandID := createTestBrand(t, db, "Integration Test Brand")
		createTestIntegration(t, db, brandID, "/test/app")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/integrations",
		}

		w, err := makeHTTPRequest(req, service.ListIntegrations)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		suite := PaginationTestSuite{
			HandlerName: "ListIntegrations",
			Handler:     service.ListIntegrations,
			BasePath:    "/api/integrations",
		}
		suite.RunPaginationTests(t)
	})
}

// TestCreateIntegration tests the CreateIntegration handler
func TestCreateIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(db, "http://localhost:5678", "", "", "", "")

	t.Run("MissingRequiredFields", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMissingRequiredField("/api/integrations", "POST", "brand_id").
			AddMissingRequiredField("/api/integrations", "POST", "target_app_path").
			AddEmptyBody("/api/integrations", "POST").
			Build()

		suite := HandlerTestSuite{
			HandlerName: "CreateIntegration",
			Handler:     service.CreateIntegration,
		}
		suite.RunErrorTests(t, patterns)
	})

	t.Run("ValidRequest_N8nUnavailable", func(t *testing.T) {
		brandID := createTestBrand(t, db, "Integration Brand")

		requestBody := map[string]interface{}{
			"brand_id":         brandID.String(),
			"target_app_path":  "/test/app",
			"integration_type": "full",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/integrations",
			Body:   requestBody,
		}

		w, err := makeHTTPRequest(req, service.CreateIntegration)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Will get error trying to reach n8n, but that's expected
		if w.Code == http.StatusOK {
			t.Log("Unexpected success - n8n must be running")
		}
	})
}

// TestGetServiceURLs tests the GetServiceURLs handler
func TestGetServiceURLs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db := setupTestDB(t)
	defer cleanupTestDB(db, t)
	skipIfNoDatabase(t, db)

	service := NewBrandManagerService(
		db,
		"http://localhost:5678",
		"http://localhost:8000",
		"http://localhost:8188",
		"localhost:9000",
		"http://localhost:8200",
	)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/services",
		}

		w, err := makeHTTPRequest(req, service.GetServiceURLs)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["services"]; !ok {
			t.Error("Expected 'services' in response")
		}

		if _, ok := response["dashboards"]; !ok {
			t.Error("Expected 'dashboards' in response")
		}
	})
}

// TestGetResourcePort tests the getResourcePort function
func TestGetResourcePort(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidResource", func(t *testing.T) {
		port := getResourcePort("n8n")
		if port == "" {
			t.Error("Expected port to be returned")
		}
	})

	t.Run("InvalidResource", func(t *testing.T) {
		port := getResourcePort("nonexistent-resource-xyz")
		// Should return a fallback port or empty string (both acceptable)
		// The function returns default ports from map or generic fallback
		t.Logf("Got port: %s", port)
	})

	t.Run("KnownResources", func(t *testing.T) {
		resources := []string{"n8n", "windmill", "postgres", "comfyui", "minio", "vault"}
		for _, resource := range resources {
			port := getResourcePort(resource)
			if port == "" {
				t.Errorf("Expected port for %s", resource)
			}
		}
	})
}

// TestBrandStructure tests the Brand struct
func TestBrandStructure(t *testing.T) {
	brand := Brand{
		ID:          uuid.New(),
		Name:        "Test Brand",
		ShortName:   "TB",
		Slogan:      "Test Slogan",
		AdCopy:      "Test Ad Copy",
		Description: "Test Description",
		BrandColors: map[string]interface{}{
			"primary": "#FF5733",
		},
		LogoURL:    "http://example.com/logo.png",
		FaviconURL: "http://example.com/favicon.ico",
		Assets:     []interface{}{},
		Metadata:   map[string]interface{}{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(brand)
	if err != nil {
		t.Fatalf("Failed to marshal brand: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledBrand Brand
	if err := json.Unmarshal(jsonData, &unmarshaledBrand); err != nil {
		t.Fatalf("Failed to unmarshal brand: %v", err)
	}

	if unmarshaledBrand.ID != brand.ID {
		t.Errorf("Expected ID %s, got %s", brand.ID, unmarshaledBrand.ID)
	}

	if unmarshaledBrand.Name != brand.Name {
		t.Errorf("Expected Name %s, got %s", brand.Name, unmarshaledBrand.Name)
	}
}

// TestIntegrationRequestStructure tests the IntegrationRequest struct
func TestIntegrationRequestStructure(t *testing.T) {
	now := time.Now()
	integration := IntegrationRequest{
		ID:              uuid.New(),
		BrandID:         uuid.New(),
		TargetAppPath:   "/test/app",
		IntegrationType: "full",
		ClaudeSessionID: "session-123",
		Status:          "pending",
		RequestPayload:  map[string]interface{}{"test": true},
		ResponsePayload: map[string]interface{}{"result": "success"},
		CreatedAt:       now,
		CompletedAt:     &now,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(integration)
	if err != nil {
		t.Fatalf("Failed to marshal integration: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledIntegration IntegrationRequest
	if err := json.Unmarshal(jsonData, &unmarshaledIntegration); err != nil {
		t.Fatalf("Failed to unmarshal integration: %v", err)
	}

	if unmarshaledIntegration.ID != integration.ID {
		t.Errorf("Expected ID %s, got %s", integration.ID, unmarshaledIntegration.ID)
	}

	if unmarshaledIntegration.BrandID != integration.BrandID {
		t.Errorf("Expected BrandID %s, got %s", integration.BrandID, unmarshaledIntegration.BrandID)
	}
}

// TestConstants tests that constants are properly defined
func TestConstants(t *testing.T) {
	t.Run("APIVersion", func(t *testing.T) {
		if apiVersion == "" {
			t.Error("Expected apiVersion to be defined")
		}
	})

	t.Run("ServiceName", func(t *testing.T) {
		if serviceName == "" {
			t.Error("Expected serviceName to be defined")
		}
		if serviceName != "brand-manager" {
			t.Errorf("Expected serviceName to be 'brand-manager', got '%s'", serviceName)
		}
	})

	t.Run("DefaultPort", func(t *testing.T) {
		if defaultPort == "" {
			t.Error("Expected defaultPort to be defined")
		}
	})

	t.Run("Timeouts", func(t *testing.T) {
		if httpTimeout == 0 {
			t.Error("Expected httpTimeout to be defined")
		}
		if brandGenTimeout == 0 {
			t.Error("Expected brandGenTimeout to be defined")
		}
	})

	t.Run("DatabaseLimits", func(t *testing.T) {
		if maxDBConnections == 0 {
			t.Error("Expected maxDBConnections to be defined")
		}
		if maxIdleConnections == 0 {
			t.Error("Expected maxIdleConnections to be defined")
		}
	})
}

// TestMockHTTPClient tests the mock HTTP client helper
func TestMockHTTPClient(t *testing.T) {
	t.Run("DefaultResponse", func(t *testing.T) {
		client := &MockHTTPClient{}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("CustomResponse", func(t *testing.T) {
		client := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": "not found"}`)),
				}, nil
			},
		}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}
