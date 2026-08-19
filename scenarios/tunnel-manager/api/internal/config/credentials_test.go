package config_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"tunnel-manager/internal/config"
)

type fakeAuthority struct{ values map[string]string }

func (f *fakeAuthority) key(_ credentialauthority.Identity, field string) string { return field }
func (f *fakeAuthority) Resolve(id credentialauthority.Identity, field string) (string, error) {
	value := f.values[f.key(id, field)]
	if value == "" {
		return "", errors.New("credential is not configured")
	}
	return value, nil
}

func (f *fakeAuthority) Put(id credentialauthority.Identity, field, value string) error {
	f.values[f.key(id, field)] = value
	return nil
}

func (f *fakeAuthority) Delete(id credentialauthority.Identity, field string) error {
	delete(f.values, f.key(id, field))
	return nil
}

func (f *fakeAuthority) Status(id credentialauthority.Identity, field string) credentialauthority.Status {
	_, err := f.Resolve(id, field)
	return credentialauthority.Status{Configured: err == nil, ProviderState: "available"}
}
func (*fakeAuthority) Provider() string { return "credential-authority" }

func TestCloudflareCredentialStoreUsesCanonicalAuthority(t *testing.T) {
	ctx := context.Background()
	authority := &fakeAuthority{values: map[string]string{}}
	store, err := config.NewCloudflareCredentialStore(config.CredentialStoreOptions{Authority: authority})
	require.NoError(t, err)

	status, err := store.Save(ctx, config.CredentialUpdate{AccountID: "acct", TunnelID: "tun", APIToken: "tok"})
	require.NoError(t, err)
	// The write-only credentials endpoint intentionally cannot claim complete
	// tunnel readiness until bootstrap has derived and stored the connector
	// token as well.
	require.False(t, status.Ready)
	require.Contains(t, status.MissingFields, "CLOUDFLARE_CONNECTOR_TOKEN")
	require.Equal(t, "credential-authority", status.Source)
	require.NotContains(t, statusString(status), "tok-value-marker")

	resolved, err := store.Resolve(ctx)
	require.NoError(t, err)
	require.Equal(t, "acct", resolved.AccountID)
	require.Equal(t, "tun", resolved.TunnelID)
	require.Equal(t, "tok", resolved.APIToken)
	require.ElementsMatch(t, []string{"CLOUDFLARE_CONNECTOR_TOKEN"}, resolved.Missing)
}

func TestCloudflareCredentialStoreDoesNotUseEnvironmentOverrides(t *testing.T) {
	ctx := context.Background()
	authority := &fakeAuthority{values: map[string]string{
		"cloudflare-account-id": "authority-account",
		"cloudflare-tunnel-id":  "authority-tunnel",
		"cloudflare-api-token":  "authority-token",
	}}
	store, err := config.NewCloudflareCredentialStore(config.CredentialStoreOptions{Authority: authority})
	require.NoError(t, err)
	resolved, err := store.Resolve(ctx)
	require.NoError(t, err)
	require.Equal(t, "authority-account", resolved.AccountID)
	require.Equal(t, "authority-tunnel", resolved.TunnelID)
	require.Equal(t, "authority-token", resolved.APIToken)
}

func TestCloudflareCredentialStoreDeleteUsesCanonicalFields(t *testing.T) {
	ctx := context.Background()
	authority := &fakeAuthority{values: map[string]string{}}
	store, err := config.NewCloudflareCredentialStore(config.CredentialStoreOptions{Authority: authority})
	require.NoError(t, err)
	_, err = store.Save(ctx, config.CredentialUpdate{AccountID: "acct", TunnelID: "tun", APIToken: "tok"})
	require.NoError(t, err)
	status, err := store.Delete(ctx, []string{"api_token"})
	require.NoError(t, err)
	require.False(t, status.Ready)
	require.ElementsMatch(t, []string{"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_CONNECTOR_TOKEN"}, status.MissingFields)
}

func statusString(status config.CredentialStatus) string {
	var b strings.Builder
	b.WriteString(status.Source)
	b.WriteString(status.Ref)
	for _, field := range status.Fields {
		b.WriteString(field.Name)
		b.WriteString(field.Source)
		b.WriteString(field.Ref)
	}
	return b.String()
}
