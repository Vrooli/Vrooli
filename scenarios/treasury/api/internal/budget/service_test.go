package budget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/budget"
)

type budgetRepository struct{ value budget.Budget }

func (r *budgetRepository) Create(_ context.Context, value budget.Budget) (budget.Budget, error) {
	r.value = value
	return value, nil
}

func (r *budgetRepository) Get(_ context.Context, _ string) (budget.Budget, error) {
	if r.value.ID == "" {
		return budget.Budget{}, budget.ErrNotFound
	}
	return r.value, nil
}

func (r *budgetRepository) Update(_ context.Context, value budget.Budget) (budget.Budget, error) {
	r.value = value
	return value, nil
}

func TestServiceCarriesEveryBudgetConstraint(t *testing.T) { // [REQ:TRS-P0-003]
	repository := &budgetRepository{}
	service := budget.NewService(repository, schedule.NewFake(time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)))
	got, err := service.Create(context.Background(), budget.Budget{
		ID: "budget-1", BookID: "book-1", Currency: "usd",
		TotalCapMinor: 10_000, PeriodicCapMinor: 5_000, PerTransactionCapMinor: 1_000,
		Period: 30 * 24 * time.Hour, AllowedCounterparties: []string{"Example.COM", "example.com"},
		DeniedCounterparties: []string{"blocked.example", "example.com"}, RequiresApproval: true,
	})
	require.NoError(t, err)
	require.Equal(t, "USD", got.Currency)
	require.Equal(t, int64(10_000), got.TotalCapMinor)
	require.Equal(t, int64(5_000), got.PeriodicCapMinor)
	require.Equal(t, int64(1_000), got.PerTransactionCapMinor)
	require.True(t, got.RequiresApproval)
	require.False(t, got.AllowsCounterparty("example.com"), "deny must outrank allow")
	require.False(t, got.AllowsCounterparty("elsewhere.example"), "a non-empty allow list is closed")
}

func TestServiceRejectsIncoherentCapOrdering(t *testing.T) {
	service := budget.NewService(&budgetRepository{}, schedule.NewFake(time.Now()))
	_, err := service.Create(context.Background(), budget.Budget{
		ID: "budget-1", BookID: "book-1", Currency: "USD",
		TotalCapMinor: 100, PeriodicCapMinor: 200, PerTransactionCapMinor: 50, Period: time.Hour,
	})
	require.ErrorIs(t, err, budget.ErrInvalid)
}

func TestMutationsPreserveIndependentSafetyControls(t *testing.T) {
	repository := &budgetRepository{}
	service := budget.NewService(repository, schedule.NewFake(time.Now()))
	created, err := service.SetCaps(context.Background(), budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 1000, PeriodicCapMinor: 500, PerTransactionCapMinor: 100, Period: time.Hour, AllowedCounterparties: []string{"api.example"}})
	require.NoError(t, err)
	require.False(t, created.RequiresApproval)

	gated, err := service.SetGating(context.Background(), created.ID, true)
	require.NoError(t, err)
	frozen, err := service.SetFrozen(context.Background(), created.ID, true)
	require.NoError(t, err)

	updated, err := service.SetCaps(context.Background(), budget.Budget{ID: created.ID, TotalCapMinor: 900, PeriodicCapMinor: 400, PerTransactionCapMinor: 90, Period: 2 * time.Hour, AllowedCounterparties: []string{"other.example"}})
	require.NoError(t, err)
	require.True(t, updated.RequiresApproval, "cap edits must not disable human gating")
	require.True(t, updated.Frozen, "cap edits must not clear the emergency freeze")
	require.Equal(t, frozen.BookID, updated.BookID)
	require.Equal(t, gated.Currency, updated.Currency)
}
