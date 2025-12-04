package uismoke

import (
	"encoding/json"
	"time"
)

// Request represents the UI smoke test request payload.
type Request struct {
	URL            string `json:"url,omitempty"`
	BrowserlessURL string `json:"browserless_url,omitempty"`
	TimeoutMs      int64  `json:"timeout_ms,omitempty"`
	NoRecovery     bool   `json:"no_recovery,omitempty"`
	SharedMode     bool   `json:"shared_mode,omitempty"`
	AutoStart      bool   `json:"auto_start,omitempty"`
}

// Response represents the UI smoke test result.
type Response struct {
	Scenario      string          `json:"scenario"`
	Status        string          `json:"status"`
	BlockedReason string          `json:"blocked_reason,omitempty"`
	Message       string          `json:"message"`
	Timestamp     time.Time       `json:"timestamp"`
	DurationMs    int64           `json:"duration_ms"`
	UIURL         string          `json:"ui_url,omitempty"`
	Handshake     json.RawMessage `json:"handshake,omitempty"`
	Artifacts     json.RawMessage `json:"artifacts,omitempty"`
	Bundle        json.RawMessage `json:"bundle,omitempty"`
}

// Exit codes compatible with legacy bash implementation.
const (
	// ExitSuccess indicates the test passed.
	ExitSuccess = 0
	// ExitFailure indicates the test failed.
	ExitFailure = 1
	// ExitBrowserlessOffline indicates Browserless service is unavailable.
	ExitBrowserlessOffline = 50
	// ExitBundleStale indicates UI bundle is outdated.
	ExitBundleStale = 60
	// ExitUIPortMissing indicates UI port is defined but not detected.
	ExitUIPortMissing = 61
)

// ExitCodeForBlockedReason returns the exit code for a blocked reason.
func ExitCodeForBlockedReason(reason string) int {
	switch reason {
	case "browserless_offline":
		return ExitBrowserlessOffline
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
	Scenario       string
	URL            string
	BrowserlessURL string
	TimeoutMs      int64
	JSON           bool
	NoRecovery     bool
	SharedMode     bool
	AutoStart      bool
}
