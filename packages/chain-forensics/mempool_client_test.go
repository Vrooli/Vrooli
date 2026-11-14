package chainforensics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const fixtureTxID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestMempoolClientCapturesAndCachesConfirmedTransaction(t *testing.T) {
	const body = `{"txid":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","status":{"confirmed":true,"block_height":42,"block_time":1700000000},"vin":[{"txid":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","vout":1,"prevout":{"scriptpubkey_address":"source","value":123456789}}],"vout":[{"scriptpubkey_address":"destination","value":123456000}]}`
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/api/tx/"+fixtureTxID {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("missing JSON Accept header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()

	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	client, err := NewMempoolClient(server.URL+"/api", server.Client(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.FetchTransaction(context.Background(), fixtureTxID)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256([]byte(body))
	if first.Source.SHA256 != hex.EncodeToString(wantHash[:]) || first.Source.URL != server.URL+"/api/tx/"+fixtureTxID {
		t.Fatalf("capture provenance mismatch: %#v", first.Source)
	}
	first.Transaction.Outputs[0].Address = "caller-mutated"
	second, err := client.FetchTransaction(context.Background(), fixtureTxID)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("immutable txid cache made %d HTTP requests, want 1", calls.Load())
	}
	if second.Transaction.Outputs[0].Address != "destination" {
		t.Fatalf("caller mutation escaped into cache: %q", second.Transaction.Outputs[0].Address)
	}
	if second.Source.Retrieved != now.Format(time.RFC3339) {
		t.Fatalf("unexpected capture time: %s", second.Source.Retrieved)
	}
}

func TestMempoolClientSurfacesRetryAfterWithoutRetrying(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewMempoolClient(server.URL, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.FetchTransaction(context.Background(), fixtureTxID)
	if !errors.Is(err, ErrProviderRateLimited) {
		t.Fatalf("expected typed rate-limit error, got %v", err)
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.RetryAfter != 5*time.Second {
		t.Fatalf("Retry-After was not preserved: %#v", providerErr)
	}
	if calls.Load() != 1 {
		t.Fatalf("rate-limit response should not be retried automatically: requests=%d", calls.Load())
	}
}

func TestMempoolClientCacheExpiresAndStaysBounded(t *testing.T) {
	var now = time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		txid := strings.TrimPrefix(r.URL.Path, "/api/tx/")
		_, _ = fmt.Fprintf(w, `{"txid":%q,"status":{"confirmed":true,"block_height":42,"block_time":1700000000},"vin":[],"vout":[]}`, txid)
	}))
	defer server.Close()
	client, err := NewMempoolClient(server.URL+"/api", server.Client(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	client.minInterval = 0
	firstID := fmt.Sprintf("%064x", 1)
	if _, err := client.FetchTransaction(context.Background(), firstID); err != nil {
		t.Fatal(err)
	}
	now = now.Add(mempoolCacheTTL + time.Second)
	if _, err := client.FetchTransaction(context.Background(), firstID); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expired entry should be fetched again; requests=%d", calls.Load())
	}

	for i := 2; i < 2+mempoolCacheLimit; i++ {
		if _, err := client.FetchTransaction(context.Background(), fmt.Sprintf("%064x", i)); err != nil {
			t.Fatal(err)
		}
	}
	if len(client.cache) != mempoolCacheLimit {
		t.Fatalf("cache retained %d entries, want bound %d", len(client.cache), mempoolCacheLimit)
	}
	if _, err := client.FetchTransaction(context.Background(), firstID); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != int32(3+mempoolCacheLimit) {
		t.Fatalf("oldest entry should be evicted and fetched again; requests=%d", calls.Load())
	}
}

func TestNewMempoolClientRequiresHTTPSOutsideLoopback(t *testing.T) {
	if _, err := NewMempoolClient("http://example.test/api", nil, nil); err == nil {
		t.Fatal("accepted non-HTTPS public endpoint")
	}
	if _, err := NewMempoolClient("https://user:pass@mempool.space/api", nil, nil); err == nil {
		t.Fatal("accepted endpoint credentials")
	}
}
