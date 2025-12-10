package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scenario-to-desktop-api/signing/types"
)

func TestDefaultValidator_ValidateConfig_NilConfig(t *testing.T) {
	v := NewValidator()
	result := v.ValidateConfig(nil)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultValidator_ValidateConfig_DisabledConfig(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{Enabled: false}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultValidator_ValidateConfig_EnabledNoPlattforms(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{Enabled: true}
	result := v.ValidateConfig(config)

	// Valid but with warning
	assert.True(t, result.Valid)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "NO_PLATFORMS_CONFIGURED", result.Warnings[0].Code)
}

func TestDefaultValidator_ValidateConfig_WindowsFileSource(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
			SignAlgorithm:     types.SignAlgorithmSHA256,
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultValidator_ValidateConfig_WindowsFileSourceMissingFile(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			// CertificateFile missing
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WIN_CERT_FILE_MISSING", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_WindowsStoreSource(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource:     types.CertSourceStore,
			CertificateThumbprint: "ABC123DEF456",
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultValidator_ValidateConfig_WindowsStoreSourceMissingThumbprint(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceStore,
			// CertificateThumbprint missing
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WIN_CERT_THUMBPRINT_MISSING", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_WindowsNoSource(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			// CertificateSource missing
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WIN_CERT_SOURCE_MISSING", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_WindowsInvalidSource(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: "invalid_source",
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WIN_CERT_SOURCE_INVALID", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_WindowsCloudKMSWarning(t *testing.T) {
	v := NewValidator()

	for _, source := range []string{types.CertSourceAzureKeyVault, types.CertSourceAWSKMS} {
		config := &types.SigningConfig{
			Enabled: true,
			Windows: &types.WindowsSigningConfig{
				CertificateSource: source,
			},
		}
		result := v.ValidateConfig(config)

		// Should be valid but with warning
		assert.True(t, result.Valid, "Cloud KMS source %s should be valid", source)
		require.Len(t, result.Warnings, 1)
		assert.Equal(t, "WIN_CLOUD_KMS_LIMITED", result.Warnings[0].Code)
	}
}

func TestDefaultValidator_ValidateConfig_WindowsInvalidAlgorithm(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
			SignAlgorithm:     "md5",
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "WIN_SIGN_ALGO_INVALID", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_MacOSValid(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test (TEAMID)",
			TeamID:          "TEAMID1234",
			HardenedRuntime: true,
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultValidator_ValidateConfig_MacOSMissingIdentity(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			// Identity missing
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "MACOS_IDENTITY_MISSING", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_MacOSNotarizeWithoutTeamID(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity:            "Developer ID Application: Test",
			Notarize:           true,
			HardenedRuntime:    true,
			AppleAPIKeyID:      "KEY123",
			AppleAPIKeyFile:    "/path/to/key.p8",
			// TeamID missing
		},
	}
	result := v.ValidateConfig(config)

	// Valid but with warning
	assert.True(t, result.Valid)
	require.NotEmpty(t, result.Warnings)
	hasTeamIDWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MACOS_TEAM_ID_RECOMMENDED" {
			hasTeamIDWarning = true
			break
		}
	}
	assert.True(t, hasTeamIDWarning)
}

func TestDefaultValidator_ValidateConfig_MacOSNotarizeWithoutHardenedRuntime(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity:         "Developer ID Application: Test",
			TeamID:           "TEAMID",
			Notarize:         true,
			HardenedRuntime:  false, // Should be true for notarization
			AppleAPIKeyID:    "KEY123",
			AppleAPIKeyFile:  "/path/to/key.p8",
		},
	}
	result := v.ValidateConfig(config)

	// Valid but with warning
	assert.True(t, result.Valid)
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MACOS_HARDENED_RUNTIME_NEEDED" {
			hasWarning = true
			break
		}
	}
	assert.True(t, hasWarning)
}

func TestDefaultValidator_ValidateConfig_MacOSNotarizeWithoutCredentials(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test",
			TeamID:          "TEAMID",
			HardenedRuntime: true,
			Notarize:        true,
			// No credentials provided
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	hasCredsError := false
	for _, e := range result.Errors {
		if e.Code == "MACOS_NOTARIZE_CREDS_MISSING" {
			hasCredsError = true
			break
		}
	}
	assert.True(t, hasCredsError)
}

func TestDefaultValidator_ValidateConfig_MacOSNotarizeWithAPIKey(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test",
			TeamID:          "TEAMID",
			HardenedRuntime: true,
			Notarize:        true,
			AppleAPIKeyID:   "KEY123",
			AppleAPIKeyFile: "/path/to/key.p8",
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	// Should have no errors about missing credentials
	for _, e := range result.Errors {
		assert.NotEqual(t, "MACOS_NOTARIZE_CREDS_MISSING", e.Code)
	}
}

func TestDefaultValidator_ValidateConfig_MacOSNotarizeWithAppPassword(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity:           "Developer ID Application: Test",
			TeamID:             "TEAMID",
			HardenedRuntime:    true,
			Notarize:           true,
			AppleIDEnv:         "APPLE_ID",
			AppleIDPasswordEnv: "APPLE_APP_PASSWORD",
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
}

func TestDefaultValidator_ValidateConfig_LinuxValid(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Linux: &types.LinuxSigningConfig{
			GPGKeyID: "ABC123DEF456",
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestDefaultValidator_ValidateConfig_LinuxMissingGPGKey(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Linux: &types.LinuxSigningConfig{
			// GPGKeyID missing
		},
	}
	result := v.ValidateConfig(config)

	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "LINUX_GPG_KEY_MISSING", result.Errors[0].Code)
}

func TestDefaultValidator_ValidateConfig_AllPlatforms(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
		MacOS: &types.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test",
			TeamID:          "TEAMID",
			HardenedRuntime: true,
		},
		Linux: &types.LinuxSigningConfig{
			GPGKeyID: "ABC123",
		},
	}
	result := v.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Len(t, result.Platforms, 3)
	assert.True(t, result.Platforms[types.PlatformWindows].Configured)
	assert.True(t, result.Platforms[types.PlatformMacOS].Configured)
	assert.True(t, result.Platforms[types.PlatformLinux].Configured)
}

func TestDefaultValidator_ValidateForPlatform_Windows(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	}
	result := v.ValidateForPlatform(config, types.PlatformWindows)

	assert.True(t, result.Valid)
}

func TestDefaultValidator_ValidateForPlatform_WindowsNotConfigured(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity: "Test",
		},
	}
	result := v.ValidateForPlatform(config, types.PlatformWindows)

	// Valid but with warning
	assert.True(t, result.Valid)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "PLATFORM_NOT_CONFIGURED", result.Warnings[0].Code)
	assert.Equal(t, "windows", result.Warnings[0].Platform)
}

func TestDefaultValidator_ValidateForPlatform_MacOS(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Identity: "Developer ID Application: Test",
		},
	}
	result := v.ValidateForPlatform(config, types.PlatformMacOS)

	assert.True(t, result.Valid)
}

func TestDefaultValidator_ValidateForPlatform_Linux(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{
		Enabled: true,
		Linux: &types.LinuxSigningConfig{
			GPGKeyID: "ABC123",
		},
	}
	result := v.ValidateForPlatform(config, types.PlatformLinux)

	assert.True(t, result.Valid)
}

func TestDefaultValidator_ValidateForPlatform_NilConfig(t *testing.T) {
	v := NewValidator()
	result := v.ValidateForPlatform(nil, types.PlatformWindows)

	assert.True(t, result.Valid)
}

func TestDefaultValidator_ValidateForPlatform_DisabledConfig(t *testing.T) {
	v := NewValidator()
	config := &types.SigningConfig{Enabled: false}
	result := v.ValidateForPlatform(config, types.PlatformWindows)

	assert.True(t, result.Valid)
}

func TestNewValidationResult(t *testing.T) {
	result := NewValidationResult()

	assert.True(t, result.Valid)
	assert.NotNil(t, result.Platforms)
	assert.NotNil(t, result.Errors)
	assert.NotNil(t, result.Warnings)
	assert.Empty(t, result.Platforms)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
}
