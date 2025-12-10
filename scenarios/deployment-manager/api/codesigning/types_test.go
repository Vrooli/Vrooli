package codesigning

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSigningConfig_JSONRoundTrip(t *testing.T) {
	original := &SigningConfig{
		Enabled: true,
		Windows: &WindowsSigningConfig{
			CertificateSource:      CertSourceFile,
			CertificateFile:        "./certs/windows.pfx",
			CertificatePasswordEnv: "WIN_CERT_PASSWORD",
			TimestampServer:        DefaultTimestampServerDigiCert,
			SignAlgorithm:          SignAlgorithmSHA256,
			DualSign:               false,
		},
		MacOS: &MacOSSigningConfig{
			Identity:         "Developer ID Application: Vrooli (ABC123XYZ)",
			TeamID:           "ABC123XYZ",
			HardenedRuntime:  true,
			Notarize:         true,
			EntitlementsFile: "build/entitlements.mac.plist",
			AppleAPIKeyID:    "KEYID123",
			AppleAPIKeyFile:  "./certs/AuthKey_KEYID123.p8",
			AppleAPIIssuerID: "issuer-uuid",
		},
		Linux: &LinuxSigningConfig{
			GPGKeyID:         "ABC123DEF456",
			GPGPassphraseEnv: "GPG_PASSPHRASE",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded SigningConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.Enabled != original.Enabled {
		t.Errorf("enabled mismatch: got %v, want %v", decoded.Enabled, original.Enabled)
	}

	if decoded.Windows == nil {
		t.Fatal("windows config is nil after round-trip")
	}
	if decoded.Windows.CertificateSource != original.Windows.CertificateSource {
		t.Errorf("windows certificate_source mismatch: got %v, want %v",
			decoded.Windows.CertificateSource, original.Windows.CertificateSource)
	}
	if decoded.Windows.CertificateFile != original.Windows.CertificateFile {
		t.Errorf("windows certificate_file mismatch: got %v, want %v",
			decoded.Windows.CertificateFile, original.Windows.CertificateFile)
	}

	if decoded.MacOS == nil {
		t.Fatal("macos config is nil after round-trip")
	}
	if decoded.MacOS.Identity != original.MacOS.Identity {
		t.Errorf("macos identity mismatch: got %v, want %v",
			decoded.MacOS.Identity, original.MacOS.Identity)
	}
	if decoded.MacOS.TeamID != original.MacOS.TeamID {
		t.Errorf("macos team_id mismatch: got %v, want %v",
			decoded.MacOS.TeamID, original.MacOS.TeamID)
	}
	if decoded.MacOS.HardenedRuntime != original.MacOS.HardenedRuntime {
		t.Errorf("macos hardened_runtime mismatch: got %v, want %v",
			decoded.MacOS.HardenedRuntime, original.MacOS.HardenedRuntime)
	}

	if decoded.Linux == nil {
		t.Fatal("linux config is nil after round-trip")
	}
	if decoded.Linux.GPGKeyID != original.Linux.GPGKeyID {
		t.Errorf("linux gpg_key_id mismatch: got %v, want %v",
			decoded.Linux.GPGKeyID, original.Linux.GPGKeyID)
	}
}

func TestSigningConfig_JSONOmitEmpty(t *testing.T) {
	config := &SigningConfig{
		Enabled: true,
		Windows: &WindowsSigningConfig{
			CertificateSource: CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
		// MacOS and Linux are nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Should not contain "macos" or "linux" keys
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["macos"]; ok {
		t.Error("expected macos to be omitted when nil")
	}
	if _, ok := raw["linux"]; ok {
		t.Error("expected linux to be omitted when nil")
	}
	if _, ok := raw["windows"]; !ok {
		t.Error("expected windows to be present")
	}
}

func TestWindowsSigningConfig_OptionalFields(t *testing.T) {
	config := &WindowsSigningConfig{
		CertificateSource: CertSourceStore,
		CertificateThumbprint: "ABC123",
		// Other fields are optional
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Optional empty fields should be omitted
	if _, ok := raw["certificate_file"]; ok {
		t.Error("expected certificate_file to be omitted when empty")
	}
	if _, ok := raw["timestamp_server"]; ok {
		t.Error("expected timestamp_server to be omitted when empty")
	}

	// Required fields should be present
	if _, ok := raw["certificate_source"]; !ok {
		t.Error("expected certificate_source to be present")
	}
}

func TestMacOSSigningConfig_NotarizeCredentials(t *testing.T) {
	// Test API Key method
	apiKeyConfig := &MacOSSigningConfig{
		Identity:         "Developer ID Application: Test",
		TeamID:           "ABCD123456",
		HardenedRuntime:  true,
		Notarize:         true,
		AppleAPIKeyID:    "KEY123",
		AppleAPIKeyFile:  "./AuthKey.p8",
		AppleAPIIssuerID: "issuer-uuid",
	}

	data, err := json.Marshal(apiKeyConfig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MacOSSigningConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.AppleAPIKeyID != apiKeyConfig.AppleAPIKeyID {
		t.Errorf("apple_api_key_id mismatch: got %v, want %v",
			decoded.AppleAPIKeyID, apiKeyConfig.AppleAPIKeyID)
	}

	// Test App-Specific Password method
	appPasswordConfig := &MacOSSigningConfig{
		Identity:           "Developer ID Application: Test",
		TeamID:             "ABCD123456",
		HardenedRuntime:    true,
		Notarize:           true,
		AppleIDEnv:         "APPLE_ID",
		AppleIDPasswordEnv: "APPLE_APP_PASSWORD",
	}

	data, err = json.Marshal(appPasswordConfig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.AppleIDEnv != appPasswordConfig.AppleIDEnv {
		t.Errorf("apple_id_env mismatch: got %v, want %v",
			decoded.AppleIDEnv, appPasswordConfig.AppleIDEnv)
	}
}

func TestValidationResult_JSONSerialization(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Platforms: map[string]PlatformValidation{
			"windows": {
				Configured:    true,
				ToolInstalled: true,
				ToolPath:      "C:\\Program Files\\signtool.exe",
				ToolVersion:   "10.0.22000.0",
				Certificate: &CertificateInfo{
					Subject:      "CN=Test Corp",
					Issuer:       "CN=DigiCert",
					SerialNumber: "123456",
					NotBefore:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					NotAfter:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					IsExpired:    false,
					DaysToExpiry: 365,
					KeyUsage:     []string{"digitalSignature", "codeSigning"},
					IsCodeSign:   true,
				},
				Errors:   []string{},
				Warnings: []string{},
			},
			"macos": {
				Configured:    true,
				ToolInstalled: true,
				ToolPath:      "/usr/bin/codesign",
				Errors:        []string{"Identity not found"},
				Warnings:      []string{},
			},
		},
		Errors: []ValidationError{
			{
				Code:        "MACOS_IDENTITY_NOT_FOUND",
				Platform:    "macos",
				Field:       "identity",
				Message:     "Signing identity not found in keychain",
				Remediation: "Import the Developer ID certificate",
			},
		},
		Warnings: []ValidationWarning{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ValidationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Valid != result.Valid {
		t.Errorf("valid mismatch: got %v, want %v", decoded.Valid, result.Valid)
	}

	if len(decoded.Errors) != len(result.Errors) {
		t.Errorf("errors length mismatch: got %d, want %d", len(decoded.Errors), len(result.Errors))
	}

	if decoded.Errors[0].Code != result.Errors[0].Code {
		t.Errorf("error code mismatch: got %v, want %v", decoded.Errors[0].Code, result.Errors[0].Code)
	}

	windowsPlatform, ok := decoded.Platforms["windows"]
	if !ok {
		t.Fatal("windows platform not found in decoded result")
	}
	if windowsPlatform.Certificate == nil {
		t.Fatal("windows certificate is nil after round-trip")
	}
	if windowsPlatform.Certificate.Subject != "CN=Test Corp" {
		t.Errorf("certificate subject mismatch: got %v, want CN=Test Corp",
			windowsPlatform.Certificate.Subject)
	}
}

func TestElectronBuilderSigningConfig_JSONKeys(t *testing.T) {
	config := &ElectronBuilderSigningConfig{
		Win: &ElectronBuilderWinSigning{
			CertificateFile:        "./cert.pfx",
			CertificatePassword:    "${WIN_CERT_PASSWORD}",
			SignAndEditExecutable:  true,
			SignDlls:               true,
			Rfc3161TimeStampServer: "http://timestamp.digicert.com",
			SigningHashAlgorithms:  []string{"sha256"},
		},
		Mac: &ElectronBuilderMacSigning{
			Identity:            "Developer ID Application: Test",
			HardenedRuntime:     true,
			GatekeeperAssess:    true,
			Entitlements:        "build/entitlements.plist",
			EntitlementsInherit: "build/entitlements.plist",
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify camelCase JSON keys (electron-builder format)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	win, ok := raw["win"].(map[string]interface{})
	if !ok {
		t.Fatal("win is not a map")
	}

	// Check camelCase keys
	if _, ok := win["certificateFile"]; !ok {
		t.Error("expected camelCase 'certificateFile' key")
	}
	if _, ok := win["signAndEditExecutable"]; !ok {
		t.Error("expected camelCase 'signAndEditExecutable' key")
	}
	if _, ok := win["rfc3161TimeStampServer"]; !ok {
		t.Error("expected camelCase 'rfc3161TimeStampServer' key")
	}

	mac, ok := raw["mac"].(map[string]interface{})
	if !ok {
		t.Fatal("mac is not a map")
	}

	if _, ok := mac["hardenedRuntime"]; !ok {
		t.Error("expected camelCase 'hardenedRuntime' key")
	}
	if _, ok := mac["gatekeeperAssess"]; !ok {
		t.Error("expected camelCase 'gatekeeperAssess' key")
	}
}

func TestCertificateInfo_ExpirationFields(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	cert := &CertificateInfo{
		Subject:      "CN=Test",
		Issuer:       "CN=Issuer",
		SerialNumber: "123",
		NotBefore:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     notAfter,
		IsExpired:    IsCertificateExpired(notAfter, now),
		DaysToExpiry: CalculateDaysToExpiry(notAfter, now),
		KeyUsage:     []string{"codeSigning"},
		IsCodeSign:   true,
	}

	if cert.IsExpired {
		t.Error("certificate should not be expired")
	}

	if cert.DaysToExpiry != 30 {
		t.Errorf("days to expiry mismatch: got %d, want 30", cert.DaysToExpiry)
	}

	// Test with expired cert
	expiredNotAfter := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !IsCertificateExpired(expiredNotAfter, now) {
		t.Error("certificate should be expired")
	}
}
