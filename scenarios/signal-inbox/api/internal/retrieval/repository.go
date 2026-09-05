package retrieval

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"signal-inbox/internal/signals"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Rebuild(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM signal_fts"); err != nil {
		return fmt.Errorf("clear FTS projection: %w", err)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO signal_fts(signal_id,content)
SELECT s.id, trim(s.source_url || ' ' || s.extracted_content || ' ' || COALESCE((SELECT a.body FROM signal_annotations a WHERE a.signal_id=s.id AND a.kind='capture_note' ORDER BY a.created_at LIMIT 1), '')) FROM signal s`)
	return err
}

func (r *sqliteRepository) Search(ctx context.Context, filter Filter) ([]Result, error) {
	where, args := filters(filter, false, time.Time{})
	join := ""
	if strings.TrimSpace(filter.Text) != "" {
		join = " JOIN signal_fts f ON f.signal_id=s.id"
		where = append(where, "f.content MATCH ?")
		args = append(args, filter.Text)
	}
	query := resultSelect + join + " WHERE " + strings.Join(where, " AND ") + " ORDER BY s.captured_at DESC,s.id DESC LIMIT ?"
	args = append(args, filter.Limit)
	return r.rows(ctx, query, args...)
}

func (r *sqliteRepository) Ambient(ctx context.Context, categoryID string, budget int, now time.Time) ([]Result, error) {
	filter := Filter{CategoryID: categoryID}
	where, args := filters(filter, true, now)
	args = append(args, budget)
	return r.rows(ctx, resultSelect+" WHERE "+strings.Join(where, " AND ")+" ORDER BY s.captured_at DESC,s.id DESC LIMIT ?", args...)
}

func filters(filter Filter, ambient bool, now time.Time) ([]string, []any) {
	where, args := []string{"1=1"}, []any{}
	if filter.CategoryID != "" {
		where, args = append(where, "COALESCE((SELECT c.confirmed_category_id FROM classification c WHERE c.signal_id=s.id AND c.confirmed_category_id<>'' ORDER BY c.created_at DESC,c.rowid DESC LIMIT 1),'')=?"), append(args, filter.CategoryID)
	}
	if filter.Disposition != "" {
		where, args = append(where, "COALESCE(d.state,'new')=?"), append(args, filter.Disposition)
	}
	if filter.SourceKind != "" {
		where, args = append(where, "s.source_kind=?"), append(args, filter.SourceKind)
	}
	if filter.CapturedAfter != nil {
		where, args = append(where, "s.captured_at>=?"), append(args, filter.CapturedAfter.UTC().Format(time.RFC3339Nano))
	}
	if filter.CapturedBefore != nil {
		where, args = append(where, "s.captured_at<=?"), append(args, filter.CapturedBefore.UTC().Format(time.RFC3339Nano))
	}
	if filter.PageAfterCapturedAt != nil && filter.PageAfterSignalID != "" {
		cursorTime := filter.PageAfterCapturedAt.UTC().Format(time.RFC3339Nano)
		where, args = append(where, "(s.captured_at < ? OR (s.captured_at = ? AND s.id < ?))"), append(args, cursorTime, cursorTime, filter.PageAfterSignalID)
	}
	for _, tag := range filter.Tags {
		where, args = append(where, "EXISTS (SELECT 1 FROM signal_tag t WHERE t.signal_id=s.id AND t.tag=?)"), append(args, tag)
	}
	if ambient {
		where, args = append(where, "COALESCE(d.state,'new') IN ('new','triaged')", "(d.revisit_at IS NULL OR d.revisit_at='' OR d.revisit_at<=?)"), append(args, now.UTC().Format(time.RFC3339Nano))
	}
	return where, args
}

const resultSelect = `SELECT s.id,s.source_kind,s.source_identity,s.source_url,s.raw_payload_ref,s.extracted_content,s.content_hash,s.needs_attention,s.captured_at,COALESCE((SELECT a.body FROM signal_annotations a WHERE a.signal_id=s.id AND a.kind='capture_note' ORDER BY a.created_at LIMIT 1),''),COALESCE((SELECT group_concat(tag, char(31)) FROM signal_tag t WHERE t.signal_id=s.id),''),COALESCE((SELECT c.confirmed_category_id FROM classification c WHERE c.signal_id=s.id AND c.confirmed_category_id<>'' ORDER BY c.created_at DESC,c.rowid DESC LIMIT 1),''),COALESCE(d.state,'new') FROM signal s LEFT JOIN disposition d ON d.signal_id=s.id`

func (r *sqliteRepository) rows(ctx context.Context, query string, args ...any) ([]Result, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []Result{}
	for rows.Next() {
		var result Result
		var captured string
		var tags string
		if err := rows.Scan(&result.Signal.ID, &result.Signal.SourceKind, &result.Signal.SourceIdentity, &result.Signal.SourceURL, &result.Signal.RawPayloadRef, &result.Signal.ExtractedContent, &result.Signal.ContentHash, &result.Signal.NeedsAttention, &captured, &result.Signal.CaptureNote, &tags, &result.CategoryID, &result.Disposition); err != nil {
			return nil, err
		}
		result.Signal.CapturedAt, err = time.Parse(time.RFC3339Nano, captured)
		if err != nil {
			return nil, err
		}
		if tags != "" {
			result.Signal.Tags = strings.Split(tags, "\x1f")
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

func (r *sqliteRepository) IndexedCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM signal_fts").Scan(&count)
	return count, err
}

func (r *sqliteRepository) JournalCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM signal").Scan(&count)
	return count, err
}

var _ = signals.SourceKindURL
