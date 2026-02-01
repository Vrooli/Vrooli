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
	CodeWrapperNotFound:     CategoryResourceMissing,
	CodeTemplateNotFound:    CategoryResourceMissing,
	CodeTemplateError:       CategoryTerminal,
	CodeGenerationFailed:    CategoryExecution,
	CodeConfigInvalid:       CategoryConfiguration,
	CodeScenarioNotFound:    CategoryResourceMissing,
	CodeScenarioPathInvalid: CategoryConfiguration,

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

// ---- Stage-specific error constructors ----
// These provide rich error information for pipeline stage failures.

// Bundle stage errors

// ErrBundleManifestNotFound creates an error for missing bundle manifest.
func ErrBundleManifestNotFound(path string) *DomainError {
	return New(CodeBundleManifestError, "bundle manifest not found").
		WithDetail("manifest_path", path).
		WithRecovery(RecoveryFixInput, "Ensure the bundle manifest exists at the expected path").
		WithManualSteps([]string{
			fmt.Sprintf("Check if manifest exists: ls -la %s", path),
			"Verify scenario has platforms/<framework>/bundle/bundle.json",
			"Run deployment-manager to generate the manifest if missing",
		}).
		InDomain("bundle")
}

// ErrBundleManifestGeneration creates an error for manifest generation failure.
func ErrBundleManifestGeneration(cause error) *DomainError {
	return Wrap(CodeBundleManifestError, cause, "failed to generate bundle manifest").
		WithRecovery(RecoveryRetry, "Retry the operation or check manifest generation logs").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check deployment-manager logs for details",
			"Verify scenario service.json is valid",
			"Ensure all required dependencies are available",
		}).
		InDomain("bundle")
}

// ErrBundlePackagingFailed creates an error for bundle packaging failure.
func ErrBundlePackagingFailed(cause error, path string) *DomainError {
	return Wrap(CodeBundlePackageError, cause, "bundle packaging failed").
		WithDetail("bundle_path", path).
		WithRecovery(RecoveryRetry, "Check bundle configuration and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Review the bundle manifest for errors",
			"Check disk space and permissions",
			"Verify all referenced files exist",
		}).
		InDomain("bundle")
}

// ErrBundleServiceNotConfigured creates an error for missing bundle service.
func ErrBundleServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Bundle packaging service unavailable").
		WithDetail("internal_error", "bundle packager not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the bundle packager is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("bundle")
}

// Preflight stage errors

// ErrPreflightServiceNotConfigured creates an error for missing preflight service.
func ErrPreflightServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Preflight validation service unavailable").
		WithDetail("internal_error", "preflight service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the preflight service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("preflight")
}

// ErrPreflightValidationFailed creates an error for preflight validation failure.
func ErrPreflightValidationFailed(cause error, validationErrors []string) *DomainError {
	err := Wrap(CodePreflightFailed, cause, "preflight validation failed").
		WithRecovery(RecoveryFixInput, "Fix validation errors and retry").
		InDomain("preflight")

	if len(validationErrors) > 0 {
		err = err.WithDetail("validation_errors", validationErrors)
		steps := []string{"Fix the following validation errors:"}
		for _, e := range validationErrors {
			steps = append(steps, "  - "+e)
		}
		steps = append(steps, "Re-run the pipeline after fixing issues")
		err = err.WithManualSteps(steps)
	}

	return err
}

// ErrPreflightBundleNotAvailable creates an error when bundle result is missing.
func ErrPreflightBundleNotAvailable() *DomainError {
	return New(CodeDependencyError, "bundle result not available from previous stage").
		WithRecovery(RecoveryRetry, "Ensure bundle stage completes successfully first").
		WithManualSteps([]string{
			"Check if the bundle stage completed successfully",
			"Review bundle stage logs for errors",
			"Restart the pipeline from the bundle stage",
		}).
		InDomain("preflight")
}

// ErrPreflightTimeout creates an error for preflight timeout.
func ErrPreflightTimeout(duration string) *DomainError {
	return New(CodePreflightTimeout, "preflight validation timed out").
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Increase timeout and retry").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Increase preflight_timeout_seconds in pipeline config",
			"Check if services are starting slowly",
			"Review resource usage during preflight",
		}).
		InDomain("preflight")
}

// Generate stage errors

// ErrGenerateAnalyzerNotConfigured creates an error for missing scenario analyzer.
func ErrGenerateAnalyzerNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Scenario analysis service unavailable").
		WithDetail("internal_error", "scenario analyzer not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the generation service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("generation")
}

// ErrGenerateServiceNotConfigured creates an error for missing generation service.
func ErrGenerateServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Code generation service unavailable").
		WithDetail("internal_error", "generation service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the generation service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("generation")
}

// ErrScenarioAnalysisFailed creates an error for scenario analysis failure.
func ErrScenarioAnalysisFailed(cause error, scenarioName string) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "scenario analysis failed").
		WithDetail("scenario_name", scenarioName).
		WithRecovery(RecoveryFixInput, "Check scenario configuration and retry").
		WithManualSteps([]string{
			"Verify scenario exists in scenarios/" + scenarioName,
			"Check scenario service.json is valid JSON",
			"Ensure required fields are present in service.json",
		}).
		InDomain("generation")
}

// ErrScenarioValidationFailed creates an error for scenario validation failure.
func ErrScenarioValidationFailed(cause error, scenarioName string) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "scenario validation failed").
		WithDetail("scenario_name", scenarioName).
		WithRecovery(RecoveryFixInput, "Fix scenario configuration for desktop deployment").
		WithManualSteps([]string{
			"Check scenario meets desktop deployment requirements",
			"Verify UI port and entry point are configured",
			"Review desktop deployment documentation",
		}).
		InDomain("generation")
}

// ErrDesktopConfigFailed creates an error for desktop config creation failure.
func ErrDesktopConfigFailed(cause error) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "failed to create desktop config").
		WithRecovery(RecoveryRetry, "Check scenario metadata and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify scenario metadata is complete",
			"Check template type is supported",
			"Review generation service logs",
		}).
		InDomain("generation")
}

// ErrGenerationTimeout creates an error for generation timeout.
func ErrGenerationTimeout(buildID string, duration string) *DomainError {
	return New(CodeTimeout, "generation timed out").
		WithDetail("build_id", buildID).
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Retry with longer timeout").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Check if generation is slow due to large template",
			"Review system resource usage",
			"Consider simplifying the desktop configuration",
		}).
		InDomain("generation")
}

// ErrGenerationFailed creates an error for generation failure.
func ErrGenerationFailed(cause error) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "generation failed").
		WithRecovery(RecoveryRetry, "Check generation logs and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Review generation error logs",
			"Verify template files are intact",
			"Check disk space and permissions",
		}).
		InDomain("generation")
}

// Build stage errors

// ErrBuildServiceNotConfigured creates an error for missing build service.
func ErrBuildServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Build service unavailable").
		WithDetail("internal_error", "build service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the build service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("build")
}

// ErrBuildStoreNotConfigured creates an error for missing build store.
func ErrBuildStoreNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Build tracking service unavailable").
		WithDetail("internal_error", "build store not configured for status polling").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the build store is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("build")
}

// ErrBuildDesktopPathMissing creates an error when desktop path is missing.
func ErrBuildDesktopPathMissing() *DomainError {
	return New(CodeDependencyError, "desktop path not available from generation stage").
		WithRecovery(RecoveryRetry, "Ensure generation stage completes successfully first").
		WithManualSteps([]string{
			"Check if the generation stage completed successfully",
			"Review generation stage logs for errors",
			"Restart the pipeline from the generation stage",
		}).
		InDomain("build")
}

// ErrBuildStartFailed creates an error for build start failure.
func ErrBuildStartFailed(cause error, platform string) *DomainError {
	return Wrap(CodeBuildFailed, cause, "build failed to start").
		WithDetail("platform", platform).
		WithRecovery(RecoveryRetry, "Check build configuration and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify electron-builder is installed",
			"Check platform-specific build tools are available",
			"Review build service logs for details",
		}).
		InDomain("build")
}

// ErrBuildTimedOut creates an error for build timeout.
func ErrBuildTimedOut(buildID string, duration string) *DomainError {
	return New(CodeTimeout, "build timed out").
		WithDetail("build_id", buildID).
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Retry with longer timeout").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Build may be slow due to large assets",
			"Check system resource usage during build",
			"Consider building fewer platforms in parallel",
		}).
		InDomain("build")
}

// ErrBuildPlatformFailed creates an error for platform build failure.
func ErrBuildPlatformFailed(cause error, platform string, lastOutput string) *DomainError {
	err := Wrap(CodeBuildFailed, cause, "build failed").
		WithDetail("platform", platform).
		WithRecovery(RecoveryRetry, "Check build logs and retry").
		WithRetryStrategy(RetryDefault).
		InDomain("build")

	if lastOutput != "" {
		err = err.WithDiagnostic(&DiagnosticContext{
			Process: &ProcessDiagnostic{
				LastOutput: lastOutput,
			},
		})
	}

	// Platform-specific manual steps
	switch platform {
	case "linux":
		err = err.WithManualSteps([]string{
			"Check Linux build dependencies are installed",
			"Verify fpm is available for package creation",
			"Review electron-builder output for errors",
		})
	case "mac":
		err = err.WithManualSteps([]string{
			"macOS builds may require code signing",
			"Check Xcode command line tools are installed",
			"Review electron-builder output for errors",
		})
	case "win":
		err = err.WithManualSteps([]string{
			"Windows builds on Linux require Wine",
			"Check if Wine is installed and configured",
			"Review electron-builder output for errors",
		})
	default:
		err = err.WithManualSteps([]string{
			"Review electron-builder output for errors",
			"Check platform-specific requirements",
			"Verify build configuration is correct",
		})
	}

	return err
}

// Distribution stage errors

// ErrDistributionServiceNotConfigured creates an error for missing distribution service.
func ErrDistributionServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Distribution service unavailable").
		WithDetail("internal_error", "distribution service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the distribution service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("distribution")
}

// ErrDistributionStoreNotConfigured creates an error for missing distribution store.
func ErrDistributionStoreNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Distribution tracking service unavailable").
		WithDetail("internal_error", "distribution store not configured for status polling").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the distribution store is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("distribution")
}

// ErrDistributionStartFailed creates an error for distribution start failure.
func ErrDistributionStartFailed(cause error, target string) *DomainError {
	return Wrap(CodeInternal, cause, "distribution failed to start").
		WithDetail("target", target).
		WithRecovery(RecoveryRetry, "Check distribution configuration and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify distribution target is configured",
			"Check target credentials are valid",
			"Review distribution service logs",
		}).
		InDomain("distribution")
}

// ErrDistributionFailed creates an error for distribution failure.
func ErrDistributionFailed(cause error, target string) *DomainError {
	return Wrap(CodeInternal, cause, "distribution failed").
		WithDetail("target", target).
		WithRecovery(RecoveryRetry, "Check upload logs and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify network connectivity to target",
			"Check target storage quota",
			"Review upload error details",
		}).
		InDomain("distribution")
}

// ErrDistributionTimeout creates an error for distribution timeout.
func ErrDistributionTimeout(distributionID string, duration string) *DomainError {
	return New(CodeTimeout, "distribution timed out").
		WithDetail("distribution_id", distributionID).
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Retry with longer timeout").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Large artifacts may need more upload time",
			"Check network speed to distribution target",
			"Consider uploading fewer artifacts in parallel",
		}).
		InDomain("distribution")
}

// Smoke test stage errors (for use by smoketest package)

// ErrSmokeTestArtifactNotFound creates an error for missing smoke test artifact.
func ErrSmokeTestArtifactNotFound(artifactPath string) *DomainError {
	return New(CodeArtifactNotFound, "smoke test artifact not found").
		WithDetail("artifact_path", artifactPath).
		WithRecovery(RecoveryFixInput, "Ensure build stage completed and produced artifacts").
		WithManualSteps([]string{
			"Verify the build stage completed successfully",
			fmt.Sprintf("Check if artifact exists: ls -la %s", artifactPath),
			"Review build logs for errors",
			"Ensure build output directory is correct",
		}).
		InDomain("smoketest")
}

// ErrSmokeTestExecutionFailed creates an error for smoke test execution failure.
func ErrSmokeTestExecutionFailed(cause error, context map[string]string) *DomainError {
	err := Wrap(CodeSmokeTestFailed, cause, "smoke test execution failed").
		WithRecovery(RecoveryRetry, "Check app startup logs and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check if the application can run manually",
			"Verify all dependencies are installed",
			"Check system logs for crash information",
			"Try running with --verbose flag for more output",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestTimeout creates an error for smoke test timeout.
func ErrSmokeTestTimeout(duration string, context map[string]string) *DomainError {
	err := New(CodeTimeout, "smoke test timed out").
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Increase timeout and retry").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Increase SMOKE_TEST_TIMEOUT_MS environment variable",
			"Check if app startup is slow due to large assets",
			"Profile app initialization to identify bottlenecks",
			"Verify network connectivity if app makes startup requests",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestValidationFailed creates an error for missing success marker.
func ErrSmokeTestValidationFailed(context map[string]string) *DomainError {
	err := New(CodeSmokeTestFailed, "smoke test validation failed: success marker not found").
		WithRecovery(RecoveryFixInput, "Ensure app outputs SMOKE_TEST_RESULT=passed").
		WithManualSteps([]string{
			"Verify app outputs SMOKE_TEST_RESULT=passed on successful startup",
			"Check if app is detecting SMOKE_TEST=1 environment variable",
			"Review app smoke test handler implementation",
			"Ensure app doesn't crash before outputting success marker",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestPlatformError creates an error for platform-specific issues.
func ErrSmokeTestPlatformError(cause error, platform string) *DomainError {
	err := Wrap(CodeSmokeTestFailed, cause, "platform-specific smoke test error").
		WithDetail("platform", platform).
		WithRecovery(RecoveryInstallDependency, "Install required platform dependencies").
		InDomain("smoketest")

	// Platform-specific recovery steps
	switch platform {
	case "linux":
		err = err.WithManualSteps([]string{
			"Install xvfb for headless display: sudo apt-get install xvfb",
			"Set DISPLAY environment variable or ensure X11 is running",
			"Verify libgtk and other Electron dependencies are installed",
		}).WithAutoFix(&AutoFix{
			Command:     "sudo apt-get install -y xvfb libgtk-3-0 libnotify4 libnss3 libxss1 libxtst6 xdg-utils libatspi2.0-0 libdrm2 libgbm1 libasound2",
			Description: "Install common Electron dependencies for Linux",
			Safe:        false,
		})
	case "mac":
		err = err.WithManualSteps([]string{
			"Ensure app is properly signed for macOS",
			"Check Gatekeeper settings: spctl --status",
			"Verify app bundle structure: Contents/MacOS/ exists",
		})
	case "win":
		err = err.WithManualSteps([]string{
			"Ensure .exe file is not blocked by Windows Defender",
			"Check Windows Firewall settings",
			"Verify Visual C++ Redistributable is installed",
		})
	default:
		err = err.WithManualSteps([]string{
			"Verify platform is supported (linux, mac, win)",
			"Check platform-specific documentation",
		})
	}

	return err
}

// ErrSmokeTestTelemetryFailed creates an error for telemetry failures.
func ErrSmokeTestTelemetryFailed(cause error, context map[string]string) *DomainError {
	err := Wrap(CodeTelemetryError, cause, "smoke test telemetry failed").
		WithRecovery(RecoveryRetry, "Check telemetry service and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check telemetry service is running and accessible",
			"Verify network connectivity to telemetry endpoint",
			"Check telemetry file permissions if using file-based fallback",
			"Review telemetry API logs for errors",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestStoreFailed creates an error for persistence failures.
func ErrSmokeTestStoreFailed(cause error) *DomainError {
	return Wrap(CodeInternal, cause, "Could not save test results").
		WithDetail("internal_error", "smoke test store operation failed").
		WithRecovery(RecoveryRetry, "Check disk space and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check available disk space: df -h",
			"Verify file system permissions",
			"Check if data directory exists and is writable",
		}).
		InDomain("smoketest")
}

// ErrSmokeTestCancelled creates an error for cancelled operations.
func ErrSmokeTestCancelled() *DomainError {
	return New(CodePipelineCancelled, "smoke test cancelled").
		WithRecovery(RecoveryNone, "Re-run smoke test if cancellation was unintentional").
		WithManualSteps([]string{
			"Re-run the smoke test if cancellation was unintentional",
			"Check if timeout was too short",
		}).
		InDomain("smoketest")
}
