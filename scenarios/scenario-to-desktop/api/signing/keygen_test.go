package signing

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type fakeAuthority struct {
	available bool
	values    map[string]string
}

func newFakeAuthority() *fakeAuthority {
	return &fakeAuthority{available: true, values: map[string]string{}}
}

func (f *fakeAuthority) key(identity credentialauthority.Identity, field string) string {
	return string(identity) + ":" + field
}

func (f *fakeAuthority) Put(identity credentialauthority.Identity, field, value string) error {
	f.values[f.key(identity, field)] = value
	return nil
}

func (f *fakeAuthority) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	value, ok := f.values[f.key(identity, field)]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (f *fakeAuthority) Status(identity credentialauthority.Identity, field string) credentialauthority.Status {
	return credentialauthority.Status{Configured: f.values[f.key(identity, field)] != ""}
}

func (f *fakeAuthority) Availability() error {
	if !f.available {
		return errors.New("credential provider is unavailable")
	}
	return nil
}

func (f *fakeAuthority) Provider() string { return "fake" }

func withFakeAuthority(t *testing.T, authority *fakeAuthority) {
	t.Helper()
	previous := openCredentialAuthority
	openCredentialAuthority = func() (CredentialAuthority, error) { return authority, nil }
	t.Cleanup(func() { openCredentialAuthority = previous })
}

// writeFakeGPG installs a fixture that models only the GPG protocol this
// package relies on, so custody behavior is testable without a real private key.
func writeFakeGPG(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	gpg := filepath.Join(bin, "gpg")
	script := `#!/bin/sh
homedir=""
previous=""
for arg in "$@"; do
  if [ "$previous" = "--homedir" ]; then homedir="$arg"; fi
  previous="$arg"
done
case " $* " in
  *" --quick-generate-key "*) : > "$homedir/generated" ; exit 0 ;;
  *" --import "*) cat > "$homedir/imported" ; : > "$homedir/generated" ; exit 0 ;;
  *" --list-secret-keys "*)
    if [ -f "$homedir/generated" ]; then
      printf 'sec::::::::\nfpr:::::::::TESTFINGERPRINT123\n'
    fi
    exit 0 ;;
  *" --export-secret-keys "*)
    # A passphrase-protected key only exports without a prompt under loopback
    # pinentry; model the real agent timeout so its absence is a test failure.
    case " $* " in
      *" --pinentry-mode loopback "*) ;;
      *)
        printf 'gpg: error receiving key from agent: Timeout - skipped\ngpg: WARNING: nothing exported\n' >&2
        exit 2 ;;
    esac
    printf -- '-----BEGIN PGP PRIVATE KEY BLOCK-----\nfake-private-key\n-----END PGP PRIVATE KEY BLOCK-----\n'
    exit 0 ;;
esac
exit 0
`
	if err := os.WriteFile(gpg, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
}

func TestGenerateLinuxKeyCustodiesMaterialInAuthority(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	withFakeAuthority(t, authority)

	h := &Handler{}
	result, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Passphrase:     "test-only-passphrase",
		Scenario:       "fixture",
		WorkingDirRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("generateLinuxKey() error = %v", err)
	}
	if result.Fingerprint != "TESTFINGERPRINT123" {
		t.Fatalf("Fingerprint = %q", result.Fingerprint)
	}
	if result.LogicalID != "vrooli/scenario-to-desktop/fixture" {
		t.Fatalf("LogicalID = %q", result.LogicalID)
	}
	identity := credentialauthority.Identity("vrooli/scenario-to-desktop/fixture")
	if got := authority.values[string(identity)+":"+"gpg-private-key"]; !strings.Contains(got, "PRIVATE KEY BLOCK") {
		t.Fatalf("private key not custodied: %q", got)
	}
	if got := authority.values[string(identity)+":"+"gpg-passphrase"]; got != "test-only-passphrase" {
		t.Fatalf("passphrase not custodied: %q", got)
	}
	if result.Homedir != "" {
		t.Fatalf("managed key must not return a durable homedir: %q", result.Homedir)
	}
}

func TestGenerateLinuxKeyUsesExplicitLogicalID(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	withFakeAuthority(t, authority)

	h := &Handler{}
	result, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Passphrase:     "test-only-passphrase",
		Scenario:       "fixture",
		LogicalID:      "vrooli/desktop-signing",
		WorkingDirRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("generateLinuxKey() error = %v", err)
	}
	if result.LogicalID != "vrooli/desktop-signing" {
		t.Fatalf("LogicalID = %q, want the explicit shared identity", result.LogicalID)
	}
	if result.Homedir != "" {
		t.Fatalf("managed key must not return a durable homedir: %q", result.Homedir)
	}
	identity := credentialauthority.Identity("vrooli/desktop-signing")
	if got := authority.values[string(identity)+":"+"gpg-private-key"]; !strings.Contains(got, "PRIVATE KEY BLOCK") {
		t.Fatalf("private key not custodied under explicit identity: %q", got)
	}
	if got := authority.values[string(identity)+":"+"gpg-passphrase"]; got != "test-only-passphrase" {
		t.Fatalf("passphrase not custodied under explicit identity: %q", got)
	}
}

func TestGenerateLinuxKeyReusesExistingManagedKey(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	identity := credentialauthority.Identity("vrooli/desktop-signing")
	authority.values[string(identity)+":"+"gpg-private-key"] = "SEEDED-MANAGED-KEY"
	authority.values[string(identity)+":"+"gpg-passphrase"] = "seeded-passphrase"
	withFakeAuthority(t, authority)

	h := &Handler{}
	result, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Scenario:       "second-scenario",
		LogicalID:      "vrooli/desktop-signing",
		WorkingDirRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("generateLinuxKey() error = %v", err)
	}
	if result.LogicalID != "vrooli/desktop-signing" {
		t.Fatalf("LogicalID = %q, want the shared identity", result.LogicalID)
	}
	if result.Fingerprint != "TESTFINGERPRINT123" {
		t.Fatalf("Fingerprint = %q", result.Fingerprint)
	}
	if got := authority.values[string(identity)+":"+"gpg-private-key"]; got != "SEEDED-MANAGED-KEY" {
		t.Fatalf("reuse must not rotate the custodied key, got %q", got)
	}
	if got := authority.values[string(identity)+":"+"gpg-passphrase"]; got != "seeded-passphrase" {
		t.Fatalf("reuse must not rotate the custodied passphrase, got %q", got)
	}
}

func TestGenerateLinuxKeyRejectsUnnamespacedLogicalID(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	withFakeAuthority(t, authority)

	h := &Handler{}
	if _, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Scenario:       "fixture",
		LogicalID:      "desktop-signing",
		WorkingDirRoot: t.TempDir(),
	}); err == nil {
		t.Fatal("expected an unnamespaced logical identity to be rejected")
	}
}

func TestGenerateLinuxKeyRejectsExternalKeyringWithLogicalID(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	withFakeAuthority(t, authority)

	h := &Handler{}
	if _, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Scenario:       "fixture",
		LogicalID:      "vrooli/desktop-signing",
		Homedir:        t.TempDir(),
		WorkingDirRoot: t.TempDir(),
	}); err == nil {
		t.Fatal("expected external keyring and managed identity to be rejected together")
	}
}

func TestGenerateLinuxKeyRefusesUnavailableAuthority(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	authority.available = false
	withFakeAuthority(t, authority)

	h := &Handler{}
	if _, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Scenario:       "fixture",
		WorkingDirRoot: t.TempDir(),
	}); err == nil {
		t.Fatal("expected an unavailable-authority error")
	}
}

func TestKeyGenerationParameterHelpers(t *testing.T) {
	if got := formatUID("Example", "example@test.invalid"); got != "Example <example@test.invalid>" {
		t.Fatalf("formatUID() = %q", got)
	}
	if got := formatUID("Example", ""); got != "Example" {
		t.Fatalf("formatUID name-only = %q", got)
	}
	if got := formatUID("", "example@test.invalid"); got != "example@test.invalid" {
		t.Fatalf("formatUID email-only = %q", got)
	}
	if got := valueOrDefault("", "rsa4096"); got != "rsa4096" {
		t.Fatalf("valueOrDefault default = %q", got)
	}
	if got := valueOrDefault("ed25519", "rsa4096"); got != "ed25519" {
		t.Fatalf("valueOrDefault supplied = %q", got)
	}
	if result, path, err := optionalExportPublicKey(context.Background(), t.TempDir(), "unused", generateLinuxKeyParams{}); err != nil || result != "" || path != "" {
		t.Fatalf("optionalExportPublicKey disabled = %q, %q, %v", result, path, err)
	}
}
