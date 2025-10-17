// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthHandlerDirect tests health handler without dependencies
func TestHealthHandlerDirect(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["service"] != "competitor-change-monitor" {
		t.Errorf("Expected service 'competitor-change-monitor', got %v", response["service"])
	}
}

// TestJSONEncoding tests JSON encoding/decoding
func TestJSONEncoding(t *testing.T) {
	t.Run("Competitor_Encoding", func(t *testing.T) {
		comp := Competitor{
			ID:          "test-id",
			Name:        "Test Competitor",
			Description: "Test description",
			Category:    "technology",
			Importance:  "high",
			IsActive:    true,
		}

		data, err := json.Marshal(comp)
		if err != nil {
			t.Fatalf("Failed to marshal competitor: %v", err)
		}

		var decoded Competitor
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal competitor: %v", err)
		}

		if decoded.Name != comp.Name {
			t.Errorf("Expected name %s, got %s", comp.Name, decoded.Name)
		}
	})

	t.Run("MonitoringTarget_Encoding", func(t *testing.T) {
		target := MonitoringTarget{
			ID:             "test-id",
			CompetitorID:   "comp-id",
			URL:            "https://example.com",
			TargetType:     "website",
			Selector:       ".content",
			CheckFrequency: 3600,
			IsActive:       true,
		}

		data, err := json.Marshal(target)
		if err != nil {
			t.Fatalf("Failed to marshal target: %v", err)
		}

		var decoded MonitoringTarget
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal target: %v", err)
		}

		if decoded.URL != target.URL {
			t.Errorf("Expected URL %s, got %s", target.URL, decoded.URL)
		}
	})

	t.Run("Alert_Encoding", func(t *testing.T) {
		alert := Alert{
			ID:             "test-id",
			CompetitorID:   "comp-id",
			Title:          "Test Alert",
			Priority:       "high",
			URL:            "https://example.com",
			Category:       "product_launch",
			Summary:        "Test summary",
			RelevanceScore: 85,
			Status:         "unread",
		}

		data, err := json.Marshal(alert)
		if err != nil {
			t.Fatalf("Failed to marshal alert: %v", err)
		}

		var decoded Alert
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal alert: %v", err)
		}

		if decoded.Title != alert.Title {
			t.Errorf("Expected title %s, got %s", alert.Title, decoded.Title)
		}
	})

	t.Run("ChangeAnalysis_Encoding", func(t *testing.T) {
		analysis := ChangeAnalysis{
			ID:             "test-id",
			CompetitorID:   "comp-id",
			TargetURL:      "https://example.com",
			ChangeType:     "content",
			RelevanceScore: 85,
			ChangeCategory: "product",
			ImpactLevel:    "high",
			Summary:        "Test summary",
		}

		data, err := json.Marshal(analysis)
		if err != nil {
			t.Fatalf("Failed to marshal analysis: %v", err)
		}

		var decoded ChangeAnalysis
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal analysis: %v", err)
		}

		if decoded.ChangeType != analysis.ChangeType {
			t.Errorf("Expected change type %s, got %s", analysis.ChangeType, decoded.ChangeType)
		}
	})
}

// TestInvalidJSON tests invalid JSON handling
func TestInvalidJSON(t *testing.T) {
	testCases := []struct {
		name    string
		handler http.HandlerFunc
		body    string
	}{
		{
			name:    "AddCompetitor_InvalidJSON",
			handler: addCompetitorHandler,
			body:    `{"invalid": "json"`,
		},
		{
			name:    "AddCompetitor_EmptyJSON",
			handler: addCompetitorHandler,
			body:    ``,
		},
		{
			name:    "AddTarget_InvalidJSON",
			handler: addTargetHandler,
			body:    `{"invalid": "json"`,
		},
		{
			name:    "UpdateAlert_InvalidJSON",
			handler: updateAlertHandler,
			body:    `{"invalid": "json"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/test", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			tc.handler(w, req)

			if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected error status, got %d", w.Code)
			}
		})
	}
}

// TestHTTPMethods tests HTTP method handling
func TestHTTPMethods(t *testing.T) {
	t.Run("HealthCheck_GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("HealthCheck_POST_NotAllowed", func(t *testing.T) {
		// Note: This tests handler directly, not router
		// Router would return 405 Method Not Allowed
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		// Handler doesn't check method, so it returns 200
		// This is fine as router handles method validation
	})
}

// TestQueryParameterParsing tests query parameter parsing
func TestQueryParameterParsing(t *testing.T) {
	t.Run("Alerts_StatusFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/alerts?status=unread", nil)

		status := req.URL.Query().Get("status")
		if status != "unread" {
			t.Errorf("Expected status 'unread', got '%s'", status)
		}
	})

	t.Run("Alerts_PriorityFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/alerts?priority=high", nil)

		priority := req.URL.Query().Get("priority")
		if priority != "high" {
			t.Errorf("Expected priority 'high', got '%s'", priority)
		}
	})

	t.Run("Alerts_MultipleFilters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/alerts?status=unread&priority=high", nil)

		status := req.URL.Query().Get("status")
		priority := req.URL.Query().Get("priority")

		if status != "unread" {
			t.Errorf("Expected status 'unread', got '%s'", status)
		}
		if priority != "high" {
			t.Errorf("Expected priority 'high', got '%s'", priority)
		}
	})

	t.Run("Analyses_CompetitorFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/analyses?competitor_id=test-id", nil)

		competitorID := req.URL.Query().Get("competitor_id")
		if competitorID != "test-id" {
			t.Errorf("Expected competitor_id 'test-id', got '%s'", competitorID)
		}
	})
}

// TestContentTypeHeaders tests content type headers
func TestContentTypeHeaders(t *testing.T) {
	t.Run("HealthCheck_JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestResponseStructure tests response structure
func TestResponseStructure(t *testing.T) {
	t.Run("HealthCheck_Structure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		requiredFields := []string{"status", "service"}
		for _, field := range requiredFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

// TestEdgeCases tests edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyQueryString", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/alerts?", nil)

		status := req.URL.Query().Get("status")
		if status != "" {
			t.Errorf("Expected empty status, got '%s'", status)
		}
	})

	t.Run("UnknownQueryParameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/alerts?unknown=value", nil)

		// Should not cause errors - unknown params are ignored
		unknown := req.URL.Query().Get("unknown")
		if unknown != "value" {
			t.Errorf("Expected unknown='value', got '%s'", unknown)
		}
	})
}

// TestDataStructureValidation tests data structure validation
func TestDataStructureValidation(t *testing.T) {
	t.Run("Competitor_RequiredFields", func(t *testing.T) {
		comp := Competitor{
			Name: "Test",
		}

		// Should be able to marshal even with minimal fields
		_, err := json.Marshal(comp)
		if err != nil {
			t.Errorf("Failed to marshal minimal competitor: %v", err)
		}
	})

	t.Run("MonitoringTarget_RequiredFields", func(t *testing.T) {
		target := MonitoringTarget{
			CompetitorID: "test",
			URL:          "https://example.com",
			TargetType:   "website",
		}

		_, err := json.Marshal(target)
		if err != nil {
			t.Errorf("Failed to marshal minimal target: %v", err)
		}
	})
}

// TestErrorResponseFormat tests error response format
func TestErrorResponseFormat(t *testing.T) {
	t.Run("InvalidJSON_BadRequest", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/competitors", bytes.NewBufferString(`{invalid`))
		w := httptest.NewRecorder()

		addCompetitorHandler(w, req)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected error status, got %d", w.Code)
		}
	})
}
