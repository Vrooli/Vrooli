package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// conversationProjectionSinceCap bounds a since-sequence projection fetch.
	conversationProjectionSinceCap = 2000
	// conversationWorkRangeCap bounds one work-detail fetch.
	conversationWorkRangeCap = 2000
	// conversationWorkPreviewChars is the default preview length of a work
	// row's text; the full text is fetched by id when asked for.
	conversationWorkPreviewChars = 1200
)

// hiddenConversationRow is the narrow shape of a work row the projection
// counts. The projection index covers it, so counting never reads row text.
type hiddenConversationRow struct {
	sequence int64
	kind     string
	at       time.Time
}

// buildConversationProjection interleaves a thread's conversation rows with
// work rows. visible is ascending; prevVisible is the thread's conversation
// row before visible[0] (0 when none); hidden are the thread's work rows after
// prevVisible; tailTo > 0 adds the trailing span after the last visible row.
// Every sequence from prevVisible+1 to the end is covered by a row, so paging
// stays contiguous even across other threads' rows.
func buildConversationProjection(sessionID, threadID string, visible []ConversationEvent, prevVisible int64, hidden []hiddenConversationRow, tailTo int64) []ConversationEvent {
	items := make([]ConversationEvent, 0, len(visible)*2+1)
	cursor := 0
	spanUntil := func(from, to int64) {
		if from > to {
			return
		}
		span := ConversationWorkSpan{FromSequence: from, ToSequence: to}
		for cursor < len(hidden) && hidden[cursor].sequence <= to {
			if hidden[cursor].sequence >= from {
				span.add(hidden[cursor].kind, hidden[cursor].at)
			}
			cursor++
		}
		items = append(items, span.workEvent(sessionID, threadID))
	}
	boundary := prevVisible
	for _, event := range visible {
		spanUntil(boundary+1, event.Sequence-1)
		items = append(items, event)
		boundary = event.Sequence
	}
	if tailTo > boundary {
		spanUntil(boundary+1, tailTo)
	}
	return items
}

func visibleKindsSQL() string {
	quoted := make([]string, len(conversationVisibleKinds))
	for index, kind := range conversationVisibleKinds {
		quoted[index] = "'" + kind + "'"
	}
	return "(" + strings.Join(quoted, ", ") + ")"
}

// ensureConversationProjectionIndex creates the index the projection reads.
// It covers thread, sequence, kind and time, so paging conversation rows and
// counting work never touch the (often very large) row text.
func ensureConversationProjectionIndex(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	_, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_conversation_events_projection
		ON conversation_events(session_id, thread_id, sequence, kind, occurred_at)`)
	return err
}

func (r *SQLConversationRepository) ListProjectionPage(ctx context.Context, sessionID string, query ConversationProjectionQuery) (ConversationProjectionPage, error) {
	page := ConversationProjectionPage{State: ConversationSessionState{SessionID: sessionID, Events: []ConversationEvent{}}}
	var lastSequence int64
	err := r.db.QueryRowContext(ctx, `SELECT last_seen_sequence, last_listened_sequence, last_sequence FROM conversation_sessions WHERE session_id = ?`, sessionID).
		Scan(&page.State.Cursor.LastSeenSequence, &page.State.Cursor.LastListenedSequence, &lastSequence)
	if errors.Is(err, sql.ErrNoRows) {
		return page, nil
	}
	if err != nil {
		return page, fmt.Errorf("load cursor: %w", err)
	}
	visibleKinds := visibleKindsSQL()
	limit := query.Limit
	if limit <= 0 {
		limit = 200
	}
	var visible []ConversationEvent
	// A since read stops at the newest row unless it is cut by its limit; a
	// cut page carries no tail span (rows it did not return are not work).
	sinceCut := false
	if query.SinceSequence > 0 {
		sinceLimit := projectionSinceLimit(query.Limit)
		visible, err = r.queryProjectionEvents(ctx, `SELECT `+conversationEventColumns+` FROM conversation_events
			WHERE session_id = ? AND thread_id = ? AND kind IN `+visibleKinds+` AND sequence > ? ORDER BY sequence LIMIT ?`,
			sessionID, query.ThreadID, query.SinceSequence, sinceLimit+1)
		if err == nil && len(visible) > sinceLimit {
			visible, sinceCut, page.HasMore = visible[:sinceLimit], true, true
		}
	} else {
		sqlText := `SELECT ` + conversationEventColumns + ` FROM conversation_events WHERE session_id = ? AND thread_id = ? AND kind IN ` + visibleKinds
		args := []any{sessionID, query.ThreadID}
		if query.BeforeSequence > 0 {
			sqlText += ` AND sequence < ?`
			args = append(args, query.BeforeSequence)
		}
		sqlText += ` ORDER BY sequence DESC LIMIT ?`
		args = append(args, limit+1)
		visible, err = r.queryProjectionEvents(ctx, sqlText, args...)
		if err == nil {
			page.HasMore = len(visible) > limit
			if page.HasMore {
				visible = visible[:limit]
			}
			for left, right := 0, len(visible)-1; left < right; left, right = left+1, right-1 {
				visible[left], visible[right] = visible[right], visible[left]
			}
		}
	}
	if err != nil {
		return page, err
	}
	newest := query.BeforeSequence == 0 && !sinceCut
	if len(visible) == 0 && !newest {
		return page, nil
	}
	upper := lastSequence + 1
	if len(visible) > 0 {
		upper = visible[0].Sequence
	}
	var prevVisible int64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM conversation_events
		WHERE session_id = ? AND thread_id = ? AND kind IN `+visibleKinds+` AND sequence < ?`,
		sessionID, query.ThreadID, upper).Scan(&prevVisible); err != nil {
		return page, fmt.Errorf("previous conversation row: %w", err)
	}
	tailTo := int64(0)
	hiddenTo := lastSequence
	if newest {
		tailTo = lastSequence
	} else {
		hiddenTo = visible[len(visible)-1].Sequence
	}
	hidden, err := r.queryHiddenRows(ctx, sessionID, query.ThreadID, prevVisible, hiddenTo)
	if err != nil {
		return page, err
	}
	page.State.Events = buildConversationProjection(sessionID, query.ThreadID, visible, prevVisible, hidden, tailTo)
	if len(page.State.Events) > 0 {
		page.OldestSequence = prevVisible + 1
	}
	if query.ThreadID == "" && newest {
		page.Threads, err = r.ListThreadSummaries(ctx, sessionID)
		if err != nil {
			return page, err
		}
	}
	return page, nil
}

func (r *SQLConversationRepository) queryProjectionEvents(ctx context.Context, query string, args ...any) ([]ConversationEvent, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query conversation rows: %w", err)
	}
	defer rows.Close()
	var events []ConversationEvent
	for rows.Next() {
		event, scanErr := scanConversationEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *SQLConversationRepository) queryHiddenRows(ctx context.Context, sessionID, threadID string, after, through int64) ([]hiddenConversationRow, error) {
	if through <= after {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT sequence, kind, COALESCE(occurred_at, '') FROM conversation_events
		WHERE session_id = ? AND thread_id = ? AND sequence > ? AND sequence <= ? AND kind NOT IN `+visibleKindsSQL()+`
		ORDER BY sequence`, sessionID, threadID, after, through)
	if err != nil {
		return nil, fmt.Errorf("query work rows: %w", err)
	}
	defer rows.Close()
	var hidden []hiddenConversationRow
	for rows.Next() {
		var row hiddenConversationRow
		var at string
		if err := rows.Scan(&row.sequence, &row.kind, &at); err != nil {
			return nil, fmt.Errorf("scan work row: %w", err)
		}
		row.at = parseConversationTime(at)
		hidden = append(hidden, row)
	}
	return hidden, rows.Err()
}

// ListWorkRange returns a thread's work rows in from..to with text cut to a
// preview, for expanding one collapsed span.
func (r *SQLConversationRepository) ListWorkRange(ctx context.Context, sessionID, threadID string, from, to int64, previewChars int) ([]ConversationEvent, error) {
	if previewChars <= 0 {
		previewChars = conversationWorkPreviewChars
	}
	columns := strings.Replace(conversationEventColumns, "source, role, text,", "source, role, substr(text, 1, ?),", 1)
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM conversation_events
		WHERE session_id = ? AND thread_id = ? AND sequence >= ? AND sequence <= ? AND kind NOT IN `+visibleKindsSQL()+`
		ORDER BY sequence LIMIT ?`, previewChars+1, sessionID, threadID, from, to, conversationWorkRangeCap)
	if err != nil {
		return nil, fmt.Errorf("query work range: %w", err)
	}
	defer rows.Close()
	events := []ConversationEvent{}
	for rows.Next() {
		event, scanErr := scanConversationEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, truncateWorkText(event, previewChars))
	}
	return events, rows.Err()
}

func truncateWorkText(event ConversationEvent, previewChars int) ConversationEvent {
	if utf8.RuneCountInString(event.Text) > previewChars {
		runes := []rune(event.Text)
		event.Text = string(runes[:previewChars])
		event.TextTruncated = true
	}
	return event
}

func (r *SQLConversationRepository) UpsertThread(ctx context.Context, sessionID string, thread ConversationThread) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO conversation_threads (session_id, thread_id, kind, label, description, parent_thread_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, thread_id) DO UPDATE SET
			kind = excluded.kind,
			label = CASE WHEN excluded.label != '' THEN excluded.label ELSE conversation_threads.label END,
			description = CASE WHEN excluded.description != '' THEN excluded.description ELSE conversation_threads.description END,
			parent_thread_id = CASE WHEN excluded.parent_thread_id != '' THEN excluded.parent_thread_id ELSE conversation_threads.parent_thread_id END`,
		sessionID, thread.ThreadID, thread.Kind, thread.Label, thread.Description, thread.ParentThreadID, formatTime(thread.CreatedAt))
	return err
}

func (r *SQLConversationRepository) ListThreadSummaries(ctx context.Context, sessionID string) ([]ConversationThreadSummary, error) {
	visibleKinds := visibleKindsSQL()
	rows, err := r.db.QueryContext(ctx, `SELECT thread_id,
			SUM(CASE WHEN kind IN `+visibleKinds+` THEN 1 ELSE 0 END),
			SUM(CASE WHEN kind IN `+visibleKinds+` THEN 0 ELSE 1 END),
			MIN(sequence), MAX(sequence), COALESCE(MIN(occurred_at), ''), COALESCE(MAX(occurred_at), '')
		FROM conversation_events WHERE session_id = ? AND thread_id != '' GROUP BY thread_id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("summarise threads: %w", err)
	}
	summaries := map[string]*ConversationThreadSummary{}
	for rows.Next() {
		summary := &ConversationThreadSummary{}
		var started, ended string
		if err := rows.Scan(&summary.ThreadID, &summary.MessageCount, &summary.WorkCount, &summary.FirstSequence, &summary.LastSequence, &started, &ended); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan thread summary: %w", err)
		}
		summary.StartedAt, summary.EndedAt = parseConversationTime(started), parseConversationTime(ended)
		summaries[summary.ThreadID] = summary
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}
	meta, err := r.db.QueryContext(ctx, `SELECT thread_id, kind, label, description, parent_thread_id, created_at FROM conversation_threads WHERE session_id = ?`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load threads: %w", err)
	}
	defer meta.Close()
	for meta.Next() {
		var thread ConversationThread
		var created string
		if err := meta.Scan(&thread.ThreadID, &thread.Kind, &thread.Label, &thread.Description, &thread.ParentThreadID, &created); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		thread.CreatedAt = parseConversationTime(created)
		if summary, ok := summaries[thread.ThreadID]; ok {
			summary.ConversationThread = thread
		}
	}
	if err := meta.Err(); err != nil {
		return nil, err
	}
	return sortedThreadSummaries(summaries), nil
}

func sortedThreadSummaries(summaries map[string]*ConversationThreadSummary) []ConversationThreadSummary {
	out := make([]ConversationThreadSummary, 0, len(summaries))
	for _, summary := range summaries {
		if summary.Kind == "" {
			summary.Kind = "subagent"
		}
		out = append(out, *summary)
	}
	sort.Slice(out, func(left, right int) bool { return out[left].FirstSequence < out[right].FirstSequence })
	return out
}

func (r *SQLConversationRepository) HasMainThreadPrompt(ctx context.Context, sessionID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM conversation_events
		WHERE session_id = ? AND thread_id = '' AND role = 'user' AND kind = 'message')`, sessionID).Scan(&exists)
	return exists == 1, err
}

// --- In-memory repository: the same projection over the session slice. ---

func (r *InMemoryConversationRepository) ListProjectionPage(ctx context.Context, sessionID string, query ConversationProjectionQuery) (ConversationProjectionPage, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return ConversationProjectionPage{}, err
	}
	page := ConversationProjectionPage{State: ConversationSessionState{SessionID: sessionID, Cursor: state.Cursor, Events: []ConversationEvent{}}}
	var lastSequence int64
	var visible []ConversationEvent
	for _, event := range state.Events {
		lastSequence = max(lastSequence, event.Sequence)
		if event.ThreadID != query.ThreadID || !isVisibleConversationKind(event.Kind) {
			continue
		}
		if query.SinceSequence > 0 && event.Sequence <= query.SinceSequence {
			continue
		}
		if query.SinceSequence == 0 && query.BeforeSequence > 0 && event.Sequence >= query.BeforeSequence {
			continue
		}
		visible = append(visible, event)
	}
	sinceCut := false
	if query.SinceSequence == 0 {
		limit := query.Limit
		if limit <= 0 {
			limit = 200
		}
		if len(visible) > limit {
			page.HasMore = true
			visible = visible[len(visible)-limit:]
		}
	} else if limit := projectionSinceLimit(query.Limit); len(visible) > limit {
		visible, sinceCut, page.HasMore = visible[:limit], true, true
	}
	newest := query.BeforeSequence == 0 && !sinceCut
	if len(visible) == 0 && !newest {
		return page, nil
	}
	upper := lastSequence + 1
	if len(visible) > 0 {
		upper = visible[0].Sequence
	}
	var prevVisible int64
	var hidden []hiddenConversationRow
	hiddenTo := lastSequence
	if !newest {
		hiddenTo = visible[len(visible)-1].Sequence
	}
	for _, event := range state.Events {
		if event.ThreadID != query.ThreadID {
			continue
		}
		if isVisibleConversationKind(event.Kind) {
			if event.Sequence < upper {
				prevVisible = max(prevVisible, event.Sequence)
			}
			continue
		}
		hidden = append(hidden, hiddenConversationRow{sequence: event.Sequence, kind: event.Kind, at: event.OccurredAt})
	}
	kept := hidden[:0]
	for _, row := range hidden {
		if row.sequence > prevVisible && row.sequence <= hiddenTo {
			kept = append(kept, row)
		}
	}
	tailTo := int64(0)
	if newest {
		tailTo = lastSequence
	}
	page.State.Events = buildConversationProjection(sessionID, query.ThreadID, visible, prevVisible, kept, tailTo)
	if len(page.State.Events) > 0 {
		page.OldestSequence = prevVisible + 1
	}
	if query.ThreadID == "" && newest {
		page.Threads, err = r.ListThreadSummaries(ctx, sessionID)
	}
	return page, err
}

func (r *InMemoryConversationRepository) ListWorkRange(ctx context.Context, sessionID, threadID string, from, to int64, previewChars int) ([]ConversationEvent, error) {
	if previewChars <= 0 {
		previewChars = conversationWorkPreviewChars
	}
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	events := []ConversationEvent{}
	for _, event := range state.Events {
		if event.ThreadID == threadID && event.Sequence >= from && event.Sequence <= to && !isVisibleConversationKind(event.Kind) {
			events = append(events, truncateWorkText(event, previewChars))
			if len(events) >= conversationWorkRangeCap {
				break
			}
		}
	}
	return events, nil
}

func (r *InMemoryConversationRepository) UpsertThread(_ context.Context, sessionID string, thread ConversationThread) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.threads == nil {
		r.threads = map[string]map[string]ConversationThread{}
	}
	if r.threads[sessionID] == nil {
		r.threads[sessionID] = map[string]ConversationThread{}
	}
	existing := r.threads[sessionID][thread.ThreadID]
	if thread.Label == "" {
		thread.Label = existing.Label
	}
	if thread.Description == "" {
		thread.Description = existing.Description
	}
	if thread.ParentThreadID == "" {
		thread.ParentThreadID = existing.ParentThreadID
	}
	if !existing.CreatedAt.IsZero() {
		thread.CreatedAt = existing.CreatedAt
	}
	r.threads[sessionID][thread.ThreadID] = thread
	return nil
}

func (r *InMemoryConversationRepository) ListThreadSummaries(ctx context.Context, sessionID string) ([]ConversationThreadSummary, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	summaries := map[string]*ConversationThreadSummary{}
	for _, event := range state.Events {
		if event.ThreadID == "" {
			continue
		}
		summary := summaries[event.ThreadID]
		if summary == nil {
			summary = &ConversationThreadSummary{ConversationThread: ConversationThread{ThreadID: event.ThreadID}, FirstSequence: event.Sequence}
			summaries[event.ThreadID] = summary
		}
		if isVisibleConversationKind(event.Kind) {
			summary.MessageCount++
		} else {
			summary.WorkCount++
		}
		summary.LastSequence = max(summary.LastSequence, event.Sequence)
		if at := event.OccurredAt; !at.IsZero() {
			if summary.StartedAt.IsZero() || at.Before(summary.StartedAt) {
				summary.StartedAt = at
			}
			if at.After(summary.EndedAt) {
				summary.EndedAt = at
			}
		}
	}
	r.mu.Lock()
	for id, thread := range r.threads[sessionID] {
		if summary, ok := summaries[id]; ok {
			summary.ConversationThread = thread
		}
	}
	r.mu.Unlock()
	if len(summaries) == 0 {
		return nil, nil
	}
	return sortedThreadSummaries(summaries), nil
}

func (r *InMemoryConversationRepository) HasMainThreadPrompt(ctx context.Context, sessionID string) (bool, error) {
	state, err := r.ListSession(ctx, sessionID)
	if err != nil {
		return false, err
	}
	for _, event := range state.Events {
		if event.ThreadID == "" && event.Role == ConversationRoleUser && (event.Kind == "" || event.Kind == "message") {
			return true, nil
		}
	}
	return false, nil
}

// projectionSinceLimit is how many conversation rows a since read returns:
// the caller's limit, never more than the cap.
func projectionSinceLimit(limit int) int {
	if limit <= 0 || limit > conversationProjectionSinceCap {
		return conversationProjectionSinceCap
	}
	return limit
}
