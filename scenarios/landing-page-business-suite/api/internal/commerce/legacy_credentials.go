package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// MigrateLegacyPaymentCredentials copies values from the pre-authority
// payment_settings columns into the authority. It is intentionally a
// one-shot operation: callers run it against a copy of an existing database
// before applying the schema that removes those columns.
//
// A provider error is never treated as an empty credential. Existing authority
// values are preserved and are not counted as migrated.
func MigrateLegacyPaymentCredentials(ctx context.Context, db PaymentSettingsLegacyStore, read func(context.Context, string) (string, error), write func(context.Context, string, string) error) (int, error) {
	if read == nil || write == nil {
		return 0, fmt.Errorf("legacy credential migration requires authority read and write functions")
	}
	row := db.QueryRowContext(ctx, `
		SELECT publishable_key, secret_key, webhook_secret
		FROM payment_settings
		WHERE id = 1
	`)
	var publishable, secret, webhook sql.NullString
	if err := row.Scan(&publishable, &secret, &webhook); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("read legacy payment credentials: %w", err)
	}

	migrated := 0
	for _, credential := range []struct {
		field string
		value sql.NullString
	}{
		{field: "stripe-publishable-key", value: publishable},
		{field: "stripe-secret-key", value: secret},
		{field: "stripe-webhook-secret", value: webhook},
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

// PaymentSettingsLegacyStore is the narrow read boundary needed by the
// one-shot migration. It is separate from PaymentSettingsStore because the
// live service must not expose or query retired cleartext columns.
type PaymentSettingsLegacyStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
