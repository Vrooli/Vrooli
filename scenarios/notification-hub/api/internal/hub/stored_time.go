package hub

import (
	"context"
	"fmt"
	"time"
)

// storedTimeLayout is the one text form of every persisted timestamp: UTC, a
// fixed nine-digit fraction, so every value is the same width. SQLite compares
// timestamps as text, and in a variable-width form such as RFC 3339 with trimmed
// zeros ("…:00Z" and "…:00.5Z") text order is not time order within a second. In
// this form it is, so ORDER BY, keyset cursors and "<" against a cutoff agree
// with the clock.
const storedTimeLayout = "2006-01-02T15:04:05.000000000Z"

// storedTime is the text a timestamp is persisted as. Every write of a
// timestamp goes through it; time.Parse(time.RFC3339Nano, ...) reads it back.
func storedTime(t time.Time) string { return t.UTC().Format(storedTimeLayout) }

// normalizeStoredTime rewrites a timestamp persisted in any RFC 3339 form to
// storedTime. An empty value (no deadline, not resolved) and one that does not
// parse are left as they are, and ok is false.
func normalizeStoredTime(text string) (normalized string, ok bool) {
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return text, false
	}
	return storedTime(parsed), true
}

// storedTimeWidth is len(storedTimeLayout) output; a value of any other width
// was written before the fixed form and still needs rewriting.
const storedTimeWidth = len(storedTimeLayout)

// storedTimeColumns are the persisted timestamps that are ordered, compared
// against a cutoff or used as a keyset: the ask domain and the notifications
// they hang from. normalizeStoredTimes rewrites rows written in the older,
// variable-width form.
var storedTimeColumns = []struct {
	table   string
	columns []string
}{
	{"asks", []string{"created_at", "updated_at", "deadline", "resolved_at", "resolution_published_at", "last_asserted_at", "default_applied_at"}},
	{"ask_revisions", []string{"deadline", "revised_at"}},
	{"ask_context_requests", []string{"requested_at", "addressed_at", "published_at"}},
	{"escalation_steps", []string{"created_at"}},
	{"resolution_outbox_cursor", []string{"resolved_at"}},
	{"notifications", []string{"created_at", "updated_at", "scheduled_at"}},
	{"receipts", []string{"delivered_at"}},
}

const storedTimeBatch = 500

// normalizeStoredTimes is the additive migration to storedTime: each value not
// already in the fixed form is parsed and rewritten, a batch per transaction.
// It is idempotent and reads only rows that still need it, so it costs a scan
// per column once the data is converted. A value that does not parse is left
// alone and logged; an error does not stop boot.
func (s *Service) normalizeStoredTimes(ctx context.Context) {
	for _, target := range storedTimeColumns {
		for _, column := range target.columns {
			rewritten, err := s.normalizeStoredTimeColumn(ctx, target.table, column)
			if err != nil {
				s.log.Printf("normalize %s.%s timestamps: %v", target.table, column, err)
			}
			if rewritten > 0 {
				s.log.Printf("normalize %s.%s timestamps: rewrote %d to the fixed-width form", target.table, column, rewritten)
			}
		}
	}
}

func (s *Service) normalizeStoredTimeColumn(ctx context.Context, table, column string) (int, error) {
	// The names come from storedTimeColumns, never from a caller.
	selectSQL := fmt.Sprintf(`SELECT rowid, %[2]s FROM %[1]s WHERE %[2]s <> '' AND length(%[2]s) <> %[3]d AND rowid > ? ORDER BY rowid LIMIT %[4]d`, table, column, storedTimeWidth, storedTimeBatch)
	updateSQL := fmt.Sprintf(`UPDATE %s SET %s = ? WHERE rowid = ?`, table, column)
	rewritten := 0
	var after int64
	for {
		type change struct {
			rowid int64
			text  string
		}
		var batch []change
		var last int64
		rows, err := s.db.QueryContext(ctx, selectSQL, after)
		if err != nil {
			return rewritten, err
		}
		for rows.Next() {
			var rowid int64
			var text string
			if err := rows.Scan(&rowid, &text); err != nil {
				_ = rows.Close()
				return rewritten, err
			}
			last = rowid
			if normalized, ok := normalizeStoredTime(text); ok {
				batch = append(batch, change{rowid, normalized})
			} else {
				s.log.Printf("normalize %s.%s timestamps: row %d holds %q, which is not a timestamp; left as it is", table, column, rowid, text)
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return rewritten, err
		}
		if err := rows.Close(); err != nil {
			return rewritten, err
		}
		if last == 0 {
			return rewritten, nil
		}
		after = last
		if len(batch) == 0 {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return rewritten, err
		}
		for _, c := range batch {
			if _, err := tx.ExecContext(ctx, updateSQL, c.text, c.rowid); err != nil {
				_ = tx.Rollback()
				return rewritten, err
			}
		}
		if err := tx.Commit(); err != nil {
			return rewritten, err
		}
		rewritten += len(batch)
	}
}
