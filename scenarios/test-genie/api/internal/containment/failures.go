// Package containment provides OS-level sandboxing for agent execution.
// This file defines failure types specific to containment operations.
package containment

import (
	"fmt"
)

// ContainmentFailureCode identifies specific containment failures.
type ContainmentFailureCode string

const (
	// FailureCodeDockerBinaryNotFound indicates docker command is not in PATH.
	FailureCodeDockerBinaryNotFound ContainmentFailureCode = "docker_binary_not_found"

	// FailureCodeDockerDaemonNotRunning indicates docker is installed but daemon is down.
	FailureCodeDockerDaemonNotRunning ContainmentFailureCode = "docker_daemon_not_running"

	// FailureCodeDockerTimeout indicates docker check timed out.
	FailureCodeDockerTimeout ContainmentFailureCode = "docker_timeout"

	// FailureCodeBubblewrapUnavailable indicates bubblewrap is not available.
	FailureCodeBubblewrapUnavailable ContainmentFailureCode = "bubblewrap_unavailable"

	// FailureCodeNoContainmentAvailable indicates no sandbox methods are available.
	FailureCodeNoContainmentAvailable ContainmentFailureCode = "no_containment_available"

	// FailureCodePrepareCommandFailed indicates container setup failed.
	FailureCodePrepareCommandFailed ContainmentFailureCode = "prepare_command_failed"
)

// ContainmentFailure provides structured information about a containment failure.
type ContainmentFailure struct {
	// Code identifies the specific failure type.
	Code ContainmentFailureCode

	// Provider is the containment type that failed.
	Provider ContainmentType

	// Message is a user-safe description.
	Message string

	// RecoveryHint provides guidance on what to do next.
	RecoveryHint string

	// InternalDetails contains debugging info (not for users).
	InternalDetails string

	// IsDegradable indicates whether operation can continue without containment.
	IsDegradable bool
}

// Error implements the error interface.
func (f ContainmentFailure) Error() string {
	return f.Message
}

// WithInternalDetails adds internal debugging info.
func (f ContainmentFailure) WithInternalDetails(details string) ContainmentFailure {
	f.InternalDetails = details
	return f
}

// --- Factory Functions ---

// NewDockerBinaryNotFoundFailure creates a failure for missing docker binary.
func NewDockerBinaryNotFoundFailure() ContainmentFailure {
	return ContainmentFailure{
		Code:         FailureCodeDockerBinaryNotFound,
		Provider:     ContainmentTypeDocker,
		Message:      "Docker is not installed",
		RecoveryHint: "Install Docker for containerized agent isolation: https://docs.docker.com/get-docker/",
		IsDegradable: true,
	}
}

// NewDockerDaemonNotRunningFailure creates a failure for stopped docker daemon.
func NewDockerDaemonNotRunningFailure() ContainmentFailure {
	return ContainmentFailure{
		Code:         FailureCodeDockerDaemonNotRunning,
		Provider:     ContainmentTypeDocker,
		Message:      "Docker daemon is not running",
		RecoveryHint: "Start Docker Desktop or run 'sudo systemctl start docker'",
		IsDegradable: true,
	}
}

// NewDockerTimeoutFailure creates a failure for docker availability check timeout.
func NewDockerTimeoutFailure(timeout int) ContainmentFailure {
	return ContainmentFailure{
		Code:            FailureCodeDockerTimeout,
		Provider:        ContainmentTypeDocker,
		Message:         "Docker availability check timed out",
		RecoveryHint:    fmt.Sprintf("Docker daemon may be slow to respond. Check 'docker info' manually or increase CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS (currently %d)", timeout),
		IsDegradable:    true,
		InternalDetails: fmt.Sprintf("availability check exceeded %ds timeout", timeout),
	}
}

// NewNoContainmentAvailableFailure creates a failure when no sandbox methods work.
func NewNoContainmentAvailableFailure(checkedProviders []ProviderCheckResult) ContainmentFailure {
	var details string
	for _, p := range checkedProviders {
		details += fmt.Sprintf("%s: %s; ", p.Type, p.Reason)
	}
	return ContainmentFailure{
		Code:            FailureCodeNoContainmentAvailable,
		Provider:        ContainmentTypeNone,
		Message:         "No OS-level containment available",
		RecoveryHint:    "Install Docker for secure agent execution, or set CONTAINMENT_ALLOW_FALLBACK=true to run without containment",
		IsDegradable:    true, // Depends on config.AllowFallback
		InternalDetails: details,
	}
}

// NewPrepareCommandFailure creates a failure for container setup errors.
func NewPrepareCommandFailure(provider ContainmentType, err error) ContainmentFailure {
	return ContainmentFailure{
		Code:            FailureCodePrepareCommandFailed,
		Provider:        provider,
		Message:         "Failed to prepare containerized execution",
		RecoveryHint:    "Check Docker logs for details or try restarting Docker",
		IsDegradable:    true,
		InternalDetails: err.Error(),
	}
}

// --- Degradation Decision ---

// ContainmentDegradationDecision describes how to handle containment failure.
type ContainmentDegradationDecision struct {
	// ShouldContinue indicates whether to proceed without containment.
	ShouldContinue bool

	// Reason explains the decision.
	Reason string

	// WarningForUser is shown if continuing in degraded mode.
	WarningForUser string

	// SecurityLevel is the effective security level (0 if degraded).
	SecurityLevel int

	// OriginalFailure is the failure that triggered this decision.
	OriginalFailure *ContainmentFailure
}

// DecideOnContainmentFailure determines whether to degrade or fail.
// This is the central decision point for containment degradation.
//
// Decision criteria:
//   - Config allows fallback + failure is degradable → continue with warning
//   - Config forbids fallback → fail (return ShouldContinue=false)
//   - Non-degradable failure → fail
func DecideOnContainmentFailure(config Config, failure ContainmentFailure) ContainmentDegradationDecision {
	// Check if degradation is allowed
	if !config.AllowFallback {
		return ContainmentDegradationDecision{
			ShouldContinue:  false,
			Reason:          "CONTAINMENT_ALLOW_FALLBACK is false; refusing to run without containment",
			OriginalFailure: &failure,
			SecurityLevel:   0,
		}
	}

	if !failure.IsDegradable {
		return ContainmentDegradationDecision{
			ShouldContinue:  false,
			Reason:          fmt.Sprintf("failure %s is not degradable", failure.Code),
			OriginalFailure: &failure,
			SecurityLevel:   0,
		}
	}

	// Allow degraded execution with warning
	warning := "Running without OS-level isolation. "
	switch failure.Code {
	case FailureCodeDockerBinaryNotFound:
		warning += "Install Docker for improved security."
	case FailureCodeDockerDaemonNotRunning:
		warning += "Start Docker for improved security."
	case FailureCodeDockerTimeout:
		warning += "Docker is slow; consider increasing the timeout."
	default:
		warning += "Consider setting up Docker for better isolation."
	}

	return ContainmentDegradationDecision{
		ShouldContinue:  true,
		Reason:          "degrading to direct execution; agent still has tool-level restrictions",
		WarningForUser:  warning,
		SecurityLevel:   0,
		OriginalFailure: &failure,
	}
}

// --- Structured Logging Support ---

// ToLogFields returns a map suitable for structured logging.
// This excludes overly detailed internal info for cleaner logs.
func (f ContainmentFailure) ToLogFields() map[string]interface{} {
	fields := map[string]interface{}{
		"containment_failure_code":     string(f.Code),
		"containment_provider":         string(f.Provider),
		"containment_failure_message":  f.Message,
		"containment_failure_degradable": f.IsDegradable,
	}
	return fields
}

// ToAPIResponse returns a map suitable for API error responses.
// This excludes internal details.
func (f ContainmentFailure) ToAPIResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"code":         string(f.Code),
		"message":      f.Message,
		"recoveryHint": f.RecoveryHint,
	}
	return resp
}
