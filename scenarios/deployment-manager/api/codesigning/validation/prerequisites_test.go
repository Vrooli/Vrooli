package validation

import (
	"context"
	"os"
	"testing"
	"time"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/mocks"
)

func TestNewPrerequisiteChecker(t *testing.T) {
	c := NewPrerequisiteChecker()
	if c == nil {
		t.Fatal("NewPrerequisiteChecker returned nil")
	}
}

func TestPrerequisiteChecker_CheckPrerequisites_NilConfig(t *testing.T) {
	c := NewPrerequisiteChecker()
	result := c.CheckPrerequisites(context.Background(), nil)

	if result == nil {
		t.Fatal("CheckPrerequisites returned nil result")
	}
	if !result.Valid {
		t.Error("Expected valid result for nil config")
	}
}

func TestPrerequisiteChecker_CheckPrerequisites_DisabledSigning(t *testing.T) {
	c := NewPrerequisiteChecker()
	config := &codesigning.SigningConfig{
		Enabled: false,
	}
	result := c.CheckPrerequisites(context.Background(), config)

	if !result.Valid {
		t.Error("Expected valid result when signing is disabled")
	}
}

func TestPrerequisiteChecker_CheckPrerequisites_WithMocks(t *testing.T) {
	// Set up mocks
	mockFS := mocks.NewMockFileSystem()
	mockFS.AddFile("./cert.pfx", []byte("mock cert data"))

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("signtool.exe", `C:\Windows\signtool.exe`)
	mockCmd.AddCommand("signtool.exe", []byte("SignTool version 10.0"), nil, nil)

	mockEnv := mocks.NewMockEnvironmentReader()
	mockEnv.SetEnv("WIN_CERT_PASSWORD", "test-password")

	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource:      codesigning.CertSourceFile,
			CertificateFile:        "./cert.pfx",
			CertificatePasswordEnv: "WIN_CERT_PASSWORD",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	// Should have Windows platform in results
	if _, ok := result.Platforms[codesigning.PlatformWindows]; !ok {
		t.Error("Expected Windows platform in results")
	}

	pv := result.Platforms[codesigning.PlatformWindows]
	if !pv.Configured {
		t.Error("Expected Windows to be configured")
	}
	if !pv.ToolInstalled {
		t.Error("Expected signtool to be installed")
	}
}

func TestPrerequisiteChecker_DetectTools(t *testing.T) {
	mockCmd := mocks.NewMockCommandRunner()

	c := NewPrerequisiteChecker(
		WithCommandRunner(mockCmd),
	)

	results, err := c.DetectTools(context.Background())
	if err != nil {
		t.Fatalf("DetectTools failed: %v", err)
	}

	// Results depend on current platform
	if results == nil {
		t.Error("Expected non-nil results")
	}
}

func TestPrerequisiteChecker_Windows_SigntoolNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPathError("signtool.exe", os.ErrNotExist)
	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	// Should have error about signtool not found
	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_SIGNTOOL_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_SIGNTOOL_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_Windows_CertificateFileNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	// Don't add the cert file - it won't exist

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("signtool.exe", `C:\Windows\signtool.exe`)

	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource: codesigning.CertSourceFile,
			CertificateFile:   "./nonexistent.pfx",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "WIN_CERT_FILE_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_CERT_FILE_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_Windows_PasswordEnvNotSet(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	mockFS.AddFile("./cert.pfx", []byte("mock cert data"))

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("signtool.exe", `C:\Windows\signtool.exe`)

	mockEnv := mocks.NewMockEnvironmentReader()
	// Don't set the password env var

	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Windows: &codesigning.WindowsSigningConfig{
			CertificateSource:      codesigning.CertSourceFile,
			CertificateFile:        "./cert.pfx",
			CertificatePasswordEnv: "WIN_CERT_PASSWORD",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, warn := range result.Warnings {
		if warn.Code == "WIN_CERT_PASSWORD_ENV_NOT_SET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected WIN_CERT_PASSWORD_ENV_NOT_SET warning")
	}
}

func TestPrerequisiteChecker_MacOS_CodesignNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPathError("codesign", os.ErrNotExist)
	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity: "Developer ID Application: Test",
			TeamID:   "ABC123XYZ0",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_CODESIGN_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_CODESIGN_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_MacOS_NotarytoolNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("codesign", "/usr/bin/codesign")
	// notarytool check fails
	mockCmd.RunFunc = func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
		if name == "xcrun" && len(args) > 0 && args[0] == "notarytool" {
			return nil, nil, os.ErrNotExist
		}
		return nil, nil, nil
	}

	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:        "Developer ID Application: Test",
			TeamID:          "ABC123XYZ0",
			HardenedRuntime: true,
			Notarize:        true,
			AppleAPIKeyID:   "KEY123",
			AppleAPIKeyFile: "./AuthKey.p8",
			AppleAPIIssuerID: "ISSUER",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_NOTARYTOOL_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_NOTARYTOOL_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_MacOS_APIKeyFileNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	// Don't add the API key file

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("codesign", "/usr/bin/codesign")

	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:         "Developer ID Application: Test",
			TeamID:           "ABC123XYZ0",
			AppleAPIKeyFile:  "./nonexistent.p8",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_API_KEY_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_API_KEY_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_MacOS_EntitlementsFileNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	// Don't add the entitlements file

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("codesign", "/usr/bin/codesign")

	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		MacOS: &codesigning.MacOSSigningConfig{
			Identity:         "Developer ID Application: Test",
			TeamID:           "ABC123XYZ0",
			EntitlementsFile: "./nonexistent.plist",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "MACOS_ENTITLEMENTS_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MACOS_ENTITLEMENTS_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_Linux_GPGNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPathError("gpg", os.ErrNotExist)
	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID: "ABCD1234",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "LINUX_GPG_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected LINUX_GPG_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_Linux_GPGKeyNotFound(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("gpg", "/usr/bin/gpg")
	mockCmd.AddCommand("gpg", nil, []byte("gpg: error reading key: No secret key"), os.ErrNotExist)

	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID: "NONEXISTENT",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, err := range result.Errors {
		if err.Code == "LINUX_KEY_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected LINUX_KEY_NOT_FOUND error")
	}
}

func TestPrerequisiteChecker_Linux_PassphraseEnvNotSet(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()

	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("gpg", "/usr/bin/gpg")
	mockCmd.AddCommand("gpg", []byte("sec   rsa4096 2020-01-01"), nil, nil)

	mockEnv := mocks.NewMockEnvironmentReader()
	// Don't set the passphrase env var

	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID:         "ABCD1234",
			GPGPassphraseEnv: "GPG_PASSPHRASE",
		},
	}

	result := c.CheckPrerequisites(context.Background(), config)

	found := false
	for _, warn := range result.Warnings {
		if warn.Code == "LINUX_GPG_PASSPHRASE_ENV_NOT_SET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected LINUX_GPG_PASSPHRASE_ENV_NOT_SET warning")
	}
}

func TestPrerequisiteChecker_CheckPlatformPrerequisites(t *testing.T) {
	mockFS := mocks.NewMockFileSystem()
	mockCmd := mocks.NewMockCommandRunner()
	mockCmd.AddLookPath("gpg", "/usr/bin/gpg")
	mockCmd.AddCommand("gpg", []byte("gpg version 2.2.20"), nil, nil)

	mockEnv := mocks.NewMockEnvironmentReader()
	mockTime := mocks.NewMockTimeProvider(time.Now())

	c := NewPrerequisiteChecker(
		WithFileSystem(mockFS),
		WithCommandRunner(mockCmd),
		WithEnvironmentReader(mockEnv),
		WithTimeProvider(mockTime),
	)

	config := &codesigning.SigningConfig{
		Enabled: true,
		Linux: &codesigning.LinuxSigningConfig{
			GPGKeyID: "ABCD1234",
		},
	}

	result := c.CheckPlatformPrerequisites(context.Background(), config, codesigning.PlatformLinux)

	if _, ok := result.Platforms[codesigning.PlatformLinux]; !ok {
		t.Error("Expected Linux platform in results")
	}
}

func TestGetCertificateType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"cert.pfx", "pkcs12"},
		{"cert.p12", "pkcs12"},
		{"cert.PFX", "pkcs12"},
		{"cert.P12", "pkcs12"},
		{"cert.pem", "pem"},
		{"cert.crt", "pem"},
		{"cert.cer", "pem"},
		{"cert.txt", "unknown"},
		{"cert", "unknown"},
		{"/path/to/cert.pfx", "pkcs12"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getCertificateType(tt.path)
			if result != tt.expected {
				t.Errorf("getCertificateType(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}
