// Package validation provides structural validation for signing configurations.
package validation

import (
	"scenario-to-desktop-api/signing/types"
)

// DefaultValidator implements signing.Validator for structural validation.
type DefaultValidator struct{}

// NewValidator creates a new default validator.
func NewValidator() *DefaultValidator {
	return &DefaultValidator{}
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

// addError adds an error and marks the result as invalid.
func addError(result *types.ValidationResult, err types.ValidationError) {
	result.Errors = append(result.Errors, err)
	result.Valid = false
}

// addWarning adds a warning.
func addWarning(result *types.ValidationResult, warn types.ValidationWarning) {
	result.Warnings = append(result.Warnings, warn)
}

// ValidateConfig checks structural validity of a SigningConfig.
func (v *DefaultValidator) ValidateConfig(config *types.SigningConfig) *types.ValidationResult {
	result := NewValidationResult()

	if config == nil {
		return result // Empty config is valid (just disabled)
	}

	if !config.Enabled {
		return result // Disabled config needs no further validation
	}

	// Validate each configured platform
	if config.Windows != nil {
		v.validateWindows(config.Windows, result)
	}
	if config.MacOS != nil {
		v.validateMacOS(config.MacOS, result)
	}
	if config.Linux != nil {
		v.validateLinux(config.Linux, result)
	}

	// If enabled but no platforms configured, that's a warning
	if config.Windows == nil && config.MacOS == nil && config.Linux == nil {
		addWarning(result, types.ValidationWarning{
			Code:    "NO_PLATFORMS_CONFIGURED",
			Message: "Signing is enabled but no platforms are configured",
		})
	}

	return result
}

// ValidateForPlatform checks config is valid for a specific target platform.
func (v *DefaultValidator) ValidateForPlatform(config *types.SigningConfig, platform string) *types.ValidationResult {
	result := NewValidationResult()

	if config == nil || !config.Enabled {
		return result
	}

	switch platform {
	case types.PlatformWindows:
		if config.Windows != nil {
			v.validateWindows(config.Windows, result)
		} else {
			addWarning(result, types.ValidationWarning{
				Code:     "PLATFORM_NOT_CONFIGURED",
				Platform: platform,
				Message:  "Windows signing is not configured",
			})
		}
	case types.PlatformMacOS:
		if config.MacOS != nil {
			v.validateMacOS(config.MacOS, result)
		} else {
			addWarning(result, types.ValidationWarning{
				Code:     "PLATFORM_NOT_CONFIGURED",
				Platform: platform,
				Message:  "macOS signing is not configured",
			})
		}
	case types.PlatformLinux:
		if config.Linux != nil {
			v.validateLinux(config.Linux, result)
		} else {
			addWarning(result, types.ValidationWarning{
				Code:     "PLATFORM_NOT_CONFIGURED",
				Platform: platform,
				Message:  "Linux signing is not configured",
			})
		}
	}

	return result
}

// validateWindows checks Windows-specific configuration.
func (v *DefaultValidator) validateWindows(config *types.WindowsSigningConfig, result *types.ValidationResult) {
	pv := types.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Validate certificate source
	switch config.CertificateSource {
	case types.CertSourceFile:
		if config.CertificateFile == "" {
			addError(result, types.ValidationError{
				Code:        "WIN_CERT_FILE_MISSING",
				Platform:    types.PlatformWindows,
				Field:       "certificate_file",
				Message:     "Certificate file path is required when certificate_source is 'file'",
				Remediation: "Provide the path to your .pfx or .p12 certificate file",
			})
			pv.Errors = append(pv.Errors, "Certificate file missing")
		}
	case types.CertSourceStore:
		if config.CertificateThumbprint == "" {
			addError(result, types.ValidationError{
				Code:        "WIN_CERT_THUMBPRINT_MISSING",
				Platform:    types.PlatformWindows,
				Field:       "certificate_thumbprint",
				Message:     "Certificate thumbprint is required when certificate_source is 'store'",
				Remediation: "Provide the SHA-1 thumbprint of the certificate in the Windows Certificate Store",
			})
			pv.Errors = append(pv.Errors, "Certificate thumbprint missing")
		}
	case types.CertSourceAzureKeyVault, types.CertSourceAWSKMS:
		addWarning(result, types.ValidationWarning{
			Code:     "WIN_CLOUD_KMS_LIMITED",
			Platform: types.PlatformWindows,
			Message:  "Cloud KMS signing (Azure/AWS) requires custom signtool configuration not fully supported by electron-builder",
		})
	case "":
		addError(result, types.ValidationError{
			Code:        "WIN_CERT_SOURCE_MISSING",
			Platform:    types.PlatformWindows,
			Field:       "certificate_source",
			Message:     "Certificate source is required",
			Remediation: "Set certificate_source to 'file' or 'store'",
		})
		pv.Errors = append(pv.Errors, "Certificate source missing")
	default:
		addError(result, types.ValidationError{
			Code:        "WIN_CERT_SOURCE_INVALID",
			Platform:    types.PlatformWindows,
			Field:       "certificate_source",
			Message:     "Invalid certificate source: " + config.CertificateSource,
			Remediation: "Use 'file', 'store', 'azure_keyvault', or 'aws_kms'",
		})
		pv.Errors = append(pv.Errors, "Invalid certificate source")
	}

	// Validate sign algorithm
	if config.SignAlgorithm != "" {
		switch config.SignAlgorithm {
		case types.SignAlgorithmSHA256, types.SignAlgorithmSHA384, types.SignAlgorithmSHA512:
			// Valid
		default:
			addError(result, types.ValidationError{
				Code:        "WIN_SIGN_ALGO_INVALID",
				Platform:    types.PlatformWindows,
				Field:       "sign_algorithm",
				Message:     "Invalid sign algorithm: " + config.SignAlgorithm,
				Remediation: "Use 'sha256', 'sha384', or 'sha512'",
			})
			pv.Errors = append(pv.Errors, "Invalid sign algorithm")
		}
	}

	result.Platforms[types.PlatformWindows] = pv
}

// validateMacOS checks macOS-specific configuration.
func (v *DefaultValidator) validateMacOS(config *types.MacOSSigningConfig, result *types.ValidationResult) {
	pv := types.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Identity is required
	if config.Identity == "" {
		addError(result, types.ValidationError{
			Code:        "MACOS_IDENTITY_MISSING",
			Platform:    types.PlatformMacOS,
			Field:       "identity",
			Message:     "Signing identity is required",
			Remediation: "Provide the signing identity (e.g., 'Developer ID Application: Your Name (TEAMID)')",
		})
		pv.Errors = append(pv.Errors, "Signing identity missing")
	}

	// Team ID should be set when using notarization
	if config.Notarize && config.TeamID == "" {
		addWarning(result, types.ValidationWarning{
			Code:     "MACOS_TEAM_ID_RECOMMENDED",
			Platform: types.PlatformMacOS,
			Message:  "Team ID is recommended when notarization is enabled",
		})
		pv.Warnings = append(pv.Warnings, "Team ID recommended")
	}

	// Hardened runtime should be enabled for notarization
	if config.Notarize && !config.HardenedRuntime {
		addWarning(result, types.ValidationWarning{
			Code:     "MACOS_HARDENED_RUNTIME_NEEDED",
			Platform: types.PlatformMacOS,
			Message:  "Hardened runtime should be enabled when notarization is enabled",
		})
		pv.Warnings = append(pv.Warnings, "Hardened runtime recommended")
	}

	// Notarization requires credentials
	if config.Notarize {
		hasAPIKey := config.AppleAPIKeyID != "" && config.AppleAPIKeyFile != ""
		hasAppPassword := config.AppleIDEnv != "" && config.AppleIDPasswordEnv != ""

		if !hasAPIKey && !hasAppPassword {
			addError(result, types.ValidationError{
				Code:        "MACOS_NOTARIZE_CREDS_MISSING",
				Platform:    types.PlatformMacOS,
				Message:     "Notarization requires either API Key credentials or Apple ID app-specific password",
				Remediation: "Set apple_api_key_id + apple_api_key_file OR apple_id_env + apple_id_password_env",
			})
			pv.Errors = append(pv.Errors, "Notarization credentials missing")
		}
	}

	result.Platforms[types.PlatformMacOS] = pv
}

// validateLinux checks Linux-specific configuration.
func (v *DefaultValidator) validateLinux(config *types.LinuxSigningConfig, result *types.ValidationResult) {
	pv := types.PlatformValidation{
		Configured: true,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// GPG key ID is required
	if config.GPGKeyID == "" {
		addError(result, types.ValidationError{
			Code:        "LINUX_GPG_KEY_MISSING",
			Platform:    types.PlatformLinux,
			Field:       "gpg_key_id",
			Message:     "GPG key ID is required for Linux signing",
			Remediation: "Provide the GPG key ID or fingerprint to sign with",
		})
		pv.Errors = append(pv.Errors, "GPG key ID missing")
	}

	result.Platforms[types.PlatformLinux] = pv
}
