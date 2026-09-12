package assignments_test

import (
	"context"
	"errors"
	"testing"

	"brand-manager/internal/assignments"
	"brand-manager/internal/assignments/mocks"

	"github.com/stretchr/testify/require"
)

func newService(t *testing.T, versions map[string]int) (assignments.Service, *mocks.FakeRepository) {
	t.Helper()
	repo := &mocks.FakeRepository{}
	resolver := mocks.FakeBrandResolver{Versions: versions}
	return assignments.NewService(repo, resolver, nil), repo
}

func TestAssign_PinsBrandVersion(t *testing.T) {
	svc, _ := newService(t, map[string]int{"b1": 4})

	got, err := svc.Assign(context.Background(), assignments.AssignInput{
		BrandID:      "b1",
		ScenarioName: "web-console",
		Elements:     []string{"logo", "colors"},
	})
	require.NoError(t, err)
	require.Equal(t, "b1", got.BrandID)
	require.Equal(t, "web-console", got.ScenarioName)
	require.Equal(t, 4, got.BrandVersion, "version is pinned from the brand at assign time")
	require.Equal(t, []string{"logo", "colors"}, got.Elements)
	require.NotEmpty(t, got.ID)
	require.False(t, got.AppliedAt.IsZero())
}

func TestAssign_RequiresBrandAndScenario(t *testing.T) {
	svc, _ := newService(t, map[string]int{"b1": 1})

	_, err := svc.Assign(context.Background(), assignments.AssignInput{ScenarioName: "x"})
	requireInvalidField(t, err, "brand_id")

	_, err = svc.Assign(context.Background(), assignments.AssignInput{BrandID: "b1"})
	requireInvalidField(t, err, "scenario_name")
}

func TestAssign_RejectsUnknownBrand(t *testing.T) {
	svc, _ := newService(t, map[string]int{"b1": 1})

	_, err := svc.Assign(context.Background(), assignments.AssignInput{BrandID: "ghost", ScenarioName: "x"})
	requireInvalidField(t, err, "brand_id")
}

func TestAssign_ResolverFailureSurfaces(t *testing.T) {
	repo := &mocks.FakeRepository{}
	boom := errors.New("lookup down")
	svc := assignments.NewService(repo, mocks.FakeBrandResolver{Err: boom}, nil)

	_, err := svc.Assign(context.Background(), assignments.AssignInput{BrandID: "b1", ScenarioName: "x"})
	require.ErrorIs(t, err, boom)
	// A genuine lookup failure must not masquerade as a validation error.
	var invalid assignments.ErrInvalidAssignment
	require.False(t, errors.As(err, &invalid))
}

func TestAssign_ReassignReplacesAndRepins(t *testing.T) {
	svc, _ := newService(t, map[string]int{"b1": 1, "b2": 7})
	ctx := context.Background()

	first, err := svc.Assign(ctx, assignments.AssignInput{BrandID: "b1", ScenarioName: "web-console"})
	require.NoError(t, err)

	second, err := svc.Assign(ctx, assignments.AssignInput{BrandID: "b2", ScenarioName: "web-console", Elements: []string{"logo"}})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "re-assigning a scenario keeps a stable id")
	require.Equal(t, "b2", second.BrandID)
	require.Equal(t, 7, second.BrandVersion)

	list, err := svc.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, list, 1, "a scenario carries at most one assignment")
}

func TestScenarioStatus_UnassignedIsNotAnError(t *testing.T) {
	svc, _ := newService(t, nil)

	status, err := svc.ScenarioStatus(context.Background(), "web-console")
	require.NoError(t, err)
	require.False(t, status.HasBrand)
	require.Equal(t, "web-console", status.Scenario)
	require.Empty(t, status.BrandID)
	require.Zero(t, status.BrandVersion)
}

func TestScenarioStatus_FromAssignment(t *testing.T) {
	svc, _ := newService(t, map[string]int{"b1": 2})
	ctx := context.Background()
	_, err := svc.Assign(ctx, assignments.AssignInput{BrandID: "b1", ScenarioName: "my-app", Elements: []string{"colors"}})
	require.NoError(t, err)

	status, err := svc.ScenarioStatus(ctx, "my-app")
	require.NoError(t, err)
	require.True(t, status.HasBrand)
	require.Equal(t, "b1", status.BrandID)
	require.Equal(t, 2, status.BrandVersion)
	require.Equal(t, []string{"colors"}, status.Elements)
	require.False(t, status.AppliedAt.IsZero())
}

func TestUnassign_IsIdempotent(t *testing.T) {
	svc, repo := newService(t, map[string]int{"b1": 1})
	ctx := context.Background()

	// Unassigning a scenario with no brand is a success.
	require.NoError(t, svc.Unassign(ctx, "ghost"))

	_, err := svc.Assign(ctx, assignments.AssignInput{BrandID: "b1", ScenarioName: "web-console"})
	require.NoError(t, err)
	require.NoError(t, svc.Unassign(ctx, "web-console"))
	require.NoError(t, svc.Unassign(ctx, "web-console"), "second unassign is still a success")
	require.GreaterOrEqual(t, repo.DeleteCalls.Load(), int64(2))
}

func TestList_FiltersByBrand(t *testing.T) {
	svc, _ := newService(t, map[string]int{"b1": 1, "b2": 1})
	ctx := context.Background()
	_, err := svc.Assign(ctx, assignments.AssignInput{BrandID: "b1", ScenarioName: "a"})
	require.NoError(t, err)
	_, err = svc.Assign(ctx, assignments.AssignInput{BrandID: "b2", ScenarioName: "b"})
	require.NoError(t, err)

	b1, err := svc.List(ctx, "b1")
	require.NoError(t, err)
	require.Len(t, b1, 1)
	require.Equal(t, "a", b1[0].ScenarioName)

	all, err := svc.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 2)
}

func requireInvalidField(t *testing.T, err error, field string) {
	t.Helper()
	var invalid assignments.ErrInvalidAssignment
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, field, invalid.Field)
}
