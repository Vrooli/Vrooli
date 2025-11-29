package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"scenario-completeness-scoring/pkg/scoring"
)

// Snapshot represents a score snapshot at a point in time
// [REQ:SCS-HIST-001] Score history storage
type Snapshot struct {
	ID             int64                  `json:"id"`
	Scenario       string                 `json:"scenario"`
	Score          int                    `json:"score"`
	Classification string                 `json:"classification"`
	Breakdown      *scoring.ScoreBreakdown `json:"breakdown"`
	ConfigSnapshot map[string]interface{} `json:"config_snapshot,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// Repository provides CRUD operations for score snapshots
// [REQ:SCS-HIST-001] Score history storage
type Repository struct {
	db *DB
}

// NewRepository creates a new snapshot repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// Save stores a new score snapshot
// [REQ:SCS-HIST-001] Store score snapshots over time
func (r *Repository) Save(scenario string, breakdown *scoring.ScoreBreakdown, configSnapshot map[string]interface{}) (*Snapshot, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	breakdownJSON, err := json.Marshal(breakdown)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal breakdown: %w", err)
	}

	var configJSON []byte
	if configSnapshot != nil {
		configJSON, err = json.Marshal(configSnapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
	}

	query := `
		INSERT INTO score_snapshots (scenario, score, classification, breakdown, config_snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now().UTC()
	result, err := r.db.conn.Exec(query,
		scenario,
		breakdown.Score,
		breakdown.Classification,
		string(breakdownJSON),
		string(configJSON),
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert snapshot: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert ID: %w", err)
	}

	return &Snapshot{
		ID:             id,
		Scenario:       scenario,
		Score:          breakdown.Score,
		Classification: breakdown.Classification,
		Breakdown:      breakdown,
		ConfigSnapshot: configSnapshot,
		CreatedAt:      now,
	}, nil
}

// GetLatest returns the most recent snapshot for a scenario
func (r *Repository) GetLatest(scenario string) (*Snapshot, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, scenario, score, classification, breakdown, config_snapshot, created_at
		FROM score_snapshots
		WHERE scenario = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanSnapshot(r.db.conn.QueryRow(query, scenario))
}

// GetHistory returns score history for a scenario
// [REQ:SCS-HIST-004] History API endpoint
func (r *Repository) GetHistory(scenario string, limit int) ([]*Snapshot, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	if limit <= 0 {
		limit = 30 // Default to 30 snapshots
	}

	query := `
		SELECT id, scenario, score, classification, breakdown, config_snapshot, created_at
		FROM score_snapshots
		WHERE scenario = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.conn.Query(query, scenario, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var snapshots []*Snapshot
	for rows.Next() {
		snapshot, err := r.scanSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return snapshots, nil
}

// GetHistorySince returns snapshots since a given time
func (r *Repository) GetHistorySince(scenario string, since time.Time, limit int) ([]*Snapshot, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, scenario, score, classification, breakdown, config_snapshot, created_at
		FROM score_snapshots
		WHERE scenario = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.conn.Query(query, scenario, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var snapshots []*Snapshot
	for rows.Next() {
		snapshot, err := r.scanSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// Count returns the total number of snapshots for a scenario
func (r *Repository) Count(scenario string) (int, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	var count int
	err := r.db.conn.QueryRow(
		"SELECT COUNT(*) FROM score_snapshots WHERE scenario = ?",
		scenario,
	).Scan(&count)
	return count, err
}

// CountAll returns the total number of snapshots across all scenarios
func (r *Repository) CountAll() (int, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	var count int
	err := r.db.conn.QueryRow("SELECT COUNT(*) FROM score_snapshots").Scan(&count)
	return count, err
}

// GetAllScenarios returns a list of all scenarios with history
func (r *Repository) GetAllScenarios() ([]string, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `SELECT DISTINCT scenario FROM score_snapshots ORDER BY scenario`
	rows, err := r.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scenarios: %w", err)
	}
	defer rows.Close()

	var scenarios []string
	for rows.Next() {
		var scenario string
		if err := rows.Scan(&scenario); err != nil {
			return nil, fmt.Errorf("failed to scan scenario: %w", err)
		}
		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// Delete removes old snapshots for a scenario, keeping the most recent N
func (r *Repository) Prune(scenario string, keepCount int) (int64, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		DELETE FROM score_snapshots
		WHERE scenario = ? AND id NOT IN (
			SELECT id FROM score_snapshots
			WHERE scenario = ?
			ORDER BY created_at DESC
			LIMIT ?
		)
	`

	result, err := r.db.conn.Exec(query, scenario, scenario, keepCount)
	if err != nil {
		return 0, fmt.Errorf("failed to prune snapshots: %w", err)
	}

	return result.RowsAffected()
}

// scanSnapshot scans a single row into a Snapshot
func (r *Repository) scanSnapshot(row *sql.Row) (*Snapshot, error) {
	var snapshot Snapshot
	var breakdownJSON string
	var configJSON sql.NullString

	err := row.Scan(
		&snapshot.ID,
		&snapshot.Scenario,
		&snapshot.Score,
		&snapshot.Classification,
		&breakdownJSON,
		&configJSON,
		&snapshot.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan snapshot: %w", err)
	}

	if err := json.Unmarshal([]byte(breakdownJSON), &snapshot.Breakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal breakdown: %w", err)
	}

	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &snapshot.ConfigSnapshot); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return &snapshot, nil
}

// scanSnapshotRow scans from sql.Rows into a Snapshot
func (r *Repository) scanSnapshotRow(rows *sql.Rows) (*Snapshot, error) {
	var snapshot Snapshot
	var breakdownJSON string
	var configJSON sql.NullString

	err := rows.Scan(
		&snapshot.ID,
		&snapshot.Scenario,
		&snapshot.Score,
		&snapshot.Classification,
		&breakdownJSON,
		&configJSON,
		&snapshot.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan snapshot: %w", err)
	}

	if err := json.Unmarshal([]byte(breakdownJSON), &snapshot.Breakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal breakdown: %w", err)
	}

	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &snapshot.ConfigSnapshot); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return &snapshot, nil
}
