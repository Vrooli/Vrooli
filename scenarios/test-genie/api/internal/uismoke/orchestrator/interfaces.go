package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DefaultTimeout is the overall smoke test timeout.
const DefaultTimeout = 90 * time.Second

// DefaultHandshakeTimeout is the maximum wait for iframe-bridge handshake.
const DefaultHandshakeTimeout = 15 * time.Second

// DefaultViewportWidth is the default browser viewport width.
const DefaultViewportWidth = 1280

// DefaultViewportHeight is the default browser viewport height.
const DefaultViewportHeight = 720

// Config holds configuration for a UI smoke test run.
type Config struct {
	// ScenarioName is the name of the scenario being tested.
	ScenarioName string

	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// BrowserlessURL is the base URL for the Browserless service.
	BrowserlessURL string

	// Timeout is the overall timeout for the smoke test.
	Timeout time.Duration

	// HandshakeTimeout is the maximum time to wait for iframe-bridge handshake.
	HandshakeTimeout time.Duration

	// Viewport defines the browser viewport dimensions.
	Viewport Viewport

	// UIURL is the URL where the scenario's UI is served.
	// If empty, it will be constructed from discovered port information.
	UIURL string

	// UIPort is the port where the scenario's UI is served.
	// Used when UIURL is not explicitly set.
	UIPort int
}

// Viewport defines browser viewport dimensions.
type Viewport struct {
	Width  int
	Height int
}

// DefaultViewport returns the standard viewport for smoke tests.
func DefaultViewport() Viewport {
	return Viewport{
		Width:  DefaultViewportWidth,
		Height: DefaultViewportHeight,
	}
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:          DefaultTimeout,
		HandshakeTimeout: DefaultHandshakeTimeout,
		Viewport:         DefaultViewport(),
	}
}

// Validate checks that the configuration is complete and valid.
func (c *Config) Validate() error {
	if c.ScenarioName == "" {
		return fmt.Errorf("scenario name is required")
	}
	if c.ScenarioDir == "" {
		return fmt.Errorf("scenario directory is required")
	}
	if c.BrowserlessURL == "" {
		return fmt.Errorf("browserless URL is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.HandshakeTimeout <= 0 {
		return fmt.Errorf("handshake timeout must be positive")
	}
	if c.Viewport.Width <= 0 || c.Viewport.Height <= 0 {
		return fmt.Errorf("viewport dimensions must be positive")
	}
	return nil
}

// TimeoutMs returns the timeout in milliseconds.
func (c *Config) TimeoutMs() int64 {
	return c.Timeout.Milliseconds()
}

// HandshakeTimeoutMs returns the handshake timeout in milliseconds.
func (c *Config) HandshakeTimeoutMs() int64 {
	return c.HandshakeTimeout.Milliseconds()
}

// ResolveUIURL returns the UI URL to test.
// If UIURL is set, it returns that. Otherwise constructs from UIPort.
func (c *Config) ResolveUIURL() string {
	if c.UIURL != "" {
		return c.UIURL
	}
	if c.UIPort > 0 {
		return fmt.Sprintf("http://localhost:%d", c.UIPort)
	}
	return ""
}

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

// PreflightChecker validates preconditions before running the smoke test.
type PreflightChecker interface {
	// CheckBrowserless verifies the Browserless service is available and healthy.
	CheckBrowserless(ctx context.Context) error

	// CheckBundleFreshness verifies the UI bundle is up-to-date.
	// Returns nil status if bundle check is not applicable (no UI).
	CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error)

	// CheckIframeBridge verifies @vrooli/iframe-bridge is installed.
	CheckIframeBridge(ctx context.Context, scenarioDir string) (*BridgeStatus, error)

	// CheckUIPort discovers and returns the UI port for the scenario.
	// Returns 0 if no UI port is detected.
	CheckUIPort(ctx context.Context, scenarioName string) (int, error)

	// CheckUIPortDefined checks if the scenario's service.json defines a UI port.
	CheckUIPortDefined(scenarioDir string) (*UIPortDefinition, error)

	// CheckUIDirectory returns true if the scenario has a UI directory.
	CheckUIDirectory(scenarioDir string) bool
}

// UIPortDefinition describes whether a scenario defines a UI port in service.json.
type UIPortDefinition struct {
	Defined     bool   // True if service.json defines a UI port
	EnvVar      string // The environment variable name (e.g., "UI_PORT")
	Description string // Description from service.json
}

// BrowserClient executes JavaScript in a browser context via Browserless.
type BrowserClient interface {
	// ExecuteFunction sends a JavaScript function to the Browserless /function endpoint.
	// Returns the raw JSON response from the browser execution.
	ExecuteFunction(ctx context.Context, payload string) (*BrowserResponse, error)

	// Health checks if the Browserless service is reachable.
	Health(ctx context.Context) error
}

// BrowserResponse holds the parsed response from a Browserless function execution.
type BrowserResponse struct {
	// Success indicates whether the browser execution completed without error.
	Success bool `json:"success"`

	// Console contains console log entries from the page.
	Console []ConsoleEntry `json:"console"`

	// Network contains failed network requests.
	Network []NetworkEntry `json:"network"`

	// PageErrors contains JavaScript errors from the page.
	PageErrors []PageError `json:"pageErrors"`

	// Handshake contains iframe-bridge handshake result.
	Handshake HandshakeRaw `json:"handshake"`

	// StorageShim contains storage API patching results.
	StorageShim []StorageShimEntry `json:"storageShim"`

	// Screenshot is the base64-encoded PNG screenshot.
	Screenshot string `json:"screenshot"`

	// HTML is the DOM snapshot.
	HTML string `json:"html"`

	// Title is the page title.
	Title string `json:"title"`

	// URL is the final page URL.
	URL string `json:"url"`

	// Timings contains performance timing data.
	Timings Timings `json:"timings"`

	// Error contains any execution error message.
	Error string `json:"error"`

	// Stack contains the error stack trace if applicable.
	Stack string `json:"stack"`

	// Raw contains the complete raw response for persistence.
	Raw json.RawMessage `json:"-"`
}

// ConsoleEntry represents a console log message.
type ConsoleEntry struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// NetworkEntry represents a failed network request.
type NetworkEntry struct {
	URL          string `json:"url"`
	Method       string `json:"method"`
	ResourceType string `json:"resourceType"`
	Status       *int   `json:"status"`
	ErrorText    string `json:"errorText"`
	Timestamp    string `json:"timestamp"`
}

// PageError represents a JavaScript error on the page.
type PageError struct {
	Message   string `json:"message"`
	Stack     string `json:"stack"`
	Timestamp string `json:"timestamp"`
}

// HandshakeRaw contains raw handshake data from the browser.
type HandshakeRaw struct {
	Signaled   bool   `json:"signaled"`
	TimedOut   bool   `json:"timedOut"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error"`
}

// Timings contains performance timing data.
type Timings struct {
	GotoMs  int64 `json:"gotoMs"`
	TotalMs int64 `json:"totalMs"`
}

// ArtifactWriter persists test artifacts to the filesystem.
type ArtifactWriter interface {
	// WriteAll writes all artifacts and returns their paths.
	WriteAll(ctx context.Context, scenarioDir, scenarioName string, response *BrowserResponse) (*ArtifactPaths, error)

	// WriteResultJSON writes the final result JSON.
	WriteResultJSON(ctx context.Context, scenarioDir, scenarioName string, result interface{}) error

	// WriteReadme generates a README.md summarizing the test results.
	WriteReadme(ctx context.Context, scenarioDir, scenarioName string, result *Result) error
}

// HandshakeDetector evaluates handshake results from the browser.
type HandshakeDetector interface {
	// Evaluate interprets raw handshake data into a structured result.
	Evaluate(raw *HandshakeRaw) HandshakeResult
}

// PortDiscoverer discovers the UI port for a scenario.
type PortDiscoverer interface {
	// DiscoverUIPort returns the port where the scenario's UI is served.
	// Returns 0 if no UI port is detected.
	DiscoverUIPort(ctx context.Context, scenarioName string) (int, error)
}

// PayloadGenerator generates JavaScript payloads for browser execution.
type PayloadGenerator interface {
	Generate(uiURL string, timeout, handshakeTimeout interface{}) string
}
