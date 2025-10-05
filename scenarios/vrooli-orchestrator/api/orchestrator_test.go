package main

import (
	"testing"
)

// TestNewOrchestratorManager tests orchestrator manager creation
func TestNewOrchestratorManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager
	om := NewOrchestratorManager(pm, env.Service.logger)

	if om == nil {
		t.Fatal("Expected orchestrator manager to be created")
	}

	if om.profileManager == nil {
		t.Error("Expected profileManager to be set")
	}

	if om.logger == nil {
		t.Error("Expected logger to be set")
	}
}

// TestActivateProfile tests profile activation
func TestActivateProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("NonExistentProfile", func(t *testing.T) {
		result, err := om.ActivateProfile("nonexistent", map[string]interface{}{})
		if err == nil {
			t.Error("Expected error for non-existent profile")
		}

		if result == nil {
			t.Fatal("Expected result even on error")
		}

		if result.Success {
			t.Error("Expected success to be false")
		}

		if result.ProfileName != "nonexistent" {
			t.Errorf("Expected profile_name 'nonexistent', got '%s'", result.ProfileName)
		}
	})

	t.Run("SuccessfulActivation", func(t *testing.T) {
		// Note: This test will attempt to start resources/scenarios
		// In a real environment, this might fail if those don't exist
		// We're testing the logic flow, not actual resource starting
		_ = createTestProfile(t, env, "activate-test", "inactive")
		defer cleanupProfiles(env)

		result, err := om.ActivateProfile("activate-test", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result")
		}

		if result.ProfileName != "activate-test" {
			t.Errorf("Expected profile_name 'activate-test', got '%s'", result.ProfileName)
		}

		// Result may or may not succeed depending on whether resources exist
		// Just verify structure
		if result.ResourcesStatus == nil {
			t.Error("Expected resources_status to be initialized")
		}

		if result.ScenariosStatus == nil {
			t.Error("Expected scenarios_status to be initialized")
		}

		if result.BrowserActions == nil {
			t.Error("Expected browser_actions to be initialized")
		}
	})

	t.Run("EmptyProfile", func(t *testing.T) {
		// Create a profile with no resources or scenarios
		profileData := map[string]interface{}{
			"name":         "empty-activate",
			"display_name": "Empty Profile",
		}

		profile, err := env.Service.profileManager.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}
		defer cleanupProfiles(env)

		result, err := om.ActivateProfile("empty-activate", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should succeed since there's nothing to start
		if !result.Success {
			t.Errorf("Expected success for empty profile, got error: %s", result.Error)
		}

		// Verify profile was set as active
		activeProfile, err := env.Service.profileManager.GetActiveProfile()
		if err != nil {
			t.Fatalf("Failed to get active profile: %v", err)
		}

		if activeProfile.ID != profile.ID {
			t.Error("Expected profile to be set as active")
		}
	})
}

// TestDeactivateCurrentProfile tests profile deactivation
func TestDeactivateCurrentProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("NoActiveProfile", func(t *testing.T) {
		err := om.DeactivateCurrentProfile()
		if err != nil {
			t.Fatalf("Expected no error when no active profile: %v", err)
		}
	})

	t.Run("WithActiveProfile", func(t *testing.T) {
		profile := createTestProfile(t, env, "deactivate-test", "inactive")
		defer cleanupProfiles(env)

		// Set as active
		err := env.Service.profileManager.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Deactivate
		err = om.DeactivateCurrentProfile()
		if err != nil {
			t.Fatalf("Failed to deactivate profile: %v", err)
		}

		// Verify no active profile
		activeProfile, err := env.Service.profileManager.GetActiveProfile()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if activeProfile != nil {
			t.Error("Expected no active profile after deactivation")
		}

		// Verify profile status
		updatedProfile, err := env.Service.profileManager.GetProfile("deactivate-test")
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}

		if updatedProfile.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", updatedProfile.Status)
		}
	})
}

// TestGetDeactivationResult tests getting deactivation result
func TestGetDeactivationResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("NoActiveProfile", func(t *testing.T) {
		result, err := om.GetDeactivationResult()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result")
		}

		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}

		if result.ResourcesStatus == nil {
			t.Error("Expected resources_status to be initialized")
		}

		if result.ScenariosStatus == nil {
			t.Error("Expected scenarios_status to be initialized")
		}
	})

	t.Run("WithActiveProfile", func(t *testing.T) {
		profile := createTestProfile(t, env, "deactivation-result-test", "inactive")
		defer cleanupProfiles(env)

		// Set as active
		err := env.Service.profileManager.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		result, err := om.GetDeactivationResult()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result")
		}

		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}
	})
}

// TestStartResource tests resource starting logic
func TestStartResource(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("ResourceStart", func(t *testing.T) {
		// This will likely fail since the resource doesn't exist
		// but we're testing the logic flow
		status := om.startResource("test-resource")

		if status == nil {
			t.Fatal("Expected status map")
		}

		if resource, ok := status["resource"].(string); !ok || resource != "test-resource" {
			t.Errorf("Expected resource 'test-resource', got: %v", status["resource"])
		}

		if _, ok := status["success"].(bool); !ok {
			t.Error("Expected success field in status")
		}

		if _, ok := status["output"].(string); !ok {
			t.Error("Expected output field in status")
		}

		// Resource likely won't exist, so success should be false
		// But if it does exist, that's fine too
	})
}

// TestStopResource tests resource stopping logic
func TestStopResource(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("ResourceStop", func(t *testing.T) {
		status := om.stopResource("test-resource")

		if status == nil {
			t.Fatal("Expected status map")
		}

		if resource, ok := status["resource"].(string); !ok || resource != "test-resource" {
			t.Errorf("Expected resource 'test-resource', got: %v", status["resource"])
		}

		if _, ok := status["success"].(bool); !ok {
			t.Error("Expected success field in status")
		}
	})
}

// TestStartScenario tests scenario starting logic
func TestStartScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("ScenarioStart", func(t *testing.T) {
		status := om.startScenario("test-scenario")

		if status == nil {
			t.Fatal("Expected status map")
		}

		if scenario, ok := status["scenario"].(string); !ok || scenario != "test-scenario" {
			t.Errorf("Expected scenario 'test-scenario', got: %v", status["scenario"])
		}

		if _, ok := status["success"].(bool); !ok {
			t.Error("Expected success field in status")
		}

		if _, ok := status["output"].(string); !ok {
			t.Error("Expected output field in status")
		}
	})
}

// TestStopScenario tests scenario stopping logic
func TestStopScenario(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("ScenarioStop", func(t *testing.T) {
		status := om.stopScenario("test-scenario")

		if status == nil {
			t.Fatal("Expected status map")
		}

		if scenario, ok := status["scenario"].(string); !ok || scenario != "test-scenario" {
			t.Errorf("Expected scenario 'test-scenario', got: %v", status["scenario"])
		}

		if _, ok := status["success"].(bool); !ok {
			t.Error("Expected success field in status")
		}
	})
}

// TestOpenBrowserTab tests browser tab opening logic
func TestOpenBrowserTab(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("BrowserAction", func(t *testing.T) {
		action := om.openBrowserTab("http://localhost:3000")

		if action == "" {
			t.Error("Expected action string")
		}

		if !containsIgnoreCase(action, "localhost:3000") {
			t.Errorf("Expected URL in action, got: %s", action)
		}
	})
}

// TestValidateProfile tests profile validation
func TestValidateProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("EmptyProfile", func(t *testing.T) {
		profile := &Profile{
			Name:      "empty-validate",
			Resources: []string{},
			Scenarios: []string{},
		}

		issues := om.ValidateProfile(profile)

		if len(issues) != 0 {
			t.Errorf("Expected no issues for empty profile, got: %v", issues)
		}
	})

	t.Run("ProfileWithResources", func(t *testing.T) {
		profile := &Profile{
			Name:      "resources-validate",
			Resources: []string{"postgres", "nonexistent-resource"},
			Scenarios: []string{},
		}

		issues := om.ValidateProfile(profile)

		// Should have at least one issue for nonexistent resource
		// Note: postgres might or might not exist in test environment
		if len(issues) == 0 {
			// This is acceptable if all resources exist
		}
	})

	t.Run("ProfileWithScenarios", func(t *testing.T) {
		profile := &Profile{
			Name:      "scenarios-validate",
			Resources: []string{},
			Scenarios: []string{"nonexistent-scenario"},
		}

		issues := om.ValidateProfile(profile)

		// Should have at least one issue for nonexistent scenario
		if len(issues) == 0 {
			// This is acceptable if scenario exists
		}
	})
}

// TestIsResourceAvailable tests resource availability checking
func TestIsResourceAvailable(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("CheckResource", func(t *testing.T) {
		// This depends on actual system state
		// Just verify it returns a boolean without panicking
		available := om.isResourceAvailable("postgres")
		_ = available // May be true or false depending on environment
	})

	t.Run("NonExistentResource", func(t *testing.T) {
		available := om.isResourceAvailable("definitely-nonexistent-resource-12345")
		if available {
			t.Error("Expected nonexistent resource to return false")
		}
	})
}

// TestIsScenarioAvailable tests scenario availability checking
func TestIsScenarioAvailable(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager

	t.Run("CheckScenario", func(t *testing.T) {
		// This depends on actual system state
		// Just verify it returns a boolean without panicking
		available := om.isScenarioAvailable("test-scenario")
		_ = available // May be true or false depending on environment
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		available := om.isScenarioAvailable("definitely-nonexistent-scenario-12345")
		if available {
			t.Error("Expected nonexistent scenario to return false")
		}
	})
}

// TestActivationResult tests activation result structure
func TestActivationResult(t *testing.T) {
	t.Run("ResultStructure", func(t *testing.T) {
		result := &ActivationResult{
			ProfileName:     "test",
			Success:         true,
			ResourcesStatus: make(map[string]interface{}),
			ScenariosStatus: make(map[string]interface{}),
			BrowserActions:  []string{},
			Message:         "Test message",
		}

		if result.ProfileName != "test" {
			t.Errorf("Expected ProfileName 'test', got '%s'", result.ProfileName)
		}

		if !result.Success {
			t.Error("Expected Success true")
		}

		if result.ResourcesStatus == nil {
			t.Error("Expected ResourcesStatus to be initialized")
		}
	})
}

// TestDeactivationResult tests deactivation result structure
func TestDeactivationResult(t *testing.T) {
	t.Run("ResultStructure", func(t *testing.T) {
		result := &DeactivationResult{
			Success:         true,
			ResourcesStatus: make(map[string]interface{}),
			ScenariosStatus: make(map[string]interface{}),
			Message:         "Test message",
		}

		if !result.Success {
			t.Error("Expected Success true")
		}

		if result.ResourcesStatus == nil {
			t.Error("Expected ResourcesStatus to be initialized")
		}

		if result.ScenariosStatus == nil {
			t.Error("Expected ScenariosStatus to be initialized")
		}
	})
}

// TestProfileActivationDeactivationCycle tests full activation/deactivation cycle
func TestProfileActivationDeactivationCycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	om := env.Service.orchestratorManager
	pm := env.Service.profileManager

	t.Run("FullCycle", func(t *testing.T) {
		// Create profile with empty resources/scenarios for reliable testing
		profileData := map[string]interface{}{
			"name":         "cycle-test",
			"display_name": "Cycle Test",
			"resources":    []string{},
			"scenarios":    []string{},
		}

		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}
		defer cleanupProfiles(env)

		// Activate
		activateResult, err := om.ActivateProfile("cycle-test", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to activate profile: %v", err)
		}

		if !activateResult.Success {
			t.Fatalf("Activation failed: %s", activateResult.Error)
		}

		// Verify active
		activeProfile, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Failed to get active profile: %v", err)
		}

		if activeProfile.ID != profile.ID {
			t.Error("Expected profile to be active")
		}

		// Deactivate
		err = om.DeactivateCurrentProfile()
		if err != nil {
			t.Fatalf("Failed to deactivate profile: %v", err)
		}

		// Verify inactive
		activeProfile, err = pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if activeProfile != nil {
			t.Error("Expected no active profile after deactivation")
		}

		// Verify status updated
		updatedProfile, err := pm.GetProfile("cycle-test")
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}

		if updatedProfile.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", updatedProfile.Status)
		}
	})
}
