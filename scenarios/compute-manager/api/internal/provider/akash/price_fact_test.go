package akash

import (
	"strings"
	"testing"
	"time"
)

func sourcedPrice(now time.Time) PriceFact {
	return PriceFact{
		Network: "sandbox-2", ChainID: "sandbox-2-fixture", DSeq: "42", GSeq: 1, OSeq: 1, Provider: "akash1fixture", Region: "sandbox-2", OfferID: "42/1/1/akash1fixture",
		Amount: "1000000", Denom: "uakt", Period: "block", SourceURL: "https://api.fixture.invalid/bids", SourceVersion: "fixture-height-100",
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute), BlockSeconds: 6, BlockSourceURL: "https://docs.akash.network/interval", BlockSourceVersion: "fixture-v1", BlockObservedAt: now.Add(-time.Minute), BlockExpiresAt: now.Add(time.Minute),
		BaseUnitsPerToken: "1000000", DenomSourceURL: "https://docs.akash.network/denomination", DenomSourceVersion: "fixture-v1", DenomObservedAt: now.Add(-time.Minute), DenomExpiresAt: now.Add(time.Minute),
		USDPerToken: "1", FXSourceURL: "https://fx.fixture.invalid/akt-usd", FXSourceVersion: "fixture-rate-1", FXObservedAt: now.Add(-time.Minute), FXExpiresAt: now.Add(time.Minute),
	}
}

func TestPriceFactNormalizationRequiresSourcedInputsAndBindsIdentity(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	f := sourcedPrice(now)
	got, err := f.NormalizeMonthlyUSD(now, 30*24*60*60)
	if err != nil {
		t.Fatal(err)
	}
	if got.Minor != 43_200_000 || got.Fingerprint == "" {
		t.Fatalf("normalized=%+v", got)
	}
	changed := f
	changed.Amount = "1000001"
	next, err := changed.NormalizeMonthlyUSD(now, 30*24*60*60)
	if err != nil {
		t.Fatal(err)
	}
	if next.Fingerprint == got.Fingerprint {
		t.Fatal("changed bid amount retained replay identity")
	}
}

func TestNormalizedCatalogueFactCarriesReplayableProvenance(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	fact, err := sourcedPrice(now).NormalizeCatalogueFact(now, 30*24*60*60)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCatalogueFact(fact, now); err != nil {
		t.Fatalf("valid fixture provenance refused: %v", err)
	}
	intent := map[string]any{"akash_lease": map[string]any{"dseq": "42", "gseq": "1", "oseq": "1", "provider": "akash1fixture", "rate_amount": "1000000", "rate_denom": "uakt"}}
	if err := ValidateIntentBinding(fact.ProvenanceJSON, intent); err != nil {
		t.Fatalf("matching selected bid refused: %v", err)
	}
	intent["akash_lease"].(map[string]any)["rate_amount"] = "1000001"
	if err := ValidateIntentBinding(fact.ProvenanceJSON, intent); err == nil {
		t.Fatal("provenance for another bid matched intent")
	}
	changed := fact
	changed.ServerMinor++
	if err := ValidateCatalogueFact(changed, now); err == nil {
		t.Fatal("changed normalized price passed provenance validation")
	}
	if err := ValidateCatalogueFact(fact, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expired source provenance passed validation")
	}
}

func TestPriceFactRefusesAbsentOrStaleConversionProvenance(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		edit func(*PriceFact)
		now  time.Time
		want string
	}{
		{"no FX", func(f *PriceFact) { f.USDPerToken = ""; f.FXSourceURL = "" }, now, "USD exchange rate"},
		{"no block interval", func(f *PriceFact) { f.BlockSeconds = 0; f.BlockSourceURL = "" }, now, "block interval"},
		{"stale bid", func(f *PriceFact) { f.ExpiresAt = now.Add(-time.Second) }, now, "bid observation/expiry"},
		{"invalid period", func(f *PriceFact) { f.Period = "hour" }, now, "rate period"},
		{"unknown denom without source", func(f *PriceFact) { f.Denom = "mystery"; f.DenomSourceURL = "" }, now, "denomination scale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := sourcedPrice(now)
			tc.edit(&f)
			_, err := f.NormalizeMonthlyUSD(tc.now, 30*24*60*60)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want %q", err, tc.want)
			}
		})
	}
}
