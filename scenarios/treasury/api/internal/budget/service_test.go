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
	return r.value, nil
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
