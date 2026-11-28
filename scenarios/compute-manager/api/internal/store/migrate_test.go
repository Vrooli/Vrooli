package store

import (
	"context"
	"database/sql"
	"testing"

	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestMigrateCompatibilityBackfillsOnlyRecordedSourceAndPreservesRows(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	_, err = db.Exec(`CREATE TABLE instances (
		id TEXT PRIMARY KEY, provider TEXT NOT NULL, provider_instance_id TEXT NOT NULL,
		state TEXT NOT NULL, region TEXT NOT NULL DEFAULT '', size TEXT NOT NULL DEFAULT '',
		image TEXT NOT NULL DEFAULT '', address TEXT NOT NULL DEFAULT '', ssh_user TEXT NOT NULL DEFAULT '',
		ssh_port INTEGER NOT NULL DEFAULT 0, bridge_machine_id TEXT NOT NULL DEFAULT '',
		tenant TEXT NOT NULL DEFAULT '', reservation_id TEXT NOT NULL DEFAULT '',
		tags_json TEXT NOT NULL DEFAULT '{}', expires_at TEXT NOT NULL, created_at TEXT NOT NULL,
		running_at TEXT NOT NULL DEFAULT '', destroyed_at TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE spend_quotes (
		id TEXT PRIMARY KEY, idempotency_key TEXT NOT NULL UNIQUE, provider TEXT NOT NULL,
		offer_id TEXT NOT NULL DEFAULT '', region TEXT NOT NULL, size TEXT NOT NULL,
		ipv4 BOOLEAN NOT NULL, term TEXT NOT NULL, currency TEXT NOT NULL,
		server_minor BIGINT NOT NULL DEFAULT 0, ipv4_minor BIGINT NOT NULL DEFAULT 0,
		monthly_minor BIGINT NOT NULL, tax_basis TEXT NOT NULL, source_version TEXT NOT NULL,
		observed_at TEXT NOT NULL, expires_at TEXT NOT NULL, state TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '', quote_json TEXT NOT NULL, created_at TEXT NOT NULL
	);
		INSERT INTO spend_quotes(id,idempotency_key,provider,region,size,ipv4,term,currency,monthly_minor,tax_basis,source_version,observed_at,expires_at,state,quote_json,created_at) VALUES
		('recoverable','recoverable','hetzner','nbg1','cx23',1,'monthly','EUR',500,'pre-tax','fixture','now','later','approved','{"source_url":"https://catalog.example/pricing"}','now'),
		('unknown','unknown','hetzner','nbg1','cx23',1,'monthly','EUR',500,'pre-tax','fixture','now','later','refused','{"currency":"EUR"}','now'),
		('existing','existing','hetzner','nbg1','cx23',1,'monthly','EUR',500,'pre-tax','fixture','now','later','approved','{"source_url":"https://stale.example/price"}','now')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("domain schema should add source_url with its safe empty default: %v", err)
	}
	// Model a database whose invoice receipt predates persisted service ranges.
	// The migration must add unknown defaults and keep the original receipt.
	if _, err := db.Exec(`DROP TABLE invoice_reconciliation_receipts;
		CREATE TABLE invoice_reconciliation_receipts (external_id TEXT PRIMARY KEY, provider TEXT NOT NULL);
		INSERT INTO invoice_reconciliation_receipts(external_id,provider) VALUES('legacy-receipt','digitalocean')`); err != nil {
		t.Fatalf("prepare legacy invoice receipt table: %v", err)
	}
	if _, err := db.Exec(`UPDATE spend_quotes SET source_url='https://existing.example/price' WHERE id='existing'`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateCompatibility(ctx, db); err != nil {
		t.Fatal(err)
	}
	columns, err := db.Query(`SELECT spend_idempotency_key FROM instances LIMIT 0`)
	if err != nil {
		t.Fatalf("additive instance field is missing: %v", err)
	}
	if err := columns.Close(); err != nil {
		t.Fatal(err)
	}
	if err := MigrateCompatibility(ctx, db); err != nil {
		t.Fatalf("migration should be idempotent: %v", err)
	}
	var legacyFrom, legacyTo, legacyProvider string
	if err := db.QueryRow(`SELECT provider,range_from,range_to FROM invoice_reconciliation_receipts WHERE external_id='legacy-receipt'`).Scan(&legacyProvider, &legacyFrom, &legacyTo); err != nil {
		t.Fatalf("migrated invoice range columns are missing: %v", err)
	}
	if legacyProvider != "digitalocean" || legacyFrom != "" || legacyTo != "" {
		t.Fatalf("legacy receipt was inferred or changed: provider=%q range=(%q,%q)", legacyProvider, legacyFrom, legacyTo)
	}
	rows, err := db.Query(`SELECT id,source_url,quote_json FROM spend_quotes ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := map[string]string{
		"existing":    "https://existing.example/price",
		"recoverable": "https://catalog.example/pricing",
		"unknown":     "",
	}
	count := 0
	for rows.Next() {
		var id, source, raw string
		if err := rows.Scan(&id, &source, &raw); err != nil {
			t.Fatal(err)
		}
		count++
		if source != want[id] {
			t.Errorf("%s source_url = %q, want %q", id, source, want[id])
		}
		if raw == "" {
			t.Errorf("%s quote payload was not preserved", id)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("preserved %d rows, want 3", count)
	}
}
