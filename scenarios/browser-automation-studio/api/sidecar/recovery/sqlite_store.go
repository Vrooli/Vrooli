package recovery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// SQLiteStore is a Store implementation using SQLite.
type SQLiteStore struct {
	db  *sqlx.DB
	log *logrus.Logger
}

// checkpointRow represents a row in the session_checkpoints table.
type checkpointRow struct {
	ID            string    `db:"id"`
	SessionID     string    `db:"session_id"`
	WorkflowID    *string   `db:"workflow_id"`
	Actions       string    `db:"actions"`       // JSON array
	CurrentURL    string    `db:"current_url"`
	BrowserConfig string    `db:"browser_config"` // JSON object
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// NewSQLiteStore creates a new SQLiteStore.
func NewSQLiteStore(db *sqlx.DB, log *logrus.Logger) *SQLiteStore {
	return &SQLiteStore{
		db:  db,
		log: log,
	}
}

// Migrate creates the session_checkpoints table if it doesn't exist.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS session_checkpoints (
			id TEXT PRIMARY KEY,
			session_id TEXT UNIQUE NOT NULL,
			workflow_id TEXT,
			actions TEXT NOT NULL DEFAULT '[]',
			current_url TEXT NOT NULL DEFAULT '',
			browser_config TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON session_checkpoints(session_id);
		CREATE INDEX IF NOT EXISTS idx_checkpoints_updated ON session_checkpoints(updated_at);
	`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create session_checkpoints table: %w", err)
	}

	s.log.Info("Session checkpoints table migrated")
	return nil
}

// Save saves or updates a checkpoint.
func (s *SQLiteStore) Save(ctx context.Context, checkpoint *SessionCheckpoint) error {
	// Ensure ID is set
	if checkpoint.ID == "" {
		checkpoint.ID = uuid.New().String()
	}

	// Serialize actions
	actionsJSON, err := json.Marshal(checkpoint.RecordedActions)
	if err != nil {
		return fmt.Errorf("failed to marshal actions: %w", err)
	}

	// Serialize browser config
	configJSON, err := json.Marshal(checkpoint.BrowserConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal browser config: %w", err)
	}

	now := time.Now()
	if checkpoint.CreatedAt.IsZero() {
		checkpoint.CreatedAt = now
	}
	checkpoint.UpdatedAt = now

	// Upsert
	query := `
		INSERT INTO session_checkpoints (id, session_id, workflow_id, actions, current_url, browser_config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			actions = excluded.actions,
			current_url = excluded.current_url,
			browser_config = excluded.browser_config,
			updated_at = excluded.updated_at
	`

	var workflowID *string
	if checkpoint.WorkflowID != "" {
		workflowID = &checkpoint.WorkflowID
	}

	_, err = s.db.ExecContext(ctx, query,
		checkpoint.ID,
		checkpoint.SessionID,
		workflowID,
		string(actionsJSON),
		checkpoint.CurrentURL,
		string(configJSON),
		checkpoint.CreatedAt,
		checkpoint.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Get retrieves a checkpoint by session ID.
func (s *SQLiteStore) Get(ctx context.Context, sessionID string) (*SessionCheckpoint, error) {
	query := `
		SELECT id, session_id, workflow_id, actions, current_url, browser_config, created_at, updated_at
		FROM session_checkpoints
		WHERE session_id = ?
	`

	var row checkpointRow
	err := s.db.GetContext(ctx, &row, query, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	return s.rowToCheckpoint(&row)
}

// Delete removes a checkpoint by session ID.
func (s *SQLiteStore) Delete(ctx context.Context, sessionID string) error {
	query := `DELETE FROM session_checkpoints WHERE session_id = ?`

	_, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	return nil
}

// ListActive returns all checkpoints.
func (s *SQLiteStore) ListActive(ctx context.Context) ([]*SessionCheckpoint, error) {
	query := `
		SELECT id, session_id, workflow_id, actions, current_url, browser_config, created_at, updated_at
		FROM session_checkpoints
		ORDER BY updated_at DESC
	`

	var rows []checkpointRow
	err := s.db.SelectContext(ctx, &rows, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	result := make([]*SessionCheckpoint, 0, len(rows))
	for _, row := range rows {
		cp, err := s.rowToCheckpoint(&row)
		if err != nil {
			s.log.WithError(err).Warn("Failed to parse checkpoint row, skipping")
			continue
		}
		result = append(result, cp)
	}

	return result, nil
}

// Cleanup removes checkpoints older than the given duration.
func (s *SQLiteStore) Cleanup(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)

	query := `DELETE FROM session_checkpoints WHERE updated_at < ?`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup checkpoints: %w", err)
	}

	count, _ := result.RowsAffected()
	return int(count), nil
}

// rowToCheckpoint converts a database row to a SessionCheckpoint.
func (s *SQLiteStore) rowToCheckpoint(row *checkpointRow) (*SessionCheckpoint, error) {
	cp := &SessionCheckpoint{
		ID:         row.ID,
		SessionID:  row.SessionID,
		CurrentURL: row.CurrentURL,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}

	if row.WorkflowID != nil {
		cp.WorkflowID = *row.WorkflowID
	}

	// Parse actions
	if err := json.Unmarshal([]byte(row.Actions), &cp.RecordedActions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal actions: %w", err)
	}

	// Parse browser config
	if err := json.Unmarshal([]byte(row.BrowserConfig), &cp.BrowserConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal browser config: %w", err)
	}

	return cp, nil
}

// compile-time check
var _ Store = (*SQLiteStore)(nil)
