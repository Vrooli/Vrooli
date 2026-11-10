package settlement_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"treasury/internal/testutil/agentclaims"

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

func TestSettlementRejectsChangedPreapprovedIntentBeforeRailDispatch(t *testing.T) {
	for name, mutate := range map[string]func(*settlement.SettleInput){
		"cross authorization":   func(in *settlement.SettleInput) { in.AuthorizationID = "auth-other" },
		"changed settlement ID": func(in *settlement.SettleInput) { in.ID = "settlement-other" },
		"changed instrument":    func(in *settlement.SettleInput) { in.InstrumentID = "instrument-other" },
		"changed retry key":     func(in *settlement.SettleInput) { in.IdempotencyKey = "settle-key-other" },
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newFixture(t, &adapter{settleResult: settledResult()})
			attempt := fixture.input
			mutate(&attempt)
			_, err := fixture.service.Settle(context.Background(), attempt)
			require.Error(t, err)
			require.Zero(t, fixture.adapter.settleCalls.Load(), "altered or cross-intent terms must be rejected before dispatch")
			var count int
			require.NoError(t, fixture.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM settlements`).Scan(&count))
			require.Zero(t, count, "a denied altered intent must not reserve a settlement")
		})
	}
}

// Settlement and authorization projection are separate durable owners. If
// projection fails after settlement completion, an idempotent retry repairs it
// without invoking the rail again.
func TestRetryRepairsAuthorizationAfterProjectionFailure(t *testing.T) {
	fake := &adapter{settleResult: settledResult()}
	fixture := newFixture(t, fake)
	projection := &failOnceAuthorizationProjection{Authorizations: fixture.authorizations}
	fixture.service = settlement.NewService(fixture.settlements, projection, fixture.instruments, fixture.registry, identityVerifier{}, fixture.clock)

	_, err := fixture.service.Settle(context.Background(), fixture.input)
	require.ErrorContains(t, err, "project settlement to authorization")
	stored, err := fixture.settlements.Get(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, stored.Outcome)
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, auth.Verdict, "failed projection leaves the original hold for retry")

	retried, err := fixture.service.Settle(context.Background(), fixture.input)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, retried.Outcome)
	asserted, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictSettled, asserted.Verdict)
	require.Zero(t, asserted.HoldMinor)
	require.EqualValues(t, 1, fake.settleCalls.Load(), "projection repair must not repeat the external call")
	require.EqualValues(t, 2, projection.attempts.Load())
}

type failOnceAuthorizationProjection struct {
	settlement.Authorizations
	attempts atomic.Int64
}

func (p *failOnceAuthorizationProjection) Settle(ctx context.Context, id string) (authorization.Record, error) {
	if p.attempts.Add(1) == 1 {
		return authorization.Record{}, errors.New("simulated authorization store outage")
	}
	return p.Authorizations.Settle(ctx, id)
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
	require.ErrorIs(t, err, settlement.ErrInProgress)
	require.Equal(t, settlement.OutcomeCalling, recovered.Outcome)
	require.Zero(t, fixture.adapter.settleCalls.Load(), "an interrupted external call must never be repeated")
	recovered, err = fixture.service.ResolveUnknown(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeUnknown, recovered.Outcome)
	require.Equal(t, "interrupted_execution", recovered.Basis)
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, auth.Verdict)
	require.EqualValues(t, 250, auth.HoldMinor, "unknown outcome retains the authorization hold")
}

// [REQ:TRS-P0-011] A crash after x402 recovery data is committed but before
// the adapter reply is recorded remains resolvable by replaying the same
// signed authorization and idempotency key.
func TestInterruptedX402CallRetainsAndReplaysExactRecoveryInput(t *testing.T) {
	payload := "exact-synthetic-x402-signature"
	digestBytes := sha256.Sum256([]byte(payload))
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	fake := &adapter{
		name:                      "x402",
		recordRecoveryBeforeCrash: true,
		crashAfterRecoveryInput:   true,
		queryResults: []rail.Result{{
			Outcome: rail.OutcomeSettled, ExternalID: "0xconfirmed", ReceiptReference: digest,
			Basis: "x402_facilitator_replay_confirmation", Detail: "synthetic exact-payload query confirmation", OccurredAt: time.Date(2026, 8, 18, 12, 0, 1, 0, time.UTC),
		}},
	}
	fixture := newFixture(t, fake)

	require.Panics(t, func() { _, _ = fixture.service.Settle(context.Background(), fixture.input) })
	orphan, err := fixture.settlements.Get(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeCalling, orphan.Outcome)
	require.Equal(t, digest, orphan.ExternalID)
	require.NotEqual(t, payload, orphan.ReceiptReference)
	require.Contains(t, orphan.ReceiptReference, "x402-recovery")
	require.EqualValues(t, 1, fake.settleCalls.Load())

	recovered, err := fixture.service.Settle(context.Background(), fixture.input)
	require.ErrorIs(t, err, settlement.ErrInProgress)
	require.Equal(t, settlement.OutcomeCalling, recovered.Outcome)
	require.EqualValues(t, 1, fake.settleCalls.Load(), "Settle retry must not dispatch again")

	resolved, err := fixture.service.ResolveUnknown(context.Background(), orphan.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeSettled, resolved.Outcome)
	require.Equal(t, "0xconfirmed", resolved.ExternalID)
	require.EqualValues(t, 1, fake.queryCalls.Load())
	require.EqualValues(t, 1, fake.dispatchCalls.Load(), "query must replay/resolve without a second initial dispatch")
	fake.mu.Lock()
	query := fake.query
	fake.mu.Unlock()
	require.Equal(t, digest, query.ExternalID)
	require.Equal(t, payload, query.ReceiptReference)
	require.Equal(t, fixture.input.IdempotencyKey, query.IdempotencyKey)
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictSettled, auth.Verdict)
	require.Zero(t, auth.HoldMinor)
	_, retained := recoveryInputs.Load(orphan.ReceiptReference)
	require.False(t, retained, "terminal reconciliation removes the protected replay authorization")
}

// A newly versioned x402 call that dies before preparing recovery data cannot
// have sent its paid request. Legacy calls without the marker stay unknown.
func TestInterruptedX402PreflightOnlyCallFailsWithoutPayment(t *testing.T) {
	fake := &adapter{name: "x402", crashBeforeRecoveryInput: true}
	fixture := newFixture(t, fake)
	require.Panics(t, func() { _, _ = fixture.service.Settle(context.Background(), fixture.input) })
	orphan, err := fixture.settlements.Get(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.BasisX402PreflightOnly, orphan.Basis)
	require.Empty(t, orphan.ExternalID)
	require.Empty(t, orphan.ReceiptReference)

	recovered, err := fixture.service.Settle(context.Background(), fixture.input)
	require.ErrorIs(t, err, settlement.ErrInProgress)
	recovered, err = fixture.service.ResolveUnknown(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeFailed, recovered.Outcome)
	require.Equal(t, "x402_dispatch_not_started", recovered.Basis)
	require.Zero(t, fake.dispatchCalls.Load())
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictReleased, auth.Verdict)
	require.Zero(t, auth.HoldMinor)
}

// A process can stop after Credential Authority stores the signed payload but
// before SQLite records its reference. The durable preflight marker proves no
// request was dispatched, so recovery must remove that orphan before release.
func TestInterruptedX402BeforeReferenceCommitDeletesOrphan(t *testing.T) {
	fake := &adapter{name: "x402", recordRecoveryBeforeCrash: true}
	fixture := newFixture(t, fake)
	fixture.service = settlement.NewService(recordRecoveryPanicRepository{Repository: fixture.settlements}, fixture.authorizations, fixture.instruments, fixture.registry, identityVerifier{}, fixture.clock)

	require.Panics(t, func() { _, _ = fixture.service.Settle(context.Background(), fixture.input) })
	stored, err := fixture.settlements.Get(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.BasisX402PreflightOnly, stored.Basis)
	require.Empty(t, stored.ReceiptReference, "SQLite never committed the CA reference")
	identity := "vrooli/treasury/x402-recovery/" + func() string {
		digest := sha256.Sum256([]byte(fixture.input.ID))
		return hex.EncodeToString(digest[:])
	}()
	_, exists := recoveryInputs.Load(identity)
	require.True(t, exists, "the simulated interrupted process left an unreferenced CA value")

	recovered, err := fixture.service.ResolveUnknown(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeFailed, recovered.Outcome)
	_, exists = recoveryInputs.Load(identity)
	require.False(t, exists, "recovery deletes the orphaned replay authorization")
	require.Zero(t, fake.dispatchCalls.Load())
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictReleased, auth.Verdict)
}

type recordRecoveryPanicRepository struct{ settlement.Repository }

func (recordRecoveryPanicRepository) RecordRecoveryInput(context.Context, string, string, string) error {
	panic("simulated process interruption before SQLite reference commit")
}

func TestInterruptedLegacyX402CallWithoutPhaseMarkerRemainsUnknown(t *testing.T) {
	fake := &adapter{name: "x402"}
	fixture := newFixture(t, fake)
	claim, err := fixture.settlements.Claim(context.Background(), settlement.Record{
		ID: fixture.input.ID, AuthorizationID: fixture.input.AuthorizationID, MandateID: "mandate-1",
		InstrumentID: fixture.input.InstrumentID, Rail: "x402", IdempotencyKey: fixture.input.IdempotencyKey,
		AmountMinor: 250, Currency: "USD", Counterparty: "api.example",
		CreatedAt: fixture.now, UpdatedAt: fixture.now, RetainUntil: fixture.now.Add(settlement.RetentionWindow),
	})
	require.NoError(t, err)
	require.True(t, claim.Claimed)

	recovered, err := fixture.service.Settle(context.Background(), fixture.input)
	require.ErrorIs(t, err, settlement.ErrInProgress)
	require.Equal(t, settlement.OutcomeCalling, recovered.Outcome)
	recovered, err = fixture.service.ResolveUnknown(context.Background(), fixture.input.ID)
	require.Error(t, err, "legacy x402 has no exact retained replay authorization")
	recovered, err = fixture.settlements.Get(context.Background(), fixture.input.ID)
	require.NoError(t, err)
	require.Equal(t, settlement.OutcomeUnknown, recovered.Outcome)
	require.Empty(t, recovered.ExternalID)
	require.Empty(t, recovered.ReceiptReference)
	auth, err := fixture.authorizations.Get(context.Background(), "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictApproved, auth.Verdict)
	require.EqualValues(t, 250, auth.HoldMinor)
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
	_, err = authorizations.Create(ctx, authorization.Record{ID: "auth-1", IdempotencyKey: "auth-key-1", BookID: "book-1", MandateID: "mandate-1", BudgetID: "budget-1", RequestingAgent: "agent:1", AmountMinor: 250, Currency: "USD", Counterparty: "api.example", SettlementID: "settlement-1", InstrumentID: "instrument-1", SettlementIdempotencyKey: "settle-key-1", Verdict: authorization.VerdictApproved, HoldMinor: 250, CreatedAt: now, ExpiresAt: now.Add(15 * time.Minute)})
	require.NoError(t, err)
	settlements := settlement.NewSQLiteRepository(handle)
	return fixture{
		now: now, db: handle, adapter: railAdapter, settlements: settlements, authorizations: authorizations, instruments: instruments, registry: registry, clock: clock,
		service: settlement.NewService(settlements, authorizations, instruments, registry, identityVerifier{}, clock),
		input:   settlement.SettleInput{ID: "settlement-1", AuthorizationID: "auth-1", InstrumentID: "instrument-1", IdempotencyKey: "settle-key-1", IdentityToken: "opaque-agent-token"},
	}
}

type credentialResolver struct{}

var recoveryInputs sync.Map

func (credentialResolver) Resolve(_ context.Context, reference, field string) (string, error) {
	if field == "signed-payment" {
		value, ok := recoveryInputs.Load(reference)
		if !ok {
			return "", errors.New("missing synthetic recovery value")
		}
		return value.(string), nil
	}
	return "in-memory-test-secret", nil
}

func (credentialResolver) Store(_ context.Context, ref, field, payload string) error {
	recoveryInputs.Store(ref, payload)
	return nil
}

func (credentialResolver) Delete(_ context.Context, ref, field string) error {
	recoveryInputs.Delete(ref)
	return nil
}

type identityVerifier struct{}

func (identityVerifier) Verify(_ context.Context, token string) (identity.Claims, error) {
	if token != "opaque-agent-token" {
		return agentclaims.For("agent:other"), nil
	}
	return agentclaims.For("agent:1"), nil
}

type adapter struct {
	settleCalls               atomic.Int64
	queryCalls                atomic.Int64
	settleResult              rail.Result
	settleErr                 error
	queryResults              []rail.Result
	queryErr                  error
	cancel                    context.CancelFunc
	recordRecoveryBeforeCrash bool
	crashBeforeRecoveryInput  bool
	crashAfterRecoveryInput   bool
	dispatchCalls             atomic.Int64
	query                     rail.Query
	mu                        sync.Mutex
	name                      string
	command                   rail.SettleCommand
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
	if a.crashBeforeRecoveryInput {
		panic("simulated process interruption during x402 preflight")
	}
	if a.recordRecoveryBeforeCrash {
		if command.RecordRecoveryInput == nil {
			return rail.Result{}, errors.New("missing recovery recorder")
		}
		payload := "exact-synthetic-x402-signature"
		digestBytes := sha256.Sum256([]byte(payload))
		digest := "sha256:" + hex.EncodeToString(digestBytes[:])
		if err := command.RecordRecoveryInput(context.Background(), digest, payload); err != nil {
			return rail.Result{}, err
		}
	}
	if a.crashAfterRecoveryInput {
		a.dispatchCalls.Add(1)
		panic("simulated process interruption after possible x402 dispatch")
	}
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
	a.query = query
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
