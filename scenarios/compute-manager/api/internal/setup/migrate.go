package setup

import (
	"context"
	"database/sql"
)

// MigrateCompatibility adds durable conversion-evidence columns to existing
// observations. Blank legacy provenance remains blank and cannot be inferred.
func MigrateCompatibility(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, column := range []string{"provenance_json", "provenance_fingerprint"} {
		rows, probeErr := tx.QueryContext(ctx, `SELECT `+column+` FROM provider_setup_prices LIMIT 0`)
		if probeErr == nil {
			if err := rows.Close(); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `ALTER TABLE provider_setup_prices ADD COLUMN `+column+` TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return tx.Commit()
}
