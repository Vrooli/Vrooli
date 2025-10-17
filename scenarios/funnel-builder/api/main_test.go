package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHealth tests the health check endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusOK, []string{"status", "time"})

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
	})

	t.Run("API v1 Health", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/api/v1/health", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertJSONResponse(t, recorder, http.StatusOK, []string{"status", "time"})
	})

	t.Run("Response Time", func(t *testing.T) {
		start := time.Now()
		req, _ := makeHTTPRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)
		assertResponseTime(t, start, 100*time.Millisecond, "Health check")
	})
}

// TestCreateFunnel tests funnel creation
func TestCreateFunnel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success_WithSteps", func(t *testing.T) {
		funnelData := map[string]interface{}{
			"name":        "Test Funnel",
			"description": "A test funnel",
			"steps": []map[string]interface{}{
				{
					"type":     "form",
					"position": 0,
					"title":    "Contact Information",
					"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email","required":true}]}`),
				},
			},
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusCreated, []string{"id", "slug", "preview_url"})
		funnelID := response["id"].(string)
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Verify funnel was created
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		funnel := assertJSONResponse(t, recorder, http.StatusOK, []string{"id", "name", "steps"})
		if funnel["name"] != "Test Funnel" {
			t.Errorf("Expected name 'Test Funnel', got '%v'", funnel["name"])
		}
	})

	t.Run("Success_WithoutSteps", func(t *testing.T) {
		funnelData := map[string]interface{}{
			"name": "Empty Funnel",
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusCreated, []string{"id", "slug"})
		defer cleanupTestData(t, testServer.Server, response["id"].(string))
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		funnelData := map[string]interface{}{
			"description": "Missing name field",
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusBadRequest)
	})

	t.Run("Error_EmptyName", func(t *testing.T) {
		funnelData := map[string]interface{}{
			"name": "",
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusBadRequest)
	})

	t.Run("Edge_UnicodeCharacters", func(t *testing.T) {
		funnelData := map[string]interface{}{
			"name": "测试漏斗 テストファネル",
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusCreated, []string{"id"})
		defer cleanupTestData(t, testServer.Server, response["id"].(string))
	})
}

// TestGetFunnel tests single funnel retrieval
func TestGetFunnel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Get Test Funnel")
		defer cleanupTestData(t, testServer.Server, funnelID)

		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusOK, []string{"id", "name", "slug", "steps"})

		if response["id"] != funnelID {
			t.Errorf("Expected id '%s', got '%v'", funnelID, response["id"])
		}

		steps := response["steps"].([]interface{})
		if len(steps) != 2 {
			t.Errorf("Expected 2 steps, got %d", len(steps))
		}
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		// Use systematic error patterns
		suite := NewHandlerTestSuite(t)
		if suite == nil {
			return
		}
		defer suite.Cleanup()

		patterns := NewTestScenarioBuilder().
			AddNonExistentFunnel("/api/v1/funnels/%s").
			Build()

		suite.TestAllPatterns(patterns)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/not-a-valid-uuid", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusNotFound)
	})
}

// TestUpdateFunnel tests funnel updates
func TestUpdateFunnel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success_FullUpdate", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Update Test Funnel")
		defer cleanupTestData(t, testServer.Server, funnelID)

		updateData := map[string]interface{}{
			"name":        "Updated Funnel Name",
			"description": "Updated description",
			"status":      "active",
			"settings":    json.RawMessage(`{"theme":{"primaryColor":"#ff0000"}}`),
		}

		req, _ := makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertJSONResponse(t, recorder, http.StatusOK, []string{"message"})

		// Verify update
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusOK, []string{"name"})
		if response["name"] != "Updated Funnel Name" {
			t.Errorf("Expected updated name, got '%v'", response["name"])
		}
	})
}

// TestDeleteFunnel tests funnel deletion
func TestDeleteFunnel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Delete Test Funnel")

		req, _ := makeHTTPRequest("DELETE", "/api/v1/funnels/"+funnelID, nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertJSONResponse(t, recorder, http.StatusOK, []string{"message"})

		// Verify deletion
		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusNotFound)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		req, _ := makeHTTPRequest("DELETE", "/api/v1/funnels/00000000-0000-0000-0000-000000000000", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Deletion of non-existent might still return OK in some implementations
		if recorder.Code != http.StatusOK && recorder.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", recorder.Code)
		}
	})
}

// TestGetAnalytics tests analytics retrieval
func TestGetAnalytics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success_WithData", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Analytics Test Funnel")
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Create some test data
		createTestLead(t, testServer.Server, funnelID)

		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusOK, []string{
			"funnelId", "totalViews", "totalLeads", "conversionRate", "dropOffPoints",
		})

		if response["funnelId"] != funnelID {
			t.Errorf("Expected funnelId '%s', got '%v'", funnelID, response["funnelId"])
		}

		dropOffPoints := response["dropOffPoints"].([]interface{})
		if len(dropOffPoints) == 0 {
			t.Error("Expected drop-off points data")
		}
	})

	t.Run("Success_NoData", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Empty Analytics Funnel")
		defer cleanupTestData(t, testServer.Server, funnelID)

		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		response := assertJSONResponse(t, recorder, http.StatusOK, []string{"totalViews", "conversionRate"})

		if response["totalViews"].(float64) != 0 {
			t.Error("Expected zero views for new funnel")
		}

		if response["conversionRate"].(float64) != 0 {
			t.Error("Expected zero conversion rate for new funnel")
		}
	})
}

// TestGetLeads tests lead retrieval
func TestGetLeads(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success_JSONFormat", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "Leads Test Funnel")
		defer cleanupTestData(t, testServer.Server, funnelID)

		createTestLead(t, testServer.Server, funnelID)

		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/leads", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var leads []interface{}
		json.Unmarshal(recorder.Body.Bytes(), &leads)

		if len(leads) == 0 {
			t.Error("Expected at least one lead")
		}
	})

	t.Run("Success_CSVFormat", func(t *testing.T) {
		funnelID := createTestFunnel(t, testServer.Server, "CSV Leads Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		createTestLead(t, testServer.Server, funnelID)

		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/leads?format=csv", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		contentType := recorder.Header().Get("Content-Type")
		if contentType != "text/csv" {
			t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
		}
	})
}
