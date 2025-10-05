package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Returns environment variable when set",
			key:          "TEST_VAR_1",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "Returns default when env var not set",
			key:          "TEST_VAR_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "Returns empty string when both are empty",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %s; expected %s",
					tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_Returns_Healthy_Status", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := testHandlerWithRequest(t, healthHandler, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify response structure
		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		// Validate response fields
		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if response.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", response.Version)
		}

		// Validate timestamp is recent
		if time.Since(response.Timestamp) > 5*time.Second {
			t.Errorf("Timestamp is too old: %v", response.Timestamp)
		}

		// Validate services
		expectedServices := []string{"axe-core", "wave", "pa11y", "postgres"}
		for _, service := range expectedServices {
			status, exists := response.Services[service]
			if !exists {
				t.Errorf("Expected service '%s' in health check", service)
			}
			if status != "healthy" {
				t.Errorf("Expected service '%s' to be healthy, got '%s'", service, status)
			}
		}
	})

	t.Run("Returns_JSON_Content_Type", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := testHandlerWithRequest(t, healthHandler, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestScansHandler tests the scans endpoint
func TestScansHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_Returns_Scans_List", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}

		w := testHandlerWithRequest(t, scansHandler, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify response is an array
		var scans []Scan
		if err := json.Unmarshal(w.Body.Bytes(), &scans); err != nil {
			t.Fatalf("Failed to parse scans response: %v", err)
		}

		// Validate we have scans
		if len(scans) == 0 {
			t.Error("Expected at least one scan in response")
		}

		// Validate first scan structure
		if len(scans) > 0 {
			scan := scans[0]
			if scan.ID == "" {
				t.Error("Expected scan to have ID")
			}
			if scan.URL == "" {
				t.Error("Expected scan to have URL")
			}
			if scan.Status == "" {
				t.Error("Expected scan to have status")
			}
			if scan.Created.IsZero() {
				t.Error("Expected scan to have created timestamp")
			}
		}
	})

	t.Run("Returns_Valid_Scan_Statuses", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}

		w := testHandlerWithRequest(t, scansHandler, req)

		var scans []Scan
		json.Unmarshal(w.Body.Bytes(), &scans)

		validStatuses := map[string]bool{
			"completed":   true,
			"in_progress": true,
			"pending":     true,
			"failed":      true,
		}

		for _, scan := range scans {
			if !validStatuses[scan.Status] {
				t.Errorf("Invalid scan status: '%s'", scan.Status)
			}
		}
	})

	t.Run("Returns_JSON_Content_Type", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}

		w := testHandlerWithRequest(t, scansHandler, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestViolationsHandler tests the violations endpoint
func TestViolationsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_Returns_Violations_List", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/violations",
		}

		w := testHandlerWithRequest(t, violationsHandler, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify response is an array
		var violations []Violation
		if err := json.Unmarshal(w.Body.Bytes(), &violations); err != nil {
			t.Fatalf("Failed to parse violations response: %v", err)
		}

		// Validate we have violations
		if len(violations) == 0 {
			t.Error("Expected at least one violation in response")
		}

		// Validate first violation structure
		if len(violations) > 0 {
			violation := violations[0]
			if violation.ID == "" {
				t.Error("Expected violation to have ID")
			}
			if violation.Type == "" {
				t.Error("Expected violation to have type")
			}
			if violation.Description == "" {
				t.Error("Expected violation to have description")
			}
			if violation.Severity == "" {
				t.Error("Expected violation to have severity")
			}
			if violation.Element == "" {
				t.Error("Expected violation to have element")
			}
		}
	})

	t.Run("Returns_Valid_Severity_Levels", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/violations",
		}

		w := testHandlerWithRequest(t, violationsHandler, req)

		var violations []Violation
		json.Unmarshal(w.Body.Bytes(), &violations)

		validSeverities := map[string]bool{
			"low":      true,
			"medium":   true,
			"high":     true,
			"critical": true,
		}

		for _, violation := range violations {
			if !validSeverities[violation.Severity] {
				t.Errorf("Invalid violation severity: '%s'", violation.Severity)
			}
		}
	})

	t.Run("Returns_JSON_Content_Type", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/violations",
		}

		w := testHandlerWithRequest(t, violationsHandler, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestReportsHandler tests the reports endpoint
func TestReportsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_Returns_Reports_List", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/reports",
		}

		w := testHandlerWithRequest(t, reportsHandler, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify response is an array
		var reports []Report
		if err := json.Unmarshal(w.Body.Bytes(), &reports); err != nil {
			t.Fatalf("Failed to parse reports response: %v", err)
		}

		// Validate we have reports
		if len(reports) == 0 {
			t.Error("Expected at least one report in response")
		}

		// Validate first report structure
		if len(reports) > 0 {
			report := reports[0]
			if report.ID == "" {
				t.Error("Expected report to have ID")
			}
			if report.ScanID == "" {
				t.Error("Expected report to have scan ID")
			}
			if report.Title == "" {
				t.Error("Expected report to have title")
			}
			if report.Score < 0 || report.Score > 100 {
				t.Errorf("Expected report score between 0-100, got %f", report.Score)
			}
			if report.Date.IsZero() {
				t.Error("Expected report to have date")
			}
		}
	})

	t.Run("Reports_Include_Issues", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/reports",
		}

		w := testHandlerWithRequest(t, reportsHandler, req)

		var reports []Report
		json.Unmarshal(w.Body.Bytes(), &reports)

		for _, report := range reports {
			if report.Issues == nil {
				t.Error("Expected report to have issues array")
			}
		}
	})

	t.Run("Returns_JSON_Content_Type", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/reports",
		}

		w := testHandlerWithRequest(t, reportsHandler, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestDataStructures tests the data structure definitions
func TestDataStructures(t *testing.T) {
	t.Run("HealthResponse_Structure", func(t *testing.T) {
		health := HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now(),
			Version:   "1.0.0",
			Services: map[string]string{
				"test": "healthy",
			},
		}

		// Verify JSON serialization
		data, err := json.Marshal(health)
		if err != nil {
			t.Errorf("Failed to marshal HealthResponse: %v", err)
		}

		// Verify JSON deserialization
		var decoded HealthResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal HealthResponse: %v", err)
		}

		if decoded.Status != health.Status {
			t.Error("Status not preserved through JSON marshal/unmarshal")
		}
	})

	t.Run("Scan_Structure", func(t *testing.T) {
		scan := Scan{
			ID:         "test-id",
			URL:        "https://example.com",
			Status:     "completed",
			Created:    time.Now(),
			Violations: 5,
		}

		// Verify JSON serialization
		data, err := json.Marshal(scan)
		if err != nil {
			t.Errorf("Failed to marshal Scan: %v", err)
		}

		// Verify JSON deserialization
		var decoded Scan
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal Scan: %v", err)
		}

		if decoded.ID != scan.ID {
			t.Error("ID not preserved through JSON marshal/unmarshal")
		}
	})

	t.Run("Violation_Structure", func(t *testing.T) {
		violation := Violation{
			ID:          "viol-001",
			Type:        "color-contrast",
			Description: "Insufficient color contrast",
			Severity:    "high",
			Element:     "button",
		}

		// Verify JSON serialization
		data, err := json.Marshal(violation)
		if err != nil {
			t.Errorf("Failed to marshal Violation: %v", err)
		}

		// Verify JSON deserialization
		var decoded Violation
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal Violation: %v", err)
		}

		if decoded.Type != violation.Type {
			t.Error("Type not preserved through JSON marshal/unmarshal")
		}
	})

	t.Run("Report_Structure", func(t *testing.T) {
		report := Report{
			ID:     "report-001",
			ScanID: "scan-001",
			Title:  "Test Report",
			Score:  85.5,
			Issues: []Violation{},
			Date:   time.Now(),
		}

		// Verify JSON serialization
		data, err := json.Marshal(report)
		if err != nil {
			t.Errorf("Failed to marshal Report: %v", err)
		}

		// Verify JSON deserialization
		var decoded Report
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal Report: %v", err)
		}

		if decoded.Score != report.Score {
			t.Error("Score not preserved through JSON marshal/unmarshal")
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Empty_Request_Body", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
			Body:   nil,
		}

		w := testHandlerWithRequest(t, healthHandler, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty body on GET, got %d", w.Code)
		}
	})

	t.Run("Large_Response_Handling", func(t *testing.T) {
		// Test that endpoints can handle returning data
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}

		w := testHandlerWithRequest(t, scansHandler, req)

		if w.Body.Len() == 0 {
			t.Error("Expected non-empty response body")
		}
	})

	t.Run("Concurrent_Requests", func(t *testing.T) {
		// Test concurrent access to handlers
		done := make(chan bool)
		errorChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				w := testHandlerWithRequest(t, healthHandler, req)
				if w.Code != http.StatusOK {
					errorChan <- http.ErrServerClosed
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		close(errorChan)
		for err := range errorChan {
			t.Errorf("Concurrent request failed: %v", err)
		}
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Invalid_Path", func(t *testing.T) {
		// While we don't have routing logic here, we test handler behavior
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/invalid/path",
		}

		w := testHandlerWithRequest(t, healthHandler, req)

		// Health handler should still work regardless of path
		if w.Code != http.StatusOK {
			t.Logf("Handler responded with: %d", w.Code)
		}
	})
}

// TestIntegration tests integration scenarios
func TestIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Health_Then_Scans_Workflow", func(t *testing.T) {
		// Test health check
		healthReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		healthW := testHandlerWithRequest(t, healthHandler, healthReq)

		if healthW.Code != http.StatusOK {
			t.Fatalf("Health check failed: %d", healthW.Code)
		}

		// Test scans endpoint
		scansReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}
		scansW := testHandlerWithRequest(t, scansHandler, scansReq)

		if scansW.Code != http.StatusOK {
			t.Fatalf("Scans endpoint failed: %d", scansW.Code)
		}
	})

	t.Run("All_Endpoints_Accessible", func(t *testing.T) {
		endpoints := []struct {
			handler http.HandlerFunc
			path    string
		}{
			{healthHandler, "/health"},
			{scansHandler, "/api/scans"},
			{violationsHandler, "/api/violations"},
			{reportsHandler, "/api/reports"},
		}

		for _, endpoint := range endpoints {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   endpoint.path,
			}

			w := testHandlerWithRequest(t, endpoint.handler, req)

			if w.Code != http.StatusOK {
				t.Errorf("Endpoint %s failed with status %d", endpoint.path, w.Code)
			}
		}
	})
}
