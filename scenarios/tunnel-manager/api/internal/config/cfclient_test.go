package config_test

import (
	"testing"

	"tunnel-manager/internal/config"

	"github.com/stretchr/testify/require"
)

func TestResolveCloudflareEnvPrefersCanonicalNames(t *testing.T) {
	env := map[string]string{
		"CLOUDFLARE_ACCOUNT_ID": "acct-canonical",
		"CF_ACCOUNT_ID":         "acct-legacy",
		"CLOUDFLARE_TUNNEL_ID":  "tunnel-canonical",
		"CF_TUNNEL_ID":          "tunnel-legacy",
		"CLOUDFLARE_API_TOKEN":  "token-canonical",
		"CF_API_TOKEN":          "token-legacy",
	}

	got := config.ResolveCloudflareEnv(func(key string) string { return env[key] })

	require.Equal(t, "acct-canonical", got.AccountID)
	require.Equal(t, "tunnel-canonical", got.TunnelID)
	require.Equal(t, "token-canonical", got.APIToken)
	require.Equal(t, "env:CLOUDFLARE_*", got.Source)
	require.Equal(t, "env:CLOUDFLARE_API_TOKEN", got.TokenRef)
	require.Empty(t, got.Missing)
}

func TestResolveCloudflareEnvFallsBackToLegacyNames(t *testing.T) {
	env := map[string]string{
		"CF_ACCOUNT_ID": "acct-legacy",
		"CF_TUNNEL_ID":  "tunnel-legacy",
		"CF_API_TOKEN":  "token-legacy",
	}

	got := config.ResolveCloudflareEnv(func(key string) string { return env[key] })

	require.Equal(t, "acct-legacy", got.AccountID)
	require.Equal(t, "tunnel-legacy", got.TunnelID)
	require.Equal(t, "token-legacy", got.APIToken)
	require.Equal(t, "env:CF_*", got.Source)
	require.Equal(t, "env:CF_API_TOKEN", got.TokenRef)
	require.Empty(t, got.Missing)
}

func TestResolveCloudflareEnvReportsCanonicalMissingFields(t *testing.T) {
	env := map[string]string{"CLOUDFLARE_ACCOUNT_ID": "acct"}

	got := config.ResolveCloudflareEnv(func(key string) string { return env[key] })

	require.Equal(t, []string{"CLOUDFLARE_TUNNEL_ID", "CLOUDFLARE_API_TOKEN"}, got.Missing)
	require.Equal(t, "env:CLOUDFLARE_*", got.Source)
	require.Empty(t, got.TokenRef)
}
