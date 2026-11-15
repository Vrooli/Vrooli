package x402

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"treasury/internal/ledger"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

// [REQ:TRS-P1-002] An unpaid caller receives a complete v2 challenge, while a
// valid payment payload is settled once and a replay returns the same receipt.
func TestInboundChallengeAndReplaySafeAdmission(t *testing.T) {
	gate, _, facilitator, price := newInboundFixture(t)
	header, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	require.NotEmpty(t, header)
	decoded, err := base64.StdEncoding.DecodeString(header)
	require.NoError(t, err)
	require.Contains(t, string(decoded), `"x402Version":2`)

	_, err = gate.Admit(context.Background(), price.ID, "")
	require.ErrorIs(t, err, ErrInvalid)
	require.Zero(t, facilitator.settleCalls.Load())

	payment := paymentForChallenge(t, header, "nonce-1")
	first, err := gate.Admit(context.Background(), price.ID, payment)
	require.NoError(t, err)
	require.Equal(t, "settled", first.Status)
	require.NotEmpty(t, first.TransactionID)
	replay, err := gate.Admit(context.Background(), price.ID, payment)
	require.NoError(t, err)
	require.Equal(t, first, replay)
	require.EqualValues(t, 1, facilitator.settleCalls.Load())
	require.EqualValues(t, 1, facilitator.verifyCalls.Load(), "a settled replay must not depend on facilitator availability")
}

func TestResourceAdmissionVerifiesFirstAndSettlesOnceAfterBinding(t *testing.T) {
	gate, handle, facilitator, price := newInboundFixture(t)
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	payment := paymentForChallenge(t, challenge, "resource-nonce-1")

	verified, err := gate.VerifyForResource(context.Background(), price.ID, "job-1", "resource-1", "order-1", payment)
	require.NoError(t, err)
	require.Equal(t, "verified", verified.Status)
	require.NotEmpty(t, verified.ID)
	require.Equal(t, "job-1", verified.JobID)
	require.Zero(t, facilitator.settleCalls.Load(), "verification must not settle before the resource verifier passes")

	replay, err := gate.VerifyForResource(context.Background(), price.ID, "job-1", "resource-1", "order-1", payment)
	require.NoError(t, err)
	require.Equal(t, verified.ID, replay.ID)
	require.EqualValues(t, 1, facilitator.verifyCalls.Load(), "verified replay should reuse the durable admission")

	_, err = gate.VerifyForResource(context.Background(), price.ID, "job-other", "resource-1", "order-1", payment)
	require.ErrorIs(t, err, ErrBindingConflict, "one payment payload cannot be rebound to another job")

	settled, err := gate.SettleAuthorization(context.Background(), verified.ID, "job-1", "resource-1", "order-1")
	require.NoError(t, err)
	require.Equal(t, "settled", settled.Status)
	require.NotEmpty(t, settled.TransactionID)
	require.EqualValues(t, 1, facilitator.settleCalls.Load())

	settledReplay, err := gate.SettleAuthorization(context.Background(), verified.ID, "job-1", "resource-1", "order-1")
	require.NoError(t, err)
	require.Equal(t, settled.ID, settledReplay.ID)
	require.Equal(t, "", settledReplay.RecoveryReference, "terminal settlement should not retain the protected signature reference")
	require.EqualValues(t, 1, facilitator.settleCalls.Load(), "settled retries must not call facilitator again")
	var emissions int
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ledger_emissions WHERE settlement_id=?`, verified.ID).Scan(&emissions))
	require.Equal(t, 1, emissions, "Treasury must own exactly one seller-side ledger emission")
	_, vaultErr := gate.recoveryVault.ResolveSettlementRecoveryInput(context.Background(), verified.RecoveryReference)
	require.ErrorIs(t, vaultErr, ErrNotFound, "terminal settlement should remove the protected payment payload")
}

func TestPaymentPayloadCannotCrossBetweenImmediateAndDeferredAdmission(t *testing.T) {
	t.Run("resource authorization blocks legacy immediate settlement", func(t *testing.T) {
		gate, _, facilitator, price := newInboundFixture(t)
		challenge, err := gate.PaymentRequired(context.Background(), price.ID)
		require.NoError(t, err)
		payment := paymentForChallenge(t, challenge, "cross-route-resource-first")
		_, err = gate.VerifyForResource(context.Background(), price.ID, "job-cross", "resource-cross", "order-cross", payment)
		require.NoError(t, err)

		_, err = gate.Admit(context.Background(), price.ID, payment)
		require.ErrorIs(t, err, ErrBindingConflict)
		require.Zero(t, facilitator.settleCalls.Load(), "a legacy route must not settle an authorization owned by a deferred job")
	})

	t.Run("legacy immediate admission blocks deferred authorization", func(t *testing.T) {
		gate, _, facilitator, price := newInboundFixture(t)
		challenge, err := gate.PaymentRequired(context.Background(), price.ID)
		require.NoError(t, err)
		payment := paymentForChallenge(t, challenge, "cross-route-legacy-first")
		_, err = gate.Admit(context.Background(), price.ID, payment)
		require.NoError(t, err)
		require.EqualValues(t, 1, facilitator.settleCalls.Load())

		_, err = gate.VerifyForResource(context.Background(), price.ID, "job-cross", "resource-cross", "order-cross", payment)
		require.ErrorIs(t, err, ErrBindingConflict)
		require.EqualValues(t, 1, facilitator.settleCalls.Load(), "deferred route must not repeat a legacy settlement")
	})
}

func TestResourceAuthorizationFailsClosedWithoutProtectedPayloadVault(t *testing.T) {
	gate, _, facilitator, price := newInboundFixture(t)
	gate.SetRecoveryVault(nil)
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	_, err = gate.VerifyForResource(context.Background(), price.ID, "job-no-vault", "resource-no-vault", "order-no-vault", paymentForChallenge(t, challenge, "no-vault-nonce"))
	require.ErrorIs(t, err, ErrRecoveryUnavailable)
	require.Zero(t, facilitator.verifyCalls.Load(), "verification must not claim a payment that Treasury cannot recover after restart")
}

func TestResourceAdmissionCancellationReleasesOnlyUnsettledBoundAuthorization(t *testing.T) {
	gate, _, facilitator, price := newInboundFixture(t)
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	payment := paymentForChallenge(t, challenge, "cancel-nonce")
	verified, err := gate.VerifyForResource(context.Background(), price.ID, "job-cancel", "resource-cancel", "order-cancel", payment)
	require.NoError(t, err)

	cancelled, err := gate.CancelAuthorization(context.Background(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", cancelled.Status)
	require.EqualValues(t, 0, facilitator.settleCalls.Load(), "cancellation must not call facilitator settlement")
	replay, err := gate.CancelAuthorization(context.Background(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", replay.Status)
	_, err = gate.SettleAuthorization(context.Background(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
	require.ErrorIs(t, err, ErrStateConflict, "cancelled authority cannot later be settled")

	second, err := gate.VerifyForResource(context.Background(), price.ID, "job-settling", "resource-settling", "order-settling", paymentForChallenge(t, challenge, "settle-first-nonce"))
	require.NoError(t, err)
	_, err = gate.SettleAuthorization(context.Background(), second.ID, second.JobID, second.ResourceID, second.OrderID)
	require.NoError(t, err)
	_, err = gate.CancelAuthorization(context.Background(), second.ID, second.JobID, second.ResourceID, second.OrderID)
	require.ErrorIs(t, err, ErrStateConflict, "terminal settlement cannot be rewritten as cancellation")
	_, err = gate.CancelAuthorization(context.Background(), second.ID, "other-job", second.ResourceID, second.OrderID)
	require.ErrorIs(t, err, ErrBindingConflict, "cancellation identity must match the durable order binding")
}

func TestResourceAdmissionRecoversUnknownOutcomeWithSameAdmission(t *testing.T) {
	gate, handle, facilitator, price := newInboundFixture(t)
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	payment := paymentForChallenge(t, challenge, "resource-recovery-nonce")
	verified, err := gate.VerifyForResource(context.Background(), price.ID, "job-recovery", "resource-recovery", "order-recovery", payment)
	require.NoError(t, err)

	facilitator.settleFailure.Store(true)
	first, err := gate.SettleAuthorization(context.Background(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
	require.ErrorIs(t, err, ErrUnknown)
	require.Equal(t, "unknown", first.Status)
	recoveredFromVerify, err := gate.VerifyForResource(context.Background(), price.ID, verified.JobID, verified.ResourceID, verified.OrderID, payment)
	require.NoError(t, err, "a verify replay must recover the durable admission ID after a lost settle response")
	require.Equal(t, verified.ID, recoveredFromVerify.ID)
	require.Equal(t, "unknown", recoveredFromVerify.Status)

	// The retry uses the same admission and exact authorization payload. It
	// may become conclusive without creating another Treasury ledger event.
	settled, err := gate.SettleAuthorization(context.Background(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
	require.NoError(t, err)
	require.Equal(t, "settled", settled.Status)
	require.Equal(t, verified.ID, settled.ID)
	require.EqualValues(t, 2, facilitator.settleCalls.Load())
	var emissions int
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ledger_emissions WHERE settlement_id=?`, verified.ID).Scan(&emissions))
	require.Equal(t, 1, emissions)
}

func TestResourceAdmissionCarriesExplicitRuntimeMode(t *testing.T) {
	_, handle, facilitator, price := newInboundFixture(t)
	repository := NewSQLiteInboundRepository(handle)
	gate, err := NewGate(repository, facilitator)
	require.NoError(t, err)
	require.Error(t, gate.SetRuntimeMode("local-fork"), "local-fork label must not be accepted without a loopback-qualified facilitator")
	require.Error(t, gate.SetRuntimeMode("mainnet-ish"), "unknown runtime modes must fail closed")

	localGate, err := NewGate(repository, localOnlyFixture{fixtureFacilitator: facilitator})
	require.NoError(t, err)
	localGate.SetRecoveryVault(&fixtureRecoveryVault{values: make(map[string]string)})
	require.NoError(t, localGate.SetRuntimeMode("local-fork"))
	challenge, err := localGate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	verified, err := localGate.VerifyForResource(context.Background(), price.ID, "job-mode", "resource-mode", "order-mode", paymentForChallenge(t, challenge, "mode-nonce"))
	require.NoError(t, err)
	require.Equal(t, "local-fork", verified.Mode)
}

func TestResourceAdmissionReclaimsStaleInFlightSettlement(t *testing.T) {
	gate, handle, facilitator, price := newInboundFixture(t)
	challenge, err := gate.PaymentRequired(context.Background(), price.ID)
	require.NoError(t, err)
	payment := paymentForChallenge(t, challenge, "stale-settle-nonce")
	verified, err := gate.VerifyForResource(context.Background(), price.ID, "job-stale", "resource-stale", "order-stale", payment)
	require.NoError(t, err)
	_, err = handle.ExecContext(context.Background(), `UPDATE x402_resource_admissions SET status='settling',updated_at=? WHERE id=?`, gate.now().UTC().Add(-2*time.Minute).Format(time.RFC3339Nano), verified.ID)
	require.NoError(t, err)

	recovered, err := gate.SettleAuthorization(context.Background(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
	require.NoError(t, err)
	require.Equal(t, "settled", recovered.Status)
	require.EqualValues(t, 1, facilitator.settleCalls.Load())
}

func newInboundFixture(t *testing.T) (*Gate, databaseHandle, *fixtureFacilitator, Price) {
	t.Helper()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), handle,
		database.SchemaProviderFunc(ledger.Schema), database.SchemaProviderFunc(Schema),
	))
	facilitator := &fixtureFacilitator{}
	gate, err := NewGate(NewSQLiteInboundRepository(handle), facilitator)
	require.NoError(t, err)
	gate.SetRecoveryVault(&fixtureRecoveryVault{values: make(map[string]string)})
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	gate.now = func() time.Time { return now }
	price, err := gate.Declare(context.Background(), Price{
		ID: "price-1", ResourceURL: "https://service.example/paid", Description: "fixture", MIMEType: "application/json",
		Network: "eip155:8453", Scheme: "exact", Amount: "10000", AmountMinor: 1, Currency: "USD",
		PayTo: "0x3333333333333333333333333333333333333333", Asset: "0x2222222222222222222222222222222222222222",
		AssetDecimals: 6, MaxTimeoutSeconds: 300,
		ExtraJSON: `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
	})
	require.NoError(t, err)
	return gate, handle, facilitator, price
}

type databaseHandle interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func paymentForChallenge(t *testing.T, challenge, nonce string) string {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(challenge)
	require.NoError(t, err)
	var required struct {
		Accepts []json.RawMessage `json:"accepts"`
	}
	require.NoError(t, json.Unmarshal(decoded, &required))
	require.Len(t, required.Accepts, 1)
	var accepted any
	require.NoError(t, json.Unmarshal(required.Accepts[0], &accepted))
	payload, err := json.Marshal(map[string]any{
		"x402Version": 2, "accepted": accepted,
		"payload":    map[string]any{"signature": "0xfixture", "authorization": map[string]any{"nonce": nonce}},
		"extensions": map[string]any{},
	})
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(payload)
}

type fixtureFacilitator struct {
	verifyCalls   atomic.Int64
	settleCalls   atomic.Int64
	settleFailure atomic.Bool
	delay         time.Duration
}

func (f *fixtureFacilitator) Verify(_ context.Context, payload, _ json.RawMessage) (VerifyResult, error) {
	f.verifyCalls.Add(1)
	digest := sha256.Sum256(payload)
	return VerifyResult{Valid: true, Payer: "payer:" + hex.EncodeToString(digest[:4])}, nil
}

func (f *fixtureFacilitator) Settle(_ context.Context, payload, _ json.RawMessage) (SettleResult, error) {
	f.settleCalls.Add(1)
	if f.settleFailure.Swap(false) {
		return SettleResult{}, errors.New("facilitator response lost after dispatch")
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	digest := sha256.Sum256(payload)
	return SettleResult{Success: true, Payer: "payer:" + hex.EncodeToString(digest[:4]), Transaction: "tx:" + hex.EncodeToString(digest[:]), Network: "eip155:8453"}, nil
}

var _ Facilitator = (*fixtureFacilitator)(nil)

type localOnlyFixture struct{ *fixtureFacilitator }

func (localOnlyFixture) LocalOnly() bool { return true }

type fixtureRecoveryVault struct {
	mu     sync.Mutex
	values map[string]string
}

func (v *fixtureRecoveryVault) StoreSettlementRecoveryInput(_ context.Context, id, payload string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.values[id] != "" && v.values[id] != payload {
		return "", errors.New("recovery payload identity conflict")
	}
	v.values[id] = payload
	return id, nil
}

func (v *fixtureRecoveryVault) ResolveSettlementRecoveryInput(_ context.Context, reference string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	value := v.values[reference]
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}

func (v *fixtureRecoveryVault) DeleteSettlementRecoveryInput(_ context.Context, reference string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.values, reference)
	return nil
}
