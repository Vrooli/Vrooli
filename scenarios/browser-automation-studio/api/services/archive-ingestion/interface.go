package archiveingestion

import (
	"context"
)

// IngestionServiceInterface defines the interface for archive import operations
// This interface allows for easier testing by enabling mock implementations
type IngestionServiceInterface interface {
	// ImportArchive imports a recording archive (ZIP file) and creates:
	// - A project (if not exists)
	// - A workflow (if not exists)
	// - An execution with steps and artifacts
	// Returns the import result with references to created entities
	ImportArchive(ctx context.Context, archivePath string, opts IngestionOptions) (*IngestionResult, error)
}

// RecordingServiceInterface is an alias for backward compatibility.
type RecordingServiceInterface = IngestionServiceInterface
