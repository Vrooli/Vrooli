package instrument_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/rail"
	"treasury/internal/rail/manual"
)

type credentialResolver struct {
	reference string
	field     string
	value     string
}

func TestProductSchemasContainNoCredentialValueColumns(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(current), ".."))
	matches, err := filepath.Glob(filepath.Join(root, "*", "schema.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, matches)
	for _, path := range matches {
		contents, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		normalized := strings.ToLower(string(contents))
		for _, forbidden := range []string{"credential_value", "secret_value", "plaintext_secret", "card_number", "password", "access_token"} {
			require.NotContains(t, normalized, forbidden, "%s must store a logical credential reference only", path)
		}
	}
}

func (r *credentialResolver) Resolve(_ context.Context, reference, field string) (string, error) {
	r.reference, r.field = reference, field
	return r.value, nil
}

// [REQ:TRS-P0-007] Instrument scope is projected from a live mandate while
// credential material is resolved only at use time and never persisted.
func TestInstrumentScopeAndCredentialUseBoundary(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema), database.SchemaProviderFunc(rail.Schema), database.SchemaProviderFunc(instrument.Schema)))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operator", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 1000, PeriodicCapMinor: 1000, PerTransactionCapMinor: 1000, Period: time.Hour, AllowedCounterparties: []string{"vendor.example"}})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("test-only-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	grant, err := mandates.Issue(ctx, mandate.IssueInput{ID: "mandate-1", IdempotencyKey: "mandate-key", BookID: "book-1", BudgetID: "budget-1", Authorizer: "operator:1", CapMinor: 750, Currency: "USD", AllowedCounterparties: []string{"vendor.example"}, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	registry, err := rail.NewRegistry(manual.New())
	require.NoError(t, err)
	resolver := &credentialResolver{value: "sensitive-value"}
	service := instrument.NewService(instrument.NewSQLiteRepository(handle), mandates, registry, resolver, clock)

	created, err := service.Register(ctx, instrument.RegisterInput{ID: "instrument-1", MandateID: grant.ID, Rail: "manual", CredentialReference: "vrooli/treasury/manual-1", Counterparty: "vendor.example"})
	require.NoError(t, err)
	require.Equal(t, grant.BookID, created.BookID)
	require.Equal(t, grant.CapMinor, created.CapMinor)
	require.Equal(t, grant.Currency, created.Currency)
	require.Equal(t, grant.ExpiresAt, created.ExpiresAt)
	var storedRef string
	require.NoError(t, handle.QueryRowContext(ctx, `SELECT credential_reference FROM instruments WHERE id='instrument-1'`).Scan(&storedRef))
	require.Equal(t, "vrooli/treasury/manual-1", storedRef)
	var leaked int
	require.NoError(t, handle.QueryRowContext(ctx, `SELECT COUNT(*) FROM instruments WHERE credential_reference='sensitive-value'`).Scan(&leaked))
	require.Zero(t, leaked)

	scoped, err := service.ResolveForUse(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "sensitive-value", scoped.Value)
	require.Equal(t, storedRef, resolver.reference)
	require.Equal(t, "value", resolver.field)
}
