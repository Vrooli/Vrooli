package policy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"network-manager/internal/resolver"
	resolvermocks "network-manager/internal/resolver/mocks"

	"github.com/stretchr/testify/require"
)

type fakePolicySecretResolver struct {
	creds resolver.Credentials
	err   error
}

func (f fakePolicySecretResolver) ResolveAdGuardCredentials(context.Context, resolver.BackendConfig) (resolver.Credentials, error) {
	return f.creds, f.err
}

func TestAdGuardPolicyAdapterAppliesAndRollsBackUserRules(t *testing.T) {
	// [REQ:NM-P0-003] Approved global policy changes mutate AdGuard user rules only after capturing rollback state.
	fake := newAdGuardPolicyFake(t, adGuardPolicyFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		userRules:       []string{"||existing.invalid^"},
		protection:      true,
	})
	defer fake.server.Close()

	adapter := newTestAdGuardPolicyAdapter(t, fake.server)
	preview, err := adapter.Preview(context.Background(), Change{Target: "network", Action: "blocklist", Values: []string{"example.invalid"}})
	require.NoError(t, err)
	require.True(t, preview.RollbackSupported)

	applied, err := adapter.Apply(context.Background(), Change{Target: "network", Action: "blocklist", Values: []string{"example.invalid"}})
	require.NoError(t, err)
	require.True(t, applied.RollbackSupported)
	require.NotEmpty(t, applied.RollbackHandle)
	require.Equal(t, []string{"||existing.invalid^", "||example.invalid^"}, fake.userRules)
	require.Contains(t, applied.Effects, "Applied 1 AdGuard Home user-defined filtering rule(s).")

	rolledBack, err := adapter.Rollback(context.Background(), Change{RollbackHandle: applied.RollbackHandle})
	require.NoError(t, err)
	require.Contains(t, rolledBack.Effects, "Restored previous AdGuard Home user-defined filtering rules.")
	require.Equal(t, []string{"||existing.invalid^"}, fake.userRules)
}

func TestAdGuardPolicyAdapterAllowlistUsesAdGuardExceptionRule(t *testing.T) {
	// [REQ:NM-P0-003] Allowlist values are translated into AdGuard exception rules with rollback support.
	fake := newAdGuardPolicyFake(t, adGuardPolicyFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      true,
	})
	defer fake.server.Close()

	adapter := newTestAdGuardPolicyAdapter(t, fake.server)
	applied, err := adapter.Apply(context.Background(), Change{Target: "global", Action: "allowlist", Values: []string{"school.example"}})
	require.NoError(t, err)
	require.True(t, applied.RollbackSupported)
	require.Equal(t, []string{"@@||school.example^"}, fake.userRules)
}

func TestAdGuardPolicyAdapterPausesAndRollsBackProtection(t *testing.T) {
	// [REQ:NM-P0-003] Protection pause uses AdGuard's rollback-capable protection endpoint.
	fake := newAdGuardPolicyFake(t, adGuardPolicyFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      true,
	})
	defer fake.server.Close()

	adapter := newTestAdGuardPolicyAdapter(t, fake.server)
	applied, err := adapter.Apply(context.Background(), Change{Target: "network", Action: "pause_filtering", Values: []string{"duration=15m"}})
	require.NoError(t, err)
	require.False(t, fake.protection)
	require.Equal(t, uint64(900000), fake.lastProtectionDuration)

	_, err = adapter.Rollback(context.Background(), Change{RollbackHandle: applied.RollbackHandle})
	require.NoError(t, err)
	require.True(t, fake.protection)
}

func TestAdGuardPolicyAdapterKeepsClientTargetsUnsupportedUntilMapped(t *testing.T) {
	// [REQ:NM-P0-003] Client/group mutations stay fail-closed until AdGuard client identity mapping exists.
	fake := newAdGuardPolicyFake(t, adGuardPolicyFakeConfig{
		requireUser:     "admin",
		requirePassword: "secret",
		protection:      true,
	})
	defer fake.server.Close()

	adapter := newTestAdGuardPolicyAdapter(t, fake.server)
	preview, err := adapter.Preview(context.Background(), Change{Target: "group:kids", Action: "blocklist", Values: []string{"example.invalid"}})
	require.NoError(t, err)
	require.False(t, preview.RollbackSupported)
	require.Contains(t, preview.Effects[0], "needs client mapping first")

	_, err = adapter.Apply(context.Background(), Change{Target: "group:kids", Action: "blocklist", Values: []string{"example.invalid"}})
	require.ErrorIs(t, err, ErrUnsupported)
}

func newTestAdGuardPolicyAdapter(t *testing.T, server *httptest.Server) AdGuardResolverPolicyAdapter {
	t.Helper()
	repo := resolvermocks.NewRepository()
	_, err := repo.SaveBackend(context.Background(), resolver.BackendConfig{
		Backend:       resolver.AdGuardHomeBackend,
		BaseURL:       server.URL,
		Username:      "admin",
		CredentialRef: "vrooli/adguard-home",
	})
	require.NoError(t, err)
	return AdGuardResolverPolicyAdapter{
		Backends: repo,
		Secrets:  fakePolicySecretResolver{creds: resolver.Credentials{Username: "admin", Password: "secret"}},
		HTTP:     server.Client().Transport,
	}
}

type adGuardPolicyFakeConfig struct {
	requireUser     string
	requirePassword string
	userRules       []string
	protection      bool
}

type adGuardPolicyFake struct {
	server                 *httptest.Server
	userRules              []string
	protection             bool
	lastProtectionDuration uint64
}

func newAdGuardPolicyFake(t *testing.T, cfg adGuardPolicyFakeConfig) *adGuardPolicyFake {
	t.Helper()
	fake := &adGuardPolicyFake{
		userRules:  append([]string(nil), cfg.userRules...),
		protection: cfg.protection,
	}
	mux := http.NewServeMux()
	requireAuth := func(w http.ResponseWriter, r *http.Request) bool {
		user, password, ok := r.BasicAuth()
		if !ok || user != cfg.requireUser || password != cfg.requirePassword {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc(adGuardFilteringStatusEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		require.Equal(t, http.MethodGet, r.Method)
		writePolicyJSON(t, w, map[string]any{"user_rules": fake.userRules})
	})
	mux.HandleFunc(adGuardFilteringRulesEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		require.Equal(t, http.MethodPost, r.Method)
		var req adGuardRulesUpdate
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		fake.userRules = append([]string(nil), req.Rules...)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc(adGuardStatusEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		require.Equal(t, http.MethodGet, r.Method)
		writePolicyJSON(t, w, map[string]any{"protection_status": fake.protection})
	})
	mux.HandleFunc(adGuardProtectionEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		require.Equal(t, http.MethodPost, r.Method)
		var req adGuardProtectionUpdate
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		fake.protection = req.Enabled
		fake.lastProtectionDuration = req.Duration
		w.WriteHeader(http.StatusOK)
	})
	fake.server = httptest.NewServer(mux)
	return fake
}

func writePolicyJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
