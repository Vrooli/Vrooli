package inventory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeSource struct {
	observations []Observation
	findings     []string
	err          error
}

func (f fakeSource) Discover(context.Context) ([]Observation, []string, error) {
	return f.observations, f.findings, f.err
}

type fakeRepo struct {
	mu      sync.Mutex
	devices map[string]Device
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{devices: map[string]Device{}}
}

func (r *fakeRepo) SaveDevice(_ context.Context, device Device) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[device.ID] = device
	return device, nil
}

func (r *fakeRepo) GetDevice(_ context.Context, id string) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	device, ok := r.devices[id]
	if !ok {
		return Device{}, ErrNotFound
	}
	return device, nil
}

func (r *fakeRepo) ListDevices(_ context.Context, group string) ([]Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Device, 0, len(r.devices))
	for _, device := range r.devices {
		if group != "" && device.Group != group {
			continue
		}
		out = append(out, device)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeRepo) UpdateGroup(ctx context.Context, id, group string) (Device, error) {
	device, err := r.GetDevice(ctx, id)
	if err != nil {
		return Device{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	device.Group = group
	r.devices[id] = device
	return device, nil
}

func TestServiceRefreshExactMatchPreservesIdentityAndGroup(t *testing.T) {
	// [REQ:NM-P0-004] Stable resolver identity matches an existing device and preserves labels.
	repo := newFakeRepo()
	now := time.Date(2026, 6, 23, 18, 0, 0, 0, time.UTC)
	_, err := repo.SaveDevice(context.Background(), Device{
		ID:                 "dev-1",
		Hostname:           "laptop",
		IPAddress:          "192.168.1.20",
		ResolverClientID:   "client-a",
		Group:              "work",
		IdentityConfidence: "medium",
		LastSeen:           now.Add(-time.Hour),
		CreatedAt:          now.Add(-time.Hour),
		UpdatedAt:          now.Add(-time.Hour),
	})
	require.NoError(t, err)
	svc := NewService(Config{Repo: repo, Now: func() time.Time { return now }, Source: fakeSource{observations: []Observation{
		{Hostname: "laptop", IPAddress: "192.168.1.21", ResolverClientID: "client-a"},
	}}})

	devices, findings, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Empty(t, findings)
	require.Len(t, devices, 1)
	require.Equal(t, "dev-1", devices[0].ID)
	require.Equal(t, "work", devices[0].Group)
	require.Equal(t, "high", devices[0].IdentityConfidence)
	require.Contains(t, strings.Join(devices[0].Notes, "\n"), "stable identity evidence")
}

func TestServiceRefreshMarksAmbiguousWeakIdentity(t *testing.T) {
	// [REQ:NM-P0-004] IP-only observations are retained but marked ambiguous.
	repo := newFakeRepo()
	now := time.Date(2026, 6, 23, 18, 5, 0, 0, time.UTC)
	svc := NewService(Config{Repo: repo, Now: func() time.Time { return now }, Source: fakeSource{observations: []Observation{
		{Hostname: "unknown", IPAddress: "192.168.1.30"},
	}}})

	devices, _, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "low", devices[0].IdentityConfidence)
	require.Contains(t, strings.Join(devices[0].Notes, "\n"), "ambiguous")
}

func TestServiceRefreshMarksRandomizedMAC(t *testing.T) {
	// [REQ:NM-P0-004] Locally administered MACs are called out as randomized identity risk.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Source: fakeSource{observations: []Observation{
		{Hostname: "phone", IPAddress: "192.168.1.31", MACAddress: "02:11:22:33:44:55"},
	}}})

	devices, _, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "low", devices[0].IdentityConfidence)
	require.Contains(t, strings.Join(devices[0].Notes, "\n"), "randomized")
}

func TestServiceRefreshNotesStaleHostname(t *testing.T) {
	// [REQ:NM-P0-004] Hostname drift is preserved as confidence evidence, not silently overwritten.
	repo := newFakeRepo()
	now := time.Date(2026, 6, 23, 18, 10, 0, 0, time.UTC)
	_, err := repo.SaveDevice(context.Background(), Device{
		ID:                 "dev-stale",
		Hostname:           "old-name",
		MACAddress:         "00:11:22:33:44:55",
		Group:              "trusted",
		IdentityConfidence: "medium",
		LastSeen:           now.Add(-24 * time.Hour),
		CreatedAt:          now.Add(-24 * time.Hour),
		UpdatedAt:          now.Add(-24 * time.Hour),
	})
	require.NoError(t, err)
	svc := NewService(Config{Repo: repo, Now: func() time.Time { return now }, Source: fakeSource{observations: []Observation{
		{Hostname: "new-name", MACAddress: "00:11:22:33:44:55"},
	}}})

	devices, _, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "new-name", devices[0].Hostname)
	require.Contains(t, strings.Join(devices[0].Notes, "\n"), "hostname changed")
}

func TestServiceUpdateGroupAndExplain(t *testing.T) {
	// [REQ:NM-P0-004] Device grouping and identity explanations are persisted.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Source: fakeSource{observations: []Observation{
		{Hostname: "tablet", IPAddress: "192.168.1.40", StableID: "stable-tablet"},
	}}})
	devices, _, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	updated, err := svc.UpdateGroup(context.Background(), devices[0].ID, "kids")
	require.NoError(t, err)
	require.Equal(t, "kids", updated.Group)

	list, err := svc.List(context.Background(), "kids")
	require.NoError(t, err)
	require.Len(t, list, 1)

	device, evidence, err := svc.Explain(context.Background(), devices[0].ID)
	require.NoError(t, err)
	require.Equal(t, devices[0].ID, device.ID)
	require.Contains(t, strings.Join(evidence, "\n"), "stable identifier observed")
}

func TestServiceRefreshUnsupportedSourceDoesNotInventDevices(t *testing.T) {
	// [REQ:NM-P0-004] Unsupported discovery sources return persisted inventory plus explicit findings.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Source: fakeSource{findings: []string{"resolver clients unavailable"}, err: ErrUnsupported}})

	devices, findings, err := svc.Refresh(context.Background(), false)
	require.NoError(t, err)
	require.Empty(t, devices)
	require.Contains(t, strings.Join(findings, "\n"), "resolver clients unavailable")
}
