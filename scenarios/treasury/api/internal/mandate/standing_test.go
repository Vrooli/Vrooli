package mandate_test

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
	"treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"
)

// [REQ:TRS-P1-005] A standing mandate exposes its next charge, advances each
// boundary once, and cancellation is one durable action that prevents any
// later recurrence without rewriting prior occurrences.
func TestStandingMandateAdvancesOnceAndCancellationStopsNextCharge(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle,
		database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema),
	))
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operating", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 10_000, PeriodicCapMinor: 1_000, PerTransactionCapMinor: 100, Period: 24 * time.Hour})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("standing-test-signing-key"))
	require.NoError(t, err)
	service := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	created, err := service.Issue(ctx, mandate.IssueInput{
		ID: "standing-1", IdempotencyKey: "recurrence-idem-1", BookID: "book-1", BudgetID: "budget-1",
		Authorizer: "operator:1", CapMinor: 1_000, Currency: "USD", AllowedCounterparties: []string{"vendor.example"},
		ExpiresAt: now.Add(90 * 24 * time.Hour), RecurrenceInterval: 30 * 24 * time.Hour, NextChargeAt: now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.Equal(t, now.Add(time.Hour), created.NextChargeAt)

	clock.Advance(time.Hour)
	occurrence, err := service.ClaimDue(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, now.Add(time.Hour), occurrence.ChargeAt)
	require.Equal(t, now.Add(time.Hour+30*24*time.Hour), occurrence.NextChargeAt)
	_, err = service.ClaimDue(ctx, created.ID)
	require.ErrorIs(t, err, mandate.ErrNotFound, "the same recurrence boundary cannot be claimed twice")

	cancelled, err := service.CancelStanding(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, mandateflow.MandateRevoked, cancelled.Status)
	require.Equal(t, clock.Now(), cancelled.CancelledAt)
	clock.Advance(30 * 24 * time.Hour)
	_, err = service.ClaimDue(ctx, created.ID)
	require.ErrorIs(t, err, mandate.ErrNotFound, "completed cancellation must beat every later charge boundary")
	stored, err := service.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, cancelled.CancelledAt, stored.CancelledAt)
}
