package continuity

import (
	"context"
	"testing"
	"time"
)

func TestSearchLocalFindsAntigravityArchiveWithoutSessionMetadata(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, source TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
		`INSERT INTO conversation_events(id,session_id,source,sequence,role,created_at,text) VALUES ('native-agy','native-antigravity','antigravity_archive',1,'assistant','2026-11-24T10:00:00Z','antigravity continuity marker')`,
		`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`,
	})
	matches, _, total, distinct, err := SearchLocalWithOptions(context.Background(), db, SearchOptions{
		Query: "antigravity continuity marker", AgentType: "antigravity", State: "archived", Limit: 10,
	})
	if err != nil || total != 1 || distinct != 1 || len(matches) != 1 || matches[0].EventID != "native-agy" {
		t.Fatalf("Antigravity archive matches=%+v total=%d distinct=%d err=%v", matches, total, distinct, err)
	}
	if _, _, _, _, err := SearchLocalWithOptions(context.Background(), db, SearchOptions{
		Query: "antigravity continuity marker", AgentType: "antigravity", CreatedAfter: time.Now().Add(time.Hour), Limit: 10,
	}); err != nil {
		t.Fatalf("Antigravity archive date-filter search: %v", err)
	}
}
