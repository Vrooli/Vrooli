package recording

import (
	"context"
)

// RecordingServiceInterface defines the interface for recording import operations
// This interface allows for easier testing by enabling mock implementations
type RecordingServiceInterface interface {
	// ImportArchive imports a recording archive (ZIP file) and creates:
	// - A project (if not exists)
	// - A workflow (if not exists)
	// - An execution with steps and artifacts
	// Returns the import result with references to created entities
	ImportArchive(ctx context.Context, archivePath string, opts RecordingImportOptions) (*RecordingImportResult, error)
}
