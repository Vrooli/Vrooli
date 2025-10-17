package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/health", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", data["status"])
		}

		if data["service"] != "Data Tools API" {
			t.Errorf("Expected service 'Data Tools API', got '%v'", data["service"])
		}

		if data["database"] != "connected" {
			t.Errorf("Expected database 'connected', got '%v'", data["database"])
		}
	})

	t.Run("NoAuthRequired", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/health", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health check should not require auth, got status %d", w.Code)
		}
	})
}

func TestResourceCRUD(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	var createdID string

	t.Run("CreateResource", func(t *testing.T) {
		createBody := map[string]interface{}{
			"name":        "Test Resource",
			"description": "A test resource",
			"config":      map[string]interface{}{"key": "value"},
		}

		req := makeHTTPRequest("POST", "/api/v1/resources/create", createBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		createdID, ok = data["id"].(string)
		if !ok || createdID == "" {
			t.Fatal("Expected non-empty id in response")
		}
	})

	t.Run("ListResources", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/resources/list", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatal("Expected array in response data")
		}

		if len(data) == 0 {
			t.Error("Expected at least one resource in list")
		}
	})

	t.Run("GetResource", func(t *testing.T) {
		if createdID == "" {
			t.Skip("Skipping because createdID is empty")
		}

		req := makeHTTPRequest("GET", "/api/v1/resources/get?id="+createdID, nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["name"] != "Test Resource" {
			t.Errorf("Expected name 'Test Resource', got '%v'", data["name"])
		}
	})

	t.Run("UpdateResource", func(t *testing.T) {
		if createdID == "" {
			t.Skip("Skipping because createdID is empty")
		}

		updateBody := map[string]interface{}{
			"name":        "Updated Resource",
			"description": "An updated resource",
			"config":      map[string]interface{}{"key": "new_value"},
		}

		req := makeHTTPRequest("POST", "/api/v1/resources/update?id="+createdID, updateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("DeleteResource", func(t *testing.T) {
		if createdID == "" {
			t.Skip("Skipping because createdID is empty")
		}

		req := makeHTTPRequest("POST", "/api/v1/resources/delete?id="+createdID, nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["deleted"] != true {
			t.Error("Expected deleted=true in response")
		}
	})
}

func TestResourceErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("GetNonExistentResource", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/resources/get?id=00000000-0000-0000-0000-000000000000", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "resource not found")
	})

	t.Run("CreateResourceWithInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/resources/create", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 400 or 500, got %d", w.Code)
		}
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/resources/list", nil, "invalid-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
	})

	t.Run("UpdateNonExistentResource", func(t *testing.T) {
		updateBody := map[string]interface{}{
			"name": "Updated",
		}

		req := makeHTTPRequest("POST", "/api/v1/resources/update?id=00000000-0000-0000-0000-000000000000", updateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "resource not found")
	})

	t.Run("DeleteNonExistentResource", func(t *testing.T) {
		req := makeHTTPRequest("POST", "/api/v1/resources/delete?id=00000000-0000-0000-0000-000000000000", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "resource not found")
	})
}

func TestWorkflowExecution(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	var executionID string

	t.Run("ExecuteWorkflow", func(t *testing.T) {
		executeBody := map[string]interface{}{
			"workflow_id": "test-workflow",
			"input": map[string]interface{}{
				"param1": "value1",
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/execute", executeBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusAccepted)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		executionID, ok = data["execution_id"].(string)
		if !ok || executionID == "" {
			t.Fatal("Expected non-empty execution_id in response")
		}

		if data["status"] != "pending" {
			t.Errorf("Expected status 'pending', got '%v'", data["status"])
		}
	})

	t.Run("ListExecutions", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/executions", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		// Check that data is an array (might be empty initially)
		if data, ok := response["data"].([]interface{}); ok {
			_ = data // Valid array response
		} else if response["data"] == nil {
			// Also acceptable (no executions yet)
		} else {
			t.Error("Expected array or null in response data")
		}
	})

	t.Run("GetExecution", func(t *testing.T) {
		// Create a test execution first
		id := createTestExecution(t, env.DB)

		req := makeHTTPRequest("GET", "/api/v1/executions/get?id="+id, nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["status"] != "completed" {
			t.Errorf("Expected status 'completed', got '%v'", data["status"])
		}
	})

	t.Run("GetNonExistentExecution", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/executions/get?id=00000000-0000-0000-0000-000000000000", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, "execution not found")
	})
}

func TestDocsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("GetDocs", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/docs", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["name"] != "Data Tools API" {
			t.Errorf("Expected name 'Data Tools API', got '%v'", data["name"])
		}

		endpoints, ok := data["endpoints"].([]interface{})
		if !ok {
			t.Fatal("Expected endpoints array in docs")
		}

		if len(endpoints) == 0 {
			t.Error("Expected at least one endpoint in docs")
		}
	})

	t.Run("DocsNoAuthRequired", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/docs", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Docs should not require auth, got status %d", w.Code)
		}
	})
}

func TestMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("AuthMiddlewareAllowsValidToken", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/resources/list", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code == http.StatusUnauthorized {
			t.Error("Valid token should be allowed")
		}
	})

	t.Run("AuthMiddlewareRejectsInvalidToken", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/resources/list", nil, "wrong-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for invalid token, got %d", w.Code)
		}
	})

	t.Run("AuthMiddlewareSkipsHealthCheck", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/health", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health check should not require auth, got %d", w.Code)
		}
	})

	t.Run("AuthMiddlewareSkipsDocs", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/docs", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Docs should not require auth, got %d", w.Code)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("EmptyBody", func(t *testing.T) {
		req := makeHTTPRequest("POST", "/api/v1/resources/create", map[string]interface{}{}, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		// Should handle empty body gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code %d for empty body", w.Code)
		}
	})

	t.Run("LargePayload", func(t *testing.T) {
		// Create a large config object
		largeConfig := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeConfig[string(rune('a'+i%26))+string(rune(i))] = "value"
		}

		createBody := map[string]interface{}{
			"name":        "Large Resource",
			"description": "Resource with large config",
			"config":      largeConfig,
		}

		req := makeHTTPRequest("POST", "/api/v1/resources/create", createBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Expected 201 or 200 for large payload, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInData", func(t *testing.T) {
		createBody := map[string]interface{}{
			"name":        "Test <script>alert('xss')</script>",
			"description": "SQL'; DROP TABLE resources; --",
			"config":      map[string]interface{}{"key": "value\n\t\r"},
		}

		req := makeHTTPRequest("POST", "/api/v1/resources/create", createBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		// Should handle special characters without crashing
		if w.Code >= 500 {
			t.Errorf("Server error handling special characters: %d", w.Code)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		// Test concurrent requests
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				req := makeHTTPRequest("GET", "/health", nil, "")
				w := httptest.NewRecorder()

				env.Server.router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Concurrent request %d failed with status %d", index, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for concurrent requests")
			}
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getEnv", func(t *testing.T) {
		// Test with existing env var
		result := getEnv("PATH", "default")
		if result == "default" {
			t.Error("Should return actual PATH value, not default")
		}

		// Test with non-existing env var
		result = getEnv("NONEXISTENT_VAR_12345", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})

	t.Run("sendJSON", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		w := httptest.NewRecorder()
		env.Server.sendJSON(w, http.StatusOK, map[string]interface{}{"test": "data"})

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response Response
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("sendError", func(t *testing.T) {
		env := setupTestDirectory(t)
		defer env.Cleanup()

		w := httptest.NewRecorder()
		env.Server.sendError(w, http.StatusBadRequest, "test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response Response
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Success {
			t.Error("Expected success=false")
		}

		if response.Error != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", response.Error)
		}
	})
}
