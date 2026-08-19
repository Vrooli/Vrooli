package x402

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"treasury/internal/ledger"
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

func newInboundFixture(t *testing.T) (*Gate, databaseHandle, *fixtureFacilitator, Price) {
	t.Helper()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), handle,
		database.SchemaProviderFunc(ledger.Schema), database.SchemaProviderFunc(Schema),
	))
	facilitator := &fixtureFacilitator{}
	gate, err := NewGate(NewSQLiteInboundRepository(handle), facilitator)
	require.NoError(t, err)
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
	verifyCalls atomic.Int64
	settleCalls atomic.Int64
	delay       time.Duration
}

func (f *fixtureFacilitator) Verify(_ context.Context, payload, _ json.RawMessage) (VerifyResult, error) {
	f.verifyCalls.Add(1)
	digest := sha256.Sum256(payload)
	return VerifyResult{Valid: true, Payer: "payer:" + hex.EncodeToString(digest[:4])}, nil
}

func (f *fixtureFacilitator) Settle(_ context.Context, payload, _ json.RawMessage) (SettleResult, error) {
	f.settleCalls.Add(1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	digest := sha256.Sum256(payload)
	return SettleResult{Success: true, Payer: "payer:" + hex.EncodeToString(digest[:4]), Transaction: "tx:" + hex.EncodeToString(digest[:]), Network: "eip155:8453"}, nil
}

var _ Facilitator = (*fixtureFacilitator)(nil)
