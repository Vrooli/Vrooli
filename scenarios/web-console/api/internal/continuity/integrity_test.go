package continuity

import (
	"context"
	"testing"
)

func TestIntegrityAuditReportsOrphansWithoutMutatingSources(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_sessions(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE TABLE agent_transcript_checkpoints(source TEXT, source_key TEXT, web_console_session_id TEXT)`,
		`CREATE TABLE workspace_panes(session_id TEXT PRIMARY KEY)`,
		`INSERT INTO sessions VALUES ('known')`, `INSERT INTO conversation_sessions VALUES ('known'), ('orphan')`,
		`INSERT INTO conversation_events VALUES ('e', 'orphan', 1, 'user', '2026-01-01T00:00:00Z', 'retained')`, `INSERT INTO agent_transcript_checkpoints VALUES ('codex', 'x', 'orphan')`,
		`INSERT INTO workspace_panes VALUES ('orphan')`,
	})
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

// The cache relies on event bodies never being rewritten in place. Rewriting
// one anyway is what lets the test tell a reused hash from a recomputed one:
// a reused hash still describes the old body. Only the conversation whose
// events changed is read again.
func TestIntegrityAuditRehashesOnlyConversationsWhoseEventsChanged(t *testing.T) {
	db := newContinuityTestDB(t, ":memory:")
	db.SetMaxOpenConns(1)
	execContinuitySQL(t, db, []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_sessions(session_id TEXT PRIMARY KEY, last_sequence INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY, session_id TEXT, sequence INTEGER, role TEXT, created_at TEXT, text TEXT)`,
		`CREATE TABLE agent_transcript_checkpoints(source TEXT, source_key TEXT, web_console_session_id TEXT)`,
		`CREATE TABLE workspace_panes(session_id TEXT PRIMARY KEY)`,
		`INSERT INTO sessions VALUES ('s'), ('quiet')`, `INSERT INTO conversation_sessions VALUES ('s', 2), ('quiet', 1)`,
		`INSERT INTO conversation_events VALUES ('e1', 's', 1, 'user', '2026-01-01T00:00:00Z', 'first'), ('e2', 's', 2, 'assistant', '2026-01-01T00:00:01Z', 'second'),
			('q1', 'quiet', 1, 'user', '2026-01-01T00:00:00Z', 'untouched')`,
	})
	cache := NewContentReceiptCache()
	hash := func(auditor *IntegrityAuditor) string {
		t.Helper()
		report, err := auditor.Audit(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		return report.EventContentHash
	}
	cached := func(wantRehashed int) string {
		t.Helper()
		value := hash(NewIntegrityAuditor(db).WithContentCache(cache))
		if cache.rehashed != wantRehashed {
			t.Fatalf("audit read bodies of %d conversations, want %d", cache.rehashed, wantRehashed)
		}
		return value
	}
	fresh := func() string { return hash(NewIntegrityAuditor(db)) }

	first := cached(2)
	execContinuitySQL(t, db, []string{`UPDATE conversation_events SET text = 'rewritten' WHERE id IN ('e1', 'q1')`})
	if again := cached(0); again != first || fresh() == first {
		t.Fatalf("unchanged events: cached %q -> %q, fresh %q; want the cached hash reused", first, again, fresh())
	}
	execContinuitySQL(t, db, []string{
		`UPDATE conversation_sessions SET last_sequence = 3 WHERE session_id = 's'`,
		`INSERT INTO conversation_events VALUES ('e3', 's', 3, 'user', '2026-01-01T00:00:02Z', 'third')`,
	})
	appended := cached(1)
	if appended == first || appended == fresh() {
		t.Fatalf("append must rehash only its conversation: cached %q, fresh %q", appended, fresh())
	}
	execContinuitySQL(t, db, []string{`UPDATE conversation_events SET text = 'untouched' WHERE id = 'q1'`})
	if settled := cached(0); settled != fresh() {
		t.Fatalf("with the quiet conversation restored, cached %q must match fresh %q", settled, fresh())
	}
	execContinuitySQL(t, db, []string{`DELETE FROM conversation_events WHERE id = 'e1'`})
	if pruned := cached(1); pruned != fresh() {
		t.Fatalf("delete must rehash its conversation: cached %q, fresh %q", pruned, fresh())
	}
}
