package chainforensics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const fixtureBitcoinAddress = "1BoatSLRHtKNngkdXEeobR76b53LETtpyT"

func TestMempoolAddressPagePaginatesAndCachesConfirmedTransactions(t *testing.T) {
	var calls atomic.Int32
	firstCursor := fmt.Sprintf("%064x", 25)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/api/address/" + fixtureBitcoinAddress + "/txs/chain":
			rows := make([]string, 0, mempoolAddressPageLimit)
			for i := 1; i <= mempoolAddressPageLimit; i++ {
				rows = append(rows, fmt.Sprintf(`{"txid":"%064x","status":{"confirmed":true,"block_height":42,"block_time":1700000000},"vin":[],"vout":[]}`, i))
			}
			_, _ = fmt.Fprintf(w, "[%s]", strings.Join(rows, ","))
		case "/api/address/" + fixtureBitcoinAddress + "/txs/chain/" + firstCursor:
			_, _ = fmt.Fprintf(w, `[{"txid":"%064x","status":{"confirmed":true,"block_height":41,"block_time":1699999999},"vin":[],"vout":[]}]`, 26)
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	client, err := NewMempoolClient(server.URL+"/api", server.Client(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	client.minInterval = 0
	first, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Source.SHA256 == "" || first.Source.ID == "" || len(first.Transactions) != 25 || first.NextAfterTxID != firstCursor {
		t.Fatalf("first page lacks bounded capture provenance/cursor: %#v", first)
	}
	first.Transactions[0].TxID = "caller-mutated"
	cached, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, "")
	if err != nil {
		t.Fatal(err)
	}
	if cached.Transactions[0].TxID != fmt.Sprintf("%064x", 1) || calls.Load() != 1 {
		t.Fatalf("cached page was not isolated or reused: txid=%q requests=%d", cached.Transactions[0].TxID, calls.Load())
	}
	second, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, first.NextAfterTxID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Transactions) != 1 || second.Transactions[0].TxID != fmt.Sprintf("%064x", 26) || second.NextAfterTxID != "" || calls.Load() != 2 {
		t.Fatalf("cursor page mismatch: %#v requests=%d", second, calls.Load())
	}
}

func TestMempoolAddressPageRejectsInvalidAddressCursorAndOversizedPage(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		rows := make([]string, 0, mempoolAddressPageLimit+1)
		for i := 1; i <= mempoolAddressPageLimit+1; i++ {
			rows = append(rows, fmt.Sprintf(`{"txid":"%064x","status":{"confirmed":true,"block_height":42,"block_time":1700000000},"vin":[],"vout":[]}`, i))
		}
		_, _ = fmt.Fprintf(w, "[%s]", strings.Join(rows, ","))
	}))
	defer server.Close()
	client, err := NewMempoolClient(server.URL, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	client.minInterval = 0
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress+"/../../tx", ""); err == nil {
		t.Fatal("accepted path-like address")
	}
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, "not-a-txid"); err == nil {
		t.Fatal("accepted invalid pagination cursor")
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid input reached provider: %d requests", calls.Load())
	}
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, ""); err == nil {
		t.Fatal("accepted provider page larger than the confirmed-history page contract")
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one bounded-page request, got %d", calls.Load())
	}
}

func TestMempoolAddressPageAllowsLargeBoundedConsolidationAndRejectsOverLimit(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()
	client, err := NewMempoolClient(server.URL, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	client.minInterval = 0
	base := fmt.Sprintf(`[{"txid":"%064x","status":{"confirmed":true,"block_height":42,"block_time":1700000000},"vin":[],"vout":[]`, 1)
	body = base + `,"padding":"` + strings.Repeat("x", 2300*1024) + `"}]`
	page, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, "")
	if err != nil {
		t.Fatalf("expected bounded >1 MiB address page to decode: %v", err)
	}
	if len(page.Source.ResultBody) <= 2<<20 || len(page.Source.ResultBody) > mempoolAddressPageMaxBytes {
		t.Fatalf("retained address source body has unexpected size %d", len(page.Source.ResultBody))
	}

	// Use a fresh client so the successful page cache cannot hide the over-limit response.
	client, err = NewMempoolClient(server.URL, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	client.minInterval = 0
	body = base + `,"padding":"` + strings.Repeat("x", mempoolAddressPageMaxBytes) + `"}]`
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, ""); err == nil {
		t.Fatal("accepted address page over the configured response-byte limit")
	}
}

func TestMempoolAddressPageCacheExpiresAndStaysBounded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = fmt.Fprintf(w, `[{"txid":"%064x","status":{"confirmed":true,"block_height":42,"block_time":1700000000},"vin":[],"vout":[]}]`, 1)
	}))
	defer server.Close()
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	client, err := NewMempoolClient(server.URL, server.Client(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	client.minInterval = 0
	firstCursor := fmt.Sprintf("%064x", 1)
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, firstCursor); err != nil {
		t.Fatal(err)
	}
	now = now.Add(mempoolCacheTTL + time.Second)
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, firstCursor); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expired address page should be fetched again; requests=%d", calls.Load())
	}
	for i := 100; i < 100+mempoolAddressCacheLimit+1; i++ {
		if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, fmt.Sprintf("%064x", i)); err != nil {
			t.Fatal(err)
		}
	}
	if len(client.addressPageCache) != mempoolAddressCacheLimit {
		t.Fatalf("address page cache retained %d entries, want bound %d", len(client.addressPageCache), mempoolAddressCacheLimit)
	}
	before := calls.Load()
	if _, err := client.FetchAddressPage(context.Background(), fixtureBitcoinAddress, firstCursor); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != before+1 {
		t.Fatalf("oldest address page should be evicted; requests=%d before=%d", calls.Load(), before)
	}
}
