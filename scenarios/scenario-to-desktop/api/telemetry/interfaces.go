// Package telemetry provides deployment telemetry ingestion and analysis.
package telemetry

import (
	"context"
)

// Service abstracts telemetry operations for testing.
type Service interface {
	// IngestEvents ingests telemetry events for a scenario.
	IngestEvents(ctx context.Context, scenario, deploymentMode, source string, events []map[string]interface{}) (filePath string, ingested int, err error)

	// GetSummary returns a summary of telemetry for a scenario.
	GetSummary(ctx context.Context, scenario string) (*SummaryResult, error)

	// GetInsights analyzes telemetry and returns insights.
	GetInsights(ctx context.Context, scenario string) (*InsightsResult, error)

	// GetTail returns the last N telemetry entries.
	GetTail(ctx context.Context, scenario string, limit int) (*TailResult, error)

	// GetFilePath returns the path to the telemetry file for a scenario.
	GetFilePath(scenario string) string

	// Delete removes all telemetry for a scenario.
	Delete(ctx context.Context, scenario string) error
}

// FileSystem abstracts file operations for testing.
type FileSystem interface {
	// Stat returns file info.
	Stat(path string) (FileInfo, error)

	// ReadFile reads a file.
	ReadFile(path string) ([]byte, error)

	// OpenFile opens a file for reading.
	OpenFile(path string) (ReadCloser, error)

	// OpenFileAppend opens a file for appending.
	OpenFileAppend(path string) (WriteCloser, error)

	// Remove removes a file.
	Remove(path string) error

	// MkdirAll creates directories.
	MkdirAll(path string) error
}

// FileInfo abstracts os.FileInfo for testing.
type FileInfo interface {
	Size() int64
	IsDir() bool
}

// ReadCloser abstracts io.ReadCloser for testing.
type ReadCloser interface {
	Read(p []byte) (n int, err error)
	Close() error
}

// WriteCloser abstracts io.WriteCloser for testing.
type WriteCloser interface {
	Write(p []byte) (n int, err error)
	Close() error
}

// TimeProvider abstracts time for testing.
type TimeProvider interface {
	Now() string // Returns RFC3339 formatted time
}

// PathResolver resolves telemetry file paths.
type PathResolver interface {
	// TelemetryDir returns the telemetry directory path.
	TelemetryDir() string

	// ScenarioFilePath returns the path to a scenario's telemetry file.
	ScenarioFilePath(scenario string) string
}
