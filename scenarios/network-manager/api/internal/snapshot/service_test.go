package snapshot

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	testmocks "network-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	results []ProbeResult
	err     error
}

func (f fakeRunner) Run(context.Context, string) ([]ProbeResult, error) {
	return f.results, f.err
}

type fakeRepo struct {
	mu    sync.Mutex
	items []Snapshot
}

func (r *fakeRepo) Create(_ context.Context, s Snapshot) (Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.ID == "" {
		s.ID = "snapshot-test"
	}
	r.items = append([]Snapshot{s}, r.items...)
	return s, nil
}

func (r *fakeRepo) List(context.Context) ([]Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Snapshot, len(r.items))
	copy(out, r.items)
	return out, nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return Snapshot{}, ErrNotFound
}

func (r *fakeRepo) Count(context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.items), nil
}

func TestServiceRunPersistsFirstSnapshotAsBaseline(t *testing.T) {
	// [REQ:NM-P0-001] First real snapshot is persisted as the baseline anchor.
	repo := &fakeRepo{}
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 23, 16, 0, 0, 0, time.UTC))
	svc := NewService(Config{Repo: repo, Clock: clk, Runner: fakeRunner{results: []ProbeResult{
		{Name: "dns_lookup_latency", Value: "18", Unit: "ms", Status: "healthy"},
		{Name: "throughput_availability", Value: "unavailable", Unit: "status", Status: "unavailable", Finding: "Throughput unavailable until approved."},
	}}})

	got, err := svc.Run(context.Background(), "", false)
	require.NoError(t, err)
	require.Equal(t, "baseline", got.Status)
	require.Equal(t, "home", got.Profile)
	require.Contains(t, got.Summary, "1 healthy")
	require.Contains(t, strings.Join(got.Findings, "\n"), "Baseline anchor")

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestServiceRunDryRunDoesNotPersist(t *testing.T) {
	// [REQ:NM-P0-001] Unsupported probes are explicit and dry runs do not create evidence.
	repo := &fakeRepo{}
	svc := NewService(Config{Repo: repo, Runner: fakeRunner{results: []ProbeResult{
		{Name: "gateway_reachability", Value: "unsupported", Unit: "status", Status: "unsupported", Finding: "Gateway adapter unsupported."},
	}}})

	got, err := svc.Run(context.Background(), "office", true)
	require.NoError(t, err)
	require.Equal(t, "dry_run", got.Status)
	require.Equal(t, "snapshot-dry-run", got.ID)

	count, err := repo.Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestServiceExportMarkdownIncludesMetricsAndFindings(t *testing.T) {
	// [REQ:NM-P0-001] Exported reports preserve metrics and unsupported-probe findings.
	repo := &fakeRepo{}
	svc := NewService(Config{Repo: repo, Runner: fakeRunner{results: []ProbeResult{
		{Name: "dns_lookup_latency", Value: "42", Unit: "ms", Status: "healthy"},
		{Name: "gateway_reachability", Value: "unsupported", Unit: "status", Status: "unsupported", Finding: "Gateway adapter unsupported."},
	}}})
	snap, err := svc.Run(context.Background(), "home", false)
	require.NoError(t, err)

	id, format, report, err := svc.Export(context.Background(), snap.ID, "markdown")
	require.NoError(t, err)
	require.Equal(t, snap.ID, id)
	require.Equal(t, "markdown", format)
	require.Contains(t, report, "# Network Snapshot")
	require.Contains(t, report, "dns_lookup_latency")
	require.Contains(t, report, "Gateway adapter unsupported.")
}
