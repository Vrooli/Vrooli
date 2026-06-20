package config_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"tunnel-manager/internal/config"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/secrets"
	repocontract "github.com/vrooli/repo-contract-go"
)

func TestCloudflareCredentialStoreSavesScenarioSecrets(t *testing.T) {
	ctx := context.Background()
	store := newCredentialStore(t, nil)

	status, err := store.Save(ctx, config.CredentialUpdate{
		AccountID: "acct-file",
		TunnelID:  "tun-file",
		APIToken:  "tok-file",
	})
	require.NoError(t, err)
	require.True(t, status.Ready)
	require.Equal(t, "file:scenario", status.Source)
	require.Equal(t, "file:scenario:cloudflare.api_token", status.Ref)
	require.NotContains(t, statusString(status), "tok-file")

	resolved, err := store.Resolve(ctx)
	require.NoError(t, err)
	require.Equal(t, "acct-file", resolved.AccountID)
	require.Equal(t, "tun-file", resolved.TunnelID)
	require.Equal(t, "tok-file", resolved.APIToken)
	require.Empty(t, resolved.Missing)
	require.Equal(t, "file:scenario", resolved.Source)
}

func TestCloudflareCredentialStoreEnvOverridesSavedValues(t *testing.T) {
	ctx := context.Background()
	env := map[string]string{
		"CLOUDFLARE_ACCOUNT_ID": "acct-env",
		"CLOUDFLARE_TUNNEL_ID":  "tun-env",
		"CLOUDFLARE_API_TOKEN":  "tok-env",
	}
	store := newCredentialStore(t, env)

	_, err := store.Save(ctx, config.CredentialUpdate{
		AccountID: "acct-file",
		TunnelID:  "tun-file",
		APIToken:  "tok-file",
	})
	require.NoError(t, err)

	resolved, err := store.Resolve(ctx)
	require.NoError(t, err)
	require.Equal(t, "acct-env", resolved.AccountID)
	require.Equal(t, "tun-env", resolved.TunnelID)
	require.Equal(t, "tok-env", resolved.APIToken)
	require.Equal(t, "env:CLOUDFLARE_*", resolved.Source)

	status, err := store.Status(ctx)
	require.NoError(t, err)
	for _, field := range status.Fields {
		require.False(t, field.Writable, "env-sourced field %s should be read-only from Tunnel Manager", field.Name)
	}
	require.NotContains(t, statusString(status), "tok-env")
}

func TestCloudflareCredentialStoreFallsBackToSharedUserSecrets(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	userStore, err := secrets.NewUserStore(secrets.Config{HomeDir: home, EnvLookup: func(string) string { return "" }})
	require.NoError(t, err)
	require.NoError(t, userStore.SaveKey("cloudflare.account_id", "acct-user"))
	require.NoError(t, userStore.SaveKey("cloudflare.tunnel_id", "tun-user"))
	require.NoError(t, userStore.SaveKey("cloudflare.api_token", "tok-user"))

	store, err := config.NewCloudflareCredentialStore(config.CredentialStoreOptions{
		HomeDir:   home,
		EnvLookup: func(string) string { return "" },
	})
	require.NoError(t, err)

	resolved, err := store.Resolve(ctx)
	require.NoError(t, err)
	require.Equal(t, "acct-user", resolved.AccountID)
	require.Equal(t, "tun-user", resolved.TunnelID)
	require.Equal(t, "tok-user", resolved.APIToken)
	require.Equal(t, "file:user", resolved.Source)
}

func TestCloudflareCredentialStoreDeleteClearsScenarioSecret(t *testing.T) {
	ctx := context.Background()
	store := newCredentialStore(t, nil)
	_, err := store.Save(ctx, config.CredentialUpdate{
		AccountID: "acct-file",
		TunnelID:  "tun-file",
		APIToken:  "tok-file",
	})
	require.NoError(t, err)

	status, err := store.Delete(ctx, []string{"api_token"})
	require.NoError(t, err)
	require.False(t, status.Ready)
	require.ElementsMatch(t, []string{"CLOUDFLARE_API_TOKEN"}, status.MissingFields)

	resolved, err := store.Resolve(ctx)
	require.NoError(t, err)
	require.Empty(t, resolved.APIToken)
	require.Equal(t, []string{"CLOUDFLARE_API_TOKEN"}, resolved.Missing)
}

func newCredentialStore(t *testing.T, env map[string]string) config.CredentialStore {
	t.Helper()
	home := t.TempDir()
	store, err := config.NewCloudflareCredentialStore(config.CredentialStoreOptions{
		HomeDir: home,
		EnvLookup: func(key string) string {
			if env == nil {
				return ""
			}
			return env[key]
		},
	})
	require.NoError(t, err)
	path, err := repocontract.UserScenarioPlaintextSecretsPath(home, "tunnel-manager")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
	return store
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
