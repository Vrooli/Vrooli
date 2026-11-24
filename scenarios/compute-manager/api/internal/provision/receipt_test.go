package provision

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestGetSpendReceiptReturnsPersistedQuoteAndCapDecision(t *testing.T) {
	db, err := sql.Open("sqlite", "file:provision-receipt?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE spend_quotes(idempotency_key TEXT PRIMARY KEY,provider TEXT,offer_id TEXT,region TEXT,size TEXT,ipv4 BOOLEAN,term TEXT,currency TEXT,server_minor INTEGER,ipv4_minor INTEGER,monthly_minor INTEGER,tax_basis TEXT,source_url TEXT,source_version TEXT,observed_at TEXT,expires_at TEXT,state TEXT,reason TEXT,quote_json TEXT,created_at TEXT);
	CREATE TABLE spend_admissions(idempotency_key TEXT PRIMARY KEY,cap_currency TEXT,cap_monthly_minor INTEGER,cap_tax_basis TEXT,decision TEXT,reason TEXT);
	CREATE TABLE instance_intents(idempotency_key TEXT PRIMARY KEY,state TEXT,instance_id TEXT);
	INSERT INTO spend_quotes VALUES('request-key','hetzner','cx23-nbg1','nbg1','cx23',1,'monthly','USD',649,60,709,'pre-tax','https://api.hetzner.cloud/v1/pricing','etag-1','2026-11-23T12:00:00Z','2026-11-23T12:05:00Z','approved','', '{"idempotency_key":"request-key","provider":"hetzner","offer_id":"cx23-nbg1","region":"nbg1","size":"cx23","currency":"USD","monthly_minor":709,"source_url":"https://api.hetzner.cloud/v1/pricing","source_version":"etag-1"}', '2026-11-23T12:00:01Z');
	INSERT INTO spend_admissions VALUES('request-key','USD',709,'pre-tax','approved','');
	INSERT INTO instance_intents VALUES('request-key','refused','');`)
	if err != nil {
		t.Fatal(err)
	}
	service := Service{DB: db}
	receipt, err := service.GetSpendReceipt(context.Background(), "request-key")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Provider != "hetzner" || receipt.OfferID != "cx23-nbg1" || receipt.Size != "cx23" || !receipt.IPv4 || receipt.ServerMinor != 649 || receipt.IPv4Minor != 60 || receipt.MonthlyMinor != 709 || receipt.Currency != "USD" || receipt.TaxBasis != "pre-tax" || receipt.SourceURL == "" || receipt.SourceVersion != "etag-1" || receipt.CapMonthly != 709 || receipt.Decision != "approved" || receipt.RequestState != "refused" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if !receipt.ObservedAt.Equal(time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)) || !receipt.ExpiresAt.Equal(time.Date(2026, 11, 23, 12, 5, 0, 0, time.UTC)) {
		t.Fatalf("receipt times = %s/%s", receipt.ObservedAt, receipt.ExpiresAt)
	}
	if _, err := service.GetSpendReceipt(context.Background(), "missing"); !errors.Is(err, ErrSpendReceiptNotFound) {
		t.Fatalf("missing receipt error = %v", err)
	}
}
