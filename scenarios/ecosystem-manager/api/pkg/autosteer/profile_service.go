package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ProfileService handles CRUD operations for Auto Steer profiles
type ProfileService struct {
	db *sql.DB
}

// NewProfileService creates a new profile service
func NewProfileService(db *sql.DB) *ProfileService {
	return &ProfileService{
		db: db,
	}
}

// CreateProfile creates a new Auto Steer profile
func (s *ProfileService) CreateProfile(profile *AutoSteerProfile) error {
	// Generate ID if not provided
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	// Validate profile
	if err := s.validateProfile(profile); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	// Marshal config to JSONB
	config, err := s.marshalConfig(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	// Insert into database
	query := `
		INSERT INTO auto_steer_profiles (id, name, description, config, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.db.Exec(query,
		profile.ID,
		profile.Name,
		profile.Description,
		config,
		pq.Array(profile.Tags),
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert profile: %w", err)
	}

	return nil
}

// GetProfile retrieves a profile by ID
func (s *ProfileService) GetProfile(id string) (*AutoSteerProfile, error) {
	query := `
		SELECT id, name, description, config, tags, created_at, updated_at
		FROM auto_steer_profiles
		WHERE id = $1
	`

	var profile AutoSteerProfile
	var configJSON []byte
	var tags []string

	err := s.db.QueryRow(query, id).Scan(
		&profile.ID,
		&profile.Name,
		&profile.Description,
		&configJSON,
		pq.Array(&tags),
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query profile: %w", err)
	}

	profile.Tags = tags

	// Unmarshal config
	if err := s.unmarshalConfig(configJSON, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
	}

	return &profile, nil
}

// ListProfiles retrieves all profiles with optional filtering
func (s *ProfileService) ListProfiles(tags []string) ([]*AutoSteerProfile, error) {
	query := `
		SELECT id, name, description, config, tags, created_at, updated_at
		FROM auto_steer_profiles
	`

	// Add tag filtering if specified
	var args []interface{}
	if len(tags) > 0 {
		query += " WHERE tags && $1"
		args = append(args, pq.Array(tags))
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	// Initialize to empty slice (not nil) so it serializes as [] instead of null
	profiles := make([]*AutoSteerProfile, 0)

	for rows.Next() {
		var profile AutoSteerProfile
		var configJSON []byte
		var profileTags []string

		err := rows.Scan(
			&profile.ID,
			&profile.Name,
			&profile.Description,
			&configJSON,
			pq.Array(&profileTags),
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan profile row: %w", err)
		}

		profile.Tags = profileTags

		// Unmarshal config
		if err := s.unmarshalConfig(configJSON, &profile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
		}

		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profiles: %w", err)
	}

	return profiles, nil
}

// UpdateProfile updates an existing profile
func (s *ProfileService) UpdateProfile(id string, updates *AutoSteerProfile) error {
	// Get existing profile to preserve created_at
	existing, err := s.GetProfile(id)
	if err != nil {
		return err
	}

	// Set ID and timestamps
	updates.ID = id
	updates.CreatedAt = existing.CreatedAt
	updates.UpdatedAt = time.Now()

	// Validate updated profile
	if err := s.validateProfile(updates); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	// Marshal config
	config, err := s.marshalConfig(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	// Update database
	query := `
		UPDATE auto_steer_profiles
		SET name = $1, description = $2, config = $3, tags = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := s.db.Exec(query,
		updates.Name,
		updates.Description,
		config,
		pq.Array(updates.Tags),
		updates.UpdatedAt,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found: %s", id)
	}

	return nil
}

// DeleteProfile deletes a profile by ID
func (s *ProfileService) DeleteProfile(id string) error {
	query := `DELETE FROM auto_steer_profiles WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found: %s", id)
	}

	return nil
}

// GetTemplates returns built-in profile templates
func (s *ProfileService) GetTemplates() []*AutoSteerProfile {
	return GetBuiltInTemplates()
}

// validateProfile validates a profile's structure and data
func (s *ProfileService) validateProfile(profile *AutoSteerProfile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	if len(profile.Phases) == 0 {
		return fmt.Errorf("profile must have at least one phase")
	}

	// Validate each phase
	for i, phase := range profile.Phases {
		if !phase.Mode.IsValid() {
			return fmt.Errorf("phase %d has invalid mode: %s", i, phase.Mode)
		}

		if phase.MaxIterations <= 0 {
			return fmt.Errorf("phase %d must have maxIterations > 0", i)
		}

		if len(phase.StopConditions) == 0 {
			return fmt.Errorf("phase %d must have at least one stop condition", i)
		}

		// Validate stop conditions
		for j, condition := range phase.StopConditions {
			if err := s.validateCondition(condition); err != nil {
				return fmt.Errorf("phase %d, condition %d: %w", i, j, err)
			}
		}
	}

	return nil
}

// validateCondition validates a stop condition
func (s *ProfileService) validateCondition(condition StopCondition) error {
	switch condition.Type {
	case ConditionTypeSimple:
		if condition.Metric == "" {
			return fmt.Errorf("simple condition must have a metric")
		}
		if condition.CompareOperator == "" {
			return fmt.Errorf("simple condition must have a compare operator")
		}

	case ConditionTypeCompound:
		if condition.Operator == "" {
			return fmt.Errorf("compound condition must have a logical operator")
		}
		if len(condition.Conditions) == 0 {
			return fmt.Errorf("compound condition must have sub-conditions")
		}

		// Recursively validate sub-conditions
		for i, subCondition := range condition.Conditions {
			if err := s.validateCondition(subCondition); err != nil {
				return fmt.Errorf("sub-condition %d: %w", i, err)
			}
		}

	default:
		return fmt.Errorf("unknown condition type: %s", condition.Type)
	}

	return nil
}

// marshalConfig marshals a profile to JSON config
func (s *ProfileService) marshalConfig(profile *AutoSteerProfile) ([]byte, error) {
	config := map[string]interface{}{
		"phases":        profile.Phases,
		"quality_gates": profile.QualityGates,
	}

	return json.Marshal(config)
}

// unmarshalConfig unmarshals JSON config into a profile
func (s *ProfileService) unmarshalConfig(configJSON []byte, profile *AutoSteerProfile) error {
	var config struct {
		Phases       []SteerPhase  `json:"phases"`
		QualityGates []QualityGate `json:"quality_gates"`
	}

	if err := json.Unmarshal(configJSON, &config); err != nil {
		return err
	}

	profile.Phases = config.Phases
	profile.QualityGates = config.QualityGates

	return nil
}
