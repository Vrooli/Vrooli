package main

import (
	"fmt"
	"strings"
	"time"
)

// DeploymentErrorKind categorizes validation errors by their type.
// This enables consistent HTTP status code mapping and error handling.
type DeploymentErrorKind int

const (
	// DeploymentErrorMissingID indicates the profile_id was not provided.
	DeploymentErrorMissingID DeploymentErrorKind = iota

	// DeploymentErrorNotFound indicates the profile does not exist.
	DeploymentErrorNotFound

	// DeploymentErrorValidation indicates a general validation failure.
	DeploymentErrorValidation
)

// DeploymentValidationError represents a typed validation error.
type DeploymentValidationError struct {
	Kind    DeploymentErrorKind
	Message string
}

// DeploymentValidationResult holds the result of validating a deployment profile.
type DeploymentValidationResult struct {
	Valid          bool
	Errors         []DeploymentValidationError
	ProfileID      string
	DeploymentID   string
	RecommendedFix string
}

// ErrorMessages returns the error messages as strings for API responses.
func (r DeploymentValidationResult) ErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		messages[i] = e.Message
	}
	return messages
}

// ValidateDeploymentProfile validates a profile for deployment readiness.
// Returns validation errors if the profile cannot be deployed.
func ValidateDeploymentProfile(profileID string) DeploymentValidationResult {
	result := DeploymentValidationResult{
		ProfileID: profileID,
		Valid:     true,
	}

	// Decision: Is profile_id provided?
	if profileID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, DeploymentValidationError{
			Kind:    DeploymentErrorMissingID,
			Message: "profile_id is required",
		})
		return result
	}

	// Decision: Does profile_id match valid format patterns?
	// Valid formats: "profile-*" or "test-*"
	if !isValidProfileIDFormat(profileID) {
		result.Valid = false
		result.Errors = append(result.Errors, DeploymentValidationError{
			Kind:    DeploymentErrorNotFound,
			Message: fmt.Sprintf("Profile '%s' not found", profileID),
		})
		return result
	}

	// Decision: Are required packagers available? (simplified example)
	if profileID == "missing-packager-profile" {
		result.Valid = false
		result.Errors = append(result.Errors, DeploymentValidationError{
			Kind:    DeploymentErrorValidation,
			Message: "Required packager 'scenario-to-desktop' not found",
		})
		result.RecommendedFix = "Install required packagers or update profile configuration"
	}

	return result
}

// isValidProfileIDFormat decides whether a profile ID matches expected naming patterns.
// Valid patterns: "profile-*" or "test-*"
func isValidProfileIDFormat(profileID string) bool {
	return strings.HasPrefix(profileID, "profile-") || strings.HasPrefix(profileID, "test-")
}

// GenerateDeploymentID creates a unique deployment ID based on timestamp.
func GenerateDeploymentID() string {
	return fmt.Sprintf("deploy-%d", time.Now().Unix())
}

// httpStatusForErrorKind decides the HTTP status code for a validation error kind.
// Decision mapping:
//   - DeploymentErrorMissingID -> 400 Bad Request (client didn't provide required field)
//   - DeploymentErrorNotFound  -> 404 Not Found (resource doesn't exist)
//   - DeploymentErrorValidation -> 400 Bad Request (validation failed)
func httpStatusForErrorKind(kind DeploymentErrorKind) int {
	switch kind {
	case DeploymentErrorMissingID:
		return 400
	case DeploymentErrorNotFound:
		return 404
	case DeploymentErrorValidation:
		return 400
	default:
		return 400 // Default to client error for unknown kinds
	}
}

// formatDeploymentValidationError converts validation errors into an HTTP status code and response body.
// Uses the error kind to determine the appropriate HTTP status.
func formatDeploymentValidationError(validation DeploymentValidationResult) (int, map[string]interface{}) {
	if len(validation.Errors) == 0 {
		return 400, map[string]interface{}{"error": "unknown validation error"}
	}

	// Use the first error's kind to determine status code
	firstError := validation.Errors[0]
	status := httpStatusForErrorKind(firstError.Kind)

	// Decision: Format response based on error kind
	switch firstError.Kind {
	case DeploymentErrorMissingID:
		return status, map[string]interface{}{"error": "profile_id required"}

	case DeploymentErrorNotFound:
		return status, map[string]interface{}{"error": firstError.Message}

	default:
		// General validation failure - include all errors and remediation
		return status, map[string]interface{}{
			"error":             "Deployment validation failed",
			"validation_errors": validation.ErrorMessages(),
			"remediation":       validation.RecommendedFix,
		}
	}
}
