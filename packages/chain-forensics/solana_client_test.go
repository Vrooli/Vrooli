package chainforensics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSolanaAddressPageCapturesFinalizedSPLDeltasAndCaches(t *testing.T) {
	signature, address, owner, mint := strings.Repeat("1", 64), strings.Repeat("1", 32), "11111111111111111111111111111112", "11111111111111111111111111111113"
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case "getSignaturesForAddress":
			var cfg map[string]any
			_ = json.Unmarshal(request.Params[1], &cfg)
			if cfg["commitment"] != "finalized" || cfg["limit"] != float64(25) {
				t.Errorf("unexpected bounded finalized request: %#v", cfg)
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":[{"signature":%q,"slot":99,"err":null,"blockTime":1790000000}]}`, signature)
		case "getTransaction":
			var config map[string]any
			if err := json.Unmarshal(request.Params[1], &config); err != nil {
				t.Error(err)
			}
			if config["encoding"] != "jsonParsed" || config["commitment"] != "finalized" || config["maxSupportedTransactionVersion"] != float64(1) {
				t.Errorf("unexpected finalized transaction config: %#v", config)
			}
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":2,"result":{"slot":99,"blockTime":1790000000,"transaction":{"signatures":[%q],"message":{"accountKeys":[%q,%q]}},"meta":{"err":null,"fee":5000,"preTokenBalances":[{"accountIndex":1,"mint":%q,"owner":%q,"uiTokenAmount":{"amount":"10","decimals":6}}],"postTokenBalances":[{"accountIndex":1,"mint":%q,"owner":%q,"uiTokenAmount":{"amount":"4","decimals":6}}]}}}`, signature, address, owner, mint, owner, mint, owner)
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.FetchAddressPage(context.Background(), address, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Signatures) != 1 || page.NextBefore != signature || len(page.Transactions) != 1 || page.Source.SHA256 == "" || page.Source.RawResponseSHA256 == "" || len(page.Source.RawResponseBody) == 0 || page.Transactions[0].Source.RawResponseSHA256 == "" || len(page.Transactions[0].Source.RawResponseBody) == 0 {
		t.Fatalf("incomplete page: %+v", page)
	}
	captured := page.Transactions[0]
	if len(captured.Occurrences) != 1 {
		t.Fatalf("expected one SPL delta, got %d", len(captured.Occurrences))
	}
	occ := captured.Occurrences[0]
	if occ.Direction != "input" || occ.From != owner || occ.Asset != "SPL:"+mint || occ.Atomic != "6" || occ.Scale != 6 || occ.Block != 99 {
		t.Fatalf("unexpected SPL balance delta: %+v", occ)
	}
	if err := Validate(Case{ID: "solana-test", Title: "test", Private: true, Sources: []Source{page.Source, captured.Source}, Occurrences: captured.Occurrences}); err != nil {
		t.Fatalf("captured Solana page failed case validation: %v", err)
	}
	replay, err := client.FetchAddressPage(context.Background(), address, "")
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || replay.Transactions[0].Occurrences[0].Atomic != "6" {
		t.Fatalf("page cache did not return an isolated repeat: calls=%d page=%+v", requests.Load(), replay)
	}
	page.Transactions[0].Occurrences[0].Atomic = "999"
	replay, err = client.FetchAddressPage(context.Background(), address, "")
	if err != nil {
		t.Fatal(err)
	}
	if replay.Transactions[0].Occurrences[0].Atomic != "6" {
		t.Fatal("caller mutation corrupted the cached Solana result")
	}
}

func TestSolanaAddressAndSignatureValidation(t *testing.T) {
	if !validSolanaPubkey(strings.Repeat("1", 32)) || validSolanaPubkey(strings.Repeat("1", 31)) {
		t.Fatal("Solana public key length validation is incorrect")
	}
	if !validSolanaSignature(strings.Repeat("1", 64)) || validSolanaSignature(strings.Repeat("1", 63)) {
		t.Fatal("Solana signature length validation is incorrect")
	}
	if _, err := NewSolanaRPCClient("http://example.com", nil, time.Now); err == nil {
		t.Fatal("non-loopback HTTP RPC endpoint was accepted")
	}
}

func TestSolanaProviderRetryAfterIsReturnedWithoutAutomaticRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), func() time.Time { return time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.FetchAddressPage(context.Background(), strings.Repeat("1", 32), "")
	var providerErr *SolanaProviderError
	if !errors.Is(err, ErrSolanaProviderRateLimited) || !errors.As(err, &providerErr) || providerErr.RetryAfter != 7*time.Second || calls.Load() != 1 {
		t.Fatalf("rate limit was not preserved without retry: err=%v calls=%d", err, calls.Load())
	}
}
