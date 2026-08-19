package config_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"tunnel-manager/internal/config"
)

type fakeBootstrapAPI struct {
	accounts     []config.CloudflareAccount
	tunnels      []config.CloudflareTunnel
	created      config.CloudflareTunnel
	connector    string
	create       bool
	verifyErr    error
	accountsErr  error
	tunnelsErr   error
	createErr    error
	connectorErr error
}

func (f *fakeBootstrapAPI) VerifyToken(context.Context, string) error { return f.verifyErr }
func (f *fakeBootstrapAPI) ListAccounts(context.Context, string) ([]config.CloudflareAccount, error) {
	return f.accounts, f.accountsErr
}

func (f *fakeBootstrapAPI) ListTunnels(context.Context, string, string) ([]config.CloudflareTunnel, error) {
	return f.tunnels, f.tunnelsErr
}

func (f *fakeBootstrapAPI) CreateTunnel(context.Context, string, string, string) (config.CloudflareTunnel, error) {
	f.create = true
	return f.created, f.createErr
}

func (f *fakeBootstrapAPI) ConnectorToken(context.Context, string, string, string) (string, error) {
	return f.connector, f.connectorErr
}

func TestBootstrapCloudflareAdoptsNamedTunnelAndWritesCompleteSet(t *testing.T) {
	authority := &fakeAuthority{values: map[string]string{}}
	api := &fakeBootstrapAPI{
		accounts:  []config.CloudflareAccount{{ID: "account-1"}},
		tunnels:   []config.CloudflareTunnel{{ID: "tunnel-1", Name: "Vrooli"}},
		connector: "connector-token",
	}

	result, err := config.BootstrapCloudflare(context.Background(), api, authority, config.BootstrapRequest{
		APIToken:   "api-token",
		TunnelName: "vrooli",
	})

	require.NoError(t, err)
	require.Equal(t, config.BootstrapResult{AccountID: "account-1", TunnelID: "tunnel-1", Adopted: true, Written: true}, result)
	require.False(t, api.create)
	require.Equal(t, "api-token", resolveBootstrapValue(t, authority, "cloudflare-api-token"))
	require.Equal(t, "account-1", resolveBootstrapValue(t, authority, "cloudflare-account-id"))
	require.Equal(t, "tunnel-1", resolveBootstrapValue(t, authority, "cloudflare-tunnel-id"))
	require.Equal(t, "connector-token", resolveBootstrapValue(t, authority, "cloudflare-connector-token"))
}

func TestBootstrapCloudflareUsesExistingConnectorTokenForAdoption(t *testing.T) {
	payload, err := json.Marshal(map[string]string{"a": "account-from-connector", "t": "tunnel-from-connector"})
	require.NoError(t, err)
	connector := base64.RawStdEncoding.EncodeToString(payload)
	authority := &fakeAuthority{values: map[string]string{
		"cloudflare-connector-token": connector,
	}}
	api := &fakeBootstrapAPI{
		accounts:  []config.CloudflareAccount{{ID: "account-from-connector"}},
		tunnels:   []config.CloudflareTunnel{{ID: "tunnel-from-connector", Name: "existing"}},
		connector: "new-connector",
	}

	result, err := config.BootstrapCloudflare(context.Background(), api, authority, config.BootstrapRequest{APIToken: "api-token"})

	require.NoError(t, err)
	require.True(t, result.Adopted)
	require.Equal(t, "account-from-connector", result.AccountID)
	require.Equal(t, "tunnel-from-connector", result.TunnelID)
	require.False(t, api.create)
}

func TestBootstrapCloudflareDryRunDoesNotCreateOrWrite(t *testing.T) {
	authority := &fakeAuthority{values: map[string]string{}}
	api := &fakeBootstrapAPI{
		accounts: []config.CloudflareAccount{{ID: "account-1"}},
		created:  config.CloudflareTunnel{ID: "tunnel-1", Name: "vrooli"},
	}

	result, err := config.BootstrapCloudflare(context.Background(), api, authority, config.BootstrapRequest{
		APIToken:   "api-token",
		TunnelName: "vrooli",
		DryRun:     true,
	})

	require.NoError(t, err)
	require.Equal(t, config.BootstrapResult{AccountID: "account-1"}, result)
	require.False(t, api.create)
	require.Empty(t, authority.values)
}

func TestBootstrapCloudflareCreatesWhenNamedTunnelIsAbsent(t *testing.T) {
	authority := &fakeAuthority{values: map[string]string{}}
	api := &fakeBootstrapAPI{
		accounts:  []config.CloudflareAccount{{ID: "account-1"}},
		created:   config.CloudflareTunnel{ID: "tunnel-new", Name: "vrooli"},
		connector: "connector-token",
	}
	result, err := config.BootstrapCloudflare(context.Background(), api, authority, config.BootstrapRequest{APIToken: "api-token", TunnelName: "vrooli"})
	require.NoError(t, err)
	require.True(t, api.create)
	require.Equal(t, config.BootstrapResult{AccountID: "account-1", TunnelID: "tunnel-new", Created: true, Written: true}, result)
}

func TestBootstrapCloudflareRejectsPermissionAndAmbiguousAccounts(t *testing.T) {
	authority := &fakeAuthority{values: map[string]string{}}
	_, err := config.BootstrapCloudflare(context.Background(), &fakeBootstrapAPI{verifyErr: errors.New("permission denied")}, authority, config.BootstrapRequest{APIToken: "api-token"})
	require.ErrorContains(t, err, "verify Cloudflare API token")

	_, err = config.BootstrapCloudflare(context.Background(), &fakeBootstrapAPI{accounts: []config.CloudflareAccount{{ID: "a"}, {ID: "b"}}}, authority, config.BootstrapRequest{APIToken: "api-token"})
	require.ErrorContains(t, err, "supply an account id")
}

type failingBootstrapAuthority struct {
	*fakeAuthority
	failField string
}

func (f *failingBootstrapAuthority) Put(id credentialauthority.Identity, field, value string) error {
	if field == f.failField {
		return errors.New("simulated authority write failure")
	}
	return f.fakeAuthority.Put(id, field, value)
}

func TestBootstrapCloudflareRollsBackPartialAuthorityWrite(t *testing.T) {
	authority := &failingBootstrapAuthority{
		fakeAuthority: &fakeAuthority{values: map[string]string{"cloudflare-api-token": "old-token"}},
		failField:     "cloudflare-account-id",
	}
	api := &fakeBootstrapAPI{
		accounts:  []config.CloudflareAccount{{ID: "account-1"}},
		tunnels:   []config.CloudflareTunnel{{ID: "tunnel-1", Name: "vrooli"}},
		connector: "connector-token",
	}

	_, err := config.BootstrapCloudflare(context.Background(), api, authority, config.BootstrapRequest{
		APIToken:   "api-token",
		TunnelName: "vrooli",
	})

	require.Error(t, err)
	require.Equal(t, "old-token", resolveBootstrapValue(t, authority.fakeAuthority, "cloudflare-api-token"))
	for _, field := range []string{"cloudflare-account-id", "cloudflare-tunnel-id", "cloudflare-connector-token"} {
		_, resolveErr := authority.Resolve(credentialauthority.Identity(""), field)
		require.Error(t, resolveErr, field)
	}
}

func resolveBootstrapValue(t *testing.T, authority *fakeAuthority, field string) string {
	t.Helper()
	value, err := authority.Resolve(credentialauthority.Identity(""), field)
	require.NoError(t, err)
	return value
}
