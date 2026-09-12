package main

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"web-console/internal/dbx"

	"github.com/google/uuid"
)

// ConversationRepository persists semantic conversation history.
type ConversationRepository interface {
	AppendEvent(ctx context.Context, event ConversationEvent) (ConversationEvent, error)
	GetEvent(ctx context.Context, sessionID, eventID string) (ConversationEvent, bool, error)
	ListSession(ctx context.Context, sessionID string) (ConversationSessionState, error)
	ListSessionPage(ctx context.Context, sessionID string, limit int, beforeSequence int64) (ConversationSessionState, bool, error)
	CountSessionEvents(ctx context.Context, sessionID string) (int64, error)
	SessionStorageBytes(ctx context.Context, sessionID string) (int64, error)
	SearchSession(ctx context.Context, sessionID string, query ConversationSearchQuery) (ConversationSearchResult, error)
	SearchArchived(ctx context.Context, filter ArchivedConversationSearchFilter) (ArchivedConversationSearchResult, error)
	ListSessionRange(ctx context.Context, sessionID string, from, to int64) ([]ConversationEvent, error)
	UpdateSpeechParagraphs(ctx context.Context, sessionID, eventID string, paragraphs []string) error
	UpdateCursor(ctx context.Context, sessionID string, patch conversationCursorPatch) (ConversationCursor, error)
	RecordPlaybackStage(ctx context.Context, sessionID, eventID, stage string) error
	DeleteSession(ctx context.Context, sessionID string) error
	CopySession(ctx context.Context, oldID, newID string) error
}

// conversationEventPruner is intentionally additive: callers that do not own
// a SQL event store can continue implementing ConversationRepository without
// also taking on retention policy.
type conversationEventPruner interface {
	PruneEvents(ctx context.Context, before time.Time, maxPerSession int) (int64, error)
}

type ConversationSearchMatch struct {
	EventID   string
	Sequence  int64
	Role      string
	CreatedAt time.Time
	Excerpt   string
	// Ranges locate the matches inside Excerpt (UTF-16 code units).
	Ranges []TextRange
}

// ArchivedConversationSearchFilter is a search over archived conversations:
// the same query as a session search, plus archive-only filters.
type ArchivedConversationSearchFilter struct {
	ConversationSearchQuery
	AgentType    string
	CreatedAfter time.Time
}

type ArchivedConversationSearchMatch struct {
	EventID   string
	SessionID string
	Sequence  int64
	Role      string
	CreatedAt time.Time
	Excerpt   string
	Ranges    []TextRange
}

type SQLConversationRepository struct {
	db dbx.Handle
}

func NewSQLConversationRepository(db dbx.Handle) *SQLConversationRepository {
	return &SQLConversationRepository{db: db}
}

func (r *SQLConversationRepository) AppendEvent(ctx context.Context, event ConversationEvent) (ConversationEvent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ConversationEvent{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := formatTime(event.CreatedAt)
	if now == "" {
		now = formatTime(time.Now())
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_sessions (session_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO NOTHING`,
		event.SessionID, now, now,
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("ensure session: %w", err)
	}

	if err := tx.QueryRowContext(ctx, `
		UPDATE conversation_sessions
		SET last_sequence = last_sequence + 1, updated_at = ?
		WHERE session_id = ?
		RETURNING last_sequence`,
		now, event.SessionID,
	).Scan(&event.Sequence); err != nil {
		return ConversationEvent{}, fmt.Errorf("reserve sequence: %w", err)
	}

	speechJSON, err := marshalStringSlice(event.SpeechParagraphs)
	if err != nil {
		return ConversationEvent{}, fmt.Errorf("marshal speech paragraphs: %w", err)
	}
	var originalJSON any
	if len(event.OriginalSpeechParagraphs) > 0 {
		originalJSON, err = marshalStringSlice(event.OriginalSpeechParagraphs)
		if err != nil {
			return ConversationEvent{}, fmt.Errorf("marshal original speech paragraphs: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_events (
			id, session_id, source, role, text, speech_paragraphs,
			original_speech_paragraphs, summarized, created_at, sequence,
			delivery_state, tts_state, consumption_state
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.SessionID,
		event.Source,
		string(event.Role),
		event.Text,
		speechJSON,
		originalJSON,
		boolToInt(event.Summarized),
		now,
		event.Sequence,
		string(event.DeliveryState),
		string(event.TTSState),
		string(event.ConsumptionState),
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return ConversationEvent{}, fmt.Errorf("commit event: %w", err)
	}
	return event, nil
}

func (r *SQLConversationRepository) GetEvent(ctx context.Context, sessionID, eventID string) (ConversationEvent, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ? AND id = ?`,
		sessionID, eventID,
	)
	event, err := scanConversationEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationEvent{}, false, nil
		}
		return ConversationEvent{}, false, fmt.Errorf("get event: %w", err)
	}
	return event, true, nil
}

func (r *SQLConversationRepository) ListSession(ctx context.Context, sessionID string) (ConversationSessionState, error) {
	state := ConversationSessionState{
		SessionID: sessionID,
		Events:    []ConversationEvent{},
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		sessionID,
	)
	switch err := row.Scan(&state.Cursor.LastSeenSequence, &state.Cursor.LastListenedSequence); err {
	case nil:
	case sql.ErrNoRows:
		return state, nil
	default:
		return ConversationSessionState{}, fmt.Errorf("load cursor: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ?
		ORDER BY sequence`,
		sessionID,
	)
	if err != nil {
		return ConversationSessionState{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		event, err := scanConversationEvent(rows)
		if err != nil {
			return ConversationSessionState{}, err
		}
		state.Events = append(state.Events, event)
	}
	if err := rows.Err(); err != nil {
		return ConversationSessionState{}, fmt.Errorf("iterate events: %w", err)
	}

	return state, nil
}

// ListSessionPage returns one bounded, ascending page ending before
// beforeSequence (or the newest events when beforeSequence is zero).
func (r *SQLConversationRepository) ListSessionPage(ctx context.Context, sessionID string, limit int, beforeSequence int64) (ConversationSessionState, bool, error) {
	if limit <= 0 {
		state, err := r.ListSession(ctx, sessionID)
		return state, false, err
	}
	state := ConversationSessionState{SessionID: sessionID, Events: []ConversationEvent{}}
	if err := r.db.QueryRowContext(ctx, `SELECT last_seen_sequence, last_listened_sequence FROM conversation_sessions WHERE session_id = ?`, sessionID).Scan(&state.Cursor.LastSeenSequence, &state.Cursor.LastListenedSequence); err != nil {
		if err == sql.ErrNoRows {
			return state, false, nil
		}
		return ConversationSessionState{}, false, fmt.Errorf("load cursor: %w", err)
	}
	query := `SELECT id, session_id, source, role, text, speech_paragraphs, COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence, delivery_state, tts_state, consumption_state FROM conversation_events WHERE session_id = ?`
	args := []any{sessionID}
	if beforeSequence > 0 {
		query += ` AND sequence < ?`
		args = append(args, beforeSequence)
	}
	query += ` ORDER BY sequence DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ConversationSessionState{}, false, fmt.Errorf("query event page: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		event, scanErr := scanConversationEvent(rows)
		if scanErr != nil {
			return ConversationSessionState{}, false, scanErr
		}
		state.Events = append(state.Events, event)
	}
	if err := rows.Err(); err != nil {
		return ConversationSessionState{}, false, fmt.Errorf("iterate event page: %w", err)
	}
	hasMore := len(state.Events) > limit
	if hasMore {
		state.Events = state.Events[:limit]
	}
	for left, right := 0, len(state.Events)-1; left < right; left, right = left+1, right-1 {
		state.Events[left], state.Events[right] = state.Events[right], state.Events[left]
	}
	return state, hasMore, nil
}

func (r *SQLConversationRepository) CountSessionEvents(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversation_events WHERE session_id = ?`, sessionID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count session events: %w", err)
	}
	return count, nil
}

func (r *SQLConversationRepository) SessionStorageBytes(ctx context.Context, sessionID string) (int64, error) {
	var size int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(
			length(id) + length(session_id) + length(source) + length(role) + length(text) +
			length(speech_paragraphs) + length(COALESCE(original_speech_paragraphs, '')) + length(created_at)
		), 0)
		FROM conversation_events WHERE session_id = ?`, sessionID).Scan(&size)
	return size, err
}

// PruneEvents removes at most one bounded batch of old events. FTS5 cleanup
// is handled by the conversation_events DELETE trigger, so the index remains
// consistent with the source table. A zero bound disables that bound.
func (r *SQLConversationRepository) PruneEvents(ctx context.Context, before time.Time, maxPerSession int) (int64, error) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if !before.IsZero() {
		conditions = append(conditions, "created_at < ?")
		args = append(args, formatTime(before))
	}
	if maxPerSession > 0 {
		conditions = append(conditions, "rank > ?")
		args = append(args, maxPerSession)
	}
	if len(conditions) == 0 {
		return 0, nil
	}
	query := `DELETE FROM conversation_events
		WHERE rowid IN (
			SELECT rowid FROM (
				SELECT rowid, created_at,
				       ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY sequence DESC) AS rank
				FROM conversation_events
			) ranked
			WHERE ` + strings.Join(conditions, " OR ") + `
			LIMIT 1000
		)`
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("prune conversation events: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count pruned conversation events: %w", err)
	}
	return n, nil
}

func (r *SQLConversationRepository) SearchSession(ctx context.Context, sessionID string, q ConversationSearchQuery) (ConversationSearchResult, error) {
	hits, capped, queryError, err := r.searchEvents(ctx, searchScope{
		where:     []string{"e.session_id = ?"},
		args:      []any{sessionID},
		ftsOrder:  "e.sequence DESC",
		scanOrder: "e.sequence DESC",
	}, q)
	if err != nil || queryError != "" {
		return ConversationSearchResult{Matches: []ConversationSearchMatch{}, Error: queryError}, err
	}
	slices.SortFunc(hits, func(a, b searchHit) int { return cmp.Compare(a.sequence, b.sequence) })
	return sessionSearchResult(hits, capped, searchLimit(q.Limit)), nil
}

// sessionSearchResult keeps the first limit hits in sequence order.
func sessionSearchResult(hits []searchHit, capped bool, limit int) ConversationSearchResult {
	kept := hits[:min(limit, len(hits))]
	result := ConversationSearchResult{
		Matches:   make([]ConversationSearchMatch, 0, len(kept)),
		Truncated: capped || len(hits) > limit,
		Total:     int64(len(hits)),
	}
	for _, hit := range kept {
		createdAt, _ := time.Parse(time.RFC3339Nano, hit.createdAt)
		result.Matches = append(result.Matches, ConversationSearchMatch{
			EventID: hit.id, Sequence: hit.sequence, Role: hit.role, CreatedAt: createdAt, Excerpt: hit.excerpt, Ranges: hit.ranges,
		})
	}
	return result
}

func (r *SQLConversationRepository) SearchArchived(ctx context.Context, filter ArchivedConversationSearchFilter) (ArchivedConversationSearchResult, error) {
	scope := searchScope{
		joins: "LEFT JOIN sessions s ON s.id = e.session_id",
		where: []string{
			// A missing session row is an orphaned projection, not proof that the
			// conversation was deleted. Keep its events searchable so reconciliation
			// can repair metadata after discovery.
			"(s.id IS NULL OR s.archived_at <> '' OR s.status IN ('dismissed', 'awaiting_recovery'))",
			"(s.id IS NULL OR s.recovered_into = '' OR NOT EXISTS (SELECT 1 FROM sessions successor WHERE successor.id = s.recovered_into))",
		},
		ftsOrder:  "bm25(conversation_events_fts), e.created_at DESC",
		scanOrder: "e.created_at DESC",
	}
	if filter.AgentType != "" {
		scope.where = append(scope.where, "s.agent_type = ?")
		scope.args = append(scope.args, filter.AgentType)
	}
	if !filter.CreatedAfter.IsZero() {
		scope.where = append(scope.where, "e.created_at >= ?")
		scope.args = append(scope.args, formatTime(filter.CreatedAfter))
	}
	q := filter.ConversationSearchQuery
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}
	hits, capped, queryError, err := r.searchEvents(ctx, scope, q)
	if err != nil || queryError != "" {
		return ArchivedConversationSearchResult{Matches: []ArchivedConversationSearchMatch{}, Error: queryError}, err
	}
	sessions := map[string]struct{}{}
	for _, hit := range hits {
		sessions[hit.sessionID] = struct{}{}
	}
	kept := hits[:min(q.Limit, len(hits))]
	result := ArchivedConversationSearchResult{
		Matches:          make([]ArchivedConversationSearchMatch, 0, len(kept)),
		Truncated:        capped || len(hits) > q.Limit,
		Total:            int64(len(hits)),
		DistinctSessions: int64(len(sessions)),
	}
	for _, hit := range kept {
		createdAt, _ := time.Parse(time.RFC3339Nano, hit.createdAt)
		result.Matches = append(result.Matches, ArchivedConversationSearchMatch{
			EventID: hit.id, SessionID: hit.sessionID, Sequence: hit.sequence, Role: hit.role,
			CreatedAt: createdAt, Excerpt: hit.excerpt, Ranges: hit.ranges,
		})
	}
	return result, nil
}

func (r *SQLConversationRepository) ListSessionRange(ctx context.Context, sessionID string, from, to int64) ([]ConversationEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, session_id, source, role, text, speech_paragraphs, COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence, delivery_state, tts_state, consumption_state FROM conversation_events WHERE session_id = ? AND sequence >= ? AND sequence <= ? ORDER BY sequence LIMIT 5001`, sessionID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []ConversationEvent{}
	for rows.Next() {
		event, err := scanConversationEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if len(events) > 5000 {
		return nil, fmt.Errorf("conversation range exceeds 5000 events")
	}
	return events, rows.Err()
}

func (r *SQLConversationRepository) UpdateSpeechParagraphs(ctx context.Context, sessionID, eventID string, paragraphs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var currentSpeech string
	var currentOriginal sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT speech_paragraphs, original_speech_paragraphs
		FROM conversation_events
		WHERE session_id = ? AND id = ?`,
		sessionID, eventID,
	).Scan(&currentSpeech, &currentOriginal); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load event speech: %w", err)
	}

	nextSpeech, err := marshalStringSlice(paragraphs)
	if err != nil {
		return fmt.Errorf("marshal summary paragraphs: %w", err)
	}

	originalSpeech := currentSpeech
	if currentOriginal.Valid && currentOriginal.String != "" {
		originalSpeech = currentOriginal.String
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE conversation_events
		SET original_speech_paragraphs = ?, speech_paragraphs = ?, summarized = 1
		WHERE session_id = ? AND id = ?`,
		originalSpeech, nextSpeech, sessionID, eventID,
	); err != nil {
		return fmt.Errorf("update speech paragraphs: %w", err)
	}

	return tx.Commit()
}

func (r *SQLConversationRepository) UpdateCursor(ctx context.Context, sessionID string, patch conversationCursorPatch) (ConversationCursor, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ConversationCursor{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := formatTime(time.Now())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_sessions (session_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO NOTHING`,
		sessionID, now, now,
	); err != nil {
		return ConversationCursor{}, fmt.Errorf("ensure session: %w", err)
	}

	var cursor ConversationCursor
	if err := tx.QueryRowContext(ctx, `
		SELECT last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		sessionID,
	).Scan(&cursor.LastSeenSequence, &cursor.LastListenedSequence); err != nil {
		return ConversationCursor{}, fmt.Errorf("load cursor: %w", err)
	}

	if patch.seenSequence != nil && *patch.seenSequence > cursor.LastSeenSequence {
		cursor.LastSeenSequence = *patch.seenSequence
	}
	if patch.listenedSequence != nil && *patch.listenedSequence > cursor.LastListenedSequence {
		cursor.LastListenedSequence = *patch.listenedSequence
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE conversation_sessions
		SET last_seen_sequence = ?, last_listened_sequence = ?, updated_at = ?
		WHERE session_id = ?`,
		cursor.LastSeenSequence, cursor.LastListenedSequence, now, sessionID,
	); err != nil {
		return ConversationCursor{}, fmt.Errorf("update cursor: %w", err)
	}

	if err := applyCursorStateUpdates(ctx, tx, sessionID, cursor); err != nil {
		return ConversationCursor{}, err
	}

	if err := tx.Commit(); err != nil {
		return ConversationCursor{}, fmt.Errorf("commit cursor: %w", err)
	}
	return cursor, nil
}

func (r *SQLConversationRepository) RecordPlaybackStage(ctx context.Context, sessionID, eventID, stage string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var event ConversationEvent
	found, err := loadConversationEventForUpdate(ctx, tx, sessionID, eventID, &event)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	var cursor ConversationCursor
	if err := tx.QueryRowContext(ctx, `
		SELECT last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		sessionID,
	).Scan(&cursor.LastSeenSequence, &cursor.LastListenedSequence); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load cursor: %w", err)
	}

	switch stage {
	case "received":
		if event.DeliveryState == ConversationDeliveryPending {
			event.DeliveryState = ConversationDeliveryReceived
		}
	case "seen", "correlated":
		event.DeliveryState = ConversationDeliverySeen
		if event.ConsumptionState == ConversationConsumptionUnseen {
			event.ConsumptionState = ConversationConsumptionSeen
		}
	case "playback_started":
		event.TTSState = ConversationTTSPlaying
		event.ConsumptionState = ConversationConsumptionListening
	case "playback_succeeded":
		event.TTSState = ConversationTTSPlayed
		event.DeliveryState = ConversationDeliverySeen
		event.ConsumptionState = ConversationConsumptionListened
		if event.Sequence > cursor.LastSeenSequence {
			cursor.LastSeenSequence = event.Sequence
		}
		if event.Sequence > cursor.LastListenedSequence {
			cursor.LastListenedSequence = event.Sequence
		}
	case "rejected":
		event.TTSState = ConversationTTSRejected
	case "playback_failed":
		event.TTSState = ConversationTTSFailed
	}

	if err := updateConversationEventState(ctx, tx, event); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE conversation_sessions
		SET last_seen_sequence = ?, last_listened_sequence = ?, updated_at = ?
		WHERE session_id = ?`,
		cursor.LastSeenSequence, cursor.LastListenedSequence, formatTime(time.Now()), sessionID,
	); err != nil {
		return fmt.Errorf("update cursor after playback: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit playback stage: %w", err)
	}
	return nil
}

func (r *SQLConversationRepository) DeleteSession(ctx context.Context, sessionID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_events WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete conversation events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_sessions WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete conversation session: %w", err)
	}
	return tx.Commit()
}

// CopySession duplicates the conversation history (cursor + all events) from
// oldID onto newID. Sequence numbers and per-event playback/consumption state
// are preserved verbatim so a recovered pane shows the prior conversation as
// already-seen history (not as fresh, unread messages that would be re-spoken).
// Event ids are regenerated because the id column is a global primary key.
// The destination's last_sequence high-water mark is carried over so freshly
// appended events continue numbering after the copied tail. No-op when the
// source has no history.
func (r *SQLConversationRepository) CopySession(ctx context.Context, oldID, newID string) error {
	if oldID == "" || newID == "" || oldID == newID {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var lastSeq, lastSeen, lastListened int64
	switch err := tx.QueryRowContext(ctx, `
		SELECT last_sequence, last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`,
		oldID,
	).Scan(&lastSeq, &lastSeen, &lastListened); {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("load source session: %w", err)
	}

	now := formatTime(time.Now())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_sessions (
			session_id, last_sequence, last_seen_sequence, last_listened_sequence, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			last_sequence = MAX(conversation_sessions.last_sequence, excluded.last_sequence),
			last_seen_sequence = MAX(conversation_sessions.last_seen_sequence, excluded.last_seen_sequence),
			last_listened_sequence = MAX(conversation_sessions.last_listened_sequence, excluded.last_listened_sequence),
			updated_at = excluded.updated_at`,
		newID, lastSeq, lastSeen, lastListened, now, now,
	); err != nil {
		return fmt.Errorf("ensure destination session: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_events (
			id, session_id, source, role, text, speech_paragraphs,
			original_speech_paragraphs, summarized, created_at, sequence,
			delivery_state, tts_state, consumption_state
		)
		SELECT lower(hex(randomblob(16))), ?, source, role, text, speech_paragraphs,
		       original_speech_paragraphs, summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ?
		ORDER BY sequence`,
		newID, oldID,
	); err != nil {
		return fmt.Errorf("copy events: %w", err)
	}

	return tx.Commit()
}

type InMemoryConversationRepository struct {
	mu       sync.Mutex
	sessions map[string]*conversationSession
}

func NewInMemoryConversationRepository() *InMemoryConversationRepository {
	return &InMemoryConversationRepository{
		sessions: make(map[string]*conversationSession),
	}
}

func (r *InMemoryConversationRepository) AppendEvent(_ context.Context, event ConversationEvent) (ConversationEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.ensureSessionLocked(event.SessionID)
	session.nextSequence++
	event.Sequence = session.nextSequence
	session.events = append(session.events, event)
	return event, nil
}

func (r *InMemoryConversationRepository) GetEvent(_ context.Context, sessionID, eventID string) (ConversationEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return ConversationEvent{}, false, nil
	}
	for i := range session.events {
		if session.events[i].ID == eventID {
			return session.events[i], true, nil
		}
	}
	return ConversationEvent{}, false, nil
}

func (r *InMemoryConversationRepository) ListSession(_ context.Context, sessionID string) (ConversationSessionState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return ConversationSessionState{
			SessionID: sessionID,
			Events:    []ConversationEvent{},
		}, nil
	}

	events := make([]ConversationEvent, len(session.events))
	copy(events, session.events)
	return ConversationSessionState{
		SessionID: sessionID,
		Events:    events,
		Cursor:    session.cursor,
	}, nil
}

func (r *InMemoryConversationRepository) ListSessionPage(ctx context.Context, sessionID string, limit int, beforeSequence int64) (ConversationSessionState, bool, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil || limit <= 0 {
		return state, false, err
	}
	end := len(state.Events)
	if beforeSequence > 0 {
		end = 0
		for index, event := range state.Events {
			if event.Sequence >= beforeSequence {
				end = index
				break
			}
			end = index + 1
		}
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	hasMore := start > 0
	state.Events = append([]ConversationEvent(nil), state.Events[start:end]...)
	return state, hasMore, nil
}

func (r *InMemoryConversationRepository) CountSessionEvents(_ context.Context, sessionID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil {
		return 0, nil
	}
	return int64(len(session.events)), nil
}

func (r *InMemoryConversationRepository) SessionStorageBytes(_ context.Context, sessionID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var size int64
	if session := r.sessions[sessionID]; session != nil {
		for _, event := range session.events {
			size += int64(len(event.ID) + len(event.SessionID) + len(event.Source) + len(string(event.Role)) + len(event.Text) + len(event.CreatedAt.Format(time.RFC3339Nano)))
			for _, paragraph := range event.SpeechParagraphs {
				size += int64(len(paragraph))
			}
			for _, paragraph := range event.OriginalSpeechParagraphs {
				size += int64(len(paragraph))
			}
		}
	}
	return size, nil
}

func (r *InMemoryConversationRepository) SearchSession(ctx context.Context, sessionID string, q ConversationSearchQuery) (ConversationSearchResult, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return ConversationSearchResult{}, err
	}
	plan, queryError := planSearch(q)
	if queryError != "" {
		return ConversationSearchResult{Matches: []ConversationSearchMatch{}, Error: queryError}, nil
	}
	var hits []searchHit
	for _, event := range state.Events {
		if q.Role != "" && string(event.Role) != q.Role {
			continue
		}
		ranges := plan.match(event.Text)
		if len(ranges) == 0 {
			continue
		}
		excerpt, textRanges := searchExcerpt(event.Text, ranges)
		hits = append(hits, searchHit{
			id: event.ID, sessionID: sessionID, sequence: event.Sequence, role: string(event.Role),
			createdAt: formatTime(event.CreatedAt), excerpt: excerpt, ranges: textRanges,
		})
	}
	return sessionSearchResult(hits, false, searchLimit(q.Limit)), nil
}

func (r *InMemoryConversationRepository) SearchArchived(_ context.Context, _ ArchivedConversationSearchFilter) (ArchivedConversationSearchResult, error) {
	return ArchivedConversationSearchResult{Matches: []ArchivedConversationSearchMatch{}}, nil
}

func (r *InMemoryConversationRepository) ListSessionRange(ctx context.Context, sessionID string, from, to int64) ([]ConversationEvent, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	events := []ConversationEvent{}
	for _, event := range state.Events {
		if event.Sequence >= from && event.Sequence <= to {
			events = append(events, event)
			if len(events) > 5000 {
				return nil, fmt.Errorf("conversation range exceeds 5000 events")
			}
		}
	}
	return events, nil
}

func (r *InMemoryConversationRepository) UpdateSpeechParagraphs(_ context.Context, sessionID, eventID string, paragraphs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}
	for i := range session.events {
		if session.events[i].ID == eventID {
			if len(session.events[i].OriginalSpeechParagraphs) == 0 {
				session.events[i].OriginalSpeechParagraphs = session.events[i].SpeechParagraphs
			}
			session.events[i].SpeechParagraphs = paragraphs
			session.events[i].Summarized = true
			return nil
		}
	}
	return nil
}

func (r *InMemoryConversationRepository) UpdateCursor(_ context.Context, sessionID string, patch conversationCursorPatch) (ConversationCursor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.ensureSessionLocked(sessionID)
	if patch.seenSequence != nil && *patch.seenSequence > session.cursor.LastSeenSequence {
		session.cursor.LastSeenSequence = *patch.seenSequence
	}
	if patch.listenedSequence != nil && *patch.listenedSequence > session.cursor.LastListenedSequence {
		session.cursor.LastListenedSequence = *patch.listenedSequence
	}

	for i := range session.events {
		event := &session.events[i]
		if event.Sequence <= session.cursor.LastListenedSequence {
			event.DeliveryState = ConversationDeliverySeen
			event.ConsumptionState = ConversationConsumptionListened
			if event.TTSState == ConversationTTSIdle || event.TTSState == ConversationTTSPlaying {
				event.TTSState = ConversationTTSPlayed
			}
			continue
		}
		if event.Sequence <= session.cursor.LastSeenSequence {
			if event.DeliveryState == ConversationDeliveryPending || event.DeliveryState == ConversationDeliveryReceived {
				event.DeliveryState = ConversationDeliverySeen
			}
			if event.ConsumptionState == ConversationConsumptionUnseen {
				event.ConsumptionState = ConversationConsumptionSeen
			}
		}
	}

	return session.cursor, nil
}

func (r *InMemoryConversationRepository) RecordPlaybackStage(_ context.Context, sessionID, eventID, stage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}
	for i := range session.events {
		event := &session.events[i]
		if event.ID != eventID {
			continue
		}
		switch stage {
		case "received":
			if event.DeliveryState == ConversationDeliveryPending {
				event.DeliveryState = ConversationDeliveryReceived
			}
		case "seen", "correlated":
			event.DeliveryState = ConversationDeliverySeen
			if event.ConsumptionState == ConversationConsumptionUnseen {
				event.ConsumptionState = ConversationConsumptionSeen
			}
		case "playback_started":
			event.TTSState = ConversationTTSPlaying
			event.ConsumptionState = ConversationConsumptionListening
		case "playback_succeeded":
			event.TTSState = ConversationTTSPlayed
			event.DeliveryState = ConversationDeliverySeen
			event.ConsumptionState = ConversationConsumptionListened
			if event.Sequence > session.cursor.LastSeenSequence {
				session.cursor.LastSeenSequence = event.Sequence
			}
			if event.Sequence > session.cursor.LastListenedSequence {
				session.cursor.LastListenedSequence = event.Sequence
			}
		case "rejected":
			event.TTSState = ConversationTTSRejected
		case "playback_failed":
			event.TTSState = ConversationTTSFailed
		}
		return nil
	}
	return nil
}

func (r *InMemoryConversationRepository) DeleteSession(_ context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
	return nil
}

func (r *InMemoryConversationRepository) PruneEvents(_ context.Context, before time.Time, maxPerSession int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var removed int64
	for _, session := range r.sessions {
		kept := session.events[:0]
		for i, event := range session.events {
			oldByAge := !before.IsZero() && !event.CreatedAt.IsZero() && event.CreatedAt.Before(before)
			oldByCount := maxPerSession > 0 && len(session.events)-i > maxPerSession
			if oldByAge || oldByCount {
				removed++
				continue
			}
			kept = append(kept, event)
		}
		session.events = kept
	}
	return removed, nil
}

func (r *InMemoryConversationRepository) CopySession(_ context.Context, oldID, newID string) error {
	if oldID == "" || newID == "" || oldID == newID {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	src, ok := r.sessions[oldID]
	if !ok {
		return nil
	}
	dst := r.ensureSessionLocked(newID)
	for i := range src.events {
		ev := src.events[i]
		ev.ID = newConversationEventID()
		ev.SessionID = newID
		dst.events = append(dst.events, ev)
	}
	if src.nextSequence > dst.nextSequence {
		dst.nextSequence = src.nextSequence
	}
	if src.cursor.LastSeenSequence > dst.cursor.LastSeenSequence {
		dst.cursor.LastSeenSequence = src.cursor.LastSeenSequence
	}
	if src.cursor.LastListenedSequence > dst.cursor.LastListenedSequence {
		dst.cursor.LastListenedSequence = src.cursor.LastListenedSequence
	}
	return nil
}

func (r *InMemoryConversationRepository) ensureSessionLocked(sessionID string) *conversationSession {
	session, ok := r.sessions[sessionID]
	if ok {
		return session
	}
	session = &conversationSession{}
	r.sessions[sessionID] = session
	return session
}

type scannable interface {
	Scan(dest ...any) error
}

func scanConversationEvent(row scannable) (ConversationEvent, error) {
	var event ConversationEvent
	var role, createdAt string
	var speechJSON string
	var originalJSON string
	var summarized int
	var deliveryState, ttsState, consumptionState string
	if err := row.Scan(
		&event.ID,
		&event.SessionID,
		&event.Source,
		&role,
		&event.Text,
		&speechJSON,
		&originalJSON,
		&summarized,
		&createdAt,
		&event.Sequence,
		&deliveryState,
		&ttsState,
		&consumptionState,
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("scan conversation event: %w", err)
	}

	event.Role = ConversationRole(role)
	event.Summarized = summarized == 1
	event.DeliveryState = ConversationDeliveryState(deliveryState)
	event.TTSState = ConversationTTSState(ttsState)
	event.ConsumptionState = ConversationConsumptionState(consumptionState)
	if parsed, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		event.CreatedAt = parsed
	} else if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		event.CreatedAt = parsed
	}
	if err := json.Unmarshal([]byte(speechJSON), &event.SpeechParagraphs); err != nil {
		return ConversationEvent{}, fmt.Errorf("decode speech paragraphs: %w", err)
	}
	if originalJSON != "" {
		if err := json.Unmarshal([]byte(originalJSON), &event.OriginalSpeechParagraphs); err != nil {
			return ConversationEvent{}, fmt.Errorf("decode original speech paragraphs: %w", err)
		}
	}
	return event, nil
}

func marshalStringSlice(values []string) (string, error) {
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func applyCursorStateUpdates(ctx context.Context, tx *sql.Tx, sessionID string, cursor ConversationCursor) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ?
		ORDER BY sequence`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("query events for cursor update: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		event, err := scanConversationEvent(rows)
		if err != nil {
			return err
		}
		original := event
		if event.Sequence <= cursor.LastListenedSequence {
			event.DeliveryState = ConversationDeliverySeen
			event.ConsumptionState = ConversationConsumptionListened
			if event.TTSState == ConversationTTSIdle || event.TTSState == ConversationTTSPlaying {
				event.TTSState = ConversationTTSPlayed
			}
		} else if event.Sequence <= cursor.LastSeenSequence {
			if event.DeliveryState == ConversationDeliveryPending || event.DeliveryState == ConversationDeliveryReceived {
				event.DeliveryState = ConversationDeliverySeen
			}
			if event.ConsumptionState == ConversationConsumptionUnseen {
				event.ConsumptionState = ConversationConsumptionSeen
			}
		}
		if event.DeliveryState != original.DeliveryState || event.TTSState != original.TTSState || event.ConsumptionState != original.ConsumptionState {
			if err := updateConversationEventState(ctx, tx, event); err != nil {
				return err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate cursor update events: %w", err)
	}
	return nil
}

func loadConversationEventForUpdate(ctx context.Context, tx *sql.Tx, sessionID, eventID string, event *ConversationEvent) (bool, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, session_id, source, role, text, speech_paragraphs,
		       COALESCE(original_speech_paragraphs, ''), summarized, created_at, sequence,
		       delivery_state, tts_state, consumption_state
		FROM conversation_events
		WHERE session_id = ? AND id = ?`,
		sessionID, eventID,
	)
	current, err := scanConversationEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	*event = current
	return true, nil
}

func updateConversationEventState(ctx context.Context, tx *sql.Tx, event ConversationEvent) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE conversation_events
		SET delivery_state = ?, tts_state = ?, consumption_state = ?
		WHERE session_id = ? AND id = ?`,
		string(event.DeliveryState),
		string(event.TTSState),
		string(event.ConsumptionState),
		event.SessionID,
		event.ID,
	); err != nil {
		return fmt.Errorf("update event state: %w", err)
	}
	return nil
}

func newConversationEventID() string {
	return uuid.New().String()
}
