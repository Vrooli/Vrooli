package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"network-manager/internal/resolver"

	"github.com/stretchr/testify/require"
)

type fakeResolverRepo struct {
	cfg resolver.BackendConfig
	err error
}

func (r fakeResolverRepo) SaveBackend(context.Context, resolver.BackendConfig) (resolver.BackendConfig, error) {
	return resolver.BackendConfig{}, errors.New("not implemented")
}

func (r fakeResolverRepo) GetBackend(context.Context, string) (resolver.BackendConfig, error) {
	if r.err != nil {
		return resolver.BackendConfig{}, r.err
	}
	return r.cfg, nil
}

func (r fakeResolverRepo) UpdateUpstreams(context.Context, string, []string) error {
	return errors.New("not implemented")
}

func (r fakeResolverRepo) GetUpstreams(context.Context, string) ([]string, error) {
	return nil, errors.New("not implemented")
}

type fakeInventorySecretResolver struct {
	creds resolver.Credentials
	err   error
}

func (f fakeInventorySecretResolver) ResolveAdGuardCredentials(context.Context, resolver.BackendConfig) (resolver.Credentials, error) {
	return f.creds, f.err
}

func TestAdGuardClientDiscoverySourceImportsConfiguredClients(t *testing.T) {
	// [REQ:NM-P0-004] Configured AdGuard clients become resolver identity evidence without query-log data.
	server := newInventoryAdGuardFake(t, "admin", "secret", map[string]any{
		"clients": []map[string]any{
			{"name": "Laptop", "ids": []string{"client-laptop", "192.0.2.10"}},
			{"name": "Phone", "ids": []string{"00:11:22:33:44:55"}},
		},
	})
	defer server.Close()

	source := AdGuardClientDiscoverySource{
		Backends: fakeResolverRepo{cfg: resolver.BackendConfig{
			Backend:       resolver.AdGuardHomeBackend,
			BaseURL:       server.URL,
			CredentialRef: "vrooli/adguard-home",
		}},
		Secrets: fakeInventorySecretResolver{creds: resolver.Credentials{Username: "admin", Password: "secret"}},
		HTTP:    server.Client().Transport,
	}

	observations, findings, err := source.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, observations, 2)
	require.Contains(t, findings[0], "without query-level DNS log data")
	require.Equal(t, "Laptop", observations[0].Hostname)
	require.Equal(t, "192.0.2.10", observations[0].IPAddress)
	require.Equal(t, "client-laptop", observations[0].ResolverClientID)
	require.Equal(t, "adguard-client:client-laptop", observations[0].StableID)
	require.Equal(t, "00:11:22:33:44:55", observations[1].MACAddress)
}

func TestAdGuardClientDiscoverySourceKeepsAutoIPOnlyWeak(t *testing.T) {
	// [REQ:NM-P0-004] Automatically discovered IP-only clients stay weak observations.
	server := newInventoryAdGuardFake(t, "admin", "secret", map[string]any{
		"auto_clients": []map[string]any{
			{"name": "dhcp-host", "ip": "192.0.2.20"},
			{"name": "localhost", "ip": "127.0.0.1"},
			{"name": "ip6-allnodes", "ip": "ff02::1"},
			{"name": "ip6-localnet", "ip": "fe00::"},
		},
	})
	defer server.Close()

	source := AdGuardClientDiscoverySource{
		Backends: fakeResolverRepo{cfg: resolver.BackendConfig{
			Backend:       resolver.AdGuardHomeBackend,
			BaseURL:       server.URL,
			CredentialRef: "vrooli/adguard-home",
		}},
		Secrets: fakeInventorySecretResolver{creds: resolver.Credentials{Username: "admin", Password: "secret"}},
		HTTP:    server.Client().Transport,
	}

	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Source: source})
	devices, findings, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Contains(t, strings.Join(findings, "\n"), "Imported 1 AdGuard Home client")
	require.Equal(t, "dhcp-host", devices[0].Hostname)
	require.Equal(t, "192.0.2.20", devices[0].IPAddress)
	require.Empty(t, devices[0].ResolverClientID)
	require.Equal(t, "low", devices[0].IdentityConfidence)
	require.Contains(t, strings.Join(devices[0].Notes, "\n"), "ambiguous")
}

func TestAdGuardClientDiscoverySourceAuthFailureDoesNotInventDevices(t *testing.T) {
	// [REQ:NM-P0-004] AdGuard auth failures fail closed and return persisted inventory only.
	server := newInventoryAdGuardFake(t, "admin", "secret", map[string]any{
		"clients": []map[string]any{{"name": "Laptop", "ids": []string{"client-laptop"}}},
	})
	defer server.Close()

	source := AdGuardClientDiscoverySource{
		Backends: fakeResolverRepo{cfg: resolver.BackendConfig{
			Backend:       resolver.AdGuardHomeBackend,
			BaseURL:       server.URL,
			CredentialRef: "vrooli/adguard-home",
		}},
		Secrets: fakeInventorySecretResolver{creds: resolver.Credentials{Username: "admin", Password: "wrong"}},
		HTTP:    server.Client().Transport,
	}

	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Source: source})
	devices, findings, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Empty(t, devices)
	require.Contains(t, strings.Join(findings, "\n"), "rejected the configured credentials")
	require.NotContains(t, strings.Join(findings, "\n"), "wrong")
}

func newInventoryAdGuardFake(t *testing.T, user, password string, clients map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(adGuardClientsEndpoint, func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPassword, ok := r.BasicAuth()
		if !ok || gotUser != user || gotPassword != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(clients))
	})
	return httptest.NewServer(mux)
}
