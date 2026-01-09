// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001]
package vrooli

import (
	"regexp"
	"strings"

	"vrooli-autoheal/internal/checks"
)

// Explicit status patterns from Vrooli CLI output (highest priority)
// These patterns match the structured status line in CLI output
var (
	// Matches "Status:        🟢 RUNNING" or similar running patterns
	runningStatusPattern = regexp.MustCompile(`(?i)status:\s*[🟢✅]\s*running`)
	// Matches "Status:        ⚫ STOPPED" or similar stopped patterns
	stoppedStatusPattern = regexp.MustCompile(`(?i)status:\s*[⚫🔴❌]\s*stopped`)
)

// CLIOutputStatus represents the detected state from CLI output parsing
type CLIOutputStatus int

const (
	// CLIStatusHealthy indicates the resource/scenario is running and healthy
	CLIStatusHealthy CLIOutputStatus = iota
	// CLIStatusStopped indicates the resource/scenario is explicitly stopped
	CLIStatusStopped
	// CLIStatusUnclear indicates the status could not be determined from output
	CLIStatusUnclear
)

// HealthyIndicators are strings that indicate a healthy/running state
// Order matters: check more specific phrases first
var HealthyIndicators = []string{
	"running",
	"healthy",
	"started",
	"active",
	"ok",
	"up",
	"ready",
	"available",
	"online",
	"listening",
}

// StoppedIndicators are strings that indicate a stopped/down state
// Order matters: check more specific phrases first (e.g., "not running" before "running")
var StoppedIndicators = []string{
	"not running",
	"not started",
	"not found",
	"not installed",
	"stopped",
	"inactive",
	"failed",
	"error",
	"down",
	"exited",
	"dead",
	"crashed",
	"terminated",
	"unavailable",
	"offline",
	"disabled",
}

// ClassifyCLIOutput determines the status from Vrooli CLI output.
// This is the central decision point for interpreting resource/scenario status.
//
// Decision logic:
//  1. Check for explicit Vrooli CLI status patterns FIRST (highest priority)
//     These are structured patterns like "Status:  🟢 RUNNING" that definitively
//     indicate the actual running state, regardless of other text in the output.
//  2. Fall back to generic stopped indicators (e.g., "not running")
//  3. Fall back to generic healthy indicators
//  4. If nothing matches → CLIStatusUnclear
//
// The explicit status patterns take priority because CLI output may contain
// error messages, warnings, or other text unrelated to the actual running status.
// For example, "[ERROR] Requirements:" doesn't mean the scenario is stopped.
func ClassifyCLIOutput(output string) CLIOutputStatus {
	// Priority 1: Check for explicit Vrooli CLI status patterns (most reliable)
	// These patterns match the structured "Status:" line in CLI output
	if runningStatusPattern.MatchString(output) {
		return CLIStatusHealthy
	}
	if stoppedStatusPattern.MatchString(output) {
		return CLIStatusStopped
	}

	lower := strings.ToLower(output)

	// Priority 2: Check for stopped indicators (more specific phrases like "not running")
	for _, indicator := range StoppedIndicators {
		if strings.Contains(lower, indicator) {
			return CLIStatusStopped
		}
	}

	// Priority 3: Check for healthy indicators
	for _, indicator := range HealthyIndicators {
		if strings.Contains(lower, indicator) {
			return CLIStatusHealthy
		}
	}

	// Could not determine status
	return CLIStatusUnclear
}

// CLIStatusToCheckStatus converts a CLI output status to a health check status.
// The isCritical parameter determines severity for non-healthy states.
//
// Decision mapping:
//   - CLIStatusHealthy → StatusOK (always)
//   - CLIStatusStopped → StatusCritical if critical, StatusWarning otherwise
//   - CLIStatusUnclear → StatusWarning (always, unknown state is concerning but not critical)
func CLIStatusToCheckStatus(cliStatus CLIOutputStatus, isCritical bool) checks.Status {
	switch cliStatus {
	case CLIStatusHealthy:
		return checks.StatusOK
	case CLIStatusStopped:
		if isCritical {
			return checks.StatusCritical
		}
		return checks.StatusWarning
	case CLIStatusUnclear:
		return checks.StatusWarning
	default:
		return checks.StatusWarning
	}
}

// CLIStatusDescription returns a human-readable description of the CLI status
func CLIStatusDescription(cliStatus CLIOutputStatus, subjectName string) string {
	switch cliStatus {
	case CLIStatusHealthy:
		return subjectName + " is running"
	case CLIStatusStopped:
		return subjectName + " is stopped"
	case CLIStatusUnclear:
		return subjectName + " status unclear"
	default:
		return subjectName + " status unknown"
	}
}
