package codesigning

import (
	"context"
	"os"
	"time"
)

// FileSystem abstracts all file operations for testing.
type FileSystem interface {
	// Exists checks if a file or directory exists.
	Exists(path string) bool

	// ReadFile reads the entire file content.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to a file with the given permissions.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// Stat returns file info.
	Stat(path string) (os.FileInfo, error)

	// MkdirAll creates a directory and all parent directories.
	MkdirAll(path string, perm os.FileMode) error
}

// CommandRunner abstracts external command execution for testing.
type CommandRunner interface {
	// Run executes a command and returns stdout, stderr, and any error.
	Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)

	// LookPath searches for an executable in the PATH.
	LookPath(name string) (string, error)
}

// CertificateParser abstracts certificate reading for testing.
type CertificateParser interface {
	// ParsePKCS12 parses a PKCS#12 (.pfx/.p12) certificate file.
	ParsePKCS12(data []byte, password string) (*CertificateInfo, error)

	// ParsePEM parses a PEM-encoded certificate.
	ParsePEM(data []byte) (*CertificateInfo, error)

	// ValidateChain validates the certificate chain.
	ValidateChain(cert *CertificateInfo) error
}

// TimeProvider abstracts time for deterministic testing.
type TimeProvider interface {
	// Now returns the current time.
	Now() time.Time
}

// EnvironmentReader abstracts environment variable access for testing.
type EnvironmentReader interface {
	// GetEnv retrieves an environment variable value.
	GetEnv(key string) string

	// LookupEnv retrieves an environment variable and reports if it exists.
	LookupEnv(key string) (string, bool)
}

// Repository persists signing configurations.
type Repository interface {
	// Get retrieves the signing config for a profile.
	Get(ctx context.Context, profileID string) (*SigningConfig, error)

	// Save stores or updates the signing config for a profile.
	Save(ctx context.Context, profileID string, config *SigningConfig) error

	// Delete removes the signing config for a profile.
	Delete(ctx context.Context, profileID string) error

	// GetForPlatform retrieves config for a specific platform only.
	GetForPlatform(ctx context.Context, profileID string, platform string) (interface{}, error)

	// SaveForPlatform updates only a specific platform's config.
	SaveForPlatform(ctx context.Context, profileID string, platform string, config interface{}) error

	// DeleteForPlatform removes the signing config for a specific platform.
	DeleteForPlatform(ctx context.Context, profileID string, platform string) error
}

// Validator validates signing configurations structurally.
// This interface is defined in the main package to avoid import cycles.
// The implementation is in the validation subpackage.
type Validator interface {
	// ValidateConfig checks structural validity of a SigningConfig.
	ValidateConfig(config *SigningConfig) *ValidationResult

	// ValidateForPlatform checks config is valid for a specific target platform.
	ValidateForPlatform(config *SigningConfig, platform string) *ValidationResult
}

// PrerequisiteChecker verifies external signing prerequisites.
// This interface is defined in the main package to avoid import cycles.
// The implementation is in the validation subpackage.
type PrerequisiteChecker interface {
	// CheckPrerequisites validates tools and certificates are available.
	CheckPrerequisites(ctx context.Context, config *SigningConfig) *ValidationResult

	// CheckPlatformPrerequisites checks prerequisites for a specific platform.
	CheckPlatformPrerequisites(ctx context.Context, config *SigningConfig, platform string) *ValidationResult

	// DetectTools returns available signing tools on the current system.
	DetectTools(ctx context.Context) ([]ToolDetectionResult, error)
}
