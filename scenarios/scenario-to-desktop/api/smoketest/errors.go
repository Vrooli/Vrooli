// Package smoketest provides smoke test execution and error handling.
//
// This file bridges the smoketest-local error types with the shared errors package.
// The ErrorKind and Error types are kept for backward compatibility with the
// Status type's JSON serialization, but internally use shared error infrastructure.
package smoketest

import (
	"fmt"

	"scenario-to-desktop-api/shared/errors"
)

// ErrorKind categorizes smoke test errors for distinct handling.
// Kept for backward compatibility with Status.ErrorKind JSON field.
type ErrorKind int

const (
	// ErrKindArtifact indicates file not found, permission denied, or invalid format.
	ErrKindArtifact ErrorKind = iota
	// ErrKindExecution indicates the process failed to start or run.
	ErrKindExecution
	// ErrKindTimeout indicates the app didn't respond in time.
	ErrKindTimeout
	// ErrKindValidation indicates missing success marker in output.
	ErrKindValidation
	// ErrKindTelemetry indicates telemetry upload/fallback failed.
	ErrKindTelemetry
	// ErrKindPlatform indicates a platform-specific issue (xvfb-run missing, etc.).
	ErrKindPlatform
	// ErrKindStore indicates a persistence failure.
	ErrKindStore
	// ErrKindCancelled indicates user/system cancelled the operation.
	ErrKindCancelled
)

// String returns the string representation of an ErrorKind.
func (k ErrorKind) String() string {
	switch k {
	case ErrKindArtifact:
		return "artifact"
	case ErrKindExecution:
		return "execution"
	case ErrKindTimeout:
		return "timeout"
	case ErrKindValidation:
		return "validation"
	case ErrKindTelemetry:
		return "telemetry"
	case ErrKindPlatform:
		return "platform"
	case ErrKindStore:
		return "store"
	case ErrKindCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Error is a structured smoke test error with context for diagnosis.
// This type is used internally for rich error handling and is converted
// to Status fields when storing results.
type Error struct {
	Kind            ErrorKind
	Message         string
	Cause           error
	Context         map[string]string // Platform, artifact path, command, etc.
	Recoverable     bool
	SuggestedAction string

	// Enhanced recovery information (from shared errors)
	RetryStrategy *errors.RetryStrategy     // Retry configuration if applicable
	AutoFix       *errors.AutoFix           // Automatic fix command if safe to run
	ManualSteps   []string                  // Ordered manual resolution steps
	Diagnostic    *errors.DiagnosticContext // Additional diagnostic information
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ToDomainError converts this Error to a shared DomainError.
// This enables integration with the HTTP error handling infrastructure.
func (e *Error) ToDomainError() *errors.DomainError {
	code := e.mapKindToCode()
	de := errors.New(code, e.Message).
		InDomain("smoketest").
		WithCause(e.Cause)

	// Copy context to details
	if len(e.Context) > 0 {
		details := make(map[string]interface{})
		for k, v := range e.Context {
			details[k] = v
		}
		de = de.WithDetails(details)
	}

	// Copy recovery information
	if e.SuggestedAction != "" {
		de = de.WithRecovery(de.DefaultRecovery(), e.SuggestedAction)
	}
	if e.RetryStrategy != nil {
		de = de.WithRetryStrategy(e.RetryStrategy)
	}
	if e.AutoFix != nil {
		de = de.WithAutoFix(e.AutoFix)
	}
	if len(e.ManualSteps) > 0 {
		de = de.WithManualSteps(e.ManualSteps)
	}
	if e.Diagnostic != nil {
		de = de.WithDiagnostic(e.Diagnostic)
	}

	return de
}

// mapKindToCode maps ErrorKind to shared ErrorCode.
func (e *Error) mapKindToCode() errors.ErrorCode {
	switch e.Kind {
	case ErrKindArtifact:
		return errors.CodeArtifactNotFound
	case ErrKindExecution:
		return errors.CodeSmokeTestFailed
	case ErrKindTimeout:
		return errors.CodeTimeout
	case ErrKindValidation:
		return errors.CodeSmokeTestFailed
	case ErrKindTelemetry:
		return errors.CodeTelemetryError
	case ErrKindPlatform:
		return errors.CodeSmokeTestFailed
	case ErrKindStore:
		return errors.CodeInternal
	case ErrKindCancelled:
		return errors.CodePipelineCancelled
	default:
		return errors.CodeInternal
	}
}

// RecoveryPaths maps error kinds to recovery suggestions.
var RecoveryPaths = map[ErrorKind]string{
	ErrKindArtifact:   "Check build stage output, verify artifact exists and is readable",
	ErrKindExecution:  "Check app startup logs, verify platform compatibility",
	ErrKindTimeout:    "Increase timeout or optimize app startup time",
	ErrKindValidation: "Ensure app outputs SMOKE_TEST_RESULT=passed on success",
	ErrKindTelemetry:  "Check telemetry service availability and network connectivity",
	ErrKindPlatform:   "Install required dependencies (e.g., xvfb for Linux headless)",
	ErrKindStore:      "Check disk space and file permissions",
	ErrKindCancelled:  "Re-run smoke test if needed",
}

// NewArtifactError creates an error for artifact-related issues.
func NewArtifactError(msg string, cause error, artifactPath string) *Error {
	return &Error{
		Kind:            ErrKindArtifact,
		Message:         msg,
		Cause:           cause,
		Context:         map[string]string{"artifact_path": artifactPath},
		Recoverable:     false,
		SuggestedAction: RecoveryPaths[ErrKindArtifact],
		ManualSteps: []string{
			"Verify the build stage completed successfully",
			fmt.Sprintf("Check if artifact exists: ls -la %s", artifactPath),
			"Review build logs for errors",
			"Ensure build output directory is correct",
		},
	}
}

// NewExecutionError creates an error for process execution failures.
func NewExecutionError(msg string, cause error, context map[string]string) *Error {
	return &Error{
		Kind:            ErrKindExecution,
		Message:         msg,
		Cause:           cause,
		Context:         context,
		Recoverable:     true,
		SuggestedAction: RecoveryPaths[ErrKindExecution],
		RetryStrategy:   errors.RetryDefault,
		ManualSteps: []string{
			"Check if the application can run manually",
			"Verify all dependencies are installed",
			"Check system logs for crash information",
			"Use --show-output with 'pipeline run' to see full app output",
		},
	}
}

// NewTimeoutError creates an error for timeout failures.
func NewTimeoutError(msg string, cause error, context map[string]string) *Error {
	return &Error{
		Kind:            ErrKindTimeout,
		Message:         msg,
		Cause:           cause,
		Context:         context,
		Recoverable:     true,
		SuggestedAction: RecoveryPaths[ErrKindTimeout],
		RetryStrategy:   errors.RetryConservative,
		ManualSteps: []string{
			"Increase SMOKE_TEST_TIMEOUT_MS environment variable",
			"Check if app startup is slow due to large assets",
			"Profile app initialization to identify bottlenecks",
			"Verify network connectivity if app makes startup requests",
		},
	}
}

// NewValidationError creates an error for missing success markers.
func NewValidationError(msg string, context map[string]string) *Error {
	return &Error{
		Kind:            ErrKindValidation,
		Message:         msg,
		Cause:           nil,
		Context:         context,
		Recoverable:     false,
		SuggestedAction: RecoveryPaths[ErrKindValidation],
		ManualSteps: []string{
			"Verify app outputs SMOKE_TEST_RESULT=passed on successful startup",
			"Check if app is detecting SMOKE_TEST=1 environment variable",
			"Review app smoke test handler implementation",
			"Ensure app doesn't crash before outputting success marker",
		},
	}
}

// NewTelemetryError creates an error for telemetry failures.
func NewTelemetryError(msg string, cause error, context map[string]string) *Error {
	return &Error{
		Kind:            ErrKindTelemetry,
		Message:         msg,
		Cause:           cause,
		Context:         context,
		Recoverable:     true,
		SuggestedAction: RecoveryPaths[ErrKindTelemetry],
		RetryStrategy:   errors.RetryDefault,
		ManualSteps: []string{
			"Check telemetry service is running and accessible",
			"Verify network connectivity to telemetry endpoint",
			"Check telemetry file permissions if using file-based fallback",
			"Review telemetry API logs for errors",
		},
	}
}

// NewPlatformError creates an error for platform-specific issues.
func NewPlatformError(msg string, cause error, platform string) *Error {
	err := &Error{
		Kind:            ErrKindPlatform,
		Message:         msg,
		Cause:           cause,
		Context:         map[string]string{"platform": platform},
		Recoverable:     false,
		SuggestedAction: RecoveryPaths[ErrKindPlatform],
	}

	// Platform-specific recovery steps
	switch platform {
	case "linux":
		err.ManualSteps = []string{
			"Install xvfb for headless display: sudo apt-get install xvfb",
			"Set DISPLAY environment variable or ensure X11 is running",
			"Verify libgtk and other Electron dependencies are installed",
		}
		err.AutoFix = &errors.AutoFix{
			Command:     "sudo apt-get install -y xvfb libgtk-3-0 libnotify4 libnss3 libxss1 libxtst6 xdg-utils libatspi2.0-0 libdrm2 libgbm1 libasound2",
			Description: "Install common Electron dependencies for Linux",
			Safe:        false,
		}
	case "mac":
		err.ManualSteps = []string{
			"Ensure app is properly signed for macOS",
			"Check Gatekeeper settings: spctl --status",
			"Verify app bundle structure: Contents/MacOS/ exists",
		}
	case "win":
		err.ManualSteps = []string{
			"Ensure .exe file is not blocked by Windows Defender",
			"Check Windows Firewall settings",
			"Verify Visual C++ Redistributable is installed",
		}
	default:
		err.ManualSteps = []string{
			"Verify platform is supported (linux, mac, win)",
			"Check platform-specific documentation",
		}
	}

	return err
}

// NewStoreError creates an error for persistence failures.
func NewStoreError(msg string, cause error) *Error {
	return &Error{
		Kind:            ErrKindStore,
		Message:         msg,
		Cause:           cause,
		Context:         nil,
		Recoverable:     true,
		SuggestedAction: RecoveryPaths[ErrKindStore],
		RetryStrategy:   errors.RetryDefault,
		ManualSteps: []string{
			"Check available disk space: df -h",
			"Verify file system permissions",
			"Check if data directory exists and is writable",
		},
	}
}

// NewCancelledError creates an error for cancelled operations.
func NewCancelledError(msg string) *Error {
	return &Error{
		Kind:            ErrKindCancelled,
		Message:         msg,
		Cause:           nil,
		Context:         nil,
		Recoverable:     false,
		SuggestedAction: RecoveryPaths[ErrKindCancelled],
		ManualSteps: []string{
			"Re-run the smoke test if cancellation was unintentional",
			"Check if timeout was too short",
		},
	}
}

// NewAppReportedError creates an error from app-side structured error output.
// This is used when the app emits SMOKE_TEST_ERROR markers that we can parse.
func NewAppReportedError(appErr *AppError) *Error {
	kind := ErrKindExecution // Default
	switch appErr.Kind {
	case "config":
		kind = ErrKindArtifact
	case "network":
		kind = ErrKindExecution
	case "validation":
		kind = ErrKindValidation
	case "runtime":
		kind = ErrKindExecution
	}

	return &Error{
		Kind:            kind,
		Message:         fmt.Sprintf("App reported error: %s", appErr.Message),
		Context:         map[string]string{"app_error_kind": appErr.Kind},
		Recoverable:     kind == ErrKindExecution,
		SuggestedAction: getRecoveryForAppError(appErr.Kind),
		ManualSteps:     getManualStepsForAppError(appErr.Kind),
	}
}

// getRecoveryForAppError returns a recovery suggestion based on the app error kind.
func getRecoveryForAppError(kind string) string {
	switch kind {
	case "config":
		return "Check build configuration and regenerate the desktop app"
	case "network":
		return "Verify the server is running and reachable, check network connectivity"
	case "validation":
		return "Review bundle validation errors and rebuild the bundle"
	case "runtime":
		return "Check app startup logs and verify platform compatibility"
	default:
		return RecoveryPaths[ErrKindExecution]
	}
}

// getManualStepsForAppError returns manual resolution steps based on the app error kind.
func getManualStepsForAppError(kind string) []string {
	switch kind {
	case "config":
		return []string{
			"Verify build configuration files exist and are valid",
			"Regenerate the desktop app with scenario-to-desktop",
			"Check for missing or corrupted bundle files",
		}
	case "network":
		return []string{
			"Verify the target server is running: curl -I <server-url>",
			"Check if the server port is accessible",
			"Verify network connectivity and firewall settings",
			"Increase SMOKE_TEST_TIMEOUT_MS if server startup is slow",
		}
	case "validation":
		return []string{
			"Review the bundle validation output for specific errors",
			"Rebuild the bundle with scenario-to-desktop",
			"Verify all required assets are present in the bundle",
		}
	case "runtime":
		return []string{
			"Check if the application can run manually",
			"Verify all dependencies are installed",
			"Check system logs for crash information",
			"Use --show-output with 'pipeline run' to see full app output",
		}
	default:
		return []string{
			"Review the smoke test output for error details",
			"Check app startup logs",
		}
	}
}
