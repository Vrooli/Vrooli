package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-completeness-scoring/pkg/scoring"
)

// Snapshot represents a score snapshot at a point in time
// [REQ:SCS-HIST-001] Score history storage
type Snapshot struct {
	ID             int64                   `json:"id"`
	Scenario       string                  `json:"scenario"`
	Score          int                     `json:"score"`
	Classification string                  `json:"classification"`
	Breakdown      *scoring.ScoreBreakdown `json:"breakdown"`
	ConfigSnapshot map[string]interface{}  `json:"config_snapshot,omitempty"`
	Source         string                  `json:"source,omitempty"`  // e.g., "ecosystem-manager", "cli", "ci"
	Tags           []string                `json:"tags,omitempty"`    // e.g., ["task:abc123", "iteration:5"]
	CreatedAt      time.Time               `json:"created_at"`
}

// SaveOptions contains optional parameters for saving a snapshot
type SaveOptions struct {
	ConfigSnapshot map[string]interface{}
	Source         string   // Source system identifier (e.g., "ecosystem-manager")
	Tags           []string // Arbitrary tags for filtering (e.g., ["task:abc123", "iteration:5"])
}

// HistoryFilter contains filter parameters for querying history
type HistoryFilter struct {
	Source string   // Filter by source (exact match)
	Tags   []string // Filter by tags (all must match)
	Limit  int      // Max results (default 30)
	Since  *time.Time // Only snapshots after this time
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
	return r.SaveWithOptions(scenario, breakdown, SaveOptions{ConfigSnapshot: configSnapshot})
}

// SaveWithOptions stores a new score snapshot with additional options (source, tags)
// [REQ:SCS-HIST-001] Store score snapshots over time
func (r *Repository) SaveWithOptions(scenario string, breakdown *scoring.ScoreBreakdown, opts SaveOptions) (*Snapshot, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	breakdownJSON, err := json.Marshal(breakdown)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal breakdown: %w", err)
	}

	var configJSON []byte
	if opts.ConfigSnapshot != nil {
		configJSON, err = json.Marshal(opts.ConfigSnapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
	}

	var tagsJSON []byte
	if len(opts.Tags) > 0 {
		tagsJSON, err = json.Marshal(opts.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
	}

	query := `
		INSERT INTO score_snapshots (scenario, score, classification, breakdown, config_snapshot, source, tags, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().UTC()

	// Handle nullable fields
	var sourceVal interface{} = nil
	if opts.Source != "" {
		sourceVal = opts.Source
	}
	var tagsVal interface{} = nil
	if len(tagsJSON) > 0 {
		tagsVal = string(tagsJSON)
	}

	result, err := r.db.conn.Exec(query,
		scenario,
		breakdown.Score,
		breakdown.Classification,
		string(breakdownJSON),
		nullableString(configJSON),
		sourceVal,
		tagsVal,
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
		ConfigSnapshot: opts.ConfigSnapshot,
		Source:         opts.Source,
		Tags:           opts.Tags,
		CreatedAt:      now,
	}, nil
}

// nullableString converts byte slice to nullable string for SQL
func nullableString(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// GetLatest returns the most recent snapshot for a scenario
func (r *Repository) GetLatest(scenario string) (*Snapshot, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, scenario, score, classification, breakdown, config_snapshot, source, tags, created_at
		FROM score_snapshots
		WHERE scenario = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanSnapshot(r.db.conn.QueryRow(query, scenario))
}

// GetLatestWithFilter returns the most recent snapshot matching the filter
func (r *Repository) GetLatestWithFilter(scenario string, filter HistoryFilter) (*Snapshot, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query, args := r.buildHistoryQuery(scenario, filter, 1)
	return r.scanSnapshot(r.db.conn.QueryRow(query, args...))
}

// GetHistory returns score history for a scenario
// [REQ:SCS-HIST-004] History API endpoint
func (r *Repository) GetHistory(scenario string, limit int) ([]*Snapshot, error) {
	return r.GetHistoryWithFilter(scenario, HistoryFilter{Limit: limit})
}

// GetHistoryWithFilter returns score history with filtering support
// [REQ:SCS-HIST-004] History API endpoint
func (r *Repository) GetHistoryWithFilter(scenario string, filter HistoryFilter) ([]*Snapshot, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	if filter.Limit <= 0 {
		filter.Limit = 30 // Default to 30 snapshots
	}

	query, args := r.buildHistoryQuery(scenario, filter, filter.Limit)

	rows, err := r.db.conn.Query(query, args...)
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

// buildHistoryQuery constructs the SQL query with filters
func (r *Repository) buildHistoryQuery(scenario string, filter HistoryFilter, limit int) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "scenario = ?")
	args = append(args, scenario)

	if filter.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, filter.Source)
	}

	if filter.Since != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *filter.Since)
	}

	// For tag filtering, we use JSON extraction
	// SQLite JSON functions: json_each() to iterate array elements
	for _, tag := range filter.Tags {
		// Check if the tag exists in the tags JSON array
		// Using INSTR as a simple contains check for the JSON string
		// This works for simple string tags
		conditions = append(conditions, "tags IS NOT NULL AND INSTR(tags, ?) > 0")
		args = append(args, fmt.Sprintf(`"%s"`, tag))
	}

	query := fmt.Sprintf(`
		SELECT id, scenario, score, classification, breakdown, config_snapshot, source, tags, created_at
		FROM score_snapshots
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ?
	`, strings.Join(conditions, " AND "))

	args = append(args, limit)

	return query, args
}

// GetHistorySince returns snapshots since a given time
func (r *Repository) GetHistorySince(scenario string, since time.Time, limit int) ([]*Snapshot, error) {
	return r.GetHistoryWithFilter(scenario, HistoryFilter{
		Since: &since,
		Limit: limit,
	})
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

// CountWithFilter returns the count of snapshots matching a filter
func (r *Repository) CountWithFilter(scenario string, filter HistoryFilter) (int, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	var conditions []string
	var args []interface{}

	conditions = append(conditions, "scenario = ?")
	args = append(args, scenario)

	if filter.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, filter.Source)
	}

	if filter.Since != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *filter.Since)
	}

	for _, tag := range filter.Tags {
		conditions = append(conditions, "tags IS NOT NULL AND INSTR(tags, ?) > 0")
		args = append(args, fmt.Sprintf(`"%s"`, tag))
	}

	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM score_snapshots WHERE %s",
		strings.Join(conditions, " AND "),
	)

	var count int
	err := r.db.conn.QueryRow(query, args...).Scan(&count)
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

// GetDistinctSources returns all unique source values in history
func (r *Repository) GetDistinctSources() ([]string, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `SELECT DISTINCT source FROM score_snapshots WHERE source IS NOT NULL ORDER BY source`
	rows, err := r.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources: %w", err)
	}
	defer rows.Close()

	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources = append(sources, source)
	}

	return sources, nil
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
	var source sql.NullString
	var tagsJSON sql.NullString

	err := row.Scan(
		&snapshot.ID,
		&snapshot.Scenario,
		&snapshot.Score,
		&snapshot.Classification,
		&breakdownJSON,
		&configJSON,
		&source,
		&tagsJSON,
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

	if source.Valid {
		snapshot.Source = source.String
	}

	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &snapshot.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	return &snapshot, nil
}

// scanSnapshotRow scans from sql.Rows into a Snapshot
func (r *Repository) scanSnapshotRow(rows *sql.Rows) (*Snapshot, error) {
	var snapshot Snapshot
	var breakdownJSON string
	var configJSON sql.NullString
	var source sql.NullString
	var tagsJSON sql.NullString

	err := rows.Scan(
		&snapshot.ID,
		&snapshot.Scenario,
		&snapshot.Score,
		&snapshot.Classification,
		&breakdownJSON,
		&configJSON,
		&source,
		&tagsJSON,
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

	if source.Valid {
		snapshot.Source = source.String
	}

	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &snapshot.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	return &snapshot, nil
}
