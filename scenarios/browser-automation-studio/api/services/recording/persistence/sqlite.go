// Package persistence provides data access for the unified recording service.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/browser-automation-studio/domain"
)

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db sqlExecutor
}

// NewSQLiteRepository creates a new SQLite-backed repository.
// sqlExecutor is the smallest persistence surface this domain needs. Both a
// standard *sql.DB and api-core's context-routed database satisfy it, keeping
// unit tests simple while production requests honor the Test Genie lease.
type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

var (
	_ sqlExecutor = (*sql.DB)(nil)
	_ sqlExecutor = (*coredb.RoutedDB)(nil)
)

func NewSQLiteRepository(db sqlExecutor) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
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
		return nil, fmt.Errorf("count session entries: %w", err)
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

// AppendTimelineEntry assigns order in the same SQLite write statement as the
// insert. SQLite serializes writers, including writers in other API processes.
func (r *SQLiteRepository) AppendTimelineEntry(ctx context.Context, entry *UnifiedTimelineEntry) (bool, error) {
	if entry == nil || entry.ID == uuid.Nil || entry.SessionID == "" {
		return false, fmt.Errorf("journal entry requires identity and session")
	}
	var actionJSON, pageEventJSON any
	if entry.Action != nil {
		data, err := json.Marshal(entry.Action)
		if err != nil {
			return false, fmt.Errorf("marshal action: %w", err)
		}
		actionJSON = string(data)
	}
	if entry.PageEvent != nil {
		data, err := json.Marshal(entry.PageEvent)
		if err != nil {
			return false, fmt.Errorf("marshal page event: %w", err)
		}
		pageEventJSON = string(data)
	}
	const query = `INSERT INTO timeline_entries (id,type,timestamp,session_id,page_id,sequence,action_json,page_event_json)
 SELECT $1,$2,$3,$4,$5,COALESCE(MAX(sequence),0)+1,$6,$7 FROM timeline_entries WHERE session_id=$4
 ON CONFLICT(id) DO NOTHING RETURNING sequence`
	err := r.db.QueryRowContext(ctx, query, entry.ID.String(), entry.Type, entry.Timestamp, entry.SessionID, entry.PageID.String(), actionJSON, pageEventJSON).Scan(&entry.Sequence)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, fmt.Errorf("append timeline entry: %w", err)
	}
	committed, err := r.GetTimelineEntry(ctx, entry.ID)
	if err != nil {
		return false, err
	}
	if committed == nil {
		return false, fmt.Errorf("journal retry lost its committed identity")
	}
	if !sameObservation(entry, committed) {
		return false, fmt.Errorf("journal identity %s conflicts with a committed observation", entry.ID)
	}
	entry.Sequence = committed.Sequence
	return false, nil
}

// Local commit metadata can differ on retry; observed fields may not.
func sameObservation(a, b *UnifiedTimelineEntry) bool {
	canonical := func(e *UnifiedTimelineEntry) []byte {
		copyEntry := *e
		copyEntry.Sequence = 0
		if e.Action != nil {
			copyAction := *e.Action
			copyAction.CreatedAt = time.Time{}
			copyEntry.Action = &copyAction
		}
		data, err := json.Marshal(copyEntry)
		if err != nil {
			return nil
		}
		return data
	}
	left, right := canonical(a), canonical(b)
	return left != nil && string(left) == string(right)
}

// GetTimelineEntry retrieves a single entry by ID.
func (r *SQLiteRepository) GetTimelineEntry(ctx context.Context, entryID uuid.UUID) (*UnifiedTimelineEntry, error) {
	query := `
		SELECT id, type, timestamp, session_id, page_id, sequence, action_json, page_event_json
		FROM timeline_entries
		WHERE id = $1
	`

	entry, err := scanTimelineEntry(r.db.QueryRowContext(ctx, query, entryID.String()))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return entry, err
}

// Both single-entry retries and paginated reads reject corrupt committed data.
func scanTimelineEntry(row interface{ Scan(...any) error }) (*UnifiedTimelineEntry, error) {
	var entry UnifiedTimelineEntry
	var id, page string
	var action, event sql.NullString
	if err := row.Scan(&id, &entry.Type, &entry.Timestamp, &entry.SessionID, &page, &entry.Sequence, &action, &event); err != nil {
		return nil, err
	}
	var err error
	if entry.ID, err = uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("journal identity: %w", err)
	}
	if entry.PageID, err = uuid.Parse(page); err != nil {
		return nil, fmt.Errorf("journal page identity: %w", err)
	}
	if action.Valid {
		if err := decodeJournalJSON(action.String, &entry.Action); err != nil {
			return nil, fmt.Errorf("committed action %s: %w", id, err)
		}
	}
	if event.Valid {
		if err := decodeJournalJSON(event.String, &entry.PageEvent); err != nil {
			return nil, fmt.Errorf("committed page event %s: %w", id, err)
		}
	}
	return &entry, nil
}

// Every observation uses the same whole-document and exact-number policy.
func decodeJournalJSON(data string, value any) error {
	if !json.Valid([]byte(data)) || strings.TrimSpace(data) == "null" {
		return fmt.Errorf("invalid committed JSON document")
	}
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(value)
}

// GetTimeline returns timeline entries matching the query.
func (r *SQLiteRepository) GetTimeline(ctx context.Context, query TimelineQuery) (*TimelineResponse, error) {
	query.ApplyDefaults()

	// Build the filtered journal query with SQLite numbered placeholders
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

	baseQuery += fmt.Sprintf(` ORDER BY sequence ASC, id ASC LIMIT $%d OFFSET $%d`, paramNum, paramNum+1)
	args = append(args, query.Limit+1, query.Offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query timeline: %w", err)
	}
	defer rows.Close()

	var entries []UnifiedTimelineEntry
	for rows.Next() {
		entry, err := scanTimelineEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close timeline rows: %w", err)
	}

	// Check for more entries
	hasMore := len(entries) > query.Limit
	if hasMore {
		entries = entries[:query.Limit]
	}

	// Get total count
	total, err := r.CountTimelineEntries(ctx, query.SessionID)
	if err != nil {
		return nil, fmt.Errorf("count timeline entries: %w", err)
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
	defer func() { _ = tx.Rollback() }()

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
