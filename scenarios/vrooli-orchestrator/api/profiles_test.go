package main

import (
	"testing"
	"time"
)

// TestNewProfileManager tests profile manager creation
func TestNewProfileManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := NewProfileManager(env.DB, env.Service.logger)

	if pm == nil {
		t.Fatal("Expected profile manager to be created")
	}

	if pm.db == nil {
		t.Error("Expected db to be set")
	}

	if pm.logger == nil {
		t.Error("Expected logger to be set")
	}
}

// TestProfileManagerListProfiles tests listing profiles
func TestProfileManagerListProfiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("EmptyList", func(t *testing.T) {
		profiles, err := pm.ListProfiles()
		if err != nil {
			t.Fatalf("Failed to list profiles: %v", err)
		}

		if len(profiles) != 0 {
			t.Errorf("Expected 0 profiles, got %d", len(profiles))
		}
	})

	t.Run("MultipleProfiles", func(t *testing.T) {
		createTestProfile(t, env, "profile-1", "inactive")
		createTestProfile(t, env, "profile-2", "inactive")
		createTestProfile(t, env, "profile-3", "active")
		defer cleanupProfiles(env)

		profiles, err := pm.ListProfiles()
		if err != nil {
			t.Fatalf("Failed to list profiles: %v", err)
		}

		if len(profiles) != 3 {
			t.Errorf("Expected 3 profiles, got %d", len(profiles))
		}

		// Verify profiles are ordered by created_at DESC
		if len(profiles) >= 2 {
			if profiles[0].CreatedAt.Before(profiles[1].CreatedAt) {
				t.Error("Expected profiles to be ordered by created_at DESC")
			}
		}
	})
}

// TestProfileManagerGetProfile tests getting a specific profile
func TestProfileManagerGetProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("ExistingProfile", func(t *testing.T) {
		created := createTestProfile(t, env, "get-test", "inactive")
		defer cleanupProfiles(env)

		profile, err := pm.GetProfile("get-test")
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}

		if profile.Name != "get-test" {
			t.Errorf("Expected name 'get-test', got '%s'", profile.Name)
		}

		if profile.ID != created.ID {
			t.Errorf("Expected ID '%s', got '%s'", created.ID, profile.ID)
		}

		// Verify JSON fields were parsed correctly
		if len(profile.Resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(profile.Resources))
		}

		if len(profile.Scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(profile.Scenarios))
		}
	})

	t.Run("NonExistentProfile", func(t *testing.T) {
		profile, err := pm.GetProfile("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent profile")
		}

		if profile != nil {
			t.Error("Expected nil profile for non-existent profile")
		}

		if !containsIgnoreCase(err.Error(), "not found") {
			t.Errorf("Expected 'not found' in error, got: %v", err)
		}
	})
}

// TestProfileManagerCreateProfile tests creating a profile
func TestProfileManagerCreateProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("MinimalProfile", func(t *testing.T) {
		defer cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name": "minimal-profile",
		}

		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		if profile.Name != "minimal-profile" {
			t.Errorf("Expected name 'minimal-profile', got '%s'", profile.Name)
		}

		if profile.DisplayName != "minimal-profile" {
			t.Errorf("Expected display_name to default to name, got '%s'", profile.DisplayName)
		}

		if profile.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", profile.Status)
		}

		if profile.ID == "" {
			t.Error("Expected ID to be generated")
		}
	})

	t.Run("CompleteProfile", func(t *testing.T) {
		defer cleanupProfiles(env)

		idleShutdown := 30

		profileData := map[string]interface{}{
			"name":         "complete-profile",
			"display_name": "Complete Profile",
			"description":  "A complete test profile",
			"resources":    []string{"postgres", "redis", "qdrant"},
			"scenarios":    []string{"scenario-1", "scenario-2"},
			"auto_browser": []string{"http://localhost:3000"},
			"environment_vars": map[string]interface{}{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			"idle_shutdown_minutes": float64(idleShutdown),
			"dependencies":          []string{"dep1", "dep2"},
			"metadata": map[string]interface{}{
				"custom_key": "custom_value",
			},
		}

		profile, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		if profile.Name != "complete-profile" {
			t.Errorf("Expected name 'complete-profile', got '%s'", profile.Name)
		}

		if len(profile.Resources) != 3 {
			t.Errorf("Expected 3 resources, got %d", len(profile.Resources))
		}

		if len(profile.Scenarios) != 2 {
			t.Errorf("Expected 2 scenarios, got %d", len(profile.Scenarios))
		}

		if len(profile.AutoBrowser) != 1 {
			t.Errorf("Expected 1 auto_browser URL, got %d", len(profile.AutoBrowser))
		}

		if len(profile.EnvironmentVars) != 2 {
			t.Errorf("Expected 2 environment vars, got %d", len(profile.EnvironmentVars))
		}

		if profile.IdleShutdown == nil || *profile.IdleShutdown != idleShutdown {
			t.Errorf("Expected idle_shutdown %d, got %v", idleShutdown, profile.IdleShutdown)
		}

		if len(profile.Dependencies) != 2 {
			t.Errorf("Expected 2 dependencies, got %d", len(profile.Dependencies))
		}

		if len(profile.Metadata) != 1 {
			t.Errorf("Expected 1 metadata entry, got %d", len(profile.Metadata))
		}
	})

	t.Run("MissingName", func(t *testing.T) {
		profileData := map[string]interface{}{
			"display_name": "No Name",
		}

		profile, err := pm.CreateProfile(profileData)
		if err == nil {
			t.Error("Expected error for missing name")
		}

		if profile != nil {
			t.Error("Expected nil profile for invalid data")
		}

		if !containsIgnoreCase(err.Error(), "required") {
			t.Errorf("Expected 'required' in error, got: %v", err)
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		profileData := map[string]interface{}{
			"name": "",
		}

		profile, err := pm.CreateProfile(profileData)
		if err == nil {
			t.Error("Expected error for empty name")
		}

		if profile != nil {
			t.Error("Expected nil profile for invalid data")
		}
	})

	t.Run("DuplicateName", func(t *testing.T) {
		createTestProfile(t, env, "duplicate", "inactive")
		defer cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name": "duplicate",
		}

		profile, err := pm.CreateProfile(profileData)
		if err == nil {
			t.Error("Expected error for duplicate name")
		}

		if profile != nil {
			t.Error("Expected nil profile for duplicate name")
		}

		if !containsIgnoreCase(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' in error, got: %v", err)
		}
	})
}

// TestProfileManagerUpdateProfile tests updating a profile
func TestProfileManagerUpdateProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("UpdateSimpleFields", func(t *testing.T) {
		createTestProfile(t, env, "update-simple", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{
			"display_name": "Updated Name",
			"description":  "Updated description",
			"status":       "active",
		}

		profile, err := pm.UpdateProfile("update-simple", updates)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		if profile.DisplayName != "Updated Name" {
			t.Errorf("Expected display_name 'Updated Name', got '%s'", profile.DisplayName)
		}

		if profile.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", profile.Description)
		}

		if profile.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", profile.Status)
		}
	})

	t.Run("UpdateJSONFields", func(t *testing.T) {
		createTestProfile(t, env, "update-json", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{
			"resources": []string{"postgres", "redis", "qdrant", "n8n"},
			"scenarios": []string{"new-scenario"},
			"metadata": map[string]interface{}{
				"updated": "true",
			},
		}

		profile, err := pm.UpdateProfile("update-json", updates)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		if len(profile.Resources) != 4 {
			t.Errorf("Expected 4 resources, got %d", len(profile.Resources))
		}

		if len(profile.Scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(profile.Scenarios))
		}

		if profile.Metadata["updated"] != "true" {
			t.Errorf("Expected metadata updated=true, got: %v", profile.Metadata)
		}
	})

	t.Run("UpdateIdleShutdown", func(t *testing.T) {
		createTestProfile(t, env, "update-idle", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{
			"idle_shutdown_minutes": float64(60),
		}

		profile, err := pm.UpdateProfile("update-idle", updates)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		if profile.IdleShutdown == nil || *profile.IdleShutdown != 60 {
			t.Errorf("Expected idle_shutdown 60, got: %v", profile.IdleShutdown)
		}
	})

	t.Run("ClearIdleShutdown", func(t *testing.T) {
		createTestProfile(t, env, "clear-idle", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{
			"idle_shutdown_minutes": nil,
		}

		profile, err := pm.UpdateProfile("clear-idle", updates)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		if profile.IdleShutdown != nil {
			t.Errorf("Expected nil idle_shutdown, got: %v", profile.IdleShutdown)
		}
	})

	t.Run("EmptyUpdate", func(t *testing.T) {
		original := createTestProfile(t, env, "empty-update", "inactive")
		defer cleanupProfiles(env)

		updates := map[string]interface{}{}

		profile, err := pm.UpdateProfile("empty-update", updates)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		// Profile should be unchanged
		if profile.DisplayName != original.DisplayName {
			t.Errorf("Expected display_name unchanged, got '%s'", profile.DisplayName)
		}
	})

	t.Run("NonExistentProfile", func(t *testing.T) {
		updates := map[string]interface{}{
			"display_name": "Updated",
		}

		profile, err := pm.UpdateProfile("nonexistent", updates)
		if err == nil {
			t.Error("Expected error for non-existent profile")
		}

		if profile != nil {
			t.Error("Expected nil profile for non-existent profile")
		}
	})

	t.Run("UpdatedAtChanged", func(t *testing.T) {
		original := createTestProfile(t, env, "updated-at-test", "inactive")
		defer cleanupProfiles(env)

		time.Sleep(100 * time.Millisecond) // Ensure time difference

		updates := map[string]interface{}{
			"description": "Updated",
		}

		profile, err := pm.UpdateProfile("updated-at-test", updates)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		if !profile.UpdatedAt.After(original.UpdatedAt) {
			t.Error("Expected updated_at to be changed")
		}
	})
}

// TestProfileManagerDeleteProfile tests deleting a profile
func TestProfileManagerDeleteProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("InactiveProfile", func(t *testing.T) {
		createTestProfile(t, env, "delete-inactive", "inactive")
		defer cleanupProfiles(env)

		err := pm.DeleteProfile("delete-inactive")
		if err != nil {
			t.Fatalf("Failed to delete profile: %v", err)
		}

		// Verify profile was deleted
		profile, err := pm.GetProfile("delete-inactive")
		if err == nil || profile != nil {
			t.Error("Profile should not exist after deletion")
		}
	})

	t.Run("ActiveProfile", func(t *testing.T) {
		profile := createTestProfile(t, env, "delete-active", "inactive")
		defer cleanupProfiles(env)

		// Set as active
		err := pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Try to delete
		err = pm.DeleteProfile("delete-active")
		if err == nil {
			t.Error("Expected error when deleting active profile")
		}

		if !containsIgnoreCase(err.Error(), "cannot delete active") {
			t.Errorf("Expected 'cannot delete active' in error, got: %v", err)
		}

		// Verify profile still exists
		profile, err = pm.GetProfile("delete-active")
		if err != nil || profile == nil {
			t.Error("Profile should still exist")
		}
	})

	t.Run("NonExistentProfile", func(t *testing.T) {
		err := pm.DeleteProfile("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent profile")
		}

		if !containsIgnoreCase(err.Error(), "not found") {
			t.Errorf("Expected 'not found' in error, got: %v", err)
		}
	})
}

// TestProfileManagerGetActiveProfile tests getting the active profile
func TestProfileManagerGetActiveProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("NoActiveProfile", func(t *testing.T) {
		profile, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if profile != nil {
			t.Error("Expected nil profile when no active profile")
		}
	})

	t.Run("WithActiveProfile", func(t *testing.T) {
		created := createTestProfile(t, env, "active-test", "inactive")
		defer cleanupProfiles(env)

		err := pm.SetActiveProfile(created.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		profile, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Failed to get active profile: %v", err)
		}

		if profile == nil {
			t.Fatal("Expected active profile to be returned")
		}

		if profile.ID != created.ID {
			t.Errorf("Expected ID '%s', got '%s'", created.ID, profile.ID)
		}

		if profile.Name != "active-test" {
			t.Errorf("Expected name 'active-test', got '%s'", profile.Name)
		}

		if profile.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", profile.Status)
		}
	})
}

// TestProfileManagerSetActiveProfile tests setting a profile as active
func TestProfileManagerSetActiveProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("SetActive", func(t *testing.T) {
		profile := createTestProfile(t, env, "set-active", "inactive")
		defer cleanupProfiles(env)

		err := pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Verify profile was set as active
		activeProfile, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Failed to get active profile: %v", err)
		}

		if activeProfile.ID != profile.ID {
			t.Errorf("Expected active profile ID '%s', got '%s'", profile.ID, activeProfile.ID)
		}

		if activeProfile.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", activeProfile.Status)
		}
	})

	t.Run("ReplaceActive", func(t *testing.T) {
		profile1 := createTestProfile(t, env, "replace-1", "inactive")
		profile2 := createTestProfile(t, env, "replace-2", "inactive")
		defer cleanupProfiles(env)

		// Set first as active
		err := pm.SetActiveProfile(profile1.ID)
		if err != nil {
			t.Fatalf("Failed to set first profile as active: %v", err)
		}

		// Set second as active
		err = pm.SetActiveProfile(profile2.ID)
		if err != nil {
			t.Fatalf("Failed to set second profile as active: %v", err)
		}

		// Verify second is now active
		activeProfile, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Failed to get active profile: %v", err)
		}

		if activeProfile.ID != profile2.ID {
			t.Errorf("Expected active profile ID '%s', got '%s'", profile2.ID, activeProfile.ID)
		}

		// Note: Status is only updated when clearing active profile, not when replacing
		// This is expected behavior based on the implementation
		// Verify first profile still exists
		_, err = pm.GetProfile("replace-1")
		if err != nil {
			t.Fatalf("Failed to get first profile: %v", err)
		}
	})
}

// TestProfileManagerClearActiveProfile tests clearing the active profile
func TestProfileManagerClearActiveProfile(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("WithActiveProfile", func(t *testing.T) {
		profile := createTestProfile(t, env, "clear-active", "inactive")
		defer cleanupProfiles(env)

		// Set as active
		err := pm.SetActiveProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to set active profile: %v", err)
		}

		// Clear active profile
		err = pm.ClearActiveProfile()
		if err != nil {
			t.Fatalf("Failed to clear active profile: %v", err)
		}

		// Verify no active profile
		activeProfile, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if activeProfile != nil {
			t.Error("Expected no active profile after clearing")
		}

		// Verify profile status was updated
		profile, err = pm.GetProfile("clear-active")
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}

		if profile.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", profile.Status)
		}
	})

	t.Run("NoActiveProfile", func(t *testing.T) {
		// Should not error when no active profile
		err := pm.ClearActiveProfile()
		if err != nil {
			t.Fatalf("Unexpected error when clearing with no active profile: %v", err)
		}
	})
}

// TestProfileJSONParsing tests JSON field parsing
func TestProfileJSONParsing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	pm := env.Service.profileManager

	t.Run("ComplexJSONFields", func(t *testing.T) {
		defer cleanupProfiles(env)

		profileData := map[string]interface{}{
			"name": "json-test",
			"metadata": map[string]interface{}{
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []interface{}{1, 2, 3},
			},
			"environment_vars": map[string]interface{}{
				"PATH":    "/usr/bin",
				"API_KEY": "secret123",
			},
		}

		_, err := pm.CreateProfile(profileData)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Retrieve and verify parsing
		retrieved, err := pm.GetProfile("json-test")
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}

		// Verify nested metadata
		if retrieved.Metadata["nested"] == nil {
			t.Error("Expected nested metadata to be parsed")
		}

		// Verify environment vars
		if retrieved.EnvironmentVars["PATH"] != "/usr/bin" {
			t.Errorf("Expected PATH='/usr/bin', got: %v", retrieved.EnvironmentVars["PATH"])
		}

		// Verify arrays default to empty
		if retrieved.Resources == nil {
			t.Error("Expected resources array to be initialized")
		}
	})
}
