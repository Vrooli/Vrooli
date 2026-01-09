package validation

import (
	"net/url"
	"regexp"
	"strings"

	"deployment-manager/codesigning"
)

// ValidationRule defines a single validation check.
type ValidationRule interface {
	// Validate performs the validation check and adds any errors/warnings to the result.
	Validate(config *codesigning.SigningConfig, result *codesigning.ValidationResult)
}

// ruleFunc is an adapter to allow ordinary functions to be used as ValidationRule.
type ruleFunc func(config *codesigning.SigningConfig, result *codesigning.ValidationResult)

func (f ruleFunc) Validate(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	f(config, result)
}

// DefaultRules returns the standard set of validation rules.
func DefaultRules() []ValidationRule {
	return []ValidationRule{
		ruleFunc(validateWindowsCertificateSource),
		ruleFunc(validateWindowsFileSource),
		ruleFunc(validateWindowsStoreSource),
		ruleFunc(validateWindowsTimestampServer),
		ruleFunc(validateWindowsSignAlgorithm),
		ruleFunc(validateWindowsEnvVars),
		ruleFunc(validateMacOSIdentity),
		ruleFunc(validateMacOSTeamID),
		ruleFunc(validateMacOSNotarization),
		ruleFunc(validateMacOSHardenedRuntime),
		ruleFunc(validateMacOSEnvVars),
		ruleFunc(validateLinuxGPGKeyID),
		ruleFunc(validateLinuxEnvVars),
	}
}

// --- Windows Validation Rules ---

// validateWindowsCertificateSource ensures Windows config has a valid certificate source.
func validateWindowsCertificateSource(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows == nil {
		return
	}

	win := config.Windows

	// Certificate source is required
	if win.CertificateSource == "" {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_SOURCE_REQUIRED",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_source",
			Message:     "Windows signing requires certificate_source",
			Remediation: "Set certificate_source to one of: file, store, azure_keyvault, aws_kms",
		})
		return
	}

	// Validate source value
	if !codesigning.IsValidCertificateSource(win.CertificateSource) {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_SOURCE_INVALID",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_source",
			Message:     "Invalid certificate source: " + win.CertificateSource,
			Remediation: "Valid sources are: file, store, azure_keyvault, aws_kms",
		})
	}
}

// validateWindowsFileSource ensures file source has required fields.
func validateWindowsFileSource(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows == nil || config.Windows.CertificateSource != codesigning.CertSourceFile {
		return
	}

	win := config.Windows

	if strings.TrimSpace(win.CertificateFile) == "" {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_FILE_REQUIRED",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_file",
			Message:     "Certificate file path required when using file source",
			Remediation: "Set certificate_file to the path of your .pfx or .p12 certificate file",
		})
	}

	// Password is recommended for file-based certificates
	if strings.TrimSpace(win.CertificatePasswordEnv) == "" {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "WIN_CERT_PASSWORD_MISSING",
			Platform: codesigning.PlatformWindows,
			Message:  "No password environment variable specified for certificate file",
		})
	}
}

// validateWindowsStoreSource ensures store source has required fields.
func validateWindowsStoreSource(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows == nil || config.Windows.CertificateSource != codesigning.CertSourceStore {
		return
	}

	win := config.Windows

	if strings.TrimSpace(win.CertificateThumbprint) == "" {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_CERT_THUMBPRINT_REQUIRED",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_thumbprint",
			Message:     "Certificate thumbprint required when using store source",
			Remediation: "Set certificate_thumbprint to the SHA-1 thumbprint of your certificate in the Windows Certificate Store",
		})
	}
}

// validateWindowsTimestampServer validates the timestamp server URL if provided.
func validateWindowsTimestampServer(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows == nil {
		return
	}

	win := config.Windows

	// Timestamp server is optional but strongly recommended
	if win.TimestampServer == "" {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "WIN_NO_TIMESTAMP_SERVER",
			Platform: codesigning.PlatformWindows,
			Message:  "No timestamp server specified. Signatures will become invalid when the certificate expires.",
		})
		return
	}

	// Validate URL format
	if !isValidURL(win.TimestampServer) {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_TIMESTAMP_URL_INVALID",
			Platform:    codesigning.PlatformWindows,
			Field:       "timestamp_server",
			Message:     "Invalid timestamp server URL: " + win.TimestampServer,
			Remediation: "Use a valid RFC 3161 timestamp server URL like http://timestamp.digicert.com",
		})
	}
}

// validateWindowsSignAlgorithm validates the signing algorithm if provided.
func validateWindowsSignAlgorithm(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows == nil || config.Windows.SignAlgorithm == "" {
		return
	}

	if !codesigning.IsValidSignAlgorithm(config.Windows.SignAlgorithm) {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_SIGN_ALGORITHM_INVALID",
			Platform:    codesigning.PlatformWindows,
			Field:       "sign_algorithm",
			Message:     "Invalid signing algorithm: " + config.Windows.SignAlgorithm,
			Remediation: "Valid algorithms are: sha256, sha384, sha512",
		})
	}
}

// validateWindowsEnvVars validates environment variable names are valid identifiers.
func validateWindowsEnvVars(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows == nil {
		return
	}

	win := config.Windows

	if win.CertificatePasswordEnv != "" && !isValidEnvVarName(win.CertificatePasswordEnv) {
		result.AddError(codesigning.ValidationError{
			Code:        "WIN_INVALID_ENV_VAR",
			Platform:    codesigning.PlatformWindows,
			Field:       "certificate_password_env",
			Message:     "Invalid environment variable name: " + win.CertificatePasswordEnv,
			Remediation: "Environment variable names must contain only alphanumeric characters and underscores, and not start with a digit",
		})
	}
}

// --- macOS Validation Rules ---

// validateMacOSIdentity validates the signing identity.
func validateMacOSIdentity(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.MacOS == nil {
		return
	}

	mac := config.MacOS

	if strings.TrimSpace(mac.Identity) == "" {
		result.AddError(codesigning.ValidationError{
			Code:        "MACOS_IDENTITY_REQUIRED",
			Platform:    codesigning.PlatformMacOS,
			Field:       "identity",
			Message:     "Signing identity is required for macOS code signing",
			Remediation: "Set identity to your signing identity (e.g., 'Developer ID Application: Your Name (TEAMID)')",
		})
	}
}

// validateMacOSTeamID validates the Apple Team ID format.
func validateMacOSTeamID(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.MacOS == nil {
		return
	}

	mac := config.MacOS

	if strings.TrimSpace(mac.TeamID) == "" {
		result.AddError(codesigning.ValidationError{
			Code:        "MACOS_TEAM_ID_REQUIRED",
			Platform:    codesigning.PlatformMacOS,
			Field:       "team_id",
			Message:     "Apple Team ID is required for macOS code signing",
			Remediation: "Set team_id to your 10-character Apple Developer Team ID",
		})
		return
	}

	// Team ID must be exactly 10 alphanumeric characters
	if !isValidTeamID(mac.TeamID) {
		result.AddError(codesigning.ValidationError{
			Code:        "MACOS_TEAM_ID_INVALID",
			Platform:    codesigning.PlatformMacOS,
			Field:       "team_id",
			Message:     "Invalid Apple Team ID format: " + mac.TeamID,
			Remediation: "Team ID must be exactly 10 alphanumeric characters (e.g., 'ABC123XYZ0')",
		})
	}
}

// validateMacOSNotarization validates notarization credentials if enabled.
func validateMacOSNotarization(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.MacOS == nil || !config.MacOS.Notarize {
		return
	}

	mac := config.MacOS

	// Check for either App-Specific Password or API Key credentials
	hasAppPassword := mac.AppleIDEnv != "" && mac.AppleIDPasswordEnv != ""
	hasAPIKey := mac.AppleAPIKeyID != "" && mac.AppleAPIKeyFile != "" && mac.AppleAPIIssuerID != ""

	if !hasAppPassword && !hasAPIKey {
		result.AddError(codesigning.ValidationError{
			Code:        "MACOS_NOTARIZE_CREDS_MISSING",
			Platform:    codesigning.PlatformMacOS,
			Field:       "notarize",
			Message:     "Notarization requires Apple credentials",
			Remediation: "Provide either Apple ID credentials (apple_id_env + apple_id_password_env) or API Key credentials (apple_api_key_id + apple_api_key_file + apple_api_issuer_id)",
		})
	}

	// Warn if both methods are configured (API Key will take precedence)
	if hasAppPassword && hasAPIKey {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "MACOS_NOTARIZE_DUAL_CREDS",
			Platform: codesigning.PlatformMacOS,
			Message:  "Both App-Specific Password and API Key credentials configured. API Key will be used.",
		})
	}
}

// validateMacOSHardenedRuntime warns if hardened runtime is disabled with notarization.
func validateMacOSHardenedRuntime(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.MacOS == nil {
		return
	}

	mac := config.MacOS

	// Hardened runtime is required for notarization
	if mac.Notarize && !mac.HardenedRuntime {
		result.AddError(codesigning.ValidationError{
			Code:        "MACOS_HARDENED_RUNTIME_REQUIRED",
			Platform:    codesigning.PlatformMacOS,
			Field:       "hardened_runtime",
			Message:     "Hardened runtime is required for notarization",
			Remediation: "Set hardened_runtime to true when notarization is enabled",
		})
	}

	// Recommend hardened runtime even without notarization
	if !mac.HardenedRuntime {
		result.AddWarning(codesigning.ValidationWarning{
			Code:     "MACOS_HARDENED_RUNTIME_RECOMMENDED",
			Platform: codesigning.PlatformMacOS,
			Message:  "Hardened runtime is recommended for all macOS applications",
		})
	}
}

// validateMacOSEnvVars validates environment variable names are valid identifiers.
func validateMacOSEnvVars(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.MacOS == nil {
		return
	}

	mac := config.MacOS

	envVars := map[string]string{
		"apple_id_env":       mac.AppleIDEnv,
		"apple_id_password_env": mac.AppleIDPasswordEnv,
	}

	for field, value := range envVars {
		if value != "" && !isValidEnvVarName(value) {
			result.AddError(codesigning.ValidationError{
				Code:        "MACOS_INVALID_ENV_VAR",
				Platform:    codesigning.PlatformMacOS,
				Field:       field,
				Message:     "Invalid environment variable name: " + value,
				Remediation: "Environment variable names must contain only alphanumeric characters and underscores, and not start with a digit",
			})
		}
	}
}

// --- Linux Validation Rules ---

// validateLinuxGPGKeyID validates the GPG key ID if Linux signing is configured.
func validateLinuxGPGKeyID(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Linux == nil {
		return
	}

	linux := config.Linux

	// GPG key ID is required for Linux signing
	if strings.TrimSpace(linux.GPGKeyID) == "" {
		result.AddError(codesigning.ValidationError{
			Code:        "LINUX_GPG_KEY_ID_REQUIRED",
			Platform:    codesigning.PlatformLinux,
			Field:       "gpg_key_id",
			Message:     "GPG key ID is required for Linux signing",
			Remediation: "Set gpg_key_id to your GPG key ID or fingerprint",
		})
	}
}

// validateLinuxEnvVars validates environment variable names are valid identifiers.
func validateLinuxEnvVars(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Linux == nil {
		return
	}

	linux := config.Linux

	if linux.GPGPassphraseEnv != "" && !isValidEnvVarName(linux.GPGPassphraseEnv) {
		result.AddError(codesigning.ValidationError{
			Code:        "LINUX_INVALID_ENV_VAR",
			Platform:    codesigning.PlatformLinux,
			Field:       "gpg_passphrase_env",
			Message:     "Invalid environment variable name: " + linux.GPGPassphraseEnv,
			Remediation: "Environment variable names must contain only alphanumeric characters and underscores, and not start with a digit",
		})
	}
}

// --- Helper Functions ---

// isValidURL checks if a string is a valid URL.
func isValidURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	// Must have a scheme (http or https for timestamp servers)
	return u.Scheme == "http" || u.Scheme == "https"
}

// isValidEnvVarName checks if a string is a valid environment variable name.
// Valid names contain only alphanumeric characters and underscores, and don't start with a digit.
var envVarNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isValidEnvVarName(name string) bool {
	return envVarNameRegex.MatchString(name)
}

// isValidTeamID checks if a string is a valid Apple Team ID.
// Team IDs are exactly 10 alphanumeric characters.
var teamIDRegex = regexp.MustCompile(`^[A-Z0-9]{10}$`)

func isValidTeamID(id string) bool {
	return teamIDRegex.MatchString(id)
}
