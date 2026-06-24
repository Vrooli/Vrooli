package resolver

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVaultSecretResolverReadsPasswordAndUsername(t *testing.T) {
	// [REQ:NM-P0-002] AdGuard credentials are resolved through resource-vault from a secret reference.
	var calls [][]string
	resolver := &VaultSecretResolver{
		Command:  "resource-vault",
		LookPath: func(string) (string, error) { return "/usr/bin/resource-vault", nil },
		Run: func(_ context.Context, name string, args ...string) (string, error) {
			calls = append(calls, append([]string{name}, args...))
			switch args[5] {
			case "password":
				return "secret-password\n", nil
			case "username":
				return "operator\n", nil
			default:
				return "", errors.New("unexpected key")
			}
		},
	}

	creds, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{
		TokenRef: "secret/resources/adguard-home/admin",
	})
	require.NoError(t, err)
	require.Equal(t, Credentials{Username: "operator", Password: "secret-password"}, creds)
	require.Equal(t, [][]string{
		{"resource-vault", "content", "get", "--path", "secret/resources/adguard-home/admin", "--key", "password", "--format", "raw"},
		{"resource-vault", "content", "get", "--path", "secret/resources/adguard-home/admin", "--key", "username", "--format", "raw"},
	}, calls)
}

func TestVaultSecretResolverUsesConfiguredUsername(t *testing.T) {
	// [REQ:NM-P0-002] Stored resolver username avoids unnecessary secret reads while password remains secret-ref backed.
	var keys []string
	resolver := &VaultSecretResolver{
		LookPath: func(string) (string, error) { return "/usr/bin/resource-vault", nil },
		Run: func(_ context.Context, _ string, args ...string) (string, error) {
			keys = append(keys, args[5])
			return "secret-password\n", nil
		},
	}

	creds, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{
		Username: "admin",
		TokenRef: "secret/resources/adguard-home/admin",
	})
	require.NoError(t, err)
	require.Equal(t, Credentials{Username: "admin", Password: "secret-password"}, creds)
	require.True(t, reflect.DeepEqual([]string{"password"}, keys))
}

func TestVaultSecretResolverFailsClosedForMissingPassword(t *testing.T) {
	// [REQ:NM-P0-002] Missing AdGuard credentials fail closed rather than falling back to empty auth.
	resolver := &VaultSecretResolver{
		LookPath: func(string) (string, error) { return "/usr/bin/resource-vault", nil },
		Run: func(context.Context, string, ...string) (string, error) {
			return "", &vaultCLIError{Stderr: "No value found at secret/resources/adguard-home/admin/password", Err: exec.ErrNotFound}
		},
	}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{TokenRef: "secret/resources/adguard-home/admin"})
	require.ErrorIs(t, err, ErrSecretMissing)
}

func TestVaultSecretResolverRejectsNonSecretRef(t *testing.T) {
	// [REQ:NM-P0-002] Plaintext-looking token refs are rejected before any vault lookup.
	resolver := &VaultSecretResolver{}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{TokenRef: "password123"})
	require.ErrorContains(t, err, "unsupported secret reference")
	require.NotContains(t, err.Error(), "password123")
}
