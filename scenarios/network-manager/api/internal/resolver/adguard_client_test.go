package resolver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeSecretResolver struct {
	creds Credentials
	err   error
}

func (f fakeSecretResolver) ResolveAdGuardCredentials(context.Context, BackendConfig) (Credentials, error) {
	return f.creds, f.err
}

func TestResourceBackedAdGuardClientHealthy(t *testing.T) {
	// [REQ:NM-P0-002] Resolver health is verified from AdGuard control API evidence when credentials resolve.
	server := newAdGuardFake(t, adGuardFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      boolPtr(true),
		upstreams:       []string{"https://dns.example/dns-query"},
		queryLogEnabled: boolPtr(false),
	})
	defer server.Close()

	client := ResourceBackedAdGuardClient{
		Secrets: fakeSecretResolver{creds: Credentials{Username: "admin", Password: "secret"}},
		HTTP:    server.Client().Transport,
	}

	status, err := client.Check(context.Background(), BackendConfig{BaseURL: server.URL, TokenRef: "secret/resources/adguard-home/admin"})
	require.NoError(t, err)
	require.Equal(t, "healthy", status.Status)
	require.True(t, status.FilteringEnabled)
	require.Equal(t, []string{"https://dns.example/dns-query"}, status.Upstreams)
	require.Contains(t, status.Checks, "Control status endpoint returned successfully.")
	require.Contains(t, status.Checks, "Query log is disabled according to /control/querylog/config.")
}

func TestResourceBackedAdGuardClientMapsAuthFailure(t *testing.T) {
	// [REQ:NM-P0-002] Bad AdGuard credentials surface as auth_failed without leaking the attempted password.
	server := newAdGuardFake(t, adGuardFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      boolPtr(true),
	})
	defer server.Close()

	client := ResourceBackedAdGuardClient{
		Secrets: fakeSecretResolver{creds: Credentials{Username: "admin", Password: "wrong"}},
		HTTP:    server.Client().Transport,
	}

	status, err := client.Check(context.Background(), BackendConfig{BaseURL: server.URL, TokenRef: "secret/resources/adguard-home/admin"})
	require.NoError(t, err)
	require.Equal(t, "auth_failed", status.Status)
	require.Contains(t, status.Warnings, "AdGuard Home rejected the configured credentials.")
	require.NotContains(t, status.Warnings[0], "wrong")
}

func TestResourceBackedAdGuardClientDegradesWhenQueryLogEnabled(t *testing.T) {
	// [REQ:NM-P0-008] Query-log-enabled posture is degraded and never exposes query entries.
	server := newAdGuardFake(t, adGuardFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      boolPtr(true),
		queryLogEnabled: boolPtr(true),
	})
	defer server.Close()

	client := ResourceBackedAdGuardClient{
		Secrets: fakeSecretResolver{creds: Credentials{Username: "admin", Password: "secret"}},
		HTTP:    server.Client().Transport,
	}

	status, err := client.Check(context.Background(), BackendConfig{BaseURL: server.URL, TokenRef: "secret/resources/adguard-home/admin"})
	require.NoError(t, err)
	require.Equal(t, "degraded", status.Status)
	require.Contains(t, status.Warnings, "Query log is enabled; Network Manager will not expose query-level DNS history.")
}

func TestResourceBackedAdGuardClientPreviewUpstreams(t *testing.T) {
	// [REQ:NM-P0-002] Upstream preview reads current AdGuard state but does not mutate resolver configuration.
	server := newAdGuardFake(t, adGuardFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      boolPtr(true),
		upstreams:       []string{"1.1.1.1", "8.8.8.8"},
	})
	defer server.Close()

	client := ResourceBackedAdGuardClient{
		Secrets: fakeSecretResolver{creds: Credentials{Username: "admin", Password: "secret"}},
		HTTP:    server.Client().Transport,
	}

	changes, err := client.PreviewUpstreams(context.Background(), BackendConfig{BaseURL: server.URL, TokenRef: "secret/resources/adguard-home/admin"}, []string{"1.1.1.1", "9.9.9.9"})
	require.NoError(t, err)
	require.Contains(t, changes, "Current upstreams: 1.1.1.1, 8.8.8.8")
	require.Contains(t, changes, "Requested upstreams: 1.1.1.1, 9.9.9.9")
	require.Contains(t, changes, "Added: 9.9.9.9")
	require.Contains(t, changes, "Removed: 8.8.8.8")
}

func TestResourceBackedAdGuardClientPersistentUpdateUnsupported(t *testing.T) {
	// [REQ:NM-P0-003] Persistent resolver writes remain fail-closed until rollback-backed policy support lands.
	client := ResourceBackedAdGuardClient{}

	_, _, err := client.UpdateUpstreams(context.Background(), BackendConfig{}, []string{"1.1.1.1"})
	require.ErrorIs(t, err, ErrClientUnsupported)
}

func TestServiceUsesResourceBackedClientWithSecretResolver(t *testing.T) {
	// [REQ:NM-P0-002] Service health can verify a stored AdGuard backend through the resource-backed client.
	server := newAdGuardFake(t, adGuardFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      boolPtr(true),
		queryLogEnabled: boolPtr(false),
	})
	defer server.Close()

	repo := newFakeRepo()
	_, err := repo.SaveBackend(context.Background(), BackendConfig{
		Backend:  AdGuardHomeBackend,
		BaseURL:  server.URL,
		Username: "admin",
		TokenRef: "secret/resources/adguard-home/admin",
	})
	require.NoError(t, err)
	svc := NewService(Config{Repo: repo, Client: ResourceBackedAdGuardClient{
		Secrets: fakeSecretResolver{creds: Credentials{Username: "admin", Password: "secret"}},
		HTTP:    server.Client().Transport,
	}})

	status, checks, err := svc.Health(context.Background())
	require.NoError(t, err)
	require.Equal(t, "healthy", status.Status)
	require.True(t, status.FilteringEnabled)
	require.NotEmpty(t, checks)
}

type adGuardFakeConfig struct {
	requireUser     string
	requirePassword string
	protection      *bool
	upstreams       []string
	queryLogEnabled *bool
}

func newAdGuardFake(t *testing.T, cfg adGuardFakeConfig) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	requireAuth := func(w http.ResponseWriter, r *http.Request) bool {
		user, password, ok := r.BasicAuth()
		if !ok || user != cfg.requireUser || password != cfg.requirePassword {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc(adGuardStatusEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		writeJSON(t, w, map[string]any{"version": "v0.107.77", "protection_status": cfg.protection})
	})
	mux.HandleFunc(adGuardDNSInfoEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		writeJSON(t, w, map[string]any{"upstream_dns": cfg.upstreams})
	})
	mux.HandleFunc(adGuardQueryLogConfigEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		if cfg.queryLogEnabled == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(t, w, map[string]any{"enabled": cfg.queryLogEnabled})
	})
	mux.HandleFunc(adGuardLegacyQueryLogEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		writeJSON(t, w, map[string]any{"enabled": false})
	})
	return httptest.NewServer(mux)
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}

func boolPtr(value bool) *bool {
	return &value
}
