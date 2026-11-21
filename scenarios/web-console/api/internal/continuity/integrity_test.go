package continuity

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestIntegrityAuditReportsOrphansWithoutMutatingSources(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_sessions(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE TABLE agent_transcript_checkpoints(source TEXT, source_key TEXT, web_console_session_id TEXT)`,
		`CREATE TABLE workspace_panes(session_id TEXT PRIMARY KEY)`,
		`INSERT INTO sessions VALUES ('known')`, `INSERT INTO conversation_sessions VALUES ('known'), ('orphan')`,
		`INSERT INTO conversation_events VALUES ('e', 'orphan', 1, 'user', '2026-01-01T00:00:00Z', 'retained')`, `INSERT INTO agent_transcript_checkpoints VALUES ('codex', 'x', 'orphan')`,
		`INSERT INTO workspace_panes VALUES ('orphan')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	a := NewIntegrityAuditor(db)
	report, err := a.Audit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.OrphanConversations != 1 || report.OrphanCheckpoints != 1 || report.OrphanWorkspacePanes != 1 {
		t.Fatalf("report=%+v", report)
	}
	after, err := a.Audit(context.Background())
	if err != nil || after.EventContentHash != report.EventContentHash {
		t.Fatalf("audit changed conservation hash: before=%+v after=%+v err=%v", report, after, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("audit mutated source events")
	}
}
