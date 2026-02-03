// Package smoketest provides smoke testing services for desktop applications.
package smoketest

import "time"

// Config holds configuration for smoke test execution.
type Config struct {
	// TimeoutSeconds is the maximum duration for a smoke test.
	// This is the default timeout; deployment-mode-specific timeouts override it.
	TimeoutSeconds int

	// BundledModeTimeoutSeconds is the timeout for bundled deployment mode.
	// Bundled mode requires longer timeouts due to runtime startup.
	// If 0, falls back to TimeoutSeconds.
	BundledModeTimeoutSeconds int

	// ExternalServerModeTimeoutSeconds is the timeout for external-server mode.
	// External server mode is typically faster since the server is already running.
	// If 0, falls back to TimeoutSeconds.
	ExternalServerModeTimeoutSeconds int

	// TelemetryPathMarker is the log marker used to extract telemetry file path.
	TelemetryPathMarker string

	// SuccessMarker indicates the smoke test passed.
	SuccessMarker string

	// UploadSuccessMarker indicates telemetry upload succeeded.
	UploadSuccessMarker string

	// UploadErrorMarker indicates telemetry upload failed.
	UploadErrorMarker string

	// MaxTelemetryEvents is the maximum number of telemetry events to read for fallback.
	MaxTelemetryEvents int

	// XvfbCommand is the command used for headless X11 execution.
	XvfbCommand string

	// TelemetryFileName is the name of the telemetry file.
	TelemetryFileName string

	// InitMarker indicates the smoke test mode has started.
	InitMarker string

	// ReadyMarker indicates the app finished initialization.
	ReadyMarker string

	// ExitMarker indicates the app is exiting cleanly.
	ExitMarker string

	// MaxOutputBytes limits the total captured output to prevent memory exhaustion.
	// Default is 10MB.
	MaxOutputBytes int

	// GranularLifecycleMarkers holds markers for detailed bundled runtime tracking.
	GranularLifecycleMarkers GranularLifecycleMarkers
}

// GranularLifecycleMarkers defines markers for detailed lifecycle state tracking.
// These are primarily used for bundled mode to help diagnose where startup fails.
type GranularLifecycleMarkers struct {
	// BundleResolving indicates bundle path resolution started.
	BundleResolving string

	// RuntimeStarting indicates the runtime process is being spawned.
	RuntimeStarting string

	// WaitingForToken indicates waiting for runtime auth token file.
	WaitingForToken string

	// RuntimeHealthz indicates waiting for the /healthz endpoint.
	RuntimeHealthz string

	// RuntimeReadyz indicates waiting for the /readyz endpoint.
	RuntimeReadyz string

	// RuntimePorts indicates querying the /ports endpoint.
	RuntimePorts string
}

// DefaultConfig returns the default smoke test configuration.
func DefaultConfig() Config {
	return Config{
		TimeoutSeconds:                   30,
		BundledModeTimeoutSeconds:        60, // Bundled mode needs longer due to runtime startup
		ExternalServerModeTimeoutSeconds: 30,
		TelemetryPathMarker:              "[Desktop App] Telemetry initialized at ",
		SuccessMarker:                    "SMOKE_TEST_RESULT=passed",
		UploadSuccessMarker:              "SMOKE_TEST_UPLOAD=ok",
		UploadErrorMarker:                "SMOKE_TEST_UPLOAD=error",
		MaxTelemetryEvents:               500,
		XvfbCommand:                      "xvfb-run",
		TelemetryFileName:                "deployment-telemetry.jsonl",
		InitMarker:                       "SMOKE_TEST_INIT=started",
		ReadyMarker:                      "SMOKE_TEST_READY=true",
		ExitMarker:                       "SMOKE_TEST_EXIT=clean",
		MaxOutputBytes:                   10 * 1024 * 1024, // 10MB
		GranularLifecycleMarkers: GranularLifecycleMarkers{
			BundleResolving: "SMOKE_TEST_STAGE=bundle_resolving",
			RuntimeStarting: "SMOKE_TEST_STAGE=runtime_starting",
			WaitingForToken: "SMOKE_TEST_STAGE=waiting_for_token",
			RuntimeHealthz:  "SMOKE_TEST_STAGE=runtime_healthz",
			RuntimeReadyz:   "SMOKE_TEST_STAGE=runtime_readyz",
			RuntimePorts:    "SMOKE_TEST_STAGE=runtime_ports",
		},
	}
}

// Timeout returns the timeout as a time.Duration.
func (c Config) Timeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// TimeoutMS returns the timeout in milliseconds.
func (c Config) TimeoutMS() int {
	return c.TimeoutSeconds * 1000
}

// TimeoutForDeploymentMode returns the appropriate timeout for the given deployment mode.
// Supported modes: "bundled", "external-server", "cloud-api".
// Falls back to default timeout for unknown modes.
func (c Config) TimeoutForDeploymentMode(deploymentMode string) time.Duration {
	switch deploymentMode {
	case "bundled":
		if c.BundledModeTimeoutSeconds > 0 {
			return time.Duration(c.BundledModeTimeoutSeconds) * time.Second
		}
	case "external-server", "cloud-api":
		if c.ExternalServerModeTimeoutSeconds > 0 {
			return time.Duration(c.ExternalServerModeTimeoutSeconds) * time.Second
		}
	}
	return c.Timeout()
}

// TimeoutMSForDeploymentMode returns the timeout in milliseconds for the given deployment mode.
func (c Config) TimeoutMSForDeploymentMode(deploymentMode string) int {
	return int(c.TimeoutForDeploymentMode(deploymentMode).Milliseconds())
}
