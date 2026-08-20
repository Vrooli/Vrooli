package x402

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"treasury/internal/rail"

	"github.com/stretchr/testify/require"
)

// [REQ:TRS-P1-001] A quoted x402 amount above the authorization's
// mandate-derived per-call cap is refused before any wallet signature exists.
func TestOutboundRefusesQuoteAbovePerCallCapBeforeSigning(t *testing.T) {
	var endpoint string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "20000"))
		writer.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"

	signer := &fixtureSigner{}
	adapter, err := New(server.Client(), signer)
	require.NoError(t, err)
	adapter.now = func() time.Time { return time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC) }
	result, err := adapter.Settle(context.Background(), command(t, endpoint, 1))
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)
	require.Contains(t, result.Detail, "does not equal the authorized per-call amount")
	require.Zero(t, signer.calls.Load(), "refused prices must never reach the wallet signer")
}

func command(t *testing.T, endpoint string, capMinor int64) rail.SettleCommand {
	t.Helper()
	credential, err := json.Marshal(Credential{
		EndpointURL: endpoint, SignerURL: "http://127.0.0.1:8545",
		Account:  "0x1111111111111111111111111111111111111111",
		Networks: []string{"eip155:8453"},
		Assets:   map[string]AssetPolicy{"0x2222222222222222222222222222222222222222": {Decimals: 6, Currency: "USD"}},
	})
	require.NoError(t, err)
	return rail.SettleCommand{
		SettlementID: "settlement-1", AuthorizationID: "auth-1", MandateReference: "mandate-1",
		InstrumentID: "instrument-1", IdempotencyKey: "settle-key-1", AmountMinor: capMinor,
		Currency: "USD", Counterparty: strings.Split(strings.TrimPrefix(endpoint, "http://"), ":")[0],
		Credential: string(credential),
	}
}

func paymentRequiredValue(t *testing.T, endpoint, amount string) string {
	t.Helper()
	value := map[string]any{
		"x402Version": 2,
		"resource":    map[string]any{"url": endpoint, "description": "fixture", "mimeType": "application/json"},
		"accepts": []any{map[string]any{
			"scheme": "exact", "network": "eip155:8453", "amount": amount,
			"payTo": "0x3333333333333333333333333333333333333333", "maxTimeoutSeconds": 300,
			"asset": "0x2222222222222222222222222222222222222222",
			"extra": map[string]any{"assetTransferMethod": "eip3009", "name": "USD Coin", "version": "2"},
		}},
		"extensions": map[string]any{},
	}
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(encoded)
}

type fixtureSigner struct {
	calls atomic.Int64
	typed map[string]any
}

func (s *fixtureSigner) SignTypedData(_ context.Context, _ Credential, typed map[string]any) (string, error) {
	s.calls.Add(1)
	s.typed = typed
	return "0x" + strings.Repeat("11", 65), nil
}
