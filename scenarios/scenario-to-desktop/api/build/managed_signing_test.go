package build

import (
	"context"
	"errors"
	"os"
	"testing"

	signingtypes "scenario-to-desktop-api/signing/types"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type fakeManagedAuthority struct {
	available bool
	values    map[string]string
}

func (f *fakeManagedAuthority) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	value, ok := f.values[string(identity)+":"+field]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (f *fakeManagedAuthority) Availability() error {
	if !f.available {
		return errors.New("credential provider is unavailable")
	}
	return nil
}

func withManagedSigningSeams(t *testing.T, authority *fakeManagedAuthority) *[]string {
	t.Helper()
	var imported []string
	previousAuthority := openSigningAuthority
	previousImport := importManagedSigningKey
	openSigningAuthority = func() (managedKeyAuthority, error) { return authority, nil }
	importManagedSigningKey = func(_ context.Context, homedir, armored string) error {
		imported = append(imported, armored)
		return os.WriteFile(homedir+"/imported", []byte(armored), 0o600)
	}
	t.Cleanup(func() {
		openSigningAuthority = previousAuthority
		importManagedSigningKey = previousImport
	})
	return &imported
}

func TestManagedSigningEnvMaterializesAndCleansUp(t *testing.T) {
	authority := &fakeManagedAuthority{
		available: true,
		values: map[string]string{
			"vrooli/desktop-signing:gpg-private-key": "armored-key",
			"vrooli/desktop-signing:gpg-passphrase":  "generated-passphrase",
		},
	}
	imported := withManagedSigningSeams(t, authority)

	env, cleanup, err := managedSigningEnv(&signingtypes.SigningConfig{
		Enabled: true,
		Linux:   &signingtypes.LinuxSigningConfig{GPGKeyID: "ABC123", ManagedKey: &signingtypes.ManagedSigningKey{LogicalID: "vrooli/desktop-signing"}},
	})
	if err != nil {
		t.Fatalf("managedSigningEnv() error = %v", err)
	}
	homedir := env[signingtypes.DefaultManagedHomedirEnv]
	if homedir == "" {
		t.Fatalf("missing %s in %#v", signingtypes.DefaultManagedHomedirEnv, env)
	}
	if _, statErr := os.Stat(homedir); statErr != nil {
		t.Fatalf("materialized homedir missing: %v", statErr)
	}
	if env[signingtypes.DefaultPassphraseEnvVar] != "generated-passphrase" {
		t.Fatalf("passphrase env = %q", env[signingtypes.DefaultPassphraseEnvVar])
	}
	if len(*imported) != 1 || (*imported)[0] != "armored-key" {
		t.Fatalf("imported keys = %#v", *imported)
	}

	cleanup()
	if _, statErr := os.Stat(homedir); !os.IsNotExist(statErr) {
		t.Fatalf("cleanup did not remove homedir: %v", statErr)
	}
}

func TestManagedSigningEnvSkipsWithoutManagedKey(t *testing.T) {
	env, _, err := managedSigningEnv(&signingtypes.SigningConfig{Enabled: true, Linux: &signingtypes.LinuxSigningConfig{GPGKeyID: "ABC123"}})
	if err != nil || env != nil {
		t.Fatalf("managedSigningEnv() = %#v, %v", env, err)
	}
}

func TestManagedSigningEnvFailsClosedWithoutAuthority(t *testing.T) {
	withManagedSigningSeams(t, &fakeManagedAuthority{available: false})
	if _, _, err := managedSigningEnv(&signingtypes.SigningConfig{
		Enabled: true,
		Linux:   &signingtypes.LinuxSigningConfig{GPGKeyID: "ABC123", ManagedKey: &signingtypes.ManagedSigningKey{LogicalID: "vrooli/desktop-signing"}},
	}); err == nil {
		t.Fatal("expected an unavailable-authority error")
	}
}

func TestResolveManagedSigningEnvForPlatformSkipsNonLinux(t *testing.T) {
	env, cleanup, err := resolveManagedSigningEnvForPlatform("demo", "mac")
	if err != nil || env != nil {
		t.Fatalf("non-linux platform = %#v, %v", env, err)
	}
	cleanup()
}
