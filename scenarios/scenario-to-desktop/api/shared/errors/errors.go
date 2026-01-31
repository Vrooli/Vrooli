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
	"strings"
)

// ErrorCode represents a semantic error code for domain errors.
type ErrorCode string

// RecoveryAction describes what a client should do to recover from an error.
// This helps both humans and agents determine the appropriate next step.
type RecoveryAction string

const (
	// RecoveryRetry indicates the operation can be retried (transient failure).
	RecoveryRetry RecoveryAction = "retry"
	// RecoveryRetryWithBackoff indicates retry after a delay (rate limiting, temporary overload).
	RecoveryRetryWithBackoff RecoveryAction = "retry_with_backoff"
	// RecoveryFixInput indicates the client should correct input and resubmit.
	RecoveryFixInput RecoveryAction = "fix_input"
	// RecoveryProvideCredentials indicates missing authentication/secrets.
	RecoveryProvideCredentials RecoveryAction = "provide_credentials"
	// RecoveryWaitForResource indicates a resource is being prepared (try again soon).
	RecoveryWaitForResource RecoveryAction = "wait_for_resource"
	// RecoveryInstallDependency indicates a system dependency must be installed.
	RecoveryInstallDependency RecoveryAction = "install_dependency"
	// RecoveryContactSupport indicates an unrecoverable error requiring human intervention.
	RecoveryContactSupport RecoveryAction = "contact_support"
	// RecoveryNone indicates no recovery is possible for this error.
	RecoveryNone RecoveryAction = "none"
)

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
	CodePipelineTimeout   ErrorCode = "PIPELINE_TIMEOUT"
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
	// Recovery indicates what action the client should take to recover
	Recovery RecoveryAction `json:"recovery,omitempty"`
	// RecoveryHint provides human-readable guidance on how to recover
	RecoveryHint string `json:"recovery_hint,omitempty"`
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
		Code:         e.Code,
		Message:      e.Message,
		Domain:       e.Domain,
		Details:      e.Details,
		Recovery:     e.Recovery,
		RecoveryHint: e.RecoveryHint,
		Cause:        cause,
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
		Code:         e.Code,
		Message:      e.Message,
		Domain:       e.Domain,
		Details:      details,
		Recovery:     e.Recovery,
		RecoveryHint: e.RecoveryHint,
		Cause:        e.Cause,
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
		Code:         e.Code,
		Message:      e.Message,
		Domain:       e.Domain,
		Details:      merged,
		Recovery:     e.Recovery,
		RecoveryHint: e.RecoveryHint,
		Cause:        e.Cause,
	}
}

// WithMessage returns a copy of the error with a custom message.
func (e *DomainError) WithMessage(msg string) *DomainError {
	return &DomainError{
		Code:         e.Code,
		Message:      msg,
		Domain:       e.Domain,
		Details:      e.Details,
		Recovery:     e.Recovery,
		RecoveryHint: e.RecoveryHint,
		Cause:        e.Cause,
	}
}

// WithMessagef returns a copy of the error with a formatted custom message.
func (e *DomainError) WithMessagef(format string, args ...interface{}) *DomainError {
	return e.WithMessage(fmt.Sprintf(format, args...))
}

// WithRecovery returns a copy of the error with recovery information.
func (e *DomainError) WithRecovery(action RecoveryAction, hint string) *DomainError {
	return &DomainError{
		Code:         e.Code,
		Message:      e.Message,
		Domain:       e.Domain,
		Details:      e.Details,
		Recovery:     action,
		RecoveryHint: hint,
		Cause:        e.Cause,
	}
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
	CodePipelineTimeout:   http.StatusGatewayTimeout,
	CodeStageSkipped:      http.StatusOK, // Not an error, just informational
	CodeStageFailed:       http.StatusInternalServerError,

	// System domain
	CodeWineNotInstalled:    http.StatusServiceUnavailable,
	CodeWineInstallFailed:   http.StatusInternalServerError,
	CodeSystemResourceError: http.StatusInternalServerError,
}

// defaultRecoveryMap maps error codes to default recovery actions.
// Errors can override these via WithRecovery.
var defaultRecoveryMap = map[ErrorCode]RecoveryAction{
	// General errors
	CodeInternal:       RecoveryRetry,
	CodeNotFound:       RecoveryFixInput,
	CodeBadRequest:     RecoveryFixInput,
	CodeUnauthorized:   RecoveryProvideCredentials,
	CodeForbidden:      RecoveryContactSupport,
	CodeConflict:       RecoveryWaitForResource,
	CodeTimeout:        RecoveryRetryWithBackoff,
	CodeValidation:     RecoveryFixInput,
	CodeUnavailable:    RecoveryRetryWithBackoff,
	CodeNotImplemented: RecoveryContactSupport,

	// Bundle domain
	CodeBundleNotFound:       RecoveryFixInput,
	CodeBundleInvalid:        RecoveryFixInput,
	CodeBundleManifestError:  RecoveryFixInput,
	CodeBundleCompileError:   RecoveryRetry,
	CodeBundleRuntimeError:   RecoveryRetry,
	CodeBundleSecretsError:   RecoveryProvideCredentials,
	CodeBundlePackageError:   RecoveryRetry,
	CodeBundleServiceTimeout: RecoveryRetryWithBackoff,

	// Build domain
	CodeBuildNotFound:       RecoveryFixInput,
	CodeBuildInProgress:     RecoveryWaitForResource,
	CodeBuildFailed:         RecoveryRetry,
	CodeBuildArtifactError:  RecoveryRetry,
	CodePlatformUnsupported: RecoveryFixInput,

	// Generation domain
	CodeWrapperNotFound:     RecoveryFixInput,
	CodeTemplateNotFound:    RecoveryFixInput,
	CodeTemplateError:       RecoveryContactSupport,
	CodeGenerationFailed:    RecoveryRetry,
	CodeConfigInvalid:       RecoveryFixInput,
	CodeScenarioNotFound:    RecoveryFixInput,
	CodeScenarioPathInvalid: RecoveryFixInput,

	// Preflight domain
	CodePreflightFailed:    RecoveryFixInput,
	CodePreflightTimeout:   RecoveryRetryWithBackoff,
	CodeSessionNotFound:    RecoveryRetry,
	CodeSessionExpired:     RecoveryRetry,
	CodeJobNotFound:        RecoveryFixInput,
	CodeServiceStartError:  RecoveryRetry,
	CodeServiceHealthError: RecoveryRetryWithBackoff,
	CodeDependencyError:    RecoveryInstallDependency,

	// Smoke test domain
	CodeSmokeTestNotFound:  RecoveryFixInput,
	CodeSmokeTestFailed:    RecoveryRetry,
	CodeTelemetryError:     RecoveryRetry,
	CodeArtifactNotFound:   RecoveryFixInput,
	CodeProcessSpawnError:  RecoveryRetry,
	CodeProcessExitError:   RecoveryRetry,
	CodeProcessKillTimeout: RecoveryRetryWithBackoff,

	// Signing domain
	CodeSigningNotConfigured:   RecoveryFixInput,
	CodeSigningCertError:       RecoveryFixInput,
	CodeSigningToolError:       RecoveryInstallDependency,
	CodeNotarizationError:      RecoveryRetry,
	CodeEntitlementsError:      RecoveryFixInput,
	CodeCertificateExpired:     RecoveryFixInput,
	CodeCertificateNotFound:    RecoveryFixInput,
	CodeCertificateInvalid:     RecoveryFixInput,
	CodeKeychainError:          RecoveryRetry,
	CodeSigningIdentityMissing: RecoveryFixInput,

	// Pipeline domain
	CodePipelineNotFound:  RecoveryFixInput,
	CodePipelineFailed:    RecoveryRetry,
	CodePipelineCancelled: RecoveryNone,
	CodePipelineTimeout:   RecoveryRetryWithBackoff,
	CodeStageSkipped:      RecoveryNone,
	CodeStageFailed:       RecoveryRetry,

	// System domain
	CodeWineNotInstalled:    RecoveryInstallDependency,
	CodeWineInstallFailed:   RecoveryRetry,
	CodeSystemResourceError: RecoveryRetry,
}

// DefaultRecovery returns the default recovery action for this error code.
// Returns RecoveryNone if no default is mapped.
func (e *DomainError) DefaultRecovery() RecoveryAction {
	if action, ok := defaultRecoveryMap[e.Code]; ok {
		return action
	}
	return RecoveryNone
}

// GetRecovery returns the recovery action, using the default if not explicitly set.
func (e *DomainError) GetRecovery() RecoveryAction {
	if e.Recovery != "" {
		return e.Recovery
	}
	return e.DefaultRecovery()
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
		Code:         e.Code,
		Message:      e.Message,
		Domain:       domain,
		Details:      e.Details,
		Recovery:     e.Recovery,
		RecoveryHint: e.RecoveryHint,
		Cause:        e.Cause,
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
		WithRecovery(RecoveryFixInput, "Verify the pipeline ID is correct or start a new pipeline").
		InDomain("pipeline")
}

// ErrPipelineCancelled creates a pipeline cancelled error.
func ErrPipelineCancelled(pipelineID string) *DomainError {
	return New(CodePipelineCancelled, "pipeline was cancelled").
		WithDetail("pipeline_id", pipelineID).
		WithRecovery(RecoveryNone, "The pipeline was cancelled as requested").
		InDomain("pipeline")
}

// ErrPipelineNotResumable creates an error when a pipeline cannot be resumed.
func ErrPipelineNotResumable(pipelineID string, reason string) *DomainError {
	return New(CodeBadRequest, "pipeline cannot be resumed: "+reason).
		WithDetail("pipeline_id", pipelineID).
		WithRecovery(RecoveryFixInput, "Start a new pipeline instead").
		InDomain("pipeline")
}

// ErrPipelineOrchestratorNotConfigured creates an error when the orchestrator is not configured.
func ErrPipelineOrchestratorNotConfigured() *DomainError {
	return New(CodeInternal, "pipeline orchestrator not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		InDomain("pipeline")
}

// ErrPipelineInvalidStage creates an error for invalid stage names.
func ErrPipelineInvalidStage(stageName string) *DomainError {
	return New(CodeValidation, "invalid stage name: "+stageName).
		WithDetail("stage_name", stageName).
		WithDetail("valid_stages", []string{"bundle", "preflight", "generate", "build", "smoketest", "distribution"}).
		WithRecovery(RecoveryFixInput, "Use one of the valid stage names").
		InDomain("pipeline")
}

// ErrPipelineScenarioRequired creates an error when scenario_name is missing.
func ErrPipelineScenarioRequired() *DomainError {
	return New(CodeValidation, "scenario_name is required").
		WithRecovery(RecoveryFixInput, "Provide a scenario_name in the request body").
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
