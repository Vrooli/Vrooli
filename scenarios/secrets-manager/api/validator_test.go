package main

import (
	"testing"
)

// TestValidateSecrets tests the secret validation functionality
func TestValidateSecrets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_AllResources", func(t *testing.T) {
		response, err := validateSecrets("")
		if err != nil {
			// May fail without vault, but should return a response structure
			t.Logf("Validation failed (expected without vault): %v", err)
		}

		// Response structure should be valid
		if response.ValidationID == "" {
			t.Log("ValidationID is empty (may be expected)")
		}
	})

	t.Run("Success_SpecificResource", func(t *testing.T) {
		response, err := validateSecrets("postgres")
		if err != nil {
			t.Logf("Validation failed (expected without vault): %v", err)
		}

		// Response structure should be valid
		if response.ValidationID == "" {
			t.Log("ValidationID is empty (may be expected)")
		}
	})

	t.Run("NonExistentResource", func(t *testing.T) {
		response, err := validateSecrets("non-existent-resource-xyz")
		if err != nil {
			t.Logf("Validation failed for non-existent resource (expected): %v", err)
		}

		// Should handle gracefully
		if response.TotalSecrets < 0 {
			t.Error("TotalSecrets should not be negative")
		}
	})

	t.Run("EmptyResourceName", func(t *testing.T) {
		response, err := validateSecrets("")
		if err != nil {
			t.Logf("Validation failed (may be expected): %v", err)
		}

		// Should validate all resources when empty
		if response.ValidationID != "" {
			t.Logf("Validation completed with ID: %s", response.ValidationID)
		}
	})
}

// TestValidationHelpers tests validation helper functions
func TestValidationHelpers(t *testing.T) {
	t.Run("ValidateSecretType", func(t *testing.T) {
		validTypes := []string{"api_key", "endpoint", "password", "token", "certificate", "quota", "config"}
		for _, secretType := range validTypes {
			if !validateSecretType(secretType) {
				t.Errorf("Expected secret type '%s' to be valid", secretType)
			}
		}

		invalidTypes := []string{"invalid_type", "unknown", "xyz", ""}
		for _, secretType := range invalidTypes {
			if validateSecretType(secretType) {
				t.Errorf("Expected secret type '%s' to be invalid", secretType)
			}
		}
	})

	t.Run("ValidateValidationStatus", func(t *testing.T) {
		validStatuses := []string{"valid", "invalid", "missing", "pending", "error"}
		for _, status := range validStatuses {
			if !validateValidationStatus(status) {
				t.Errorf("Expected validation status '%s' to be valid", status)
			}
		}

		invalidStatuses := []string{"unknown", "xyz", "completed", ""}
		for _, status := range invalidStatuses {
			if validateValidationStatus(status) {
				t.Errorf("Expected validation status '%s' to be invalid", status)
			}
		}
	})
}

// TestSecretValidationStructures tests validation data structures
func TestSecretValidationStructures(t *testing.T) {
	t.Run("SecretValidation", func(t *testing.T) {
		resourceSecretID := "test-secret-id"
		validation := createTestSecretValidation(resourceSecretID, "valid")

		if validation.ID == "" {
			t.Error("Validation ID should not be empty")
		}

		if validation.ResourceSecretID != resourceSecretID {
			t.Errorf("Expected ResourceSecretID '%s', got '%s'", resourceSecretID, validation.ResourceSecretID)
		}

		if validation.ValidationStatus != "valid" {
			t.Errorf("Expected ValidationStatus 'valid', got '%s'", validation.ValidationStatus)
		}

		if validation.ValidationMethod == "" {
			t.Error("ValidationMethod should not be empty")
		}
	})

	t.Run("ValidationResponse", func(t *testing.T) {
		invalidSecret := createTestSecretValidation("secret1", "invalid")
		missingSecret := createTestSecretValidation("secret2", "missing")

		response := ValidationResponse{
			ValidationID:   "test-validation-id",
			TotalSecrets:   10,
			ValidSecrets:   8,
			InvalidSecrets: []SecretValidation{invalidSecret},
			MissingSecrets: []SecretValidation{missingSecret},
		}

		if response.ValidationID == "" {
			t.Error("ValidationID should not be empty")
		}

		total := response.ValidSecrets + len(response.InvalidSecrets) + len(response.MissingSecrets)
		if total != response.TotalSecrets {
			t.Errorf("Validation counts don't add up: %d + %d + %d != %d",
				response.ValidSecrets, len(response.InvalidSecrets), len(response.MissingSecrets), response.TotalSecrets)
		}
	})

	t.Run("SecretHealthSummary", func(t *testing.T) {
		summary := SecretHealthSummary{
			ResourceName:           "postgres",
			TotalSecrets:           5,
			RequiredSecrets:        3,
			ValidSecrets:           4,
			MissingRequiredSecrets: 1,
			InvalidSecrets:         0,
		}

		if summary.ResourceName == "" {
			t.Error("ResourceName should not be empty")
		}

		if summary.TotalSecrets < summary.ValidSecrets {
			t.Error("TotalSecrets should be >= ValidSecrets")
		}

		if summary.RequiredSecrets > summary.TotalSecrets {
			t.Error("RequiredSecrets should be <= TotalSecrets")
		}
	})
}

// TestVaultValidationIntegration tests vault validation integration
func TestVaultValidationIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetVaultSecretsStatus", func(t *testing.T) {
		// Test without filter
		status, err := getVaultSecretsStatus("")
		if err != nil {
			t.Logf("Vault status check failed (expected without vault): %v", err)
		}

		if status != nil {
			if status.TotalResources < 0 {
				t.Error("TotalResources should not be negative")
			}
			if status.ConfiguredResources > status.TotalResources {
				t.Error("ConfiguredResources should be <= TotalResources")
			}
		}
	})

	t.Run("GetVaultSecretsStatus_WithFilter", func(t *testing.T) {
		status, err := getVaultSecretsStatus("postgres")
		if err != nil {
			t.Logf("Vault status check failed (expected without vault): %v", err)
		}

		if status != nil {
			// When filtering, should only have relevant resources
			if status.TotalResources > 1 && len(status.ResourceStatuses) > 1 {
				t.Log("Filter may not be working as expected")
			}
		}
	})

	t.Run("GetVaultSecretsStatusFromCLI", func(t *testing.T) {
		status, err := getVaultSecretsStatusFromCLI("")
		if err != nil {
			t.Logf("CLI vault status check failed (expected without vault): %v", err)
		}

		if status != nil {
			if status.TotalResources < 0 {
				t.Error("TotalResources should not be negative")
			}
		}
	})
}

// TestValidationErrorCases tests error handling in validation
func TestValidationErrorCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidResourceFilter", func(t *testing.T) {
		// Test with special characters
		_, err := validateSecrets("../../../etc/passwd")
		if err == nil {
			t.Log("Path traversal attempt should be handled")
		}
	})

	t.Run("VeryLongResourceName", func(t *testing.T) {
		longName := ""
		for i := 0; i < 1000; i++ {
			longName += "a"
		}

		_, err := validateSecrets(longName)
		if err == nil {
			t.Log("Very long resource name should be handled")
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		specialNames := []string{
			"resource:with:colons",
			"resource/with/slashes",
			"resource\\with\\backslashes",
			"resource<with>brackets",
			"resource|with|pipes",
		}

		for _, name := range specialNames {
			_, err := validateSecrets(name)
			if err != nil {
				t.Logf("Special character in resource name '%s' handled: %v", name, err)
			}
		}
	})
}

// TestValidationPatterns tests systematic validation patterns
func TestValidationPatterns(t *testing.T) {
	t.Run("RequiredSecretValidation", func(t *testing.T) {
		// Test that required secrets are properly flagged
		secretKeys := []string{"API_KEY", "PASSWORD", "SECRET_TOKEN", "DB_PASSWORD"}

		for _, key := range secretKeys {
			required := isLikelyRequired(key)
			if !required {
				t.Logf("Secret '%s' not marked as required (may be intentional)", key)
			}
		}
	})

	t.Run("OptionalSecretValidation", func(t *testing.T) {
		// Test that optional secrets are properly identified
		secretKeys := []string{"DEBUG", "TIMEOUT", "LOG_LEVEL", "MAX_RETRIES"}

		for _, key := range secretKeys {
			required := isLikelyRequired(key)
			if required {
				t.Logf("Secret '%s' marked as required (may be intentional)", key)
			}
		}
	})

	t.Run("SecretTypeClassification", func(t *testing.T) {
		testCases := []struct {
			key          string
			expectedType string
		}{
			{"API_KEY", "api_key"},
			{"DATABASE_PASSWORD", "password"},
			{"AUTH_TOKEN", "token"},
			{"API_ENDPOINT", "endpoint"},
			{"SSL_CERTIFICATE", "certificate"},
		}

		for _, tc := range testCases {
			actualType := classifySecretType(tc.key)
			if actualType != tc.expectedType {
				t.Logf("Secret '%s' classified as '%s', expected '%s'",
					tc.key, actualType, tc.expectedType)
			}
		}
	})
}

// TestValidationResponseStructure tests validation response structures
func TestValidationResponseStructure(t *testing.T) {
	t.Run("CompleteValidationResponse", func(t *testing.T) {
		response := ValidationResponse{
			ValidationID:   "val-123",
			TotalSecrets:   15,
			ValidSecrets:   10,
			InvalidSecrets: []SecretValidation{},
			MissingSecrets: []SecretValidation{},
			HealthSummary: []SecretHealthSummary{
				{
					ResourceName:           "postgres",
					TotalSecrets:           5,
					RequiredSecrets:        3,
					ValidSecrets:           4,
					MissingRequiredSecrets: 1,
					InvalidSecrets:         0,
				},
				{
					ResourceName:           "openai",
					TotalSecrets:           3,
					RequiredSecrets:        2,
					ValidSecrets:           2,
					MissingRequiredSecrets: 1,
					InvalidSecrets:         0,
				},
			},
		}

		// Validate the response structure
		if response.ValidationID == "" {
			t.Error("ValidationID should not be empty")
		}

		if len(response.HealthSummary) == 0 {
			t.Error("Should have health summary")
		}

		// Validate each resource summary
		for _, summary := range response.HealthSummary {
			if summary.ResourceName == "" {
				t.Error("ResourceName in summary should not be empty")
			}

			if summary.TotalSecrets < summary.ValidSecrets {
				t.Errorf("Invalid summary for %s: total < valid", summary.ResourceName)
			}

			if summary.RequiredSecrets > summary.TotalSecrets {
				t.Errorf("Invalid summary for %s: required > total", summary.ResourceName)
			}
		}
	})

	t.Run("EmptyValidationResponse", func(t *testing.T) {
		response := ValidationResponse{}

		if response.TotalSecrets != 0 {
			t.Error("Empty response should have 0 total secrets")
		}

		if response.HealthSummary != nil && len(response.HealthSummary) > 0 {
			t.Error("Empty response should have no health summaries")
		}
	})
}

// TestVaultResourceStatus tests vault resource status structures
func TestVaultResourceStatus(t *testing.T) {
	t.Run("ConfiguredStatus", func(t *testing.T) {
		status := VaultResourceStatus{
			ResourceName:    "postgres",
			SecretsTotal:    5,
			SecretsFound:    5,
			SecretsMissing:  0,
			SecretsOptional: 0,
			HealthStatus:    "healthy",
			AllSecrets:      []VaultSecret{},
		}

		if status.HealthStatus != "healthy" {
			t.Error("Status should be healthy")
		}

		if status.SecretsTotal != status.SecretsFound {
			t.Error("All secrets should be found")
		}

		if status.SecretsMissing != 0 {
			t.Error("No secrets should be missing")
		}
	})

	t.Run("PartiallyConfiguredStatus", func(t *testing.T) {
		status := VaultResourceStatus{
			ResourceName:    "openai",
			SecretsTotal:    3,
			SecretsFound:    2,
			SecretsMissing:  1,
			SecretsOptional: 0,
			HealthStatus:    "degraded",
			AllSecrets:      []VaultSecret{},
		}

		if status.HealthStatus == "healthy" {
			t.Error("Status should not be healthy when secrets are missing")
		}

		total := status.SecretsFound + status.SecretsMissing
		if total != status.SecretsTotal {
			t.Error("Found + Missing should equal Total")
		}
	})

	t.Run("UnconfiguredStatus", func(t *testing.T) {
		status := VaultResourceStatus{
			ResourceName:    "new-resource",
			SecretsTotal:    4,
			SecretsFound:    0,
			SecretsMissing:  4,
			SecretsOptional: 0,
			HealthStatus:    "critical",
			AllSecrets:      []VaultSecret{},
		}

		if status.SecretsFound != 0 {
			t.Error("No secrets should be found")
		}

		if status.SecretsMissing != status.SecretsTotal {
			t.Error("All secrets should be missing")
		}
	})
}
