package validation

import (
	"testing"

	"deployment-manager/codesigning"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
}

func TestValidator_ValidateConfig_NilConfig(t *testing.T) {
	v := NewValidator()
	result := v.ValidateConfig(nil)

	if result == nil {
		t.Fatal("ValidateConfig returned nil result")
	}
	if !result.Valid {
		t.Error("Expected valid result for nil config")
	}
	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}

func TestValidator_ValidateConfig_DisabledSigning(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: false,
		Windows: &codesigning.WindowsSigningConfig{
			// Invalid config that would fail if enabled
		},
	}
	result := v.ValidateConfig(config)

	if !result.Valid {
		t.Error("Expected valid result when signing is disabled")
	}
}

func TestValidator_ValidateConfig_WindowsValid(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource:      codesigning.CertSourceFile,
			CertificateFile:        "./cert.pfx",
			CertificatePasswordEnv: "WIN_CERT_PASSWORD",
			TimestampServer:        "http://timestamp.digicert.com",
			SignAlgorithm:          "sha256",
		},
	}
	result := v.ValidateConfig(config)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}
	if _, ok := result.Platforms[codesigning.PlatformWindows]; !ok {
		t.Error("Expected Windows platform in result")
	}
}

func TestValidator_ValidateConfig_WindowsMissingCertSource(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateFile: "./cert.pfx",
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for missing certificate source")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_CERT_SOURCE_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_CERT_SOURCE_REQUIRED error")
	}
}

func TestValidator_ValidateConfig_WindowsMissingCertFile(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			// Missing CertificateFile
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for missing certificate file")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_CERT_FILE_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_CERT_FILE_REQUIRED error")
	}
}

func TestValidator_ValidateConfig_WindowsInvalidTimestampURL(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			CertificateFile:   "./cert.pfx",
			TimestampServer:   "not-a-valid-url",
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for invalid timestamp URL")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_TIMESTAMP_URL_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_TIMESTAMP_URL_INVALID error")
	}
}

func TestValidator_ValidateConfig_MacOSValid(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test (ABC123XYZ0)",
			TeamID:          "ABC123XYZ0",
			HardenedRuntime: true,
			Notarize:        true,
			AppleAPIKeyID:   "KEY123",
			AppleAPIKeyFile: "./AuthKey.p8",
			AppleAPIIssuerID: "ISSUER-UUID",
		},
	}
	result := v.ValidateConfig(config)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidator_ValidateConfig_MacOSMissingIdentity(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			TeamID: "ABC123XYZ0",
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for missing identity")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_IDENTITY_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_IDENTITY_REQUIRED error")
	}
}

func TestValidator_ValidateConfig_MacOSInvalidTeamID(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity: "Developer ID Application: Test",
			TeamID:   "invalid",
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for invalid team ID")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_TEAM_ID_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_TEAM_ID_INVALID error")
	}
}

func TestValidator_ValidateConfig_MacOSNotarizeMissingCreds(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test (ABC123XYZ0)",
			TeamID:          "ABC123XYZ0",
			HardenedRuntime: true,
			Notarize:        true,
			// Missing notarization credentials
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for missing notarization credentials")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_NOTARIZE_CREDS_MISSING" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_NOTARIZE_CREDS_MISSING error")
	}
}

func TestValidator_ValidateConfig_MacOSNotarizeNoHardenedRuntime(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:          "Developer ID Application: Test (ABC123XYZ0)",
			TeamID:            "ABC123XYZ0",
			HardenedRuntime:   false,
			Notarize:          true,
			AppleIDEnv:        "APPLE_ID",
			AppleIDPasswordEnv: "APPLE_PASSWORD",
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for notarization without hardened runtime")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_HARDENED_RUNTIME_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_HARDENED_RUNTIME_REQUIRED error")
	}
}

func TestValidator_ValidateConfig_LinuxValid(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID:         "ABCD1234",
			GPGPassphraseEnv: "GPG_PASSPHRASE",
		},
	}
	result := v.ValidateConfig(config)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidator_ValidateConfig_LinuxMissingKeyID(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			// Missing GPGKeyID
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for missing GPG key ID")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "LINUX_GPG_KEY_ID_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected LINUX_GPG_KEY_ID_REQUIRED error")
	}
}

func TestValidator_ValidateForPlatform_InvalidPlatform(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
	}
	result := v.ValidateForPlatform(config, "invalid")

	if result.Valid {
		t.Error("Expected invalid result for invalid platform")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "INVALID_PLATFORM" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected INVALID_PLATFORM error")
	}
}

func TestValidator_ValidateForPlatform_PlatformNotConfigured(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		// Windows not configured
	}
	result := v.ValidateForPlatform(config, codesigning.PlatformWindows)

	if result.Valid {
		t.Error("Expected invalid result for unconfigured platform")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "PLATFORM_NOT_CONFIGURED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected PLATFORM_NOT_CONFIGURED error")
	}
}

func TestValidator_ValidateForPlatform_FiltersResults(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			CertificateFile:   "./cert.pfx",
			TimestampServer:   "http://timestamp.digicert.com",
		},
		MacOS: &codesigning.MacOSSigningConfig{
			// Invalid macOS config
		},
	}

	// Validate for Windows only - should not include macOS errors
	result := v.ValidateForPlatform(config, codesigning.PlatformWindows)

	// Should be valid for Windows
	for _, err := range result.Errors {
		if err.Platform == codesigning.PlatformMacOS {
			t.Errorf("Expected no macOS errors when validating for Windows, got: %s", err.Code)
		}
	}
}

func TestValidator_ValidateConfig_InvalidEnvVarName(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource:      codesigning.CertSourceFile,
			CertificateFile:        "./cert.pfx",
			CertificatePasswordEnv: "123_INVALID", // Invalid: starts with digit
			TimestampServer:        "http://timestamp.digicert.com",
		},
	}
	result := v.ValidateConfig(config)

	if result.Valid {
		t.Error("Expected invalid result for invalid env var name")
	}

	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_INVALID_ENV_VAR" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_INVALID_ENV_VAR error")
	}
}

func TestValidator_ValidateConfig_AllPlatforms(t *testing.T) {
	v := NewValidator()
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			CertificateFile:   "./windows.pfx",
			TimestampServer:   "http://timestamp.digicert.com",
		},
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test (ABC123XYZ0)",
			TeamID:          "ABC123XYZ0",
			HardenedRuntime: true,
		},
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID: "ABCD1234",
		},
	}
	result := v.ValidateConfig(config)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}

	// Check all platforms are represented
	if _, ok := result.Platforms[codesigning.PlatformWindows]; !ok {
		t.Error("Expected Windows in platforms")
	}
	if _, ok := result.Platforms[codesigning.PlatformMacOS]; !ok {
		t.Error("Expected macOS in platforms")
	}
	if _, ok := result.Platforms[codesigning.PlatformLinux]; !ok {
		t.Error("Expected Linux in platforms")
	}
}

func TestValidator_WithCustomRules(t *testing.T) {
	customRule := ruleFunc(func(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
		result.AddError(codesigning.ValidationError{
			Code:    "CUSTOM_ERROR",
			Message: "Custom validation error",
		})
	})

	v := NewValidator(WithRules(customRule))
	config := &codesigning.SigningConfig{Enabled: true}
	result := v.ValidateConfig(config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "CUSTOM_ERROR" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected custom rule to be applied")
	}
}
