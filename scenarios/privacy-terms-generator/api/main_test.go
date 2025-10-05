package main

import (
	"net/http"
	"testing"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		})

		healthHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response != nil {
			// Check components exist
			if _, exists := response["components"]; !exists {
				t.Error("Expected components field in health response")
			}
			if _, exists := response["timestamp"]; !exists {
				t.Error("Expected timestamp field in health response")
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		})

		healthHandler(w, req)

		assertContentType(t, w, "application/json")
	})
}

// TestGenerateHandler tests the document generation endpoint
func TestGenerateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		genReq := TestData.GenerateRequest("Test Business", "privacy-policy", []string{"US"})

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body:   genReq,
		})

		generateHandler(w, req)

		// Note: This may fail if CLI is not available, which is expected in test environment
		// We're testing the handler logic, not the CLI integration
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("MissingBusinessName", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"document_type": "privacy-policy",
				"jurisdictions": []string{"US"},
			},
		})

		generateHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields")
	})

	t.Run("MissingDocumentType", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": "Test Business",
				"jurisdictions": []string{"US"},
			},
		})

		generateHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields")
	})

	t.Run("MissingJurisdictions", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": "Test Business",
				"document_type": "privacy-policy",
			},
		})

		generateHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields")
	})

	t.Run("EmptyJurisdictions", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": "Test Business",
				"document_type": "privacy-policy",
				"jurisdictions": []string{},
			},
		})

		generateHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Missing required fields")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body:   `{"invalid": "json"`,
		})

		generateHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/generate",
		})

		generateHandler(w, req)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("WithOptionalFields", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name":  "Test Business",
				"business_type":  "SaaS",
				"document_type":  "privacy-policy",
				"jurisdictions":  []string{"US", "EU"},
				"email":          "contact@test.com",
				"website":        "https://test.com",
				"data_types":     []string{"email", "name"},
				"custom_clauses": []string{"custom clause 1"},
				"format":         "html",
			},
		})

		generateHandler(w, req)

		// Handler should accept request (may fail at CLI stage which is ok for this test)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestTemplateFreshnessHandler tests the template freshness endpoint
func TestTemplateFreshnessHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/templates/freshness",
		})

		templateFreshnessHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			// Check required fields
			if _, exists := response["last_update"]; !exists {
				t.Error("Expected last_update field in response")
			}
			if _, exists := response["stale_templates"]; !exists {
				t.Error("Expected stale_templates field in response")
			}
			if _, exists := response["update_available"]; !exists {
				t.Error("Expected update_available field in response")
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/templates/freshness",
		})

		templateFreshnessHandler(w, req)

		assertContentType(t, w, "application/json")
	})
}

// TestDocumentHistoryHandler tests the document history endpoint
func TestDocumentHistoryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingDocumentID", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/documents/history",
		})

		documentHistoryHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Document ID required")
	})

	t.Run("WithDocumentID", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/documents/history",
			QueryParams: map[string]string{
				"id": "doc_123456",
			},
		})

		documentHistoryHandler(w, req)

		// Handler should attempt to call CLI (may fail which is ok for this test)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestSearchClausesHandler tests the search clauses endpoint
func TestSearchClausesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		searchReq := TestData.SearchClauseRequest("data retention", 10)

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body:   searchReq,
		})

		searchClausesHandler(w, req)

		// Handler should attempt to call CLI (may fail which is ok for this test)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("MissingQuery", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body: map[string]interface{}{
				"limit": 10,
			},
		})

		searchClausesHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Search query required")
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body: map[string]interface{}{
				"query": "",
				"limit": 10,
			},
		})

		searchClausesHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Search query required")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body:   `{"invalid": "json"`,
		})

		searchClausesHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/clauses/search",
		})

		searchClausesHandler(w, req)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("WithOptionalFilters", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body: map[string]interface{}{
				"query":        "data retention",
				"limit":        5,
				"clause_type":  "privacy",
				"jurisdiction": "US",
			},
		})

		searchClausesHandler(w, req)

		// Handler should accept request (may fail at CLI stage which is ok for this test)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(testHandler)

	t.Run("AddsCORSHeaders", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})

		wrappedHandler(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS Allow-Origin header to be '*'")
		}
		if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
			t.Error("Expected CORS Allow-Methods header")
		}
		if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
			t.Error("Expected CORS Allow-Headers header")
		}
	})

	t.Run("HandlesOPTIONS", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/test",
		})

		wrappedHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("PassesThroughOtherMethods", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
		})

		wrappedHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestGenerateHandlerErrorPatterns uses systematic error testing
func TestGenerateHandlerErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	suite := &HandlerTestSuite{
		HandlerName: "generateHandler",
		Handler:     generateHandler,
		BaseURL:     "/api/v1/legal/generate",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/legal/generate").
		AddInvalidMethod("GET", "/api/v1/legal/generate").
		AddInvalidMethod("PUT", "/api/v1/legal/generate").
		AddInvalidMethod("DELETE", "/api/v1/legal/generate").
		AddEmptyBody("POST", "/api/v1/legal/generate").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestSearchClausesHandlerErrorPatterns uses systematic error testing
func TestSearchClausesHandlerErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	suite := &HandlerTestSuite{
		HandlerName: "searchClausesHandler",
		Handler:     searchClausesHandler,
		BaseURL:     "/api/v1/legal/clauses/search",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/legal/clauses/search").
		AddInvalidMethod("GET", "/api/v1/legal/clauses/search").
		AddInvalidMethod("PUT", "/api/v1/legal/clauses/search").
		AddInvalidMethod("DELETE", "/api/v1/legal/clauses/search").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestEdgeCases tests boundary conditions and edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GenerateWithVeryLongBusinessName", func(t *testing.T) {
		longName := string(make([]byte, 10000))
		for range longName {
			longName = "A" + longName
		}

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": longName,
				"document_type": "privacy-policy",
				"jurisdictions": []string{"US"},
			},
		})

		generateHandler(w, req)

		// Should handle gracefully (may fail at CLI which is acceptable)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200, 400, or 500, got %d", w.Code)
		}
	})

	t.Run("GenerateWithManyJurisdictions", func(t *testing.T) {
		jurisdictions := []string{}
		for i := 0; i < 100; i++ {
			jurisdictions = append(jurisdictions, "US")
		}

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body: map[string]interface{}{
				"business_name": "Test",
				"document_type": "privacy-policy",
				"jurisdictions": jurisdictions,
			},
		})

		generateHandler(w, req)

		// Should handle multiple jurisdictions
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("SearchWithZeroLimit", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body: map[string]interface{}{
				"query": "test",
				"limit": 0,
			},
		})

		searchClausesHandler(w, req)

		// Should handle zero limit (CLI will use default)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("SearchWithNegativeLimit", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body: map[string]interface{}{
				"query": "test",
				"limit": -10,
			},
		})

		searchClausesHandler(w, req)

		// Should handle negative limit
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}
