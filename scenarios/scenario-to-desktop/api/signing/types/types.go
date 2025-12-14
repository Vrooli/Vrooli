// Package types contains domain types for code signing configuration.
// This package has no dependencies on other signing subpackages to avoid import cycles.
package types

import "time"

// Platform constants
const (
	PlatformWindows = "windows"
	PlatformMacOS   = "macos"
	PlatformLinux   = "linux"
)

// Certificate source constants
const (
	CertSourceFile          = "file"
	CertSourceStore         = "store"
	CertSourceAzureKeyVault = "azure_keyvault"
	CertSourceAWSKMS        = "aws_kms"
)

// Signing algorithm constants
const (
	SignAlgorithmSHA256 = "sha256"
	SignAlgorithmSHA384 = "sha384"
	SignAlgorithmSHA512 = "sha512"
)

// Default timestamp server URLs
const (
	DefaultTimestampServerDigiCert   = "http://timestamp.digicert.com"
	DefaultTimestampServerSectigo    = "http://timestamp.sectigo.com"
	DefaultTimestampServerGlobalSign = "http://timestamp.globalsign.com/tsa/r6advanced1"
)

// SchemaVersion is the current config schema version
const SchemaVersion = "1.0"

// SigningConfig is the top-level code signing configuration.
// It contains platform-specific settings and controls whether signing is enabled.
type SigningConfig struct {
	// SchemaVersion is the config schema version for migration handling.
	SchemaVersion string `json:"schema_version,omitempty"`

	// Enabled controls whether code signing is applied during packaging.
	// When false, all platform configs are ignored.
	Enabled bool `json:"enabled"`

	// Windows contains Authenticode signing configuration.
	Windows *WindowsSigningConfig `json:"windows,omitempty"`

	// MacOS contains Apple code signing and notarization configuration.
	MacOS *MacOSSigningConfig `json:"macos,omitempty"`

	// Linux contains GPG signing configuration.
	Linux *LinuxSigningConfig `json:"linux,omitempty"`
}

// WindowsSigningConfig contains Windows Authenticode signing settings.
type WindowsSigningConfig struct {
	// CertificateSource specifies how the certificate is provided.
	// Values: "file", "store", "azure_keyvault", "aws_kms"
	CertificateSource string `json:"certificate_source"`

	// CertificateFile is the path to the .pfx/.p12 certificate file.
	// Required when CertificateSource is "file".
	CertificateFile string `json:"certificate_file,omitempty"`

	// CertificatePasswordEnv is the environment variable containing the certificate password.
	// The actual password is never stored in the config.
	CertificatePasswordEnv string `json:"certificate_password_env,omitempty"`

	// CertificateThumbprint identifies the certificate in Windows Certificate Store.
	// Required when CertificateSource is "store".
	CertificateThumbprint string `json:"certificate_thumbprint,omitempty"`

	// TimestampServer is the RFC 3161 timestamp server URL.
	// Recommended: "http://timestamp.digicert.com" or "http://timestamp.sectigo.com"
	TimestampServer string `json:"timestamp_server,omitempty"`

	// SignAlgorithm specifies the digest algorithm.
	// Values: "sha256" (recommended), "sha384", "sha512"
	SignAlgorithm string `json:"sign_algorithm,omitempty"`

	// DualSign enables SHA-1 + SHA-256 dual signing for Windows 7 compatibility.
	DualSign bool `json:"dual_sign,omitempty"`
}

// MacOSSigningConfig contains Apple code signing and notarization settings.
type MacOSSigningConfig struct {
	// Identity is the signing identity (e.g., "Developer ID Application: Name (TEAMID)").
	// Can be the full name or just the Team ID.
	Identity string `json:"identity"`

	// TeamID is the Apple Developer Team ID (10-character alphanumeric).
	TeamID string `json:"team_id"`

	// HardenedRuntime enables the hardened runtime, required for notarization.
	HardenedRuntime bool `json:"hardened_runtime"`

	// Notarize enables Apple notarization for Gatekeeper approval.
	// Requires AppleIDEnv and AppleIDPasswordEnv (or AppleAPIKey*).
	Notarize bool `json:"notarize"`

	// EntitlementsFile is the path to the entitlements.plist file.
	// Required for apps that need specific entitlements (JIT, unsigned memory, etc.)
	EntitlementsFile string `json:"entitlements_file,omitempty"`

	// ProvisioningProfile is the path to the .provisionprofile file.
	// Required for Mac App Store or enterprise distribution.
	ProvisioningProfile string `json:"provisioning_profile,omitempty"`

	// GatekeeperAssess runs gatekeeper assessment after signing.
	GatekeeperAssess bool `json:"gatekeeper_assess,omitempty"`

	// --- Notarization Credentials (App-Specific Password method) ---

	// AppleIDEnv is the environment variable containing the Apple ID email.
	AppleIDEnv string `json:"apple_id_env,omitempty"`

	// AppleIDPasswordEnv is the environment variable containing the app-specific password.
	AppleIDPasswordEnv string `json:"apple_id_password_env,omitempty"`

	// --- Notarization Credentials (API Key method - preferred) ---

	// AppleAPIKeyID is the API Key ID from App Store Connect.
	AppleAPIKeyID string `json:"apple_api_key_id,omitempty"`

	// AppleAPIKeyFile is the path to the .p8 API key file.
	AppleAPIKeyFile string `json:"apple_api_key_file,omitempty"`

	// AppleAPIIssuerID is the Issuer ID from App Store Connect.
	AppleAPIIssuerID string `json:"apple_api_issuer_id,omitempty"`
}

// LinuxSigningConfig contains Linux GPG signing settings.
type LinuxSigningConfig struct {
	// GPGKeyID is the GPG key ID or fingerprint for signing.
	GPGKeyID string `json:"gpg_key_id,omitempty"`

	// GPGPassphraseEnv is the environment variable containing the GPG passphrase.
	GPGPassphraseEnv string `json:"gpg_passphrase_env,omitempty"`

	// GPGHomedir overrides the default GPG home directory.
	GPGHomedir string `json:"gpg_homedir,omitempty"`
}

// ValidationResult contains the outcome of signing configuration validation.
type ValidationResult struct {
	Valid     bool                          `json:"valid"`
	Platforms map[string]PlatformValidation `json:"platforms"`
	Errors    []ValidationError             `json:"errors"`
	Warnings  []ValidationWarning           `json:"warnings"`
}

// Merge combines two ValidationResults.
func (v *ValidationResult) Merge(other *ValidationResult) {
	if other == nil {
		return
	}
	if !other.Valid {
		v.Valid = false
	}
	v.Errors = append(v.Errors, other.Errors...)
	v.Warnings = append(v.Warnings, other.Warnings...)
	for platform, validation := range other.Platforms {
		if existing, ok := v.Platforms[platform]; ok {
			existing.Errors = append(existing.Errors, validation.Errors...)
			existing.Warnings = append(existing.Warnings, validation.Warnings...)
			v.Platforms[platform] = existing
		} else {
			if v.Platforms == nil {
				v.Platforms = make(map[string]PlatformValidation)
			}
			v.Platforms[platform] = validation
		}
	}
}

// PlatformValidation contains validation results for a specific platform.
type PlatformValidation struct {
	Configured    bool             `json:"configured"`
	ToolInstalled bool             `json:"tool_installed"`
	ToolPath      string           `json:"tool_path,omitempty"`
	ToolVersion   string           `json:"tool_version,omitempty"`
	Certificate   *CertificateInfo `json:"certificate,omitempty"`
	Errors        []string         `json:"errors"`
	Warnings      []string         `json:"warnings"`
}

// CertificateInfo contains parsed certificate details.
type CertificateInfo struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serial_number"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	IsExpired    bool      `json:"is_expired"`
	DaysToExpiry int       `json:"days_to_expiry"`
	KeyUsage     []string  `json:"key_usage"`
	IsCodeSign   bool      `json:"is_code_sign"`
}

// ValidationError represents a blocking validation issue.
type ValidationError struct {
	Code        string `json:"code"`
	Platform    string `json:"platform"`
	Field       string `json:"field,omitempty"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

// ValidationWarning represents a non-blocking validation issue.
type ValidationWarning struct {
	Code     string `json:"code"`
	Platform string `json:"platform"`
	Message  string `json:"message"`
}

// ElectronBuilderSigningConfig is the signing portion of electron-builder config.
type ElectronBuilderSigningConfig struct {
	Win *ElectronBuilderWinSigning `json:"win,omitempty"`
	Mac *ElectronBuilderMacSigning `json:"mac,omitempty"`
}

// ElectronBuilderWinSigning maps to electron-builder's win signing config.
type ElectronBuilderWinSigning struct {
	CertificateFile        string   `json:"certificateFile,omitempty"`
	CertificatePassword    string   `json:"certificatePassword,omitempty"`
	CertificateSubjectName string   `json:"certificateSubjectName,omitempty"`
	CertificateSha1        string   `json:"certificateSha1,omitempty"`
	SignAndEditExecutable  bool     `json:"signAndEditExecutable"`
	SignDlls               bool     `json:"signDlls"`
	TimeStampServer        string   `json:"timeStampServer,omitempty"`
	Rfc3161TimeStampServer string   `json:"rfc3161TimeStampServer,omitempty"`
	SigningHashAlgorithms  []string `json:"signingHashAlgorithms,omitempty"`
}

// ElectronBuilderMacSigning maps to electron-builder's mac signing config.
type ElectronBuilderMacSigning struct {
	Identity            string      `json:"identity,omitempty"`
	HardenedRuntime     bool        `json:"hardenedRuntime"`
	GatekeeperAssess    bool        `json:"gatekeeperAssess"`
	Entitlements        string      `json:"entitlements,omitempty"`
	EntitlementsInherit string      `json:"entitlementsInherit,omitempty"`
	ProvisioningProfile string      `json:"provisioningProfile,omitempty"`
	Notarize            interface{} `json:"notarize,omitempty"` // bool or NotarizeConfig
}

// NotarizeConfig is the notarization configuration for electron-builder.
type NotarizeConfig struct {
	TeamID string `json:"teamId,omitempty"`
}

// ToolDetectionResult contains the results of signing tool detection.
type ToolDetectionResult struct {
	Platform    string `json:"platform"`
	Tool        string `json:"tool"`
	Installed   bool   `json:"installed"`
	Path        string `json:"path,omitempty"`
	Version     string `json:"version,omitempty"`
	Error       string `json:"error,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// DiscoveredCertificate represents a signing certificate or identity found on the system.
type DiscoveredCertificate struct {
	// ID is the unique identifier (thumbprint, identity name, or key ID)
	ID string `json:"id"`

	// Name is the human-readable display name
	Name string `json:"name"`

	// Subject is the certificate subject (CN, O, etc.)
	Subject string `json:"subject,omitempty"`

	// Issuer is the certificate issuer
	Issuer string `json:"issuer,omitempty"`

	// ExpiresAt is the expiration date (empty for GPG keys)
	ExpiresAt string `json:"expires_at,omitempty"`

	// DaysToExpiry is days until expiration (-1 if N/A or already expired)
	DaysToExpiry int `json:"days_to_expiry"`

	// IsExpired indicates if the certificate is expired
	IsExpired bool `json:"is_expired"`

	// IsCodeSign indicates if this can be used for code signing
	IsCodeSign bool `json:"is_code_sign"`

	// Type describes the certificate type (e.g., "Developer ID", "EV Code Signing", "GPG")
	Type string `json:"type,omitempty"`

	// Platform is the platform this certificate is for
	Platform string `json:"platform"`

	// UsageHint provides guidance on how to use this certificate
	UsageHint string `json:"usage_hint,omitempty"`
}

// ReadinessResponse contains the result of a signing readiness check.
// This is used by deployment-manager to verify signing is configured before packaging.
type ReadinessResponse struct {
	Ready     bool                      `json:"ready"`
	Scenario  string                    `json:"scenario"`
	Issues    []string                  `json:"issues,omitempty"`
	Platforms map[string]PlatformStatus `json:"platforms"`
}

// PlatformStatus contains readiness status for a specific platform.
type PlatformStatus struct {
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

// SigningConfigResponse wraps a signing config with metadata.
type SigningConfigResponse struct {
	Scenario   string         `json:"scenario"`
	Config     *SigningConfig `json:"config"`
	ConfigPath string         `json:"config_path"`
}
