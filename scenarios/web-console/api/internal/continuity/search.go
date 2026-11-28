package continuity

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"web-console/internal/dbx"
)

type SearchMatch struct {
	EventID, SessionID, Role, CreatedAt, OccurredAt, Excerpt, LifecycleState string
	Sequence                                                                 int64
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

// NativeArchiveSourceForAgentType returns the stable own-store source label
// used by WC4's admitted archive adapters. Native projections can survive
// without a matching sessions/catalog row, so text retrieval uses this source
// label as its harness identity fallback.
func NativeArchiveSourceForAgentType(agentType string) string {
	switch agentType {
	case "claude", "codex", "grok", "opencode", "gemini-cli", "qwen-code", "antigravity", "cursor":
		return agentType + "_archive"
	default:
		return ""
	}
}

// SearchLocal is exact-text FTS over all retained Web Console events. It has
// no dependency on Agent Manager, Search Hub, embeddings, or an LLM.
func SearchLocal(ctx context.Context, db dbx.Handle, query, state, agent, cwd string, after time.Time, limit int) ([]SearchMatch, bool, int64, int64, error) {
	return SearchLocalWithOptions(ctx, db, SearchOptions{Query: query, State: state, AgentType: agent, CWD: cwd, CreatedAfter: after, Limit: limit})
}

func SearchLocalWithOptions(ctx context.Context, db dbx.Handle, options SearchOptions) ([]SearchMatch, bool, int64, int64, error) {
	query := strings.TrimSpace(options.Query)
	if query == "" && strings.TrimSpace(options.SessionID) == "" && strings.TrimSpace(options.AgentSessionID) == "" && strings.TrimSpace(options.Title) == "" && strings.TrimSpace(options.TopicSummary) == "" {
		return []SearchMatch{}, false, 0, 0, nil
	}
	limit := options.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where := make([]string, 0, 8)
	args := make([]any, 0, 16)
	catalogAvailable := tableExists(ctx, db, "conversation_catalog")
	catalogIdentityAvailable := catalogAvailable && tableHasColumns(ctx, db, "conversation_catalog", "session_id", "agent_session_id")
	catalogTitlesAvailable := catalogAvailable && tableHasColumns(ctx, db, "conversation_catalog", "current_title", "original_title", "topic_summary")
	if query != "" {
		// Every way a query can name an event is an indexed rowid source, and
		// the candidates are their union. An OR across these terms in the outer
		// WHERE would instead force a scan of every retained event (tens of
		// seconds on a production archive) because SQLite cannot index an OR
		// that mixes the full-text mirror with joined catalog columns.
		//
		// There is no LIKE scan over event text. The FTS mirror is maintained by
		// triggers in the same transaction as every insert, delete and text
		// update, and is rebuilt during schema initialisation before the server
		// serves, so it cannot lag the durable rows it indexes.
		candidates := make([]string, 0, 4)
		if fts := plainSearchQuery(query); fts != "" {
			candidates = append(candidates, "SELECT rowid FROM conversation_events_fts WHERE conversation_events_fts MATCH ?")
			args = append(args, fts)
		}
		// Identity lookup is deliberately additive to FTS. A recovered pane can
		// have no live session row, and its thread identity is metadata rather
		// than conversation text; both must remain discoverable without an LLM.
		candidates = append(candidates, "SELECT rowid FROM conversation_events WHERE id = ?", "SELECT rowid FROM conversation_events WHERE session_id = ?")
		args = append(args, query, query)
		if catalogIdentityAvailable {
			like := "%" + escapeLike(query) + "%"
			catalogTerms := []string{"session_id LIKE ? ESCAPE '\\'", "agent_session_id LIKE ? ESCAPE '\\'"}
			args = append(args, like, like)
			if catalogTitlesAvailable {
				catalogTerms = append(catalogTerms, "current_title LIKE ? ESCAPE '\\'", "original_title LIKE ? ESCAPE '\\'", "topic_summary LIKE ? ESCAPE '\\'")
				args = append(args, like, like, like)
			}
			candidates = append(candidates, "SELECT rowid FROM conversation_events WHERE session_id IN (SELECT session_id FROM conversation_catalog WHERE "+strings.Join(catalogTerms, " OR ")+")")
		}
		where = append(where, "e.rowid IN ("+strings.Join(candidates, " UNION ")+")")
	}
	if options.AgentType != "" {
		agentTypeMatch := "s.agent_type = ?"
		if catalogTitlesAvailable {
			agentTypeMatch = "COALESCE(NULLIF(s.agent_type, ''), c.agent_type) = ?"
		}
		if source := NativeArchiveSourceForAgentType(options.AgentType); source != "" && tableHasColumns(ctx, db, "conversation_events", "source") {
			where = append(where, "("+agentTypeMatch+" OR (s.id IS NULL AND e.source = ?))")
			args = append(args, options.AgentType, source)
		} else {
			where = append(where, agentTypeMatch)
			args = append(args, options.AgentType)
		}
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
	// Catalog filters select sessions, so they are stated as session-id sets
	// the (session_id, sequence) index can serve rather than as per-event
	// predicates on the joined catalog row, which only a full scan can test.
	if options.AgentSessionID != "" && catalogIdentityAvailable {
		where = append(where, "e.session_id IN (SELECT session_id FROM conversation_catalog WHERE agent_session_id = ?)")
		args = append(args, options.AgentSessionID)
	} else if options.AgentSessionID != "" {
		return []SearchMatch{}, false, 0, 0, nil
	}
	if options.Title != "" {
		if !catalogTitlesAvailable {
			return []SearchMatch{}, false, 0, 0, nil
		}
		where = append(where, "e.session_id IN (SELECT session_id FROM conversation_catalog WHERE current_title LIKE ? ESCAPE '\\' OR original_title LIKE ? ESCAPE '\\')")
		like := "%" + escapeLike(options.Title) + "%"
		args = append(args, like, like)
	}
	if options.TopicSummary != "" {
		if !catalogTitlesAvailable {
			return []SearchMatch{}, false, 0, 0, nil
		}
		where = append(where, "e.session_id IN (SELECT session_id FROM conversation_catalog WHERE topic_summary LIKE ? ESCAPE '\\')")
		args = append(args, "%"+escapeLike(options.TopicSummary)+"%")
	}
	if !options.CreatedAfter.IsZero() {
		// Native occurrence timestamps have a fixed-width UTC representation, so
		// their range check stays indexable. Legacy created_at is variable-width
		// RFC3339Nano and must be parsed to compare fractional seconds correctly.
		where = append(where, `((e.source IN ('claude_archive','codex_archive','grok_archive','opencode_archive','gemini-cli_archive','qwen-code_archive','antigravity_archive','cursor_archive') AND e.occurred_at >= ?) OR (e.source NOT IN ('claude_archive','codex_archive','grok_archive','opencode_archive','gemini-cli_archive','qwen-code_archive','antigravity_archive','cursor_archive') AND julianday(e.created_at) >= julianday(?)))`)
		afterNative := options.CreatedAfter.UTC().Format("2006-01-02T15:04:05.000000000Z")
		afterIngested := options.CreatedAfter.UTC().Format(time.RFC3339Nano)
		args = append(args, afterNative, afterIngested)
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
	// One pass: the matched set is materialised once, and the totals and the
	// newest page are read from it. Only the page rows load their text.
	statement := `WITH matched AS MATERIALIZED (
		SELECT e.rowid AS rid, e.session_id AS sid, COALESCE(e.occurred_at, e.created_at) AS at, e.sequence AS seq
		FROM conversation_events e LEFT JOIN sessions s ON s.id=e.session_id` + catalogJoin + `
		WHERE ` + strings.Join(where, " AND ") + `
	), page AS (
		SELECT rid, at, seq FROM matched ORDER BY at DESC, seq DESC LIMIT ?
	)
	SELECT (SELECT COUNT(*) FROM matched), (SELECT COUNT(DISTINCT sid) FROM matched),
		e.id, e.session_id, e.sequence, e.role, e.created_at, COALESCE(e.occurred_at, ''), e.text,
		COALESCE(s.id, ''), COALESCE(s.archived_at, ''), COALESCE(s.status, '')
	FROM page JOIN conversation_events e ON e.rowid = page.rid LEFT JOIN sessions s ON s.id = e.session_id
	ORDER BY page.at DESC, page.seq DESC`
	rows, err := db.QueryContext(ctx, statement, append(args, limit+1)...)
	if err != nil {
		return nil, false, 0, 0, err
	}
	defer rows.Close()
	var total, distinct int64
	out := make([]SearchMatch, 0, limit)
	for rows.Next() {
		var m SearchMatch
		var text, sessionMetadataID, archivedAt, status string
		if err := rows.Scan(&total, &distinct, &m.EventID, &m.SessionID, &m.Sequence, &m.Role, &m.CreatedAt, &m.OccurredAt, &text, &sessionMetadataID, &archivedAt, &status); err != nil {
			return nil, false, 0, 0, err
		}
		m.Excerpt = excerpt(text, query)
		m.LifecycleState = catalogStates[m.SessionID]
		if m.LifecycleState == "" {
			if sessionMetadataID == "" {
				m.LifecycleState = "archived"
			} else {
				m.LifecycleState = "live"
			}
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

// plainSearchQuery turns a typed query into an FTS5 expression: each
// whitespace-separated term is a phrase (so "plan-manager" matches the token
// pair plan, manager), and all terms must match. The final term also matches
// as a prefix once it is long enough to stay selective, so a word still being
// typed finds its completions. Terms without a letter or digit cannot match
// the unicode61 tokenizer and are dropped; such a query is served by the
// identity and catalog sources alone.
func plainSearchQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	out := make([]string, 0, len(terms))
	for index, term := range terms {
		tokens := strings.FieldsFunc(term, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		if len(tokens) == 0 {
			continue
		}
		phrase := `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
		if index == len(terms)-1 && utf8.RuneCountInString(tokens[len(tokens)-1]) >= minPrefixTokenRunes {
			last, _ := utf8.DecodeLastRuneInString(term)
			if unicode.IsLetter(last) || unicode.IsDigit(last) {
				phrase += "*"
			}
		}
		out = append(out, phrase)
	}
	return strings.Join(out, " AND ")
}

// minPrefixTokenRunes keeps prefix expansion selective: a one- or two-letter
// prefix expands to most of the vocabulary and costs seconds on a large index.
const minPrefixTokenRunes = 3

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
