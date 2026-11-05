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
)

func TestGrantSpinePersistsAcrossDomainSchemas(t *testing.T) {
	ctx := context.Background()
	databaseHandle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, databaseHandle,
		database.SchemaProviderFunc(book.Schema),
		database.SchemaProviderFunc(budget.Schema),
		database.SchemaProviderFunc(mandate.Schema),
	))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)

	bookService := book.NewService(book.NewSQLiteRepository(databaseHandle), clock)
	_, err := bookService.Create(ctx, book.CreateInput{ID: "book-1", Name: "Business", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)

	budgetService := budget.NewService(budget.NewSQLiteRepository(databaseHandle), clock)
	_, err = budgetService.Create(ctx, budget.Budget{
		ID: "budget-1", BookID: "book-1", Currency: "USD",
		TotalCapMinor: 10_000, PeriodicCapMinor: 5_000, PerTransactionCapMinor: 1_000,
		Period: 30 * 24 * time.Hour, AllowedCounterparties: []string{"api.example"}, RequiresApproval: true,
	})
	require.NoError(t, err)

	signer, err := mandate.NewHMACSigner([]byte("test-only-signing-key"))
	require.NoError(t, err)
	mandateService := mandate.NewService(mandate.NewSQLiteRepository(databaseHandle), clock, signer)
	issued, err := mandateService.Issue(ctx, validIssue(now))
	require.NoError(t, err)
	require.NotEmpty(t, issued.Signature)

	got, err := mandateService.Get(ctx, issued.ID)
	require.NoError(t, err)
	require.Equal(t, issued, got)
}
