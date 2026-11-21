package commerce

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestOfferDeskServiceOrderSourceUsesCanonicalRecordTerms(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != offerDeskServiceOfferPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"offer":{"id":"company-market-scan","version":1,"proposed_price":{"amount_minor":4900,"currency":"USD"},"x402_price_id":"company-market-scan-v1"}}`)
	}))
	defer server.Close()

	source, err := NewOfferDeskServiceOrderSource(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOfferDeskServiceOrderSource() error = %v", err)
	}
	quote, err := source.ResolveServiceOrderQuote(context.Background(), "company-market-scan", 1)
	if err != nil {
		t.Fatalf("ResolveServiceOrderQuote() error = %v", err)
	}
	want := ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD", X402PriceID: "company-market-scan-v1"}
	if quote != want {
		t.Fatalf("quote = %+v, want %+v", quote, want)
	}
}

func TestOfferDeskServiceOrderSourceRejectsIdentityAndTermsDrift(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"offer":{"id":"company-market-scan","version":2,"proposed_price":{"amount_minor":1,"currency":"USD"},"x402_price_id":"other-price"}}`)
	}))
	defer server.Close()
	source, err := NewOfferDeskServiceOrderSource(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOfferDeskServiceOrderSource() error = %v", err)
	}
	if _, err := source.ResolveServiceOrderQuote(context.Background(), "company-market-scan", 1); err == nil {
		t.Fatal("changed Offer Desk version was accepted")
	}
	if _, err := source.ResolveServiceOrderQuote(context.Background(), "other-offer", 2); err == nil {
		t.Fatal("different Offer Desk record was accepted")
	}
}

func TestOfferDeskServiceOrderSourceRejectsNonLocalAndInvalidPrices(t *testing.T) {
	if _, err := NewOfferDeskServiceOrderSource("https://offers.example", http.DefaultClient); err == nil {
		t.Fatal("non-loopback offer source was accepted for test-mode order composition")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"offer":{"id":"company-market-scan","version":1,"proposed_price":{"amount_minor":0,"currency":"EUR"},"x402_price_id":""}}`)
	}))
	defer server.Close()
	source, err := NewOfferDeskServiceOrderSource(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOfferDeskServiceOrderSource() error = %v", err)
	}
	if _, err := source.ResolveServiceOrderQuote(context.Background(), "company-market-scan", 1); err == nil {
		t.Fatal("invalid price was accepted")
	}
}

func TestOfferDeskServiceOrderSourceDoesNotFollowRedirects(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()
	source, err := NewOfferDeskServiceOrderSource(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOfferDeskServiceOrderSource() error = %v", err)
	}
	if _, err := source.ResolveServiceOrderQuote(context.Background(), "company-market-scan", 1); err == nil {
		t.Fatal("redirect response was accepted as an offer record")
	}
	if redirected {
		t.Fatal("Offer Desk response redirected the quote client")
	}
}

func TestOfferDeskServiceOrderSourceReadsRunningPreview(t *testing.T) {
	baseURL := os.Getenv("LPBS_TEST_OFFER_DESK_URL")
	if baseURL == "" {
		t.Skip("set LPBS_TEST_OFFER_DESK_URL to exercise an already-running local Offer Desk preview")
	}
	source, err := NewOfferDeskServiceOrderSource(baseURL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewOfferDeskServiceOrderSource() error = %v", err)
	}
	quote, err := source.ResolveServiceOrderQuote(context.Background(), "company-market-scan", 1)
	if err != nil {
		t.Fatalf("ResolveServiceOrderQuote() error = %v", err)
	}
	want := ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD", X402PriceID: "company-market-scan-v1"}
	if quote != want {
		t.Fatalf("running Offer Desk quote = %+v, want %+v", quote, want)
	}
}
