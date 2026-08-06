// Package repoctx resolves a Vrooli destination name into everything needed to
// run a kopia command against that repository: the registry entry, the kopia
// global args that address its config file/cache, and the secret env overlay
// (KOPIA_PASSWORD + S3 creds) sourced from the credential authority. It is shared by the repo,
// snapshot, policy, and maintenance command groups so secret-injection and
// repo-addressing logic lives in exactly one place.
package repoctx

import (
	"context"
	"fmt"
	"strings"

	"resource-kopia/cli/internal/credentials"
	"resource-kopia/cli/internal/registry"

	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// Env var names kopia reads for the passphrase and S3 credentials. Secrets are
// passed exclusively through these — never as argv flags.
const (
	EnvPassword     = "KOPIA_PASSWORD"
	EnvAWSAccessKey = "AWS_ACCESS_KEY_ID"
	EnvAWSSecretKey = "AWS_SECRET_ACCESS_KEY"
)

// Target bundles a resolved repository's addressing and secret env.
type Target struct {
	Entry registry.Entry
	// Env is the secret overlay for kexec.Call.Env (KOPIA_PASSWORD, AWS_*).
	Env map[string]string
}

// GlobalArgs returns the kopia global flags that address this repository's
// config file. --config-file is a kopia global (persistent) flag and must
// prefix the subcommand verb so concurrent repositories never share a config.
// The cache directory is a create/connect-time flag (persisted in the config),
// not a per-command global, so it is not emitted here.
func (t Target) GlobalArgs() []string {
	return []string{"--config-file", t.Entry.ConfigFile}
}

// Resolver loads registry entries and sources every repository credential from
// the credential authority and has no provider-shaped dependency.
type Resolver struct {
	Registry    *registry.Registry
	Credentials credentials.Store
}

// Resolve returns the Target for a registered repository, failing closed if the
// repo is unknown or its authority credentials are absent.
func (r Resolver) Resolve(ctx context.Context, name string) (Target, error) {
	entry, found, err := r.Registry.Get(name)
	if err != nil {
		return Target{}, err
	}
	if !found {
		return Target{}, fmt.Errorf("unknown repository %q; create or connect it first", name)
	}
	return r.targetFor(ctx, entry)
}

// targetFor builds the secret env overlay for an entry.
func (r Resolver) targetFor(ctx context.Context, entry registry.Entry) (Target, error) {
	identity, err := kopiaregistry.PassphraseIdentity(entry.Name)
	if err != nil {
		return Target{}, err
	}
	if r.Credentials == nil {
		return Target{}, fmt.Errorf("credential authority is unavailable")
	}
	passphrase, err := r.Credentials.Resolve(identity, kopiaregistry.PassphraseField)
	if err != nil {
		return Target{}, fmt.Errorf("read repository passphrase for %q: %w", entry.Name, err)
	}
	if strings.TrimSpace(passphrase) == "" {
		return Target{}, fmt.Errorf("repository passphrase for %q is not configured", entry.Name)
	}
	env := map[string]string{EnvPassword: passphrase}

	if entry.Backend == registry.BackendS3 {
		creds, found, err := credentials.S3CredentialsFor(r.Credentials, entry.Name)
		if err != nil {
			return Target{}, err
		}
		if !found || !creds.Valid() {
			return Target{}, fmt.Errorf("S3 credentials for repository %q are not configured in the credential authority", entry.Name)
		}
		env[EnvAWSAccessKey] = creds.AccessKeyID
		env[EnvAWSSecretKey] = creds.SecretAccessKey
	}

	return Target{Entry: entry, Env: env}, nil
}
