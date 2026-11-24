package provision

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/provider/akash"
	internalstore "compute-manager/internal/store"
	_ "modernc.org/sqlite"
)

func TestAkashPriceProvenanceSurvivesQuoteReloadAndReceiptReadback(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "quote.db")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	db := open()
	if _, err := db.Exec(internalstore.Schema()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	rawFact := akash.PriceFact{Network: "sandbox-2", ChainID: "sandbox-2-fixture", DSeq: "42", GSeq: 1, OSeq: 1, Provider: "akash1fixture", Region: "sandbox-2", OfferID: "akash-lease", Amount: "1000000", Denom: "uakt", Period: "block", SourceURL: "https://fixture.invalid/bid", SourceVersion: "synthetic-fixture", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), BlockSeconds: 6, BlockSourceURL: "https://fixture.invalid/block", BlockSourceVersion: "synthetic-block", BlockObservedAt: now.Add(-time.Minute), BlockExpiresAt: now.Add(time.Hour), BaseUnitsPerToken: "1000000", DenomSourceURL: "https://fixture.invalid/denom", DenomSourceVersion: "synthetic-denom", DenomObservedAt: now.Add(-time.Minute), DenomExpiresAt: now.Add(time.Hour), USDPerToken: "1", FXSourceURL: "https://fixture.invalid/fx", FXSourceVersion: "synthetic-fx", FXObservedAt: now.Add(-time.Minute), FXExpiresAt: now.Add(time.Hour)}
	catalogueFact, err := rawFact.NormalizeCatalogueFact(now, 30*24*60*60)
	if err != nil {
		t.Fatal(err)
	}
	quote, err := priceSelection(context.Background(), fixturePrices{prices: map[string]PriceFact{"lease": catalogueFact}}, "akash", "sandbox-2", "lease", "akash-price-key", true, MoneyCap{Currency: "USD", MonthlyMinor: 100_000_000, TaxBasis: "pre-tax"}, now)
	if err != nil {
		t.Fatal(err)
	}
	quote.State = "approved"
	quote.CreatedAt = now
	provenance := quote.PriceProvenanceJSON
	inserted, err := persistQuote(context.Background(), db, &quote, MoneyCap{Currency: "USD", MonthlyMinor: 100_000_000, TaxBasis: "pre-tax"})
	if err != nil || !inserted {
		t.Fatalf("persist quote inserted=%t err=%v", inserted, err)
	}
	changed := quote
	changed.PriceProvenanceFingerprint = "changed-conversion-input"
	changed.PriceProvenanceJSON = `{"version":"akash-price-provenance/v1","fingerprint":"changed-conversion-input"}`
	if inserted, err := persistQuote(context.Background(), db, &changed, MoneyCap{Currency: "USD", MonthlyMinor: 100_000_000, TaxBasis: "pre-tax"}); err != nil || inserted {
		t.Fatalf("same-key changed provenance inserted=%t err=%v; want immutable existing quote", inserted, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal(err)
	}
	db = open()
	defer db.Close()
	receipt, err := (Service{DB: db}).GetSpendReceipt(context.Background(), quote.IdempotencyKey)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.PriceProvenanceJSON != provenance || receipt.PriceProvenanceFingerprint != quote.PriceProvenanceFingerprint || receipt.MonthlyMinor != 43_200_000 || receipt.Currency != "USD" {
		t.Fatalf("reloaded quote lost price provenance or normalized amount: %+v", receipt)
	}
	intentRecord := intent.Record{IdempotencyKey: quote.IdempotencyKey, Provider: "akash", Spec: map[string]any{"region": "sandbox-2", "size": "lease", "ipv4": true, "akash_lease": map[string]any{"dseq": "42", "gseq": "1", "oseq": "1", "provider": "akash1fixture", "rate_amount": "1000000", "rate_denom": "uakt"}}}
	policy := MoneyCap{Currency: "USD", MonthlyMinor: 100_000_000, TaxBasis: "pre-tax"}
	replayService := Service{DB: db, Prices: exactPriceCatalogue{fact: catalogueFact}, MoneyCap: policy, Now: func() time.Time { return now }}
	if err := replayService.ValidateAkashQuoteReplay(context.Background(), intentRecord, receipt); err != nil {
		t.Fatalf("unchanged fresh quote replay refused: %v", err)
	}
	changedRaw := rawFact
	changedRaw.USDPerToken = "2"
	changedRaw.FXSourceVersion = "synthetic-fx-v2"
	changedFact, err := changedRaw.NormalizeCatalogueFact(now, 30*24*60*60)
	if err != nil {
		t.Fatal(err)
	}
	replayService.Prices = exactPriceCatalogue{fact: changedFact}
	if err := replayService.ValidateAkashQuoteReplay(context.Background(), intentRecord, receipt); err == nil {
		t.Fatal("changed FX conversion replay was accepted")
	}
	staleFact := catalogueFact
	staleFact.ExpiresAt = now.Add(-time.Second)
	replayService.Prices = exactPriceCatalogue{fact: staleFact}
	if err := replayService.ValidateAkashQuoteReplay(context.Background(), intentRecord, receipt); err == nil {
		t.Fatal("stale source replay was accepted")
	}
}
