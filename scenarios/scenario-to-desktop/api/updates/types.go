package updates

import "time"

// ManifestRequest contains the inputs for generating update manifests.
type ManifestRequest struct {
	// Version is the application version (e.g., "1.2.3").
	Version string

	// Channel is the update channel (e.g., "stable", "beta", "dev").
	Channel string

	// Artifacts maps platform names to artifact file paths.
	// Platform names: "win", "mac", "linux"
	// Example: {"win": "/path/to/app-1.2.3.exe", "mac": "/path/to/app-1.2.3.dmg"}
	Artifacts map[string]string

	// BaseURL is the URL prefix where update files will be hosted.
	// For generic provider, this is the update server URL.
	// Example: "https://updates.example.com/my-app/stable"
	BaseURL string

	// OutputDir is the directory where manifest files should be written.
	// If empty, manifests are returned in-memory only.
	OutputDir string

	// ReleaseDate is the timestamp for the release.
	// Defaults to time.Now() if not specified.
	ReleaseDate time.Time
}

// ManifestResult contains the generated manifests.
type ManifestResult struct {
	// Manifests maps filename to content.
	// Keys are typically "latest.yml", "latest-mac.yml", "latest-linux.yml".
	Manifests map[string][]byte

	// ManifestPaths maps filename to output path (if OutputDir was specified).
	ManifestPaths map[string]string

	// Warnings contains non-fatal issues encountered during generation.
	Warnings []ValidationWarning
}

// ElectronManifest represents the electron-updater YAML manifest format.
// This is the structure for latest.yml, latest-mac.yml, etc.
// Reference: https://www.electron.build/auto-update#file-urls
type ElectronManifest struct {
	// Version is the application version.
	Version string `yaml:"version"`

	// Path is the relative path or filename of the update file.
	// For single-file releases, this is the filename.
	Path string `yaml:"path,omitempty"`

	// Files lists all update files with their metadata.
	// Required for multi-file releases or when SHA-512 hashes are provided.
	Files []ElectronManifestFile `yaml:"files,omitempty"`

	// Sha512 is the SHA-512 hash of the update file (base64 encoded).
	// Deprecated in favor of Files[].sha512, but still supported.
	Sha512 string `yaml:"sha512,omitempty"`

	// ReleaseDate is the ISO 8601 formatted release date.
	ReleaseDate string `yaml:"releaseDate,omitempty"`

	// StagingPercentage is the percentage of users who should receive this update.
	// Used for gradual rollouts. Default is 100 (all users).
	StagingPercentage float64 `yaml:"stagingPercentage,omitempty"`
}

// ElectronManifestFile represents a single file in the update manifest.
type ElectronManifestFile struct {
	// URL is the download URL for the file.
	// Can be relative (resolved against baseUrl) or absolute.
	URL string `yaml:"url"`

	// Sha512 is the SHA-512 hash of the file (base64 encoded).
	Sha512 string `yaml:"sha512"`

	// Size is the file size in bytes.
	Size int64 `yaml:"size"`
}

// ValidationWarning represents a non-fatal configuration issue.
type ValidationWarning struct {
	// Code is a unique identifier for the warning type.
	// Examples: "MISSING_URL", "EMPTY_ARTIFACTS", "INVALID_PLATFORM"
	Code string

	// Message is a human-readable description of the warning.
	Message string

	// Field is the configuration field that caused the warning.
	// Optional; may be empty for general warnings.
	Field string
}

// PlatformManifestFilename returns the manifest filename for a platform.
// Windows: latest.yml
// macOS: latest-mac.yml
// Linux: latest-linux.yml
func PlatformManifestFilename(platform string) string {
	switch platform {
	case "win", "windows":
		return "latest.yml"
	case "mac", "macos", "darwin":
		return "latest-mac.yml"
	case "linux":
		return "latest-linux.yml"
	default:
		// Unknown platforms get a custom filename
		return "latest-" + platform + ".yml"
	}
}

// NormalizePlatform converts platform aliases to canonical names.
func NormalizePlatform(platform string) string {
	switch platform {
	case "windows", "win32":
		return "win"
	case "macos", "darwin", "osx":
		return "mac"
	default:
		return platform
	}
}
