// Package agents provides agent lifecycle management.
// This file contains spawn request validation logic, extracted from handlers
// to localize change and enable testing.
//
// CHANGE_AXIS: Spawn Validation Architecture
// There are TWO layers of spawn validation, each with a different concern:
//
// 1. FIELD VALIDATION (this file - spawn_validator.go)
//    - Validates that required fields are present and properly formatted
//    - Checks: prompts non-empty, model specified, scenario specified
//    - Does NOT involve security policy - just "is the request well-formed?"
//    - Use ValidateSpawnRequest() for this layer
//
// 2. SECURITY VALIDATION (spawn_security_validator.go)
//    - Validates that the request is SAFE to execute
//    - Checks: tools allowlist, prompt content, path boundaries, skipPermissions
//    - Involves security policy decisions
//    - Use SpawnSecurityValidator.ValidateAll() for this layer
//
// Handler flow: Field validation → Security validation → Execute
// Field validation fails fast on malformed requests before expensive security checks.
package agents

import (
	"fmt"
	"strings"
)

// SpawnValidationInput contains all inputs needed to validate a spawn request.
type SpawnValidationInput struct {
	Prompts         []string
	Model           string
	Scenario        string
	Scope           []string
	SkipPermissions bool
	AllowedTools    []string
	MaxPrompts      int  // From config
	NetworkEnabled  bool // Request flag
}

// SpawnValidationResult contains the outcome of spawn validation.
type SpawnValidationResult struct {
	// IsValid indicates whether the spawn request passed all validation.
	IsValid bool

	// Prompts contains the validated and sanitized prompts.
	Prompts []string

	// Capped indicates whether prompts were truncated due to max limit.
	Capped bool

	// ValidationErrors contains any errors that caused validation to fail.
	ValidationErrors []SpawnValidationError

	// Warnings contains non-fatal issues to surface to the caller.
	Warnings []string
}

// SpawnValidationError represents a specific validation failure.
type SpawnValidationError struct {
	Field   string // Which field failed validation
	Code    string // Machine-readable error code
	Message string // Human-readable error message
}

// Error returns the validation error message.
func (e SpawnValidationError) Error() string {
	return e.Message
}

// SpawnValidationErrorCode defines machine-readable validation error codes.
type SpawnValidationErrorCode string

const (
	// ValidationCodeNoPrompts indicates no valid prompts were provided.
	ValidationCodeNoPrompts SpawnValidationErrorCode = "no_prompts"

	// ValidationCodeNoModel indicates no model was specified.
	ValidationCodeNoModel SpawnValidationErrorCode = "no_model"

	// ValidationCodeNoScenario indicates no scenario was specified.
	ValidationCodeNoScenario SpawnValidationErrorCode = "no_scenario"

	// ValidationCodeSkipPermissions indicates skipPermissions was set (not allowed).
	ValidationCodeSkipPermissions SpawnValidationErrorCode = "skip_permissions_blocked"

	// ValidationCodeInvalidTools indicates invalid tools were specified.
	ValidationCodeInvalidTools SpawnValidationErrorCode = "invalid_tools"

	// ValidationCodeInvalidScope indicates invalid scope paths were specified.
	ValidationCodeInvalidScope SpawnValidationErrorCode = "invalid_scope"

	// ValidationCodeDangerousPrompt indicates a prompt contains blocked content.
	ValidationCodeDangerousPrompt SpawnValidationErrorCode = "dangerous_prompt"
)

// ValidateSpawnRequest performs all validation checks on a spawn request.
// This is the central validation decision point for agent spawning.
//
// Validation steps (in order):
//  1. Sanitize and validate prompts (trim, remove empty, check count)
//  2. Validate model is specified
//  3. Validate scenario is specified
//  4. Block skipPermissions (security check)
//
// Additional validation (scope paths, tools, prompt content) should be performed
// by the caller using the security package, as they require additional context
// (repo root, tool validator, etc.).
//
// Decision: Validation is performed in priority order - early failures prevent
// unnecessary work on later checks.
func ValidateSpawnRequest(input SpawnValidationInput) SpawnValidationResult {
	result := SpawnValidationResult{
		IsValid:          true,
		ValidationErrors: make([]SpawnValidationError, 0),
		Warnings:         make([]string, 0),
	}

	// Step 1: Sanitize and validate prompts
	prompts := sanitizePrompts(input.Prompts)
	if len(prompts) == 0 {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, SpawnValidationError{
			Field:   "prompts",
			Code:    string(ValidationCodeNoPrompts),
			Message: "at least one prompt is required",
		})
		return result // Early return - no point continuing
	}

	// Apply prompt limit
	if input.MaxPrompts > 0 && len(prompts) > input.MaxPrompts {
		prompts = prompts[:input.MaxPrompts]
		result.Capped = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("prompts truncated to max %d", input.MaxPrompts))
	}
	result.Prompts = prompts

	// Step 2: Validate model
	model := strings.TrimSpace(input.Model)
	if model == "" {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, SpawnValidationError{
			Field:   "model",
			Code:    string(ValidationCodeNoModel),
			Message: "model is required",
		})
	}

	// Step 3: Validate scenario
	scenario := strings.TrimSpace(input.Scenario)
	if scenario == "" {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, SpawnValidationError{
			Field:   "scenario",
			Code:    string(ValidationCodeNoScenario),
			Message: "scenario is required for agent spawning",
		})
	}

	// Step 4: Block skipPermissions (SECURITY)
	if input.SkipPermissions {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, SpawnValidationError{
			Field:   "skipPermissions",
			Code:    string(ValidationCodeSkipPermissions),
			Message: "skipPermissions is not allowed for spawned agents; this would bypass all safety controls",
		})
	}

	return result
}

// sanitizePrompts removes empty prompts and trims whitespace.
func sanitizePrompts(prompts []string) []string {
	sanitized := make([]string, 0, len(prompts))
	for _, p := range prompts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			sanitized = append(sanitized, trimmed)
		}
	}
	return sanitized
}

// FirstError returns the first validation error message, or empty string if valid.
// Useful for simple error responses.
func (r SpawnValidationResult) FirstError() string {
	if len(r.ValidationErrors) > 0 {
		return r.ValidationErrors[0].Message
	}
	return ""
}

// ErrorsByField groups validation errors by field name.
// Useful for rendering field-level errors in forms.
func (r SpawnValidationResult) ErrorsByField() map[string][]SpawnValidationError {
	result := make(map[string][]SpawnValidationError)
	for _, err := range r.ValidationErrors {
		result[err.Field] = append(result[err.Field], err)
	}
	return result
}
