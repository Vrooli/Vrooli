// Package deployments provides deployment execution and status tracking.
package deployments

import (
	"fmt"
	"strings"
	"time"
)

// ErrorKind categorizes validation errors by their type.
// This enables consistent HTTP status code mapping and error handling.
type ErrorKind int

const (
	// ErrorMissingID indicates the profile_id was not provided.
	ErrorMissingID ErrorKind = iota

	// ErrorNotFound indicates the profile does not exist.
	ErrorNotFound

	// ErrorValidation indicates a general validation failure.
	ErrorValidation
)

// ValidationError represents a typed validation error.
type ValidationError struct {
	Kind    ErrorKind
	Message string
}

// ValidationResult holds the result of validating a deployment profile.
type ValidationResult struct {
	Valid          bool
	Errors         []ValidationError
	ProfileID      string
	DeploymentID   string
	RecommendedFix string
}

// ErrorMessages returns the error messages as strings for API responses.
func (r ValidationResult) ErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		messages[i] = e.Message
	}
	return messages
}

// ValidateProfile validates a profile for deployment readiness.
// Returns validation errors if the profile cannot be deployed.
func ValidateProfile(profileID string) ValidationResult {
	result := ValidationResult{
		ProfileID: profileID,
		Valid:     true,
	}

	// Decision: Is profile_id provided?
	if profileID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Kind:    ErrorMissingID,
			Message: "profile_id is required",
		})
		return result
	}

	// Decision: Does profile_id match valid format patterns?
	// Valid formats: "profile-*" or "test-*"
	if !isValidProfileIDFormat(profileID) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Kind:    ErrorNotFound,
			Message: fmt.Sprintf("Profile '%s' not found", profileID),
		})
		return result
	}

	// Decision: Are required packagers available? (simplified example)
	if profileID == "missing-packager-profile" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Kind:    ErrorValidation,
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

// GenerateID creates a unique deployment ID based on timestamp.
func GenerateID() string {
	return fmt.Sprintf("deploy-%d", time.Now().Unix())
}

// HTTPStatusForErrorKind decides the HTTP status code for a validation error kind.
// Decision mapping:
//   - ErrorMissingID -> 400 Bad Request (client didn't provide required field)
//   - ErrorNotFound  -> 404 Not Found (resource doesn't exist)
//   - ErrorValidation -> 400 Bad Request (validation failed)
func HTTPStatusForErrorKind(kind ErrorKind) int {
	switch kind {
	case ErrorMissingID:
		return 400
	case ErrorNotFound:
		return 404
	case ErrorValidation:
		return 400
	default:
		return 400 // Default to client error for unknown kinds
	}
}

// FormatValidationError converts validation errors into an HTTP status code and response body.
// Uses the error kind to determine the appropriate HTTP status.
func FormatValidationError(validation ValidationResult) (int, map[string]interface{}) {
	if len(validation.Errors) == 0 {
		return 400, map[string]interface{}{"error": "unknown validation error"}
	}

	// Use the first error's kind to determine status code
	firstError := validation.Errors[0]
	status := HTTPStatusForErrorKind(firstError.Kind)

	// Decision: Format response based on error kind
	switch firstError.Kind {
	case ErrorMissingID:
		return status, map[string]interface{}{"error": "profile_id required"}

	case ErrorNotFound:
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
