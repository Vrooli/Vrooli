package delivery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// MigrateLegacyStorageCredentials copies the pre-authority S3 columns into
// the authority. Run this one-shot operation against an existing database
// before applying the schema that removes the columns.
func MigrateLegacyStorageCredentials(ctx context.Context, db LegacyStorageStore, read func(context.Context, string) (string, error), write func(context.Context, string, string) error) (int, error) {
	if read == nil || write == nil {
		return 0, fmt.Errorf("legacy storage credential migration requires authority read and write functions")
	}
	row := db.QueryRowContext(ctx, `
		SELECT access_key_id, secret_access_key, session_token
		FROM download_storage_settings
		WHERE bundle_key = $1
		LIMIT 1
	`, "business_suite")
	var accessKey, secretKey, sessionToken sql.NullString
	if err := row.Scan(&accessKey, &secretKey, &sessionToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("read legacy storage credentials: %w", err)
	}

	migrated := 0
	for _, credential := range []struct {
		field string
		value sql.NullString
	}{
		{field: "delivery-s3-access-key-id", value: accessKey},
		{field: "delivery-s3-secret-access-key", value: secretKey},
		{field: "delivery-s3-session-token", value: sessionToken},
	} {
		value := strings.TrimSpace(credential.value.String)
		if !credential.value.Valid || value == "" {
			continue
		}
		existing, err := read(ctx, credential.field)
		if err == nil && strings.TrimSpace(existing) != "" {
			continue
		}
		if err != nil && !errors.Is(err, credentialauthority.ErrUnconfigured) {
			return migrated, fmt.Errorf("read existing authority credential %s: %w", credential.field, err)
		}
		if err := write(ctx, credential.field, value); err != nil {
			return migrated, fmt.Errorf("write legacy credential %s to authority: %w", credential.field, err)
		}
		migrated++
	}
	return migrated, nil
}

// LegacyStorageStore is intentionally separate from Store. Live delivery
// reads never select retired credential columns.
type LegacyStorageStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
