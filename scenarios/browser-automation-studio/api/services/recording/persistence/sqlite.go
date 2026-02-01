// Package persistence provides data access for the unified recording service.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/domain"
)

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db  *sql.DB
	log *logrus.Logger
}

// NewSQLiteRepository creates a new SQLite-backed repository.
func NewSQLiteRepository(db *sql.DB, log *logrus.Logger) *SQLiteRepository {
	return &SQLiteRepository{
		db:  db,
		log: log,
	}
}

// CreateSession persists a new recording session.
func (r *SQLiteRepository) CreateSession(ctx context.Context, session *domain.RecordingSession) error {
	query := `
		INSERT INTO recording_sessions (id, profile_id, status, viewport_width, viewport_height, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID,
		nilIfEmpty(session.ProfileID),
		session.Status,
		session.ViewportWidth,
		session.ViewportHeight,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by ID.
func (r *SQLiteRepository) GetSession(ctx context.Context, sessionID string) (*domain.RecordingSession, error) {
	query := `
		SELECT id, profile_id, status, viewport_width, viewport_height, created_at, closed_at
		FROM recording_sessions
		WHERE id = $1
	`
	var session domain.RecordingSession
	var profileID sql.NullString
	var closedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&profileID,
		&session.Status,
		&session.ViewportWidth,
		&session.ViewportHeight,
		&session.CreatedAt,
		&closedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if profileID.Valid {
		session.ProfileID = profileID.String
	}
	if closedAt.Valid {
		session.ClosedAt = &closedAt.Time
	}

	// Get action count
	countQuery := `SELECT COUNT(*) FROM timeline_entries WHERE session_id = $1`
	if err := r.db.QueryRowContext(ctx, countQuery, sessionID).Scan(&session.ActionCount); err != nil {
		r.log.WithError(err).Warn("Failed to count session actions")
	}

	return &session, nil
}

// CloseSession marks a session as closed.
func (r *SQLiteRepository) CloseSession(ctx context.Context, sessionID string, closedAt time.Time) error {
	query := `UPDATE recording_sessions SET status = $1, closed_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, domain.SessionStatusClosed, closedAt, sessionID)
	if err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	return nil
}

// ListSessions returns sessions with optional filtering.
func (r *SQLiteRepository) ListSessions(ctx context.Context, profileID *string, limit, offset int) ([]*domain.RecordingSession, error) {
	var query string
	var args []interface{}

	if profileID != nil {
		query = `
			SELECT id, profile_id, status, viewport_width, viewport_height, created_at, closed_at
			FROM recording_sessions
			WHERE profile_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{*profileID, limit, offset}
	} else {
		query = `
			SELECT id, profile_id, status, viewport_width, viewport_height, created_at, closed_at
			FROM recording_sessions
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.RecordingSession
	for rows.Next() {
		var session domain.RecordingSession
		var profileIDVal sql.NullString
		var closedAt sql.NullTime

		if err := rows.Scan(
			&session.ID,
			&profileIDVal,
			&session.Status,
			&session.ViewportWidth,
			&session.ViewportHeight,
			&session.CreatedAt,
			&closedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		if profileIDVal.Valid {
			session.ProfileID = profileIDVal.String
		}
		if closedAt.Valid {
			session.ClosedAt = &closedAt.Time
		}

		sessions = append(sessions, &session)
	}

	return sessions, rows.Err()
}

// DeleteSession removes a session and all its entries.
func (r *SQLiteRepository) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete entries first (foreign key)
	if _, err := r.db.ExecContext(ctx, `DELETE FROM timeline_entries WHERE session_id = $1`, sessionID); err != nil {
		return fmt.Errorf("delete session entries: %w", err)
	}

	// Delete session
	if _, err := r.db.ExecContext(ctx, `DELETE FROM recording_sessions WHERE id = $1`, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// SaveTimelineEntry persists a single timeline entry.
func (r *SQLiteRepository) SaveTimelineEntry(ctx context.Context, entry *UnifiedTimelineEntry) error {
	// Serialize action or page event to JSON
	var actionJSON, pageEventJSON sql.NullString

	if entry.Action != nil {
		data, err := json.Marshal(entry.Action)
		if err != nil {
			return fmt.Errorf("marshal action: %w", err)
		}
		actionJSON = sql.NullString{String: string(data), Valid: true}
	}

	if entry.PageEvent != nil {
		data, err := json.Marshal(entry.PageEvent)
		if err != nil {
			return fmt.Errorf("marshal page event: %w", err)
		}
		pageEventJSON = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO timeline_entries (id, type, timestamp, session_id, page_id, sequence, action_json, page_event_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		entry.ID.String(),
		entry.Type,
		entry.Timestamp,
		entry.SessionID,
		entry.PageID.String(),
		entry.Sequence,
		actionJSON,
		pageEventJSON,
	)
	if err != nil {
		return fmt.Errorf("save timeline entry: %w", err)
	}

	return nil
}

// SaveTimelineEntries persists multiple entries in a batch.
func (r *SQLiteRepository) SaveTimelineEntries(ctx context.Context, entries []*UnifiedTimelineEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO timeline_entries (id, type, timestamp, session_id, page_id, sequence, action_json, page_event_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		var actionJSON, pageEventJSON sql.NullString

		if entry.Action != nil {
			data, err := json.Marshal(entry.Action)
			if err != nil {
				return fmt.Errorf("marshal action: %w", err)
			}
			actionJSON = sql.NullString{String: string(data), Valid: true}
		}

		if entry.PageEvent != nil {
			data, err := json.Marshal(entry.PageEvent)
			if err != nil {
				return fmt.Errorf("marshal page event: %w", err)
			}
			pageEventJSON = sql.NullString{String: string(data), Valid: true}
		}

		_, err = stmt.ExecContext(ctx,
			entry.ID.String(),
			entry.Type,
			entry.Timestamp,
			entry.SessionID,
			entry.PageID.String(),
			entry.Sequence,
			actionJSON,
			pageEventJSON,
		)
		if err != nil {
			return fmt.Errorf("insert entry: %w", err)
		}
	}

	return tx.Commit()
}

// GetTimelineEntry retrieves a single entry by ID.
func (r *SQLiteRepository) GetTimelineEntry(ctx context.Context, entryID uuid.UUID) (*UnifiedTimelineEntry, error) {
	query := `
		SELECT id, type, timestamp, session_id, page_id, sequence, action_json, page_event_json
		FROM timeline_entries
		WHERE id = $1
	`

	var entry UnifiedTimelineEntry
	var idStr, pageIDStr string
	var entryType string
	var actionJSON, pageEventJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, entryID.String()).Scan(
		&idStr,
		&entryType,
		&entry.Timestamp,
		&entry.SessionID,
		&pageIDStr,
		&entry.Sequence,
		&actionJSON,
		&pageEventJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get timeline entry: %w", err)
	}

	entry.ID, _ = uuid.Parse(idStr)
	entry.PageID, _ = uuid.Parse(pageIDStr)
	entry.Type = TimelineEntryType(entryType)

	if actionJSON.Valid {
		var action domain.RecordingAction
		if err := json.Unmarshal([]byte(actionJSON.String), &action); err != nil {
			r.log.WithError(err).Warn("Failed to unmarshal action JSON")
		} else {
			entry.Action = &action
		}
	}

	if pageEventJSON.Valid {
		var event domain.PageEvent
		if err := json.Unmarshal([]byte(pageEventJSON.String), &event); err != nil {
			r.log.WithError(err).Warn("Failed to unmarshal page event JSON")
		} else {
			entry.PageEvent = &event
		}
	}

	return &entry, nil
}

// GetTimeline returns timeline entries matching the query.
func (r *SQLiteRepository) GetTimeline(ctx context.Context, query TimelineQuery) (*TimelineResponse, error) {
	query.ApplyDefaults()

	// Build query with PostgreSQL numbered placeholders
	baseQuery := `
		SELECT id, type, timestamp, session_id, page_id, sequence, action_json, page_event_json
		FROM timeline_entries
		WHERE session_id = $1
	`
	args := []interface{}{query.SessionID}
	paramNum := 2 // Next placeholder number

	if query.PageID != nil {
		baseQuery += fmt.Sprintf(` AND page_id = $%d`, paramNum)
		args = append(args, query.PageID.String())
		paramNum++
	}

	if query.Since != nil {
		baseQuery += fmt.Sprintf(` AND timestamp > $%d`, paramNum)
		args = append(args, *query.Since)
		paramNum++
	}

	if len(query.EntryTypes) > 0 {
		baseQuery += ` AND type IN (`
		for i, t := range query.EntryTypes {
			if i > 0 {
				baseQuery += ","
			}
			baseQuery += fmt.Sprintf("$%d", paramNum)
			args = append(args, t)
			paramNum++
		}
		baseQuery += `)`
	}

	baseQuery += fmt.Sprintf(` ORDER BY sequence ASC LIMIT $%d OFFSET $%d`, paramNum, paramNum+1)
	args = append(args, query.Limit+1, query.Offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query timeline: %w", err)
	}
	defer rows.Close()

	var entries []UnifiedTimelineEntry
	for rows.Next() {
		var entry UnifiedTimelineEntry
		var idStr, pageIDStr string
		var entryType string
		var actionJSON, pageEventJSON sql.NullString

		if err := rows.Scan(
			&idStr,
			&entryType,
			&entry.Timestamp,
			&entry.SessionID,
			&pageIDStr,
			&entry.Sequence,
			&actionJSON,
			&pageEventJSON,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		entry.ID, _ = uuid.Parse(idStr)
		entry.PageID, _ = uuid.Parse(pageIDStr)
		entry.Type = TimelineEntryType(entryType)

		if actionJSON.Valid {
			var action domain.RecordingAction
			if err := json.Unmarshal([]byte(actionJSON.String), &action); err == nil {
				entry.Action = &action
			}
		}

		if pageEventJSON.Valid {
			var event domain.PageEvent
			if err := json.Unmarshal([]byte(pageEventJSON.String), &event); err == nil {
				entry.PageEvent = &event
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	// Check for more entries
	hasMore := len(entries) > query.Limit
	if hasMore {
		entries = entries[:query.Limit]
	}

	// Get total count
	total, err := r.CountTimelineEntries(ctx, query.SessionID)
	if err != nil {
		r.log.WithError(err).Warn("Failed to count timeline entries")
		total = len(entries)
	}

	return &TimelineResponse{
		Entries:    entries,
		HasMore:    hasMore,
		TotalCount: total,
	}, nil
}

// CountTimelineEntries returns the total entry count for a session.
func (r *SQLiteRepository) CountTimelineEntries(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM timeline_entries WHERE session_id = $1`, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count entries: %w", err)
	}
	return count, nil
}

// DeleteSessionEntries removes all timeline entries for a session.
func (r *SQLiteRepository) DeleteSessionEntries(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM timeline_entries WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("delete session entries: %w", err)
	}
	return nil
}

// PruneOldSessions removes sessions older than the given time.
func (r *SQLiteRepository) PruneOldSessions(ctx context.Context, olderThan time.Time) (int, error) {
	// Get session IDs to delete
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM recording_sessions WHERE created_at < $1`, olderThan)
	if err != nil {
		return 0, fmt.Errorf("query old sessions: %w", err)
	}

	var sessionIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan session id: %w", err)
		}
		sessionIDs = append(sessionIDs, id)
	}
	rows.Close()

	if len(sessionIDs) == 0 {
		return 0, nil
	}

	// Delete entries and sessions in a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, id := range sessionIDs {
		if _, err := tx.ExecContext(ctx, `DELETE FROM timeline_entries WHERE session_id = $1`, id); err != nil {
			return 0, fmt.Errorf("delete entries for session %s: %w", id, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM recording_sessions WHERE id = $1`, id); err != nil {
			return 0, fmt.Errorf("delete session %s: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return len(sessionIDs), nil
}

// nilIfEmpty returns nil for empty strings, otherwise the string pointer.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Compile-time interface compliance
var _ Repository = (*SQLiteRepository)(nil)
