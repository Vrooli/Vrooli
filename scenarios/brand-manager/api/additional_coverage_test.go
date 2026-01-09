package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestJSONUnmarshalBrand tests Brand JSON unmarshaling edge cases
func TestJSONUnmarshalBrand(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		expectErr bool
	}{
		{
			name: "ValidBrand",
			jsonStr: `{
				"id": "00000000-0000-0000-0000-000000000000",
				"name": "Test",
				"brand_colors": {"primary": "#FF0000"},
				"assets": [],
				"metadata": {},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}`,
			expectErr: false,
		},
		{
			name:      "InvalidJSON",
			jsonStr:   `{"invalid": json}`,
			expectErr: true,
		},
		{
			name: "MissingFields",
			jsonStr: `{
				"id": "00000000-0000-0000-0000-000000000000"
			}`,
			expectErr: false, // Missing fields are optional
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var brand Brand
			err := json.Unmarshal([]byte(tt.jsonStr), &brand)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestJSONUnmarshalIntegration tests IntegrationRequest JSON unmarshaling
func TestJSONUnmarshalIntegration(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		expectErr bool
	}{
		{
			name: "ValidIntegration",
			jsonStr: `{
				"id": "00000000-0000-0000-0000-000000000000",
				"brand_id": "00000000-0000-0000-0000-000000000001",
				"target_app_path": "/test",
				"integration_type": "full",
				"status": "pending",
				"request_payload": {},
				"created_at": "2024-01-01T00:00:00Z"
			}`,
			expectErr: false,
		},
		{
			name:      "InvalidJSON",
			jsonStr:   `{"invalid": json}`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var integration IntegrationRequest
			err := json.Unmarshal([]byte(tt.jsonStr), &integration)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestHTTPErrorWithNilError tests HTTPError with nil error
func TestHTTPErrorWithNilError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	w := httptest.NewRecorder()
	HTTPError(w, "test error without underlying error", http.StatusBadRequest, nil)

	assertErrorResponse(t, w, http.StatusBadRequest, "test error without underlying error")
}

// TestNewBrandManagerServiceNilDB tests service creation with nil database
func TestNewBrandManagerServiceNilDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	service := NewBrandManagerService(
		nil, // nil DB
		"http://n8n:5678",
		"http://comfyui:8188",
		"minio:9000",
		"http://vault:8200",
	)

	if service == nil {
		t.Fatal("Expected service to be created even with nil DB")
	}

	if service.n8nBaseURL != "http://n8n:5678" {
		t.Errorf("Expected n8nBaseURL to be set correctly")
	}

	if service.comfyUIURL != "http://comfyui:8188" {
		t.Errorf("Expected comfyUIURL to be set correctly")
	}

	if service.minioEndpoint != "minio:9000" {
		t.Errorf("Expected minioEndpoint to be set correctly")
	}

	if service.vaultAddr != "http://vault:8200" {
		t.Errorf("Expected vaultAddr to be set correctly")
	}

	if service.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}

	// Verify timeout is set
	if service.httpClient.Timeout != httpTimeout {
		t.Errorf("Expected timeout %v, got %v", httpTimeout, service.httpClient.Timeout)
	}
}

// TestHealthEndpointHeaders tests Health endpoint headers
func TestHealthEndpointHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	w, err := makeHTTPRequest(req, Health)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got: %s", contentType)
	}
}

// TestTimeoutConstants validates timeout values are reasonable
func TestTimeoutConstants(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		min     time.Duration
		max     time.Duration
	}{
		{"httpTimeout", httpTimeout, 1 * time.Second, 5 * time.Minute},
		{"brandGenTimeout", brandGenTimeout, 30 * time.Second, 10 * time.Minute},
		{"integrationTimeout", integrationTimeout, 1 * time.Minute, 1 * time.Hour},
		{"discoveryDelay", discoveryDelay, 1 * time.Second, 1 * time.Minute},
		{"connMaxLifetime", connMaxLifetime, 1 * time.Minute, 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout < tt.min {
				t.Errorf("%s (%v) is less than minimum (%v)", tt.name, tt.timeout, tt.min)
			}
			if tt.timeout > tt.max {
				t.Errorf("%s (%v) exceeds maximum (%v)", tt.name, tt.timeout, tt.max)
			}
		})
	}
}

// TestDatabaseConnectionLimits validates database connection settings
func TestDatabaseConnectionLimits(t *testing.T) {
	if maxDBConnections <= 0 {
		t.Error("maxDBConnections must be positive")
	}

	if maxIdleConnections <= 0 {
		t.Error("maxIdleConnections must be positive")
	}

	if maxIdleConnections > maxDBConnections {
		t.Error("maxIdleConnections should not exceed maxDBConnections")
	}

	if maxDBConnections < 5 {
		t.Error("maxDBConnections seems too low for production")
	}
}

// TestDefaultValues tests default values are sensible
func TestDefaultValues(t *testing.T) {
	if defaultLimit <= 0 {
		t.Error("defaultLimit must be positive")
	}

	if defaultLimit > 100 {
		t.Error("defaultLimit seems too high")
	}

	if defaultPort == "" {
		t.Error("defaultPort should be set")
	}
}

// TestBrandColorsValidJSON tests that brand colors can be marshaled/unmarshaled
func TestBrandColorsValidJSON(t *testing.T) {
	colors := map[string]interface{}{
		"primary":   "#FF5733",
		"secondary": "#33FF57",
		"accent":    "#3357FF",
	}

	brand := Brand{
		ID:          uuid.New(),
		Name:        "Test Brand",
		BrandColors: colors,
		Assets:      []interface{}{},
		Metadata:    map[string]interface{}{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Marshal
	data, err := json.Marshal(brand)
	if err != nil {
		t.Fatalf("Failed to marshal brand: %v", err)
	}

	// Unmarshal
	var decoded Brand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal brand: %v", err)
	}

	// Verify colors
	if len(decoded.BrandColors) != len(colors) {
		t.Errorf("Expected %d colors, got %d", len(colors), len(decoded.BrandColors))
	}
}

// TestIntegrationPayloadsValidJSON tests integration payloads JSON handling
func TestIntegrationPayloadsValidJSON(t *testing.T) {
	reqPayload := map[string]interface{}{
		"brandId":       uuid.New().String(),
		"targetAppPath": "/test/app",
		"options": map[string]interface{}{
			"createBackup": true,
			"verbose":      false,
		},
	}

	respPayload := map[string]interface{}{
		"status":  "success",
		"message": "Integration completed",
		"details": []string{"step1", "step2"},
	}

	now := time.Now()
	integration := IntegrationRequest{
		ID:              uuid.New(),
		BrandID:         uuid.New(),
		TargetAppPath:   "/test/app",
		IntegrationType: "full",
		Status:          "completed",
		RequestPayload:  reqPayload,
		ResponsePayload: respPayload,
		CreatedAt:       now,
		CompletedAt:     &now,
	}

	// Marshal
	data, err := json.Marshal(integration)
	if err != nil {
		t.Fatalf("Failed to marshal integration: %v", err)
	}

	// Unmarshal
	var decoded IntegrationRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal integration: %v", err)
	}

	// Verify payloads
	if len(decoded.RequestPayload) != len(reqPayload) {
		t.Errorf("Request payload mismatch")
	}

	if len(decoded.ResponsePayload) != len(respPayload) {
		t.Errorf("Response payload mismatch")
	}
}

// TestMakeHTTPRequestWithHeaders tests HTTP request with custom headers
func TestMakeHTTPRequestWithHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	}

	w, err := makeHTTPRequest(req, Health)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestMakeHTTPRequestWithBody tests HTTP request body handling
func TestMakeHTTPRequestWithBody(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testBody := map[string]interface{}{
		"test":  "value",
		"count": 42,
		"flag":  true,
	}

	// Create a simple handler that echoes back the body
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(body)
	}

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/test",
		Body:   testBody,
	}

	w, err := makeHTTPRequest(req, handler)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["test"] != "value" {
		t.Errorf("Expected test=value, got %v", response["test"])
	}
}

// TestServiceNameConstant ensures service name matches expected value
func TestServiceNameConstant(t *testing.T) {
	if serviceName != "brand-manager" {
		t.Errorf("Expected service name 'brand-manager', got '%s'", serviceName)
	}
}

// TestAPIVersionFormat validates API version format
func TestAPIVersionFormat(t *testing.T) {
	// Should be semantic versioning format
	if len(apiVersion) < 5 { // At minimum "1.0.0"
		t.Errorf("API version seems invalid: %s", apiVersion)
	}

	// Check contains dots
	if !bytes.Contains([]byte(apiVersion), []byte(".")) {
		t.Errorf("API version should use semantic versioning: %s", apiVersion)
	}
}
