// Package smoketest provides smoke testing services for desktop applications.
package smoketest

import "time"

// Config holds configuration for smoke test execution.
type Config struct {
	// TimeoutSeconds is the maximum duration for a smoke test.
	TimeoutSeconds int

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
}

// DefaultConfig returns the default smoke test configuration.
func DefaultConfig() Config {
	return Config{
		TimeoutSeconds:      30,
		TelemetryPathMarker: "[Desktop App] Telemetry initialized at ",
		SuccessMarker:       "SMOKE_TEST_RESULT=passed",
		UploadSuccessMarker: "SMOKE_TEST_UPLOAD=ok",
		UploadErrorMarker:   "SMOKE_TEST_UPLOAD=error",
		MaxTelemetryEvents:  500,
		XvfbCommand:         "xvfb-run",
		TelemetryFileName:   "deployment-telemetry.jsonl",
		InitMarker:          "SMOKE_TEST_INIT=started",
		ReadyMarker:         "SMOKE_TEST_READY=true",
		ExitMarker:          "SMOKE_TEST_EXIT=clean",
		MaxOutputBytes:      10 * 1024 * 1024, // 10MB
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
