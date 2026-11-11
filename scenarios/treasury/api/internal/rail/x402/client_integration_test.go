package x402

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestOutboundPersistsExactPayloadBeforePaidDispatch(t *testing.T) {
	var endpoint string
	var recordedDigest, recordedPayload string
	var recordedMu sync.Mutex
	var paidRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		paidRequests.Add(1)
		recordedMu.Lock()
		digest, payload := recordedDigest, recordedPayload
		recordedMu.Unlock()
		require.NotEmpty(t, payload, "recovery payload must be durable before dispatch")
		require.NotEmpty(t, digest)
		require.Equal(t, payload, request.Header.Get(paymentSignatureHeader))
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	cmd := command(t, endpoint, 1)
	cmd.RecordRecoveryInput = func(_ context.Context, digest, payload string) error {
		recordedMu.Lock()
		defer recordedMu.Unlock()
		recordedDigest, recordedPayload = digest, payload
		return nil
	}

	result, err := adapter.Settle(context.Background(), cmd)
	require.Error(t, err)
	require.Equal(t, rail.OutcomeUnknown, result.Outcome)
	recordedMu.Lock()
	digest, payload := recordedDigest, recordedPayload
	recordedMu.Unlock()
	require.Equal(t, digest, result.ExternalID)
	require.Equal(t, payload, result.ReceiptReference)
	require.EqualValues(t, 1, paidRequests.Load())
}

func TestOutboundDoesNotDispatchWhenRecoveryInputCannotBePersisted(t *testing.T) {
	var endpoint string
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.Header.Get(paymentSignatureHeader) != "" {
			t.Error("paid request was sent despite recovery persistence failure")
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
		writer.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	cmd := command(t, endpoint, 1)
	cmd.RecordRecoveryInput = func(context.Context, string, string) error { return errors.New("disk unavailable") }

	result, err := adapter.Settle(context.Background(), cmd)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)
	require.Equal(t, "x402_recovery_not_durable", result.Basis)
	require.EqualValues(t, 1, attempts.Load(), "only the initial 402 price fetch is allowed")
}

func TestQueryOutcomeRejectsChangedEndpointPathBeforeReplay(t *testing.T) {
	var endpoint string
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced/original"
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	cmd := command(t, endpoint, 1)
	first, err := adapter.Settle(context.Background(), cmd)
	require.Error(t, err)
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	requestsBeforeQuery := attempts.Load()

	credential := Credential{
		EndpointURL: server.URL + "/priced/rotated", SignerURL: "http://127.0.0.1:8545",
		Account: "0x1111111111111111111111111111111111111111", Networks: []string{"eip155:8453"},
		Assets: map[string]AssetPolicy{"0x2222222222222222222222222222222222222222": {Decimals: 6, Currency: "USD"}},
	}
	credentialJSON, err := json.Marshal(credential)
	require.NoError(t, err)
	_, err = adapter.QueryOutcome(context.Background(), rail.Query{
		ExternalID: first.ExternalID, ReceiptReference: first.ReceiptReference,
		IdempotencyKey: cmd.IdempotencyKey, Counterparty: cmd.Counterparty, Credential: string(credentialJSON),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exact endpoint")
	require.Equal(t, requestsBeforeQuery, attempts.Load(), "changed path must be rejected before replay HTTP")
}

func TestQueryOutcomeKeepsMismatchedFacilitatorIdentityUnknown(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payer   string
		network string
	}{
		{name: "different payer", payer: "0x9999999999999999999999999999999999999999", network: "eip155:8453"},
		{name: "different network", payer: "0x1111111111111111111111111111111111111111", network: "eip155:84532"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var endpoint string
			var attempts atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Header.Get(paymentSignatureHeader) == "" {
					writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
					writer.WriteHeader(http.StatusPaymentRequired)
					return
				}
				if attempts.Add(1) == 1 {
					writer.WriteHeader(http.StatusInternalServerError)
					return
				}
				receipt, err := json.Marshal(map[string]any{"success": true, "payer": tc.payer, "transaction": "0xabc123", "network": tc.network})
				require.NoError(t, err)
				writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
				writer.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			endpoint = server.URL + "/priced"

			adapter, err := New(server.Client(), &fixtureSigner{})
			require.NoError(t, err)
			cmd := command(t, endpoint, 1)
			first, err := adapter.Settle(context.Background(), cmd)
			require.Error(t, err)
			require.Equal(t, rail.OutcomeUnknown, first.Outcome)

			credential, err := json.Marshal(Credential{
				EndpointURL: endpoint, SignerURL: "http://127.0.0.1:8545",
				Account: "0x1111111111111111111111111111111111111111", Networks: []string{"eip155:8453"},
				Assets: map[string]AssetPolicy{"0x2222222222222222222222222222222222222222": {Decimals: 6, Currency: "USD"}},
			})
			require.NoError(t, err)
			reconciled, err := adapter.QueryOutcome(context.Background(), rail.Query{
				ExternalID: first.ExternalID, ReceiptReference: first.ReceiptReference,
				IdempotencyKey: cmd.IdempotencyKey, Counterparty: cmd.Counterparty, Credential: string(credential),
			})
			require.NoError(t, err)
			require.Equal(t, rail.OutcomeUnknown, reconciled.Outcome)
			require.Equal(t, "x402_replay_inconclusive", reconciled.Basis)
		})
	}
}
