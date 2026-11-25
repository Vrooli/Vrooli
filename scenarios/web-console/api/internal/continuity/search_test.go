package continuity

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSearchLocalFindsEventsAcrossLifecycleStates(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
		`INSERT INTO sessions VALUES ('live','','live','codex','/workspace/live'), ('arch','2026-01-01T00:00:00Z','live','claude','/workspace/archive')`,
		`INSERT INTO conversation_events(id,session_id,sequence,role,created_at,text) VALUES ('e1','live',1,'user','2026-01-01T00:00:00Z','plan-manager live'), ('e2','arch',1,'assistant','2026-01-02T00:00:00Z','plan-manager archived')`,
		`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`,
	})
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

func TestSearchLocalHarnessFilterFindsNativeArchiveWithoutSessionMetadata(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, source TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
		`INSERT INTO conversation_events(id,session_id,source,sequence,role,created_at,text) VALUES ('native-c','native-claude','claude_archive',1,'assistant','2026-01-01T00:00:00Z','native harness filter marker'), ('native-x','native-codex','codex_archive',1,'assistant','2026-01-01T00:00:00Z','native harness filter marker'), ('native-q','native-qwen','qwen-code_archive',1,'assistant','2026-01-01T00:00:00Z','native harness filter marker')`,
		`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`,
	})
	matches, _, total, distinct, err := SearchLocalWithOptions(context.Background(), db, SearchOptions{Query: "native harness filter marker", AgentType: "claude", State: "archived", Limit: 10})
	if err != nil || total != 1 || distinct != 1 || len(matches) != 1 || matches[0].EventID != "native-c" || matches[0].LifecycleState != "archived" {
		t.Fatalf("native harness-filtered matches=%+v total=%d distinct=%d err=%v", matches, total, distinct, err)
	}
	qwenMatches, _, qwenTotal, qwenDistinct, qwenErr := SearchLocalWithOptions(context.Background(), db, SearchOptions{Query: "native harness filter marker", AgentType: "qwen-code", State: "archived", Limit: 10})
	if qwenErr != nil || qwenTotal != 1 || qwenDistinct != 1 || len(qwenMatches) != 1 || qwenMatches[0].EventID != "native-q" {
		t.Fatalf("Qwen harness-filtered matches=%+v total=%d distinct=%d err=%v", qwenMatches, qwenTotal, qwenDistinct, qwenErr)
	}
}

func TestSearchLocalDateFilterUsesCanonicalNativeOccurrencePrecision(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, source TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
		`INSERT INTO conversation_events(id,session_id,source,sequence,role,created_at,occurred_at,text) VALUES
		 ('before-cutoff','native-claude','claude_archive',1,'assistant','2026-11-23T12:00:00Z','2026-11-23T12:00:00.000000000Z','native date boundary marker'),
		 ('after-cutoff','native-claude','claude_archive',2,'assistant','2026-10-01T00:00:00Z','2026-11-23T12:00:01.000000000Z','native date boundary marker'),
		 ('unknown-time','native-claude','claude_archive',3,'assistant','2026-11-23T12:00:05Z',NULL,'native date boundary marker'),
		 ('ordinary-before','ordinary-session','codex_app_server',1,'assistant','2026-11-23T12:00:00Z',NULL,'native date boundary marker'),
		 ('ordinary-after','ordinary-session','codex_app_server',2,'assistant','2026-11-23T12:00:01Z',NULL,'native date boundary marker'),
		 ('qwen-before','native-qwen','qwen-code_archive',1,'assistant','2026-11-23T12:00:00Z','2026-11-23T12:00:00.000000000Z','native date boundary marker'),
		 ('qwen-after','native-qwen','qwen-code_archive',2,'assistant','2026-10-01T00:00:00Z','2026-11-23T12:00:01.000000000Z','native date boundary marker'),
		 ('qwen-unknown','native-qwen','qwen-code_archive',3,'assistant','2026-11-23T12:00:05Z',NULL,'native date boundary marker')`,
		`INSERT INTO conversation_events_fts(rowid,text) SELECT rowid,text FROM conversation_events`,
	})
	cutoff := time.Date(2026, 11, 23, 12, 0, 0, 500_000_000, time.UTC)
	matches, _, total, _, err := SearchLocalWithOptions(context.Background(), db, SearchOptions{Query: "native date boundary marker", State: "any", CreatedAfter: cutoff, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(matches) != 3 {
		t.Fatalf("date-filtered local matches=%+v total=%d, want after-cutoff Claude/Qwen plus ordinary capture", matches, total)
	}
	ids := map[string]bool{}
	for _, match := range matches {
		ids[match.EventID] = true
	}
	if !ids["after-cutoff"] || !ids["ordinary-after"] || !ids["qwen-after"] || ids["before-cutoff"] || ids["unknown-time"] || ids["ordinary-before"] || ids["qwen-before"] || ids["qwen-unknown"] {
		t.Fatalf("date filter violated source-time precision/unknown exclusion: %+v", matches)
	}
}

func TestSearchLocalIncidentRegressionFindsOrphanedCodexThread(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
	})
	const paneID = "a7e71c3c-e422-4c89-916a-03f92906fb89"
	const threadID = "01a06a6b-88da-7422-b391-bb59c5f5e5e0"
	for i := 1; i <= 88; i++ {
		text := "retained discussion"
		if i == 88 {
			text = "plan-manager Agent Manager Search Hub continuity"
		}
		if _, err := db.Exec(`INSERT INTO conversation_events(id,session_id,sequence,role,created_at,text) VALUES (?, ?, ?, 'assistant', ?, ?)`, fmt.Sprintf("incident-event-%02d", i), paneID, i, "2026-09-03T23:17:07Z", text); err != nil {
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
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, agent_session_id TEXT, lifecycle_state TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
	})
	const paneID = "a7e71c3c-e422-4c89-916a-03f92906fb89"
	const threadID = "01a06a6b-88da-7422-b391-bb59c5f5e5e0"
	if _, err := db.Exec(`INSERT INTO conversation_events(id,session_id,sequence,role,created_at,text) VALUES ('incident-event','` + paneID + `',88,'assistant','2026-09-03T23:17:07Z','retained discussion')`); err != nil {
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
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY, archived_at TEXT, status TEXT, agent_type TEXT, cwd TEXT)`,
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, agent_session_id TEXT, lifecycle_state TEXT, agent_type TEXT, cwd TEXT, original_title TEXT, current_title TEXT, topic_summary TEXT)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, occurred_at TEXT, text TEXT)`,
		`CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid')`,
	})
	const paneID = "orphan-pane"
	if _, err := db.Exec(`INSERT INTO conversation_events(id,session_id,sequence,role,created_at,text) VALUES ('orphan-event', ?, 1, 'assistant', '2026-09-06T00:00:00Z', 'retained body')`, paneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_catalog VALUES (?, 'native-thread', 'recoverable', 'codex', '/workspace/orphan', 'Onboarding', 'Plan Manager discussion', 'durable continuity topic')`, paneID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_events(id,session_id,sequence,role,created_at,text) VALUES ('other-event', 'other-pane', 1, 'assistant', '2026-09-06T00:00:00Z', 'other body')`); err != nil {
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
