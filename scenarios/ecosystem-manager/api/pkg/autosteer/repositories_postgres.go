package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PostgresProfileRepository implements ProfileRepository using PostgreSQL.
type PostgresProfileRepository struct {
	db *sql.DB
}

// NewPostgresProfileRepository creates a new PostgreSQL profile repository.
func NewPostgresProfileRepository(db *sql.DB) *PostgresProfileRepository {
	return &PostgresProfileRepository{db: db}
}

// GetByID retrieves a profile by ID.
func (r *PostgresProfileRepository) GetByID(id string) (*AutoSteerProfile, error) {
	query := `
		SELECT id, name, description, config, tags, created_at, updated_at
		FROM auto_steer_profiles
		WHERE id = $1
	`

	var profile AutoSteerProfile
	var configJSON []byte
	var tags []string

	err := r.db.QueryRow(query, id).Scan(
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

	if err := unmarshalProfileConfig(configJSON, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
	}

	return &profile, nil
}

// Create inserts a new profile.
func (r *PostgresProfileRepository) Create(profile *AutoSteerProfile) error {
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	config, err := marshalProfileConfig(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	query := `
		INSERT INTO auto_steer_profiles (id, name, description, config, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.Exec(query,
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

// Update updates an existing profile.
func (r *PostgresProfileRepository) Update(id string, profile *AutoSteerProfile) error {
	profile.UpdatedAt = time.Now()

	config, err := marshalProfileConfig(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	query := `
		UPDATE auto_steer_profiles
		SET name = $1, description = $2, config = $3, tags = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := r.db.Exec(query,
		profile.Name,
		profile.Description,
		config,
		pq.Array(profile.Tags),
		profile.UpdatedAt,
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

// Delete removes a profile by ID.
func (r *PostgresProfileRepository) Delete(id string) error {
	query := `DELETE FROM auto_steer_profiles WHERE id = $1`

	result, err := r.db.Exec(query, id)
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

// List retrieves all profiles with optional tag filtering.
func (r *PostgresProfileRepository) List(tags []string) ([]*AutoSteerProfile, error) {
	query := `
		SELECT id, name, description, config, tags, created_at, updated_at
		FROM auto_steer_profiles
	`

	var args []interface{}
	if len(tags) > 0 {
		query += " WHERE tags && $1"
		args = append(args, pq.Array(tags))
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

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

		if err := unmarshalProfileConfig(configJSON, &profile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
		}

		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profiles: %w", err)
	}

	return profiles, nil
}

// PostgresExecutionStateRepository implements ExecutionStateRepository using PostgreSQL.
type PostgresExecutionStateRepository struct {
	db *sql.DB
}

// NewPostgresExecutionStateRepository creates a new PostgreSQL execution state repository.
func NewPostgresExecutionStateRepository(db *sql.DB) *PostgresExecutionStateRepository {
	return &PostgresExecutionStateRepository{db: db}
}

// Get retrieves the execution state for a task.
func (r *PostgresExecutionStateRepository) Get(taskID string) (*ProfileExecutionState, error) {
	query := `
		SELECT task_id, profile_id, current_phase_index, current_phase_iteration,
		       auto_steer_iteration, phase_started_at, phase_history, metrics, phase_start_metrics, started_at, last_updated
		FROM profile_execution_state
		WHERE task_id = $1
	`

	var state ProfileExecutionState
	var phaseHistoryJSON, metricsJSON, phaseStartMetricsJSON []byte
	var phaseStartedAt sql.NullTime

	err := r.db.QueryRow(query, taskID).Scan(
		&state.TaskID,
		&state.ProfileID,
		&state.CurrentPhaseIndex,
		&state.CurrentPhaseIteration,
		&state.AutoSteerIteration,
		&phaseStartedAt,
		&phaseHistoryJSON,
		&metricsJSON,
		&phaseStartMetricsJSON,
		&state.StartedAt,
		&state.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query execution state: %w", err)
	}

	if err := json.Unmarshal(phaseHistoryJSON, &state.PhaseHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phase history: %w", err)
	}

	if err := json.Unmarshal(metricsJSON, &state.Metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	if err := json.Unmarshal(phaseStartMetricsJSON, &state.PhaseStartMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phase start metrics: %w", err)
	}

	if phaseStartedAt.Valid {
		state.PhaseStartedAt = phaseStartedAt.Time
	} else {
		state.PhaseStartedAt = state.StartedAt
	}

	return &state, nil
}

// Save persists the execution state (upsert).
func (r *PostgresExecutionStateRepository) Save(state *ProfileExecutionState) error {
	phaseHistoryJSON, err := json.Marshal(state.PhaseHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal phase history: %w", err)
	}

	metricsJSON, err := json.Marshal(state.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	phaseStartMetricsJSON, err := json.Marshal(state.PhaseStartMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal phase start metrics: %w", err)
	}

	query := `
		INSERT INTO profile_execution_state (
			task_id, profile_id, current_phase_index, current_phase_iteration, auto_steer_iteration, phase_started_at,
			phase_history, metrics, phase_start_metrics, started_at, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (task_id) DO UPDATE SET
			profile_id = EXCLUDED.profile_id,
			current_phase_index = EXCLUDED.current_phase_index,
			current_phase_iteration = EXCLUDED.current_phase_iteration,
			auto_steer_iteration = EXCLUDED.auto_steer_iteration,
			phase_started_at = EXCLUDED.phase_started_at,
			phase_history = EXCLUDED.phase_history,
			metrics = EXCLUDED.metrics,
			phase_start_metrics = EXCLUDED.phase_start_metrics,
			last_updated = EXCLUDED.last_updated
	`

	_, err = r.db.Exec(query,
		state.TaskID,
		state.ProfileID,
		state.CurrentPhaseIndex,
		state.CurrentPhaseIteration,
		state.AutoSteerIteration,
		state.PhaseStartedAt,
		phaseHistoryJSON,
		metricsJSON,
		phaseStartMetricsJSON,
		state.StartedAt,
		state.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to save execution state: %w", err)
	}

	return nil
}

// Delete removes the execution state for a task.
func (r *PostgresExecutionStateRepository) Delete(taskID string) error {
	query := `DELETE FROM profile_execution_state WHERE task_id = $1`
	if _, err := r.db.Exec(query, taskID); err != nil {
		return fmt.Errorf("failed to delete execution state for task %s: %w", taskID, err)
	}
	return nil
}

// Helper functions for config marshaling

func marshalProfileConfig(profile *AutoSteerProfile) ([]byte, error) {
	config := map[string]interface{}{
		"phases":        profile.Phases,
		"quality_gates": profile.QualityGates,
	}
	return json.Marshal(config)
}

func unmarshalProfileConfig(configJSON []byte, profile *AutoSteerProfile) error {
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
