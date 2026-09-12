package continuity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"web-console/internal/dbx"
)

type SearchMatch struct {
	EventID, SessionID, Role, CreatedAt, Excerpt, LifecycleState string
	Sequence                                                     int64
}

// SearchOptions keeps local continuity search independent of provider
// availability. Catalog fields are deliberately first-class filters because
// the catalog may be the only surviving metadata when the session row is
// missing.
type SearchOptions struct {
	Query          string
	State          string
	AgentType      string
	CWD            string
	CreatedAfter   time.Time
	Limit          int
	SessionID      string
	AgentSessionID string
	Title          string
	TopicSummary   string
}

// SearchLocal is exact-text FTS over all retained Web Console events. It has
// no dependency on Agent Manager, Search Hub, embeddings, or an LLM.
func SearchLocal(ctx context.Context, db dbx.Handle, query, state, agent, cwd string, after time.Time, limit int) ([]SearchMatch, bool, int64, int64, error) {
	return SearchLocalWithOptions(ctx, db, SearchOptions{Query: query, State: state, AgentType: agent, CWD: cwd, CreatedAfter: after, Limit: limit})
}

func SearchLocalWithOptions(ctx context.Context, db dbx.Handle, options SearchOptions) ([]SearchMatch, bool, int64, int64, error) {
	query := strings.TrimSpace(options.Query)
	fts := plainSearchQuery(query)
	if fts == "" && strings.TrimSpace(options.SessionID) == "" && strings.TrimSpace(options.AgentSessionID) == "" && strings.TrimSpace(options.Title) == "" && strings.TrimSpace(options.TopicSummary) == "" {
		return []SearchMatch{}, false, 0, 0, nil
	}
	limit := options.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where := make([]string, 0, 8)
	args := make([]any, 0, 16)
	catalogAvailable := tableExists(ctx, db, "conversation_catalog")
	catalogTitlesAvailable := catalogAvailable && tableHasColumns(ctx, db, "conversation_catalog", "current_title", "original_title", "topic_summary")
	textTerms := make([]string, 0, 3)
	if fts != "" {
		// FTS is a rebuildable projection. Keep the durable event text as a
		// correctness fallback so a stale or interrupted index cannot make an
		// otherwise retained conversation undiscoverable.
		textTerms = append(textTerms, "(e.rowid IN (SELECT rowid FROM conversation_events_fts WHERE conversation_events_fts MATCH ?) OR e.text LIKE ? ESCAPE '\\')")
		args = append(args, fts, "%"+escapeLike(query)+"%")
	}
	// Identity lookup is deliberately additive to FTS. A recovered pane can
	// have no live session row, and its thread identity is metadata rather than
	// conversation text; both must remain discoverable without an LLM.
	identityIDs := catalogSessionIDs(ctx, db, query)
	identity := make([]string, 0, 4+len(identityIDs))
	identityArgs := make([]any, 0, 4+len(identityIDs))
	if query != "" {
		identity = append(identity, "e.session_id = ?", "e.id = ?")
		identityArgs = append(identityArgs, query, query)
		if catalogAvailable {
			identity = append(identity, "c.session_id = ?", "c.agent_session_id = ?")
			identityArgs = append(identityArgs, query, query)
		}
	}
	for _, id := range identityIDs {
		identity = append(identity, "e.session_id = ?")
		identityArgs = append(identityArgs, id)
	}
	if len(identity) > 0 {
		textTerms = append(textTerms, strings.Join(identity, " OR "))
	}
	if query != "" && catalogTitlesAvailable {
		textTerms = append(textTerms, "c.current_title LIKE ? ESCAPE '\\'", "c.original_title LIKE ? ESCAPE '\\'", "c.topic_summary LIKE ? ESCAPE '\\'")
		like := "%" + escapeLike(query) + "%"
		identityArgs = append(identityArgs, like, like, like)
	}
	if len(textTerms) > 0 {
		where = append(where, "("+strings.Join(textTerms, " OR ")+")")
	}
	args = append(args, identityArgs...)
	if options.AgentType != "" {
		if !catalogTitlesAvailable {
			where = append(where, "s.agent_type = ?")
		} else {
			where = append(where, "COALESCE(NULLIF(s.agent_type, ''), c.agent_type) = ?")
		}
		args = append(args, options.AgentType)
	}
	if options.CWD != "" {
		if !catalogTitlesAvailable {
			where = append(where, "s.cwd = ?")
		} else {
			where = append(where, "COALESCE(NULLIF(s.cwd, ''), c.cwd) = ?")
		}
		args = append(args, options.CWD)
	}
	if options.SessionID != "" {
		where = append(where, "e.session_id = ?")
		args = append(args, options.SessionID)
	}
	if options.AgentSessionID != "" && catalogAvailable {
		where = append(where, "c.agent_session_id = ?")
		args = append(args, options.AgentSessionID)
	} else if options.AgentSessionID != "" {
		return []SearchMatch{}, false, 0, 0, nil
	}
	if options.Title != "" {
		if !catalogTitlesAvailable {
			return []SearchMatch{}, false, 0, 0, nil
		}
		where = append(where, "(c.current_title LIKE ? ESCAPE '\\' OR c.original_title LIKE ? ESCAPE '\\')")
		like := "%" + escapeLike(options.Title) + "%"
		args = append(args, like, like)
	}
	if options.TopicSummary != "" {
		if !catalogTitlesAvailable {
			return []SearchMatch{}, false, 0, 0, nil
		}
		where = append(where, "c.topic_summary LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escapeLike(options.TopicSummary)+"%")
	}
	if !options.CreatedAfter.IsZero() {
		where = append(where, "e.created_at >= ?")
		args = append(args, options.CreatedAfter.UTC().Format(time.RFC3339Nano))
	}
	catalogStates := catalogLifecycleStates(ctx, db)
	switch options.State {
	case "live":
		if catalogAvailable {
			where = append(where, "((s.id IS NOT NULL AND s.archived_at = '' AND s.status = 'live') OR c.lifecycle_state = 'live')")
		} else {
			where = append(where, "s.id IS NOT NULL AND s.archived_at = '' AND s.status = 'live'")
		}
	case "archived":
		if catalogAvailable {
			where = append(where, "(s.id IS NULL OR s.archived_at <> '' OR s.status = 'dismissed' OR c.lifecycle_state IN ('archived', 'repaired'))")
		} else {
			where = append(where, "(s.id IS NULL OR s.archived_at <> '' OR s.status = 'dismissed')")
		}
	case "recoverable":
		recoverable := "s.id IS NOT NULL AND (s.status = 'awaiting_recovery' OR s.archived_at <> '')"
		if ids := sessionIDsForState(catalogStates, "recoverable"); len(ids) > 0 {
			placeholders := make([]string, len(ids))
			for i := range ids {
				placeholders[i] = "?"
				args = append(args, ids[i])
			}
			recoverable += " OR e.session_id IN (" + strings.Join(placeholders, ",") + ")"
		}
		if catalogAvailable {
			where = append(where, "("+recoverable+" OR c.lifecycle_state = 'recoverable')")
		} else {
			where = append(where, "("+recoverable+")")
		}
	case "exited":
		// SQLite session metadata represents a provider exit as awaiting
		// recovery; the continuity result still exposes it as recoverable.
		if catalogAvailable {
			where = append(where, "((s.id IS NOT NULL AND s.status = 'awaiting_recovery') OR c.lifecycle_state = 'exited')")
		} else {
			where = append(where, "s.id IS NOT NULL AND s.status = 'awaiting_recovery'")
		}
	case "any", "":
	default:
		return nil, false, 0, 0, fmt.Errorf("unknown lifecycle state %q", options.State)
	}
	catalogJoin := ""
	if catalogAvailable {
		catalogJoin = " LEFT JOIN conversation_catalog c ON c.session_id=e.session_id"
	}
	from := " FROM conversation_events e LEFT JOIN sessions s ON s.id=e.session_id" + catalogJoin + " WHERE " + strings.Join(where, " AND ")
	var total, distinct int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*)"+from, args...).Scan(&total); err != nil {
		return nil, false, 0, 0, err
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT e.session_id)"+from, args...).Scan(&distinct); err != nil {
		return nil, false, 0, 0, err
	}
	rows, err := db.QueryContext(ctx, "SELECT e.id,e.session_id,e.sequence,e.role,e.created_at,e.text,COALESCE(s.archived_at,''),COALESCE(s.status,'')"+from+" ORDER BY e.created_at DESC, e.sequence DESC LIMIT ?", append(args, limit+1)...)
	if err != nil {
		return nil, false, 0, 0, err
	}
	defer rows.Close()
	out := make([]SearchMatch, 0, limit)
	for rows.Next() {
		var m SearchMatch
		var text, archivedAt, status string
		if err := rows.Scan(&m.EventID, &m.SessionID, &m.Sequence, &m.Role, &m.CreatedAt, &text, &archivedAt, &status); err != nil {
			return nil, false, 0, 0, err
		}
		m.Excerpt = excerpt(text, query)
		m.LifecycleState = catalogStates[m.SessionID]
		if m.LifecycleState == "" {
			m.LifecycleState = "live"
		}
		if archivedAt != "" || status == "dismissed" {
			m.LifecycleState = "archived"
		}
		if status == "awaiting_recovery" {
			m.LifecycleState = "recoverable"
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, 0, 0, err
	}
	truncated := len(out) > limit
	if truncated {
		out = out[:limit]
	}
	return out, truncated, total, distinct, nil
}

func tableExists(ctx context.Context, db dbx.Handle, table string) bool {
	var found int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&found)
	return err == nil && found == 1
}

func tableHasColumns(ctx context.Context, db dbx.Handle, table string, columns ...string) bool {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false
	}
	defer rows.Close()
	found := make(map[string]bool, len(columns))
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey) == nil {
			found[name] = true
		}
	}
	for _, column := range columns {
		if !found[column] {
			return false
		}
	}
	return true
}

func escapeLike(value string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value)
}

func catalogLifecycleStates(ctx context.Context, db dbx.Handle) map[string]string {
	rows, err := db.QueryContext(ctx, `SELECT session_id, lifecycle_state FROM conversation_catalog`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	states := make(map[string]string)
	for rows.Next() {
		var id, state string
		if rows.Scan(&id, &state) == nil && id != "" {
			states[id] = state
		}
	}
	return states
}

func sessionIDsForState(states map[string]string, state string) []string {
	ids := make([]string, 0)
	for id, candidate := range states {
		if candidate == state {
			ids = append(ids, id)
		}
	}
	return ids
}

func catalogSessionIDs(ctx context.Context, db dbx.Handle, query string) []string {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	pattern := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(query, "\\", "\\\\"), "%", "\\%"), "_", "\\_")
	rows, err := db.QueryContext(ctx, "SELECT session_id FROM conversation_catalog WHERE agent_session_id LIKE ? ESCAPE '\\' OR session_id LIKE ? ESCAPE '\\'", "%"+pattern+"%", "%"+pattern+"%")
	if err != nil {
		// Older disposable fixtures may not include the optional catalog table.
		// Native event identity search still works in that case.
		return nil
	}
	defer rows.Close()
	ids := make([]string, 0, 4)
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func plainSearchQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		out = append(out, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
	}
	return strings.Join(out, " AND ")
}

func excerpt(text, query string) string {
	start := strings.Index(strings.ToLower(text), strings.ToLower(query))
	if start < 0 {
		start = 0
	}
	from := start - 60
	if from < 0 {
		from = 0
	}
	to := from + 160
	if to > len(text) {
		to = len(text)
	}
	return text[from:to]
}
