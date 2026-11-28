package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/dbx"
)

// conversationClassificationMigration names the classification backfill's
// durable progress row in web_console_migrations. Bump the version when the
// recognisers in agenttranscript/message-classes.json change in a way that
// should reclassify history; the pass is idempotent, so re-running only
// rewrites rows whose classification changed.
const conversationClassificationMigration = "conversation_event_classification_v1"

// conversationClassificationBackfill classifies rows stored before rows
// carried a kind and an author. It runs in the background after boot in
// bounded transactions (a rowid cursor per batch, committed with the batch),
// so startup is never held by a multi-gigabyte history, an interrupted pass
// resumes where it stopped, and nothing is ever deleted.
type conversationClassificationBackfill struct {
	db    dbx.Handle
	batch int
	pause time.Duration
	// textPrefix bounds how much of each row's text is read: every recogniser
	// matches near the start of the text.
	textPrefix int
	logf       func(string, ...any)
}

// classificationBackfillReport counts what one pass changed.
type classificationBackfillReport struct {
	Scanned       int64
	Changed       int64
	ByKind        map[string]int64
	LaunchPrompts int64
}

func newConversationClassificationBackfill(db dbx.Handle) *conversationClassificationBackfill {
	return &conversationClassificationBackfill{db: db, batch: 500, pause: 20 * time.Millisecond, textPrefix: 4096, logf: log.Printf}
}

// StartConversationClassificationBackfill builds the projection index and
// classifies existing history in the background. Production-only, like the
// native archive index, so isolated test servers never touch history.
func (srv *Server) StartConversationClassificationBackfill() {
	if srv == nil || srv.db == nil {
		return
	}
	backfill := newConversationClassificationBackfill(srv.db.Primary())
	go func() {
		started := time.Now()
		report, err := backfill.Run(context.Background())
		if err != nil {
			log.Printf("conversation-classification: stopped after %d rows: %v", report.Scanned, err)
			return
		}
		if report.Scanned > 0 {
			log.Printf("conversation-classification: classified %d rows (%d changed, %v, %d launch prompts) in %s",
				report.Scanned, report.Changed, report.ByKind, report.LaunchPrompts, time.Since(started).Round(time.Second))
		}
	}()
}

// Run builds the projection index, then classifies every row after the
// durable cursor, then attributes programmatic sessions' launch prompts.
func (b *conversationClassificationBackfill) Run(ctx context.Context) (classificationBackfillReport, error) {
	report := classificationBackfillReport{ByKind: map[string]int64{}}
	if err := ensureConversationProjectionIndex(ctx, b.db); err != nil {
		return report, fmt.Errorf("projection index: %w", err)
	}
	cursor, completed, err := b.progress(ctx)
	if err != nil || completed {
		return report, err
	}
	for {
		next, scanned, err := b.classifyBatch(ctx, cursor, &report)
		if err != nil {
			return report, err
		}
		if scanned == 0 {
			break
		}
		cursor = next
		if b.pause > 0 {
			select {
			case <-ctx.Done():
				return report, ctx.Err()
			case <-time.After(b.pause):
			}
		}
	}
	launch, err := b.attributeLaunchPrompts(ctx)
	report.LaunchPrompts = launch
	if err != nil {
		return report, err
	}
	if _, err = b.db.ExecContext(ctx, `INSERT INTO web_console_migrations(name, last_rowid, completed_at) VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET last_rowid = excluded.last_rowid, completed_at = excluded.completed_at`,
		conversationClassificationMigration, cursor, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return report, err
	}
	// Most rows just changed kind, so the planner's statistics for the new
	// indexes are refreshed once. The sampling limit bounds the time the
	// single writer is held on a multi-gigabyte table.
	_, err = b.db.ExecContext(ctx, `PRAGMA analysis_limit=400; ANALYZE conversation_events;`)
	return report, err
}

func (b *conversationClassificationBackfill) progress(ctx context.Context) (int64, bool, error) {
	var cursor int64
	var completed string
	err := b.db.QueryRowContext(ctx, `SELECT last_rowid, completed_at FROM web_console_migrations WHERE name = ?`, conversationClassificationMigration).Scan(&cursor, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return cursor, completed != "", err
}

type storedClassification struct {
	rowid                                    int64
	sessionID, source, role, text            string
	kind, author, authorName, label, created string
	threadID                                 string
}

// classifiedRow returns the row's classification, or false when the stored
// one already matches. A row already filed under a thread keeps it.
func classifiedRow(row storedClassification) (agenttranscript.Classification, bool) {
	class := classifyConversationRow(row.source, row.role, row.text)
	threadKey := class.Thread.Key()
	if row.threadID != "" {
		threadKey = row.threadID
	}
	unchanged := string(class.Kind) == row.kind && string(class.Author) == row.author &&
		class.AuthorName == row.authorName && class.Label == row.label && threadKey == row.threadID
	return class, !unchanged
}

func (b *conversationClassificationBackfill) classifyBatch(ctx context.Context, after int64, report *classificationBackfillReport) (int64, int, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT rowid, session_id, source, role, substr(text, 1, ?), kind, author, author_name, label, thread_id, created_at
		FROM conversation_events WHERE rowid > ? ORDER BY rowid LIMIT ?`, b.textPrefix, after, b.batch)
	if err != nil {
		return after, 0, fmt.Errorf("read batch: %w", err)
	}
	var batch []storedClassification
	for rows.Next() {
		var row storedClassification
		if err := rows.Scan(&row.rowid, &row.sessionID, &row.source, &row.role, &row.text, &row.kind, &row.author, &row.authorName, &row.label, &row.threadID, &row.created); err != nil {
			rows.Close()
			return after, 0, fmt.Errorf("scan batch: %w", err)
		}
		batch = append(batch, row)
	}
	if err := rows.Close(); err != nil {
		return after, 0, err
	}
	if len(batch) == 0 {
		return after, 0, nil
	}
	last := batch[len(batch)-1].rowid
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return after, 0, err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, row := range batch {
		report.Scanned++
		class, changed := classifiedRow(row)
		if !changed {
			continue
		}
		threadID := row.threadID
		if threadID == "" {
			threadID = class.Thread.Key()
		}
		if _, err := tx.ExecContext(ctx, `UPDATE conversation_events SET kind = ?, author = ?, author_name = ?, label = ?, thread_id = ? WHERE rowid = ?`,
			string(class.Kind), string(class.Author), class.AuthorName, class.Label, threadID, row.rowid); err != nil {
			return after, 0, fmt.Errorf("classify row %d: %w", row.rowid, err)
		}
		if threadID != "" && row.threadID == "" {
			thread := conversationThreadFromRef(class.Thread, parseConversationTime(row.created))
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO conversation_threads (session_id, thread_id, kind, label, description, parent_thread_id, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, row.sessionID, thread.ThreadID, thread.Kind, thread.Label, thread.Description, thread.ParentThreadID, row.created); err != nil {
				return after, 0, fmt.Errorf("register thread for row %d: %w", row.rowid, err)
			}
		}
		report.Changed++
		report.ByKind[string(class.Kind)]++
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO web_console_migrations(name, last_rowid, completed_at) VALUES (?, ?, '')
		ON CONFLICT(name) DO UPDATE SET last_rowid = excluded.last_rowid`, conversationClassificationMigration, last); err != nil {
		return after, 0, fmt.Errorf("save cursor: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return after, 0, err
	}
	return last, len(batch), nil
}

// attributeLaunchPrompts applies the launch-prompt rule (sessionLaunchPrompts)
// to history: the first main-thread prompt of each programmatic session was
// typed by the program that opened it.
func (b *conversationClassificationBackfill) attributeLaunchPrompts(ctx context.Context) (int64, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT id, owner FROM sessions WHERE origin = ?`, "programmatic")
	if err != nil {
		return 0, fmt.Errorf("list programmatic sessions: %w", err)
	}
	type launched struct{ id, owner string }
	var sessions []launched
	for rows.Next() {
		var session launched
		if err := rows.Scan(&session.id, &session.owner); err != nil {
			rows.Close()
			return 0, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	var changed int64
	for _, session := range sessions {
		result, err := b.db.ExecContext(ctx, `UPDATE conversation_events SET author = 'automation', author_name = ?, label = 'launch_prompt'
			WHERE rowid = (SELECT rowid FROM conversation_events WHERE session_id = ? AND thread_id = '' AND role = 'user'
				AND kind = 'message' AND author IN ('human', 'automation') ORDER BY sequence LIMIT 1)
			AND author = 'human'`, launchOwnerLabel(session.owner), session.id)
		if err != nil {
			return changed, fmt.Errorf("attribute launch prompt for %s: %w", sanitizeID(session.id), err)
		}
		count, _ := result.RowsAffected()
		changed += count
	}
	return changed, nil
}
