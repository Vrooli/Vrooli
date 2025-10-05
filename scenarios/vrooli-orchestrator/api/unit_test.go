package main

import (
	"testing"
)

// Unit tests that don't require database connection

// TestProfile tests the Profile struct
func TestProfile(t *testing.T) {
	t.Run("ProfileStructure", func(t *testing.T) {
		profile := Profile{
			ID:          "test-id",
			Name:        "test-name",
			DisplayName: "Test Display Name",
			Description: "Test description",
			Status:      "active",
			Metadata:    make(map[string]interface{}),
			Resources:   []string{"postgres", "redis"},
			Scenarios:   []string{"scenario1"},
		}

		if profile.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", profile.ID)
		}

		if profile.Name != "test-name" {
			t.Errorf("Expected Name 'test-name', got '%s'", profile.Name)
		}

		if len(profile.Resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(profile.Resources))
		}

		if len(profile.Scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(profile.Scenarios))
		}
	})

	t.Run("ProfileWithIdleShutdown", func(t *testing.T) {
		idle := 30
		profile := Profile{
			ID:           "test-id",
			Name:         "test-name",
			IdleShutdown: &idle,
		}

		if profile.IdleShutdown == nil {
			t.Fatal("Expected IdleShutdown to be set")
		}

		if *profile.IdleShutdown != 30 {
			t.Errorf("Expected IdleShutdown 30, got %d", *profile.IdleShutdown)
		}
	})

	t.Run("ProfileWithNullIdleShutdown", func(t *testing.T) {
		profile := Profile{
			ID:           "test-id",
			Name:         "test-name",
			IdleShutdown: nil,
		}

		if profile.IdleShutdown != nil {
			t.Error("Expected IdleShutdown to be nil")
		}
	})
}

// TestActivationResultStructure tests ActivationResult
func TestActivationResultStructure(t *testing.T) {
	t.Run("BasicResult", func(t *testing.T) {
		result := ActivationResult{
			ProfileName:     "test-profile",
			Success:         true,
			ResourcesStatus: map[string]interface{}{"postgres": "started"},
			ScenariosStatus: map[string]interface{}{"scenario1": "started"},
			BrowserActions:  []string{"http://localhost:3000"},
			Message:         "Activated successfully",
		}

		if result.ProfileName != "test-profile" {
			t.Errorf("Expected ProfileName 'test-profile', got '%s'", result.ProfileName)
		}

		if !result.Success {
			t.Error("Expected Success true")
		}

		if len(result.BrowserActions) != 1 {
			t.Errorf("Expected 1 browser action, got %d", len(result.BrowserActions))
		}
	})

	t.Run("ErrorResult", func(t *testing.T) {
		result := ActivationResult{
			ProfileName:     "test-profile",
			Success:         false,
			ResourcesStatus: make(map[string]interface{}),
			ScenariosStatus: make(map[string]interface{}),
			BrowserActions:  []string{},
			Error:           "Failed to start resource",
		}

		if result.Success {
			t.Error("Expected Success false")
		}

		if result.Error == "" {
			t.Error("Expected Error to be set")
		}
	})
}

// TestDeactivationResultStructure tests DeactivationResult
func TestDeactivationResultStructure(t *testing.T) {
	t.Run("SuccessResult", func(t *testing.T) {
		result := DeactivationResult{
			Success:         true,
			ResourcesStatus: map[string]interface{}{"postgres": "stopped"},
			ScenariosStatus: map[string]interface{}{"scenario1": "stopped"},
			Message:         "Deactivated successfully",
		}

		if !result.Success {
			t.Error("Expected Success true")
		}

		if result.Message == "" {
			t.Error("Expected Message to be set")
		}
	})

	t.Run("ErrorResult", func(t *testing.T) {
		result := DeactivationResult{
			Success:         false,
			ResourcesStatus: make(map[string]interface{}),
			ScenariosStatus: make(map[string]interface{}),
			Error:           "Failed to stop resource",
		}

		if result.Success {
			t.Error("Expected Success false")
		}

		if result.Error == "" {
			t.Error("Expected Error to be set")
		}
	})
}

// TestConstants tests defined constants
func TestConstants(t *testing.T) {
	t.Run("APIVersion", func(t *testing.T) {
		if apiVersion == "" {
			t.Error("Expected apiVersion to be set")
		}
	})

	t.Run("ServiceName", func(t *testing.T) {
		if serviceName == "" {
			t.Error("Expected serviceName to be set")
		}

		if serviceName != "vrooli-orchestrator" {
			t.Errorf("Expected serviceName 'vrooli-orchestrator', got '%s'", serviceName)
		}
	})

	t.Run("Timeouts", func(t *testing.T) {
		if httpTimeout == 0 {
			t.Error("Expected httpTimeout to be set")
		}
	})

	t.Run("DatabaseLimits", func(t *testing.T) {
		if maxDBConnections == 0 {
			t.Error("Expected maxDBConnections to be set")
		}

		if maxIdleConnections == 0 {
			t.Error("Expected maxIdleConnections to be set")
		}

		if connMaxLifetime == 0 {
			t.Error("Expected connMaxLifetime to be set")
		}
	})
}

// TestHelperFunctions tests non-database helper functions
func TestContainsIgnoreCase(t *testing.T) {
	t.Run("ExactMatch", func(t *testing.T) {
		if !containsIgnoreCase("test", "test") {
			t.Error("Expected exact match to return true")
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		if !containsIgnoreCase("Test String", "test") {
			t.Error("Expected case-insensitive match")
		}

		if !containsIgnoreCase("Test String", "STRING") {
			t.Error("Expected case-insensitive match")
		}
	})

	t.Run("Substring", func(t *testing.T) {
		if !containsIgnoreCase("This is a test", "is a") {
			t.Error("Expected substring match")
		}
	})

	t.Run("NoMatch", func(t *testing.T) {
		if containsIgnoreCase("test", "xyz") {
			t.Error("Expected no match")
		}
	})

	t.Run("EmptySubstring", func(t *testing.T) {
		if !containsIgnoreCase("test", "") {
			t.Error("Expected empty substring to match")
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		if containsIgnoreCase("", "test") {
			t.Error("Expected empty string not to contain non-empty substring")
		}
	})
}

// TestTestScenarioBuilder tests the scenario builder
func TestTestScenarioBuilder(t *testing.T) {
	t.Run("EmptyBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.Build()

		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns, got %d", len(patterns))
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("POST", "/api/v1/test")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidJSON" {
			t.Errorf("Expected name 'InvalidJSON', got '%s'", patterns[0].Name)
		}
	})

	t.Run("MissingRequiredField", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddMissingRequiredField("POST", "/api/v1/test", "name")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("NonExistentProfile", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddNonExistentProfile("GET", "/api/v1/profiles/test")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].ExpectedStatus != 404 {
			t.Errorf("Expected status 404, got %d", patterns[0].ExpectedStatus)
		}
	})

	t.Run("ChainedBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.
			AddNonExistentProfile("GET", "/test1").
			AddNonExistentProfile("PUT", "/test2").
			AddNonExistentProfile("DELETE", "/test3").
			Build()

		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}
	})
}

// TestProfileCRUDErrorPatterns tests common error patterns
func TestProfileCRUDErrorPatterns(t *testing.T) {
	patterns := ProfileCRUDErrorPatterns()

	if len(patterns) == 0 {
		t.Error("Expected error patterns to be returned")
	}

	// Verify all patterns have required fields
	for _, pattern := range patterns {
		if pattern.Name == "" {
			t.Error("Expected all patterns to have a name")
		}

		if pattern.ExpectedStatus == 0 {
			t.Error("Expected all patterns to have an expected status")
		}
	}
}

// TestProfileActivationErrorPatterns tests activation error patterns
func TestProfileActivationErrorPatterns(t *testing.T) {
	patterns := ProfileActivationErrorPatterns()

	if len(patterns) == 0 {
		t.Error("Expected error patterns to be returned")
	}

	for _, pattern := range patterns {
		if pattern.Name == "" {
			t.Error("Expected all patterns to have a name")
		}

		if pattern.ExpectedStatus == 0 {
			t.Error("Expected all patterns to have an expected status")
		}
	}
}

// TestHTTPTestRequest tests the request structure
func TestHTTPTestRequest(t *testing.T) {
	t.Run("BasicRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/test",
		}

		if req.Method != "GET" {
			t.Errorf("Expected Method 'GET', got '%s'", req.Method)
		}

		if req.Path != "/api/v1/test" {
			t.Errorf("Expected Path '/api/v1/test', got '%s'", req.Path)
		}
	})

	t.Run("RequestWithBody", func(t *testing.T) {
		body := map[string]interface{}{
			"key": "value",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/test",
			Body:   body,
		}

		if req.Body == nil {
			t.Error("Expected Body to be set")
		}
	})

	t.Run("RequestWithHeaders", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token",
		}

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/test",
			Headers: headers,
		}

		if len(req.Headers) != 1 {
			t.Errorf("Expected 1 header, got %d", len(req.Headers))
		}
	})
}
