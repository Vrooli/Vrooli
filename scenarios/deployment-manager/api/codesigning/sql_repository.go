package codesigning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a signing config is not found.
var ErrNotFound = errors.New("signing config not found")

// ErrProfileNotFound is returned when the parent profile doesn't exist.
var ErrProfileNotFound = errors.New("profile not found")

// SQLRepository implements Repository using a SQL database.
// Signing configurations are stored as JSONB in the profiles table.
type SQLRepository struct {
	db *sql.DB
}

// NewSQLRepository creates a new SQL-backed signing config repository.
func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

// EnsureSchema ensures the signing_config column exists in the profiles table.
// This is a safe operation that can be called on startup.
func (r *SQLRepository) EnsureSchema(ctx context.Context) error {
	// Check if column exists
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'profiles' AND column_name = 'signing_config'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check signing_config column: %w", err)
	}

	if !exists {
		// Add the column
		_, err = r.db.ExecContext(ctx, `
			ALTER TABLE profiles ADD COLUMN signing_config JSONB DEFAULT NULL
		`)
		if err != nil {
			return fmt.Errorf("add signing_config column: %w", err)
		}
	}

	return nil
}

// Get retrieves the signing config for a profile.
// Returns nil (not error) if profile exists but has no signing config.
// Returns ErrProfileNotFound if the profile doesn't exist.
func (r *SQLRepository) Get(ctx context.Context, profileID string) (*SigningConfig, error) {
	var signingConfigJSON sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT signing_config
		FROM profiles
		WHERE id = $1 OR name = $1
	`, profileID).Scan(&signingConfigJSON)

	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get signing config: %w", err)
	}

	// No signing config configured
	if !signingConfigJSON.Valid || signingConfigJSON.String == "" {
		return nil, nil
	}

	var config SigningConfig
	if err := json.Unmarshal([]byte(signingConfigJSON.String), &config); err != nil {
		return nil, fmt.Errorf("parse signing config: %w", err)
	}

	return &config, nil
}

// Save stores or updates the signing config for a profile.
func (r *SQLRepository) Save(ctx context.Context, profileID string, config *SigningConfig) error {
	// Serialize config to JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("serialize signing config: %w", err)
	}

	// Update the profile's signing config
	result, err := r.db.ExecContext(ctx, `
		UPDATE profiles
		SET signing_config = $1, updated_at = NOW(), version = version + 1
		WHERE id = $2 OR name = $2
	`, configJSON, profileID)
	if err != nil {
		return fmt.Errorf("save signing config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrProfileNotFound
	}

	return nil
}

// Delete removes the signing config for a profile.
func (r *SQLRepository) Delete(ctx context.Context, profileID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE profiles
		SET signing_config = NULL, updated_at = NOW(), version = version + 1
		WHERE id = $1 OR name = $1
	`, profileID)
	if err != nil {
		return fmt.Errorf("delete signing config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrProfileNotFound
	}

	return nil
}

// GetForPlatform retrieves config for a specific platform only.
func (r *SQLRepository) GetForPlatform(ctx context.Context, profileID string, platform string) (interface{}, error) {
	config, err := r.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	switch platform {
	case PlatformWindows:
		return config.Windows, nil
	case PlatformMacOS:
		return config.MacOS, nil
	case PlatformLinux:
		return config.Linux, nil
	default:
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}
}

// SaveForPlatform updates only a specific platform's config.
func (r *SQLRepository) SaveForPlatform(ctx context.Context, profileID string, platform string, platformConfig interface{}) error {
	// Get current config
	config, err := r.Get(ctx, profileID)
	if err == ErrProfileNotFound {
		return err
	}
	if err != nil {
		return fmt.Errorf("get current signing config: %w", err)
	}

	// Initialize if needed
	if config == nil {
		config = &SigningConfig{Enabled: true}
	}

	// Update the platform-specific config
	switch platform {
	case PlatformWindows:
		if c, ok := platformConfig.(*WindowsSigningConfig); ok {
			config.Windows = c
		} else {
			return fmt.Errorf("invalid windows config type")
		}
	case PlatformMacOS:
		if c, ok := platformConfig.(*MacOSSigningConfig); ok {
			config.MacOS = c
		} else {
			return fmt.Errorf("invalid macos config type")
		}
	case PlatformLinux:
		if c, ok := platformConfig.(*LinuxSigningConfig); ok {
			config.Linux = c
		} else {
			return fmt.Errorf("invalid linux config type")
		}
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}

	return r.Save(ctx, profileID, config)
}

// DeleteForPlatform removes the signing config for a specific platform only.
func (r *SQLRepository) DeleteForPlatform(ctx context.Context, profileID string, platform string) error {
	config, err := r.Get(ctx, profileID)
	if err != nil {
		return err
	}
	if config == nil {
		return nil // Nothing to delete
	}

	switch platform {
	case PlatformWindows:
		config.Windows = nil
	case PlatformMacOS:
		config.MacOS = nil
	case PlatformLinux:
		config.Linux = nil
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}

	// If no platforms configured, delete the entire config
	if config.Windows == nil && config.MacOS == nil && config.Linux == nil {
		return r.Delete(ctx, profileID)
	}

	return r.Save(ctx, profileID, config)
}
