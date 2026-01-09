package platforms

import (
	"context"
	"fmt"
	"testing"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/mocks"
)

func TestLinuxDetector_DetectGPG_Found(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	cmd.AddLookPath("gpg", "/usr/bin/gpg")
	cmd.AddCommand("gpg --version", []byte("gpg (GnuPG) 2.2.27\nlibgcrypt 1.9.3"), nil, nil)

	detector := &testableLinuxDetector{LinuxDetector: NewLinuxDetector(fs, cmd, env)}
	results := detector.DetectTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Installed {
		t.Error("expected gpg to be installed")
	}
	if result.Tool != "gpg" {
		t.Errorf("expected tool gpg, got %s", result.Tool)
	}
	if result.Path != "/usr/bin/gpg" {
		t.Errorf("expected path /usr/bin/gpg, got %s", result.Path)
	}
	if result.Version != "gpg (GnuPG) 2.2.27" {
		t.Errorf("expected version 'gpg (GnuPG) 2.2.27', got %s", result.Version)
	}
}

func TestLinuxDetector_DetectGPG_NotFound(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Don't add gpg to LookPath

	detector := &testableLinuxDetector{LinuxDetector: NewLinuxDetector(fs, cmd, env)}
	results := detector.DetectTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Installed {
		t.Error("expected gpg to not be installed")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
	if result.Remediation == "" {
		t.Error("expected remediation message")
	}
}

func TestLinuxDetector_ListGPGKeys(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Mock gpg list-secret-keys output
	// Note: GPG key IDs are hexadecimal (0-9, A-F only)
	gpgOutput := `sec   rsa4096/ABC123DEF456789A 2020-01-01 [SC] [expires: 2025-01-01]
      1234567890ABCDEF1234567890ABCDEF12345678
uid           [ultimate] Test User <test@example.com>
uid           [ultimate] Test User (work) <work@example.com>

sec   ed25519/DEF789ABC1234560 2022-06-15 [SC]
      ABCDEF1234567890ABCDEF1234567890ABCDEF12
uid           [ultimate] Another User <another@example.com>
`
	cmd.AddCommand("gpg --list-secret-keys --keyid-format long", []byte(gpgOutput), nil, nil)

	detector := &testableLinuxDetector{LinuxDetector: NewLinuxDetector(fs, cmd, env)}
	certs, err := detector.DiscoverCertificates(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(certs))
	}

	// Check first key
	key1 := certs[0]
	if key1.ID != "1234567890ABCDEF1234567890ABCDEF12345678" {
		t.Errorf("expected first key fingerprint, got %s", key1.ID)
	}
	if key1.Name != "Test User <test@example.com>" {
		t.Errorf("expected name 'Test User <test@example.com>', got %s", key1.Name)
	}
	if key1.Type != "GPG Secret Key" {
		t.Errorf("expected type 'GPG Secret Key', got %s", key1.Type)
	}
	if key1.ExpiresAt != "2025-01-01" {
		t.Errorf("expected expiry '2025-01-01', got %s", key1.ExpiresAt)
	}
	if !key1.IsCodeSign {
		t.Error("expected IsCodeSign to be true")
	}
	if key1.Platform != codesigning.PlatformLinux {
		t.Errorf("expected platform linux, got %s", key1.Platform)
	}

	// Check second key (no expiration)
	key2 := certs[1]
	if key2.ExpiresAt != "never" {
		t.Errorf("expected expiry 'never', got %s", key2.ExpiresAt)
	}
}

func TestLinuxDetector_ListGPGKeys_NoKeys(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	cmd.AddCommand("gpg --list-secret-keys --keyid-format long", []byte(""), nil, nil)

	detector := &testableLinuxDetector{LinuxDetector: NewLinuxDetector(fs, cmd, env)}
	certs, err := detector.DiscoverCertificates(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 0 {
		t.Errorf("expected 0 keys, got %d", len(certs))
	}
}

func TestLinuxDetector_ValidateKeyExists(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Mock successful key lookup - GPG returns non-zero exit code when key not found
	cmd.RunFunc = func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
		if name == "gpg" && len(args) > 0 && args[0] == "--list-secret-keys" {
			keyID := args[len(args)-1]
			if keyID == "VALIDKEY123" {
				return []byte("sec   rsa4096/VALIDKEY123"), nil, nil
			}
			// GPG returns error when key not found
			return nil, []byte("gpg: error reading key: No secret key"), fmt.Errorf("exit status 2")
		}
		return nil, nil, nil
	}

	detector := NewLinuxDetector(fs, cmd, env)

	// Test valid key
	exists, err := detector.ValidateKeyExists(ctx, "VALIDKEY123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}

	// Test invalid key
	exists, err = detector.ValidateKeyExists(ctx, "INVALIDKEY", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}
}

func TestLinuxDetector_ListKeysWithHomedir(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Track the args passed to gpg
	var capturedArgs []string
	cmd.RunFunc = func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
		if name == "gpg" {
			capturedArgs = args
			return []byte("sec   rsa4096/ABC123 2020-01-01\n      FINGERPRINT123\nuid           [ultimate] Test <test@example.com>\n"), nil, nil
		}
		return nil, nil, nil
	}

	detector := NewLinuxDetector(fs, cmd, env)
	_, err := detector.ListKeysForSigning(ctx, "/custom/gnupg")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify homedir was passed
	foundHomedir := false
	for i, arg := range capturedArgs {
		if arg == "--homedir" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "/custom/gnupg" {
			foundHomedir = true
			break
		}
	}

	if !foundHomedir {
		t.Errorf("expected --homedir /custom/gnupg in args, got %v", capturedArgs)
	}
}
