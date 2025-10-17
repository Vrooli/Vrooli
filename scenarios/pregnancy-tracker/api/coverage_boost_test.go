// +build testing

package main

import (
	"os"
	"testing"
	"time"
)

// TestTestInfrastructure tests the testing helper functions
func TestTestInfrastructure(t *testing.T) {
	t.Run("setupTestLogger", func(t *testing.T) {
		cleanup := setupTestLogger()
		if cleanup == nil {
			t.Error("Expected cleanup function")
		}
		cleanup()
	})

	t.Run("getEnvOrDefault_WithValue", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test-value")
		result := getEnvOrDefault("TEST_VAR", "default")
		if result != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", result)
		}
		os.Unsetenv("TEST_VAR")
	})

	t.Run("getEnvOrDefault_WithoutValue", func(t *testing.T) {
		os.Unsetenv("NONEXISTENT_VAR")
		result := getEnvOrDefault("NONEXISTENT_VAR", "default-value")
		if result != "default-value" {
			t.Errorf("Expected 'default-value', got '%s'", result)
		}
	})

	t.Run("getEnvOrDefault_EmptyValue", func(t *testing.T) {
		os.Setenv("EMPTY_VAR", "")
		result := getEnvOrDefault("EMPTY_VAR", "default")
		if result != "default" {
			t.Errorf("Expected 'default' for empty env var, got '%s'", result)
		}
		os.Unsetenv("EMPTY_VAR")
	})
}

// TestTestPatternBuilders tests the test pattern builder infrastructure
func TestTestPatternBuilders(t *testing.T) {
	t.Run("NewTestScenarioBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		if builder == nil {
			t.Error("Expected non-nil builder")
		}
		if builder.patterns == nil {
			t.Error("Expected patterns slice to be initialized")
		}
	})

	t.Run("TestScenarioBuilder_AddMissingUserID", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddMissingUserID("GET", "/test/path")

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "MissingUserID" {
			t.Errorf("Expected pattern name 'MissingUserID', got '%s'", patterns[0].Name)
		}
	})

	t.Run("TestScenarioBuilder_AddInvalidJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("POST", "/test/path", "user-123")

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidJSON" {
			t.Errorf("Expected pattern name 'InvalidJSON', got '%s'", patterns[0].Name)
		}
	})

	t.Run("TestScenarioBuilder_AddInvalidMethod", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidMethod("POST", "/test/path", "user-123")

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("TestScenarioBuilder_Chaining", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.
			AddMissingUserID("GET", "/path1").
			AddInvalidJSON("POST", "/path2", "user").
			AddInvalidMethod("GET", "/path3", "user").
			Build()

		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}
	})

	t.Run("TestScenarioBuilder_AddCustom", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "A custom test pattern",
			ExpectedStatus: 400,
		}

		builder.AddCustom(customPattern)
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "CustomTest" {
			t.Errorf("Expected pattern name 'CustomTest', got '%s'", patterns[0].Name)
		}
	})

	t.Run("TestScenarioBuilder_AddMissingRequiredField", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddMissingRequiredField("POST", "/test", "user", map[string]string{})

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("TestScenarioBuilder_AddNoActivePregnancy", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddNoActivePregnancy("GET", "/test", "user", nil)

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})
}

// TestErrorPatternExecution tests error pattern execution
func TestErrorPatternExecution(t *testing.T) {
	env, cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("missingUserIDPattern_Execution", func(t *testing.T) {
		pattern := missingUserIDPattern("GET", "/api/v1/pregnancy/current")

		if pattern.ExpectedStatus != 401 {
			t.Errorf("Expected status 401, got %d", pattern.ExpectedStatus)
		}

		req := pattern.Execute(t, nil)
		if req.UserID != "" {
			t.Error("Expected empty UserID")
		}
	})

	t.Run("invalidJSONPattern_Execution", func(t *testing.T) {
		pattern := invalidJSONPattern("POST", "/api/v1/logs/daily", "user-123")

		if pattern.ExpectedStatus != 400 {
			t.Errorf("Expected status 400, got %d", pattern.ExpectedStatus)
		}

		req := pattern.Execute(t, nil)
		if req.Method != "POST" {
			t.Errorf("Expected POST method, got %s", req.Method)
		}
	})

	t.Run("invalidMethodPattern_Execution", func(t *testing.T) {
		pattern := invalidMethodPattern("GET", "/api/v1/logs/daily", "user-123")

		req := pattern.Execute(t, nil)
		if req.Method != "GET" {
			t.Errorf("Expected GET method, got %s", req.Method)
		}
	})

	t.Run("missingRequiredFieldPattern_Execution", func(t *testing.T) {
		pattern := missingRequiredFieldPattern("POST", "/test", "user", map[string]string{})

		req := pattern.Execute(t, nil)
		if req.UserID != "user" {
			t.Errorf("Expected user ID 'user', got '%s'", req.UserID)
		}
	})

	t.Run("noActivePregnancyPattern_Execution", func(t *testing.T) {
		pattern := noActivePregnancyPattern("GET", "/test", "user-no-preg", nil)

		req := pattern.Execute(t, nil)
		if req.UserID != "user-no-preg" {
			t.Errorf("Expected user ID 'user-no-preg', got '%s'", req.UserID)
		}
	})

	_ = env
}

// TestTestDataGenerator tests the test data generation utilities
func TestTestDataGenerator(t *testing.T) {
	t.Run("PregnancyStartRequest", func(t *testing.T) {
		// Use the global TestData generator
		req := TestData.PregnancyStartRequest(time.Now().AddDate(0, 0, -70))

		if req == nil {
			t.Error("Expected non-nil request")
		}

		if _, ok := req["lmp_date"]; !ok {
			t.Error("Expected lmp_date in request")
		}

		if _, ok := req["due_date"]; !ok {
			t.Error("Expected due_date in request")
		}
	})

	t.Run("DailyLogRequest", func(t *testing.T) {
		logReq := TestData.DailyLogRequest()

		if logReq.Weight == nil {
			t.Error("Expected non-nil weight")
		}

		if logReq.BloodPressure == "" {
			t.Error("Expected blood pressure value")
		}

		if len(logReq.Symptoms) == 0 {
			t.Error("Expected symptoms array")
		}

		if logReq.Mood == 0 {
			t.Error("Expected mood value")
		}

		if logReq.Energy == 0 {
			t.Error("Expected energy value")
		}
	})

	t.Run("KickCountRequest", func(t *testing.T) {
		kickReq := TestData.KickCountRequest()

		if kickReq.SessionEnd == nil {
			t.Error("Expected non-nil session end")
		}

		if kickReq.Count == 0 {
			t.Error("Expected count value")
		}

		if kickReq.Duration == 0 {
			t.Error("Expected duration value")
		}
	})

	t.Run("AppointmentRequest", func(t *testing.T) {
		aptReq := TestData.AppointmentRequest()

		if aptReq.Type == "" {
			t.Error("Expected appointment type")
		}

		if aptReq.Provider == "" {
			t.Error("Expected provider name")
		}

		if aptReq.Location == "" {
			t.Error("Expected location")
		}

		if aptReq.Results == nil {
			t.Error("Expected results map")
		}
	})
}

// TestMakeHTTPRequestVariations tests different makeHTTPRequest scenarios
func TestMakeHTTPRequestVariations(t *testing.T) {
	t.Run("makeHTTPRequest_WithStringBody", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   "string body",
			UserID: "user-123",
		})

		if w == nil || req == nil {
			t.Error("Expected non-nil response and request")
		}

		if req.Header.Get("X-User-ID") != "user-123" {
			t.Error("Expected X-User-ID header")
		}
	})

	t.Run("makeHTTPRequest_WithByteBody", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   []byte("byte body"),
		})

		if w == nil || req == nil {
			t.Error("Expected non-nil response and request")
		}
	})

	t.Run("makeHTTPRequest_WithStructBody", func(t *testing.T) {
		type TestStruct struct {
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
		}

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body: TestStruct{
				Field1: "value1",
				Field2: 42,
			},
		})

		if w == nil || req == nil {
			t.Error("Expected non-nil response and request")
		}

		if req.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}
	})

	t.Run("makeHTTPRequest_WithCustomHeaders", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"Custom-Header": "custom-value",
			},
		})

		if w == nil || req == nil {
			t.Error("Expected non-nil response and request")
		}

		if req.Header.Get("Custom-Header") != "custom-value" {
			t.Error("Expected Custom-Header")
		}
	})

	t.Run("makeHTTPRequest_NoBody", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})

		if w == nil || req == nil {
			t.Error("Expected non-nil response and request")
		}
	})
}

// TestAssertionHelpers tests the assertion helper functions
func TestAssertionHelpers(t *testing.T) {
	t.Run("assertJSONResponse_ValidResponse", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		handleHealth(w, req)

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"status": "healthy",
		})

		if response == nil {
			t.Error("Expected non-nil response")
		}
	})

	t.Run("assertErrorResponse_ValidError", func(t *testing.T) {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/pregnancy/current",
			UserID: "", // Missing user ID
		})
		handleCurrentPregnancy(w, req)

		assertErrorResponse(t, w, 401)
	})
}

// TestConfigurationVariables tests configuration loading
func TestConfigurationVariables(t *testing.T) {
	t.Run("EncryptionKey_Required", func(t *testing.T) {
		key := os.Getenv("ENCRYPTION_KEY")
		if key == "" {
			t.Skip("ENCRYPTION_KEY not set, test setup handles this")
		}

		if len(key) < 16 {
			t.Error("Encryption key should be at least 16 bytes")
		}
	})

	t.Run("PrivacyMode_Values", func(t *testing.T) {
		mode := os.Getenv("PRIVACY_MODE")
		if mode != "" {
			validModes := map[string]bool{
				"high":   true,
				"medium": true,
				"low":    true,
			}

			if !validModes[mode] {
				t.Logf("Privacy mode '%s' may not be recognized", mode)
			}
		}
	})
}
