package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate response fields
		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got: %v", response["status"])
		}

		if service, ok := response["service"].(string); !ok || service != serviceName {
			t.Errorf("Expected service '%s', got: %v", serviceName, response["service"])
		}

		if version, ok := response["version"].(string); !ok || version != apiVersion {
			t.Errorf("Expected version '%s', got: %v", apiVersion, response["version"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp field in response")
		}
	})
}

// TestListProfiles tests the list profiles endpoint
func TestListProfiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		profiles, ok := response["profiles"].([]interface{})
		if !ok {
			t.Fatalf("Expected profiles array, got: %T", response["profiles"])
		}

		if len(profiles) != 0 {
			t.Errorf("Expected empty profiles list, got %d profiles", len(profiles))
		}

		count, ok := response["count"].(float64)
		if !ok || int(count) != 0 {
			t.Errorf("Expected count 0, got: %v", response["count"])
		}
	})

	t.Run("WithProfiles", func(t *testing.T) {
		// Create test profiles
		createTestProfile(t, env, "test-profile-1", "inactive")
		createTestProfile(t, env, "test-profile-2", "inactive")
		defer cleanupProfiles(env)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		profiles, ok := response["profiles"].([]interface{})
		if !ok {
			t.Fatalf("Expected profiles array, got: %T", response["profiles"])
		}

		if len(profiles) != 2 {
			t.Errorf("Expected 2 profiles, got %d", len(profiles))
		}

		count, ok := response["count"].(float64)
		if !ok || int(count) != 2 {
			t.Errorf("Expected count 2, got: %v", response["count"])
		}
	})
}

// TestGetProfile tests the get profile endpoint
func TestGetProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		profile := createTestProfile(t, env, "test-profile", "inactive")
		defer cleanupProfiles(env)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles/test-profile",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate profile fields
		if name, ok := response["name"].(string); !ok || name != profile.Name {
			t.Errorf("Expected name '%s', got: %v", profile.Name, response["name"])
		}

		if id, ok := response["id"].(string); !ok || id != profile.ID {
			t.Errorf("Expected id '%s', got: %v", profile.ID, response["id"])
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles/nonexistent",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusNotFound, "not found")
	})

	t.Run("EmptyName", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles/",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// This should return 404 due to route not matching
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
	})
}

// TestCreateProfile tests the create profile endpoint
func TestCreateProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		defer cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name":         "new-profile",
			"display_name": "New Test Profile",
			"description":  "A new test profile",
			"resources":    []string{"postgres", "redis"},
			"scenarios":    []string{"test-scenario"},
			"metadata": map[string]interface{}{
				"key": "value",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body:   profileData,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusCreated)

		// Validate created profile
		if name, ok := response["name"].(string); !ok || name != "new-profile" {
			t.Errorf("Expected name 'new-profile', got: %v", response["name"])
		}

		if displayName, ok := response["display_name"].(string); !ok || displayName != "New Test Profile" {
			t.Errorf("Expected display_name 'New Test Profile', got: %v", response["display_name"])
		}

		if status, ok := response["status"].(string); !ok || status != "inactive" {
			t.Errorf("Expected status 'inactive', got: %v", response["status"])
		}

		// Verify ID was generated
		if _, ok := response["id"].(string); !ok {
			t.Error("Expected id field in response")
		}
	})

	t.Run("MissingName", func(t *testing.T) {
		profileData := map[string]interface{}{
			"display_name": "No Name Profile",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body:   profileData,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "required")
	})

	t.Run("DuplicateName", func(t *testing.T) {
		createTestProfile(t, env, "duplicate-test", "inactive")
		defer cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name":         "duplicate-test",
			"display_name": "Duplicate",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body:   profileData,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusConflict, "already exists")
	})

	t.Run("EmptyName", func(t *testing.T) {
		profileData := map[string]interface{}{
			"name":         "",
			"display_name": "Empty Name",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body:   profileData,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "required")
	})
}

// TestUpdateProfile tests the update profile endpoint
func TestUpdateProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		createTestProfile(t, env, "update-test", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{
			"display_name": "Updated Display Name",
			"description":  "Updated description",
			"resources":    []string{"postgres", "redis", "qdrant"},
		}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/profiles/update-test",
			Body:   updates,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate updates
		if displayName, ok := response["display_name"].(string); !ok || displayName != "Updated Display Name" {
			t.Errorf("Expected display_name 'Updated Display Name', got: %v", response["display_name"])
		}

		if description, ok := response["description"].(string); !ok || description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got: %v", response["description"])
		}

		resources, ok := response["resources"].([]interface{})
		if !ok || len(resources) != 3 {
			t.Errorf("Expected 3 resources, got: %v", response["resources"])
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		updates := map[string]interface{}{
			"display_name": "Updated",
		}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/profiles/nonexistent",
			Body:   updates,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusNotFound, "not found")
	})

	t.Run("EmptyUpdate", func(t *testing.T) {
		profile := createTestProfile(t, env, "empty-update-test", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/profiles/empty-update-test",
			Body:   updates,
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Should return unchanged profile
		if name, ok := response["name"].(string); !ok || name != profile.Name {
			t.Errorf("Expected name '%s', got: %v", profile.Name, response["name"])
		}
	})
}

// TestDeleteProfile tests the delete profile endpoint
func TestDeleteProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		createTestProfile(t, env, "delete-test", "inactive")
		defer cleanupProfiles(env)

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/profiles/delete-test",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate success message
		if message, ok := response["message"].(string); !ok || !containsIgnoreCase(message, "deleted") {
			t.Errorf("Expected deletion message, got: %v", response["message"])
		}

		// Verify profile was deleted
		getReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles/delete-test",
		}

		getRr, _ := makeHTTPRequest(env.Router, getReq)
		if getRr.Code != http.StatusNotFound {
			t.Error("Profile should not exist after deletion")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/profiles/nonexistent",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusNotFound, "not found")
	})

	t.Run("ActiveProfile", func(t *testing.T) {
		profile := createTestProfile(t, env, "active-delete-test", "inactive")
		defer cleanupProfiles(env)

		// Set profile as active
		err := env.Service.profileManager.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/profiles/active-delete-test",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusConflict, "cannot delete active")
	})
}

// TestGetStatus tests the status endpoint
func TestGetStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("NoActiveProfile", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate response fields
		if service, ok := response["service"].(string); !ok || service != serviceName {
			t.Errorf("Expected service '%s', got: %v", serviceName, response["service"])
		}

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got: %v", response["status"])
		}

		// active_profile should be null when no profile is active
		if response["active_profile"] != nil {
			t.Errorf("Expected null active_profile, got: %v", response["active_profile"])
		}
	})

	t.Run("WithActiveProfile", func(t *testing.T) {
		profile := createTestProfile(t, env, "status-test", "inactive")
		defer cleanupProfiles(env)

		// Set profile as active
		err := env.Service.profileManager.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate active profile is present
		activeProfile, ok := response["active_profile"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected active_profile object, got: %T", response["active_profile"])
		}

		if name, ok := activeProfile["name"].(string); !ok || name != "status-test" {
			t.Errorf("Expected active profile name 'status-test', got: %v", activeProfile["name"])
		}

		// Validate resource and scenario counts
		resourceCount, ok := response["resource_count"].(float64)
		if !ok || int(resourceCount) != 2 { // postgres and redis from createTestProfile
			t.Errorf("Expected resource_count 2, got: %v", response["resource_count"])
		}

		scenarioCount, ok := response["scenario_count"].(float64)
		if !ok || int(scenarioCount) != 1 { // test-scenario from createTestProfile
			t.Errorf("Expected scenario_count 1, got: %v", response["scenario_count"])
		}
	})
}

// TestHTTPError tests the error response helper
func TestHTTPError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ErrorResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()

		HTTPError(rr, "Test error message", http.StatusBadRequest, nil)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		if errorMsg, ok := response["error"].(string); !ok || errorMsg != "Test error message" {
			t.Errorf("Expected error 'Test error message', got: %v", response["error"])
		}

		if status, ok := response["status"].(float64); !ok || int(status) != http.StatusBadRequest {
			t.Errorf("Expected status %d, got: %v", http.StatusBadRequest, response["status"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp in response")
		}
	})
}

// TestNewLogger tests logger creation
func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Error("Expected logger to be created")
	}
	if logger.Logger == nil {
		t.Error("Expected internal logger to be initialized")
	}
}

// TestLoggerMethods tests logger methods
func TestLoggerMethods(t *testing.T) {
	logger := NewLogger()

	t.Run("Info", func(t *testing.T) {
		// Should not panic
		logger.Info("Test info message")
	})

	t.Run("Error", func(t *testing.T) {
		// Should not panic
		logger.Error("Test error message", nil)
	})
}

// TestNewOrchestratorService tests service creation
func TestNewOrchestratorService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	service := NewOrchestratorService(env.DB)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.db == nil {
		t.Error("Expected db to be set")
	}

	if service.logger == nil {
		t.Error("Expected logger to be set")
	}

	if service.profileManager == nil {
		t.Error("Expected profileManager to be set")
	}

	if service.orchestratorManager == nil {
		t.Error("Expected orchestratorManager to be set")
	}
}

// TestGetResourcePort tests the port registry helper
func TestGetResourcePort(t *testing.T) {
	t.Run("KnownResource", func(t *testing.T) {
		port := getResourcePort("postgres")
		if port == "" {
			t.Error("Expected port to be returned for postgres")
		}
	})

	t.Run("UnknownResource", func(t *testing.T) {
		port := getResourcePort("nonexistent-resource-12345")
		// Port may be empty string or a default fallback - either is acceptable
		_ = port
	})
}
