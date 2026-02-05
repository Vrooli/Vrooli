// Package updates provides auto-update manifest generation for desktop applications.
// It implements the provider pattern to support multiple update distribution methods
// (self-hosted generic servers, GitHub releases, etc.).
//
// The package follows these architectural patterns:
// - Provider interface for different update backends
// - Factory pattern for provider instantiation
// - Option pattern for dependency injection
// - Testing seams for hash calculation and file operations
package updates

import (
	"context"

	"scenario-to-desktop-api/generation"
)

// Provider abstracts update provider behavior for different distribution methods.
// Implementations exist for:
// - generic: Self-hosted update servers (e.g., S3, LPBS, any HTTP server)
// - github: GitHub Releases (electron-builder handles manifest generation)
// - none: Updates disabled
type Provider interface {
	// Name returns the provider identifier (e.g., "generic", "github", "none").
	Name() string

	// Validate checks if the provider configuration is valid.
	// Returns an error if required configuration is missing or invalid.
	Validate() error

	// GetPublishConfig returns the electron-builder publish configuration
	// for the given channel. Returns nil if the provider doesn't use
	// electron-builder's publish feature.
	GetPublishConfig(channel string) (map[string]interface{}, error)

	// GenerateManifest creates update manifest files for the given artifacts.
	// For generic provider, this generates latest.yml, latest-mac.yml, etc.
	// For github provider, this returns nil (electron-builder handles it).
	// Returns nil, nil if manifest generation is not applicable.
	GenerateManifest(ctx context.Context, req *ManifestRequest) (*ManifestResult, error)

	// RequiresManifestUpload returns true if the provider needs manifest files
	// uploaded alongside artifacts. Generic returns true, GitHub returns false.
	RequiresManifestUpload() bool
}

// ProviderFactory creates Provider instances from configuration.
// This is the primary DI seam for testing provider creation logic.
type ProviderFactory interface {
	// Create instantiates a Provider based on the update configuration.
	// Returns validation warnings for non-fatal configuration issues.
	// Defaults to GenericProvider if no provider is specified.
	Create(config *generation.UpdateConfig) (Provider, []ValidationWarning, error)
}

// HashCalculator abstracts file hash computation for testing.
// Production implementation uses crypto/sha512.
type HashCalculator interface {
	// CalculateSHA512 computes the SHA-512 hash of the file at the given path.
	// Returns the hash as a base64-encoded string.
	CalculateSHA512(filePath string) (string, error)
}

// FileStatProvider abstracts file metadata retrieval for testing.
// Production implementation uses os.Stat.
type FileStatProvider interface {
	// Stat returns file information including size.
	Stat(path string) (FileInfo, error)
}

// FileInfo contains file metadata needed for manifest generation.
type FileInfo struct {
	// Size is the file size in bytes.
	Size int64
	// Name is the base name of the file.
	Name string
}

// Logger provides structured logging for update operations.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// ManifestWriter abstracts file writing for manifest output.
// Production implementation uses os.WriteFile.
type ManifestWriter interface {
	// WriteFile writes content to the specified path.
	WriteFile(path string, content []byte) error
}
