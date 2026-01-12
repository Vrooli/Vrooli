// Package errors provides domain-specific error handling with HTTP status mapping.
//
// This package enables clean separation between domain errors and HTTP concerns.
// Services return DomainErrors with semantic codes, and HTTP handlers automatically
// map them to appropriate HTTP status codes.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents a semantic error code for domain errors.
type ErrorCode string

// Domain error codes - grouped by domain for discoverability.
const (
	// General errors
	CodeInternal       ErrorCode = "INTERNAL_ERROR"
	CodeNotFound       ErrorCode = "NOT_FOUND"
	CodeBadRequest     ErrorCode = "BAD_REQUEST"
	CodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	CodeForbidden      ErrorCode = "FORBIDDEN"
	CodeConflict       ErrorCode = "CONFLICT"
	CodeTimeout        ErrorCode = "TIMEOUT"
	CodeValidation     ErrorCode = "VALIDATION_ERROR"
	CodeUnavailable    ErrorCode = "SERVICE_UNAVAILABLE"
	CodeNotImplemented ErrorCode = "NOT_IMPLEMENTED"

	// Bundle domain errors
	CodeBundleNotFound       ErrorCode = "BUNDLE_NOT_FOUND"
	CodeBundleInvalid        ErrorCode = "BUNDLE_INVALID"
	CodeBundleManifestError  ErrorCode = "BUNDLE_MANIFEST_ERROR"
	CodeBundleCompileError   ErrorCode = "BUNDLE_COMPILE_ERROR"
	CodeBundleRuntimeError   ErrorCode = "BUNDLE_RUNTIME_ERROR"
	CodeBundleSecretsError   ErrorCode = "BUNDLE_SECRETS_ERROR"
	CodeBundlePackageError   ErrorCode = "BUNDLE_PACKAGE_ERROR"
	CodeBundleServiceTimeout ErrorCode = "BUNDLE_SERVICE_TIMEOUT"

	// Build domain errors
	CodeBuildNotFound       ErrorCode = "BUILD_NOT_FOUND"
	CodeBuildInProgress     ErrorCode = "BUILD_IN_PROGRESS"
	CodeBuildFailed         ErrorCode = "BUILD_FAILED"
	CodeBuildArtifactError  ErrorCode = "BUILD_ARTIFACT_ERROR"
	CodePlatformUnsupported ErrorCode = "PLATFORM_UNSUPPORTED"

	// Generation domain errors
	CodeWrapperNotFound     ErrorCode = "WRAPPER_NOT_FOUND"
	CodeTemplateNotFound    ErrorCode = "TEMPLATE_NOT_FOUND"
	CodeTemplateError       ErrorCode = "TEMPLATE_ERROR"
	CodeGenerationFailed    ErrorCode = "GENERATION_FAILED"
	CodeConfigInvalid       ErrorCode = "CONFIG_INVALID"
	CodeScenarioNotFound    ErrorCode = "SCENARIO_NOT_FOUND"
	CodeScenarioPathInvalid ErrorCode = "SCENARIO_PATH_INVALID"

	// Preflight domain errors
	CodePreflightFailed    ErrorCode = "PREFLIGHT_FAILED"
	CodePreflightTimeout   ErrorCode = "PREFLIGHT_TIMEOUT"
	CodeSessionNotFound    ErrorCode = "SESSION_NOT_FOUND"
	CodeSessionExpired     ErrorCode = "SESSION_EXPIRED"
	CodeJobNotFound        ErrorCode = "JOB_NOT_FOUND"
	CodeServiceStartError  ErrorCode = "SERVICE_START_ERROR"
	CodeServiceHealthError ErrorCode = "SERVICE_HEALTH_ERROR"
	CodeDependencyError    ErrorCode = "DEPENDENCY_ERROR"

	// Smoke test domain errors
	CodeSmokeTestNotFound  ErrorCode = "SMOKE_TEST_NOT_FOUND"
	CodeSmokeTestFailed    ErrorCode = "SMOKE_TEST_FAILED"
	CodeTelemetryError     ErrorCode = "TELEMETRY_ERROR"
	CodeArtifactNotFound   ErrorCode = "ARTIFACT_NOT_FOUND"
	CodeProcessSpawnError  ErrorCode = "PROCESS_SPAWN_ERROR"
	CodeProcessExitError   ErrorCode = "PROCESS_EXIT_ERROR"
	CodeProcessKillTimeout ErrorCode = "PROCESS_KILL_TIMEOUT"

	// Signing domain errors
	CodeSigningNotConfigured   ErrorCode = "SIGNING_NOT_CONFIGURED"
	CodeSigningCertError       ErrorCode = "SIGNING_CERT_ERROR"
	CodeSigningToolError       ErrorCode = "SIGNING_TOOL_ERROR"
	CodeNotarizationError      ErrorCode = "NOTARIZATION_ERROR"
	CodeEntitlementsError      ErrorCode = "ENTITLEMENTS_ERROR"
	CodeCertificateExpired     ErrorCode = "CERTIFICATE_EXPIRED"
	CodeCertificateNotFound    ErrorCode = "CERTIFICATE_NOT_FOUND"
	CodeCertificateInvalid     ErrorCode = "CERTIFICATE_INVALID"
	CodeKeychainError          ErrorCode = "KEYCHAIN_ERROR"
	CodeSigningIdentityMissing ErrorCode = "SIGNING_IDENTITY_MISSING"

	// Pipeline domain errors
	CodePipelineNotFound  ErrorCode = "PIPELINE_NOT_FOUND"
	CodePipelineFailed    ErrorCode = "PIPELINE_FAILED"
	CodePipelineCancelled ErrorCode = "PIPELINE_CANCELLED"
	CodeStageSkipped      ErrorCode = "STAGE_SKIPPED"
	CodeStageFailed       ErrorCode = "STAGE_FAILED"

	// System domain errors
	CodeWineNotInstalled    ErrorCode = "WINE_NOT_INSTALLED"
	CodeWineInstallFailed   ErrorCode = "WINE_INSTALL_FAILED"
	CodeSystemResourceError ErrorCode = "SYSTEM_RESOURCE_ERROR"
)

// DomainError represents a structured error with semantic meaning.
// It carries context about what went wrong and where, enabling proper
// error handling at the HTTP boundary.
type DomainError struct {
	// Code is the semantic error code
	Code ErrorCode `json:"code"`
	// Message is a human-readable error message
	Message string `json:"message"`
	// Domain identifies which part of the system generated the error
	Domain string `json:"domain,omitempty"`
	// Details provides additional context (e.g., validation errors, IDs)
	Details map[string]interface{} `json:"details,omitempty"`
	// Cause is the underlying error (not serialized to JSON)
	Cause error `json:"-"`
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// WithCause returns a copy of the error with the given underlying cause.
func (e *DomainError) WithCause(cause error) *DomainError {
	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Domain:  e.Domain,
		Details: e.Details,
		Cause:   cause,
	}
}

// WithDetail returns a copy of the error with an additional detail field.
func (e *DomainError) WithDetail(key string, value interface{}) *DomainError {
	details := make(map[string]interface{})
	for k, v := range e.Details {
		details[k] = v
	}
	details[key] = value
	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Domain:  e.Domain,
		Details: details,
		Cause:   e.Cause,
	}
}

// WithDetails returns a copy of the error with additional detail fields.
func (e *DomainError) WithDetails(details map[string]interface{}) *DomainError {
	merged := make(map[string]interface{})
	for k, v := range e.Details {
		merged[k] = v
	}
	for k, v := range details {
		merged[k] = v
	}
	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Domain:  e.Domain,
		Details: merged,
		Cause:   e.Cause,
	}
}

// WithMessage returns a copy of the error with a custom message.
func (e *DomainError) WithMessage(msg string) *DomainError {
	return &DomainError{
		Code:    e.Code,
		Message: msg,
		Domain:  e.Domain,
		Details: e.Details,
		Cause:   e.Cause,
	}
}

// WithMessagef returns a copy of the error with a formatted custom message.
func (e *DomainError) WithMessagef(format string, args ...interface{}) *DomainError {
	return e.WithMessage(fmt.Sprintf(format, args...))
}

// httpStatusMap maps error codes to HTTP status codes.
var httpStatusMap = map[ErrorCode]int{
	// General errors
	CodeInternal:       http.StatusInternalServerError,
	CodeNotFound:       http.StatusNotFound,
	CodeBadRequest:     http.StatusBadRequest,
	CodeUnauthorized:   http.StatusUnauthorized,
	CodeForbidden:      http.StatusForbidden,
	CodeConflict:       http.StatusConflict,
	CodeTimeout:        http.StatusGatewayTimeout,
	CodeValidation:     http.StatusUnprocessableEntity,
	CodeUnavailable:    http.StatusServiceUnavailable,
	CodeNotImplemented: http.StatusNotImplemented,

	// Bundle domain
	CodeBundleNotFound:       http.StatusNotFound,
	CodeBundleInvalid:        http.StatusBadRequest,
	CodeBundleManifestError:  http.StatusBadRequest,
	CodeBundleCompileError:   http.StatusInternalServerError,
	CodeBundleRuntimeError:   http.StatusInternalServerError,
	CodeBundleSecretsError:   http.StatusBadRequest,
	CodeBundlePackageError:   http.StatusInternalServerError,
	CodeBundleServiceTimeout: http.StatusGatewayTimeout,

	// Build domain
	CodeBuildNotFound:       http.StatusNotFound,
	CodeBuildInProgress:     http.StatusConflict,
	CodeBuildFailed:         http.StatusInternalServerError,
	CodeBuildArtifactError:  http.StatusInternalServerError,
	CodePlatformUnsupported: http.StatusBadRequest,

	// Generation domain
	CodeWrapperNotFound:     http.StatusNotFound,
	CodeTemplateNotFound:    http.StatusNotFound,
	CodeTemplateError:       http.StatusInternalServerError,
	CodeGenerationFailed:    http.StatusInternalServerError,
	CodeConfigInvalid:       http.StatusBadRequest,
	CodeScenarioNotFound:    http.StatusNotFound,
	CodeScenarioPathInvalid: http.StatusBadRequest,

	// Preflight domain
	CodePreflightFailed:    http.StatusInternalServerError,
	CodePreflightTimeout:   http.StatusGatewayTimeout,
	CodeSessionNotFound:    http.StatusNotFound,
	CodeSessionExpired:     http.StatusGone,
	CodeJobNotFound:        http.StatusNotFound,
	CodeServiceStartError:  http.StatusInternalServerError,
	CodeServiceHealthError: http.StatusInternalServerError,
	CodeDependencyError:    http.StatusFailedDependency,

	// Smoke test domain
	CodeSmokeTestNotFound:  http.StatusNotFound,
	CodeSmokeTestFailed:    http.StatusInternalServerError,
	CodeTelemetryError:     http.StatusInternalServerError,
	CodeArtifactNotFound:   http.StatusNotFound,
	CodeProcessSpawnError:  http.StatusInternalServerError,
	CodeProcessExitError:   http.StatusInternalServerError,
	CodeProcessKillTimeout: http.StatusGatewayTimeout,

	// Signing domain
	CodeSigningNotConfigured:   http.StatusBadRequest,
	CodeSigningCertError:       http.StatusBadRequest,
	CodeSigningToolError:       http.StatusInternalServerError,
	CodeNotarizationError:      http.StatusInternalServerError,
	CodeEntitlementsError:      http.StatusInternalServerError,
	CodeCertificateExpired:     http.StatusBadRequest,
	CodeCertificateNotFound:    http.StatusNotFound,
	CodeCertificateInvalid:     http.StatusBadRequest,
	CodeKeychainError:          http.StatusInternalServerError,
	CodeSigningIdentityMissing: http.StatusBadRequest,

	// Pipeline domain
	CodePipelineNotFound:  http.StatusNotFound,
	CodePipelineFailed:    http.StatusInternalServerError,
	CodePipelineCancelled: http.StatusConflict,
	CodeStageSkipped:      http.StatusOK, // Not an error, just informational
	CodeStageFailed:       http.StatusInternalServerError,

	// System domain
	CodeWineNotInstalled:    http.StatusServiceUnavailable,
	CodeWineInstallFailed:   http.StatusInternalServerError,
	CodeSystemResourceError: http.StatusInternalServerError,
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
		Code:    e.Code,
		Message: e.Message,
		Domain:  domain,
		Details: e.Details,
		Cause:   e.Cause,
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

// ---- Domain-specific convenience constructors ----

// Bundle domain constructors

// ErrBundleNotFound creates a bundle not found error.
func ErrBundleNotFound(bundlePath string) *DomainError {
	return New(CodeBundleNotFound, "bundle not found").
		WithDetail("bundle_path", bundlePath).
		InDomain("bundle")
}

// ErrBundleManifest creates a bundle manifest error.
func ErrBundleManifest(cause error) *DomainError {
	return Wrap(CodeBundleManifestError, cause, "failed to parse bundle manifest").
		InDomain("bundle")
}

// Build domain constructors

// ErrBuildNotFound creates a build not found error.
func ErrBuildNotFound(buildID string) *DomainError {
	return New(CodeBuildNotFound, "build not found").
		WithDetail("build_id", buildID).
		InDomain("build")
}

// ErrBuildFailed creates a build failed error.
func ErrBuildFailed(cause error, platform string) *DomainError {
	return Wrap(CodeBuildFailed, cause, "build failed").
		WithDetail("platform", platform).
		InDomain("build")
}

// Generation domain constructors

// ErrWrapperNotFound creates a wrapper not found error.
func ErrWrapperNotFound(scenario string) *DomainError {
	return New(CodeWrapperNotFound, "wrapper not found").
		WithDetail("scenario", scenario).
		InDomain("generation")
}

// ErrScenarioNotFound creates a scenario not found error.
func ErrScenarioNotFound(scenario string) *DomainError {
	return New(CodeScenarioNotFound, "scenario not found").
		WithDetail("scenario", scenario).
		InDomain("generation")
}

// ErrTemplateNotFound creates a template not found error.
func ErrTemplateNotFound(templateType string) *DomainError {
	return New(CodeTemplateNotFound, "template not found").
		WithDetail("template_type", templateType).
		InDomain("generation")
}

// Preflight domain constructors

// ErrSessionNotFound creates a session not found error.
func ErrSessionNotFound(sessionID string) *DomainError {
	return New(CodeSessionNotFound, "session not found").
		WithDetail("session_id", sessionID).
		InDomain("preflight")
}

// ErrSessionExpired creates a session expired error.
func ErrSessionExpired(sessionID string) *DomainError {
	return New(CodeSessionExpired, "session expired").
		WithDetail("session_id", sessionID).
		InDomain("preflight")
}

// ErrJobNotFound creates a job not found error.
func ErrJobNotFound(jobID string) *DomainError {
	return New(CodeJobNotFound, "job not found").
		WithDetail("job_id", jobID).
		InDomain("preflight")
}

// ErrPreflightFailed creates a preflight failed error.
func ErrPreflightFailed(cause error) *DomainError {
	return Wrap(CodePreflightFailed, cause, "preflight validation failed").
		InDomain("preflight")
}

// Smoke test domain constructors

// ErrSmokeTestNotFound creates a smoke test not found error.
func ErrSmokeTestNotFound(testID string) *DomainError {
	return New(CodeSmokeTestNotFound, "smoke test not found").
		WithDetail("smoke_test_id", testID).
		InDomain("smoketest")
}

// ErrArtifactNotFound creates an artifact not found error.
func ErrArtifactNotFound(artifactPath string) *DomainError {
	return New(CodeArtifactNotFound, "artifact not found").
		WithDetail("artifact_path", artifactPath).
		InDomain("smoketest")
}

// Pipeline domain constructors

// ErrPipelineNotFound creates a pipeline not found error.
func ErrPipelineNotFound(pipelineID string) *DomainError {
	return New(CodePipelineNotFound, "pipeline not found").
		WithDetail("pipeline_id", pipelineID).
		InDomain("pipeline")
}

// ErrPipelineCancelled creates a pipeline cancelled error.
func ErrPipelineCancelled(pipelineID string) *DomainError {
	return New(CodePipelineCancelled, "pipeline was cancelled").
		WithDetail("pipeline_id", pipelineID).
		InDomain("pipeline")
}

// Signing domain constructors

// ErrCertificateNotFound creates a certificate not found error.
func ErrCertificateNotFound(certID string) *DomainError {
	return New(CodeCertificateNotFound, "certificate not found").
		WithDetail("certificate_id", certID).
		InDomain("signing")
}

// ErrCertificateExpired creates a certificate expired error.
func ErrCertificateExpired(certID string, expiresAt string) *DomainError {
	return New(CodeCertificateExpired, "certificate has expired").
		WithDetail("certificate_id", certID).
		WithDetail("expires_at", expiresAt).
		InDomain("signing")
}

// System domain constructors

// ErrWineNotInstalled creates a Wine not installed error.
func ErrWineNotInstalled() *DomainError {
	return New(CodeWineNotInstalled, "Wine is not installed").
		InDomain("system")
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
