package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type fakeCredentialStore struct {
	values map[string]string
	errors map[string]error
}

func (f fakeCredentialStore) Require(identity credentialauthority.Identity, field string) (string, error) {
	if err := f.errors[string(identity)+":"+field]; err != nil {
		return "", err
	}
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

func TestCredentialAuthorityResolverRejectsMissingStore(t *testing.T) {
	resolver := &CredentialAuthorityResolver{}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: "vrooli/adguard-home"})
	require.ErrorContains(t, err, "credential authority is not configured")
}

func TestCredentialAuthorityResolverWrapsPasswordReadFailure(t *testing.T) {
	readErr := errors.New("credential provider unavailable")
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{errors: map[string]error{
		"vrooli/adguard-home:password": readErr,
	}}}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: "vrooli/adguard-home"})
	require.ErrorIs(t, err, readErr)
	require.ErrorContains(t, err, "read AdGuard Home password")
}

func TestCredentialAuthorityResolverUsesDefaultUsernameWhenUnconfigured(t *testing.T) {
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{values: map[string]string{
		"vrooli/adguard-home:password": " secret-password ",
	}}}

	creds, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: " vrooli/adguard-home "})
	require.NoError(t, err)
	require.Equal(t, Credentials{Username: "admin", Password: "secret-password"}, creds)
}

func TestCredentialAuthorityResolverWrapsUsernameReadFailure(t *testing.T) {
	readErr := errors.New("credential provider unavailable")
	resolver := &CredentialAuthorityResolver{Authority: fakeCredentialStore{
		values: map[string]string{"vrooli/adguard-home:password": "secret-password"},
		errors: map[string]error{"vrooli/adguard-home:username": readErr},
	}}

	_, err := resolver.ResolveAdGuardCredentials(context.Background(), BackendConfig{CredentialRef: "vrooli/adguard-home"})
	require.ErrorIs(t, err, readErr)
	require.ErrorContains(t, err, "read AdGuard Home username")
}

func TestNormalizeCredentialRefDefaultsBlankReference(t *testing.T) {
	require.Equal(t, "vrooli/adguard-home", normalizeCredentialRef(" \t "))
}

func TestUnavailableCredentialStoreReturnsInitializationError(t *testing.T) {
	initializationErr := errors.New("credential authority initialization failed")
	_, err := (unavailableCredentialStore{err: initializationErr}).Require(credentialauthority.Identity("vrooli/adguard-home"), "password")
	require.ErrorIs(t, err, initializationErr)
}
