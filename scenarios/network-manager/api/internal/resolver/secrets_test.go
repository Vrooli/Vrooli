package resolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type fakeCredentialStore struct {
	values map[string]string
}

func (f fakeCredentialStore) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	value, ok := f.values[string(identity)+":"+field]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func TestCredentialAuthorityResolverReadsPasswordAndUsername(t *testing.T) {
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{values: map[string]string{
		"vrooli/adguard-home:password": "secret-password",
		"vrooli/adguard-home:username": "operator",
	}}}

	creds, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: "vrooli/adguard-home"})
	require.NoError(t, err)
	require.Equal(t, Credentials{Username: "operator", Password: "secret-password"}, creds)
}

func TestCredentialAuthorityResolverUsesConfiguredUsername(t *testing.T) {
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{values: map[string]string{
		"vrooli/adguard-home:password": "secret-password",
	}}}

	creds, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{Username: "admin", CredentialRef: "vrooli/adguard-home"})
	require.NoError(t, err)
	require.Equal(t, Credentials{Username: "admin", Password: "secret-password"}, creds)
}

func TestCredentialAuthorityResolverFailsClosedForMissingPassword(t *testing.T) {
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{values: map[string]string{}}}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: "vrooli/adguard-home"})
	require.ErrorIs(t, err, ErrSecretMissing)
}

func TestCredentialAuthorityResolverRejectsInvalidRef(t *testing.T) {
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{}}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: "password123"})
	require.ErrorContains(t, err, "credential logical identity must be namespaced")
	require.NotContains(t, err.Error(), "password123")
}
