package x402

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"treasury/internal/rail"

	"github.com/stretchr/testify/require"
)

// [REQ:TRS-P1-001] A local priced endpoint exercises the complete 402,
// operator-wallet signature, paid retry, and receipt cycle in one rail call.
func TestOutboundPayAndRetryReturnsFacilitatorReceipt(t *testing.T) {
	var endpoint string
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		payload, err := base64.StdEncoding.DecodeString(request.Header.Get(paymentSignatureHeader))
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(payload, &body))
		require.Equal(t, float64(2), body["x402Version"])
		require.NotNil(t, body["accepted"])
		receipt, err := json.Marshal(map[string]any{"success": true, "payer": "0x1111111111111111111111111111111111111111", "transaction": "0xabc123", "network": "eip155:8453"})
		require.NoError(t, err)
		writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"

	signer := &fixtureSigner{}
	adapter, err := New(server.Client(), signer)
	require.NoError(t, err)
	adapter.now = func() time.Time { return time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC) }
	result, err := adapter.Settle(context.Background(), command(t, endpoint, 1))
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, result.Outcome)
	require.Equal(t, "0xabc123", result.ExternalID)
	require.Equal(t, "x402_facilitator_confirmation", result.Basis)
	require.NotEmpty(t, result.ReceiptReference)
	require.EqualValues(t, 1, signer.calls.Load())
	require.EqualValues(t, 2, attempts.Load())
	require.NotNil(t, signer.typed["message"])
}
