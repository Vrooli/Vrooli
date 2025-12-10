package codesigning

import (
	"os"
	"time"
)

// Certificate source constants.
const (
	CertSourceFile         = "file"
	CertSourceStore        = "store"
	CertSourceAzureKeyVault = "azure_keyvault"
	CertSourceAWSKMS       = "aws_kms"
)

// Sign algorithm constants.
const (
	SignAlgorithmSHA256 = "sha256"
	SignAlgorithmSHA384 = "sha384"
	SignAlgorithmSHA512 = "sha512"
)

// Platform constants.
const (
	PlatformWindows = "windows"
	PlatformMacOS   = "macos"
	PlatformLinux   = "linux"
)

// Default timestamp servers.
const (
	DefaultTimestampServerDigiCert = "http://timestamp.digicert.com"
	DefaultTimestampServerSectigo  = "http://timestamp.sectigo.com"
	DefaultTimestampServerGlobalSign = "http://timestamp.globalsign.com/tsa/r6advanced1"
)

// Certificate expiration warning thresholds.
const (
	CertExpiryWarningDays  = 30
	CertExpiryCriticalDays = 7
)

// ValidCertificateSources contains all valid certificate source values.
var ValidCertificateSources = []string{
	CertSourceFile,
	CertSourceStore,
	CertSourceAzureKeyVault,
	CertSourceAWSKMS,
}

// ValidSignAlgorithms contains all valid signing algorithm values.
var ValidSignAlgorithms = []string{
	SignAlgorithmSHA256,
	SignAlgorithmSHA384,
	SignAlgorithmSHA512,
}

// ValidPlatforms contains all valid platform values.
var ValidPlatforms = []string{
	PlatformWindows,
	PlatformMacOS,
	PlatformLinux,
}

// DefaultSigningConfig returns a new SigningConfig with sensible defaults.
func DefaultSigningConfig() *SigningConfig {
	return &SigningConfig{
		Enabled: false,
		Windows: nil,
		MacOS:   nil,
		Linux:   nil,
	}
}

// DefaultWindowsConfig returns a WindowsSigningConfig with sensible defaults.
func DefaultWindowsConfig() *WindowsSigningConfig {
	return &WindowsSigningConfig{
		CertificateSource: CertSourceFile,
		TimestampServer:   DefaultTimestampServerDigiCert,
		SignAlgorithm:     SignAlgorithmSHA256,
		DualSign:          false,
	}
}

// DefaultMacOSConfig returns a MacOSSigningConfig with sensible defaults.
func DefaultMacOSConfig() *MacOSSigningConfig {
	return &MacOSSigningConfig{
		HardenedRuntime: true,
		Notarize:        true,
	}
}

// DefaultLinuxConfig returns a LinuxSigningConfig with sensible defaults.
func DefaultLinuxConfig() *LinuxSigningConfig {
	return &LinuxSigningConfig{}
}

// NewValidationResult creates an empty validation result.
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:     true,
		Platforms: make(map[string]PlatformValidation),
		Errors:    []ValidationError{},
		Warnings:  []ValidationWarning{},
	}
}

// AddError adds a validation error and marks the result as invalid.
func (r *ValidationResult) AddError(err ValidationError) {
	r.Valid = false
	r.Errors = append(r.Errors, err)
}

// AddWarning adds a validation warning without affecting validity.
func (r *ValidationResult) AddWarning(warn ValidationWarning) {
	r.Warnings = append(r.Warnings, warn)
}

// realTimeProvider implements TimeProvider using the system clock.
type realTimeProvider struct{}

// Now returns the current time.
func (r *realTimeProvider) Now() time.Time {
	return time.Now()
}

// NewRealTimeProvider creates a TimeProvider using the system clock.
func NewRealTimeProvider() TimeProvider {
	return &realTimeProvider{}
}

// realEnvironmentReader implements EnvironmentReader using os package.
type realEnvironmentReader struct{}

// GetEnv retrieves an environment variable value.
func (r *realEnvironmentReader) GetEnv(key string) string {
	return os.Getenv(key)
}

// LookupEnv retrieves an environment variable and reports if it exists.
func (r *realEnvironmentReader) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// NewRealEnvironmentReader creates an EnvironmentReader using the real environment.
func NewRealEnvironmentReader() EnvironmentReader {
	return &realEnvironmentReader{}
}

// realFileSystem implements FileSystem using the os package.
type realFileSystem struct{}

// Exists checks if a file or directory exists.
func (r *realFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads the entire file content.
func (r *realFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file with the given permissions.
func (r *realFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat returns file info.
func (r *realFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// MkdirAll creates a directory and all parent directories.
func (r *realFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// NewRealFileSystem creates a FileSystem using the real filesystem.
func NewRealFileSystem() FileSystem {
	return &realFileSystem{}
}

// IsValidCertificateSource checks if a certificate source is valid.
func IsValidCertificateSource(source string) bool {
	for _, s := range ValidCertificateSources {
		if s == source {
			return true
		}
	}
	return false
}

// IsValidSignAlgorithm checks if a signing algorithm is valid.
func IsValidSignAlgorithm(algo string) bool {
	for _, a := range ValidSignAlgorithms {
		if a == algo {
			return true
		}
	}
	return false
}

// IsValidPlatform checks if a platform is valid.
func IsValidPlatform(platform string) bool {
	for _, p := range ValidPlatforms {
		if p == platform {
			return true
		}
	}
	return false
}

// ResolveEnvValue resolves an environment variable reference.
// If the value starts with "$", it looks up the environment variable.
// Otherwise, returns the value as-is.
func ResolveEnvValue(value string, env EnvironmentReader) string {
	if len(value) > 0 && value[0] == '$' {
		return env.GetEnv(value[1:])
	}
	return value
}

// CalculateDaysToExpiry calculates days until certificate expiration.
func CalculateDaysToExpiry(notAfter time.Time, now time.Time) int {
	duration := notAfter.Sub(now)
	return int(duration.Hours() / 24)
}

// IsCertificateExpired checks if a certificate is expired.
func IsCertificateExpired(notAfter time.Time, now time.Time) bool {
	return now.After(notAfter)
}

// IsCertificateExpiringWarning checks if a certificate is expiring within warning threshold.
func IsCertificateExpiringWarning(notAfter time.Time, now time.Time) bool {
	days := CalculateDaysToExpiry(notAfter, now)
	return days > 0 && days <= CertExpiryWarningDays
}

// IsCertificateExpiringCritical checks if a certificate is expiring within critical threshold.
func IsCertificateExpiringCritical(notAfter time.Time, now time.Time) bool {
	days := CalculateDaysToExpiry(notAfter, now)
	return days > 0 && days <= CertExpiryCriticalDays
}
