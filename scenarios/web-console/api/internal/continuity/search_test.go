package continuity

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSearchLocalFindsEventsAcrossLifecycleStates(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
		`INSERT INTO sessions VALUES ('live','','live','codex','/workspace/live'), ('arch','2026-01-01T00:00:00Z','live','claude','/workspace/archive')`,
		`INSERT INTO conversation_events VALUES ('e1','live',1,'user','2026-01-01T00:00:00Z','plan-manager live'), ('e2','arch',1,'assistant','2026-01-02T00:00:00Z','plan-manager archived')`,
		`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	matches, _, total, distinct, err := SearchLocal(context.Background(), db, "plan-manager", "any", "", "", time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || distinct != 2 || len(matches) != 2 {
		t.Fatalf("matches=%+v total=%d distinct=%d", matches, total, distinct)
	}
	live, _, total, _, err := SearchLocal(context.Background(), db, "plan-manager", "live", "", "", time.Time{}, 10)
	if err != nil || total != 1 || len(live) != 1 || live[0].SessionID != "live" {
		t.Fatalf("live matches=%+v total=%d err=%v", live, total, err)
	}
	archived, _, total, _, err := SearchLocal(context.Background(), db, "plan-manager", "archived", "", "", time.Time{}, 10)
	if err != nil || total != 1 || len(archived) != 1 || archived[0].SessionID != "arch" {
		t.Fatalf("archived matches=%+v total=%d err=%v", archived, total, err)
	}
	workspace, _, total, _, err := SearchLocal(context.Background(), db, "plan-manager", "any", "", "/workspace/archive", time.Time{}, 10)
	if err != nil || total != 1 || len(workspace) != 1 || workspace[0].SessionID != "arch" {
		t.Fatalf("workspace matches=%+v total=%d err=%v", workspace, total, err)
	}
}

func TestSearchLocalIncidentRegressionFindsOrphanedCodexThread(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const paneID = "a7e71c3c-e422-4c89-916a-03f92906fb89"
	const threadID = "01a06a6b-88da-7422-b391-bb59c5f5e5e0"
	for i := 1; i <= 88; i++ {
		text := "retained discussion"
		if i == 88 {
			text = "plan-manager Agent Manager Search Hub continuity"
		}
		if _, err := db.Exec(`INSERT INTO conversation_events VALUES (?, ?, ?, 'assistant', ?, ?)`, fmt.Sprintf("incident-event-%02d", i), paneID, i, "2026-09-03T23:17:07Z", text); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`); err != nil {
		t.Fatal(err)
	}
	matches, _, total, distinct, err := SearchLocal(context.Background(), db, "plan-manager", "any", "", "", time.Time{}, 10)
	if err != nil || total != 1 || distinct != 1 || len(matches) != 1 || matches[0].SessionID != paneID {
		t.Fatalf("matches=%+v total=%d distinct=%d err=%v", matches, total, distinct, err)
	}
	if threadID == "" {
		t.Fatal("incident thread identity must remain a named regression anchor")
	}
}

func TestSearchLocalIncidentIdentityFindsPaneAndThread(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, agent_session_id TEXT, lifecycle_state TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const paneID = "a7e71c3c-e422-4c89-916a-03f92906fb89"
	const threadID = "01a06a6b-88da-7422-b391-bb59c5f5e5e0"
	if _, err := db.Exec(`INSERT INTO conversation_events VALUES ('incident-event','` + paneID + `',88,'assistant','2026-09-03T23:17:07Z','retained discussion')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_catalog VALUES (?, ?, 'recoverable', 'codex', '')`, paneID, threadID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`); err != nil {
		t.Fatal(err)
	}

	for _, query := range []string{paneID, threadID, "a7e71c3c"} {
		matches, _, total, distinct, err := SearchLocal(context.Background(), db, query, "recoverable", "", "", time.Time{}, 10)
		if err != nil || total != 1 || distinct != 1 || len(matches) != 1 || matches[0].SessionID != paneID || matches[0].LifecycleState != "recoverable" {
			t.Fatalf("query=%q matches=%+v total=%d distinct=%d err=%v", query, matches, total, distinct, err)
		}
	}
}

func TestSearchLocalUsesCatalogMetadataWhenSessionRowIsMissing(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, agent_session_id TEXT, lifecycle_state TEXT, agent_type TEXT, cwd TEXT, original_title TEXT, current_title TEXT, topic_summary TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const paneID = "orphan-pane"
	if _, err := db.Exec(`INSERT INTO conversation_events VALUES ('orphan-event', ?, 1, 'assistant', '2026-09-06T00:00:00Z', 'retained body')`, paneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_catalog VALUES (?, 'native-thread', 'recoverable', 'codex', '/workspace/orphan', 'Onboarding', 'Plan Manager discussion', 'durable continuity topic')`, paneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_events VALUES ('other-event', 'other-pane', 1, 'assistant', '2026-09-06T00:00:00Z', 'other body')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_catalog VALUES ('other-pane', 'other-thread', 'recoverable', 'claude', '/workspace/other', 'Other topic', 'Unrelated conversation', 'unrelated')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`); err != nil {
		t.Fatal(err)
	}

	byTitle, _, total, _, err := SearchLocalWithOptions(context.Background(), db, SearchOptions{Title: "Plan Manager", State: "recoverable", Limit: 10})
	if err != nil || total != 1 || len(byTitle) != 1 || byTitle[0].SessionID != paneID {
		t.Fatalf("title matches=%+v total=%d err=%v", byTitle, total, err)
	}
	byTopic, _, total, _, err := SearchLocalWithOptions(context.Background(), db, SearchOptions{Query: "continuity", TopicSummary: "durable", State: "recoverable", AgentSessionID: "native-thread", Limit: 10})
	if err != nil || total != 1 || len(byTopic) != 1 || byTopic[0].SessionID != paneID {
		t.Fatalf("topic matches=%+v total=%d err=%v", byTopic, total, err)
	}
}
