package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

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

type IntegrityAuditor struct{ db dbx.Handle }

func NewIntegrityAuditor(db dbx.Handle) *IntegrityAuditor { return &IntegrityAuditor{db: db} }

func (a *IntegrityAuditor) Audit(ctx context.Context) (IntegrityReport, error) {
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
		if err := a.db.QueryRowContext(ctx, queries[i].query).Scan(&values[i]); err != nil {
			return IntegrityReport{}, fmt.Errorf("audit %s: %w", queries[i].name, err)
		}
	}
	report := IntegrityReport{Sessions: values[0], ConversationSessions: values[1], ConversationEvents: values[2], Checkpoints: values[3], WorkspacePanes: values[4], OrphanConversations: values[5], OrphanCheckpoints: values[6], OrphanWorkspacePanes: values[7]}
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversation_sessions c LEFT JOIN sessions s ON s.id=c.session_id LEFT JOIN conversation_catalog k ON k.session_id=c.session_id WHERE s.id IS NULL AND k.session_id IS NULL`).Scan(&report.UncatalogedConversations); err != nil {
		// Keep read-only integrity audits usable against pre-catalog disposable
		// fixtures; production schemas always register conversation_catalog.
		report.UncatalogedConversations = report.OrphanConversations
	}
	ids, err := a.inventoryIDs(ctx)
	if err != nil {
		return IntegrityReport{}, err
	}
	digest := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	report.Generation = "sha256:" + hex.EncodeToString(digest[:])
	report.EventContentHash, err = a.eventContentHash(ctx)
	if err != nil {
		return IntegrityReport{}, err
	}
	return report, nil
}

// eventContentHash is a sanitized conservation receipt: it fingerprints event
// identity and content without returning or logging the content itself.
func (a *IntegrityAuditor) eventContentHash(ctx context.Context) (string, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, session_id, sequence, role, created_at, text FROM conversation_events ORDER BY id`)
	if err != nil {
		return "", fmt.Errorf("inventory event content: %w", err)
	}
	defer rows.Close()
	h := sha256.New()
	for rows.Next() {
		var id, sessionID, role, createdAt, text string
		var sequence int64
		if err := rows.Scan(&id, &sessionID, &sequence, &role, &createdAt, &text); err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00%d\x00%s\x00%s\x00%s\n", id, sessionID, sequence, role, createdAt, text)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func (a *IntegrityAuditor) inventoryIDs(ctx context.Context) ([]string, error) {
	queries := []struct{ prefix, query string }{
		{"session", `SELECT id FROM sessions`},
		{"conversation", `SELECT session_id FROM conversation_sessions`},
		{"checkpoint", `SELECT source || ':' || source_key || ':' || web_console_session_id FROM agent_transcript_checkpoints`},
		{"pane", `SELECT session_id FROM workspace_panes`},
	}
	var ids []string
	for _, q := range queries {
		rows, err := a.db.QueryContext(ctx, q.query)
		if err != nil {
			return nil, fmt.Errorf("inventory %s: %w", q.prefix, err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, q.prefix+":"+id)
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
