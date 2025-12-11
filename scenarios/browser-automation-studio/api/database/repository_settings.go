package database

import (
	"context"
	"database/sql"
	"errors"
)

// GetSetting retrieves a setting value by key.
func (r *repository) GetSetting(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM user_settings WHERE key = $1`

	var value string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // Return empty string for missing keys
		}
		return "", err
	}

	return value, nil
}

// SetSetting stores a setting value by key (upsert).
func (r *repository) SetSetting(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO user_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, key, value)
	return err
}

// DeleteSetting removes a setting by key.
func (r *repository) DeleteSetting(ctx context.Context, key string) error {
	query := `DELETE FROM user_settings WHERE key = $1`
	_, err := r.db.ExecContext(ctx, query, key)
	return err
}
