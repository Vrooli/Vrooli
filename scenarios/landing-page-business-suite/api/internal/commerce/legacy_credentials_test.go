package commerce

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestMigrateLegacyPaymentCredentialsWritesOnlyMissingValues(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE payment_settings (id INTEGER PRIMARY KEY, publishable_key TEXT, secret_key TEXT, webhook_secret TEXT)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret) VALUES (1, 'pk_old', 'rk_old', 'whsec_old')`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	stored := map[string]string{"stripe-secret-key": "rk_existing"}
	read := func(_ context.Context, field string) (string, error) {
		if value, ok := stored[field]; ok {
			return value, nil
		}
		return "", credentialauthority.ErrUnconfigured
	}
	writes := 0
	write := func(_ context.Context, field, value string) error {
		stored[field] = value
		writes++
		return nil
	}

	migrated, err := MigrateLegacyPaymentCredentials(context.Background(), db, read, write)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 2 || writes != 2 {
		t.Fatalf("migrated=%d writes=%d, want 2", migrated, writes)
	}
	if stored["stripe-publishable-key"] != "pk_old" || stored["stripe-webhook-secret"] != "whsec_old" {
		t.Fatalf("legacy values were not written: %#v", stored)
	}
}

func TestMigrateLegacyPaymentCredentialsRefusesAuthorityFailure(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE payment_settings (id INTEGER PRIMARY KEY, publishable_key TEXT, secret_key TEXT, webhook_secret TEXT)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO payment_settings (id, secret_key) VALUES (1, 'rk_old')`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	providerErr := errors.New("credential store locked")
	_, err = MigrateLegacyPaymentCredentials(context.Background(), db,
		func(context.Context, string) (string, error) { return "", providerErr },
		func(context.Context, string, string) error { return nil },
	)
	if err == nil || !errors.Is(err, providerErr) {
		t.Fatalf("migration error=%v, want provider error", err)
	}
}
