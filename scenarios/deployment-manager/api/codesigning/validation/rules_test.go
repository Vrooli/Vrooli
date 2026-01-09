package validation

import (
	"testing"

	"deployment-manager/codesigning"
)

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()
	if len(rules) == 0 {
		t.Error("Expected default rules to be non-empty")
	}
}

// --- Windows Rules Tests ---

func TestValidateWindowsCertificateSource_Valid(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"file source", codesigning.CertSourceFile},
		{"store source", codesigning.CertSourceStore},
		{"azure keyvault", codesigning.CertSourceAzureKeyVault},
		{"aws kms", codesigning.CertSourceAWSKMS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &codesigning.SigningConfig{
				Enabled: true,
				Windows: &codesigning.WindowsSigningConfig{
					CertificateSource: tt.source,
					CertificateFile:   "./cert.pfx", // Needed for file source
				},
			}
			result := codesigning.NewValidationResult()
			validateWindowsCertificateSource(config, result)

			if !result.Valid {
				t.Errorf("Expected valid for source %s", tt.source)
			}
		})
	}
}

func TestValidateWindowsCertificateSource_Invalid(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: "invalid_source",
		},
	}
	result := codesigning.NewValidationResult()
	validateWindowsCertificateSource(config, result)

	if result.Valid {
		t.Error("Expected invalid for unknown source")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for invalid source")
	}
	if result.Errors[0].Code != "WIN_CERT_SOURCE_INVALID" {
		t.Errorf("Expected WIN_CERT_SOURCE_INVALID, got %s", result.Errors[0].Code)
	}
}

func TestValidateWindowsCertificateSource_Missing(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{},
	}
	result := codesigning.NewValidationResult()
	validateWindowsCertificateSource(config, result)

	if result.Valid {
		t.Error("Expected invalid for missing source")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for missing source")
	}
	if result.Errors[0].Code != "WIN_CERT_SOURCE_REQUIRED" {
		t.Errorf("Expected WIN_CERT_SOURCE_REQUIRED, got %s", result.Errors[0].Code)
	}
}

func TestValidateWindowsFileSource_MissingFile(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			// Missing CertificateFile
		},
	}
	result := codesigning.NewValidationResult()
	validateWindowsFileSource(config, result)

	if result.Valid {
		t.Error("Expected invalid for missing file")
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

func TestValidateWindowsFileSource_NoPasswordWarning(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			CertificateFile:   "./cert.pfx",
			// Missing CertificatePasswordEnv
		},
	}
	result := codesigning.NewValidationResult()
	validateWindowsFileSource(config, result)

	found := false
	for _, warn := range result.Warnings {
		if warn.Code == "WIN_CERT_PASSWORD_MISSING" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_CERT_PASSWORD_MISSING warning")
	}
}

func TestValidateWindowsStoreSource_MissingThumbprint(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceStore,
			// Missing CertificateThumbprint
		},
	}
	result := codesigning.NewValidationResult()
	validateWindowsStoreSource(config, result)

	if result.Valid {
		t.Error("Expected invalid for missing thumbprint")
	}
	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_CERT_THUMBPRINT_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_CERT_THUMBPRINT_REQUIRED error")
	}
}

func TestValidateWindowsTimestampServer_Valid(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"digicert", "http://timestamp.digicert.com"},
		{"sectigo", "http://timestamp.sectigo.com"},
		{"https", "https://timestamp.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &codesigning.SigningConfig{
				Enabled: true,
				Windows: &codesigning.WindowsSigningConfig{
					TimestampServer: tt.url,
				},
			}
			result := codesigning.NewValidationResult()
			validateWindowsTimestampServer(config, result)

			if !result.Valid {
				t.Errorf("Expected valid for URL %s, got errors: %v", tt.url, result.Errors)
			}
		})
	}
}

func TestValidateWindowsTimestampServer_Invalid(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"no scheme", "timestamp.digicert.com"},
		{"invalid scheme", "ftp://timestamp.example.com"},
		{"random string", "not-a-url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &codesigning.SigningConfig{
				Enabled: true,
				Windows: &codesigning.WindowsSigningConfig{
					TimestampServer: tt.url,
				},
			}
			result := codesigning.NewValidationResult()
			validateWindowsTimestampServer(config, result)

			if result.Valid {
				t.Errorf("Expected invalid for URL %s", tt.url)
			}
		})
	}
}

func TestValidateWindowsTimestampServer_MissingWarning(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			// Missing TimestampServer
		},
	}
	result := codesigning.NewValidationResult()
	validateWindowsTimestampServer(config, result)

	found := false
	for _, warn := range result.Warnings {
		if warn.Code == "WIN_NO_TIMESTAMP_SERVER" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_NO_TIMESTAMP_SERVER warning")
	}
}

func TestValidateWindowsSignAlgorithm_Valid(t *testing.T) {
	algorithms := []string{"sha256", "sha384", "sha512"}
	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			config := &codesigning.SigningConfig{
				Enabled: true,
				Windows: &codesigning.WindowsSigningConfig{
					SignAlgorithm: algo,
				},
			}
			result := codesigning.NewValidationResult()
			validateWindowsSignAlgorithm(config, result)

			if !result.Valid {
				t.Errorf("Expected valid for algorithm %s", algo)
			}
		})
	}
}

func TestValidateWindowsSignAlgorithm_Invalid(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			SignAlgorithm: "md5",
		},
	}
	result := codesigning.NewValidationResult()
	validateWindowsSignAlgorithm(config, result)

	if result.Valid {
		t.Error("Expected invalid for md5 algorithm")
	}
}

// --- macOS Rules Tests ---

func TestValidateMacOSIdentity_Missing(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS:   &codesigning.MacOSSigningConfig{},
	}
	result := codesigning.NewValidationResult()
	validateMacOSIdentity(config, result)

	if result.Valid {
		t.Error("Expected invalid for missing identity")
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

func TestValidateMacOSTeamID_Valid(t *testing.T) {
	tests := []struct {
		name   string
		teamID string
	}{
		{"all letters", "ABCDEFGHIJ"},
		{"all numbers", "1234567890"},
		{"mixed", "ABC123XYZ0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &codesigning.SigningConfig{
				Enabled: true,
				MacOS: &codesigning.MacOSSigningConfig{
					TeamID: tt.teamID,
				},
			}
			result := codesigning.NewValidationResult()
			validateMacOSTeamID(config, result)

			// Check for TEAM_ID_INVALID error specifically
			for _, err := range result.Errors {
				if err.Code == "MACOS_TEAM_ID_INVALID" {
					t.Errorf("Expected valid for team ID %s, got error", tt.teamID)
				}
			}
		})
	}
}

func TestValidateMacOSTeamID_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		teamID string
	}{
		{"too short", "ABC"},
		{"too long", "ABCDEFGHIJK"},
		{"lowercase", "abcdefghij"},
		{"special chars", "ABC123!@#$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &codesigning.SigningConfig{
				Enabled: true,
				MacOS: &codesigning.MacOSSigningConfig{
					TeamID: tt.teamID,
				},
			}
			result := codesigning.NewValidationResult()
			validateMacOSTeamID(config, result)

			found := false
			for _, err := range result.Errors {
				if err.Code == "MACOS_TEAM_ID_INVALID" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected MACOS_TEAM_ID_INVALID for %s", tt.teamID)
			}
		})
	}
}

func TestValidateMacOSTeamID_Missing(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS:   &codesigning.MacOSSigningConfig{},
	}
	result := codesigning.NewValidationResult()
	validateMacOSTeamID(config, result)

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_TEAM_ID_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_TEAM_ID_REQUIRED error")
	}
}

func TestValidateMacOSNotarization_AppPassword(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Notarize:           true,
			AppleIDEnv:         "APPLE_ID",
			AppleIDPasswordEnv: "APPLE_PASSWORD",
		},
	}
	result := codesigning.NewValidationResult()
	validateMacOSNotarization(config, result)

	for _, err := range result.Errors {
		if err.Code == "MACOS_NOTARIZE_CREDS_MISSING" {
			t.Error("Should not require creds when app password is provided")
		}
	}
}

func TestValidateMacOSNotarization_APIKey(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Notarize:         true,
			AppleAPIKeyID:    "KEY123",
			AppleAPIKeyFile:  "./AuthKey.p8",
			AppleAPIIssuerID: "ISSUER-UUID",
		},
	}
	result := codesigning.NewValidationResult()
	validateMacOSNotarization(config, result)

	for _, err := range result.Errors {
		if err.Code == "MACOS_NOTARIZE_CREDS_MISSING" {
			t.Error("Should not require creds when API key is provided")
		}
	}
}

func TestValidateMacOSNotarization_BothMethods(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Notarize:           true,
			AppleIDEnv:         "APPLE_ID",
			AppleIDPasswordEnv: "APPLE_PASSWORD",
			AppleAPIKeyID:      "KEY123",
			AppleAPIKeyFile:    "./AuthKey.p8",
			AppleAPIIssuerID:   "ISSUER-UUID",
		},
	}
	result := codesigning.NewValidationResult()
	validateMacOSNotarization(config, result)

	found := false
	for _, warn := range result.Warnings {
		if warn.Code == "MACOS_NOTARIZE_DUAL_CREDS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_NOTARIZE_DUAL_CREDS warning")
	}
}

func TestValidateMacOSNotarization_NoCreds(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Notarize: true,
		},
	}
	result := codesigning.NewValidationResult()
	validateMacOSNotarization(config, result)

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

func TestValidateMacOSHardenedRuntime_RequiredForNotarize(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Notarize:        true,
			HardenedRuntime: false,
		},
	}
	result := codesigning.NewValidationResult()
	validateMacOSHardenedRuntime(config, result)

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

func TestValidateMacOSHardenedRuntime_RecommendedWarning(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Notarize:        false,
			HardenedRuntime: false,
		},
	}
	result := codesigning.NewValidationResult()
	validateMacOSHardenedRuntime(config, result)

	found := false
	for _, warn := range result.Warnings {
		if warn.Code == "MACOS_HARDENED_RUNTIME_RECOMMENDED" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_HARDENED_RUNTIME_RECOMMENDED warning")
	}
}

// --- Linux Rules Tests ---

func TestValidateLinuxGPGKeyID_Missing(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux:   &codesigning.LinuxSigningConfig{},
	}
	result := codesigning.NewValidationResult()
	validateLinuxGPGKeyID(config, result)

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

func TestValidateLinuxGPGKeyID_Valid(t *testing.T) {
	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID: "ABCD1234",
		},
	}
	result := codesigning.NewValidationResult()
	validateLinuxGPGKeyID(config, result)

	if !result.Valid {
		t.Errorf("Expected valid, got errors: %v", result.Errors)
	}
}

// --- Helper Function Tests ---

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"https://example.com/path", true},
		{"ftp://example.com", false},
		{"example.com", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if isValidURL(tt.url) != tt.valid {
				t.Errorf("isValidURL(%s) = %v, want %v", tt.url, !tt.valid, tt.valid)
			}
		})
	}
}

func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"MY_VAR", true},
		{"_VAR", true},
		{"var", true},
		{"VAR123", true},
		{"VAR_123_TEST", true},
		{"123VAR", false},
		{"MY-VAR", false},
		{"MY.VAR", false},
		{"MY VAR", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isValidEnvVarName(tt.name) != tt.valid {
				t.Errorf("isValidEnvVarName(%s) = %v, want %v", tt.name, !tt.valid, tt.valid)
			}
		})
	}
}

func TestIsValidTeamID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"ABC123XYZ0", true},
		{"1234567890", true},
		{"ABCDEFGHIJ", true},
		{"ABC123", false},
		{"ABCDEFGHIJK", false},
		{"abcdefghij", false},
		{"ABC123XYZ!", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if isValidTeamID(tt.id) != tt.valid {
				t.Errorf("isValidTeamID(%s) = %v, want %v", tt.id, !tt.valid, tt.valid)
			}
		})
	}
}
