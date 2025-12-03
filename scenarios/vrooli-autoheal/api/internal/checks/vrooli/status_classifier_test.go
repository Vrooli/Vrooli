// Package vrooli provides Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001]
package vrooli

import (
	"testing"

	"vrooli-autoheal/internal/checks"
)

func TestClassifyCLIOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected CLIOutputStatus
	}{
		// Healthy indicators
		{
			name:     "running keyword",
			output:   "postgres: RUNNING (pid 1234)",
			expected: CLIStatusHealthy,
		},
		{
			name:     "healthy keyword",
			output:   "Status: healthy",
			expected: CLIStatusHealthy,
		},
		{
			name:     "started keyword",
			output:   "Service started successfully",
			expected: CLIStatusHealthy,
		},
		{
			name:     "active keyword",
			output:   "Container is active",
			expected: CLIStatusHealthy,
		},
		{
			name:     "case insensitive healthy",
			output:   "RUNNING",
			expected: CLIStatusHealthy,
		},
		// Stopped indicators
		{
			name:     "stopped keyword",
			output:   "postgres: stopped",
			expected: CLIStatusStopped,
		},
		{
			name:     "not running phrase",
			output:   "Service is not running",
			expected: CLIStatusStopped,
		},
		{
			name:     "down keyword",
			output:   "Container is down",
			expected: CLIStatusStopped,
		},
		{
			name:     "exited keyword",
			output:   "Container exited with code 1",
			expected: CLIStatusStopped,
		},
		{
			name:     "dead keyword",
			output:   "Process is dead",
			expected: CLIStatusStopped,
		},
		// Unclear status
		{
			name:     "empty output",
			output:   "",
			expected: CLIStatusUnclear,
		},
		{
			name:     "unknown status",
			output:   "Status: unknown",
			expected: CLIStatusUnclear,
		},
		{
			name:     "no relevant keywords",
			output:   "postgres version 15.2",
			expected: CLIStatusUnclear,
		},
		// Edge cases - stopped indicators are checked first to handle "not running" etc.
		{
			name:     "stopped takes precedence when both present",
			output:   "Service stopped then started, now running",
			expected: CLIStatusStopped, // "stopped" checked before "running" to handle "not running"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyCLIOutput(tt.output)
			if result != tt.expected {
				t.Errorf("ClassifyCLIOutput(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}

func TestCLIStatusToCheckStatus(t *testing.T) {
	tests := []struct {
		name       string
		cliStatus  CLIOutputStatus
		isCritical bool
		expected   checks.Status
	}{
		// Healthy always returns OK
		{
			name:       "healthy critical resource",
			cliStatus:  CLIStatusHealthy,
			isCritical: true,
			expected:   checks.StatusOK,
		},
		{
			name:       "healthy non-critical scenario",
			cliStatus:  CLIStatusHealthy,
			isCritical: false,
			expected:   checks.StatusOK,
		},
		// Stopped returns Critical or Warning based on criticality
		{
			name:       "stopped critical resource",
			cliStatus:  CLIStatusStopped,
			isCritical: true,
			expected:   checks.StatusCritical,
		},
		{
			name:       "stopped non-critical scenario",
			cliStatus:  CLIStatusStopped,
			isCritical: false,
			expected:   checks.StatusWarning,
		},
		// Unclear always returns Warning
		{
			name:       "unclear critical",
			cliStatus:  CLIStatusUnclear,
			isCritical: true,
			expected:   checks.StatusWarning,
		},
		{
			name:       "unclear non-critical",
			cliStatus:  CLIStatusUnclear,
			isCritical: false,
			expected:   checks.StatusWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CLIStatusToCheckStatus(tt.cliStatus, tt.isCritical)
			if result != tt.expected {
				t.Errorf("CLIStatusToCheckStatus(%v, %v) = %v, want %v",
					tt.cliStatus, tt.isCritical, result, tt.expected)
			}
		})
	}
}

func TestCLIStatusDescription(t *testing.T) {
	tests := []struct {
		name        string
		cliStatus   CLIOutputStatus
		subjectName string
		expected    string
	}{
		{
			name:        "healthy postgres",
			cliStatus:   CLIStatusHealthy,
			subjectName: "postgres resource",
			expected:    "postgres resource is running",
		},
		{
			name:        "stopped scenario",
			cliStatus:   CLIStatusStopped,
			subjectName: "my-app scenario",
			expected:    "my-app scenario is stopped",
		},
		{
			name:        "unclear redis",
			cliStatus:   CLIStatusUnclear,
			subjectName: "redis resource",
			expected:    "redis resource status unclear",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CLIStatusDescription(tt.cliStatus, tt.subjectName)
			if result != tt.expected {
				t.Errorf("CLIStatusDescription(%v, %q) = %q, want %q",
					tt.cliStatus, tt.subjectName, result, tt.expected)
			}
		})
	}
}
