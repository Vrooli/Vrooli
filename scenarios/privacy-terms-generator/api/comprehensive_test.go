package main

import (
	"net/http"
	"strings"
	"testing"
)

// TestComprehensiveGenerateHandler provides comprehensive coverage of generateHandler
func TestComprehensiveGenerateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	testCases := []struct {
		name           string
		request        map[string]interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body string)
	}{
		{
			name: "ValidPrivacyPolicy",
			request: map[string]interface{}{
				"business_name": "ACME Corp",
				"document_type": "privacy",
				"jurisdictions": []string{"US"},
				"format":        "markdown",
			},
			expectedStatus: http.StatusInternalServerError, // CLI not available in test
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Document generation failed") {
					t.Log("Expected CLI error, got:", body)
				}
			},
		},
		{
			name: "ValidTermsOfService",
			request: map[string]interface{}{
				"business_name": "ACME Corp",
				"document_type": "terms",
				"jurisdictions": []string{"US"},
			},
			expectedStatus: http.StatusInternalServerError, // CLI not available
			validateBody:   nil,
		},
		{
			name: "MultipleJurisdictions",
			request: map[string]interface{}{
				"business_name": "GlobalCo",
				"document_type": "privacy",
				"jurisdictions": []string{"US", "EU", "UK"},
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
		{
			name: "WithAllOptionalFields",
			request: map[string]interface{}{
				"business_name":  "TechStartup",
				"business_type":  "SaaS",
				"document_type":  "cookie",
				"jurisdictions":  []string{"US"},
				"email":          "legal@techstartup.com",
				"website":        "https://techstartup.com",
				"data_types":     []string{"email", "name", "location"},
				"custom_clauses": []string{"Custom clause 1"},
				"format":         "html",
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
		{
			name: "MissingBusinessName",
			request: map[string]interface{}{
				"document_type": "privacy",
				"jurisdictions": []string{"US"},
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Missing required fields") {
					t.Error("Expected missing required fields error")
				}
			},
		},
		{
			name: "MissingDocumentType",
			request: map[string]interface{}{
				"business_name": "Test",
				"jurisdictions": []string{"US"},
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Missing required fields") {
					t.Error("Expected missing required fields error")
				}
			},
		},
		{
			name: "EmptyJurisdictions",
			request: map[string]interface{}{
				"business_name": "Test",
				"document_type": "privacy",
				"jurisdictions": []string{},
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Missing required fields") {
					t.Error("Expected missing required fields error")
				}
			},
		},
		{
			name:           "EmptyBody",
			request:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			validateBody:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, req := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/legal/generate",
				Body:   tc.request,
			})

			generateHandler(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.validateBody != nil {
				tc.validateBody(t, w.Body.String())
			}
		})
	}
}

// TestComprehensiveSearchClausesHandler provides comprehensive coverage
func TestComprehensiveSearchClausesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name           string
		request        map[string]interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body string)
	}{
		{
			name: "ValidSearchWithAllFilters",
			request: map[string]interface{}{
				"query":        "data retention",
				"limit":        10,
				"clause_type":  "privacy",
				"jurisdiction": "US",
			},
			expectedStatus: http.StatusInternalServerError, // CLI not available
			validateBody:   nil,
		},
		{
			name: "ValidSearchMinimal",
			request: map[string]interface{}{
				"query": "gdpr",
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
		{
			name: "MissingQuery",
			request: map[string]interface{}{
				"limit": 5,
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Search query required") {
					t.Error("Expected search query required error")
				}
			},
		},
		{
			name: "EmptyQuery",
			request: map[string]interface{}{
				"query": "",
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Search query required") {
					t.Error("Expected search query required error")
				}
			},
		},
		{
			name: "LargeLimit",
			request: map[string]interface{}{
				"query": "privacy",
				"limit": 10000,
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
		{
			name: "NegativeLimit",
			request: map[string]interface{}{
				"query": "privacy",
				"limit": -5,
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, req := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/legal/clauses/search",
				Body:   tc.request,
			})

			searchClausesHandler(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.validateBody != nil {
				tc.validateBody(t, w.Body.String())
			}
		})
	}
}

// TestComprehensiveDocumentHistoryHandler provides comprehensive coverage
func TestComprehensiveDocumentHistoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		validateBody   func(t *testing.T, body string)
	}{
		{
			name: "ValidDocumentID",
			queryParams: map[string]string{
				"id": "doc_123456789",
			},
			expectedStatus: http.StatusInternalServerError, // CLI not available
			validateBody:   nil,
		},
		{
			name: "UUIDDocumentID",
			queryParams: map[string]string{
				"id": "550e8400-e29b-41d4-a716-446655440000",
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody:   nil,
		},
		{
			name:           "MissingDocumentID",
			queryParams:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Document ID required") {
					t.Error("Expected document ID required error")
				}
			},
		},
		{
			name: "EmptyDocumentID",
			queryParams: map[string]string{
				"id": "",
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Document ID required") {
					t.Error("Expected document ID required error")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, req := makeHTTPRequest(HTTPTestRequest{
				Method:      "GET",
				Path:        "/api/v1/legal/documents/history",
				QueryParams: tc.queryParams,
			})

			documentHistoryHandler(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.validateBody != nil {
				tc.validateBody(t, w.Body.String())
			}
		})
	}
}

// TestHTTPMethodValidation ensures all handlers validate HTTP methods
func TestHTTPMethodValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		handler        http.HandlerFunc
		path           string
		allowedMethods []string
		deniedMethods  []string
	}{
		{
			handler:        generateHandler,
			path:           "/api/v1/legal/generate",
			allowedMethods: []string{"POST"},
			deniedMethods:  []string{"GET", "PUT", "DELETE", "PATCH"},
		},
		{
			handler:        searchClausesHandler,
			path:           "/api/v1/legal/clauses/search",
			allowedMethods: []string{"POST"},
			deniedMethods:  []string{"GET", "PUT", "DELETE", "PATCH"},
		},
		{
			handler:        healthHandler,
			path:           "/api/health",
			allowedMethods: []string{"GET", "POST", "PUT", "DELETE"}, // No method check
			deniedMethods:  []string{},
		},
	}

	for _, tc := range testCases {
		for _, method := range tc.deniedMethods {
			t.Run(tc.path+"_"+method, func(t *testing.T) {
				w, req := makeHTTPRequest(HTTPTestRequest{
					Method: method,
					Path:   tc.path,
				})

				tc.handler(w, req)

				if w.Code != http.StatusMethodNotAllowed {
					t.Errorf("Expected 405 for %s %s, got %d",
						method, tc.path, w.Code)
				}
			})
		}
	}
}

// TestJSONResponseStructure validates JSON response structures
func TestJSONResponseStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthResponse", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		})

		healthHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		// Validate required fields
		requiredFields := []string{"status", "timestamp", "components"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}

		// Validate components structure
		if components, ok := response["components"].(map[string]interface{}); ok {
			if len(components) == 0 {
				t.Error("Components should not be empty")
			}
		}
	})

	t.Run("TemplateFreshnessResponse", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/templates/freshness",
		})

		templateFreshnessHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		// Validate required fields
		requiredFields := []string{"last_update", "stale_templates", "update_available"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

// TestEdgeCasesAndBoundaries tests boundary conditions
func TestEdgeCasesAndBoundaries(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("VeryLongBusinessName", func(t *testing.T) {
		longName := strings.Repeat("A", 10000)

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": longName,
				"document_type": "privacy",
				"jurisdictions": []string{"US"},
			},
		})

		generateHandler(w, req)

		// Should handle gracefully (may fail at CLI)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200, 400, or 500, got %d", w.Code)
		}
	})

	t.Run("VeryLongSearchQuery", func(t *testing.T) {
		longQuery := strings.Repeat("privacy ", 1000)

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body: map[string]interface{}{
				"query": longQuery,
			},
		})

		searchClausesHandler(w, req)

		// Should handle gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("ManyDataTypes", func(t *testing.T) {
		dataTypes := make([]string, 100)
		for i := range dataTypes {
			dataTypes[i] = "type" + string(rune(i))
		}

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": "Test",
				"document_type": "privacy",
				"jurisdictions": []string{"US"},
				"data_types":    dataTypes,
			},
		})

		generateHandler(w, req)

		// Should handle gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInBusinessName", func(t *testing.T) {
		specialNames := []string{
			"Test & Co.",
			"Test<script>alert('xss')</script>",
			"Test'; DROP TABLE documents;--",
			"Test\x00null",
			"Test™ © ®",
			"Test\n\r\t",
		}

		for _, name := range specialNames {
			w, req := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/legal/generate",
				Body: map[string]interface{}{
					"business_name": name,
					"document_type": "privacy",
					"jurisdictions": []string{"US"},
				},
			})

			generateHandler(w, req)

			// Should handle special characters
			if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
				t.Errorf("Failed to handle special characters in: %s (status: %d)", name, w.Code)
			}
		}
	})
}

// TestConcurrentRequests tests thread safety
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ParallelHealthChecks", func(t *testing.T) {
		concurrency := 50
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				w, req := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/api/health",
				})

				healthHandler(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Health check failed: %d", w.Code)
				}

				done <- true
			}()
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}
