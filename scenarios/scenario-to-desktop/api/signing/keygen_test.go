package signing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateLinuxKeyUsesIsolatedGPGHomeAndReturnsFingerprint(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	// The fixture models only the GPG protocol this package relies on. It lets
	// the orchestration contract be tested without creating a real private key.
	gpg := filepath.Join(bin, "gpg")
	script := `#!/bin/sh
homedir=""
previous=""
for arg in "$@"; do
  if [ "$previous" = "--homedir" ]; then homedir="$arg"; break; fi
  previous="$arg"
done
case " $* " in
  *" --quick-generate-key "*) : > "$homedir/generated" ; exit 0 ;;
  *" --list-secret-keys "*)
    if [ -f "$homedir/generated" ]; then
      printf 'sec::::::::\nfpr:::::::::TESTFINGERPRINT123\n'
    fi
    exit 0 ;;
esac
exit 0
`
	if err := os.WriteFile(gpg, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	h := &Handler{}
	result, err := h.generateLinuxKey(context.Background(), generateLinuxKeyParams{
		Name:           "Example Publisher",
		Email:          "publisher@example.test",
		Passphrase:     "test-only-passphrase",
		Scenario:       "fixture",
		WorkingDirRoot: root,
	})
	if err != nil {
		t.Fatalf("generateLinuxKey() error = %v", err)
	}
	if result.Fingerprint != "TESTFINGERPRINT123" {
		t.Fatalf("Fingerprint = %q", result.Fingerprint)
	}
	if info, err := os.Stat(result.Homedir); err != nil || !info.IsDir() {
		t.Fatalf("homedir was not created safely: %v", err)
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
