package profiles

import (
	"context"
	"database/sql"
	"time"

	"deployment-manager/shared"
)

// LPBSReleaseConfig holds the LPBS coordinates that a profile needs to
// publish desktop artifacts. Stored 1:1 with profiles.
type LPBSReleaseConfig struct {
	ProfileID         string    `json:"profile_id"`
	LPBSDomain        string    `json:"lpbs_domain"`
	LPBSRemoteProfile string    `json:"lpbs_remote_profile"`
	LPBSAppKey        string    `json:"lpbs_app_key"`
	DefaultChannel    string    `json:"default_channel"`
	UpdateURL         string    `json:"update_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// LPBSReleaseConfigRepository persists LPBS release config rows.
type LPBSReleaseConfigRepository interface {
	Get(ctx context.Context, profileID string) (*LPBSReleaseConfig, error)
	Upsert(ctx context.Context, cfg *LPBSReleaseConfig) error
	Delete(ctx context.Context, profileID string) error
}

// SQLLPBSReleaseConfigRepository implements LPBSReleaseConfigRepository with PostgreSQL.
type SQLLPBSReleaseConfigRepository struct {
	db shared.RoutedDBTX
}

// NewSQLLPBSReleaseConfigRepository creates a new SQL-backed LPBS config repository.
func NewSQLLPBSReleaseConfigRepository(db shared.RoutedDBTX) *SQLLPBSReleaseConfigRepository {
	return &SQLLPBSReleaseConfigRepository{db: db}
}

// Get returns the LPBS config for a profile, or (nil, nil) if none is set.
func (r *SQLLPBSReleaseConfigRepository) Get(ctx context.Context, profileID string) (*LPBSReleaseConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT profile_id, lpbs_domain, lpbs_remote_profile, lpbs_app_key,
		       default_channel, update_url, created_at, updated_at
		FROM profile_lpbs_release_config
		WHERE profile_id = $1
	`, profileID)

	cfg := &LPBSReleaseConfig{}
	err := row.Scan(
		&cfg.ProfileID, &cfg.LPBSDomain, &cfg.LPBSRemoteProfile, &cfg.LPBSAppKey,
		&cfg.DefaultChannel, &cfg.UpdateURL, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// Upsert writes the LPBS config row, creating or updating in place.
func (r *SQLLPBSReleaseConfigRepository) Upsert(ctx context.Context, cfg *LPBSReleaseConfig) error {
	if cfg.DefaultChannel == "" {
		cfg.DefaultChannel = "stable"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO profile_lpbs_release_config
			(profile_id, lpbs_domain, lpbs_remote_profile, lpbs_app_key,
			 default_channel, update_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (profile_id) DO UPDATE SET
			lpbs_domain         = EXCLUDED.lpbs_domain,
			lpbs_remote_profile = EXCLUDED.lpbs_remote_profile,
			lpbs_app_key        = EXCLUDED.lpbs_app_key,
			default_channel     = EXCLUDED.default_channel,
			update_url          = EXCLUDED.update_url,
			updated_at          = CURRENT_TIMESTAMP
	`,
		cfg.ProfileID, cfg.LPBSDomain, cfg.LPBSRemoteProfile, cfg.LPBSAppKey,
		cfg.DefaultChannel, cfg.UpdateURL,
	)
	return err
}

// Delete removes the LPBS config for a profile.
func (r *SQLLPBSReleaseConfigRepository) Delete(ctx context.Context, profileID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM profile_lpbs_release_config WHERE profile_id = $1`, profileID)
	return err
}
