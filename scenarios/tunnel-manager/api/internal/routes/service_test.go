package routes_test

import (
	"context"
	"testing"

	"tunnel-manager/internal/routes"

	"github.com/stretchr/testify/require"
)

// fakeRepo is a minimal in-memory routes.Repository for service tests
// that don't want the sqlite round-trip. It records the last route
// passed to Create/Update so tests can assert on the defaults the
// service applied.
type fakeRepo struct {
	created   routes.Route
	updated   routes.Route
	getResult routes.Route
	getErr    error
	createErr error
	listTier  routes.Tier
}

func (f *fakeRepo) Create(_ context.Context, r routes.Route) (routes.Route, error) {
	f.created = r
	if f.createErr != nil {
		return routes.Route{}, f.createErr
	}
	r.ID = "generated-id"
	return r, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (routes.Route, error) {
	if f.getErr != nil {
		return routes.Route{}, f.getErr
	}
	return f.getResult, nil
}

func (f *fakeRepo) GetBySubdomain(_ context.Context, _ string) (routes.Route, error) {
	return routes.Route{}, routes.ErrRouteNotFound{}
}

func (f *fakeRepo) List(_ context.Context, tier routes.Tier) ([]routes.Route, error) {
	f.listTier = tier
	return nil, nil
}

func (f *fakeRepo) Update(_ context.Context, r routes.Route) (routes.Route, error) {
	f.updated = r
	return r, nil
}

func (f *fakeRepo) Delete(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func boolPtr(b bool) *bool { return &b }

func TestCreate_AppliesDefaults(t *testing.T) {
	repo := &fakeRepo{}
	svc := routes.NewService(repo)

	got, err := svc.Create(context.Background(), routes.CreateInput{
		Subdomain: "agent-manager",
		Scenario:  "agent-manager",
		LocalPort: 21100,
	})
	require.NoError(t, err)
	require.Equal(t, routes.DefaultDomain, repo.created.Domain)
	require.Equal(t, routes.DefaultHealthPath, repo.created.HealthPath)
	require.Equal(t, routes.TierLeased, repo.created.Tier)
	require.True(t, repo.created.Enabled)
	require.Equal(t, "https://agent-manager.itsagitime.com", got.PublicURL())
}

func TestCreate_RespectsExplicitFields(t *testing.T) {
	repo := &fakeRepo{}
	svc := routes.NewService(repo)

	_, err := svc.Create(context.Background(), routes.CreateInput{
		Subdomain:  "web-console",
		Scenario:   "web-console",
		Domain:     "example.com",
		LocalPort:  3000,
		Tier:       routes.TierCore,
		HealthPath: "/ready",
		Enabled:    boolPtr(false),
	})
	require.NoError(t, err)
	require.Equal(t, "example.com", repo.created.Domain)
	require.Equal(t, "/ready", repo.created.HealthPath)
	require.Equal(t, routes.TierCore, repo.created.Tier)
	require.False(t, repo.created.Enabled)
}

func TestCreate_ValidationErrors(t *testing.T) {
	svc := routes.NewService(&fakeRepo{})
	cases := []struct {
		name  string
		in    routes.CreateInput
		field string
	}{
		{"empty subdomain", routes.CreateInput{Scenario: "s", LocalPort: 1}, "subdomain"},
		{"bad subdomain", routes.CreateInput{Subdomain: "Not_A_Label", Scenario: "s", LocalPort: 1}, "subdomain"},
		{"empty scenario", routes.CreateInput{Subdomain: "ok", LocalPort: 1}, "scenario"},
		{"port too low", routes.CreateInput{Subdomain: "ok", Scenario: "s", LocalPort: 0}, "local_port"},
		{"port too high", routes.CreateInput{Subdomain: "ok", Scenario: "s", LocalPort: 70000}, "local_port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), tc.in)
			var invalid routes.ErrInvalidRoute
			require.ErrorAs(t, err, &invalid)
			require.Equal(t, tc.field, invalid.Field)
		})
	}
}

func TestUpdate_PartialMerge(t *testing.T) {
	repo := &fakeRepo{getResult: routes.Route{
		ID: "r1", Subdomain: "old", Scenario: "s", Domain: "itsagitime.com",
		LocalPort: 100, Tier: routes.TierLeased, Enabled: true, HealthPath: "/health",
	}}
	svc := routes.NewService(repo)

	_, err := svc.Update(context.Background(), "r1", routes.UpdateInput{
		LocalPort: 200,
		Enabled:   boolPtr(false),
	})
	require.NoError(t, err)
	require.Equal(t, "old", repo.updated.Subdomain, "untouched field preserved")
	require.Equal(t, 200, repo.updated.LocalPort)
	require.False(t, repo.updated.Enabled)
}

func TestUpdate_NotFoundPropagates(t *testing.T) {
	repo := &fakeRepo{getErr: routes.ErrRouteNotFound{ID: "missing"}}
	svc := routes.NewService(repo)
	_, err := svc.Update(context.Background(), "missing", routes.UpdateInput{LocalPort: 1})
	var nf routes.ErrRouteNotFound
	require.ErrorAs(t, err, &nf)
}

func TestList_PassesTierFilter(t *testing.T) {
	repo := &fakeRepo{}
	svc := routes.NewService(repo)
	_, err := svc.List(context.Background(), routes.TierCore)
	require.NoError(t, err)
	require.Equal(t, routes.TierCore, repo.listTier)
}
