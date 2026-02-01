package smoketest

import "fmt"

// ErrorKind categorizes smoke test errors for distinct handling.
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
type Error struct {
	Kind            ErrorKind
	Message         string
	Cause           error
	Context         map[string]string // Platform, artifact path, command, etc.
	Recoverable     bool
	SuggestedAction string

	// Enhanced recovery information
	RetryStrategy *RetryStrategy     // Retry configuration if applicable
	AutoFix       *AutoFix           // Automatic fix command if safe to run
	ManualSteps   []string           // Ordered manual resolution steps
	Diagnostic    *DiagnosticContext // Additional diagnostic information
}

// RetryStrategy defines how to retry a failed operation.
type RetryStrategy struct {
	// MaxAttempts is the maximum number of retry attempts.
	MaxAttempts int

	// BackoffMs is the initial backoff duration in milliseconds.
	BackoffMs int

	// BackoffMultiplier increases backoff between attempts (default 2.0).
	BackoffMultiplier float64
}

// AutoFix describes an automatic fix that can be applied.
type AutoFix struct {
	// Command is the shell command to run.
	Command string

	// Description explains what the command does.
	Description string

	// Safe indicates if the command is safe to run without confirmation.
	Safe bool
}

// DiagnosticContext provides additional information for debugging.
type DiagnosticContext struct {
	// Process contains process-related diagnostics.
	Process *ProcessDiagnostic

	// System contains system-related information.
	System map[string]string
}

// ProcessDiagnostic contains process execution details.
type ProcessDiagnostic struct {
	// PID is the process ID if available.
	PID int

	// ExitCode is the exit code if the process terminated.
	ExitCode int

	// RuntimeMs is how long the process ran in milliseconds.
	RuntimeMs int64

	// LastOutput is the last portion of output before failure.
	LastOutput string
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
		RetryStrategy: &RetryStrategy{
			MaxAttempts:       3,
			BackoffMs:         1000,
			BackoffMultiplier: 2.0,
		},
		ManualSteps: []string{
			"Check if the application can run manually",
			"Verify all dependencies are installed",
			"Check system logs for crash information",
			"Try running with --verbose flag for more output",
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
		RetryStrategy: &RetryStrategy{
			MaxAttempts:       2,
			BackoffMs:         5000,
			BackoffMultiplier: 1.5,
		},
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
		RetryStrategy: &RetryStrategy{
			MaxAttempts:       3,
			BackoffMs:         2000,
			BackoffMultiplier: 2.0,
		},
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
		err.AutoFix = &AutoFix{
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
		RetryStrategy: &RetryStrategy{
			MaxAttempts:       3,
			BackoffMs:         500,
			BackoffMultiplier: 2.0,
		},
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
