package agents

import (
	"testing"
)

func TestValidateSpawnRequest(t *testing.T) {
	t.Run("valid request passes", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:    []string{"Write a test for the user handler"},
			Model:      "openrouter/anthropic/claude-3-opus",
			Scenario:   "test-genie",
			MaxPrompts: 20,
		}

		result := ValidateSpawnRequest(input)

		if !result.IsValid {
			t.Errorf("Expected valid result, got errors: %v", result.ValidationErrors)
		}
		if len(result.Prompts) != 1 {
			t.Errorf("Expected 1 prompt, got %d", len(result.Prompts))
		}
		if result.Capped {
			t.Error("Expected not capped")
		}
	})

	t.Run("empty prompts fails", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:  []string{},
			Model:    "openrouter/anthropic/claude-3-opus",
			Scenario: "test-genie",
		}

		result := ValidateSpawnRequest(input)

		if result.IsValid {
			t.Error("Expected validation to fail for empty prompts")
		}
		if len(result.ValidationErrors) == 0 {
			t.Error("Expected at least one validation error")
		}
		if result.ValidationErrors[0].Code != string(ValidationCodeNoPrompts) {
			t.Errorf("Expected error code %s, got %s", ValidationCodeNoPrompts, result.ValidationErrors[0].Code)
		}
	})

	t.Run("whitespace-only prompts fail", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:  []string{"   ", "\t", "\n"},
			Model:    "openrouter/anthropic/claude-3-opus",
			Scenario: "test-genie",
		}

		result := ValidateSpawnRequest(input)

		if result.IsValid {
			t.Error("Expected validation to fail for whitespace-only prompts")
		}
	})

	t.Run("prompts are trimmed", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:    []string{"  test prompt  ", "", "  another  "},
			Model:      "openrouter/anthropic/claude-3-opus",
			Scenario:   "test-genie",
			MaxPrompts: 20,
		}

		result := ValidateSpawnRequest(input)

		if !result.IsValid {
			t.Errorf("Expected valid result, got errors: %v", result.ValidationErrors)
		}
		if len(result.Prompts) != 2 {
			t.Errorf("Expected 2 prompts after sanitization, got %d", len(result.Prompts))
		}
		if result.Prompts[0] != "test prompt" {
			t.Errorf("Expected trimmed prompt, got %q", result.Prompts[0])
		}
	})

	t.Run("prompts capped at max", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:    []string{"one", "two", "three", "four", "five"},
			Model:      "openrouter/anthropic/claude-3-opus",
			Scenario:   "test-genie",
			MaxPrompts: 3,
		}

		result := ValidateSpawnRequest(input)

		if !result.IsValid {
			t.Errorf("Expected valid result, got errors: %v", result.ValidationErrors)
		}
		if len(result.Prompts) != 3 {
			t.Errorf("Expected 3 prompts after capping, got %d", len(result.Prompts))
		}
		if !result.Capped {
			t.Error("Expected Capped to be true")
		}
	})

	t.Run("missing model fails", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:  []string{"test prompt"},
			Model:    "",
			Scenario: "test-genie",
		}

		result := ValidateSpawnRequest(input)

		if result.IsValid {
			t.Error("Expected validation to fail for missing model")
		}
		found := false
		for _, err := range result.ValidationErrors {
			if err.Code == string(ValidationCodeNoModel) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error code %s, got %v", ValidationCodeNoModel, result.ValidationErrors)
		}
	})

	t.Run("missing scenario fails", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:  []string{"test prompt"},
			Model:    "openrouter/anthropic/claude-3-opus",
			Scenario: "",
		}

		result := ValidateSpawnRequest(input)

		if result.IsValid {
			t.Error("Expected validation to fail for missing scenario")
		}
		found := false
		for _, err := range result.ValidationErrors {
			if err.Code == string(ValidationCodeNoScenario) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error code %s, got %v", ValidationCodeNoScenario, result.ValidationErrors)
		}
	})

	t.Run("skipPermissions blocked", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:         []string{"test prompt"},
			Model:           "openrouter/anthropic/claude-3-opus",
			Scenario:        "test-genie",
			SkipPermissions: true,
		}

		result := ValidateSpawnRequest(input)

		if result.IsValid {
			t.Error("Expected validation to fail for skipPermissions")
		}
		found := false
		for _, err := range result.ValidationErrors {
			if err.Code == string(ValidationCodeSkipPermissions) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error code %s, got %v", ValidationCodeSkipPermissions, result.ValidationErrors)
		}
	})

	t.Run("multiple errors collected", func(t *testing.T) {
		input := SpawnValidationInput{
			Prompts:         []string{"test"},
			Model:           "", // missing
			Scenario:        "", // missing
			SkipPermissions: true,
		}

		result := ValidateSpawnRequest(input)

		if result.IsValid {
			t.Error("Expected validation to fail")
		}
		if len(result.ValidationErrors) < 3 {
			t.Errorf("Expected at least 3 errors, got %d: %v", len(result.ValidationErrors), result.ValidationErrors)
		}
	})
}

func TestSpawnValidationResultHelpers(t *testing.T) {
	t.Run("FirstError returns empty for valid", func(t *testing.T) {
		result := SpawnValidationResult{IsValid: true}
		if result.FirstError() != "" {
			t.Errorf("Expected empty string, got %q", result.FirstError())
		}
	})

	t.Run("FirstError returns first error", func(t *testing.T) {
		result := SpawnValidationResult{
			IsValid: false,
			ValidationErrors: []SpawnValidationError{
				{Field: "model", Code: "no_model", Message: "model is required"},
				{Field: "scenario", Code: "no_scenario", Message: "scenario is required"},
			},
		}
		if result.FirstError() != "model is required" {
			t.Errorf("Expected 'model is required', got %q", result.FirstError())
		}
	})

	t.Run("ErrorsByField groups correctly", func(t *testing.T) {
		result := SpawnValidationResult{
			IsValid: false,
			ValidationErrors: []SpawnValidationError{
				{Field: "model", Code: "no_model", Message: "model is required"},
				{Field: "prompts", Code: "no_prompts", Message: "prompts required"},
				{Field: "model", Code: "invalid_model", Message: "model not found"},
			},
		}
		byField := result.ErrorsByField()
		if len(byField["model"]) != 2 {
			t.Errorf("Expected 2 model errors, got %d", len(byField["model"]))
		}
		if len(byField["prompts"]) != 1 {
			t.Errorf("Expected 1 prompts error, got %d", len(byField["prompts"]))
		}
	})
}
