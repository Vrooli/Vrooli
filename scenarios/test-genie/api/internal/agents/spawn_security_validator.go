// Package agents provides agent lifecycle management.
// This file contains security validation for spawn requests, consolidating
// all "is this request allowed?" logic in one place.
//
// CHANGE_AXIS: Spawn Security Policy
// When modifying what spawn requests are allowed, changes should be localized here:
//   - SpawnSecurityValidator interface: Contract for validation
//   - DefaultSpawnSecurityValidator: Production implementation
//   - SecurityValidationResult: What checks were performed and outcomes
//
// The handler should NOT implement security validation logic directly.
// It should call the validator and act on results.
//
// NOTE: This is the SECURITY VALIDATION layer. For field validation (required fields,
// formatting), see spawn_validator.go. The two layers work together:
// Field validation → Security validation → Execute
package agents

import (
	"fmt"

	"test-genie/internal/security"
)

// SpawnSecurityValidator validates spawn requests for security concerns.
// This is the central decision point for "is this spawn request safe?"
//
// CHANGE_AXIS: Spawn Security Policy
// All security checks for spawn requests should be methods on this interface.
// Adding a new security check = adding a method here + implementing it.
type SpawnSecurityValidator interface {
	// ValidateTools checks if the allowed tools list is safe.
	ValidateTools(tools []string) SecurityValidationResult

	// ValidatePrompts checks if prompts contain dangerous content.
	ValidatePrompts(prompts []string) SecurityValidationResult

	// ValidateScopePaths checks if scope paths are within allowed boundaries.
	ValidateScopePaths(scenario string, scopePaths []string, repoRoot string) SecurityValidationResult

	// ValidateSkipPermissions checks if skipPermissions is allowed (it's not).
	ValidateSkipPermissions(skipPermissions bool) SecurityValidationResult

	// ValidateAll performs all security validations and returns combined result.
	// This is the primary entry point for spawn request security validation.
	ValidateAll(input SpawnSecurityInput) SpawnSecurityResult
}

// SpawnSecurityInput contains all inputs needed for security validation.
// This consolidates all the fields that affect security decisions.
type SpawnSecurityInput struct {
	Prompts         []string
	AllowedTools    []string
	SkipPermissions bool
	Scenario        string
	ScopePaths      []string
	RepoRoot        string
}

// SecurityValidationResult describes the outcome of a single security check.
type SecurityValidationResult struct {
	Valid   bool
	Code    string   // Machine-readable code for the check type
	Message string   // Human-readable message
	Details []string // Additional details (e.g., which prompt failed)
}

// SpawnSecurityResult contains the combined outcome of all security validations.
type SpawnSecurityResult struct {
	// Valid is true only if ALL checks passed.
	Valid bool

	// Checks contains results for each individual check performed.
	Checks []SecurityValidationResult

	// FirstError returns the first failing check's message, or empty if valid.
	FirstError string

	// SanitizedTools contains the validated and sanitized tools list.
	// Only populated if tools validation passed.
	SanitizedTools []string
}

// GetFailedChecks returns only the checks that failed.
func (r SpawnSecurityResult) GetFailedChecks() []SecurityValidationResult {
	failed := make([]SecurityValidationResult, 0)
	for _, check := range r.Checks {
		if !check.Valid {
			failed = append(failed, check)
		}
	}
	return failed
}

// --- Security Check Codes ---
// These provide machine-readable identifiers for each type of security check.

const (
	// SecurityCheckTools validates the allowed tools list.
	SecurityCheckTools = "tools_validation"

	// SecurityCheckPrompts validates prompt content.
	SecurityCheckPrompts = "prompt_validation"

	// SecurityCheckScopePaths validates scope path boundaries.
	SecurityCheckScopePaths = "scope_paths_validation"

	// SecurityCheckSkipPermissions validates that skipPermissions is not set.
	SecurityCheckSkipPermissions = "skip_permissions_validation"
)

// --- Default Implementation ---

// DefaultSpawnSecurityValidator is the production implementation of SpawnSecurityValidator.
// It uses the security package for actual validation logic.
type DefaultSpawnSecurityValidator struct {
	bashValidator *security.BashCommandValidator
}

// NewSpawnSecurityValidator creates a new spawn security validator.
func NewSpawnSecurityValidator() *DefaultSpawnSecurityValidator {
	return &DefaultSpawnSecurityValidator{
		bashValidator: security.DefaultValidator(),
	}
}

// NewSpawnSecurityValidatorWithBashValidator creates a validator with a custom bash validator.
// This is primarily for testing.
func NewSpawnSecurityValidatorWithBashValidator(bv *security.BashCommandValidator) *DefaultSpawnSecurityValidator {
	return &DefaultSpawnSecurityValidator{
		bashValidator: bv,
	}
}

// ValidateTools checks if the allowed tools list is safe.
//
// Decision criteria:
//   - Empty tools list is OK (will use defaults)
//   - Wildcard (*) is NOT allowed
//   - Unrestricted bash is NOT allowed
//   - Bash patterns must only contain allowlisted commands
func (v *DefaultSpawnSecurityValidator) ValidateTools(tools []string) SecurityValidationResult {
	result := SecurityValidationResult{
		Valid: true,
		Code:  SecurityCheckTools,
	}

	if len(tools) == 0 {
		result.Message = "using default safe tools"
		return result
	}

	sanitized, err := security.SanitizeAllowedTools(tools)
	if err != nil {
		result.Valid = false
		result.Message = fmt.Sprintf("invalid allowed tools: %s", err.Error())
		return result
	}

	result.Message = fmt.Sprintf("validated %d tools", len(sanitized))
	result.Details = sanitized
	return result
}

// ValidatePrompts checks if prompts contain dangerous content.
//
// Decision criteria:
//   - Prompts must not be empty
//   - Prompts must not exceed max length (from config)
//   - Prompts must not contain blocked patterns (destructive commands, etc.)
func (v *DefaultSpawnSecurityValidator) ValidatePrompts(prompts []string) SecurityValidationResult {
	result := SecurityValidationResult{
		Valid: true,
		Code:  SecurityCheckPrompts,
	}

	if len(prompts) == 0 {
		result.Valid = false
		result.Message = "at least one prompt is required"
		return result
	}

	for i, prompt := range prompts {
		if err := v.bashValidator.ValidatePrompt(prompt); err != nil {
			result.Valid = false
			result.Message = fmt.Sprintf("prompt %d contains blocked content: %s", i+1, err.Error())
			result.Details = append(result.Details, fmt.Sprintf("prompt[%d]: %s", i, err.Error()))
			// Continue checking other prompts to report all issues
		}
	}

	if result.Valid {
		result.Message = fmt.Sprintf("validated %d prompts", len(prompts))
	}
	return result
}

// ValidateScopePaths checks if scope paths are within allowed boundaries.
//
// Decision criteria:
//   - Empty scope is OK (means entire scenario directory)
//   - Paths must not escape scenario directory
//   - Paths must not contain path traversal sequences
//   - Home directory references (~) are NOT allowed
func (v *DefaultSpawnSecurityValidator) ValidateScopePaths(scenario string, scopePaths []string, repoRoot string) SecurityValidationResult {
	result := SecurityValidationResult{
		Valid: true,
		Code:  SecurityCheckScopePaths,
	}

	if len(scopePaths) == 0 {
		result.Message = "empty scope (entire scenario directory allowed)"
		return result
	}

	if err := security.ValidateScopePaths(scenario, scopePaths, repoRoot); err != nil {
		result.Valid = false
		result.Message = fmt.Sprintf("invalid scope paths: %s", err.Error())
		return result
	}

	result.Message = fmt.Sprintf("validated %d scope paths", len(scopePaths))
	result.Details = scopePaths
	return result
}

// ValidateSkipPermissions checks if skipPermissions is allowed.
//
// Decision criteria:
//   - skipPermissions is NEVER allowed for spawned agents
//   - This would bypass ALL safety controls
//   - This is a hard security boundary, not configurable
func (v *DefaultSpawnSecurityValidator) ValidateSkipPermissions(skipPermissions bool) SecurityValidationResult {
	result := SecurityValidationResult{
		Valid: true,
		Code:  SecurityCheckSkipPermissions,
	}

	if skipPermissions {
		result.Valid = false
		result.Message = "skipPermissions is not allowed for spawned agents; this would bypass all safety controls"
		return result
	}

	result.Message = "skipPermissions not requested"
	return result
}

// ValidateAll performs all security validations and returns combined result.
// This is the primary entry point for spawn request security validation.
//
// Validation order:
//  1. skipPermissions (fast fail - hard security boundary)
//  2. tools (affects what agent can do)
//  3. scope paths (affects where agent can access)
//  4. prompts (affects what agent is asked to do)
//
// All checks are performed even if early ones fail, to provide complete feedback.
func (v *DefaultSpawnSecurityValidator) ValidateAll(input SpawnSecurityInput) SpawnSecurityResult {
	result := SpawnSecurityResult{
		Valid:  true,
		Checks: make([]SecurityValidationResult, 0, 4),
	}

	// Check 1: skipPermissions (hard security boundary)
	skipCheck := v.ValidateSkipPermissions(input.SkipPermissions)
	result.Checks = append(result.Checks, skipCheck)
	if !skipCheck.Valid {
		result.Valid = false
		if result.FirstError == "" {
			result.FirstError = skipCheck.Message
		}
	}

	// Check 2: tools validation
	toolsCheck := v.ValidateTools(input.AllowedTools)
	result.Checks = append(result.Checks, toolsCheck)
	if !toolsCheck.Valid {
		result.Valid = false
		if result.FirstError == "" {
			result.FirstError = toolsCheck.Message
		}
	} else if len(toolsCheck.Details) > 0 {
		// Store sanitized tools for use by caller
		result.SanitizedTools = toolsCheck.Details
	}

	// Check 3: scope paths validation
	scopeCheck := v.ValidateScopePaths(input.Scenario, input.ScopePaths, input.RepoRoot)
	result.Checks = append(result.Checks, scopeCheck)
	if !scopeCheck.Valid {
		result.Valid = false
		if result.FirstError == "" {
			result.FirstError = scopeCheck.Message
		}
	}

	// Check 4: prompts validation
	promptsCheck := v.ValidatePrompts(input.Prompts)
	result.Checks = append(result.Checks, promptsCheck)
	if !promptsCheck.Valid {
		result.Valid = false
		if result.FirstError == "" {
			result.FirstError = promptsCheck.Message
		}
	}

	return result
}

// --- Convenience Functions ---

// ValidateSpawnSecurity is a convenience function that creates a validator and runs all checks.
// Use this for simple cases where you don't need to customize the validator.
func ValidateSpawnSecurity(input SpawnSecurityInput) SpawnSecurityResult {
	validator := NewSpawnSecurityValidator()
	return validator.ValidateAll(input)
}

// GetDefaultSafeTools returns the default safe tools list.
// This is a pass-through to the security package for convenience.
func GetDefaultSafeTools() []string {
	return security.DefaultSafeTools()
}

// SanitizePrompts removes empty prompts and trims whitespace.
// This is a convenience alias for the sanitizePrompts function in spawn_validator.go.
// Use this before security validation to ensure prompts are normalized.
func SanitizePrompts(prompts []string) []string {
	return sanitizePrompts(prompts)
}
