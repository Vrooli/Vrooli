package setup

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"compute-manager/internal/provider"
	_ "modernc.org/sqlite"
)

func TestDecimalMinorRequiresExactMinorUnits(t *testing.T) {
	for in, want := range map[string]int64{"1": 100, "0.07": 7, "12.34": 1234} {
		got, err := decimalMinor(in)
		if err != nil || got != want {
			t.Fatalf("decimalMinor(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "1.001", "not-a-price"} {
		if _, err := decimalMinor(in); err == nil {
			t.Fatalf("decimalMinor(%q) unexpectedly succeeded", in)
		}
	}
}

func TestAkashOnChainRateIsNotFabricatedAsMonthlyCataloguePrice(t *testing.T) {
	fact, err := FetchPriceFact(context.Background(), nil, "", "akash", "sandbox-2", "lease", false, time.Now().UTC())
	if err == nil || fact.Provider != "" {
		t.Fatalf("Akash missing bid source returned a catalogue fact: fact=%+v err=%v", fact, err)
	}
	if !strings.Contains(err.Error(), "no configured read-only source for the exact selected sandbox bid") || !strings.Contains(err.Error(), "requires an API key and an existing deployment sequence (dseq)") || !strings.Contains(err.Error(), "sourced denomination scale, block interval, explicit admission-month duration") {
		t.Fatalf("unexpected refusal: %v", err)
	}
}

func TestCatalogProvidesDeclaredZeroCostLocalQEMUFact(t *testing.T) {
	now := time.Date(2026, 11, 23, 23, 0, 0, 0, time.UTC)
	db, err := sql.Open("sqlite", "file:setup-local-qemu-catalog?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	catalog := Catalog{Repo: repo, Now: func() time.Time { return now }}
	fact, err := catalog.RecordLocalQEMUOffer(context.Background(), "headless-linux")
	if err != nil {
		t.Fatal(err)
	}
	lookedUp, err := catalog.LookupPrice(context.Background(), "local-qemu", "local", "headless-linux", true)
	if err != nil || lookedUp != fact {
		t.Fatalf("LookupPrice() = %#v, %v; want persisted declaration %#v", lookedUp, err, fact)
	}
	if fact.Provider != "local-qemu" || fact.OfferID != "host-local-qemu:headless-linux" || fact.Region != "local" || fact.Size != "headless-linux" || fact.ServerMinor != 0 || fact.IPv4Minor != 0 || fact.Currency != "USD" || fact.TaxBasis != "pre-tax" || fact.SourceURL != localQEMUPriceSource || fact.SourceVersion != localQEMUPriceVersion || !fact.ObservedAt.Equal(now) || !fact.ExpiresAt.Equal(now.Add(30*24*time.Hour)) {
		t.Fatalf("local quote = %#v", fact)
	}
	stored, err := repo.Prices(context.Background(), "local-qemu")
	if err != nil || len(stored) != 1 || stored[0] != fact {
		t.Fatalf("persisted local facts = %#v, err=%v", stored, err)
	}
	for _, tc := range []struct{ region, profile string }{
		{region: "nbg1", profile: "headless-linux"},
		{region: "local", profile: "unsupported"},
		{region: "local", profile: "Headless-Linux"},
	} {
		if _, err := catalog.LookupPrice(context.Background(), "local-qemu", tc.region, tc.profile, true); err == nil {
			t.Fatalf("local-qemu accepted mismatched region/profile %q/%q", tc.region, tc.profile)
		}
	}
	if _, err := (Catalog{Now: func() time.Time { return now }}).LookupPrice(context.Background(), "local-qemu", "local", "headless-linux", true); err == nil {
		t.Fatal("local-qemu accepted a missing repository")
	}
	if _, err := catalog.LookupPrice(context.Background(), "local-qemu", "local", "desktop-linux", true); err == nil {
		t.Fatal("catalogue synthesized an unrecorded desktop profile")
	}
	if _, err := catalog.RecordLocalQEMUOffer(context.Background(), "desktop-linux"); err != nil {
		t.Fatalf("catalogue could not record the supported second profile: %v", err)
	}
	if _, err := catalog.LookupPrice(context.Background(), "local-qemu", "local", "desktop-linux", true); err != nil {
		t.Fatalf("catalogue could not return the explicitly recorded second profile: %v", err)
	}
	if _, err := catalog.LookupPrice(context.Background(), "local-qemu", "local", "headless-linux", false); err != nil {
		t.Fatalf("local quote without requested IPv4 failed despite explicit zero component: %v", err)
	}
}

func TestCatalogRejectsStaleOrCorruptedLocalQEMUFact(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*provider.PriceFact)
	}{
		{name: "stale", mutate: func(f *provider.PriceFact) { f.ExpiresAt = f.ObservedAt }},
		{name: "wrong offer profile", mutate: func(f *provider.PriceFact) { f.OfferID = "host-local-qemu:desktop-linux" }},
		{name: "nonzero provider price", mutate: func(f *provider.PriceFact) { f.ServerMinor = 1 }},
		{name: "nonzero ipv4", mutate: func(f *provider.PriceFact) { f.IPv4Minor = 1 }},
		{name: "wrong source", mutate: func(f *provider.PriceFact) { f.SourceURL = "https://fixture.invalid/local-qemu" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", "file:setup-local-qemu-invalid-"+strings.ReplaceAll(tc.name, " ", "-")+"?mode=memory&cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(Schema()); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 11, 23, 23, 0, 0, 0, time.UTC)
			fact := provider.PriceFact{Provider: "local-qemu", OfferID: "host-local-qemu:headless-linux", Region: "local", Size: "headless-linux", Currency: "USD", TaxBasis: "pre-tax", SourceURL: localQEMUPriceSource, SourceVersion: localQEMUPriceVersion, ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
			tc.mutate(&fact)
			repo := NewRepository(db)
			if err := repo.PutPrice(context.Background(), fact); err != nil {
				t.Fatal(err)
			}
			catalog := Catalog{Repo: repo, Now: func() time.Time { return now }}
			if _, err := catalog.LookupPrice(context.Background(), "local-qemu", "local", "headless-linux", true); err == nil {
				t.Fatal("catalogue returned stale or corrupted local-QEMU fact")
			}
		})
	}
}

func TestCatalogReturnsOnlyFreshExactObservation(t *testing.T) {
	db, err := sql.Open("sqlite", "file:setup-catalog-prices?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 15, 0, 0, 0, time.UTC)
	repo := NewRepository(db)
	fact := provider.PriceFact{Provider: "hetzner", OfferID: "cx23", Region: "nbg1", Size: "cx23", Currency: "USD", ServerMinor: 500, IPv4Minor: 200, TaxBasis: "pre-tax", SourceURL: "https://api.hetzner.cloud/v1/pricing", SourceVersion: "sha256:fixture", ObservedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := repo.PutPrice(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	catalog := Catalog{Repo: repo, Now: func() time.Time { return now }}
	got, err := catalog.LookupPrice(context.Background(), "hetzner", "nbg1", "cx23", true)
	if err != nil || got.OfferID != fact.OfferID || got.ServerMinor != 500 || got.IPv4Minor != 200 {
		t.Fatalf("LookupPrice() = %#v, %v", got, err)
	}
	if _, err := catalog.LookupPrice(context.Background(), "hetzner", "fsn1", "cx23", true); err == nil {
		t.Fatal("lookup returned a different region")
	}
	catalog.Now = func() time.Time { return now.Add(time.Minute) }
	if _, err := catalog.LookupPrice(context.Background(), "hetzner", "nbg1", "cx23", true); err == nil {
		t.Fatal("lookup returned an expired fact")
	}
}

func TestCatalogRecordsAdoptedVPSOnlyAsExplicitFreshDeclaredFact(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:setup-adopted-vps-catalog?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 23, 0, 0, 0, time.UTC)
	catalog := Catalog{Repo: NewRepository(db), Now: func() time.Time { return now }}
	fact, err := catalog.RecordAdoptedVPSOffer(ctx, "request-alpha", 709, "USD", "pre-tax")
	if err != nil {
		t.Fatal(err)
	}
	if fact.Provider != "adopted-vps" || fact.OfferID == "" || fact.Region != "manual" || !strings.HasPrefix(fact.Size, "declared-") || fact.Currency != "USD" || fact.ServerMinor != 709 || fact.IPv4Minor != 0 || fact.TaxBasis != "pre-tax" || !strings.HasPrefix(fact.SourceURL, "urn:vrooli:compute-manager:operator-declared/adopted-vps/") || fact.SourceVersion != adoptedVPSOfferVersion || !fact.ObservedAt.Equal(now) || !fact.ExpiresAt.Equal(now.Add(adoptedVPSOfferValidity)) {
		t.Fatalf("declared adopted-vps fact = %#v", fact)
	}
	lookedUp, err := catalog.LookupPrice(ctx, "adopted-vps", "manual", fact.Size, true)
	if err != nil || lookedUp != fact {
		t.Fatalf("LookupPrice() = %#v, %v; want exact stored declaration %#v", lookedUp, err, fact)
	}
	if _, err := catalog.RecordAdoptedVPSOffer(ctx, "request-alpha", 710, "USD", "pre-tax"); err == nil {
		t.Fatal("reused idempotency identity accepted changed billing facts")
	}
	for _, tc := range []struct {
		name, key, currency, tax string
		monthly                  int64
	}{
		{name: "missing key", key: "", currency: "USD", tax: "pre-tax", monthly: 0},
		{name: "negative", key: "negative", currency: "USD", tax: "pre-tax", monthly: -1},
		{name: "bad currency", key: "bad-currency", currency: "US", tax: "pre-tax", monthly: 0},
		{name: "missing tax basis", key: "no-tax", currency: "USD", tax: "", monthly: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := catalog.RecordAdoptedVPSOffer(ctx, tc.key, tc.monthly, tc.currency, tc.tax); err == nil {
				t.Fatal("invalid declared billing fact was accepted")
			}
		})
	}
	if _, err := catalog.LookupPrice(ctx, "adopted-vps", "other", fact.Size, true); err == nil {
		t.Fatal("catalogue returned adopted fact for a mismatched region")
	}
}
