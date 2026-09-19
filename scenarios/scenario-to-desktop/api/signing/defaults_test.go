package signing

import (
	"context"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"scenario-to-desktop-api/signing/types"
)

func TestDefaultManagedLinuxConfigReturnsNilWhenKeyAbsent(t *testing.T) {
	writeFakeGPG(t)
	withFakeAuthority(t, newFakeAuthority())

	config, err := DefaultManagedLinuxConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("DefaultManagedLinuxConfig() error = %v", err)
	}
	if config != nil {
		t.Fatalf("expected no config when the shared key is absent, got %#v", config)
	}
}

func TestDefaultManagedLinuxConfigReturnsNilWhenAuthorityUnavailable(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	authority.available = false
	withFakeAuthority(t, authority)

	config, err := DefaultManagedLinuxConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("DefaultManagedLinuxConfig() error = %v", err)
	}
	if config != nil {
		t.Fatalf("expected no config when the authority is unavailable, got %#v", config)
	}
}

func TestDefaultManagedLinuxConfigMaterializesSharedKey(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	identity := credentialauthority.Identity(DefaultSharedLogicalID)
	authority.values[string(identity)+":"+types.DefaultPrivateKeyField] = "SEEDED-MANAGED-KEY"
	authority.values[string(identity)+":"+types.DefaultPassphraseField] = "seeded-passphrase"
	withFakeAuthority(t, authority)

	config, err := DefaultManagedLinuxConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("DefaultManagedLinuxConfig() error = %v", err)
	}
	if config == nil || !config.Enabled || config.Linux == nil {
		t.Fatalf("expected an enabled Linux config, got %#v", config)
	}
	if config.Linux.GPGKeyID != "TESTFINGERPRINT123" {
		t.Fatalf("fingerprint = %q", config.Linux.GPGKeyID)
	}
	if config.Linux.ManagedKey == nil || config.Linux.ManagedKey.LogicalID != DefaultSharedLogicalID {
		t.Fatalf("managed identity = %#v", config.Linux.ManagedKey)
	}
	if config.Linux.GPGPassphraseEnv != types.DefaultPassphraseEnvVar {
		t.Fatalf("passphrase env = %q", config.Linux.GPGPassphraseEnv)
	}
}
