package continuity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// Compare the full audit previously used by every health probe with the
// catalog-link observation health actually needs. The full audit is unchanged.
func BenchmarkContinuityProbeWithRetainedContent(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE sessions(id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_sessions(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY)`,
		`CREATE TABLE conversation_events(id TEXT PRIMARY KEY,session_id TEXT,sequence INTEGER,role TEXT,created_at TEXT,text TEXT)`,
		`CREATE TABLE agent_transcript_checkpoints(source TEXT,source_key TEXT,web_console_session_id TEXT)`,
		`CREATE TABLE workspace_panes(session_id TEXT PRIMARY KEY)`,
		`INSERT INTO sessions VALUES ('known')`,
		`INSERT INTO conversation_sessions VALUES ('known')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			b.Fatal(err)
		}
	}
	tx, err := db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	statement, err := tx.Prepare(`INSERT INTO conversation_events VALUES (?,'known',?,'assistant','2026-11-05T00:00:00Z',?)`)
	if err != nil {
		b.Fatal(err)
	}
	text := strings.Repeat("x", 32<<10)
	for i := range 1000 {
		if _, err := statement.Exec(fmt.Sprintf("event-%04d", i), i, text); err != nil {
			b.Fatal(err)
		}
	}
	statement.Close()
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	auditor := NewIntegrityAuditor(db)
	b.Run("full-audit", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := auditor.Audit(context.Background()); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("health-observation", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := auditor.ObserveHealth(context.Background()); err != nil {
				b.Fatal(err)
			}
		}
	})
}
