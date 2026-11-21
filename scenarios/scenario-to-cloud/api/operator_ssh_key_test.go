package main

import (
	"errors"
	"path/filepath"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"scenario-to-cloud/credentials"
)

func scopedOperatorAuthority(t *testing.T) *credentialauthority.Authority {
	t.Helper()
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "credentials.enc"), "test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	return authority
}

// An unconfigured key means "use ambient SSH identities"; an absent credential
// store is a fault that must surface instead of reading as "no key".
func TestResolveOperatorSSHKeyPathSeparatesUnconfiguredFromProviderFaults(t *testing.T) {
	if path, err := resolveOperatorSSHKeyPath(scopedOperatorAuthority(t)); err != nil || path != "" {
		t.Fatalf("unconfigured key = %q, %v; want empty path and no error", path, err)
	}

	absent, err := credentialauthority.Unavailable("no store on this host")
	if err != nil {
		t.Fatal(err)
	}
	if path, err := resolveOperatorSSHKeyPath(absent); path != "" || !errors.Is(err, credentialauthority.ErrProviderAbsent) {
		t.Fatalf("absent provider = %q, %v; want empty path and ErrProviderAbsent", path, err)
	}

	configured := scopedOperatorAuthority(t)
	identity, err := credentialauthority.ParseIdentity(credentials.SSHKeyDescriptor.LogicalID)
	if err != nil {
		t.Fatal(err)
	}
	if err := configured.Put(identity, credentials.SSHKeyDescriptor.Field, " /home/operator/.ssh/id_ed25519 "); err != nil {
		t.Fatal(err)
	}
	if path, err := resolveOperatorSSHKeyPath(configured); err != nil || path != "/home/operator/.ssh/id_ed25519" {
		t.Fatalf("configured key = %q, %v; want the trimmed path", path, err)
	}
}
