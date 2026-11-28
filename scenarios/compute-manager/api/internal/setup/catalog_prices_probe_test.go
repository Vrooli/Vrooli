package setup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchHetznerPriceFactUsesReadOnlyIPv4InclusiveMonthlyComponents(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != "https://api.hetzner.cloud/v1/pricing" {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("authorization header missing fixture token")
		}
		body := `{"pricing":{"currency":"USD","vat_rate":"0.00","server_types":[{"name":"cx23","prices":[{"location":"nbg1","price_monthly":{"net":"6.49","gross":"7.78"}}]}],"primary_ips":[{"type":"ipv4","prices":[{"location":"nbg1","price_monthly":{"net":"0.60","gross":"0.72"}}]}]}}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Etag": []string{"fixture-etag"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	fact, err := FetchPriceFact(context.Background(), client, "fixture-token", "hetzner", "nbg1", "cx23", true, now)
	if err != nil {
		t.Fatal(err)
	}
	if fact.Provider != "hetzner" || fact.OfferID != "cx23" || fact.Region != "nbg1" || fact.Size != "cx23" || fact.Currency != "USD" || fact.ServerMinor != 649 || fact.IPv4Minor != 60 || fact.TaxBasis != "pre-tax" || fact.SourceVersion != "fixture-etag" || fact.ObservedAt != now || !fact.ExpiresAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("fact = %#v", fact)
	}
}
