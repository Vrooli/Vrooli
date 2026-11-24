package setup

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"compute-manager/internal/provider"
	_ "modernc.org/sqlite"
)

func TestCompatibilityMigrationPreservesLegacyPricesWithoutInventingProvenance(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE provider_setup_prices(provider_id TEXT,region TEXT,size TEXT,offer_id TEXT,currency TEXT,server_minor INTEGER,ipv4_minor INTEGER,tax_basis TEXT,source_url TEXT,source_version TEXT,observed_at TEXT,expires_at TEXT,PRIMARY KEY(provider_id,region,size)); INSERT INTO provider_setup_prices VALUES('akash','sandbox-2','lease','offer','USD',123,0,'pre-tax','https://old.invalid','old', '2026-11-24T11:00:00Z','2026-11-24T13:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateCompatibility(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	facts, err := NewRepository(db).Prices(context.Background(), "akash")
	if err != nil || len(facts) != 1 {
		t.Fatalf("legacy facts=%+v err=%v", facts, err)
	}
	if facts[0].ProvenanceJSON != "" || facts[0].ProvenanceFingerprint != "" || facts[0].ServerMinor != 123 {
		t.Fatalf("migration inferred or lost values: %+v", facts[0])
	}
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	newFact := provider.PriceFact{Provider: "akash", Region: "sandbox-2", Size: "lease", OfferID: "offer", Currency: "USD", ServerMinor: 123, TaxBasis: "pre-tax", SourceURL: "https://new.invalid", SourceVersion: "new", ObservedAt: now, ExpiresAt: now.Add(time.Hour), ProvenanceJSON: `{"version":"fixture-v1"}`, ProvenanceFingerprint: "fingerprint"}
	if err := NewRepository(db).PutPrice(context.Background(), newFact); err != nil {
		t.Fatal(err)
	}
	facts, err = NewRepository(db).Prices(context.Background(), "akash")
	if err != nil || len(facts) != 1 {
		t.Fatalf("updated facts=%+v err=%v", facts, err)
	}
	if facts[0].ProvenanceJSON != newFact.ProvenanceJSON || facts[0].ProvenanceFingerprint != newFact.ProvenanceFingerprint {
		t.Fatalf("price repository lost provenance: %+v", facts[0])
	}
}
