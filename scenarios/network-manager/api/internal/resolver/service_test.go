package resolver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeAdGuardClient struct {
	checkStatus  ClientStatus
	checkErr     error
	updateStatus ClientStatus
	updateErr    error
}

type fakeDNSInspector struct {
	inspection DNSInspection
}

func (f fakeDNSInspector) InspectHostDNS(context.Context, string) DNSInspection {
	return f.inspection
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

func TestConfigureAdGuardHomeDefaultsFromResourceExports(t *testing.T) {
	// [REQ:NM-P0-002] Resource-managed AdGuard exports can satisfy the default backend shape without plaintext secrets.
	t.Setenv("ADGUARD_HOME_BASE_URL", "http://localhost:3000")
	t.Setenv("ADGUARD_HOME_USERNAME", "admin")
	t.Setenv("ADGUARD_HOME_CREDENTIAL_REF", "secret/resources/adguard-home/admin")
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Client: fakeAdGuardClient{checkStatus: ClientStatus{
		Status:           "configured_unverified",
		FilteringEnabled: false,
		Checks:           []string{"reachable"},
	}}})

	status, steps, err := svc.ConfigureAdGuardHome(context.Background(), "", "", "", true)
	require.NoError(t, err)
	require.Equal(t, "dry_run", status.Status)
	require.Equal(t, "http://localhost:3000", status.BaseURL)
	require.NotEmpty(t, steps)

	status, _, err = svc.ConfigureAdGuardHome(context.Background(), "", "", "", false)
	require.NoError(t, err)
	require.Equal(t, "configured_unverified", status.Status)

	cfg, err := repo.GetBackend(context.Background(), AdGuardHomeBackend)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:3000", cfg.BaseURL)
	require.Equal(t, "admin", cfg.Username)
	require.Equal(t, "secret/resources/adguard-home/admin", cfg.TokenRef)
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

func TestAdGuardRolloutKeepsRouterManualWhenOnlyClientEvidenceExists(t *testing.T) {
	// [REQ:NM-P0-002] Client evidence must not be promoted to household-wide router enforcement.
	repo := newFakeRepo()
	_, err := repo.SaveBackend(context.Background(), BackendConfig{Backend: AdGuardHomeBackend, BaseURL: "http://adguard.local", TokenRef: "secret://adguard/token"})
	require.NoError(t, err)
	svc := NewService(Config{
		Repo: repo,
		Client: fakeAdGuardClient{checkStatus: ClientStatus{
			Status:              "healthy",
			FilteringEnabled:    true,
			Checks:              []string{"reachable"},
			EnforcementStatus:   "client_evidence_observed",
			EnforcementEvidence: []string{"AdGuard reports 2 usable client observation(s): 0 configured, 2 automatically discovered."},
		}},
		DNSInspector: fakeDNSInspector{inspection: DNSInspection{Servers: []string{defaultAdGuardDNSBindIP()}, Evidence: []string{"host protected"}}},
	})

	report, err := svc.AdGuardRollout(context.Background())
	require.NoError(t, err)
	require.Equal(t, "host_protected_router_manual", report.Status)
	require.Equal(t, "manual_required", rolloutCheckStatus(report, "router-dhcp"))
	require.Equal(t, "review_required", rolloutCheckStatus(report, "client-evidence"))
	require.Contains(t, strings.Join(report.NextSteps, "\n"), "Set router DHCP IPv4 DNS")
}

func TestAdGuardRolloutBlocksRouterInstructionsWhenAdGuardUnhealthy(t *testing.T) {
	// [REQ:NM-P0-002] Router rollout guidance stays blocked until AdGuard is healthy and filtering.
	repo := newFakeRepo()
	_, err := repo.SaveBackend(context.Background(), BackendConfig{Backend: AdGuardHomeBackend, BaseURL: "http://adguard.local", TokenRef: "secret://adguard/token"})
	require.NoError(t, err)
	svc := NewService(Config{
		Repo:         repo,
		Client:       fakeAdGuardClient{checkStatus: ClientStatus{Status: "degraded", FilteringEnabled: false}},
		DNSInspector: fakeDNSInspector{inspection: DNSInspection{Servers: []string{"192.168.1.1"}}},
	})

	report, err := svc.AdGuardRollout(context.Background())
	require.NoError(t, err)
	require.Equal(t, "blocked", report.Status)
	require.Equal(t, "blocked", rolloutCheckStatus(report, "adguard-resource"))
	require.Contains(t, report.Summary, "not ready")
}

func rolloutCheckStatus(report RolloutReport, id string) string {
	for _, check := range report.Checks {
		if check.ID == id {
			return check.Status
		}
	}
	return ""
}
