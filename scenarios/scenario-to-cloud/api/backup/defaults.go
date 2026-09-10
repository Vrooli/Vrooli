package backup

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"scenario-to-cloud/domain"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// AuthorityKeys resolves a recovery key reference of the form
// `<logical-identity>:<field>` through the operator's credential authority.
// The authority's backend is the operator's credential store, which lives
// with the operator and not on the deployment target being protected, so a
// replacement host restore needs only the reference.
type AuthorityKeys struct{}

// ResolveKey implements recoverypoint.KeyResolver.
func (AuthorityKeys) ResolveKey(_ context.Context, keyRef string) ([]byte, error) {
	idx := strings.LastIndex(keyRef, ":")
	if idx <= 0 || idx == len(keyRef)-1 {
		return nil, fmt.Errorf("%w: reference %q is not logical_id:field", recoverypoint.ErrKeyUnavailable, keyRef)
	}
	identity, err := credentialauthority.ParseIdentity(keyRef[:idx])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", recoverypoint.ErrKeyUnavailable, err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", recoverypoint.ErrKeyUnavailable, err)
	}
	value, err := authority.Require(identity, keyRef[idx+1:])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", recoverypoint.ErrKeyUnavailable, err)
	}
	return []byte(value), nil
}

// PostgresCredentialIdentity is the postgres resource's declared credential
// logical identity (resources/postgres/resource.json credentials.descriptors).
const PostgresCredentialIdentity = "vrooli/postgres"

// AuthorityPostgresEnv resolves the managed PostgreSQL password through the
// credential authority and returns it as PGPASSWORD for the child process;
// it never enters argv.
func AuthorityPostgresEnv(_ context.Context, _ recoverypoint.Binding) ([]string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	password, err := authority.Require(credentialauthority.Identity(PostgresCredentialIdentity), "password")
	if err != nil {
		return nil, err
	}
	return []string{"PGPASSWORD=" + password}, nil
}

// OSRunner runs a tool with an explicit environment.
type OSRunner struct{}

// Run implements recoverypoint.Runner.
func (OSRunner) Run(ctx context.Context, tool string, argv []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, tool, argv...) //nolint:gosec // argv built by the provider from typed fields
	cmd.Env = env
	return cmd.CombinedOutput()
}

// DefaultProviders is the production registry: the resource owner's
// database-native PostgreSQL tooling, the embedded SQLite engine and the
// object-store inventory provider.
func DefaultProviders() recoverypoint.Registry {
	return recoverypoint.Registry{
		domain.BackupProviderPostgres:    recoverypoint.Postgres{Runner: OSRunner{}, CredentialEnv: AuthorityPostgresEnv},
		domain.BackupProviderSQLite:      SQLite{},
		domain.BackupProviderObjectStore: recoverypoint.ObjectStore{},
	}
}
