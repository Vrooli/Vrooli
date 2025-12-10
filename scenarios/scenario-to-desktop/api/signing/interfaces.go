package signing

import (
	"context"
	"os"
	"time"

	"scenario-to-desktop-api/signing/types"
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

	// Remove removes a file or empty directory.
	Remove(path string) error
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
	ParsePKCS12(data []byte, password string) (*types.CertificateInfo, error)

	// ParsePEM parses a PEM-encoded certificate.
	ParsePEM(data []byte) (*types.CertificateInfo, error)

	// ValidateChain validates the certificate chain.
	ValidateChain(cert *types.CertificateInfo) error
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
// Unlike deployment-manager's SQL-based storage, this uses file-based storage
// with signing.json stored per-scenario in the scenario directory.
type Repository interface {
	// Get retrieves the signing config for a scenario.
	Get(ctx context.Context, scenario string) (*types.SigningConfig, error)

	// Save stores or updates the signing config for a scenario.
	Save(ctx context.Context, scenario string, config *types.SigningConfig) error

	// Delete removes the signing config for a scenario.
	Delete(ctx context.Context, scenario string) error

	// Exists checks if a signing config exists for a scenario.
	Exists(ctx context.Context, scenario string) (bool, error)

	// GetPath returns the path to the signing.json file for a scenario.
	GetPath(scenario string) string

	// GetForPlatform retrieves config for a specific platform only.
	GetForPlatform(ctx context.Context, scenario string, platform string) (interface{}, error)

	// SaveForPlatform updates only a specific platform's config.
	SaveForPlatform(ctx context.Context, scenario string, platform string, config interface{}) error

	// DeleteForPlatform removes config for a specific platform.
	DeleteForPlatform(ctx context.Context, scenario string, platform string) error
}

// Validator validates signing configurations structurally.
type Validator interface {
	// ValidateConfig checks structural validity of a SigningConfig.
	ValidateConfig(config *types.SigningConfig) *types.ValidationResult

	// ValidateForPlatform checks config is valid for a specific target platform.
	ValidateForPlatform(config *types.SigningConfig, platform string) *types.ValidationResult
}

// PrerequisiteChecker verifies external signing prerequisites.
type PrerequisiteChecker interface {
	// CheckPrerequisites validates tools and certificates are available.
	CheckPrerequisites(ctx context.Context, config *types.SigningConfig) *types.ValidationResult

	// CheckPlatformPrerequisites checks prerequisites for a specific platform.
	CheckPlatformPrerequisites(ctx context.Context, config *types.SigningConfig, platform string) *types.ValidationResult

	// DetectTools returns available signing tools on the current system.
	DetectTools(ctx context.Context) ([]types.ToolDetectionResult, error)
}

// CertificateDiscoverer finds available signing identities.
type CertificateDiscoverer interface {
	// DiscoverCertificates discovers available certificates/identities for a platform.
	DiscoverCertificates(ctx context.Context, platform string) ([]types.DiscoveredCertificate, error)
}

// ConfigGenerator creates build tool configs from signing settings.
type ConfigGenerator interface {
	// GenerateElectronBuilder creates electron-builder signing config from a SigningConfig.
	GenerateElectronBuilder(config *types.SigningConfig) (*types.ElectronBuilderSigningConfig, error)

	// GenerateEntitlements creates macOS entitlements.plist content.
	GenerateEntitlements(config *types.MacOSSigningConfig, capabilities []string) ([]byte, error)

	// GenerateNotarizeScript creates the afterSign JavaScript for electron-builder.
	GenerateNotarizeScript(config *types.MacOSSigningConfig) ([]byte, error)

	// GenerateAll creates all signing-related files and returns them as a map.
	GenerateAll(config *types.SigningConfig) (map[string][]byte, error)
}

// ScenarioLocator finds scenario directories.
type ScenarioLocator interface {
	// GetScenarioPath returns the absolute path to a scenario directory.
	GetScenarioPath(scenario string) (string, error)

	// ListScenarios returns all available scenario names.
	ListScenarios() ([]string, error)
}
