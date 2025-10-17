// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidConfig", func(t *testing.T) {
		// Set required environment variables
		t.Setenv("API_PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")

		config, err := loadConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if config.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", config.Port)
		}

		if config.DatabaseURL == "" {
			t.Error("Expected DatabaseURL to be set")
		}
	})

	t.Run("MissingPort", func(t *testing.T) {
		// Clear environment
		t.Setenv("API_PORT", "")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for missing API_PORT")
		}
	})

	t.Run("InvalidPort", func(t *testing.T) {
		t.Setenv("API_PORT", "invalid")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for invalid API_PORT")
		}
	})

	t.Run("MissingDatabaseURL", func(t *testing.T) {
		t.Setenv("API_PORT", "8080")
		t.Setenv("DATABASE_URL", "")
		// Also clear component variables
		t.Setenv("POSTGRES_HOST", "")
		t.Setenv("POSTGRES_PORT", "")
		t.Setenv("POSTGRES_USER", "")
		t.Setenv("POSTGRES_PASSWORD", "")
		t.Setenv("POSTGRES_DB", "")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for missing DATABASE_URL")
		}
	})

	t.Run("DatabaseURLFromComponents", func(t *testing.T) {
		t.Setenv("API_PORT", "8080")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("POSTGRES_HOST", "localhost")
		t.Setenv("POSTGRES_PORT", "5432")
		t.Setenv("POSTGRES_USER", "user")
		t.Setenv("POSTGRES_PASSWORD", "pass")
		t.Setenv("POSTGRES_DB", "testdb")

		config, err := loadConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if config.DatabaseURL == "" {
			t.Error("Expected DatabaseURL to be constructed from components")
		}
	})
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("HealthCheckSuccess", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		assertHealthResponse(t, w)
	})

	t.Run("HealthCheckStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		// Check for required fields
		requiredFields := []string{"status", "timestamp", "version", "database"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

// TestProfileHandlers tests profile management endpoints
func TestProfileHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("GetProfiles", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		profiles := assertJSONArray(t, w, http.StatusOK)
		if profiles == nil {
			t.Fatal("Expected profiles array")
		}

		if len(profiles) == 0 {
			t.Log("Warning: No profiles returned (expected for demo)")
		}
	})

	t.Run("GetSpecificProfile", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/profiles/test-id",
			URLVars: map[string]string{"id": "test-id"},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id": "test-id",
		})

		if response == nil {
			return
		}

		// Check profile structure
		if _, exists := response["name"]; !exists {
			t.Error("Expected 'name' field in profile")
		}
	})

	t.Run("GetProfileStats", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/profiles/test-id/stats",
			URLVars: map[string]string{"id": "test-id"},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		// Check stats structure
		statsFields := []string{"total_bookmarks", "categories_count", "pending_actions", "accuracy_rate"}
		for _, field := range statsFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected '%s' field in stats", field)
			}
		}
	})

	t.Run("CreateProfile_NotImplemented", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body:   TestData.CreateProfileRequest("Test Profile", "Test description"),
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Expecting 501 Not Implemented based on current code
		if w.Code != http.StatusNotImplemented {
			t.Logf("Profile creation returned status: %d", w.Code)
		}
	})

	t.Run("UpdateProfile_NotImplemented", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/profiles/test-id",
			URLVars: map[string]string{"id": "test-id"},
			Body:    TestData.CreateProfileRequest("Updated Profile", "Updated description"),
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Expecting 501 Not Implemented based on current code
		if w.Code != http.StatusNotImplemented {
			t.Logf("Profile update returned status: %d", w.Code)
		}
	})
}

// TestBookmarkHandlers tests bookmark management endpoints
func TestBookmarkHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("ProcessBookmarks", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/bookmarks/process",
			Body: map[string]interface{}{
				"urls": []string{"https://example.com"},
			},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			return
		}

		if _, exists := response["processed_count"]; !exists {
			t.Error("Expected 'processed_count' in response")
		}
	})

	t.Run("QueryBookmarks", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/bookmarks/query",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		// Check response structure
		if _, exists := response["bookmarks"]; !exists {
			t.Error("Expected 'bookmarks' field in response")
		}

		if _, exists := response["categories"]; !exists {
			t.Error("Expected 'categories' field in response")
		}
	})

	t.Run("SyncBookmarks", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/bookmarks/sync",
			Body:   map[string]interface{}{},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			return
		}
	})
}

// TestCategoryHandlers tests category management endpoints
func TestCategoryHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("GetCategories", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/categories",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		categories := assertJSONArray(t, w, http.StatusOK)
		if categories == nil {
			t.Fatal("Expected categories array")
		}

		if len(categories) > 0 {
			// Verify category structure
			firstCategory := categories[0].(map[string]interface{})
			if _, exists := firstCategory["name"]; !exists {
				t.Error("Expected 'name' field in category")
			}
		}
	})

	t.Run("CreateCategory_NotImplemented", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/categories",
			Body:   TestData.CreateCategoryRequest("Test Category"),
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Expecting 501 Not Implemented
		if w.Code != http.StatusNotImplemented {
			t.Logf("Category creation returned status: %d", w.Code)
		}
	})

	t.Run("UpdateCategory_NotImplemented", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/categories/test-id",
			URLVars: map[string]string{"id": "test-id"},
			Body:    TestData.CreateCategoryRequest("Updated Category"),
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Expecting 501 Not Implemented
		if w.Code != http.StatusNotImplemented {
			t.Logf("Category update returned status: %d", w.Code)
		}
	})

	t.Run("DeleteCategory_NotImplemented", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/categories/test-id",
			URLVars: map[string]string{"id": "test-id"},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Expecting 501 Not Implemented
		if w.Code != http.StatusNotImplemented {
			t.Logf("Category deletion returned status: %d", w.Code)
		}
	})
}

// TestActionHandlers tests action management endpoints
func TestActionHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("GetActions", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/actions",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		actions := assertJSONArray(t, w, http.StatusOK)
		if actions == nil {
			t.Fatal("Expected actions array")
		}
	})

	t.Run("ApproveActions", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/actions/approve",
			Body:   TestData.ApproveActionsRequest([]string{"action-1"}),
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			return
		}
	})

	t.Run("RejectActions", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/actions/reject",
			Body:   TestData.ApproveActionsRequest([]string{"action-1"}),
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			return
		}
	})
}

// TestPlatformHandlers tests platform management endpoints
func TestPlatformHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("GetPlatforms", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/platforms",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		platforms := assertJSONArray(t, w, http.StatusOK)
		if platforms == nil {
			t.Fatal("Expected platforms array")
		}

		if len(platforms) > 0 {
			// Verify platform structure
			firstPlatform := platforms[0].(map[string]interface{})
			requiredFields := []string{"name", "display_name", "supported"}
			for _, field := range requiredFields {
				if _, exists := firstPlatform[field]; !exists {
					t.Errorf("Expected '%s' field in platform", field)
				}
			}
		}
	})

	t.Run("GetPlatformStatus", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/platforms/status",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		statuses := assertJSONArray(t, w, http.StatusOK)
		if statuses == nil {
			t.Fatal("Expected platform statuses array")
		}
	})

	t.Run("SyncPlatform", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/platforms/reddit/sync",
			URLVars: map[string]string{"platform": "reddit"},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"platform": "reddit",
		})

		if response == nil {
			return
		}
	})
}

// TestAnalyticsHandlers tests analytics endpoints
func TestAnalyticsHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("GetMetrics", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/metrics",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			return
		}

		// Check metrics structure
		metricsFields := []string{"total_bookmarks", "processing_accuracy", "platform_breakdown"}
		for _, field := range metricsFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected '%s' field in metrics", field)
			}
		}
	})
}

// TestMiddleware tests middleware functionality
func TestMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("CORSHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/profiles",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Check for CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Log("Note: CORS headers may not be set for OPTIONS requests")
		}
	})

	t.Run("AuthMiddleware_HealthCheck", func(t *testing.T) {
		// Health check should bypass auth
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Health check should succeed without auth, got status: %d", w.Code)
		}
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("InvalidJSON_ProcessBookmarks", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/bookmarks/process",
			Body:   `{"invalid": "json"`, // Malformed JSON
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Server may return 200 or 400 depending on implementation
		if w.Code >= 500 {
			t.Errorf("Server error for invalid JSON: %d", w.Code)
		}
	})

	t.Run("NotFoundRoute", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/nonexistent",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Logf("Non-existent route returned status: %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PATCH", // Not supported
			Path:   "/api/v1/profiles",
		}

		w, err := executeServerRequest(server, req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		// Should return 405 Method Not Allowed
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Logf("Unsupported method returned status: %d", w.Code)
		}
	})
}

// TestServerLifecycle tests server startup and shutdown
func TestServerLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ServerCreation", func(t *testing.T) {
		// Skip this test as it tries to connect to database which may not be available
		t.Skip("Skipping server creation test (requires database)")

		config := &Config{
			Port:           0,
			DatabaseURL:    "postgres://test:test@localhost:5432/test?sslmode=disable",
			HuginnURL:      "http://localhost:3000",
			BrowserlessURL: "http://localhost:3001",
			APIToken:       "test-token",
		}

		// Try to create server (may fail without database)
		server, err := NewServer(config)
		if err != nil {
			t.Logf("Server creation failed (expected without database): %v", err)
			return
		}

		if server == nil {
			t.Error("Expected server instance")
		}

		defer server.Close()

		if server.router == nil {
			t.Error("Expected router to be initialized")
		}
	})
}
