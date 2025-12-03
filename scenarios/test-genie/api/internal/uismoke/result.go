package uismoke

import (
	"encoding/json"
	"time"
)

// Status represents the outcome of a UI smoke test.
type Status string

const (
	// StatusPassed indicates the smoke test completed successfully.
	StatusPassed Status = "passed"
	// StatusFailed indicates the smoke test encountered errors.
	StatusFailed Status = "failed"
	// StatusSkipped indicates the smoke test was skipped (e.g., no UI).
	StatusSkipped Status = "skipped"
	// StatusBlocked indicates the smoke test could not run due to preconditions.
	StatusBlocked Status = "blocked"
)

// IsSuccess returns true if the status represents a successful outcome.
func (s Status) IsSuccess() bool {
	return s == StatusPassed
}

// IsTerminal returns true if the status represents a final state.
func (s Status) IsTerminal() bool {
	return s == StatusPassed || s == StatusFailed || s == StatusSkipped || s == StatusBlocked
}

// Result holds the complete outcome of a UI smoke test.
type Result struct {
	// Scenario is the name of the tested scenario.
	Scenario string `json:"scenario"`

	// Status is the overall test outcome.
	Status Status `json:"status"`

	// Message provides a human-readable summary.
	Message string `json:"message"`

	// Timestamp is when the test completed.
	Timestamp time.Time `json:"timestamp"`

	// DurationMs is the total test duration in milliseconds.
	DurationMs int64 `json:"duration_ms"`

	// UIURL is the URL that was tested.
	UIURL string `json:"ui_url,omitempty"`

	// Handshake contains iframe-bridge handshake results.
	Handshake HandshakeResult `json:"handshake,omitempty"`

	// Artifacts contains paths to generated artifacts.
	Artifacts ArtifactPaths `json:"artifacts,omitempty"`

	// Bundle contains bundle freshness information.
	Bundle *BundleStatus `json:"bundle,omitempty"`

	// IframeBridge contains bridge dependency information.
	IframeBridge *BridgeStatus `json:"iframe_bridge,omitempty"`

	// Browserless contains browserless resource information.
	Browserless json.RawMessage `json:"browserless,omitempty"`

	// StorageShim contains storage shim patching results.
	StorageShim []StorageShimEntry `json:"storage_shim,omitempty"`

	// Raw contains the raw browserless response (excluding screenshot).
	Raw json.RawMessage `json:"raw,omitempty"`
}

// HandshakeResult describes the iframe-bridge handshake outcome.
type HandshakeResult struct {
	// Signaled indicates whether the bridge signaled ready.
	Signaled bool `json:"signaled"`

	// TimedOut indicates whether the handshake wait timed out.
	TimedOut bool `json:"timed_out"`

	// DurationMs is how long the handshake took.
	DurationMs int64 `json:"duration_ms"`

	// Error contains any handshake error message.
	Error string `json:"error,omitempty"`
}

// ArtifactPaths contains paths to generated test artifacts.
type ArtifactPaths struct {
	// Screenshot is the path to the PNG screenshot.
	Screenshot string `json:"screenshot,omitempty"`

	// Console is the path to the console log JSON.
	Console string `json:"console,omitempty"`

	// Network is the path to the network failures JSON.
	Network string `json:"network,omitempty"`

	// HTML is the path to the DOM snapshot.
	HTML string `json:"html,omitempty"`

	// Raw is the path to the raw response JSON.
	Raw string `json:"raw,omitempty"`
}

// BundleStatus describes UI bundle freshness.
type BundleStatus struct {
	// Fresh indicates whether the bundle is up-to-date.
	Fresh bool `json:"fresh"`

	// Reason describes why the bundle is stale (if applicable).
	Reason string `json:"reason,omitempty"`

	// Config contains the bundle check configuration.
	Config json.RawMessage `json:"config,omitempty"`
}

// BridgeStatus describes iframe-bridge dependency status.
type BridgeStatus struct {
	// DependencyPresent indicates whether @vrooli/iframe-bridge is installed.
	DependencyPresent bool `json:"dependency_present"`

	// Version is the installed version of iframe-bridge.
	Version string `json:"version,omitempty"`

	// Details provides additional information.
	Details string `json:"details,omitempty"`
}

// StorageShimEntry describes a storage API patching result.
type StorageShimEntry struct {
	// Prop is the storage property name (e.g., "localStorage").
	Prop string `json:"prop"`

	// Patched indicates whether the property was successfully patched.
	Patched bool `json:"patched"`

	// Reason describes why patching failed (if applicable).
	Reason string `json:"reason,omitempty"`
}

// NewResult creates a new Result with the given status and message.
func NewResult(scenario string, status Status, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    status,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// WithDuration sets the duration on the result.
func (r *Result) WithDuration(d time.Duration) *Result {
	r.DurationMs = d.Milliseconds()
	return r
}

// WithHandshake sets the handshake result.
func (r *Result) WithHandshake(h HandshakeResult) *Result {
	r.Handshake = h
	return r
}

// WithArtifacts sets the artifact paths.
func (r *Result) WithArtifacts(a ArtifactPaths) *Result {
	r.Artifacts = a
	return r
}

// Passed creates a successful result.
func Passed(scenario, uiURL string, duration time.Duration) *Result {
	return &Result{
		Scenario:   scenario,
		Status:     StatusPassed,
		Message:    "UI loaded successfully",
		Timestamp:  time.Now().UTC(),
		DurationMs: duration.Milliseconds(),
		UIURL:      uiURL,
	}
}

// Failed creates a failed result.
func Failed(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusFailed,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// Skipped creates a skipped result.
func Skipped(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusSkipped,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// Blocked creates a blocked result.
func Blocked(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusBlocked,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}
