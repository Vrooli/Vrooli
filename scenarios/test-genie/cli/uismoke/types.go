package uismoke

import (
	"encoding/json"
	"time"
)

// Request represents the UI smoke test request payload.
type Request struct {
	URL       string `json:"url,omitempty"`
	TimeoutMs int64  `json:"timeout_ms,omitempty"`
	AutoStart bool   `json:"auto_start,omitempty"`

	// ScenarioPath overrides the scenario directory path. Set by the CLI
	// when running inside a sandboxed agent.
	// See packages/cli-core/cliutil/sandbox.go for sandbox path resolution.
	ScenarioPath string `json:"scenarioPath,omitempty"`
}

// Response represents the UI smoke test result.
type Response struct {
	Scenario            string          `json:"scenario"`
	Status              string          `json:"status"`
	BlockedReason       string          `json:"blocked_reason,omitempty"`
	Message             string          `json:"message"`
	Timestamp           time.Time       `json:"timestamp"`
	DurationMs          int64           `json:"duration_ms"`
	UIURL               string          `json:"ui_url,omitempty"`
	Handshake           json.RawMessage `json:"handshake,omitempty"`
	NetworkFailureCount int             `json:"network_failure_count"`
	PageErrorCount      int             `json:"page_error_count"`
	ConsoleErrorCount   int             `json:"console_error_count"`
	Artifacts           json.RawMessage `json:"artifacts,omitempty"`
	Bundle              json.RawMessage `json:"bundle,omitempty"`
}

// Exit codes mirror the API smoke runner's BlockedReason.ExitCode mapping
// (scenarios/test-genie/api/internal/smoke/result.go).
const (
	// ExitSuccess indicates the test passed.
	ExitSuccess = 0
	// ExitFailure indicates the test failed.
	ExitFailure = 1
	// ExitBASUnavailable indicates the Browser Automation Studio workflow engine
	// is unreachable, so the smoke capture could not run.
	ExitBASUnavailable = 50
	// ExitBundleStale indicates UI bundle is outdated.
	ExitBundleStale = 60
	// ExitUIPortMissing indicates UI port is defined but not detected.
	ExitUIPortMissing = 61
)

// ExitCodeForBlockedReason returns the exit code for a blocked reason.
func ExitCodeForBlockedReason(reason string) int {
	switch reason {
	case "bas_unavailable":
		return ExitBASUnavailable
	case "bundle_stale":
		return ExitBundleStale
	case "ui_port_missing":
		return ExitUIPortMissing
	default:
		return ExitFailure
	}
}

// Args holds parsed command line arguments.
type Args struct {
	Scenario  string
	URL       string
	TimeoutMs int64
	JSON      bool
	AutoStart bool
}
