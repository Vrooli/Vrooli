package instrument_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/modules"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
	"treasury/internal/rail/manual"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
)

type credentialResolver struct {
	reference       string
	field           string
	value           string
	storedReference string
	storedField     string
	storedValue     string
}

func TestProductSchemasContainNoCredentialValueColumns(t *testing.T) {
	// Inspect the actual boot-wired embedded schemas. runtime.Caller paths are
	// module-relative under -trimpath and cannot locate source files at runtime.
	schemas := modules.AllSchemas()
	require.NotEmpty(t, schemas)
	for index, provider := range schemas {
		normalized := strings.ToLower(provider.Schema())
		for _, forbidden := range []string{"credential_value", "secret_value", "plaintext_secret", "card_number", "password", "access_token"} {
			require.NotContains(t, normalized, forbidden, "boot schema %d must store a logical credential reference only", index)
		}
	}
}

func (r *credentialResolver) Resolve(_ context.Context, reference, field string) (string, error) {
	r.reference, r.field = reference, field
	if reference == r.storedReference && field == r.storedField && r.storedValue != "" {
		return r.storedValue, nil
	}
	return r.value, nil
}

func (r *credentialResolver) Store(_ context.Context, reference, field, value string) error {
	r.storedReference, r.storedField, r.storedValue = reference, field, value
	return nil
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
	registry, err := rail.NewRegistry(credentialRail{})
	require.NoError(t, err)
	resolver := &credentialResolver{value: "sensitive-value"}
	service := instrument.NewService(instrument.NewSQLiteRepository(handle), mandates, registry, resolver, clock)

	created, err := service.Register(ctx, instrument.RegisterInput{ID: "instrument-1", MandateID: grant.ID, Rail: "credential-test", CredentialReference: "vrooli/treasury/manual-1", Counterparty: "vendor.example"})
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

type credentialRail struct{}

func (credentialRail) Name() string { return "credential-test" }
func (credentialRail) Settle(context.Context, rail.SettleCommand) (rail.Result, error) {
	return rail.Result{}, nil
}

func (credentialRail) QueryOutcome(context.Context, rail.Query) (rail.Result, error) {
	return rail.Result{}, nil
}

type scopedCardIssuer struct {
	received card.IssueCommand
	calls    atomic.Int64
	failNext bool
	started  chan struct{}
	release  chan struct{}
}

func (*scopedCardIssuer) Name() string { return "fixture-scoped-card" }
func (i *scopedCardIssuer) Issue(_ context.Context, command card.IssueCommand) (card.Issued, error) {
	i.calls.Add(1)
	i.received = command
	if i.started != nil {
		close(i.started)
		<-i.release
	}
	if i.failNext {
		i.failNext = false
		return card.Issued{}, context.DeadlineExceeded
	}
	return card.Issued{ExternalID: "provider-card-1", Credential: "issued-card-secret", Scope: command.Scope}, nil
}

func (i *scopedCardIssuer) Inspect(context.Context, card.InspectQuery) (card.Issued, error) {
	return card.Issued{}, nil
}

// [REQ:TRS-P1-003] Registration resolves the provider credential at issuance,
// projects every scope field from the live mandate, and writes the resulting
// card secret back to Credential Authority instead of Treasury's database.
func TestScopedCardIssuanceUsesMandateAndCredentialAuthority(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema), database.SchemaProviderFunc(rail.Schema), database.SchemaProviderFunc(instrument.Schema)))
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "card-book", Name: "Operator", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "card-budget", BookID: "card-book", Currency: "USD", TotalCapMinor: 2000, PeriodicCapMinor: 2000, PerTransactionCapMinor: 2000, Period: time.Hour, AllowedCounterparties: []string{"MERCHANT123"}})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("test-only-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	grant, err := mandates.Issue(ctx, mandate.IssueInput{ID: "card-mandate", IdempotencyKey: "card-mandate-key", BookID: "card-book", BudgetID: "card-budget", Authorizer: "operator:1", CapMinor: 875, Currency: "USD", AllowedCounterparties: []string{"MERCHANT123"}, ExpiresAt: now.Add(60 * 24 * time.Hour)})
	require.NoError(t, err)
	issuer := &scopedCardIssuer{}
	issuers, err := card.NewRegistry(issuer)
	require.NoError(t, err)
	rails, err := rail.NewRegistry()
	require.NoError(t, err)
	credentials := &credentialResolver{value: `{"api_key":"provider-secret"}`}
	service := instrument.NewServiceWithCardIssuers(instrument.NewSQLiteRepository(handle), mandates, rails, credentials, clock, issuers)

	input := instrument.RegisterInput{ID: "card-instrument", MandateID: grant.ID, Rail: issuer.Name(), CredentialReference: "vrooli/treasury/lithic", Counterparty: "MERCHANT123"}
	issuer.started, issuer.release = make(chan struct{}), make(chan struct{})
	type registerResult struct {
		value instrument.Instrument
		err   error
	}
	firstResult := make(chan registerResult, 1)
	go func() {
		value, registerErr := service.Register(ctx, input)
		firstResult <- registerResult{value: value, err: registerErr}
	}()
	<-issuer.started
	_, concurrentErr := service.Register(ctx, input)
	require.ErrorIs(t, concurrentErr, instrument.ErrCardIssueInProgress)
	require.EqualValues(t, 1, issuer.calls.Load(), "a concurrent registration must not dispatch a second issuer call")
	close(issuer.release)
	result := <-firstResult
	require.NoError(t, result.err)
	created := result.value
	issuer.started, issuer.release = nil, nil
	require.Equal(t, grant.ID, issuer.received.Scope.MandateReference)
	require.Equal(t, grant.CapMinor, issuer.received.Scope.AmountMinor)
	require.Equal(t, grant.Currency, issuer.received.Scope.Currency)
	require.Equal(t, "merchant123", issuer.received.Scope.Counterparty)
	require.Equal(t, grant.ExpiresAt, issuer.received.Scope.ExpiresAt)
	require.Equal(t, "vrooli/treasury/lithic", credentials.reference)
	require.Equal(t, "value", credentials.field)
	require.Equal(t, "issued-card-secret", credentials.storedValue)
	require.Equal(t, "value", credentials.storedField)
	require.NotEqual(t, "vrooli/treasury/lithic", credentials.storedReference)
	require.Equal(t, credentials.storedReference, created.CredentialReference)

	var storedReference string
	require.NoError(t, handle.QueryRowContext(ctx, `SELECT credential_reference FROM instruments WHERE id='card-instrument'`).Scan(&storedReference))
	require.Equal(t, credentials.storedReference, storedReference)
	require.NotContains(t, storedReference, "provider-secret")
	require.NotContains(t, storedReference, "issued-card-secret")
	scoped, err := service.ResolveForUse(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "issued-card-secret", scoped.Value)
	require.Equal(t, storedReference, credentials.reference)

	// Repeating a successful registration must return the committed instrument
	// without asking the provider to create or bind another card.
	repeated, err := service.Register(ctx, instrument.RegisterInput{ID: "card-instrument", MandateID: grant.ID, Rail: issuer.Name(), CredentialReference: "vrooli/treasury/lithic", Counterparty: "MERCHANT123"})
	require.NoError(t, err)
	require.Equal(t, created, repeated)
	require.EqualValues(t, 1, issuer.calls.Load())

	// A retry inside provider retention uses the same instrument identity. Once
	// an ambiguous attempt ages beyond that window, reconciliation is required
	// and Register must not dispatch a second issue request.
	issuer.failNext = true
	uncertainInput := instrument.RegisterInput{ID: "uncertain-card", MandateID: grant.ID, Rail: issuer.Name(), CredentialReference: "vrooli/treasury/lithic", Counterparty: "MERCHANT123"}
	_, err = service.Register(ctx, uncertainInput)
	require.Error(t, err)
	require.EqualValues(t, 2, issuer.calls.Load())
	clock.Advance(31 * 24 * time.Hour)
	_, err = service.Register(ctx, uncertainInput)
	require.ErrorIs(t, err, instrument.ErrCardIssueReconciliation)
	require.EqualValues(t, 2, issuer.calls.Load(), "expired provider idempotency must never be treated as permission to reissue")
}

// [REQ:TRS-P1-003] Durable issuance journal refuses parallel attempts and
// request identity changes before they reach the issuer.
func TestCardIssueJournalSerializesConcurrentFirstAttempt(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(instrument.Schema)))
	repository := instrument.NewSQLiteRepository(handle)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	first, err := repository.BeginCardIssue(ctx, "instrument-concurrent", strings.Repeat("a", 64), now)
	require.NoError(t, err)
	require.Equal(t, "in_flight", first.Status)
	_, err = repository.BeginCardIssue(ctx, "instrument-concurrent", strings.Repeat("a", 64), now.Add(time.Minute))
	require.ErrorIs(t, err, instrument.ErrCardIssueInProgress)
	_, err = repository.BeginCardIssue(ctx, "instrument-concurrent", strings.Repeat("b", 64), now.Add(time.Minute))
	require.ErrorIs(t, err, instrument.ErrInvalid)
	require.NoError(t, repository.SetCardIssueAttemptStatus(ctx, "instrument-concurrent", strings.Repeat("a", 64), "complete", now.Add(2*time.Minute)))
	require.NoError(t, repository.SetCardIssueAttemptStatus(ctx, "instrument-concurrent", strings.Repeat("a", 64), "unknown", now.Add(3*time.Minute)), "a late issuer timeout cannot downgrade completed durable evidence")
	completed, err := repository.GetCardIssueAttempt(ctx, "instrument-concurrent")
	require.NoError(t, err)
	require.Equal(t, "complete", completed.Status)
}

func TestManualInstrumentRequiresNoCredentialMaterial(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema), database.SchemaProviderFunc(rail.Schema), database.SchemaProviderFunc(instrument.Schema)))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "manual-book", Name: "Operator", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "manual-budget", BookID: "manual-book", Currency: "USD", TotalCapMinor: 1000, PeriodicCapMinor: 1000, PerTransactionCapMinor: 1000, Period: time.Hour, AllowedCounterparties: []string{"vendor.example"}})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("test-only-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	grant, err := mandates.Issue(ctx, mandate.IssueInput{ID: "manual-mandate", IdempotencyKey: "manual-key", BookID: "manual-book", BudgetID: "manual-budget", Authorizer: "operator:1", CapMinor: 750, Currency: "USD", AllowedCounterparties: []string{"vendor.example"}, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	registry, err := rail.NewRegistry(manual.New())
	require.NoError(t, err)
	service := instrument.NewService(instrument.NewSQLiteRepository(handle), mandates, registry, nil, clock)
	created, err := service.Register(ctx, instrument.RegisterInput{ID: "manual-instrument", MandateID: grant.ID, Rail: "manual", Counterparty: "vendor.example"})
	require.NoError(t, err)
	require.Equal(t, "manual/operator-attestation", created.CredentialReference)
	scoped, err := service.ResolveForUse(ctx, created.ID)
	require.NoError(t, err)
	require.Empty(t, scoped.Value)
}
