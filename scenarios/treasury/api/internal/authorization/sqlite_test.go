package authorization_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/authorization"
	"treasury/internal/book"
	"treasury/internal/budget"
	internalevidence "treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/mandate"
)

// [REQ:TRS-P0-002] [REQ:TRS-P0-003] SQLite persistence serializes derived headroom holds and stores no balance column.
func TestSQLiteGrantSpineReservesHeadroomAtomicallyWithinTheService(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle,
		database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema),
		database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(internalevidence.Schema),
	))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operator", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 100, PeriodicCapMinor: 100, PerTransactionCapMinor: 100, Period: time.Hour, AllowedCounterparties: []string{"api.example"}, RequiresApproval: true})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("test-only-key"))
	require.NoError(t, err)
	mandatesService := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	_, err = mandatesService.Issue(ctx, mandate.IssueInput{ID: "mandate-1", IdempotencyKey: "mandate-key", BookID: "book-1", BudgetID: "budget-1", Authorizer: "operator:1", CapMinor: 100, Currency: "USD", AllowedCounterparties: []string{"api.example"}, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)

	service := authorization.NewService(authorization.NewSQLiteRepository(handle), verifier{claims: identity.Claims{Subject: "operator:1"}}, mandatesService, budget.NewService(budget.NewSQLiteRepository(handle), clock), internalevidence.NewRecorder(internalevidence.NewSQLiteRecorder(handle)), clock, &approvalQueue{})
	results := make(chan authorization.Record, 2)
	errorsSeen := make(chan error, 2)
	var group sync.WaitGroup
	for _, id := range []string{"auth-1", "auth-2"} {
		group.Add(1)
		go func() {
			defer group.Done()
			result, proposeErr := service.Propose(ctx, validInput(id, 75))
			results <- result
			errorsSeen <- proposeErr
		}()
	}
	group.Wait()
	close(results)
	close(errorsSeen)
	for proposeErr := range errorsSeen {
		require.NoError(t, proposeErr)
	}
	counts := map[authorization.Verdict]int{}
	for result := range results {
		counts[result.Verdict]++
	}
	require.Equal(t, 1, counts[authorization.VerdictPending])
	require.Equal(t, 1, counts[authorization.VerdictRefused])

	rows, err := handle.QueryContext(ctx, `PRAGMA table_info(budgets)`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var position, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		require.NoError(t, rows.Scan(&position, &name, &columnType, &notNull, &defaultValue, &primaryKey))
		require.NotEqual(t, "headroom", name, "headroom must be derived from immutable records, never stored")
	}
	require.NoError(t, rows.Err())

	// [REQ:TRS-P1-004] The durable authorization ownership binding cannot be
	// reassigned to another book, even through a raw SQL write.
	_, err = handle.ExecContext(ctx, `UPDATE authorization_book_bindings SET book_id='book-2' WHERE authorization_id=(SELECT authorization_id FROM authorization_book_bindings LIMIT 1)`)
	require.Error(t, err)
}
