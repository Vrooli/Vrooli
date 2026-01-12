// Package smoketest provides smoke testing services for desktop applications.
// This domain handles running smoke tests on built applications to verify
// they start correctly and can report telemetry.
package smoketest

import "context"

// Service orchestrates smoke test operations.
type Service interface {
	// PerformSmokeTest runs a smoke test on a built application.
	PerformSmokeTest(ctx context.Context, smokeTestID, scenarioName, artifactPath, platform string)

	// CurrentPlatform returns the current platform identifier.
	CurrentPlatform() string
}

// Store manages smoke test status tracking.
type Store interface {
	// Save inserts or replaces a smoke test status.
	Save(status *Status)

	// Get returns the status for the given smoke test ID if it exists.
	Get(id string) (*Status, bool)

	// Update executes fn while holding a write lock on the requested smoke test.
	// It returns false when the smoke test ID is unknown.
	Update(id string, fn func(status *Status)) bool
}

// CancelManager manages cancellation of running smoke tests.
type CancelManager interface {
	// SetCancel registers a cancellation function for a smoke test.
	SetCancel(id string, cancel context.CancelFunc)

	// TakeCancel retrieves and removes the cancellation function for a smoke test.
	TakeCancel(id string) context.CancelFunc

	// Clear removes the cancellation function without calling it.
	Clear(id string)
}

// PackageFinder locates built packages in dist directories.
type PackageFinder interface {
	// FindBuiltPackage finds the built package file for a specific platform.
	FindBuiltPackage(distPath, platform string) (string, error)
}

// TelemetryIngestor ingests telemetry events from smoke tests.
type TelemetryIngestor interface {
	// IngestEvents ingests telemetry events from a smoke test.
	IngestEvents(scenarioName, instanceID, source string, events []map[string]interface{}) (string, int, error)
}

// Logger provides structured logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}
