// Package errors provides domain-specific error handling with HTTP status mapping.
//
// This package enables clean separation between domain errors and HTTP concerns.
// Services return DomainErrors with semantic codes, and HTTP handlers automatically
// map them to appropriate HTTP status codes.
package errors

import (
	"fmt"
	"net/http"
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

// SemanticCategory groups error codes into recovery-distinct categories.
// This simplifies client-side error handling by providing a higher-level classification
// than the 51+ individual error codes.
type SemanticCategory string

const (
	// CategoryConfiguration indicates user configuration errors that require fixing input.
	// Recovery: fix_input - User needs to correct configuration and retry.
	CategoryConfiguration SemanticCategory = "configuration"

	// CategoryResourceMissing indicates a required resource was not found.
	// Recovery: fix_input or install_dependency - User needs to ensure resource exists.
	CategoryResourceMissing SemanticCategory = "resource_missing"

	// CategoryTransient indicates temporary failures that may succeed on retry.
	// Recovery: retry_with_backoff - Wait and retry automatically.
	CategoryTransient SemanticCategory = "transient"

	// CategoryExecution indicates execution errors that may succeed on immediate retry.
	// Recovery: retry - Retry immediately.
	CategoryExecution SemanticCategory = "execution"

	// CategoryCredentials indicates authentication or authorization issues.
	// Recovery: provide_credentials - User needs to provide valid credentials.
	CategoryCredentials SemanticCategory = "credentials"

	// CategoryTerminal indicates unrecoverable errors requiring human intervention.
	// Recovery: none or contact_support - No automatic recovery possible.
	CategoryTerminal SemanticCategory = "terminal"
)

// categoryMap maps error codes to semantic categories.
// This provides a simplified classification for client error handling.
var categoryMap = map[ErrorCode]SemanticCategory{
	// General errors
	CodeInternal:       CategoryExecution,
	CodeNotFound:       CategoryResourceMissing,
	CodeBadRequest:     CategoryConfiguration,
	CodeUnauthorized:   CategoryCredentials,
	CodeForbidden:      CategoryCredentials,
	CodeConflict:       CategoryTransient,
	CodeTimeout:        CategoryTransient,
	CodeValidation:     CategoryConfiguration,
	CodeUnavailable:    CategoryTransient,
	CodeNotImplemented: CategoryTerminal,

	// Bundle domain errors
	CodeBundleNotFound:       CategoryResourceMissing,
	CodeBundleInvalid:        CategoryConfiguration,
	CodeBundleManifestError:  CategoryConfiguration,
	CodeBundleCompileError:   CategoryExecution,
	CodeBundleRuntimeError:   CategoryExecution,
	CodeBundleSecretsError:   CategoryCredentials,
	CodeBundlePackageError:   CategoryExecution,
	CodeBundleServiceTimeout: CategoryTransient,

	// Build domain errors
	CodeBuildNotFound:       CategoryResourceMissing,
	CodeBuildInProgress:     CategoryTransient,
	CodeBuildFailed:         CategoryExecution,
	CodeBuildArtifactError:  CategoryExecution,
	CodePlatformUnsupported: CategoryConfiguration,

	// Generation domain errors
	CodeWrapperNotFound:       CategoryResourceMissing,
	CodeTemplateNotFound:      CategoryResourceMissing,
	CodeTemplateError:         CategoryTerminal,
	CodeGenerationFailed:      CategoryExecution,
	CodeConfigInvalid:         CategoryConfiguration,
	CodeScenarioNotFound:      CategoryResourceMissing,
	CodeScenarioPathInvalid:   CategoryConfiguration,
	CodeScenarioUnbundleable:  CategoryConfiguration,
	CodeExternalDependencyReq: CategoryConfiguration,

	// Preflight domain errors
	CodePreflightFailed:    CategoryConfiguration,
	CodePreflightTimeout:   CategoryTransient,
	CodeSessionNotFound:    CategoryResourceMissing,
	CodeSessionExpired:     CategoryTransient,
	CodeJobNotFound:        CategoryResourceMissing,
	CodeServiceStartError:  CategoryExecution,
	CodeServiceHealthError: CategoryTransient,
	CodeDependencyError:    CategoryResourceMissing,

	// Smoke test domain errors
	CodeSmokeTestNotFound:  CategoryResourceMissing,
	CodeSmokeTestFailed:    CategoryExecution,
	CodeTelemetryError:     CategoryExecution,
	CodeArtifactNotFound:   CategoryResourceMissing,
	CodeProcessSpawnError:  CategoryExecution,
	CodeProcessExitError:   CategoryExecution,
	CodeProcessKillTimeout: CategoryTransient,

	// Signing domain errors
	CodeSigningNotConfigured:   CategoryConfiguration,
	CodeSigningCertError:       CategoryConfiguration,
	CodeSigningToolError:       CategoryResourceMissing,
	CodeNotarizationError:      CategoryExecution,
	CodeEntitlementsError:      CategoryConfiguration,
	CodeCertificateExpired:     CategoryConfiguration,
	CodeCertificateNotFound:    CategoryResourceMissing,
	CodeCertificateInvalid:     CategoryConfiguration,
	CodeKeychainError:          CategoryExecution,
	CodeSigningIdentityMissing: CategoryConfiguration,

	// Pipeline domain errors
	CodePipelineNotFound:  CategoryResourceMissing,
	CodePipelineFailed:    CategoryExecution,
	CodePipelineCancelled: CategoryTerminal,
	CodePipelineTimeout:   CategoryTransient,
	CodeStageSkipped:      CategoryTerminal, // Not an error, informational
	CodeStageFailed:       CategoryExecution,

	// System domain errors
	CodeWineNotInstalled:    CategoryResourceMissing,
	CodeWineInstallFailed:   CategoryExecution,
	CodeSystemResourceError: CategoryExecution,
}

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
	CodeWrapperNotFound       ErrorCode = "WRAPPER_NOT_FOUND"
	CodeTemplateNotFound      ErrorCode = "TEMPLATE_NOT_FOUND"
	CodeTemplateError         ErrorCode = "TEMPLATE_ERROR"
	CodeGenerationFailed      ErrorCode = "GENERATION_FAILED"
	CodeConfigInvalid         ErrorCode = "CONFIG_INVALID"
	CodeScenarioNotFound      ErrorCode = "SCENARIO_NOT_FOUND"
	CodeScenarioPathInvalid   ErrorCode = "SCENARIO_PATH_INVALID"
	CodeScenarioUnbundleable  ErrorCode = "SCENARIO_UNBUNDLEABLE"
	CodeExternalDependencyReq ErrorCode = "EXTERNAL_DEPENDENCY_REQUIRED"

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

	// RetryStrategy provides configuration for automatic retries
	RetryStrategy *RetryStrategy `json:"retry_strategy,omitempty"`
	// AutoFix describes an automatic fix command if available
	AutoFix *AutoFix `json:"auto_fix,omitempty"`
	// ManualSteps provides ordered manual resolution steps
	ManualSteps []string `json:"manual_steps,omitempty"`
	// Diagnostic provides additional debugging context
	Diagnostic *DiagnosticContext `json:"diagnostic,omitempty"`
}

// RetryStrategy defines how to retry a failed operation.
type RetryStrategy struct {
	// MaxAttempts is the maximum number of retry attempts.
	MaxAttempts int `json:"max_attempts"`
	// BackoffMs is the initial backoff duration in milliseconds.
	BackoffMs int `json:"backoff_ms"`
	// BackoffMultiplier increases backoff between attempts.
	BackoffMultiplier float64 `json:"backoff_multiplier"`
}

// AutoFix describes an automatic fix that can be applied.
type AutoFix struct {
	// Command is the shell command to run.
	Command string `json:"command"`
	// Description explains what the command does.
	Description string `json:"description"`
	// Safe indicates if the command is safe to run without confirmation.
	Safe bool `json:"safe"`
}

// DiagnosticContext provides additional information for debugging.
type DiagnosticContext struct {
	// Process contains process-related diagnostics.
	Process *ProcessDiagnostic `json:"process,omitempty"`
	// System contains system-related information.
	System map[string]string `json:"system,omitempty"`
}

// ProcessDiagnostic contains process execution details.
type ProcessDiagnostic struct {
	// PID is the process ID if available.
	PID int `json:"pid,omitempty"`
	// ExitCode is the exit code if the process terminated.
	ExitCode int `json:"exit_code,omitempty"`
	// RuntimeMs is how long the process ran in milliseconds.
	RuntimeMs int64 `json:"runtime_ms,omitempty"`
	// LastOutput is the last portion of output before failure.
	LastOutput string `json:"last_output,omitempty"`
}

// Predefined retry strategies for common scenarios.
var (
	// RetryDefault is a balanced retry strategy for most operations.
	RetryDefault = &RetryStrategy{MaxAttempts: 3, BackoffMs: 1000, BackoffMultiplier: 2.0}
	// RetryAggressive is for operations that should be retried quickly.
	RetryAggressive = &RetryStrategy{MaxAttempts: 5, BackoffMs: 500, BackoffMultiplier: 1.5}
	// RetryConservative is for operations that need longer delays.
	RetryConservative = &RetryStrategy{MaxAttempts: 2, BackoffMs: 5000, BackoffMultiplier: 2.0}
)

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
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       e.Details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
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
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
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
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       merged,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
	}
}

// WithMessage returns a copy of the error with a custom message.
func (e *DomainError) WithMessage(msg string) *DomainError {
	return &DomainError{
		Code:          e.Code,
		Message:       msg,
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
}

// WithMessagef returns a copy of the error with a formatted custom message.
func (e *DomainError) WithMessagef(format string, args ...interface{}) *DomainError {
	return e.WithMessage(fmt.Sprintf(format, args...))
}

// WithRecovery returns a copy of the error with recovery information.
func (e *DomainError) WithRecovery(action RecoveryAction, hint string) *DomainError {
	return &DomainError{
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       e.Details,
		Recovery:      action,
		RecoveryHint:  hint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
	}
}

// WithRetryStrategy returns a copy of the error with a retry strategy.
func (e *DomainError) WithRetryStrategy(s *RetryStrategy) *DomainError {
	return &DomainError{
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       e.Details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: s,
		AutoFix:       e.AutoFix,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
	}
}

// WithAutoFix returns a copy of the error with an auto-fix command.
func (e *DomainError) WithAutoFix(f *AutoFix) *DomainError {
	return &DomainError{
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       e.Details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       f,
		ManualSteps:   e.ManualSteps,
		Diagnostic:    e.Diagnostic,
	}
}

// WithManualSteps returns a copy of the error with manual resolution steps.
func (e *DomainError) WithManualSteps(steps []string) *DomainError {
	return &DomainError{
		Code:          e.Code,
		Message:       e.Message,
		Domain:        e.Domain,
		Details:       e.Details,
		Recovery:      e.Recovery,
		RecoveryHint:  e.RecoveryHint,
		Cause:         e.Cause,
		RetryStrategy: e.RetryStrategy,
		AutoFix:       e.AutoFix,
		ManualSteps:   steps,
		Diagnostic:    e.Diagnostic,
	}
}

// WithDiagnostic returns a copy of the error with diagnostic context.
func (e *DomainError) WithDiagnostic(d *DiagnosticContext) *DomainError {
	return &DomainError{
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
		Diagnostic:    d,
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
	CodeWrapperNotFound:       http.StatusNotFound,
	CodeTemplateNotFound:      http.StatusNotFound,
	CodeTemplateError:         http.StatusInternalServerError,
	CodeGenerationFailed:      http.StatusInternalServerError,
	CodeConfigInvalid:         http.StatusBadRequest,
	CodeScenarioNotFound:      http.StatusNotFound,
	CodeScenarioPathInvalid:   http.StatusBadRequest,
	CodeScenarioUnbundleable:  http.StatusBadRequest,
	CodeExternalDependencyReq: http.StatusBadRequest,

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

// Category returns the semantic category for this error.
// Categories group the 51+ error codes into 6 recovery-distinct categories
// for simpler client-side error handling.
func (e *DomainError) Category() SemanticCategory {
	if cat, ok := categoryMap[e.Code]; ok {
		return cat
	}
	// Default to execution for unknown codes (retry is safe)
	return CategoryExecution
}
