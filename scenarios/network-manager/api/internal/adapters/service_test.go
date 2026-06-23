package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeRegistry struct {
	report Report
	err    error
}

func (f fakeRegistry) Report(context.Context) (Report, error) {
	return f.report, f.err
}

type fakeRepo struct {
	reports []Report
}

func (r *fakeRepo) SaveReport(_ context.Context, report Report) error {
	r.reports = append([]Report{report}, r.reports...)
	return nil
}

func (r *fakeRepo) LatestCapabilities(context.Context) ([]Capability, error) {
	if len(r.reports) == 0 {
		return nil, ErrNotFound
	}
	return r.reports[0].Capabilities, nil
}

func (r *fakeRepo) LatestPlatformSummary(context.Context) (PlatformSummary, error) {
	if len(r.reports) == 0 {
		return PlatformSummary{}, ErrNotFound
	}
	return r.reports[0].Platform, nil
}

func TestServiceListCapabilitiesPersistsCapabilityReport(t *testing.T) {
	// [REQ:NM-P0-006] Capabilities are reported before actions are offered and persisted as evidence.
	repo := &fakeRepo{}
	observedAt := time.Date(2026, 6, 23, 17, 0, 0, 0, time.UTC)
	svc := NewService(Config{
		Repo: repo,
		Registry: fakeRegistry{report: Report{
			ObservedAt: observedAt,
			Platform:   PlatformSummary{OS: "linux", Arch: "amd64", Profile: "host-diagnostics"},
			Capabilities: []Capability{
				{Adapter: "host-linux", Action: "read_network_status", Supported: true},
				{Adapter: "manual-router", Action: "router_dns_enforcement", Supported: false, Reason: "manual only"},
			},
		}},
	})

	caps, err := svc.ListCapabilities(context.Background())
	require.NoError(t, err)
	require.Len(t, caps, 2)
	require.Equal(t, observedAt, caps[0].ObservedAt)

	require.Len(t, repo.reports, 1)
	require.Equal(t, "linux", repo.reports[0].Platform.OS)
	require.Equal(t, observedAt, repo.reports[0].Platform.ObservedAt)
}

func TestServiceExplainUnsupportedActionReturnsManualSteps(t *testing.T) {
	// [REQ:NM-P0-006] Unsupported actions include reasons and manual-safe next steps.
	svc := NewService(Config{Registry: fakeRegistry{report: Report{Capabilities: []Capability{
		{Adapter: "manual-router", Action: "router_dns_enforcement", Supported: false, Reason: "P0 excludes router writes."},
	}}}})

	cap, steps, err := svc.ExplainUnsupportedAction(context.Background(), "router_dns_enforcement")
	require.NoError(t, err)
	require.False(t, cap.Supported)
	require.Contains(t, cap.Reason, "router writes")
	require.NotEmpty(t, steps)
	require.Contains(t, steps[0], "router")
}

func TestStaticRegistryPlatformCombinations(t *testing.T) {
	// [REQ:NM-P0-006] Platform summaries stay explicit across supported and manual profiles.
	for _, tc := range []struct {
		os      string
		profile string
	}{
		{os: "linux", profile: "host-diagnostics"},
		{os: "darwin", profile: "host-diagnostics"},
		{os: "windows", profile: "host-diagnostics"},
		{os: "plan9", profile: "manual"},
	} {
		t.Run(tc.os, func(t *testing.T) {
			registry := StaticRegistry{OS: tc.os, Arch: "amd64", Now: func() time.Time {
				return time.Date(2026, 6, 23, 17, 30, 0, 0, time.UTC)
			}}
			report, err := registry.Report(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.profile, report.Platform.Profile)
			require.NotEmpty(t, report.Platform.Notes)
			require.NotEmpty(t, report.Capabilities)
		})
	}
}
