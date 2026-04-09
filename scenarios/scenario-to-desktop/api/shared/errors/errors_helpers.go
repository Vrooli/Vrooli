package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// defaultManualStepsMap provides default manual steps for error codes.
// These are used when an error doesn't have explicit manual steps set.
var defaultManualStepsMap = map[ErrorCode][]string{
	// General errors
	CodeInternal: {
		"Check server logs for detailed error information",
		"Verify the service is running correctly",
		"Contact support if the issue persists",
	},
	CodeNotFound: {
		"Verify the resource identifier is correct",
		"Check that the resource exists",
		"Ensure you have permission to access the resource",
	},
	CodeBadRequest: {
		"Review the request parameters",
		"Check the API documentation for required fields",
		"Ensure all values are in the expected format",
	},
	CodeUnauthorized: {
		"Verify your credentials are correct",
		"Check that your session hasn't expired",
		"Ensure you're using the correct authentication method",
	},
	CodeForbidden: {
		"Verify you have permission for this operation",
		"Check with your administrator about access rights",
		"Ensure the resource isn't restricted",
	},
	CodeConflict: {
		"Wait a moment and retry the operation",
		"Check if a similar operation is already in progress",
		"Refresh the current state before retrying",
	},
	CodeTimeout: {
		"Check network connectivity",
		"Verify the target service is responsive",
		"Consider increasing the timeout duration",
	},
	CodeValidation: {
		"Review the input values for errors",
		"Check required fields are provided",
		"Ensure values match expected formats",
	},
	CodeUnavailable: {
		"Wait a moment and retry",
		"Check if the service is undergoing maintenance",
		"Verify network connectivity",
	},
	CodeNotImplemented: {
		"This feature is not yet available",
		"Check documentation for alternative approaches",
		"Contact support for feature requests",
	},
}

// defaultAutoFixMap provides default auto-fix suggestions for error codes.
// These are safe commands that can help resolve common issues.
var defaultAutoFixMap = map[ErrorCode]*AutoFix{
	CodeBundleManifestError: {
		Command:     "vrooli scenario generate-manifest <scenario-name>",
		Description: "Generate a bundle manifest for the scenario",
		Safe:        true,
	},
	CodeWineNotInstalled: {
		Command:     "sudo apt-get install -y wine64",
		Description: "Install Wine for Windows cross-compilation",
		Safe:        false, // Requires sudo
	},
	CodeDependencyError: {
		Command:     "vrooli setup --resources enabled",
		Description: "Install required dependencies",
		Safe:        true,
	},
}

// DefaultManualSteps returns the default manual steps for this error code.
// Returns nil if no defaults are defined.
func (e *DomainError) DefaultManualSteps() []string {
	if steps, ok := defaultManualStepsMap[e.Code]; ok {
		return steps
	}
	return nil
}

// DefaultAutoFix returns the default auto-fix for this error code.
// Returns nil if no default is defined.
func (e *DomainError) DefaultAutoFix() *AutoFix {
	if fix, ok := defaultAutoFixMap[e.Code]; ok {
		return fix
	}
	return nil
}

// EnrichRecovery adds sensible default recovery information if not already present.
// This fills in ManualSteps, AutoFix, RecoveryHint, and RetryStrategy based on
// the error code when these fields are not explicitly set.
// Returns a new DomainError with enriched recovery information.
func (e *DomainError) EnrichRecovery() *DomainError {
	enriched := &DomainError{
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       e.Details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
	}

	// Add default manual steps if not set
	if len(enriched.ManualSteps) == 0 {
		if steps := e.DefaultManualSteps(); steps != nil {
			enriched.ManualSteps = steps
		}
	}

	// Add default auto-fix if not set
	if enriched.AutoFix == nil {
		if fix := e.DefaultAutoFix(); fix != nil {
			enriched.AutoFix = fix
		}
	}

	// Add recovery hint based on category if not set
	if enriched.RecoveryHint == "" {
		switch e.Category() {
		case CategoryConfiguration:
			enriched.RecoveryHint = "Check configuration and correct any invalid values"
		case CategoryResourceMissing:
			enriched.RecoveryHint = "Ensure the required resource exists and is accessible"
		case CategoryTransient:
			enriched.RecoveryHint = "This may be a temporary issue - wait and retry"
		case CategoryExecution:
			enriched.RecoveryHint = "Retry the operation"
		case CategoryCredentials:
			enriched.RecoveryHint = "Verify credentials and permissions"
		case CategoryTerminal:
			enriched.RecoveryHint = "This error requires manual intervention"
		}
	}

	// Add default retry strategy based on category if not set
	if enriched.RetryStrategy == nil {
		switch e.Category() {
		case CategoryTransient:
			enriched.RetryStrategy = RetryConservative
		case CategoryExecution:
			enriched.RetryStrategy = RetryDefault
		}
	}

	return enriched
}

// HTTPStatus returns the appropriate HTTP status code for this error.
func (e *DomainError) HTTPStatus() int {
	if status, ok := httpStatusMap[e.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// IsDomainError checks if an error is a DomainError and returns it.
func IsDomainError(err error) (*DomainError, bool) {
	var de *DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

// GetHTTPStatus returns the HTTP status for any error.
// If it's a DomainError, uses its code mapping.
// Otherwise returns 500 Internal Server Error.
func GetHTTPStatus(err error) int {
	if de, ok := IsDomainError(err); ok {
		return de.HTTPStatus()
	}
	return http.StatusInternalServerError
}

// New creates a new DomainError with the given code and message.
func New(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new DomainError with a formatted message.
func Newf(code ErrorCode, format string, args ...interface{}) *DomainError {
	return &DomainError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with domain context.
func Wrap(code ErrorCode, cause error, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf wraps an existing error with a formatted message.
func Wrapf(code ErrorCode, cause error, format string, args ...interface{}) *DomainError {
	return &DomainError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// InDomain returns a copy of the error with the domain set.
func (e *DomainError) InDomain(domain string) *DomainError {
	return &DomainError{
		Code:          e.Code,
		Message:       e.Message,
		Domain:        domain,
		Details:       e.Details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
	}
}

// ---- Predefined error constructors for common cases ----

// ErrNotFound creates a generic not found error.
func ErrNotFound(resource string) *DomainError {
	return New(CodeNotFound, fmt.Sprintf("%s not found", resource))
}

// ErrBadRequest creates a bad request error.
func ErrBadRequest(message string) *DomainError {
	return New(CodeBadRequest, message)
}

// ErrValidation creates a validation error with details.
func ErrValidation(message string, details map[string]interface{}) *DomainError {
	return &DomainError{
		Code:    CodeValidation,
		Message: message,
		Details: details,
	}
}

// ErrInternal creates an internal server error.
func ErrInternal(message string) *DomainError {
	return New(CodeInternal, message)
}

// ErrInternalf creates an internal server error with formatting.
func ErrInternalf(format string, args ...interface{}) *DomainError {
	return Newf(CodeInternal, format, args...)
}

// ErrTimeout creates a timeout error.
func ErrTimeout(operation string) *DomainError {
	return New(CodeTimeout, fmt.Sprintf("%s timed out", operation))
}

// ErrUnavailable creates a service unavailable error.
func ErrUnavailable(service string) *DomainError {
	return New(CodeUnavailable, fmt.Sprintf("%s is unavailable", service))
}

// IsNotFound returns true if the error is a not found error.
func IsNotFound(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return false
	}
	switch de.Code {
	case CodeNotFound, CodeBundleNotFound, CodeBuildNotFound, CodeWrapperNotFound,
		CodeTemplateNotFound, CodeScenarioNotFound, CodeSessionNotFound,
		CodeJobNotFound, CodeSmokeTestNotFound, CodeArtifactNotFound,
		CodePipelineNotFound, CodeCertificateNotFound:
		return true
	}
	return false
}

// IsTimeout returns true if the error is a timeout error.
func IsTimeout(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return false
	}
	switch de.Code {
	case CodeTimeout, CodePreflightTimeout, CodeBundleServiceTimeout, CodeProcessKillTimeout:
		return true
	}
	return false
}

// IsValidation returns true if the error is a validation error.
func IsValidation(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return false
	}
	return de.Code == CodeValidation || de.Code == CodeBadRequest || de.Code == CodeConfigInvalid
}

// IsConflict returns true if the error is a conflict error.
func IsConflict(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return false
	}
	return de.Code == CodeConflict || de.Code == CodeBuildInProgress || de.Code == CodePipelineCancelled
}

// IsUnavailable returns true if the error is a service unavailable error.
func IsUnavailable(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return false
	}
	return de.Code == CodeUnavailable || de.Code == CodeWineNotInstalled
}

// ShouldRetry returns true if the error indicates the operation should be retried.
// This checks the error's recovery action to determine if retry is appropriate.
// Use this for automatic retry logic in resilient systems.
func ShouldRetry(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		// For non-domain errors, check if it looks transient
		return isLikelyTransient(err)
	}
	recovery := de.GetRecovery()
	return recovery == RecoveryRetry || recovery == RecoveryRetryWithBackoff
}

// IsUserError returns true if the error is due to user input (4xx-class errors).
// This includes validation errors, bad requests, and configuration issues.
// Use this to determine if the user should fix something vs. an internal issue.
func IsUserError(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return false
	}
	status := de.HTTPStatus()
	return status >= 400 && status < 500
}

// IsTransient returns true if the error is likely transient and may succeed on retry.
// This is similar to ShouldRetry but focuses on the nature of the error rather than
// the recovery action. Useful for logging and metrics categorization.
func IsTransient(err error) bool {
	de, ok := IsDomainError(err)
	if !ok {
		return isLikelyTransient(err)
	}
	// Transient errors are typically 5xx that might succeed on retry
	switch de.Code {
	case CodeInternal, CodeTimeout, CodeUnavailable,
		CodePreflightTimeout, CodeBundleServiceTimeout,
		CodeServiceHealthError, CodeSystemResourceError,
		CodeProcessKillTimeout, CodeKeychainError:
		return true
	}
	return false
}

// isLikelyTransient checks if a non-domain error appears to be transient.
// This provides a best-effort classification for wrapped or standard errors.
func isLikelyTransient(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return containsAny(errMsg,
		"timeout", "temporary", "unavailable", "connection refused",
		"connection reset", "too many requests", "try again",
	)
}

// MapErrorToStatus maps any error to an appropriate HTTP status code.
// For DomainErrors, uses the error code mapping.
// For standard errors, attempts to infer status from error message patterns.
// This provides a migration path from string-based error matching to typed errors.
func MapErrorToStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// First, check if it's a domain error
	if de, ok := IsDomainError(err); ok {
		return de.HTTPStatus()
	}

	// Fallback: infer from error message for legacy code migration
	// This allows gradual migration to typed errors
	errMsg := err.Error()
	switch {
	case containsAny(errMsg, "not found"):
		return http.StatusNotFound
	case containsAny(errMsg, "already in progress", "conflict"):
		return http.StatusConflict
	case containsAny(errMsg, "not available", "unavailable"):
		return http.StatusServiceUnavailable
	case containsAny(errMsg, "invalid", "required", "at least one"):
		return http.StatusBadRequest
	case containsAny(errMsg, "unauthorized", "authentication"):
		return http.StatusUnauthorized
	case containsAny(errMsg, "forbidden", "permission"):
		return http.StatusForbidden
	case containsAny(errMsg, "timeout"):
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// containsAny checks if s contains any of the substrings (case-insensitive).
func containsAny(s string, substrings ...string) bool {
	lowered := strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(lowered, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
