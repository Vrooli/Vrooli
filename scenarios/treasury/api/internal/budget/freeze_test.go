package budget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	"treasury/internal/book"
	"treasury/internal/budget"
)

// [REQ:TRS-P1-006] The kill switch composes budget, book, and scenario scope,
// with the broadest active scope reported to the caller.
func TestFreezeHierarchy(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema)))
	clock := schedule.NewFake(time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC))
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operating", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	service := budget.NewService(budget.NewSQLiteRepository(handle), clock)
	_, err = service.Create(ctx, budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 1_000, PeriodicCapMinor: 1_000, PerTransactionCapMinor: 100, Period: time.Hour})
	require.NoError(t, err)

	frozen, scope, err := service.IsFrozen(ctx, "book-1", "budget-1")
	require.NoError(t, err)
	require.False(t, frozen)
	require.Empty(t, scope)

	_, err = service.SetScopeFrozen(ctx, budget.FreezeScopeBook, "book-1", true)
	require.NoError(t, err)
	frozen, scope, err = service.IsFrozen(ctx, "book-1", "budget-1")
	require.NoError(t, err)
	require.True(t, frozen)
	require.Equal(t, budget.FreezeScopeBook, scope)

	_, err = service.SetScopeFrozen(ctx, budget.FreezeScopeScenario, "ignored", true)
	require.NoError(t, err)
	status, err := service.ScenarioFreezeStatus(ctx)
	require.NoError(t, err)
	require.True(t, status.Frozen)
	require.Equal(t, budget.FreezeScopeScenario, status.Scope)
	frozen, scope, err = service.IsFrozen(ctx, "book-1", "budget-1")
	require.NoError(t, err)
	require.True(t, frozen)
	require.Equal(t, budget.FreezeScopeScenario, scope)

	_, err = service.SetScopeFrozen(ctx, budget.FreezeScopeScenario, "*", false)
	require.NoError(t, err)
	_, err = service.SetScopeFrozen(ctx, budget.FreezeScopeBook, "book-1", false)
	require.NoError(t, err)
	_, err = service.SetFrozen(ctx, "budget-1", true)
	require.NoError(t, err)
	frozen, scope, err = service.IsFrozen(ctx, "book-1", "budget-1")
	require.NoError(t, err)
	require.True(t, frozen)
	require.Equal(t, budget.FreezeScopeBudget, scope)
}

func TestFreezeTransitionMatrix(t *testing.T) {
	for _, test := range []struct {
		initial bool
		event   budget.FreezeEvent
		want    bool
	}{
		{false, budget.FreezeEventEngage, true},
		{true, budget.FreezeEventEngage, true},
		{true, budget.FreezeEventRelease, false},
		{false, budget.FreezeEventRelease, false},
	} {
		next, err := budget.TransitionFreeze(budget.FreezeState{Frozen: test.initial}, test.event)
		require.NoError(t, err)
		require.NoError(t, budget.CheckFreezeInvariants(next))
		require.Equal(t, test.want, next.Frozen)
	}
}
