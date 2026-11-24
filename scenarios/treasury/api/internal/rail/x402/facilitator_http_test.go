package x402

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPFacilitatorEnforcesTransportAndURLBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		url     string
		client  *http.Client
		wantErr bool
	}{
		{name: "https remote", url: "https://facilitator.example", client: &http.Client{}},
		{name: "http loopback", url: "http://127.0.0.1:14021", client: &http.Client{}},
		{name: "remote http", url: "http://facilitator.example", client: &http.Client{}, wantErr: true},
		{name: "credentials", url: "https://user:secret@facilitator.example", client: &http.Client{}, wantErr: true},
		{name: "query", url: "https://facilitator.example?token=secret", client: &http.Client{}, wantErr: true},
		{name: "fragment", url: "https://facilitator.example#verify", client: &http.Client{}, wantErr: true},
		{name: "missing client", url: "https://facilitator.example", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facilitator, err := NewHTTPFacilitator(tc.url, tc.client)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, facilitator)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, facilitator)
			require.Equal(t, tc.name == "http loopback", facilitator.LocalOnly())
		})
	}
}

func TestHTTPFacilitatorSendsVersionTwoRequestAndParsesResponses(t *testing.T) {
	type requestRecord struct {
		path string
		body map[string]json.RawMessage
	}
	requests := make(chan requestRecord, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		requests <- requestRecord{path: r.URL.Path, body: body}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/verify") {
			_, _ = w.Write([]byte(`{"isValid":true,"payer":"0xabc"}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"payer":"0xabc","transaction":"0xtx","network":"eip155:8453"}`))
	}))
	defer server.Close()

	facilitator, err := NewHTTPFacilitator(server.URL+"/payments/", server.Client())
	require.NoError(t, err)
	payload := json.RawMessage(`{"signature":"fixture-signature"}`)
	requirements := json.RawMessage(`{"scheme":"exact","amount":"10000"}`)

	verified, err := facilitator.Verify(context.Background(), payload, requirements)
	require.NoError(t, err)
	require.Equal(t, VerifyResult{Valid: true, Payer: "0xabc"}, verified)
	verifyRequest := <-requests
	require.Equal(t, "/payments/verify", verifyRequest.path)
	require.JSONEq(t, `{"signature":"fixture-signature"}`, string(verifyRequest.body["paymentPayload"]))
	require.JSONEq(t, `{"scheme":"exact","amount":"10000"}`, string(verifyRequest.body["paymentRequirements"]))
	require.Equal(t, json.RawMessage("2"), verifyRequest.body["x402Version"])

	settled, err := facilitator.Settle(context.Background(), payload, requirements)
	require.NoError(t, err)
	require.Equal(t, SettleResult{Success: true, Payer: "0xabc", Transaction: "0xtx", Network: "eip155:8453"}, settled)
	settleRequest := <-requests
	require.Equal(t, "/payments/settle", settleRequest.path)
	require.JSONEq(t, `{"signature":"fixture-signature"}`, string(settleRequest.body["paymentPayload"]))
	require.JSONEq(t, `{"scheme":"exact","amount":"10000"}`, string(settleRequest.body["paymentRequirements"]))
	require.Equal(t, json.RawMessage("2"), settleRequest.body["x402Version"])
}

func TestHTTPFacilitatorDoesNotForwardPaymentPayloadAcrossRedirect(t *testing.T) {
	redirectedRequests := make(chan string, 1)
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		redirectedRequests <- string(data)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"isValid":true,"payer":"0xredirected"}`))
	}))
	defer redirectTarget.Close()
	var primaryRequests atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryRequests.Add(1)
		http.Redirect(w, r, redirectTarget.URL+"/collect", http.StatusTemporaryRedirect)
	}))
	defer primary.Close()

	facilitator, err := NewHTTPFacilitator(primary.URL, &http.Client{})
	require.NoError(t, err)
	const privatePaymentPayload = `{"authorization":"fixture-private-payment"}`
	_, err = facilitator.Verify(context.Background(), json.RawMessage(privatePaymentPayload), json.RawMessage(`{"amount":"10000"}`))
	require.EqualValues(t, 1, primaryRequests.Load())
	select {
	case body := <-redirectedRequests:
		require.Empty(t, body, "a redirect must not disclose the signed payment request to another origin")
		t.Fatal("HTTP facilitator followed a redirect")
	default:
	}
	require.ErrorContains(t, err, "HTTP 307")
}

func TestHTTPFacilitatorRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat("x", (1<<20)+1))
	}))
	defer server.Close()

	facilitator, err := NewHTTPFacilitator(server.URL, server.Client())
	require.NoError(t, err)
	_, err = facilitator.Verify(context.Background(), json.RawMessage(`{}`), json.RawMessage(`{}`))
	require.ErrorContains(t, err, "response exceeds limit")
}
