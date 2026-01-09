package generation

import (
	"encoding/json"
	"strings"
	"testing"

	"deployment-manager/codesigning"
)

func TestNewGenerator(t *testing.T) {
	t.Run("with nil options uses defaults", func(t *testing.T) {
		g := NewGenerator(nil)
		if g == nil {
			t.Fatal("expected non-nil generator")
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		opts := &Options{
			OutputDir:          "custom",
			EntitlementsPath:   "entitlements.plist",
			NotarizeScriptPath: "notarize.js",
		}
		g := NewGenerator(opts)
		if g == nil {
			t.Fatal("expected non-nil generator")
		}
	})
}

func TestGenerator_GenerateElectronBuilder_Disabled(t *testing.T) {
	g := NewGenerator(nil)

	t.Run("nil config returns nil", func(t *testing.T) {
		result, err := g.GenerateElectronBuilder(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for nil config")
		}
	})

	t.Run("disabled config returns nil", func(t *testing.T) {
		config := &codesigning.SigningConfig{Enabled: false}
		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for disabled config")
		}
	})
}

func TestGenerator_GenerateElectronBuilder_Windows(t *testing.T) {
	g := NewGenerator(nil)

	t.Run("file source with password env", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{
				CertificateSource:      codesigning.CertSourceFile,
				CertificateFile:        "./certs/windows.pfx",
				CertificatePasswordEnv: "WIN_CERT_PASSWORD",
				TimestampServer:        "http://timestamp.digicert.com",
				SignAlgorithm:          "sha256",
			},
		}

		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil || result.Win == nil {
			t.Fatal("expected Windows config")
		}

		if result.Win.CertificateFile != "./certs/windows.pfx" {
			t.Errorf("expected certificate file './certs/windows.pfx', got %q", result.Win.CertificateFile)
		}
		if result.Win.CertificatePassword != "${WIN_CERT_PASSWORD}" {
			t.Errorf("expected password env reference, got %q", result.Win.CertificatePassword)
		}
		if result.Win.Rfc3161TimeStampServer != "http://timestamp.digicert.com" {
			t.Errorf("expected timestamp server, got %q", result.Win.Rfc3161TimeStampServer)
		}
		if !result.Win.SignAndEditExecutable {
			t.Error("expected SignAndEditExecutable to be true")
		}
		if !result.Win.SignDlls {
			t.Error("expected SignDlls to be true")
		}
	})

	t.Run("store source with thumbprint", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{
				CertificateSource:     codesigning.CertSourceStore,
				CertificateThumbprint: "ABC123DEF456",
			},
		}

		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Win.CertificateSha1 != "ABC123DEF456" {
			t.Errorf("expected thumbprint 'ABC123DEF456', got %q", result.Win.CertificateSha1)
		}
	})

	t.Run("dual signing algorithms", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{
				CertificateSource: codesigning.CertSourceFile,
				CertificateFile:   "./cert.pfx",
				DualSign:          true,
			},
		}

		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Win.SigningHashAlgorithms) != 2 {
			t.Errorf("expected 2 algorithms for dual sign, got %d", len(result.Win.SigningHashAlgorithms))
		}
		if result.Win.SigningHashAlgorithms[0] != "sha1" || result.Win.SigningHashAlgorithms[1] != "sha256" {
			t.Errorf("expected [sha1, sha256], got %v", result.Win.SigningHashAlgorithms)
		}
	})
}

func TestGenerator_GenerateElectronBuilder_MacOS(t *testing.T) {
	g := NewGenerator(nil)

	t.Run("basic config with hardened runtime", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			MacOS: &codesigning.MacOSSigningConfig{
				Identity:        "Developer ID Application: Test (ABC123)",
				TeamID:          "ABC123",
				HardenedRuntime: true,
			},
		}

		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil || result.Mac == nil {
			t.Fatal("expected macOS config")
		}

		if result.Mac.Identity != "Developer ID Application: Test (ABC123)" {
			t.Errorf("expected identity, got %q", result.Mac.Identity)
		}
		if !result.Mac.HardenedRuntime {
			t.Error("expected HardenedRuntime to be true")
		}
		if !result.Mac.GatekeeperAssess {
			t.Error("expected GatekeeperAssess to be true")
		}
	})

	t.Run("with notarization", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			MacOS: &codesigning.MacOSSigningConfig{
				Identity:        "Developer ID Application: Test (ABC123)",
				TeamID:          "ABC123",
				HardenedRuntime: true,
				Notarize:        true,
			},
		}

		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Mac.Notarize == nil {
			t.Fatal("expected Notarize config")
		}

		// Should be a NotarizeConfig with TeamID
		notarizeConfig, ok := result.Mac.Notarize.(*codesigning.NotarizeConfig)
		if !ok {
			t.Fatalf("expected *NotarizeConfig, got %T", result.Mac.Notarize)
		}
		if notarizeConfig.TeamID != "ABC123" {
			t.Errorf("expected TeamID 'ABC123', got %q", notarizeConfig.TeamID)
		}
	})

	t.Run("with custom entitlements file", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			MacOS: &codesigning.MacOSSigningConfig{
				Identity:         "Developer ID Application: Test (ABC123)",
				EntitlementsFile: "custom/entitlements.plist",
			},
		}

		result, err := g.GenerateElectronBuilder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Mac.Entitlements != "custom/entitlements.plist" {
			t.Errorf("expected custom entitlements path, got %q", result.Mac.Entitlements)
		}
		if result.Mac.EntitlementsInherit != "custom/entitlements.plist" {
			t.Errorf("expected custom entitlements inherit path, got %q", result.Mac.EntitlementsInherit)
		}
	})
}

func TestGenerator_GenerateEntitlements(t *testing.T) {
	g := NewGenerator(nil)

	t.Run("nil config returns nil", func(t *testing.T) {
		result, err := g.GenerateEntitlements(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for nil config")
		}
	})

	t.Run("generates valid plist with default entitlements", func(t *testing.T) {
		config := &codesigning.MacOSSigningConfig{
			HardenedRuntime: true,
		}

		result, err := g.GenerateEntitlements(config, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := string(result)

		// Check XML header
		if !strings.HasPrefix(content, "<?xml version=\"1.0\"") {
			t.Error("expected XML header")
		}

		// Check for default Electron entitlements
		if !strings.Contains(content, "com.apple.security.cs.allow-jit") {
			t.Error("expected allow-jit entitlement")
		}
		if !strings.Contains(content, "com.apple.security.cs.allow-unsigned-executable-memory") {
			t.Error("expected allow-unsigned-executable-memory entitlement")
		}
		if !strings.Contains(content, "com.apple.security.cs.disable-library-validation") {
			t.Error("expected disable-library-validation entitlement")
		}
	})

	t.Run("adds capability-based entitlements", func(t *testing.T) {
		config := &codesigning.MacOSSigningConfig{}

		result, err := g.GenerateEntitlements(config, []string{"network", "camera", "bluetooth"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := string(result)

		if !strings.Contains(content, "com.apple.security.network.client") {
			t.Error("expected network.client entitlement")
		}
		if !strings.Contains(content, "com.apple.security.device.camera") {
			t.Error("expected device.camera entitlement")
		}
		if !strings.Contains(content, "com.apple.security.device.bluetooth") {
			t.Error("expected device.bluetooth entitlement")
		}
	})
}

func TestGenerator_GenerateNotarizeScript(t *testing.T) {
	g := NewGenerator(nil)

	t.Run("nil config returns nil", func(t *testing.T) {
		result, err := g.GenerateNotarizeScript(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for nil config")
		}
	})

	t.Run("notarize disabled returns nil", func(t *testing.T) {
		config := &codesigning.MacOSSigningConfig{
			Notarize: false,
		}

		result, err := g.GenerateNotarizeScript(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for disabled notarization")
		}
	})

	t.Run("API key method generates correct script", func(t *testing.T) {
		config := &codesigning.MacOSSigningConfig{
			TeamID:           "ABC123XYZ",
			Notarize:         true,
			AppleAPIKeyID:    "KEYID123",
			AppleAPIIssuerID: "issuer-uuid",
		}

		result, err := g.GenerateNotarizeScript(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := string(result)

		if !strings.Contains(content, "notarytool") {
			t.Error("expected notarytool reference")
		}
		if !strings.Contains(content, "KEYID123") {
			t.Error("expected API key ID in script")
		}
		if !strings.Contains(content, "ABC123XYZ") {
			t.Error("expected team ID in script")
		}
		if !strings.Contains(content, "issuer-uuid") {
			t.Error("expected issuer ID in script")
		}
	})

	t.Run("app password method generates correct script", func(t *testing.T) {
		config := &codesigning.MacOSSigningConfig{
			TeamID:             "ABC123XYZ",
			Notarize:           true,
			AppleIDEnv:         "MY_APPLE_ID",
			AppleIDPasswordEnv: "MY_APPLE_PASSWORD",
		}

		result, err := g.GenerateNotarizeScript(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := string(result)

		if !strings.Contains(content, "MY_APPLE_ID") {
			t.Error("expected Apple ID env var in script")
		}
		if !strings.Contains(content, "MY_APPLE_PASSWORD") {
			t.Error("expected Apple password env var in script")
		}
		if !strings.Contains(content, "appleId:") {
			t.Error("expected appleId property in script")
		}
	})
}

func TestGenerator_GenerateAll(t *testing.T) {
	g := NewGenerator(nil)

	t.Run("nil config returns nil", func(t *testing.T) {
		result, err := g.GenerateAll(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for nil config")
		}
	})

	t.Run("disabled config returns nil", func(t *testing.T) {
		config := &codesigning.SigningConfig{Enabled: false}
		result, err := g.GenerateAll(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result for disabled config")
		}
	})

	t.Run("macOS config generates entitlements and notarize script", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			MacOS: &codesigning.MacOSSigningConfig{
				Identity:        "Developer ID Application: Test",
				TeamID:          "ABC123",
				HardenedRuntime: true,
				Notarize:        true,
				AppleAPIKeyID:   "KEYID123",
			},
		}

		result, err := g.GenerateAll(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 files, got %d", len(result))
		}

		// Check entitlements file
		if _, ok := result["build/entitlements.mac.plist"]; !ok {
			t.Error("expected entitlements.mac.plist in output")
		}

		// Check notarize script
		if _, ok := result["scripts/notarize.js"]; !ok {
			t.Error("expected scripts/notarize.js in output")
		}
	})
}

func TestGenerateElectronBuilderJSON(t *testing.T) {
	t.Run("generates complete config", func(t *testing.T) {
		config := &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{
				CertificateSource:      codesigning.CertSourceFile,
				CertificateFile:        "./cert.pfx",
				CertificatePasswordEnv: "WIN_CERT_PASSWORD",
			},
			MacOS: &codesigning.MacOSSigningConfig{
				Identity:        "Developer ID Application: Test",
				TeamID:          "ABC123",
				HardenedRuntime: true,
				Notarize:        true,
			},
		}

		result, err := GenerateElectronBuilderJSON(config, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check Windows config present
		if _, ok := result["win"]; !ok {
			t.Error("expected 'win' in result")
		}

		// Check macOS config present
		if _, ok := result["mac"]; !ok {
			t.Error("expected 'mac' in result")
		}

		// Check afterSign hook present
		if afterSign, ok := result["afterSign"]; !ok {
			t.Error("expected 'afterSign' in result")
		} else if afterSign != "scripts/notarize.js" {
			t.Errorf("expected 'scripts/notarize.js', got %v", afterSign)
		}

		// Verify JSON serialization works
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("failed to serialize to JSON: %v", err)
		}
		if len(jsonBytes) == 0 {
			t.Error("expected non-empty JSON output")
		}
	})
}

func TestParseCapabilities(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"network", []string{"network"}},
		{"network,camera", []string{"network", "camera"}},
		{"network, camera, bluetooth", []string{"network", "camera", "bluetooth"}},
		{" network , camera ", []string{"network", "camera"}},
	}

	for _, tt := range tests {
		result := ParseCapabilities(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseCapabilities(%q): expected %v, got %v", tt.input, tt.expected, result)
			continue
		}
		for i, v := range tt.expected {
			if result[i] != v {
				t.Errorf("ParseCapabilities(%q)[%d]: expected %q, got %q", tt.input, i, v, result[i])
			}
		}
	}
}

func TestNotarizeCredentialMethod(t *testing.T) {
	tests := []struct {
		name     string
		config   *codesigning.MacOSSigningConfig
		expected string
	}{
		{"nil config", nil, ""},
		{"notarize disabled", &codesigning.MacOSSigningConfig{Notarize: false}, ""},
		{"API key method", &codesigning.MacOSSigningConfig{Notarize: true, AppleAPIKeyID: "KEY123"}, "api_key"},
		{"app password method", &codesigning.MacOSSigningConfig{Notarize: true, AppleIDEnv: "APPLE_ID"}, "app_password"},
		{"no credentials", &codesigning.MacOSSigningConfig{Notarize: true}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NotarizeCredentialMethod(tt.config)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
