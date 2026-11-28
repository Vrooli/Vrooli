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

	"web-console/internal/continuity"
	"web-console/internal/dbx"

	"github.com/google/uuid"
)

// ConversationRepository persists semantic conversation history.
type ConversationRepository interface {
	AppendEvent(ctx context.Context, event ConversationEvent) (ConversationEvent, error)
	GetEvent(ctx context.Context, sessionID, eventID string) (ConversationEvent, bool, error)
	FindNativeEvent(ctx context.Context, sessionID string, provenance *NativeProvenance) (ConversationEvent, bool, error)
	ListSession(ctx context.Context, sessionID string) (ConversationSessionState, error)
	ListSessionPage(ctx context.Context, sessionID string, query ConversationPageQuery) (ConversationSessionState, bool, error)
	CountSessionEvents(ctx context.Context, sessionID string) (int64, error)
	SessionStorageBytes(ctx context.Context, sessionID string) (int64, error)
	SearchSession(ctx context.Context, sessionID string, query ConversationSearchQuery) (ConversationSearchResult, error)
	SearchArchived(ctx context.Context, filter ArchivedConversationSearchFilter) (ArchivedConversationSearchResult, error)
	ListSessionRange(ctx context.Context, sessionID string, from, to int64) ([]ConversationEvent, error)
	TruncateSessionAfter(ctx context.Context, sessionID string, sequence int64) error
	UpdateSpeechParagraphs(ctx context.Context, sessionID, eventID string, paragraphs []string) error
	UpdateCursor(ctx context.Context, sessionID string, patch conversationCursorPatch) (ConversationCursor, error)
	RecordPlaybackStage(ctx context.Context, sessionID, eventID, stage string) error
	DeleteSession(ctx context.Context, sessionID string) error
	CopySession(ctx context.Context, oldID, newID string) error
	// SessionCursor reads the seen/listened cursor without loading events.
	SessionCursor(ctx context.Context, sessionID string) (ConversationCursor, error)
	// LatestEventCreatedAt is the ingestion time of the session's newest
	// event; found is false for a session with no events.
	LatestEventCreatedAt(ctx context.Context, sessionID string) (createdAt time.Time, found bool, err error)
}

// ConversationPageQuery bounds one read of a session's events. Every bound is
// applied in SQL, so a poll or a page reads only the rows it returns.
type ConversationPageQuery struct {
	// Limit keeps the newest Limit events of the window; zero returns the
	// whole window.
	Limit int
	// BeforeSequence, when positive, keeps events with a lower sequence.
	BeforeSequence int64
	// SinceSequence, when positive, keeps events with a higher sequence.
	SinceSequence int64
}

// conversationEventPruner is intentionally additive: callers that do not own
// a SQL event store can continue implementing ConversationRepository without
// also taking on retention policy.
type conversationEventPruner interface {
	PruneEvents(ctx context.Context, before time.Time, maxPerSession int) (int64, error)
}

type conversationSessionPrefixLister interface {
	ListSessionIDsByPrefix(ctx context.Context, prefix string) ([]string, error)
}

type ConversationSearchMatch struct {
	EventID    string
	Sequence   int64
	Role       string
	Kind       string
	ThreadID   string
	CreatedAt  time.Time
	OccurredAt time.Time
	Excerpt    string
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
	EventID    string
	SessionID  string
	Sequence   int64
	Role       string
	CreatedAt  time.Time
	OccurredAt time.Time
	Excerpt    string
	Ranges     []TextRange
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

	if err := ensureProjection(ctx, tx, event.SessionID, now, event.NativeProvenance); err != nil {
		return ConversationEvent{}, err
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
			original_speech_paragraphs, summarized, created_at, occurred_at, sequence,
			delivery_state, tts_state, consumption_state, native_provenance_json,
			kind, author, author_name, label, thread_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.SessionID,
		event.Source,
		string(event.Role),
		event.Text,
		speechJSON,
		originalJSON,
		boolToInt(event.Summarized),
		now,
		nullableConversationTime(event.OccurredAt),
		event.Sequence,
		string(event.DeliveryState),
		string(event.TTSState),
		string(event.ConsumptionState),
		nativeProvenanceJSON(event.NativeProvenance),
		storedConversationKind(event.Kind),
		event.Author,
		event.AuthorName,
		event.Label,
		event.ThreadID,
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return ConversationEvent{}, fmt.Errorf("commit event: %w", err)
	}
	return event, nil
}

// ensureProjection creates the session's conversation projection when it is
// absent and, in the same transaction, catalogs a new projection that has no
// live session row.
func ensureProjection(ctx context.Context, tx *sql.Tx, sessionID, now string, provenance *NativeProvenance) error {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_sessions (session_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO NOTHING`,
		sessionID, now, now,
	)
	if err != nil {
		return fmt.Errorf("ensure session: %w", err)
	}
	if created, err := result.RowsAffected(); err != nil || created == 0 {
		return err
	}
	return catalogNewProjection(ctx, tx, sessionID, now, provenance)
}

// catalogNewProjection gives a projection without a live session row (a
// native archive import, typically) its continuity catalog entry as it is
// created, so continuity health never finds it uncataloged and the
// missing-catalog repair stays a recovery tool. The record comes from the
// planner's own builder and is written under the repair's own predicate; a
// projection whose session row exists is left to that session.
//
// The record claims no agent-session alias. An archive import often mirrors a
// conversation a Web Console pane already owns under that native id, and a
// second claim would be an alias collision needing review.
func catalogNewProjection(ctx context.Context, tx *sql.Tx, sessionID, now string, provenance *NativeProvenance) error {
	at, _ := time.Parse(time.RFC3339Nano, now)
	evidence := continuity.Evidence{SessionID: sessionID, HasConversation: true, CreatedAt: at, LastActivityAt: at}
	if provenance != nil {
		evidence.AgentType = provenance.Provider
		evidence.HasNativeHistory = true
		if provenance.Provider != "" && provenance.SessionID != "" {
			evidence.OriginalTitle = provenance.Provider + " conversation " + provenance.SessionID[:min(8, len(provenance.SessionID))]
		}
	}
	record, err := continuity.BuildCatalogRecord(evidence)
	if err != nil {
		return fmt.Errorf("catalog new projection: %w", err)
	}
	if _, err := continuity.CatalogMissingConversation(ctx, tx, record); err != nil {
		return fmt.Errorf("catalog new projection: %w", err)
	}
	return nil
}

func (r *SQLConversationRepository) UpdateNativeEventOccurredAt(ctx context.Context, sessionID string, provenance *NativeProvenance, occurredAt time.Time) (bool, error) {
	if provenance == nil || occurredAt.IsZero() {
		return false, nil
	}
	result, err := r.db.ExecContext(ctx, `UPDATE conversation_events SET occurred_at = ? WHERE session_id = ? AND native_provenance_json = ?`, formatOccurredAt(occurredAt), sessionID, nativeProvenanceJSON(provenance))
	if err != nil {
		return false, fmt.Errorf("update native occurrence timestamp: %w", err)
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

func (r *SQLConversationRepository) GetEvent(ctx context.Context, sessionID, eventID string) (ConversationEvent, bool, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+conversationEventColumns+`
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

const findNativeEventByIdentityQuery = `
	SELECT ` + conversationEventColumns + `
	FROM conversation_events INDEXED BY idx_conversation_events_native_identity
	WHERE session_id = ? AND native_provenance_json = ?
	LIMIT 1`

func (r *SQLConversationRepository) FindNativeEvent(ctx context.Context, sessionID string, provenance *NativeProvenance) (ConversationEvent, bool, error) {
	if provenance == nil {
		return ConversationEvent{}, false, nil
	}
	event, err := scanConversationEvent(r.db.QueryRowContext(ctx, findNativeEventByIdentityQuery, sessionID, nativeProvenanceJSON(provenance)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationEvent{}, false, nil
		}
		return ConversationEvent{}, false, fmt.Errorf("find native conversation event: %w", err)
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
		SELECT `+conversationEventColumns+`
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

// ListSessionPage returns one ascending page of the window the query bounds:
// the newest Limit events after SinceSequence and before BeforeSequence, or
// the whole window when Limit is zero. hasMore reports that the window holds
// older events than the page. Both bounds and the limit are SQL predicates on
// the (session_id, sequence) index, so a poll for "anything newer" costs one
// index probe however long the session is.
func (r *SQLConversationRepository) ListSessionPage(ctx context.Context, sessionID string, page ConversationPageQuery) (ConversationSessionState, bool, error) {
	state := ConversationSessionState{SessionID: sessionID, Events: []ConversationEvent{}}
	cursor, found, err := r.sessionCursor(ctx, sessionID)
	if err != nil || !found {
		return state, false, err
	}
	state.Cursor = cursor
	query := `SELECT ` + conversationEventColumns + ` FROM conversation_events WHERE session_id = ?`
	args := []any{sessionID}
	if page.BeforeSequence > 0 {
		query += ` AND sequence < ?`
		args = append(args, page.BeforeSequence)
	}
	if page.SinceSequence > 0 {
		query += ` AND sequence > ?`
		args = append(args, page.SinceSequence)
	}
	if page.Limit > 0 {
		query += ` ORDER BY sequence DESC LIMIT ?`
		args = append(args, page.Limit+1)
	} else {
		query += ` ORDER BY sequence`
	}
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
	if page.Limit <= 0 {
		return state, false, nil
	}
	hasMore := len(state.Events) > page.Limit
	if hasMore {
		state.Events = state.Events[:page.Limit]
	}
	slices.Reverse(state.Events)
	return state, hasMore, nil
}

// SessionCursor reads the cursor alone; an unknown session has a zero cursor.
func (r *SQLConversationRepository) SessionCursor(ctx context.Context, sessionID string) (ConversationCursor, error) {
	cursor, _, err := r.sessionCursor(ctx, sessionID)
	return cursor, err
}

func (r *SQLConversationRepository) sessionCursor(ctx context.Context, sessionID string) (ConversationCursor, bool, error) {
	var cursor ConversationCursor
	err := r.db.QueryRowContext(ctx, `SELECT last_seen_sequence, last_listened_sequence FROM conversation_sessions WHERE session_id = ?`, sessionID).Scan(&cursor.LastSeenSequence, &cursor.LastListenedSequence)
	switch {
	case err == nil:
		return cursor, true, nil
	case errors.Is(err, sql.ErrNoRows):
		return ConversationCursor{}, false, nil
	default:
		return ConversationCursor{}, false, fmt.Errorf("load cursor: %w", err)
	}
}

// LatestEventCreatedAt reads one row: the session's highest sequence. Events
// are stamped with their ingestion time in the transaction that reserves their
// sequence, and copied history keeps its older times below the copy's
// sequences, so the highest sequence carries the latest ingestion time.
func (r *SQLConversationRepository) LatestEventCreatedAt(ctx context.Context, sessionID string) (time.Time, bool, error) {
	var createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT created_at FROM conversation_events WHERE session_id = ? ORDER BY sequence DESC LIMIT 1`, sessionID).Scan(&createdAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return time.Time{}, false, nil
	case err != nil:
		return time.Time{}, false, fmt.Errorf("latest event time: %w", err)
	}
	return parseConversationTime(createdAt), true, nil
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
			length(speech_paragraphs) + length(COALESCE(original_speech_paragraphs, '')) + length(created_at) + length(COALESCE(occurred_at, ''))
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
	where := []string{"e.session_id = ?"}
	args := []any{sessionID}
	order := "DESC"
	if q.AfterSequence > 0 {
		where = append(where, "e.sequence > ?")
		args = append(args, q.AfterSequence)
		order = "ASC"
	}
	hits, capped, queryError, err := r.searchEvents(ctx, searchScope{
		where:     where,
		args:      args,
		ftsOrder:  "e.sequence " + order,
		scanOrder: "e.sequence " + order,
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
		HasMore:   capped || len(hits) > limit,
	}
	for _, hit := range kept {
		createdAt, _ := time.Parse(time.RFC3339Nano, hit.createdAt)
		result.Matches = append(result.Matches, ConversationSearchMatch{
			EventID: hit.id, Sequence: hit.sequence, Role: hit.role, Kind: hit.kind, ThreadID: hit.threadID, CreatedAt: createdAt, OccurredAt: parseConversationTime(hit.occurredAt), Excerpt: hit.excerpt, Ranges: hit.ranges,
		})
	}
	if len(result.Matches) > 0 && result.HasMore {
		result.NextAfterSequence = result.Matches[len(result.Matches)-1].Sequence
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
		// RFC3339Nano omits trailing fractional zeroes, so text order misplaces
		// values such as .12 and .123. Parse the chosen source/ingest timestamp
		// for chronological ranking; native range filters retain their indexable
		// fixed-width predicate below.
		ftsOrder:  "bm25(conversation_events_fts), julianday(COALESCE(e.occurred_at, e.created_at)) DESC, e.session_id DESC, e.sequence DESC, e.id DESC",
		scanOrder: "julianday(COALESCE(e.occurred_at, e.created_at)) DESC, e.session_id DESC, e.sequence DESC, e.id DESC",
		ftsDriven: true,
	}
	if filter.AgentType != "" {
		if source := nativeArchiveSourceForAgentType(filter.AgentType); source != "" {
			scope.where = append(scope.where, "(s.agent_type = ? OR e.source = ?)")
			scope.args = append(scope.args, filter.AgentType, source)
		} else {
			scope.where = append(scope.where, "s.agent_type = ?")
			scope.args = append(scope.args, filter.AgentType)
		}
	}
	if !filter.CreatedAfter.IsZero() {
		// Native occurred_at is persisted at fixed-width UTC precision and remains
		// indexable. Legacy/non-native created_at values are variable-width RFC3339Nano,
		// so use SQLite's time parser to avoid fractional-second lexical misordering.
		scope.where = append(scope.where, `((e.source IN ('claude_archive','codex_archive','grok_archive','opencode_archive','gemini-cli_archive','qwen-code_archive','cursor_archive') AND e.occurred_at >= ?) OR (e.source NOT IN ('claude_archive','codex_archive','grok_archive','opencode_archive','gemini-cli_archive','qwen-code_archive','cursor_archive') AND julianday(e.created_at) >= julianday(?)))`)
		scope.args = append(scope.args, formatOccurredAt(filter.CreatedAfter), formatTime(filter.CreatedAfter))
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
			CreatedAt: createdAt, OccurredAt: parseConversationTime(hit.occurredAt), Excerpt: hit.excerpt, Ranges: hit.ranges,
		})
	}
	return result, nil
}

func nativeArchiveSourceForAgentType(agentType string) string {
	return continuity.NativeArchiveSourceForAgentType(agentType)
}

func (r *SQLConversationRepository) ListSessionRange(ctx context.Context, sessionID string, from, to int64) ([]ConversationEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+conversationEventColumns+` FROM conversation_events WHERE session_id = ? AND sequence >= ? AND sequence <= ? ORDER BY sequence LIMIT 5001`, sessionID, from, to)
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

func (r *SQLConversationRepository) TruncateSessionAfter(ctx context.Context, sessionID string, sequence int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM conversation_events WHERE session_id = ? AND sequence > ?`, sessionID, sequence)
	return err
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
	if err := ensureProjection(ctx, tx, sessionID, now, nil); err != nil {
		return ConversationCursor{}, err
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_threads WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete conversation threads: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_sessions WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete conversation session: %w", err)
	}
	return tx.Commit()
}

func (r *SQLConversationRepository) ListSessionIDsByPrefix(ctx context.Context, prefix string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT session_id FROM conversation_events WHERE substr(session_id, 1, ?) = ? ORDER BY session_id`, len(prefix), prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CopySession duplicates the conversation history (cursor + all events) from
// oldID onto newID. Source sequence numbers and per-event playback/consumption
// state are preserved verbatim so a recovered pane shows the prior conversation
// as already-seen history (not as fresh, unread messages that would be re-spoken).
// Events emitted by the replacement pane before this copy are shifted after that
// history in the same transaction, avoiding duplicate (session_id, sequence)
// values without dropping the pane's early output.
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

	var destLastSeq, destLastSeen, destLastListened int64
	destSessionExists := true
	switch err := tx.QueryRowContext(ctx, `
		SELECT last_sequence, last_seen_sequence, last_listened_sequence
		FROM conversation_sessions
		WHERE session_id = ?`, newID,
	).Scan(&destLastSeq, &destLastSeen, &destLastListened); {
	case errors.Is(err, sql.ErrNoRows):
		destSessionExists = false
	case err != nil:
		return fmt.Errorf("load destination session: %w", err)
	}
	// Move any replacement-pane output out of the source history's sequence
	// range before inserting copied rows. The offset is above both high-water
	// marks, so a UNIQUE(session_id, sequence) constraint cannot collide while
	// the existing rows move.
	offset := lastSeq + destLastSeq + 1
	if destLastSeq > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE conversation_events
			SET sequence = sequence + ?
			WHERE session_id = ?`, offset, newID); err != nil {
			return fmt.Errorf("reserve destination event sequence range: %w", err)
		}
	}

	now := formatTime(time.Now())
	destSeen := lastSeen
	destListened := lastListened
	if destSessionExists {
		destSeen = maxInt64(destSeen, lastSeq+destLastSeen)
		destListened = maxInt64(destListened, lastSeq+destLastListened)
	}
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
	if !destSessionExists {
		if err := catalogNewProjection(ctx, tx, newID, now, nil); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_events (
			id, session_id, source, role, text, speech_paragraphs,
			original_speech_paragraphs, summarized, created_at, occurred_at, sequence,
			delivery_state, tts_state, consumption_state, native_provenance_json,
			kind, author, author_name, label, thread_id
		)
		SELECT lower(hex(randomblob(16))), ?, source, role, text, speech_paragraphs,
		       original_speech_paragraphs, summarized, created_at, occurred_at, sequence,
		       delivery_state, tts_state, consumption_state, native_provenance_json,
		       kind, author, author_name, label, thread_id
		FROM conversation_events
		WHERE session_id = ?
		ORDER BY sequence`,
		newID, oldID,
	); err != nil {
		return fmt.Errorf("copy events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO conversation_threads (session_id, thread_id, kind, label, description, parent_thread_id, created_at)
		SELECT ?, thread_id, kind, label, description, parent_thread_id, created_at
		FROM conversation_threads WHERE session_id = ?`, newID, oldID); err != nil {
		return fmt.Errorf("copy threads: %w", err)
	}
	if destLastSeq > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE conversation_events
			SET sequence = sequence - ? + ?
			WHERE session_id = ? AND sequence > ?`, offset, lastSeq, newID, offset); err != nil {
			return fmt.Errorf("place replacement events after copied history: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE conversation_sessions
		SET last_sequence = MAX(last_sequence, ?),
		    last_seen_sequence = MAX(last_seen_sequence, ?),
		    last_listened_sequence = MAX(last_listened_sequence, ?),
		    updated_at = ?
		WHERE session_id = ?`,
		lastSeq+destLastSeq,
		destSeen,
		destListened,
		now, newID,
	); err != nil {
		return fmt.Errorf("update destination conversation cursor: %w", err)
	}

	return tx.Commit()
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

type InMemoryConversationRepository struct {
	mu       sync.Mutex
	sessions map[string]*conversationSession
	threads  map[string]map[string]ConversationThread
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

func (r *InMemoryConversationRepository) FindNativeEvent(_ context.Context, sessionID string, provenance *NativeProvenance) (ConversationEvent, bool, error) {
	if provenance == nil {
		return ConversationEvent{}, false, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[sessionID]
	if !ok {
		return ConversationEvent{}, false, nil
	}
	identity := nativeIdentity(provenance)
	for _, event := range session.events {
		if nativeIdentity(event.NativeProvenance) == identity {
			return event, true, nil
		}
	}
	return ConversationEvent{}, false, nil
}

func (r *InMemoryConversationRepository) UpdateNativeEventOccurredAt(_ context.Context, sessionID string, provenance *NativeProvenance, occurredAt time.Time) (bool, error) {
	if provenance == nil || occurredAt.IsZero() {
		return false, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[sessionID]
	if !ok {
		return false, nil
	}
	identity := nativeIdentity(provenance)
	for index := range session.events {
		if nativeIdentity(session.events[index].NativeProvenance) == identity {
			session.events[index].OccurredAt = occurredAt.UTC()
			return true, nil
		}
	}
	return false, nil
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

func (r *InMemoryConversationRepository) ListSessionPage(ctx context.Context, sessionID string, page ConversationPageQuery) (ConversationSessionState, bool, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return state, false, err
	}
	window := make([]ConversationEvent, 0, len(state.Events))
	for _, event := range state.Events {
		if (page.BeforeSequence <= 0 || event.Sequence < page.BeforeSequence) && (page.SinceSequence <= 0 || event.Sequence > page.SinceSequence) {
			window = append(window, event)
		}
	}
	hasMore := page.Limit > 0 && len(window) > page.Limit
	if hasMore {
		window = window[len(window)-page.Limit:]
	}
	state.Events = window
	return state, hasMore, nil
}

func (r *InMemoryConversationRepository) SessionCursor(_ context.Context, sessionID string) (ConversationCursor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session := r.sessions[sessionID]; session != nil {
		return session.cursor, nil
	}
	return ConversationCursor{}, nil
}

func (r *InMemoryConversationRepository) LatestEventCreatedAt(_ context.Context, sessionID string) (time.Time, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil || len(session.events) == 0 {
		return time.Time{}, false, nil
	}
	return session.events[len(session.events)-1].CreatedAt, true, nil
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
		if q.AfterSequence > 0 && event.Sequence <= q.AfterSequence {
			continue
		}
		if q.Role != "" && string(event.Role) != q.Role {
			continue
		}
		if !searchKindAllowed(q.Kinds, event.Kind) {
			continue
		}
		ranges := plan.match(event.Text)
		if len(ranges) == 0 {
			continue
		}
		excerpt, textRanges := searchExcerpt(event.Text, ranges)
		hits = append(hits, searchHit{
			id: event.ID, sessionID: sessionID, sequence: event.Sequence, role: string(event.Role), kind: event.Kind, threadID: event.ThreadID,
			createdAt: formatTime(event.CreatedAt), occurredAt: formatOptionalOccurredAt(event.OccurredAt), excerpt: excerpt, ranges: textRanges,
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

func (r *InMemoryConversationRepository) TruncateSessionAfter(_ context.Context, sessionID string, sequence int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}
	kept := session.events[:0]
	for _, event := range session.events {
		if event.Sequence <= sequence {
			kept = append(kept, event)
		}
	}
	session.events = kept
	return nil
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

func (r *InMemoryConversationRepository) ListSessionIDsByPrefix(_ context.Context, prefix string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := []string{}
	for id, session := range r.sessions {
		if !strings.HasPrefix(id, prefix) || len(session.events) == 0 {
			continue
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
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
	destLastSeq := dst.nextSequence
	for i := range dst.events {
		dst.events[i].Sequence += src.nextSequence
	}
	history := make([]ConversationEvent, 0, len(src.events)+len(dst.events))
	for i := range src.events {
		ev := src.events[i]
		ev.ID = newConversationEventID()
		ev.SessionID = newID
		history = append(history, ev)
	}
	history = append(history, dst.events...)
	dst.events = history
	dst.nextSequence = src.nextSequence + destLastSeq
	if seen := src.nextSequence + dst.cursor.LastSeenSequence; seen > src.cursor.LastSeenSequence {
		dst.cursor.LastSeenSequence = seen
	}
	if listened := src.nextSequence + dst.cursor.LastListenedSequence; listened > src.cursor.LastListenedSequence {
		dst.cursor.LastListenedSequence = listened
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

// conversationEventColumns is the one column list scanConversationEvent reads.
const conversationEventColumns = `id, session_id, source, role, text, speech_paragraphs,
	COALESCE(original_speech_paragraphs, ''), summarized, created_at, occurred_at, sequence,
	delivery_state, tts_state, consumption_state, native_provenance_json,
	kind, author, author_name, label, thread_id`

// storedConversationKind keeps the column's default for rows appended
// without a classification (tests and legacy fixtures).
func storedConversationKind(kind string) string {
	if kind == "" {
		return "message"
	}
	return kind
}

type scannable interface {
	Scan(dest ...any) error
}

func nullableConversationTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return formatOccurredAt(value)
}

// formatOccurredAt keeps a fixed-width UTC representation so SQLite's
// lexicographic range comparison remains chronological across whole-second
// and fractional-second timestamps.
func formatOccurredAt(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func parseConversationTime(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return time.Time{}
}

func scanConversationEvent(row scannable) (ConversationEvent, error) {
	var event ConversationEvent
	var role, createdAt string
	var occurredAt sql.NullString
	var speechJSON string
	var originalJSON string
	var summarized int
	var deliveryState, ttsState, consumptionState string
	var nativeJSON sql.NullString
	var kind string
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
		&occurredAt,
		&event.Sequence,
		&deliveryState,
		&ttsState,
		&consumptionState,
		&nativeJSON,
		&kind,
		&event.Author,
		&event.AuthorName,
		&event.Label,
		&event.ThreadID,
	); err != nil {
		return ConversationEvent{}, fmt.Errorf("scan conversation event: %w", err)
	}
	event.Kind = kind

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
	event.OccurredAt = parseConversationTime(occurredAt.String)
	if err := json.Unmarshal([]byte(speechJSON), &event.SpeechParagraphs); err != nil {
		return ConversationEvent{}, fmt.Errorf("decode speech paragraphs: %w", err)
	}
	if originalJSON != "" {
		if err := json.Unmarshal([]byte(originalJSON), &event.OriginalSpeechParagraphs); err != nil {
			return ConversationEvent{}, fmt.Errorf("decode original speech paragraphs: %w", err)
		}
	}
	if nativeJSON.Valid && nativeJSON.String != "" {
		var provenance NativeProvenance
		if err := json.Unmarshal([]byte(nativeJSON.String), &provenance); err != nil {
			return ConversationEvent{}, fmt.Errorf("decode native provenance: %w", err)
		}
		event.NativeProvenance = &provenance
	}
	return event, nil
}

func nativeProvenanceJSON(provenance *NativeProvenance) any {
	if provenance == nil {
		return nil
	}
	data, err := json.Marshal(provenance)
	if err != nil {
		return nil
	}
	return string(data)
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
		SELECT `+conversationEventColumns+`
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
		SELECT `+conversationEventColumns+`
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
