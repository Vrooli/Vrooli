package profiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"deployment-manager/shared"
)

// SQLRepository implements Repository using a SQL database.
type SQLRepository struct {
	db shared.RoutedDBTX
}

// NewSQLRepository creates a new SQLRepository.
func NewSQLRepository(db shared.RoutedDBTX) *SQLRepository {
	return &SQLRepository{db: db}
}

// List returns all profiles ordered by creation date (newest first).
func (r *SQLRepository) List(ctx context.Context) ([]Profile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, scenario, tiers, swaps, secrets, settings, version, created_at, updated_at, created_by, updated_by
		FROM profiles
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var tiersJSON, swapsJSON, secretsJSON, settingsJSON []byte

		if err := rows.Scan(&p.ID, &p.Name, &p.Scenario, &tiersJSON, &swapsJSON, &secretsJSON, &settingsJSON, &p.Version, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy); err != nil {
			return nil, fmt.Errorf("scan profile: %w", err)
		}

		if err := decodeProfileJSON(&p, tiersJSON, swapsJSON, secretsJSON, settingsJSON); err != nil {
			return nil, fmt.Errorf("decode profile %q: %w", p.ID, err)
		}

		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}

// Get retrieves a profile by ID or name.
func (r *SQLRepository) Get(ctx context.Context, idOrName string) (*Profile, error) {
	var p Profile
	var tiersJSON, swapsJSON, secretsJSON, settingsJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, scenario, tiers, swaps, secrets, settings, version, created_at, updated_at, created_by, updated_by
		FROM profiles
		WHERE id = $1 OR name = $1
	`, idOrName).Scan(&p.ID, &p.Name, &p.Scenario, &tiersJSON, &swapsJSON, &secretsJSON, &settingsJSON, &p.Version, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := decodeProfileJSON(&p, tiersJSON, swapsJSON, secretsJSON, settingsJSON); err != nil {
		return nil, fmt.Errorf("decode profile %q: %w", p.ID, err)
	}

	return &p, nil
}

// Create stores a new profile and returns its generated ID.
func (r *SQLRepository) Create(ctx context.Context, profile *Profile) (string, error) {
	if profile == nil {
		return "", fmt.Errorf("profile is required")
	}
	tiersJSON, swapsJSON, secretsJSON, settingsJSON, err := marshalProfileJSON(profile)
	if err != nil {
		return "", err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin profile creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO profiles (id, name, scenario, tiers, swaps, secrets, settings, version, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, 'system', 'system')
	`, profile.ID, profile.Name, profile.Scenario, tiersJSON, swapsJSON, secretsJSON, settingsJSON)
	if err != nil {
		return "", err
	}

	// Version history is part of the durable profile record.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO profile_versions (profile_id, version, name, scenario, tiers, swaps, secrets, settings, created_by, change_description)
		VALUES ($1, 1, $2, $3, $4, $5, $6, $7, 'system', 'Initial profile creation')
	`, profile.ID, profile.Name, profile.Scenario, tiersJSON, swapsJSON, secretsJSON, settingsJSON); err != nil {
		return "", fmt.Errorf("create profile history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit profile creation: %w", err)
	}
	return profile.ID, nil
}

// Update modifies an existing profile and increments its version.
func (r *SQLRepository) Update(ctx context.Context, idOrName string, updates map[string]interface{}) (*Profile, error) {
	// Fetch current profile
	current, err := r.Get(ctx, idOrName)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}

	// Apply updates
	if updates["tiers"] != nil {
		current.Tiers = updates["tiers"]
	}
	if updates["swaps"] != nil {
		current.Swaps = updates["swaps"]
	}
	if updates["secrets"] != nil {
		current.Secrets = updates["secrets"]
	}
	if updates["settings"] != nil {
		current.Settings = updates["settings"]
	}

	newVersion := current.Version + 1
	tiersJSON, swapsJSON, secretsJSON, settingsJSON, err := marshalProfileJSON(current)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin profile update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
		UPDATE profiles
		SET tiers = $1, swaps = $2, secrets = $3, settings = $4, version = $5, updated_at = CURRENT_TIMESTAMP, updated_by = 'system'
		WHERE id = $6
	`, tiersJSON, swapsJSON, secretsJSON, settingsJSON, newVersion, current.ID)
	if err != nil {
		return nil, err
	}

	// Version history is part of the durable profile record.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO profile_versions (profile_id, version, name, scenario, tiers, swaps, secrets, settings, created_by, change_description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'system', 'Profile updated')
	`, current.ID, newVersion, current.Name, current.Scenario, tiersJSON, swapsJSON, secretsJSON, settingsJSON); err != nil {
		return nil, fmt.Errorf("create profile history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit profile update: %w", err)
	}

	current.Version = newVersion
	return current, nil
}

// Delete removes a profile by ID or name. Returns true if a row was deleted.
func (r *SQLRepository) Delete(ctx context.Context, idOrName string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = $1 OR name = $1`, idOrName)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read deleted profile count: %w", err)
	}
	return rowsAffected > 0, nil
}

// GetVersions returns the version history for a profile.
func (r *SQLRepository) GetVersions(ctx context.Context, idOrName string) ([]Version, error) {
	// Resolve to actual ID first
	var actualID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE id = $1 OR name = $1`, idOrName).Scan(&actualID)
	if err == sql.ErrNoRows {
		return []Version{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT profile_id, version, name, scenario, tiers, swaps, secrets, settings, created_at, created_by, COALESCE(change_description, '')
		FROM profile_versions
		WHERE profile_id = $1
		ORDER BY version DESC
	`, actualID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		var v Version
		var tiersJSON, swapsJSON, secretsJSON, settingsJSON []byte

		if err := rows.Scan(&v.ProfileID, &v.Version, &v.Name, &v.Scenario, &tiersJSON, &swapsJSON, &secretsJSON, &settingsJSON, &v.CreatedAt, &v.CreatedBy, &v.ChangeDescription); err != nil {
			return nil, fmt.Errorf("scan profile version: %w", err)
		}

		if err := decodeVersionJSON(&v, tiersJSON, swapsJSON, secretsJSON, settingsJSON); err != nil {
			return nil, fmt.Errorf("decode profile version %d: %w", v.Version, err)
		}

		versions = append(versions, v)
	}

	return versions, rows.Err()
}

// GetScenarioAndTier retrieves just the scenario name and tier count for a profile.
// Returns ErrNotFound if the profile does not exist.
func (r *SQLRepository) GetScenarioAndTier(ctx context.Context, idOrName string) (string, int, error) {
	var scenario string
	var tierCount int

	err := r.db.QueryRowContext(ctx, `
		SELECT scenario, COALESCE(jsonb_array_length(tiers), 0)
		FROM profiles
		WHERE name = $1 OR id = $1
	`, idOrName).Scan(&scenario, &tierCount)

	if err == sql.ErrNoRows {
		return "", 0, ErrNotFound
	}
	if err != nil {
		return "", 0, err
	}

	return scenario, tierCount, nil
}

// AddSwap adds a swap to a profile's swap list.
func (r *SQLRepository) AddSwap(ctx context.Context, idOrName string, swap Swap) error {
	// Get current swaps
	var swapsJSON []byte
	var profileID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(swaps, '[]'::jsonb)
		FROM profiles
		WHERE id = $1 OR name = $1
	`, idOrName).Scan(&profileID, &swapsJSON)

	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	// Parse existing swaps
	var swaps []Swap
	if err := json.Unmarshal(swapsJSON, &swaps); err != nil {
		return fmt.Errorf("decode profile swaps: %w", err)
	}

	// Check if swap already exists (same from->to)
	for i, existing := range swaps {
		if existing.From == swap.From && existing.To == swap.To {
			// Update existing swap
			swaps[i] = swap
			newSwapsJSON, err := json.Marshal(swaps)
			if err != nil {
				return fmt.Errorf("encode profile swaps: %w", err)
			}
			_, err = r.db.ExecContext(ctx, `
				UPDATE profiles
				SET swaps = $1, updated_at = CURRENT_TIMESTAMP, version = version + 1
				WHERE id = $2
			`, newSwapsJSON, profileID)
			return err
		}
	}

	// Add new swap
	swaps = append(swaps, swap)
	newSwapsJSON, err := json.Marshal(swaps)
	if err != nil {
		return fmt.Errorf("encode profile swaps: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE profiles
		SET swaps = $1, updated_at = CURRENT_TIMESTAMP, version = version + 1
		WHERE id = $2
	`, newSwapsJSON, profileID)

	return err
}

// GetSwaps returns the swaps configured for a profile.
func (r *SQLRepository) GetSwaps(ctx context.Context, idOrName string) ([]Swap, error) {
	var swapsJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(swaps, '[]'::jsonb)
		FROM profiles
		WHERE id = $1 OR name = $1
	`, idOrName).Scan(&swapsJSON)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var swaps []Swap
	if err := json.Unmarshal(swapsJSON, &swaps); err != nil {
		return nil, fmt.Errorf("decode profile swaps: %w", err)
	}

	return swaps, nil
}

func marshalProfileJSON(profile *Profile) ([]byte, []byte, []byte, []byte, error) {
	tiers, err := json.Marshal(profile.Tiers)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode profile tiers: %w", err)
	}
	swaps, err := json.Marshal(profile.Swaps)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode profile swaps: %w", err)
	}
	secrets, err := json.Marshal(profile.Secrets)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode profile secrets: %w", err)
	}
	settings, err := json.Marshal(profile.Settings)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode profile settings: %w", err)
	}
	return tiers, swaps, secrets, settings, nil
}

func decodeProfileJSON(profile *Profile, tiers, swaps, secrets, settings []byte) error {
	if err := json.Unmarshal(tiers, &profile.Tiers); err != nil {
		return fmt.Errorf("tiers: %w", err)
	}
	if err := json.Unmarshal(swaps, &profile.Swaps); err != nil {
		return fmt.Errorf("swaps: %w", err)
	}
	if err := json.Unmarshal(secrets, &profile.Secrets); err != nil {
		return fmt.Errorf("secrets: %w", err)
	}
	if err := json.Unmarshal(settings, &profile.Settings); err != nil {
		return fmt.Errorf("settings: %w", err)
	}
	return nil
}

func decodeVersionJSON(version *Version, tiers, swaps, secrets, settings []byte) error {
	if err := json.Unmarshal(tiers, &version.Tiers); err != nil {
		return fmt.Errorf("tiers: %w", err)
	}
	if err := json.Unmarshal(swaps, &version.Swaps); err != nil {
		return fmt.Errorf("swaps: %w", err)
	}
	if err := json.Unmarshal(secrets, &version.Secrets); err != nil {
		return fmt.Errorf("secrets: %w", err)
	}
	if err := json.Unmarshal(settings, &version.Settings); err != nil {
		return fmt.Errorf("settings: %w", err)
	}
	return nil
}
