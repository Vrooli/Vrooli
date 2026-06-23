package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeAdGuardClient struct {
	checkStatus  ClientStatus
	checkErr     error
	updateStatus ClientStatus
	updateErr    error
}

func (f fakeAdGuardClient) Check(context.Context, BackendConfig) (ClientStatus, error) {
	return f.checkStatus, f.checkErr
}

func (f fakeAdGuardClient) PreviewUpstreams(_ context.Context, _ BackendConfig, upstreams []string) ([]string, error) {
	return []string{"preview " + joinUpstreams(upstreams)}, nil
}

func (f fakeAdGuardClient) UpdateUpstreams(context.Context, BackendConfig, []string) (ClientStatus, []string, error) {
	return f.updateStatus, []string{"updated"}, f.updateErr
}

type fakeRepo struct {
	backends  map[string]BackendConfig
	upstreams map[string][]string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{backends: map[string]BackendConfig{}, upstreams: map[string][]string{}}
}

func (r *fakeRepo) SaveBackend(_ context.Context, cfg BackendConfig) (BackendConfig, error) {
	r.backends[cfg.Backend] = cfg
	return cfg, nil
}

func (r *fakeRepo) GetBackend(_ context.Context, backend string) (BackendConfig, error) {
	cfg, ok := r.backends[backend]
	if !ok {
		return BackendConfig{}, ErrNotFound
	}
	return cfg, nil
}

func (r *fakeRepo) UpdateUpstreams(_ context.Context, backend string, upstreams []string) error {
	r.upstreams[backend] = append([]string(nil), upstreams...)
	return nil
}

func (r *fakeRepo) GetUpstreams(_ context.Context, backend string) ([]string, error) {
	return append([]string(nil), r.upstreams[backend]...), nil
}

func TestConfigureAdGuardHomeDryRunDoesNotPersist(t *testing.T) {
	// [REQ:NM-P0-002] Dry-run configuration validates shape without storing resolver credentials.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Client: fakeAdGuardClient{}})

	status, steps, err := svc.ConfigureAdGuardHome(context.Background(), "http://adguard.local", "admin", "secret://adguard/token", true)
	require.NoError(t, err)
	require.Equal(t, "dry_run", status.Status)
	require.NotEmpty(t, steps)

	_, err = repo.GetBackend(context.Background(), AdGuardHomeBackend)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestConfigureAdGuardHomePersistsSecretReferenceOnly(t *testing.T) {
	// [REQ:NM-P0-002] Resolver backend config stores token references, not plaintext tokens.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Client: fakeAdGuardClient{checkStatus: ClientStatus{
		Status:           "healthy",
		FilteringEnabled: true,
		Upstreams:        []string{"https://dns.example/dns-query"},
		Checks:           []string{"reachable"},
	}}})

	status, _, err := svc.ConfigureAdGuardHome(context.Background(), "http://adguard.local/", "admin", "secret://adguard/token", false)
	require.NoError(t, err)
	require.Equal(t, "healthy", status.Status)
	require.True(t, status.FilteringEnabled)
	require.Equal(t, "http://adguard.local", status.BaseURL)

	cfg, err := repo.GetBackend(context.Background(), AdGuardHomeBackend)
	require.NoError(t, err)
	require.Equal(t, "secret://adguard/token", cfg.TokenRef)
	require.NotContains(t, cfg.TokenRef, "password")
}

func TestHealthMapsClientStates(t *testing.T) {
	// [REQ:NM-P0-002] AdGuard client states are surfaced without claiming unsupported filtering.
	for _, tc := range []struct {
		name      string
		client    fakeAdGuardClient
		wantState string
		wantWarn  string
		wantErr   error
	}{
		{name: "reachable", client: fakeAdGuardClient{checkStatus: ClientStatus{Status: "healthy", FilteringEnabled: true, Checks: []string{"reachable"}}}, wantState: "healthy"},
		{name: "auth failure", client: fakeAdGuardClient{checkStatus: ClientStatus{Status: "auth_failed", Warnings: []string{"authentication failed"}, Checks: []string{"auth failed"}}}, wantState: "auth_failed", wantWarn: "authentication failed"},
		{name: "degraded", client: fakeAdGuardClient{checkStatus: ClientStatus{Status: "degraded", Warnings: []string{"query API slow"}, Checks: []string{"slow"}}}, wantState: "degraded", wantWarn: "query API slow"},
		{name: "unsupported", client: fakeAdGuardClient{checkErr: ErrClientUnsupported}, wantErr: ErrClientUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			_, err := repo.SaveBackend(context.Background(), BackendConfig{Backend: AdGuardHomeBackend, BaseURL: "http://adguard.local", TokenRef: "secret://adguard/token"})
			require.NoError(t, err)
			svc := NewService(Config{Repo: repo, Client: tc.client})

			status, _, err := svc.Health(context.Background())
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantState, status.Status)
			if tc.wantWarn != "" {
				require.Contains(t, status.Warnings, tc.wantWarn)
			}
		})
	}
}

func TestConfigureAdGuardHomeRequiresSecretReference(t *testing.T) {
	// [REQ:NM-P0-002] Plaintext or missing token inputs are rejected before persistence.
	svc := NewService(Config{Repo: newFakeRepo(), Client: fakeAdGuardClient{}})

	_, _, err := svc.ConfigureAdGuardHome(context.Background(), "http://adguard.local", "admin", "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "token_ref")
}

func TestUpdateUpstreamsRequiresConfiguredClientSupport(t *testing.T) {
	// [REQ:NM-P0-002] Persistent upstream writes fail closed when the adapter cannot apply them.
	repo := newFakeRepo()
	_, err := repo.SaveBackend(context.Background(), BackendConfig{Backend: AdGuardHomeBackend, BaseURL: "http://adguard.local", TokenRef: "secret://adguard/token"})
	require.NoError(t, err)
	svc := NewService(Config{Repo: repo, Client: fakeAdGuardClient{updateErr: ErrClientUnsupported}})

	_, _, err = svc.UpdateUpstreams(context.Background(), []string{"1.1.1.1"}, false)
	require.ErrorIs(t, err, ErrClientUnsupported)

	status, changes, err := svc.UpdateUpstreams(context.Background(), []string{"1.1.1.1"}, true)
	require.NoError(t, err)
	require.Equal(t, "unknown", status.Status)
	require.Contains(t, changes[0], "preview")
}

func TestStatusReturnsClientError(t *testing.T) {
	repo := newFakeRepo()
	_, err := repo.SaveBackend(context.Background(), BackendConfig{Backend: AdGuardHomeBackend, BaseURL: "http://adguard.local", TokenRef: "secret://adguard/token"})
	require.NoError(t, err)
	svc := NewService(Config{Repo: repo, Client: fakeAdGuardClient{checkErr: errors.New("boom")}})

	_, err = svc.Status(context.Background())
	require.ErrorContains(t, err, "boom")
}
