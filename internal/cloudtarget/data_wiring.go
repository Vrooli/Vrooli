package cloudtarget

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// Reason codes added by the data verbs. They are the recoverypoint codes so
// the cloud side, the target and the certification matrix share one name.
const (
	CodeRecoveryPointCorrupt  = recoverypoint.CodeRecoveryPointCorrupt
	CodeRecoveryKeyMissing    = recoverypoint.CodeRecoveryKeyMissing
	CodeRestoreTargetNotClean = recoverypoint.CodeRestoreTargetNotClean
	CodeBackupProviderMissing = recoverypoint.CodeProviderUnavailable
)

const recoveryPointsDirName = "recovery-points"

// RecoveryPointDir returns the canonical directory of a recovery point.
func (s *Store) RecoveryPointDir(deploymentID, recoveryPointID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	if err := validIdentifier("recovery point id", recoveryPointID); err != nil {
		return "", err
	}
	return filepath.Join(dir, recoveryPointsDirName, recoveryPointID), nil
}

// DataDeps are the seams the data verbs run through. Zero values select
// production wiring: the OS process runner, the credential authority for
// the recovery key and PostgreSQL credentials, and the built-in providers.
type DataDeps struct {
	Runner    recoverypoint.Runner
	Keys      recoverypoint.KeyResolver
	Providers recoverypoint.Registry
	Sealer    recoverypoint.Sealer
	// PostgresEnv resolves the PG* environment for a SQL binding. The
	// default reads the postgres resource credential through the authority
	// and never places it in argv.
	PostgresEnv func(ctx context.Context, b recoverypoint.Binding) ([]string, error)
}

func (d DataDeps) runner() recoverypoint.Runner {
	if d.Runner != nil {
		return d.Runner
	}
	return EnvRunner{}
}

func (d DataDeps) keys() recoverypoint.KeyResolver {
	if d.Keys != nil {
		return d.Keys
	}
	return AuthorityKeys{}
}

func (d DataDeps) providers() recoverypoint.Registry {
	if d.Providers != nil {
		return d.Providers
	}
	env := d.PostgresEnv
	if env == nil {
		env = AuthorityPostgresEnv
	}
	return recoverypoint.Registry{
		recoverypoint.ProviderObjectStore: recoverypoint.ObjectStore{},
		recoverypoint.ProviderPostgres:    recoverypoint.Postgres{Runner: d.runner(), CredentialEnv: env},
	}
}

// EnvRunner executes argv with an explicit environment through the shared
// shell boundary. The environment is the child's whole environment, so a
// credential passed here never mixes with the operator's shell.
type EnvRunner struct{}

// Run implements recoverypoint.Runner.
func (EnvRunner) Run(ctx context.Context, tool string, argv []string, env []string) ([]byte, error) {
	return shell.CombinedOutput(shell.Spec{Context: ctx, Name: tool, Args: argv, Env: env})
}

// AuthorityKeys resolves a recovery key reference of the form
// `<logical-identity>:<field>` through the credential authority. The
// authority's backend is the operator's credential store, which lives with
// the operator, not on the deployment target being protected.
type AuthorityKeys struct{}

// ResolveKey implements recoverypoint.KeyResolver.
func (AuthorityKeys) ResolveKey(_ context.Context, keyRef string) ([]byte, error) {
	identity, field, err := SplitCredentialRef(keyRef)
	if err != nil {
		return nil, err
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", recoverypoint.ErrKeyUnavailable, err)
	}
	value, err := authority.Resolve(identity, field)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", recoverypoint.ErrKeyUnavailable, err)
	}
	return []byte(value), nil
}

// SplitCredentialRef parses `logical_id:field`.
func SplitCredentialRef(ref string) (credentialauthority.Identity, string, error) {
	idx := strings.LastIndex(ref, ":")
	if idx <= 0 || idx == len(ref)-1 {
		return "", "", fmt.Errorf("%w: reference %q is not logical_id:field", recoverypoint.ErrKeyUnavailable, ref)
	}
	identity, err := credentialauthority.ParseIdentity(ref[:idx])
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", recoverypoint.ErrKeyUnavailable, err)
	}
	return identity, ref[idx+1:], nil
}

// PostgresCredentialIdentity is the postgres resource's declared credential
// logical identity (resources/postgres/resource.json credentials.descriptors).
const PostgresCredentialIdentity = "vrooli/postgres"

// AuthorityPostgresEnv resolves the managed PostgreSQL password through the
// credential authority and returns it as PGPASSWORD for the child process.
func AuthorityPostgresEnv(_ context.Context, _ recoverypoint.Binding) ([]string, error) {
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, err
	}
	password, err := authority.Resolve(credentialauthority.Identity(PostgresCredentialIdentity), "password")
	if err != nil {
		return nil, err
	}
	return []string{"PGPASSWORD=" + password}, nil
}
