package signing

import (
	"context"
	"fmt"
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"scenario-to-desktop-api/signing/types"
)

// DefaultSharedLogicalID is the credential-authority identity of the shared
// publisher signing key. One key signs every desktop app so users trust a
// single publisher fingerprint and a rotation has one blast radius.
const DefaultSharedLogicalID = "vrooli/desktop-signing"

// DefaultManagedLinuxConfig returns the managed Linux signing configuration for
// the shared publisher key when the credential authority holds it, and publishes
// the public half to the scenario so verifiers can obtain the trust anchor.
//
// It returns (nil, nil) when the key is not configured or the authority is
// unavailable: automatic signing applies only when it is available, and a
// caller must not treat absence as an error. A non-nil error means the key is
// configured but could not be materialized; callers should surface it rather
// than silently shipping an unsigned build.
func DefaultManagedLinuxConfig(ctx context.Context, scenario string) (*SigningConfig, error) {
	identity, err := credentialauthority.ParseIdentity(DefaultSharedLogicalID)
	if err != nil {
		return nil, fmt.Errorf("invalid shared signing identity %q: %w", DefaultSharedLogicalID, err)
	}

	authority, err := openCredentialAuthority()
	if err != nil {
		return nil, nil
	}
	if err := authority.Availability(); err != nil {
		return nil, nil
	}
	if !authority.Status(identity, types.DefaultPrivateKeyField).Configured ||
		!authority.Status(identity, types.DefaultPassphraseField).Configured {
		return nil, nil
	}

	secret, err := authority.Resolve(identity, types.DefaultPrivateKeyField)
	if err != nil {
		return nil, fmt.Errorf("resolve shared signing key %s: %w", identity, err)
	}

	homedir, err := os.MkdirTemp("", "vrooli-signing-gnupg-")
	if err != nil {
		return nil, fmt.Errorf("create signing homedir: %w", err)
	}
	defer func() { _ = os.RemoveAll(homedir) }()
	_ = os.Chmod(homedir, 0o700)

	if err := importSecretKey(ctx, homedir, secret); err != nil {
		return nil, err
	}
	fingerprint, err := latestFingerprint(homedir)
	if err != nil {
		return nil, fmt.Errorf("read shared signing key: %w", err)
	}

	if strings.TrimSpace(scenario) != "" {
		if public, pubErr := exportPublicKey(ctx, homedir, fingerprint); pubErr == nil {
			_, _ = writePublicKey(scenario, public)
		}
	}

	return &SigningConfig{
		SchemaVersion: types.SchemaVersion,
		Enabled:       true,
		Linux: &LinuxSigningConfig{
			GPGKeyID:         fingerprint,
			GPGPassphraseEnv: types.DefaultPassphraseEnvVar,
			ManagedKey:       &ManagedSigningKey{LogicalID: DefaultSharedLogicalID},
		},
	}, nil
}
