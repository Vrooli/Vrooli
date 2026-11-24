package setup

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/provider"
	_ "modernc.org/sqlite"
)

func TestFetchDigitalOceanBundledPriceFactHasExactIPv4InclusiveMonthlySelection(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != digitalOceanSizesURL {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatal("Authorization header did not use the fixture credential")
		}
		body := `{"sizes":[{"available":true,"slug":"s-2vcpu-4gb","price_hourly":0.01042,"price_monthly":7.09,"regions":["nyc1","sfo3"]}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Etag": []string{"fixture-do-etag"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	fact, err := fetchDigitalOceanPrice(context.Background(), client, "fixture-token", "nyc1", "s-2vcpu-4gb", true, now)
	if err != nil {
		t.Fatal(err)
	}
	if fact.Provider != "digitalocean" || fact.OfferID != "s-2vcpu-4gb" || fact.Region != "nyc1" || fact.Size != "s-2vcpu-4gb" || fact.Currency != "USD" || fact.ServerMinor != 709 || fact.IPv4Minor != 0 || fact.TaxBasis != "pre-tax" || fact.SourceURL != digitalOceanSizesURL || !strings.Contains(fact.SourceVersion, "fixture-do-etag") || fact.ObservedAt != now || !fact.ExpiresAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("fact = %#v", fact)
	}
}

func TestDigitalOceanCatalogReturnsOnlyFreshExactSourcedObservation(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-do-exact-price?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	repo := NewRepository(db)
	catalog := Catalog{Repo: repo, Now: func() time.Time { return now }}
	if _, err := catalog.LookupPrice(context.Background(), "digitalocean", "nyc1", "s-2vcpu-4gb", true); err == nil {
		t.Fatal("unknown DigitalOcean price was synthesized")
	}
	fact := provider.PriceFact{Provider: "digitalocean", OfferID: "s-2vcpu-4gb", Region: "nyc1", Size: "s-2vcpu-4gb", Currency: "USD", ServerMinor: 709, TaxBasis: "pre-tax", SourceURL: digitalOceanSizesURL, SourceVersion: "FIXTURE ONLY: synthetic-etag", ObservedAt: now, ExpiresAt: now.Add(5 * time.Minute)}
	if err := repo.PutPrice(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	got, err := catalog.LookupPrice(context.Background(), "digitalocean", "nyc1", "s-2vcpu-4gb", true)
	if err != nil || got != fact {
		t.Fatalf("exact DigitalOcean fact = %#v, %v; want %#v", got, err, fact)
	}
	if _, err := catalog.LookupPrice(context.Background(), "digitalocean", "sfo3", "s-2vcpu-4gb", true); err == nil {
		t.Fatal("DigitalOcean price lookup returned a different region")
	}
	catalog.Now = func() time.Time { return now.Add(5 * time.Minute) }
	if _, err := catalog.LookupPrice(context.Background(), "digitalocean", "nyc1", "s-2vcpu-4gb", true); err == nil {
		t.Fatal("expired DigitalOcean price observation was admitted")
	}
}

func TestFetchDigitalOceanPriceFailsClosedForUnsupportedOrUncertainFacts(t *testing.T) {
	clientFor := func(body string) *http.Client {
		return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
		})}
	}
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, region, size, body string
	}{
		{name: "wrong region", region: "sfo3", size: "s-1vcpu-1gb", body: `{"sizes":[{"available":true,"slug":"s-1vcpu-1gb","price_monthly":5.00,"regions":["nyc1"]}]}`},
		{name: "unavailable", region: "nyc1", size: "s-1vcpu-1gb", body: `{"sizes":[{"available":false,"slug":"s-1vcpu-1gb","price_monthly":5.00,"regions":["nyc1"]}]}`},
		{name: "sub-cent price", region: "nyc1", size: "s-1vcpu-1gb", body: `{"sizes":[{"available":true,"slug":"s-1vcpu-1gb","price_monthly":5.001,"regions":["nyc1"]}]}`},
		{name: "paginated response", region: "nyc1", size: "s-1vcpu-1gb", body: `{"links":{"pages":{"next":"https://api.digitalocean.com/v2/sizes?page=2"}},"sizes":[]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fetchDigitalOceanPrice(context.Background(), clientFor(tc.body), "fixture-token", tc.region, tc.size, true, now)
			if err == nil {
				t.Fatal("uncertain or mismatched size data was admitted")
			}
		})
	}
	calls := 0
	v5Client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })}
	if _, err := fetchDigitalOceanPrice(context.Background(), v5Client, "fixture-token", "nyc1", "s5-8vcpu-16gb-30gb", true, now); err == nil || calls != 0 {
		t.Fatalf("v5 error=%v calls=%d; want fail-closed before fetching unsupported shape", err, calls)
	}
}
