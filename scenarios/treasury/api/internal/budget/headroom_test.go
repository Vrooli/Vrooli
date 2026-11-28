package budget_test

import (
	"context"
	"testing"
	"time"

	"treasury/internal/authorization"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/mandate"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
)

// [REQ:TRS-P1-007] Headroom is derived entirely from Treasury's own
// authorization history. Active holds reduce it immediately; released holds
// do not, and no Money Ledger dependency participates in the calculation.
func TestHeadroomIsComputedFromAuthorizationRecords(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle,
		database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema),
		database.SchemaProviderFunc(mandate.Schema), database.SchemaProviderFunc(authorization.Schema),
	))
	now := time.Date(2026, 8, 19, 12, 30, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operating", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	budgets := budget.NewSQLiteRepository(handle)
	policy, err := budget.NewService(budgets, clock).Create(ctx, budget.Budget{
		ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 1_000,
		PeriodicCapMinor: 500, PerTransactionCapMinor: 100, Period: time.Hour,
	})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("headroom-test-signing-key"))
	require.NoError(t, err)
	_, err = mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer).Issue(ctx, mandate.IssueInput{
		ID: "mandate-1", IdempotencyKey: "mandate-key-1", BookID: "book-1", BudgetID: policy.ID,
		Authorizer: "operator:1", CapMinor: 1_000, Currency: "USD", AllowedCounterparties: []string{"vendor.example"}, ExpiresAt: now.Add(24 * time.Hour),
	})
	require.NoError(t, err)
	authorizations := authorization.NewSQLiteRepository(handle)
	for _, record := range []authorization.Record{
		{ID: "pending", IdempotencyKey: "pending-key", BookID: "book-1", MandateID: "mandate-1", BudgetID: policy.ID, RequestingAgent: "agent:1", AmountMinor: 200, Currency: "USD", Counterparty: "vendor.example", Verdict: authorization.VerdictPending, HoldMinor: 200, CreatedAt: now.Add(-10 * time.Minute), ExpiresAt: now.Add(5 * time.Minute)},
		{ID: "settled", IdempotencyKey: "settled-key", BookID: "book-1", MandateID: "mandate-1", BudgetID: policy.ID, RequestingAgent: "agent:1", AmountMinor: 150, Currency: "USD", Counterparty: "vendor.example", Verdict: authorization.VerdictSettled, CreatedAt: now.Add(-20 * time.Minute), ExpiresAt: now.Add(-5 * time.Minute)},
		{ID: "released", IdempotencyKey: "released-key", BookID: "book-1", MandateID: "mandate-1", BudgetID: policy.ID, RequestingAgent: "agent:1", AmountMinor: 50, Currency: "USD", Counterparty: "vendor.example", Verdict: authorization.VerdictReleased, CreatedAt: now.Add(-5 * time.Minute), ExpiresAt: now.Add(5 * time.Minute)},
	} {
		_, err = authorizations.Create(ctx, record)
		require.NoError(t, err)
	}

	service := budget.NewService(budgets, clock, authorizations)
	headroom, err := service.Headroom(ctx, policy.ID)
	require.NoError(t, err)
	require.EqualValues(t, 350, headroom.TotalUsedMinor)
	require.EqualValues(t, 650, headroom.TotalRemainingMinor)
	require.EqualValues(t, 350, headroom.PeriodUsedMinor)
	require.EqualValues(t, 150, headroom.PeriodRemainingMinor)
	require.EqualValues(t, 100, headroom.AvailableMinor)
	require.Equal(t, now, headroom.ComputedAt)

	_, err = service.SetFrozen(ctx, policy.ID, true)
	require.NoError(t, err)
	headroom, err = service.Headroom(ctx, policy.ID)
	require.NoError(t, err)
	require.Zero(t, headroom.AvailableMinor)
}
