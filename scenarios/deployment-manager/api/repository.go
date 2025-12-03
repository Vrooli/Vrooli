package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Domain-specific errors for the repository layer.
// These errors abstract away database implementation details from handlers.
var (
	// ErrProfileNotFound indicates the requested profile does not exist.
	ErrProfileNotFound = errors.New("profile not found")
)

// Profile represents a deployment profile stored in the database.
type Profile struct {
	ID         string
	Name       string
	Scenario   string
	Tiers      interface{}
	Swaps      interface{}
	Secrets    interface{}
	Settings   interface{}
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CreatedBy  string
	UpdatedBy  string
}

// ProfileVersion represents a versioned snapshot of a profile.
type ProfileVersion struct {
	ProfileID         string
	Version           int
	Name              string
	Scenario          string
	Tiers             interface{}
	Swaps             interface{}
	Secrets           interface{}
	Settings          interface{}
	CreatedAt         time.Time
	CreatedBy         string
	ChangeDescription string
}

// ProfileRepository defines the interface for profile storage operations.
// This seam allows tests to substitute the real database with mocks.
type ProfileRepository interface {
	// List returns all profiles ordered by creation date (newest first).
	List(ctx context.Context) ([]Profile, error)

	// Get retrieves a profile by ID or name.
	Get(ctx context.Context, idOrName string) (*Profile, error)

	// Create stores a new profile and returns its generated ID.
	Create(ctx context.Context, profile *Profile) (string, error)

	// Update modifies an existing profile and increments its version.
	Update(ctx context.Context, idOrName string, updates map[string]interface{}) (*Profile, error)

	// Delete removes a profile by ID or name. Returns true if a row was deleted.
	Delete(ctx context.Context, idOrName string) (bool, error)

	// GetVersions returns the version history for a profile.
	GetVersions(ctx context.Context, idOrName string) ([]ProfileVersion, error)

	// GetScenarioAndTier retrieves just the scenario name and tier count for a profile.
	GetScenarioAndTier(ctx context.Context, idOrName string) (scenario string, tierCount int, err error)
}

// SQLProfileRepository implements ProfileRepository using a SQL database.
type SQLProfileRepository struct {
	db *sql.DB
}

// NewSQLProfileRepository creates a new SQLProfileRepository.
func NewSQLProfileRepository(db *sql.DB) *SQLProfileRepository {
	return &SQLProfileRepository{db: db}
}

// List returns all profiles ordered by creation date (newest first).
func (r *SQLProfileRepository) List(ctx context.Context) ([]Profile, error) {
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
			continue
		}

		json.Unmarshal(tiersJSON, &p.Tiers)
		json.Unmarshal(swapsJSON, &p.Swaps)
		json.Unmarshal(secretsJSON, &p.Secrets)
		json.Unmarshal(settingsJSON, &p.Settings)

		profiles = append(profiles, p)
	}

	return profiles, nil
}

// Get retrieves a profile by ID or name.
func (r *SQLProfileRepository) Get(ctx context.Context, idOrName string) (*Profile, error) {
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

	json.Unmarshal(tiersJSON, &p.Tiers)
	json.Unmarshal(swapsJSON, &p.Swaps)
	json.Unmarshal(secretsJSON, &p.Secrets)
	json.Unmarshal(settingsJSON, &p.Settings)

	return &p, nil
}

// Create stores a new profile and returns its generated ID.
func (r *SQLProfileRepository) Create(ctx context.Context, profile *Profile) (string, error) {
	tiersJSON, _ := json.Marshal(profile.Tiers)
	swapsJSON, _ := json.Marshal(profile.Swaps)
	secretsJSON, _ := json.Marshal(profile.Secrets)
	settingsJSON, _ := json.Marshal(profile.Settings)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO profiles (id, name, scenario, tiers, swaps, secrets, settings, version, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, 'system', 'system')
	`, profile.ID, profile.Name, profile.Scenario, tiersJSON, swapsJSON, secretsJSON, settingsJSON)

	if err != nil {
		return "", err
	}

	// Create initial version history entry
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO profile_versions (profile_id, version, name, scenario, tiers, swaps, secrets, settings, created_by, change_description)
		VALUES ($1, 1, $2, $3, $4, $5, $6, $7, 'system', 'Initial profile creation')
	`, profile.ID, profile.Name, profile.Scenario, tiersJSON, swapsJSON, secretsJSON, settingsJSON)

	// Don't fail if version history insert fails
	return profile.ID, nil
}

// Update modifies an existing profile and increments its version.
func (r *SQLProfileRepository) Update(ctx context.Context, idOrName string, updates map[string]interface{}) (*Profile, error) {
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
	tiersJSON, _ := json.Marshal(current.Tiers)
	swapsJSON, _ := json.Marshal(current.Swaps)
	secretsJSON, _ := json.Marshal(current.Secrets)
	settingsJSON, _ := json.Marshal(current.Settings)

	_, err = r.db.ExecContext(ctx, `
		UPDATE profiles
		SET tiers = $1, swaps = $2, secrets = $3, settings = $4, version = $5, updated_at = NOW(), updated_by = 'system'
		WHERE id = $6
	`, tiersJSON, swapsJSON, secretsJSON, settingsJSON, newVersion, current.ID)

	if err != nil {
		return nil, err
	}

	// Create version history entry
	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO profile_versions (profile_id, version, name, scenario, tiers, swaps, secrets, settings, created_by, change_description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'system', 'Profile updated')
	`, current.ID, newVersion, current.Name, current.Scenario, tiersJSON, swapsJSON, secretsJSON, settingsJSON)

	current.Version = newVersion
	return current, nil
}

// Delete removes a profile by ID or name. Returns true if a row was deleted.
func (r *SQLProfileRepository) Delete(ctx context.Context, idOrName string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = $1 OR name = $1`, idOrName)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

// GetVersions returns the version history for a profile.
func (r *SQLProfileRepository) GetVersions(ctx context.Context, idOrName string) ([]ProfileVersion, error) {
	// Resolve to actual ID first
	var actualID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE id = $1 OR name = $1`, idOrName).Scan(&actualID)
	if err == sql.ErrNoRows {
		return []ProfileVersion{}, nil
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

	var versions []ProfileVersion
	for rows.Next() {
		var v ProfileVersion
		var tiersJSON, swapsJSON, secretsJSON, settingsJSON []byte

		if err := rows.Scan(&v.ProfileID, &v.Version, &v.Name, &v.Scenario, &tiersJSON, &swapsJSON, &secretsJSON, &settingsJSON, &v.CreatedAt, &v.CreatedBy, &v.ChangeDescription); err != nil {
			continue
		}

		json.Unmarshal(tiersJSON, &v.Tiers)
		json.Unmarshal(swapsJSON, &v.Swaps)
		json.Unmarshal(secretsJSON, &v.Secrets)
		json.Unmarshal(settingsJSON, &v.Settings)

		versions = append(versions, v)
	}

	return versions, nil
}

// GetScenarioAndTier retrieves just the scenario name and tier count for a profile.
// Returns ErrProfileNotFound if the profile does not exist.
func (r *SQLProfileRepository) GetScenarioAndTier(ctx context.Context, idOrName string) (string, int, error) {
	var scenario string
	var tierCount int

	err := r.db.QueryRowContext(ctx, `
		SELECT scenario, COALESCE(jsonb_array_length(tiers), 0)
		FROM profiles
		WHERE name = $1 OR id = $1
	`, idOrName).Scan(&scenario, &tierCount)

	if err == sql.ErrNoRows {
		return "", 0, ErrProfileNotFound
	}
	if err != nil {
		return "", 0, err
	}

	return scenario, tierCount, nil
}
