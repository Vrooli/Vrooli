package settlement_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/authorization"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/ledger"
	"treasury/internal/mandate"
	"treasury/internal/rail"
	"treasury/internal/settlement"
)

// [REQ:TRS-P0-011] N simultaneous submissions share one durable outcome and
// dispatch exactly one rail call.
func TestConcurrentRetrySettlesExactlyOnce(t *testing.T) {
	fixture := newFixture(t, &adapter{settleResult: settledResult()})
	missingKey := fixture.input
	missingKey.IdempotencyKey = ""
	_, err := fixture.service.Settle(context.Background(), missingKey)
	require.ErrorIs(t, err, settlement.ErrInvalid)
	wrongIdentity := fixture.input
	wrongIdentity.IdentityToken = "different-agent-token"
	_, err = fixture.service.Settle(context.Background(), wrongIdentity)
	require.ErrorIs(t, err, settlement.ErrInvalid)
	require.Zero(t, fixture.adapter.settleCalls.Load())

	const callers = 32
	results := make(chan settlement.Record, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := fixture.service.Settle(context.Background(), fixture.input)
			results <- value
			errs <- err
		}()
	}
	group.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	for value := range results {
		require.Equal(t, settlement.OutcomeSettled, value.Outcome)
		require.Equal(t, "processor-charge-1", value.ExternalID)
		require.Equal(t, fixture.now.Add(settlement.RetentionWindow), value.RetainUntil)
	}
	require.EqualValues(t, 1, fixture.adapter.settleCalls.Load())

	stored, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictSettled, stored.Verdict)
	require.Zero(t, stored.HoldMinor)
	var count int
	require.NoError(t, fixture.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM settlements WHERE idempotency_key='settle-key-1'`).Scan(&count))
	require.Equal(t, 1, count)
	replayed, err := evidence.NewRecorder(evidence.NewSQLiteRecorder(fixture.db)).Replay(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, "settled", replayed.Outcome)
	require.Equal(t, "settlement-1", replayed.SettlementID)
	var replayRequest map[string]any
	require.NoError(t, json.Unmarshal([]byte(replayed.RequestJSON), &replayRequest))
	require.Equal(t, float64(250), replayRequest["amount_minor"])
	require.Equal(t, "api.example", replayRequest["counterparty"])
	emission, err := ledger.NewSQLiteRepository(fixture.db).GetBySettlement(context.Background(), "settlement-1")
	require.NoError(t, err)
	require.Equal(t, ledger.StatusQueued, emission.Status)
	require.EqualValues(t, -250, emission.AmountMinor)
	require.Equal(t, "treasury:settlement-1", emission.ExternalID)

	alias := fixture.input
	alias.ID = "settlement-alias"
	_, err = fixture.service.Settle(context.Background(), alias)
	require.ErrorIs(t, err, settlement.ErrInvalid)
	require.EqualValues(t, 1, fixture.adapter.settleCalls.Load())
	require.NoError(t, fixture.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ledger_emissions WHERE settlement_id='settlement-1'`).Scan(&count))
	require.Equal(t, 1, count)
}

// [REQ:TRS-P0-011] A lost response remains unknown on every Settle retry and
// can become failed only after the adapter's Query returns a definite failure.
func TestUnknownCanFailOnlyFromRailQuery(t *testing.T) {
	adapter := &adapter{
		settleErr: errors.New("response timed out after dispatch"),
		queryResults: []rail.Result{{
			Outcome: rail.OutcomeFailed,
			Basis:   "processor_query",
			Detail:  "processor confirms no transfer was created",
		}},
	}
	fixture := newFixture(t, adapter)

	first, err := fixture.service.Settle(context.Background(), fixture.input)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeUnknown, first.Outcome)
	retry, err := fixture.service.Settle(context.Background(), fixture.input)
	require.NoError(t, err)
	require.Equal(t, first, retry)
	require.EqualValues(t, 1, adapter.settleCalls.Load(), "retry must not call the rail again")

	_, err = fixture.settlements.Complete(context.Background(), first.ID, settlement.OutcomeFailed, settlement.RailResult{Basis: "guess", Detail: "timeout guessed failed"}, fixture.now.Add(time.Second).Format(time.RFC3339Nano), fixture.now.Add(settlement.RetentionWindow).Format(time.RFC3339Nano))
	require.Error(t, err, "repository must reject unknown->failed without query provenance")
	stillUnknown, err := fixture.settlements.Get(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeUnknown, stillUnknown.Outcome)

	resolved, err := fixture.service.ResolveUnknown(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeFailed, resolved.Outcome)
	require.EqualValues(t, 1, adapter.queryCalls.Load())
	require.EqualValues(t, 1, adapter.settleCalls.Load())
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictReleased, auth.Verdict)
	require.Zero(t, auth.HoldMinor)
	replayed, err := evidence.NewRecorder(evidence.NewSQLiteRecorder(fixture.db)).Replay(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, "failed", replayed.Outcome)
	require.Contains(t, replayed.RailResponseJSON, "processor confirms no transfer was created")
}

// [REQ:TRS-P0-011] Once a rail call has returned, client cancellation cannot
// prevent Treasury from durably recording the observed outcome.
func TestOutcomeCommitSurvivesClientCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	adapter := &adapter{settleResult: settledResult(), cancel: cancel}
	fixture := newFixture(t, adapter)
	value, err := fixture.service.Settle(ctx, fixture.input)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, value.Outcome)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	stored, err := fixture.settlements.Get(context.Background(), value.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, stored.Outcome)
}

// [REQ:TRS-P0-011] A durable calling fence left by an interrupted process is
// conservatively recovered as unknown. A retry must never dispatch a second
// payment merely because the first process died before recording its reply.
func TestInterruptedCallingFenceBecomesUnknownWithoutRetry(t *testing.T) {
	fixture := newFixture(t, &adapter{settleResult: settledResult()})
	claim, err := fixture.settlements.Claim(context.Background(), settlement.Record{
		ID: "settlement-1", AuthorizationID: "auth-1", MandateID: "mandate-1",
		InstrumentID: "instrument-1", Rail: "test-rail", IdempotencyKey: "settle-key-1",
		AmountMinor: 250, Currency: "USD", Counterparty: "api.example",
		CreatedAt: fixture.now, UpdatedAt: fixture.now, RetainUntil: fixture.now.Add(settlement.RetentionWindow),
	})
	require.NoError(t, err)
	require.True(t, claim.Claimed)
	require.Equal(t, settlement.OutcomeCalling, claim.Record.Outcome)

	recovered, err := fixture.service.Settle(context.Background(), fixture.input)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeUnknown, recovered.Outcome)
	require.Equal(t, "interrupted_execution", recovered.Basis)
	require.Zero(t, fixture.adapter.settleCalls.Load(), "an interrupted external call must never be repeated")
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, auth.Verdict)
	require.EqualValues(t, 250, auth.HoldMinor, "unknown outcome retains the authorization hold")
}

// [REQ:TRS-P1-006] A kill switch engaged after authorization but before rail
// dispatch releases the hold and prevents the external side effect.
func TestFreezeStopsAlreadyAuthorizedSettlementBeforeRailDispatch(t *testing.T) {
	fixture := newFixture(t, &adapter{settleResult: settledResult()})
	freezes := budget.NewService(budget.NewSQLiteRepository(fixture.db), fixture.clock)
	_, err := freezes.SetScopeFrozen(context.Background(), budget.FreezeScopeBook, "book-1", true)
	require.NoError(t, err)
	service := settlement.NewService(fixture.settlements, fixture.authorizations, fixture.instruments, fixture.registry, identityVerifier{}, fixture.clock, freezes)

	_, err = service.Settle(context.Background(), fixture.input)
	require.ErrorIs(t, err, settlement.ErrInvalid)
	require.Zero(t, fixture.adapter.settleCalls.Load())
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictReleased, auth.Verdict)
	require.Zero(t, auth.HoldMinor)
}

// [REQ:TRS-P0-011] The manual rail is callable only through the operator path,
// and the service overwrites any asserted actor with the authenticated one.
func TestOperatorManualSettlementBindsActor(t *testing.T) {
	adapter := &adapter{name: "manual", settleResult: settledResult()}
	fixture := newFixture(t, adapter)
	input := fixture.input
	input.IdentityToken = ""
	input.Attestation = &rail.Attestation{ActorIdentity: "forged", ExternalReference: "bank-1", ReceiptReference: "receipt-1", OccurredAt: fixture.now}
	value, err := fixture.service.SettleOperator(context.Background(), input, "operator:verified")
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, value.Outcome)
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	require.Equal(t, "operator:verified", adapter.command.Attestation.ActorIdentity)
}

type fixture struct {
	now            time.Time
	db             *sql.DB
	adapter        *adapter
	service        *settlement.Service
	settlements    *settlement.SQLiteRepository
	authorizations *authorization.SQLiteRepository
	instruments    *instrument.Service
	registry       *rail.Registry
	clock          *schedule.Fake
	input          settlement.SettleInput
}

func newFixture(t *testing.T, railAdapter *adapter) fixture {
	t.Helper()
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle,
		database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema),
		database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(instrument.Schema), database.SchemaProviderFunc(settlement.Schema),
		database.SchemaProviderFunc(evidence.Schema), database.SchemaProviderFunc(ledger.Schema),
	))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operator", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 10_000, PeriodicCapMinor: 10_000, PerTransactionCapMinor: 1_000, Period: time.Hour, AllowedCounterparties: []string{"api.example"}})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("settlement-test-signing-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	_, err = mandates.Issue(ctx, mandate.IssueInput{ID: "mandate-1", IdempotencyKey: "mandate-key-1", BookID: "book-1", BudgetID: "budget-1", Authorizer: "operator:1", CapMinor: 1_000, Currency: "USD", AllowedCounterparties: []string{"api.example"}, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	registry, err := rail.NewRegistry(railAdapter)
	require.NoError(t, err)
	instruments := instrument.NewService(instrument.NewSQLiteRepository(handle), mandates, registry, credentialResolver{}, clock)
	_, err = instruments.Register(ctx, instrument.RegisterInput{ID: "instrument-1", MandateID: "mandate-1", Rail: railAdapter.Name(), CredentialReference: "treasury/test-credential", Counterparty: "api.example"})
	require.NoError(t, err)
	authorizations := authorization.NewSQLiteRepository(handle)
	_, err = authorizations.Create(ctx, authorization.Record{ID: "auth-1", IdempotencyKey: "auth-key-1", MandateID: "mandate-1", BudgetID: "budget-1", RequestingAgent: "agent:1", AmountMinor: 250, Currency: "USD", Counterparty: "api.example", Verdict: authorization.VerdictApproved, HoldMinor: 250, CreatedAt: now, ExpiresAt: now.Add(15 * time.Minute)})
	require.NoError(t, err)
	settlements := settlement.NewSQLiteRepository(handle)
	return fixture{
		now: now, db: handle, adapter: railAdapter, settlements: settlements, authorizations: authorizations, instruments: instruments, registry: registry, clock: clock,
		service: settlement.NewService(settlements, authorizations, instruments, registry, identityVerifier{}, clock),
		input:   settlement.SettleInput{ID: "settlement-1", AuthorizationID: "auth-1", InstrumentID: "instrument-1", IdempotencyKey: "settle-key-1", IdentityToken: "opaque-agent-token"},
	}
}

type credentialResolver struct{}

func (credentialResolver) Resolve(context.Context, string, string) (string, error) {
	return "in-memory-test-secret", nil
}

type identityVerifier struct{}

func (identityVerifier) Verify(_ context.Context, token string) (identity.Claims, error) {
	if token != "opaque-agent-token" {
		return identity.Claims{Subject: "agent:other"}, nil
	}
	return identity.Claims{Subject: "agent:1"}, nil
}

type adapter struct {
	settleCalls  atomic.Int64
	queryCalls   atomic.Int64
	settleResult rail.Result
	settleErr    error
	queryResults []rail.Result
	queryErr     error
	cancel       context.CancelFunc
	mu           sync.Mutex
	name         string
	command      rail.SettleCommand
}

func (a *adapter) Name() string {
	if a.name != "" {
		return a.name
	}
	return "test-rail"
}

func (a *adapter) Settle(_ context.Context, command rail.SettleCommand) (rail.Result, error) {
	a.settleCalls.Add(1)
	a.mu.Lock()
	a.command = command
	a.mu.Unlock()
	if command.MandateReference == "" || a.Name() != "manual" && command.Credential == "" {
		return rail.Result{}, errors.New("missing governed execution scope")
	}
	if a.cancel != nil {
		a.cancel()
	}
	return a.settleResult, a.settleErr
}

func (a *adapter) QueryOutcome(_ context.Context, query rail.Query) (rail.Result, error) {
	a.queryCalls.Add(1)
	if query.IdempotencyKey == "" {
		return rail.Result{}, errors.New("missing idempotency query key")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.queryErr != nil {
		return rail.Result{}, a.queryErr
	}
	if len(a.queryResults) == 0 {
		return rail.Result{Outcome: rail.OutcomeUnknown}, nil
	}
	result := a.queryResults[0]
	a.queryResults = a.queryResults[1:]
	return result, nil
}

func settledResult() rail.Result {
	return rail.Result{Outcome: rail.OutcomeSettled, ExternalID: "processor-charge-1", ReceiptReference: "receipt-1", Basis: "processor_confirmation", Detail: "processor confirmed transfer", OccurredAt: time.Date(2026, 8, 18, 12, 0, 1, 0, time.UTC)}
}

var _ rail.Adapter = (*adapter)(nil)
