package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"

	"web-console/internal/dbx"
)

// IntegrityReport is a read-only snapshot. Counts are evidence for operator
// decisions; they are never treated as a deletion list.
type IntegrityReport struct {
	Sessions             int64 `json:"sessions"`
	ConversationSessions int64 `json:"conversation_sessions"`
	ConversationEvents   int64 `json:"conversation_events"`
	Checkpoints          int64 `json:"checkpoints"`
	WorkspacePanes       int64 `json:"workspace_panes"`
	OrphanConversations  int64 `json:"orphan_conversations"`
	// UncatalogedConversations is the actionable health signal. A native
	// session row may be absent after metadata drift while the continuity
	// catalog still preserves a first-class recoverable identity.
	UncatalogedConversations int64  `json:"uncataloged_conversations"`
	OrphanCheckpoints        int64  `json:"orphan_checkpoints"`
	OrphanWorkspacePanes     int64  `json:"orphan_workspace_panes"`
	Generation               string `json:"generation"`
	EventContentHash         string `json:"event_content_hash"`
}

type IntegrityAuditor struct {
	db    dbx.Handle
	cache *ContentReceiptCache
}

func NewIntegrityAuditor(db dbx.Handle) *IntegrityAuditor { return &IntegrityAuditor{db: db} }

// WithContentCache reuses each conversation's content hash across audits while
// its events are unchanged; see ContentReceiptCache.
func (a *IntegrityAuditor) WithContentCache(cache *ContentReceiptCache) *IntegrityAuditor {
	a.cache = cache
	return a
}

// ContentReceiptCache keeps one content hash per conversation with the stamp
// it was computed under. Hashing every retained message body takes minutes on
// a large archive, and live capture changes some conversation every few
// seconds, so a single archive-wide hash would never be reusable. Per
// conversation, an audit rehashes only the conversations whose events changed
// and combines the rest from the cache. Concurrent audits share one
// computation instead of each starting their own.
//
// A stamp is a conversation's event count, highest rowid and sequence sum,
// read from the session index without touching bodies. It changes whenever a
// hashed field can: an append adds a row with a new rowid, a delete lowers the
// count, and a sequence shift (recovery copies renumber a destination) moves
// the sum. Event text, role, id and created_at are never rewritten in place.
type ContentReceiptCache struct {
	gate     chan struct{}
	mu       sync.Mutex
	sessions map[string]sessionContentHash
	// rehashed counts the conversations the last computation read bodies
	// for, which is how tests tell reuse from recomputation.
	rehashed int
}

type sessionContentHash struct {
	stamp, hash string
}

func NewContentReceiptCache() *ContentReceiptCache {
	return &ContentReceiptCache{gate: make(chan struct{}, 1), sessions: map[string]sessionContentHash{}}
}

// Audit reads every figure in the report from one snapshot, so the counts,
// the generation and the content hash describe the same committed state even
// while capture keeps writing.
func (a *IntegrityAuditor) Audit(ctx context.Context) (IntegrityReport, error) {
	var report IntegrityReport
	err := dbx.ReadSnapshot(ctx, a.db, func(q dbx.Querier) error {
		var err error
		report, err = a.audit(ctx, q)
		return err
	})
	return report, err
}

func (a *IntegrityAuditor) audit(ctx context.Context, q dbx.Querier) (IntegrityReport, error) {
	queries := []struct {
		name, query string
		dst         *int64
	}{
		{"sessions", `SELECT COUNT(*) FROM sessions`, nil},
		{"conversation_sessions", `SELECT COUNT(*) FROM conversation_sessions`, nil},
		{"conversation_events", `SELECT COUNT(*) FROM conversation_events`, nil},
		{"checkpoints", `SELECT COUNT(*) FROM agent_transcript_checkpoints`, nil},
		{"workspace_panes", `SELECT COUNT(*) FROM workspace_panes`, nil},
		{"orphan_conversations", `SELECT COUNT(*) FROM conversation_sessions c LEFT JOIN sessions s ON s.id=c.session_id WHERE s.id IS NULL`, nil},
		{"orphan_checkpoints", `SELECT COUNT(*) FROM agent_transcript_checkpoints c LEFT JOIN sessions s ON s.id=c.web_console_session_id WHERE s.id IS NULL`, nil},
		{"orphan_workspace_panes", `SELECT COUNT(*) FROM workspace_panes p LEFT JOIN sessions s ON s.id=p.session_id WHERE s.id IS NULL`, nil},
	}
	values := make([]int64, len(queries))
	for i := range queries {
		if err := q.QueryRowContext(ctx, queries[i].query).Scan(&values[i]); err != nil {
			return IntegrityReport{}, fmt.Errorf("audit %s: %w", queries[i].name, err)
		}
	}
	report := IntegrityReport{Sessions: values[0], ConversationSessions: values[1], ConversationEvents: values[2], Checkpoints: values[3], WorkspacePanes: values[4], OrphanConversations: values[5], OrphanCheckpoints: values[6], OrphanWorkspacePanes: values[7]}
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversation_sessions c LEFT JOIN sessions s ON s.id=c.session_id LEFT JOIN conversation_catalog k ON k.session_id=c.session_id WHERE s.id IS NULL AND k.session_id IS NULL`).Scan(&report.UncatalogedConversations); err != nil {
		// Keep read-only integrity audits usable against pre-catalog disposable
		// fixtures; production schemas always register conversation_catalog.
		report.UncatalogedConversations = report.OrphanConversations
	}
	var err error
	report.Generation, err = generation(ctx, q)
	if err != nil {
		return IntegrityReport{}, err
	}
	report.EventContentHash, err = a.cachedEventContentHash(ctx, q)
	if err != nil {
		return IntegrityReport{}, err
	}
	return report, nil
}

// cachedEventContentHash combines per-conversation hashes, reading bodies
// only for conversations whose stamp moved since the cache last saw them. Its
// reads happen in the audit's snapshot, so every stamp and hash it stores
// describes the same committed state.
func (a *IntegrityAuditor) cachedEventContentHash(ctx context.Context, q dbx.Querier) (string, error) {
	cache := a.cache
	if cache == nil {
		cache = NewContentReceiptCache()
	} else {
		select {
		case cache.gate <- struct{}{}:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		defer func() { <-cache.gate }()
	}
	stamps, err := sessionContentStamps(ctx, q)
	if err != nil {
		return "", err
	}
	cache.mu.Lock()
	known := cache.sessions
	cache.mu.Unlock()
	next := make(map[string]sessionContentHash, len(stamps))
	rehashed := 0
	for _, stamp := range stamps {
		if prior, ok := known[stamp.sessionID]; ok && prior.stamp == stamp.stamp {
			next[stamp.sessionID] = prior
			continue
		}
		hash, err := sessionEventContentHash(ctx, q, stamp.sessionID)
		if err != nil {
			return "", err
		}
		next[stamp.sessionID] = sessionContentHash{stamp: stamp.stamp, hash: hash}
		rehashed++
	}
	cache.mu.Lock()
	cache.sessions, cache.rehashed = next, rehashed
	cache.mu.Unlock()
	h := sha256.New()
	for _, stamp := range stamps {
		_, _ = fmt.Fprintf(h, "%s\x00%s\n", stamp.sessionID, next[stamp.sessionID].hash)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

type sessionContentStamp struct {
	sessionID, stamp string
}

// sessionContentStamps reads every conversation's stamp, ordered by session,
// from the covering session index.
func sessionContentStamps(ctx context.Context, q dbx.Querier) ([]sessionContentStamp, error) {
	rows, err := q.QueryContext(ctx, `SELECT session_id, COUNT(*), MAX(rowid), SUM(sequence) FROM conversation_events GROUP BY session_id ORDER BY session_id`)
	if err != nil {
		return nil, fmt.Errorf("stamp event content: %w", err)
	}
	defer rows.Close()
	var stamps []sessionContentStamp
	for rows.Next() {
		var sessionID string
		var count, maxRowID, sequences int64
		if err := rows.Scan(&sessionID, &count, &maxRowID, &sequences); err != nil {
			return nil, err
		}
		stamps = append(stamps, sessionContentStamp{sessionID: sessionID, stamp: fmt.Sprintf("%d:%d:%d", count, maxRowID, sequences)})
	}
	return stamps, rows.Err()
}

// Generation fingerprints inventory identities without scanning message bodies.
// Its four inventory reads share one snapshot: read separately, a write landing
// between them yields a generation no committed state ever had.
func (a *IntegrityAuditor) Generation(ctx context.Context) (string, error) {
	var value string
	err := dbx.ReadSnapshot(ctx, a.db, func(q dbx.Querier) error {
		var err error
		value, err = generation(ctx, q)
		return err
	})
	return value, err
}

func generation(ctx context.Context, q dbx.Querier) (string, error) {
	ids, err := inventoryIDs(ctx, q)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

// sessionEventContentHash is a sanitized conservation receipt for one
// conversation: it fingerprints event identity and content without returning
// or logging the content itself.
func sessionEventContentHash(ctx context.Context, q dbx.Querier, sessionID string) (string, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, sequence, role, created_at, text FROM conversation_events WHERE session_id = ? ORDER BY sequence, id`, sessionID)
	if err != nil {
		return "", fmt.Errorf("inventory event content: %w", err)
	}
	defer rows.Close()
	h := sha256.New()
	for rows.Next() {
		var id, role, createdAt, text string
		var sequence int64
		if err := rows.Scan(&id, &sequence, &role, &createdAt, &text); err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(h, "%s\x00%d\x00%s\x00%s\x00%s\n", id, sequence, role, createdAt, text)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func inventoryIDs(ctx context.Context, q dbx.Querier) ([]string, error) {
	queries := []struct{ prefix, query string }{
		{"session", `SELECT id FROM sessions`},
		{"conversation", `SELECT session_id FROM conversation_sessions`},
		{"checkpoint", `SELECT source || ':' || source_key || ':' || web_console_session_id FROM agent_transcript_checkpoints`},
		{"pane", `SELECT session_id FROM workspace_panes`},
	}
	var ids []string
	for _, inventory := range queries {
		rows, err := q.QueryContext(ctx, inventory.query)
		if err != nil {
			return nil, fmt.Errorf("inventory %s: %w", inventory.prefix, err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, inventory.prefix+":"+id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	sort.Strings(ids)
	return ids, nil
}
