package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage"
)

// Rehearse a table swap in a disposable database. Live adoption also needs a
// writer fence and a restore-qualified backup. A startup migration would
// rewrite the event table and its indexes before Web Console could serve.
func TestConversationRowIDAliasRehearsal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		orphan bool
	}{
		{name: "explicit identity preserves search and cursor through vacuum"},
		{name: "foreign key failure rolls back the swap", orphan: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "rehearsal.db")
			dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
			if err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite", dsn)
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			if err := initSchema(ctx, db); err != nil {
				t.Fatal(err)
			}
			if err := ensureConversationProjectionIndex(ctx, db); err != nil {
				t.Fatal(err)
			}
			execRowIDRehearsal(t, db, `INSERT INTO conversation_sessions (session_id) VALUES ('kept'), ('second')`)
			execRowIDRehearsal(t, db, `INSERT INTO conversation_events
				(rowid, id, session_id, source, role, text, created_at, sequence, delivery_state, tts_state, consumption_state) VALUES
				(31, 'first', 'kept', 'test', 'user', 'proof first', '2026-01-01T00:00:00Z', 1, 'pending', 'idle', 'unseen'),
				(509, 'next', 'second', 'test', 'assistant', 'proof next', '2026-01-01T00:00:01Z', 1, 'pending', 'idle', 'unseen')`)
			execRowIDRehearsal(t, db, `INSERT INTO web_console_migrations (name, last_rowid) VALUES (?, 31)`, conversationClassificationMigration)
			wantDocIDs := []int64{31, 509}
			if tc.orphan {
				execRowIDRehearsal(t, db, `PRAGMA foreign_keys = OFF;
					INSERT INTO conversation_events
					(rowid, id, session_id, source, role, text, created_at, sequence, delivery_state, tts_state, consumption_state)
					VALUES (901, 'orphan', 'missing', 'test', 'user', 'proof orphan', '2026-01-01T00:00:02Z', 1, 'pending', 'idle', 'unseen');
					PRAGMA foreign_keys = ON`)
				wantDocIDs = append(wantDocIDs, 901)
			}

			err = exerciseConversationRowIDAliasSwap(ctx, db)
			if tc.orphan {
				if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
					t.Fatalf("foreign-key refusal = %v", err)
				}
				var stageExists bool
				if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM sqlite_schema WHERE name = 'conversation_events_rowid_new')`).Scan(&stageExists); err != nil || stageExists {
					t.Fatalf("staging table after rollback = %t, %v", stageExists, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				execRowIDRehearsal(t, db, `PRAGMA auto_vacuum = INCREMENTAL; VACUUM`)
				assertConsoleSQLiteVacuumMode(t, db, "incremental")
				execRowIDRehearsal(t, db, `INSERT INTO conversation_events_fts(conversation_events_fts, rank) VALUES ('integrity-check', 1)`)
			}

			var gotDocIDs []int64
			rows, err := db.QueryContext(ctx, `SELECT e.rowid FROM conversation_events_fts
				JOIN conversation_events e ON e.rowid = conversation_events_fts.rowid
				WHERE conversation_events_fts MATCH 'proof' ORDER BY e.rowid`)
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				var rowID int64
				if err := rows.Scan(&rowID); err != nil {
					_ = rows.Close()
					t.Fatal(err)
				}
				gotDocIDs = append(gotDocIDs, rowID)
			}
			if err := rows.Close(); err != nil {
				t.Fatal(err)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(gotDocIDs, wantDocIDs) {
				t.Fatalf("search document IDs = %v, want %v", gotDocIDs, wantDocIDs)
			}

			backfill := newConversationClassificationBackfill(db)
			cursor, completed, err := backfill.progress(ctx)
			if err != nil || cursor != 31 || completed {
				t.Fatalf("classification cursor = %d, completed %t, %v", cursor, completed, err)
			}
			if !tc.orphan {
				backfill.batch = 1
				report := classificationBackfillReport{ByKind: make(map[string]int64)}
				last, count, err := backfill.classifyBatch(ctx, cursor, &report)
				if err != nil || last != 509 || count != 1 {
					t.Fatalf("classification resumed at %d (%d rows), %v", last, count, err)
				}
				execRowIDRehearsal(t, db, `UPDATE conversation_events SET text = 'edited' WHERE id = 'next'`)
				var editedDocID int64
				if err := db.QueryRowContext(ctx, `SELECT rowid FROM conversation_events_fts WHERE conversation_events_fts MATCH 'edited'`).Scan(&editedDocID); err != nil || editedDocID != 509 {
					t.Fatalf("updated search document ID = %d, %v", editedDocID, err)
				}
			}
		})
	}
}

func execRowIDRehearsal(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatal(err)
	}
}

func exerciseConversationRowIDAliasSwap(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Keep the source's checks, defaults, foreign keys and historical column
	// ordering. A TEXT PRIMARY KEY here permits NULL, so UNIQUE preserves
	// the existing event-ID constraint when the integer alias becomes the key.
	var ddl, columns string
	if err := tx.QueryRowContext(ctx, `SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = 'conversation_events'`).Scan(&ddl); err != nil {
		return err
	}
	_, body, ok := strings.Cut(ddl, "(\n    id TEXT PRIMARY KEY,")
	if !ok {
		return errors.New("rehearsal expected the shipped conversation_events definition")
	}
	if err := tx.QueryRowContext(ctx, `SELECT group_concat('"' || replace(name, '"', '""') || '"', ', ') FROM
		(SELECT name FROM pragma_table_xinfo('conversation_events') WHERE hidden = 0 ORDER BY cid)`).Scan(&columns); err != nil {
		return err
	}
	sqlRows, err := tx.QueryContext(ctx, `SELECT sql FROM sqlite_schema WHERE tbl_name = 'conversation_events' AND type IN ('index','trigger') AND sql IS NOT NULL ORDER BY type, name`)
	if err != nil {
		return err
	}
	var recreate []string
	for sqlRows.Next() {
		var statement string
		if err := sqlRows.Scan(&statement); err != nil {
			_ = sqlRows.Close()
			return err
		}
		recreate = append(recreate, statement)
	}
	if err := sqlRows.Close(); err != nil {
		return err
	}
	if err := sqlRows.Err(); err != nil {
		return err
	}

	fields := "rowid, " + columns
	for _, statement := range []string{
		"CREATE TABLE conversation_events_rowid_new (rowid INTEGER PRIMARY KEY, id TEXT UNIQUE," + body,
		"INSERT INTO conversation_events_rowid_new (" + fields + ") SELECT " + fields + " FROM conversation_events",
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	sourceRows := "SELECT " + fields + " FROM conversation_events"
	stagedRows := "SELECT " + fields + " FROM conversation_events_rowid_new"
	var changed bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS("+sourceRows+" EXCEPT "+stagedRows+
		") OR EXISTS("+stagedRows+" EXCEPT "+sourceRows+")").Scan(&changed); err != nil {
		return err
	}
	if changed {
		return errors.New("staged events differ from the original row IDs or values")
	}

	swap := []string{
		"DROP TABLE conversation_events",
		"ALTER TABLE conversation_events_rowid_new RENAME TO conversation_events",
	}
	for _, statement := range append(swap, recreate...) {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}
