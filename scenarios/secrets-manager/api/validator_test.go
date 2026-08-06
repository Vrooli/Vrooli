package main

import (
	"context"
	"testing"
)

// TestNewSecretValidator tests validator initialization
// [REQ:SEC-VLT-003] Secret validation framework
func TestNewSecretValidator(t *testing.T) {
	validator := NewSecretValidator(nil)
	if validator == nil {
		t.Fatal("NewSecretValidator() returned nil")
	}
	if validator.db != nil {
		t.Error("Expected nil database, got non-nil")
	}
}

// TestValidateSecrets tests the secret validation functionality
// [REQ:SEC-VLT-003] Secret validation framework
func TestValidateSecrets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabase_EmptySecrets", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		// With no DB, should return empty results without error
		result, err := validator.ValidateSecrets("")
		if err != nil {
			t.Fatalf("ValidateSecrets failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if result.ValidationID == "" {
			t.Error("Expected validation ID to be set")
		}

		// Should return empty lists since no DB means no secrets
		if result.TotalSecrets != 0 {
			t.Errorf("Expected 0 total secrets without DB, got %d", result.TotalSecrets)
		}
	})

	t.Run("NoDatabase_SpecificResource", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		// With no DB, should return empty results even for specific resource
		result, err := validator.ValidateSecrets("postgres")
		if err != nil {
			t.Fatalf("ValidateSecrets failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if result.TotalSecrets != 0 {
			t.Errorf("Expected 0 total secrets without DB, got %d", result.TotalSecrets)
		}
	})
}

// TestValidateSecret tests single secret validation
// [REQ:SEC-VLT-003] Secret validation framework
func TestValidateSecret(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	prior := credentialStatusCommand
	credentialStatusCommand = func(_ context.Context, _, _ string) ([]byte, error) { return []byte(`{"configured":true}`), nil }
	t.Cleanup(func() { credentialStatusCommand = prior })
	result := NewSecretValidator(nil).validateSecret(ResourceSecret{ID: "test-id", ResourceName: "openrouter", SecretKey: "OPENROUTER_API_KEY", Required: true})
	if result.ValidationStatus != "valid" || result.ValidationMethod != string(ValidationMethodAuthority) {
		t.Fatalf("validation = %#v", result)
	}
}

// TestStoreValidationResult tests validation result storage
func TestStoreValidationResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabase", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		validation := SecretValidation{
			ID:               "test-id",
			ResourceSecretID: "secret-id",
			ValidationStatus: "valid",
			ValidationMethod: "env",
		}

		// Should not error with nil DB
		err := validator.storeValidationResult(validation)
		if err != nil {
			t.Errorf("Expected no error with nil DB, got: %v", err)
		}
	})
}

// TestGetSecretsForValidation tests secret retrieval for validation
func TestGetSecretsForValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabase_AllResources", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		secrets, err := validator.getSecretsForValidation("")
		if err != nil {
			t.Fatalf("Expected no error with nil DB, got: %v", err)
		}

		if len(secrets) != 0 {
			t.Errorf("Expected empty secrets list with nil DB, got %d secrets", len(secrets))
		}
	})

	t.Run("NoDatabase_SpecificResource", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		secrets, err := validator.getSecretsForValidation("postgres")
		if err != nil {
			t.Fatalf("Expected no error with nil DB, got: %v", err)
		}

		if len(secrets) != 0 {
			t.Errorf("Expected empty secrets list with nil DB, got %d secrets", len(secrets))
		}
	})
}

// TestGenerateHealthSummary tests health summary generation
func TestGenerateHealthSummary(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabase", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		summary, err := validator.generateHealthSummary("")
		if err != nil {
			t.Fatalf("Expected no error with nil DB, got: %v", err)
		}

		if len(summary) != 0 {
			t.Errorf("Expected empty summary with nil DB, got %d entries", len(summary))
		}
	})

	t.Run("NoDatabase_SpecificResource", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		summary, err := validator.generateHealthSummary("postgres")
		if err != nil {
			t.Fatalf("Expected no error with nil DB, got: %v", err)
		}

		if len(summary) != 0 {
			t.Errorf("Expected empty summary with nil DB, got %d entries", len(summary))
		}
	})
}

// TestGetValidationHistory tests validation history retrieval
func TestGetValidationHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabase", func(t *testing.T) {
		validator := NewSecretValidator(nil)

		history, err := validator.GetValidationHistory("", 10)
		if err != nil {
			t.Fatalf("Expected no error with nil DB, got: %v", err)
		}

		if len(history) != 0 {
			t.Errorf("Expected empty history with nil DB, got %d entries", len(history))
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

// TestValidationPatterns tests systematic validation patterns
func TestValidationPatterns(t *testing.T) {
	t.Run("RequiredSecretValidation", func(t *testing.T) {
		// Test that required secrets are properly flagged
		secretKeys := []string{"API_KEY", "PASSWORD", "SECRET_TOKEN", "DB_PASSWORD"}

		for _, key := range secretKeys {
			required := IsLikelyRequired(key)
			if !required {
				t.Errorf("IsLikelyRequired(%q) = false, want true", key)
			}
		}
	})

	t.Run("OptionalSecretValidation", func(t *testing.T) {
		// Test that optional secrets are properly identified
		secretKeys := []string{"DEBUG", "TIMEOUT", "LOG_LEVEL", "MAX_RETRIES"}

		for _, key := range secretKeys {
			required := IsLikelyRequired(key)
			if required {
				t.Errorf("IsLikelyRequired(%q) = true, want false", key)
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
			{"API_ENDPOINT", "env_var"},
			{"SSL_CERTIFICATE", "certificate"},
		}

		for _, tc := range testCases {
			actualType := ClassifySecretType(tc.key)
			if actualType != tc.expectedType {
				t.Errorf("ClassifySecretType(%q) = %q, want %q", tc.key, actualType, tc.expectedType)
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

		if len(response.HealthSummary) > 0 {
			t.Error("Empty response should have no health summaries")
		}
	})
}

// TestCredentialResourceStatus tests vault resource status structures
func TestCredentialResourceStatus(t *testing.T) {
	t.Run("ConfiguredStatus", func(t *testing.T) {
		status := CredentialResourceStatus{
			ResourceName:    "postgres",
			SecretsTotal:    5,
			SecretsFound:    5,
			SecretsMissing:  0,
			SecretsOptional: 0,
			HealthStatus:    "healthy",
			AllSecrets:      []CredentialStatus{},
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
		status := CredentialResourceStatus{
			ResourceName:    "openai",
			SecretsTotal:    3,
			SecretsFound:    2,
			SecretsMissing:  1,
			SecretsOptional: 0,
			HealthStatus:    "degraded",
			AllSecrets:      []CredentialStatus{},
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
		status := CredentialResourceStatus{
			ResourceName:    "new-resource",
			SecretsTotal:    4,
			SecretsFound:    0,
			SecretsMissing:  4,
			SecretsOptional: 0,
			HealthStatus:    "critical",
			AllSecrets:      []CredentialStatus{},
		}

		if status.SecretsFound != 0 {
			t.Error("No secrets should be found")
		}

		if status.SecretsMissing != status.SecretsTotal {
			t.Error("All secrets should be missing")
		}
	})
}
