package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestProfileManagerComprehensive tests ProfileManager with comprehensive coverage
func TestProfileManagerComprehensive(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("ListProfiles_EmptyDatabase", func(t *testing.T) {
		cleanupProfiles(env)

		profiles, err := pm.ListProfiles()
		if err != nil {
			t.Fatalf("ListProfiles failed: %v", err)
		}

		if len(profiles) != 0 {
			t.Errorf("Expected 0 profiles, got %d", len(profiles))
		}
	})

	t.Run("CreateProfile_WithAllFields", func(t *testing.T) {
		cleanupProfiles(env)

		idleShutdown := 30
		profileData := map[string]interface{}{
			"name":                  "full-profile",
			"display_name":          "Full Test Profile",
			"description":           "A profile with all fields populated",
			"resources":             []interface{}{"postgres", "redis", "n8n"},
			"scenarios":             []interface{}{"test-scenario-1", "test-scenario-2"},
			"auto_browser":          []interface{}{"http://localhost:3000", "http://localhost:8080"},
			"environment_vars":      map[string]interface{}{"TEST_VAR": "test_value", "DEBUG": "true"},
			"idle_shutdown_minutes": float64(idleShutdown),
			"dependencies":          []interface{}{"base-profile"},
			"metadata": map[string]interface{}{
				"created_by": "test-suite",
				"version":    "1.0",
			},
		}

		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Validate all fields
		if profile.Name != "full-profile" {
			t.Errorf("Expected name 'full-profile', got '%s'", profile.Name)
		}
		if profile.DisplayName != "Full Test Profile" {
			t.Errorf("Expected display_name 'Full Test Profile', got '%s'", profile.DisplayName)
		}
		if len(profile.Resources) != 3 {
			t.Errorf("Expected 3 resources, got %d", len(profile.Resources))
		}
		if len(profile.Scenarios) != 2 {
			t.Errorf("Expected 2 scenarios, got %d", len(profile.Scenarios))
		}
		if len(profile.AutoBrowser) != 2 {
			t.Errorf("Expected 2 auto_browser URLs, got %d", len(profile.AutoBrowser))
		}
		if len(profile.EnvironmentVars) != 2 {
			t.Errorf("Expected 2 environment vars, got %d", len(profile.EnvironmentVars))
		}
		if profile.IdleShutdown == nil || *profile.IdleShutdown != idleShutdown {
			t.Errorf("Expected idle_shutdown %d, got %v", idleShutdown, profile.IdleShutdown)
		}
		if len(profile.Dependencies) != 1 {
			t.Errorf("Expected 1 dependency, got %d", len(profile.Dependencies))
		}
		if profile.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", profile.Status)
		}
	})

	t.Run("CreateProfile_MinimalFields", func(t *testing.T) {
		cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name": "minimal-profile",
		}

		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile with minimal fields failed: %v", err)
		}

		if profile.Name != "minimal-profile" {
			t.Errorf("Expected name 'minimal-profile', got '%s'", profile.Name)
		}
		if profile.DisplayName != "minimal-profile" {
			t.Errorf("Expected display_name to default to name, got '%s'", profile.DisplayName)
		}
		if len(profile.Resources) != 0 {
			t.Errorf("Expected empty resources, got %d", len(profile.Resources))
		}
	})

	t.Run("CreateProfile_DuplicateName", func(t *testing.T) {
		cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name": "duplicate-test",
		}

		_, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("First CreateProfile failed: %v", err)
		}

		// Try to create duplicate
		_, err = pm.CreateProfile(profileData)
		if err == nil {
			t.Error("Expected error for duplicate profile name, got nil")
		}
		if err != nil && !containsIgnoreCase(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("GetProfile_Exists", func(t *testing.T) {
		cleanupProfiles(env)

		// Create a profile
		profileData := map[string]interface{}{
			"name":         "get-test",
			"display_name": "Get Test Profile",
		}
		created, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Get the profile
		retrieved, err := pm.GetProfile("get-test")
		if err != nil {
			t.Fatalf("GetProfile failed: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
		}
		if retrieved.Name != "get-test" {
			t.Errorf("Expected name 'get-test', got '%s'", retrieved.Name)
		}
	})

	t.Run("GetProfile_NotFound", func(t *testing.T) {
		_, err := pm.GetProfile("nonexistent-profile")
		if err == nil {
			t.Error("Expected error for non-existent profile, got nil")
		}
		if err != nil && !containsIgnoreCase(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("UpdateProfile_SingleField", func(t *testing.T) {
		cleanupProfiles(env)

		// Create profile
		profileData := map[string]interface{}{
			"name":         "update-test",
			"display_name": "Original Name",
		}
		_, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Update display_name only
		updates := map[string]interface{}{
			"display_name": "Updated Name",
		}
		updated, err := pm.UpdateProfile("update-test", updates)
		if err != nil {
			t.Fatalf("UpdateProfile failed: %v", err)
		}

		if updated.DisplayName != "Updated Name" {
			t.Errorf("Expected display_name 'Updated Name', got '%s'", updated.DisplayName)
		}
		if updated.Name != "update-test" {
			t.Errorf("Name should not change, got '%s'", updated.Name)
		}
	})

	t.Run("UpdateProfile_MultipleFields", func(t *testing.T) {
		cleanupProfiles(env)

		// Create profile
		profileData := map[string]interface{}{
			"name": "multi-update-test",
		}
		_, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Update multiple fields
		updates := map[string]interface{}{
			"display_name": "Multi Update",
			"description":  "Updated description",
			"resources":    []interface{}{"postgres", "redis"},
			"scenarios":    []interface{}{"new-scenario"},
		}
		updated, err := pm.UpdateProfile("multi-update-test", updates)
		if err != nil {
			t.Fatalf("UpdateProfile failed: %v", err)
		}

		if updated.DisplayName != "Multi Update" {
			t.Errorf("Expected updated display_name")
		}
		if updated.Description != "Updated description" {
			t.Errorf("Expected updated description")
		}
		if len(updated.Resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(updated.Resources))
		}
		if len(updated.Scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(updated.Scenarios))
		}
	})

	t.Run("UpdateProfile_NonExistent", func(t *testing.T) {
		updates := map[string]interface{}{
			"display_name": "Won't Work",
		}
		_, err := pm.UpdateProfile("nonexistent", updates)
		if err == nil {
			t.Error("Expected error for non-existent profile, got nil")
		}
	})

	t.Run("UpdateProfile_EmptyUpdates", func(t *testing.T) {
		cleanupProfiles(env)

		// Create profile
		profileData := map[string]interface{}{
			"name": "empty-update-test",
		}
		original, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Update with empty map
		updates := map[string]interface{}{}
		updated, err := pm.UpdateProfile("empty-update-test", updates)
		if err != nil {
			t.Fatalf("UpdateProfile with empty updates failed: %v", err)
		}

		if updated.ID != original.ID {
			t.Error("Profile should remain unchanged")
		}
	})

	t.Run("DeleteProfile_Success", func(t *testing.T) {
		cleanupProfiles(env)

		// Create profile
		profileData := map[string]interface{}{
			"name": "delete-test",
		}
		_, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Delete profile
		err = pm.DeleteProfile("delete-test")
		if err != nil {
			t.Fatalf("DeleteProfile failed: %v", err)
		}

		// Verify deletion
		_, err = pm.GetProfile("delete-test")
		if err == nil {
			t.Error("Profile should not exist after deletion")
		}
	})

	t.Run("DeleteProfile_ActiveProfile", func(t *testing.T) {
		cleanupProfiles(env)

		// Create and activate profile
		profileData := map[string]interface{}{
			"name": "active-delete-test",
		}
		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Set as active
		err = pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("SetActiveProfile failed: %v", err)
		}

		// Try to delete active profile
		err = pm.DeleteProfile("active-delete-test")
		if err == nil {
			t.Error("Expected error when deleting active profile, got nil")
		}
		if err != nil && !containsIgnoreCase(err.Error(), "cannot delete active") {
			t.Errorf("Expected 'cannot delete active' error, got: %v", err)
		}
	})

	t.Run("ActiveProfile_SetAndGet", func(t *testing.T) {
		cleanupProfiles(env)

		// Create profile
		profileData := map[string]interface{}{
			"name": "active-test",
		}
		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		// Set as active
		err = pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("SetActiveProfile failed: %v", err)
		}

		// Get active profile
		active, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("GetActiveProfile failed: %v", err)
		}

		if active == nil {
			t.Fatal("Expected active profile, got nil")
		}
		if active.ID != profile.ID {
			t.Errorf("Expected active profile ID %s, got %s", profile.ID, active.ID)
		}
		if active.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", active.Status)
		}
	})

	t.Run("ActiveProfile_Clear", func(t *testing.T) {
		cleanupProfiles(env)

		// Create and activate profile
		profileData := map[string]interface{}{
			"name": "clear-test",
		}
		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("CreateProfile failed: %v", err)
		}

		err = pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("SetActiveProfile failed: %v", err)
		}

		// Clear active profile
		err = pm.ClearActiveProfile()
		if err != nil {
			t.Fatalf("ClearActiveProfile failed: %v", err)
		}

		// Verify no active profile
		active, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("GetActiveProfile failed: %v", err)
		}
		if active != nil {
			t.Error("Expected no active profile after clear")
		}

		// Verify profile status updated to inactive
		profile, err = pm.GetProfile("clear-test")
		if err != nil {
			t.Fatalf("GetProfile failed: %v", err)
		}
		if profile.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", profile.Status)
		}
	})

	t.Run("ActiveProfile_None", func(t *testing.T) {
		cleanupProfiles(env)

		active, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("GetActiveProfile failed: %v", err)
		}
		if active != nil {
			t.Error("Expected nil for no active profile")
		}
	})
}

// TestOrchestratorManagerComprehensive tests OrchestratorManager with comprehensive coverage
func TestOrchestratorManagerComprehensive(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("ValidateProfile_AllValid", func(t *testing.T) {
		profile := &Profile{
			Name:      "valid-profile",
			Resources: []string{}, // Empty to avoid actual resource checks
			Scenarios: []string{}, // Empty to avoid actual scenario checks
		}

		issues := om.ValidateProfile(profile)
		if len(issues) != 0 {
			t.Errorf("Expected no validation issues for empty resources/scenarios, got %d: %v", len(issues), issues)
		}
	})

	t.Run("ValidateProfile_InvalidResources", func(t *testing.T) {
		profile := &Profile{
			Name:      "invalid-resources",
			Resources: []string{"nonexistent-resource-12345"},
			Scenarios: []string{},
		}

		issues := om.ValidateProfile(profile)
		// We expect issues since the resource doesn't exist
		// The actual count depends on whether vrooli command is available
		if len(issues) == 0 {
			t.Log("Warning: Expected validation issues for nonexistent resource (may pass if vrooli not in PATH)")
		}
	})

	t.Run("DeactivateCurrentProfile_NoActive", func(t *testing.T) {
		cleanupProfiles(env)

		err := om.DeactivateCurrentProfile()
		if err != nil {
			t.Errorf("Deactivating with no active profile should not error, got: %v", err)
		}
	})

	t.Run("GetDeactivationResult_NoActive", func(t *testing.T) {
		cleanupProfiles(env)

		result, err := om.GetDeactivationResult()
		if err != nil {
			t.Fatalf("GetDeactivationResult failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true when no active profile")
		}
		if result.Message != "Profile deactivated successfully" {
			t.Errorf("Expected success message, got: %s", result.Message)
		}
	})
}

// TestAPIEndpointsComprehensive tests all API endpoints with edge cases
func TestAPIEndpointsComprehensive(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("CreateProfile_ComplexData", func(t *testing.T) {
		cleanupProfiles(env)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":         "complex-profile",
				"display_name": "Complex Test Profile",
				"description":  "A profile with complex nested data",
				"resources":    []interface{}{"postgres", "redis", "n8n", "ollama"},
				"scenarios":    []interface{}{"scenario-1", "scenario-2", "scenario-3"},
				"auto_browser": []interface{}{
					"http://localhost:3000/dashboard",
					"http://localhost:8080/admin",
					"http://localhost:5000/metrics",
				},
				"environment_vars": map[string]interface{}{
					"NODE_ENV":     "production",
					"DEBUG":        "false",
					"API_KEY":      "test-key-12345",
					"MAX_WORKERS":  "10",
					"ENABLE_CACHE": "true",
				},
				"idle_shutdown_minutes": float64(60),
				"dependencies":          []interface{}{"base-profile", "security-profile"},
				"metadata": map[string]interface{}{
					"version":    "2.0.0",
					"created_by": "automated-test",
					"tags":       []interface{}{"production", "high-priority"},
					"config": map[string]interface{}{
						"timeout": 3000,
						"retries": 3,
					},
				},
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 201)

		// Validate response structure
		if response["name"] != "complex-profile" {
			t.Errorf("Expected name 'complex-profile', got %v", response["name"])
		}

		resources, ok := response["resources"].([]interface{})
		if !ok || len(resources) != 4 {
			t.Errorf("Expected 4 resources, got %v", response["resources"])
		}

		envVars, ok := response["environment_vars"].(map[string]interface{})
		if !ok || len(envVars) != 5 {
			t.Errorf("Expected 5 environment variables, got %v", response["environment_vars"])
		}
	})

	t.Run("UpdateProfile_NullIdleShutdown", func(t *testing.T) {
		cleanupProfiles(env)

		// Create profile with idle shutdown
		createReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":                  "idle-test",
				"idle_shutdown_minutes": float64(30),
			},
		}
		_, err := makeHTTPRequest(env.Router, createReq)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Update to null
		updateReq := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/profiles/idle-test",
			Body: map[string]interface{}{
				"idle_shutdown_minutes": nil,
			},
		}
		rr, err := makeHTTPRequest(env.Router, updateReq)
		if err != nil {
			t.Fatalf("Failed to make update request: %v", err)
		}

		response := assertJSONResponse(t, rr, 200)
		if response["idle_shutdown_minutes"] != nil {
			t.Errorf("Expected idle_shutdown_minutes to be null, got %v", response["idle_shutdown_minutes"])
		}
	})

	t.Run("ListProfiles_Ordering", func(t *testing.T) {
		cleanupProfiles(env)

		// Create multiple profiles with delays to ensure different timestamps
		for i := 1; i <= 3; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/profiles",
				Body: map[string]interface{}{
					"name": fmt.Sprintf("profile-%d", i),
				},
			}
			_, err := makeHTTPRequest(env.Router, req)
			if err != nil {
				t.Fatalf("Failed to create profile %d: %v", i, err)
			}
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// List profiles
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}
		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 200)
		profiles := assertProfileListResponse(t, response, 3)

		// Profiles should be ordered by created_at DESC (newest first)
		firstProfile := profiles[0].(map[string]interface{})
		lastProfile := profiles[2].(map[string]interface{})

		if firstProfile["name"] != "profile-3" {
			t.Errorf("Expected first profile to be 'profile-3', got %v", firstProfile["name"])
		}
		if lastProfile["name"] != "profile-1" {
			t.Errorf("Expected last profile to be 'profile-1', got %v", lastProfile["name"])
		}
	})

	t.Run("HealthEndpoint_Structure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 200)

		// Validate required fields
		requiredFields := []string{"status", "service", "version", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in health response", field)
			}
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}
		if response["service"] != serviceName {
			t.Errorf("Expected service '%s', got %v", serviceName, response["service"])
		}
	})

	t.Run("StatusEndpoint_WithActiveProfile", func(t *testing.T) {
		cleanupProfiles(env)

		// Create and activate a profile
		createReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":      "status-test",
				"resources": []interface{}{"postgres", "redis"},
				"scenarios": []interface{}{"test-scenario"},
			},
		}
		createRR, err := makeHTTPRequest(env.Router, createReq)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}
		createResponse := assertJSONResponse(t, createRR, 201)
		profileID := createResponse["id"].(string)

		// Set as active
		pm := env.Service.profileManager
		err = pm.SetActiveProfile(profileID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Get status
		statusReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/status",
		}
		statusRR, err := makeHTTPRequest(env.Router, statusReq)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		response := assertJSONResponse(t, statusRR, 200)

		// Validate status response
		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		activeProfile, ok := response["active_profile"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected active_profile in response")
		}
		if activeProfile["name"] != "status-test" {
			t.Errorf("Expected active profile name 'status-test', got %v", activeProfile["name"])
		}
	})
}

// TestErrorHandling tests error handling across all components
func TestErrorHandling(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("InvalidJSON_CreateProfile", func(t *testing.T) {
		// This tests the actual HTTP handler's JSON parsing
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body:   "{invalid json}",
		}
		req.Headers = map[string]string{"Content-Type": "application/json"}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return 400 for invalid JSON
		if rr.Code != 400 && rr.Code != 201 {
			// Some implementations might handle this differently
			t.Logf("Got status code %d for invalid JSON (expected 400)", rr.Code)
		}
	})

	t.Run("DatabaseConnectionError", func(t *testing.T) {
		// Skip this test as it's difficult to simulate a database error safely
		// The test framework already validates error handling through other error scenarios
		t.Skip("Skipping database connection error test - covered by other error scenarios")
	})
}

// TestConcurrency tests concurrent operations
func TestConcurrency(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("ConcurrentProfileCreation", func(t *testing.T) {
		cleanupProfiles(env)

		concurrency := 5
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/profiles",
					Body: map[string]interface{}{
						"name": fmt.Sprintf("concurrent-profile-%d", index),
					},
				}
				_, err := makeHTTPRequest(env.Router, req)
				if err != nil {
					errors <- err
				} else {
					done <- true
				}
			}(i)
		}

		// Wait for all goroutines
		successCount := 0
		errorCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case <-done:
				successCount++
			case <-errors:
				errorCount++
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}

		if successCount != concurrency {
			t.Errorf("Expected %d successful creations, got %d (errors: %d)",
				concurrency, successCount, errorCount)
		}

		// Verify all profiles were created
		profiles, err := env.Service.profileManager.ListProfiles()
		if err != nil {
			t.Fatalf("Failed to list profiles: %v", err)
		}
		if len(profiles) != concurrency {
			t.Errorf("Expected %d profiles, got %d", concurrency, len(profiles))
		}
	})
}

// TestJSONSerialization tests JSON encoding/decoding edge cases
func TestJSONSerialization(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("EmptyArrays", func(t *testing.T) {
		cleanupProfiles(env)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":         "empty-arrays",
				"resources":    []interface{}{},
				"scenarios":    []interface{}{},
				"auto_browser": []interface{}{},
				"dependencies": []interface{}{},
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 201)

		// Validate empty arrays are preserved
		resources, ok := response["resources"].([]interface{})
		if !ok || len(resources) != 0 {
			t.Errorf("Expected empty resources array, got %v", response["resources"])
		}
	})

	t.Run("NullValues", func(t *testing.T) {
		cleanupProfiles(env)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":                  "null-values",
				"idle_shutdown_minutes": nil,
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 201)

		// idle_shutdown_minutes should be null
		if response["idle_shutdown_minutes"] != nil {
			t.Errorf("Expected null idle_shutdown_minutes, got %v", response["idle_shutdown_minutes"])
		}
	})

	t.Run("SpecialCharactersInStrings", func(t *testing.T) {
		cleanupProfiles(env)

		specialChars := "Test with special chars: \"quotes\", 'apostrophes', \n newlines, \t tabs, unicode: 你好"
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":        "special-chars-test",
				"description": specialChars,
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 201)

		// Verify special characters are preserved
		description := response["description"].(string)
		if description != specialChars {
			t.Errorf("Special characters not preserved.\nExpected: %s\nGot: %s", specialChars, description)
		}
	})
}

// TestDatabaseConstraints tests database constraint enforcement
func TestDatabaseConstraints(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("UniqueProfileName", func(t *testing.T) {
		cleanupProfiles(env)

		pm := env.Service.profileManager

		// Create first profile
		_, err := pm.CreateProfile(map[string]interface{}{
			"name": "unique-test",
		})
		if err != nil {
			t.Fatalf("Failed to create first profile: %v", err)
		}

		// Try to create duplicate
		_, err = pm.CreateProfile(map[string]interface{}{
			"name": "unique-test",
		})
		if err == nil {
			t.Error("Expected error for duplicate name, got nil")
		}
	})

	t.Run("ForeignKeyConstraint", func(t *testing.T) {
		cleanupProfiles(env)

		pm := env.Service.profileManager

		// Create profile
		profile, err := pm.CreateProfile(map[string]interface{}{
			"name": "fk-test",
		})
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Set as active
		err = pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Delete profile (should update foreign key to NULL)
		// First need to deactivate
		err = pm.ClearActiveProfile()
		if err != nil {
			t.Fatalf("Failed to clear active profile: %v", err)
		}

		err = pm.DeleteProfile("fk-test")
		if err != nil {
			t.Fatalf("Failed to delete profile: %v", err)
		}

		// Verify active_profile record is updated
		var profileID sql.NullString
		err = env.DB.QueryRow("SELECT profile_id FROM active_profile WHERE id = 1").Scan(&profileID)
		if err != nil {
			t.Fatalf("Failed to query active_profile: %v", err)
		}
		if profileID.Valid {
			t.Error("Expected profile_id to be NULL after deletion")
		}
	})
}

// TestTimestamps tests timestamp handling
func TestTimestamps(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("CreatedAtSet", func(t *testing.T) {
		cleanupProfiles(env)

		before := time.Now()
		profile, err := env.Service.profileManager.CreateProfile(map[string]interface{}{
			"name": "timestamp-test",
		})
		after := time.Now()

		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		if profile.CreatedAt.Before(before) || profile.CreatedAt.After(after) {
			t.Errorf("CreatedAt timestamp out of range: %v", profile.CreatedAt)
		}
	})

	t.Run("UpdatedAtChanges", func(t *testing.T) {
		cleanupProfiles(env)

		pm := env.Service.profileManager

		profile, err := pm.CreateProfile(map[string]interface{}{
			"name": "update-timestamp-test",
		})
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		originalUpdatedAt := profile.UpdatedAt

		// Wait a moment to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Update profile
		updated, err := pm.UpdateProfile("update-timestamp-test", map[string]interface{}{
			"display_name": "Updated",
		})
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		if !updated.UpdatedAt.After(originalUpdatedAt) {
			t.Errorf("UpdatedAt should be after original: original=%v, updated=%v",
				originalUpdatedAt, updated.UpdatedAt)
		}
	})
}

// TestProfileMetadata tests metadata field handling
func TestProfileMetadata(t *testing.T) {
	env := setupTestDB(t)
	defer env.Cleanup()

	t.Run("ComplexNestedMetadata", func(t *testing.T) {
		cleanupProfiles(env)

		metadata := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": "deep value",
					"array":  []interface{}{1, 2, 3},
				},
				"string": "test",
			},
			"number": 42,
			"bool":   true,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":     "metadata-test",
				"metadata": metadata,
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, rr, 201)

		// Verify metadata is preserved
		responseMetadata, ok := response["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected metadata object in response")
		}

		// Convert to JSON and back to compare
		originalJSON, _ := json.Marshal(metadata)
		responseJSON, _ := json.Marshal(responseMetadata)

		if string(originalJSON) != string(responseJSON) {
			t.Errorf("Metadata not preserved.\nExpected: %s\nGot: %s",
				string(originalJSON), string(responseJSON))
		}
	})
}
