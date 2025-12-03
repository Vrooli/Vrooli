// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001]
package vrooli

import (
	"strings"

	"vrooli-autoheal/internal/checks"
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
var HealthyIndicators = []string{"running", "healthy", "started", "active"}

// StoppedIndicators are strings that indicate a stopped/down state
var StoppedIndicators = []string{"stopped", "not running", "down", "exited", "dead"}

// ClassifyCLIOutput determines the status from Vrooli CLI output.
// This is the central decision point for interpreting resource/scenario status.
//
// Decision logic:
//   - Check for stopped indicators FIRST (they are more specific, e.g., "not running")
//   - Then check for healthy indicators
//   - If neither matches → CLIStatusUnclear
//
// The indicators are checked case-insensitively.
// Note: Stopped indicators are checked first because phrases like "not running"
// contain the word "running" which would otherwise match as healthy.
func ClassifyCLIOutput(output string) CLIOutputStatus {
	lower := strings.ToLower(output)

	// Check for stopped indicators FIRST (more specific, e.g., "not running" vs "running")
	for _, indicator := range StoppedIndicators {
		if strings.Contains(lower, indicator) {
			return CLIStatusStopped
		}
	}

	// Check for healthy indicators
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
