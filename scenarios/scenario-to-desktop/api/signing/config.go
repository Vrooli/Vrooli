package signing

import (
	"os"
	"time"

	"scenario-to-desktop-api/signing/types"
)

// Signing config filename
const SigningConfigFilename = "signing.json"

// Certificate expiration thresholds (in days)
const (
	CertExpiryWarningDays  = 60 // Warn if cert expires in less than 60 days
	CertExpiryCriticalDays = 14 // Critical warning if expires in less than 14 days
)

// NewSigningConfig creates a new signing config with default values.
func NewSigningConfig() *types.SigningConfig {
	return &types.SigningConfig{
		SchemaVersion: types.SchemaVersion,
		Enabled:       false,
	}
}

// NewDefaultWindowsConfig creates a Windows config with sensible defaults.
func NewDefaultWindowsConfig() *types.WindowsSigningConfig {
	return &types.WindowsSigningConfig{
		CertificateSource: types.CertSourceFile,
		TimestampServer:   types.DefaultTimestampServerDigiCert,
		SignAlgorithm:     types.SignAlgorithmSHA256,
		DualSign:          false,
	}
}

// NewDefaultMacOSConfig creates a macOS config with sensible defaults.
func NewDefaultMacOSConfig() *types.MacOSSigningConfig {
	return &types.MacOSSigningConfig{
		HardenedRuntime:  true, // Required for notarization
		GatekeeperAssess: true,
		Notarize:         false, // User must explicitly enable
	}
}

// NewDefaultLinuxConfig creates a Linux config with sensible defaults.
func NewDefaultLinuxConfig() *types.LinuxSigningConfig {
	return &types.LinuxSigningConfig{}
}

// NewValidationResult creates an empty validation result.
func NewValidationResult() *types.ValidationResult {
	return &types.ValidationResult{
		Valid:     true,
		Platforms: make(map[string]types.PlatformValidation),
		Errors:    make([]types.ValidationError, 0),
		Warnings:  make([]types.ValidationWarning, 0),
	}
}

// IsCertificateExpired checks if a certificate is expired.
func IsCertificateExpired(notAfter, now time.Time) bool {
	return now.After(notAfter)
}

// IsCertificateExpiringWarning checks if a certificate expires within warning threshold.
func IsCertificateExpiringWarning(notAfter, now time.Time) bool {
	warningDate := now.AddDate(0, 0, CertExpiryWarningDays)
	return notAfter.Before(warningDate)
}

// IsCertificateExpiringCritical checks if a certificate expires within critical threshold.
func IsCertificateExpiringCritical(notAfter, now time.Time) bool {
	criticalDate := now.AddDate(0, 0, CertExpiryCriticalDays)
	return notAfter.Before(criticalDate)
}

// CalculateDaysToExpiry calculates days until certificate expiration.
func CalculateDaysToExpiry(notAfter, now time.Time) int {
	duration := notAfter.Sub(now)
	return int(duration.Hours() / 24)
}

// --- Real implementations of interfaces ---

// RealFileSystem implements FileSystem using the actual filesystem.
type RealFileSystem struct{}

// NewRealFileSystem creates a new real filesystem.
func NewRealFileSystem() FileSystem {
	return &RealFileSystem{}
}

// Exists checks if a file or directory exists.
func (f *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads the entire file content.
func (f *RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file with the given permissions.
func (f *RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat returns file info.
func (f *RealFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// MkdirAll creates a directory and all parent directories.
func (f *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove removes a file or empty directory.
func (f *RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// RealEnvironmentReader implements EnvironmentReader using actual environment variables.
type RealEnvironmentReader struct{}

// NewRealEnvironmentReader creates a new real environment reader.
func NewRealEnvironmentReader() EnvironmentReader {
	return &RealEnvironmentReader{}
}

// GetEnv retrieves an environment variable value.
func (e *RealEnvironmentReader) GetEnv(key string) string {
	return os.Getenv(key)
}

// LookupEnv retrieves an environment variable and reports if it exists.
func (e *RealEnvironmentReader) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// RealTimeProvider implements TimeProvider using actual time.
type RealTimeProvider struct{}

// NewRealTimeProvider creates a new real time provider.
func NewRealTimeProvider() TimeProvider {
	return &RealTimeProvider{}
}

// Now returns the current time.
func (t *RealTimeProvider) Now() time.Time {
	return time.Now()
}
