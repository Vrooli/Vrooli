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

	"github.com/stretchr/testify/require"
	"treasury/internal/rail"
)

// A legacy upload challenge cannot authorize a v2 payment. This public body is
// the shape observed at the unsigned provider route; no provider is contacted.
func TestLegacyUploadChallengeCannotReachSignerOrPaidRetry(t *testing.T) {
	const legacy = `{"x402Version":1,"accepts":[{"scheme":"exact","network":"base","maxAmountRequired":"20481","resource":"https://upload.ardrive.io/v1/tx","payTo":"0x6A0A10FFD285c971B841bee8892878c0d583Bf67","maxTimeoutSeconds":3600,"asset":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","extra":{"name":"USD Coin","version":"2"}}]}`
	for _, header := range []bool{false, true} {
		t.Run(map[bool]string{false: "body_only", true: "legacy_header"}[header], func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				require.Empty(t, r.Header.Get(paymentSignatureHeader))
				if header {
					w.Header().Set(paymentRequiredHeader, base64.StdEncoding.EncodeToString([]byte(legacy)))
				}
				w.WriteHeader(http.StatusPaymentRequired)
				_, _ = w.Write([]byte(legacy))
			}))
			defer server.Close()
			signer := &fixtureSigner{}
			adapter, err := New(server.Client(), signer)
			require.NoError(t, err)
			result, err := adapter.Settle(context.Background(), command(t, server.URL, 3))
			require.NoError(t, err)
			require.Equal(t, rail.OutcomeFailed, result.Outcome)
			require.Equal(t, "x402_price_invalid", result.Basis)
			if header {
				require.Contains(t, result.Detail, "x402 v2")
			} else {
				require.Contains(t, result.Detail, "Payment-Required header is missing")
			}
			require.Zero(t, signer.calls.Load())
			require.EqualValues(t, 1, requests.Load())
		})
	}
}

func TestFractionalCentQuoteCannotBeRoundedIntoAnAuthorization(t *testing.T) {
	for _, cents := range []int64{2, 3} {
		t.Run(map[int64]string{2: "two_cents", 3: "three_cents"}[cents], func(t *testing.T) {
			var endpoint string
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				require.Empty(t, r.Header.Get(paymentSignatureHeader))
				w.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "20481"))
				w.WriteHeader(http.StatusPaymentRequired)
			}))
			defer server.Close()
			endpoint = server.URL
			signer := &fixtureSigner{}
			adapter, err := New(server.Client(), signer)
			require.NoError(t, err)
			result, err := adapter.Settle(context.Background(), command(t, endpoint, cents))
			require.NoError(t, err)
			require.Equal(t, rail.OutcomeFailed, result.Outcome)
			require.Contains(t, result.Detail, "20481 does not equal")
			require.Zero(t, signer.calls.Load())
			require.EqualValues(t, 1, requests.Load())
		})
	}
	// Positive selection control only: no signature or HTTP request.
	endpoint := "https://example.invalid/priced"
	decoded, err := base64.StdEncoding.DecodeString(paymentRequiredValue(t, endpoint, "10000"))
	require.NoError(t, err)
	var required paymentRequired
	require.NoError(t, json.Unmarshal(decoded, &required))
	var credential Credential
	require.NoError(t, json.Unmarshal([]byte(command(t, endpoint, 1).Credential), &credential))
	_, selected, decimals, err := selectRequirement(required, credential, endpoint, 1, "USD")
	require.NoError(t, err)
	require.Equal(t, "10000", selected.Amount)
	require.Equal(t, 6, decimals)
	require.True(t, strings.HasPrefix(selected.Network, "eip155:"))
}
