package delivery

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestMigrateLegacyStorageCredentialsWritesOnlyMissingValues(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE download_storage_settings (bundle_key TEXT PRIMARY KEY, access_key_id TEXT, secret_access_key TEXT, session_token TEXT)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO download_storage_settings (bundle_key, access_key_id, secret_access_key, session_token) VALUES ('business_suite', 'AKIA_old', 'secret_old', 'token_old')`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	stored := map[string]string{"delivery-s3-secret-access-key": "secret_existing"}
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

	migrated, err := MigrateLegacyStorageCredentials(context.Background(), db, read, write)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 2 || writes != 2 {
		t.Fatalf("migrated=%d writes=%d, want 2", migrated, writes)
	}
	if stored["delivery-s3-access-key-id"] != "AKIA_old" || stored["delivery-s3-session-token"] != "token_old" {
		t.Fatalf("legacy values were not written: %#v", stored)
	}
}

func TestMigrateLegacyStorageCredentialsRefusesAuthorityFailure(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE download_storage_settings (bundle_key TEXT PRIMARY KEY, access_key_id TEXT, secret_access_key TEXT, session_token TEXT)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO download_storage_settings (bundle_key, secret_access_key) VALUES ('business_suite', 'secret_old')`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	providerErr := errors.New("credential store locked")
	_, err = MigrateLegacyStorageCredentials(context.Background(), db,
		func(context.Context, string) (string, error) { return "", providerErr },
		func(context.Context, string, string) error { return nil },
	)
	if err == nil || !errors.Is(err, providerErr) {
		t.Fatalf("migration error=%v, want provider error", err)
	}
}
