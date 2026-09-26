package validation_record

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

const recordTimeFormat = time.RFC3339Nano

const insertRecordSQL = `
INSERT INTO validation_records (
  id, tuple_kind, subject_id, golden_slug,
  started_at, ended_at, duration_ms,
  tokens_used, cost_usd_micro,
  verdict,
  diff_hash, diff_path_count,
  agent_manager_run_id,
  manifest_template_version_at_run, manifest_skill_version_at_run,
  error_message,
  tool_detail, tool_raw_output
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const selectRecordByIDSQL = `
SELECT id, tuple_kind, subject_id, golden_slug,
       started_at, ended_at, duration_ms,
       tokens_used, cost_usd_micro,
       verdict,
       diff_hash, diff_path_count,
       agent_manager_run_id,
       manifest_template_version_at_run, manifest_skill_version_at_run,
       error_message,
       tool_detail, tool_raw_output
FROM validation_records
WHERE id = ?
`

// SQLExecutor is the narrow database surface this package's repository
// depends on. Both *sql.DB (used by repository unit tests via
// testutil/db.NewSQLite) and *database.RoutedDB (production main.go)
// satisfy it, so production wiring participates in per-request routing
// without forcing test fixtures to wrap their handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) Append(ctx context.Context, r Record) error {
	_, err := s.db.ExecContext(ctx, insertRecordSQL,
		r.ID, int(r.TupleKind), r.SubjectID, r.GoldenSlug,
		r.StartedAt.UTC().Format(recordTimeFormat),
		r.EndedAt.UTC().Format(recordTimeFormat),
		r.DurationMS,
		r.TokensUsed, r.CostUSDMicro,
		int(r.Verdict),
		r.DiffHash, r.DiffPathCount,
		r.AgentManagerRunID,
		r.ManifestTemplateVersionAtRun, r.ManifestSkillVersionAtRun,
		r.ErrorMessage,
		r.ToolDetail, r.ToolRawOutput,
	)
	if err != nil {
		return fmt.Errorf("append validation_record %q: %w", r.ID, err)
	}
	return nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Record, error) {
	row := s.db.QueryRowContext(ctx, selectRecordByIDSQL, id)
	r, err := scanRecord(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrRecordNotFound{ID: id}
	}
	if err != nil {
		return Record{}, fmt.Errorf("get validation_record %q: %w", id, err)
	}
	return r, nil
}

func (s *sqliteRepository) List(ctx context.Context, f ListFilter, pageSize int, pageToken string) (ListResult, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	var (
		clauses []string
		args    []any
	)
	if f.GoldenSlug != "" {
		clauses = append(clauses, "golden_slug = ?")
		args = append(args, f.GoldenSlug)
	}
	if f.SubjectID != "" {
		clauses = append(clauses, "subject_id = ?")
		args = append(args, f.SubjectID)
	}
	if f.TupleKind != TupleKindUnspecified {
		clauses = append(clauses, "tuple_kind = ?")
		args = append(args, int(f.TupleKind))
	}
	cursorEnded, cursorID, hasCursor, err := decodeCursor(pageToken)
	if err != nil {
		return ListResult{}, ErrInvalidRecord{Field: "page_token", Reason: err.Error()}
	}
	if hasCursor {
		// Stable order: ended_at DESC, id DESC. Cursor seeks strictly
		// past the last row.
		clauses = append(clauses, "(ended_at < ? OR (ended_at = ? AND id < ?))")
		args = append(args, cursorEnded, cursorEnded, cursorID)
	}

	query := `
SELECT id, tuple_kind, subject_id, golden_slug,
       started_at, ended_at, duration_ms,
       tokens_used, cost_usd_micro,
       verdict,
       diff_hash, diff_path_count,
       agent_manager_run_id,
       manifest_template_version_at_run, manifest_skill_version_at_run,
       error_message,
       tool_detail, tool_raw_output
FROM validation_records
`
	if len(clauses) > 0 {
		query += "WHERE " + strings.Join(clauses, " AND ") + "\n"
	}
	query += "ORDER BY ended_at DESC, id DESC LIMIT ?"
	args = append(args, pageSize+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list validation_records: %w", err)
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return ListResult{}, fmt.Errorf("scan validation_record: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate validation_records: %w", err)
	}

	res := ListResult{Records: out}
	if len(out) > pageSize {
		last := out[pageSize-1]
		res.Records = out[:pageSize]
		res.NextPageToken = encodeCursor(last.EndedAt.UTC().Format(recordTimeFormat), last.ID)
	}
	return res, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(s rowScanner) (Record, error) {
	var (
		r            Record
		tupleKindInt int
		verdictInt   int
		startedRaw   string
		endedRaw     string
	)
	if err := s.Scan(
		&r.ID, &tupleKindInt, &r.SubjectID, &r.GoldenSlug,
		&startedRaw, &endedRaw, &r.DurationMS,
		&r.TokensUsed, &r.CostUSDMicro,
		&verdictInt,
		&r.DiffHash, &r.DiffPathCount,
		&r.AgentManagerRunID,
		&r.ManifestTemplateVersionAtRun, &r.ManifestSkillVersionAtRun,
		&r.ErrorMessage,
		&r.ToolDetail, &r.ToolRawOutput,
	); err != nil {
		return Record{}, err
	}
	r.TupleKind = TupleKind(tupleKindInt)
	r.Verdict = Verdict(verdictInt)
	t, err := time.Parse(recordTimeFormat, startedRaw)
	if err != nil {
		return Record{}, fmt.Errorf("parse started_at %q: %w", startedRaw, err)
	}
	r.StartedAt = t
	t, err = time.Parse(recordTimeFormat, endedRaw)
	if err != nil {
		return Record{}, fmt.Errorf("parse ended_at %q: %w", endedRaw, err)
	}
	r.EndedAt = t
	return r, nil
}

// encodeCursor produces an opaque, URL-safe page token over the
// (ended_at, id) tuple. Format: base64("ended_at\x00id"). Opaque to
// callers; the structure may change without breaking the API contract
// because callers always echo the token verbatim.
func encodeCursor(endedAt, id string) string {
	return base64.URLEncoding.EncodeToString([]byte(endedAt + "\x00" + id))
}

func decodeCursor(token string) (endedAt, id string, ok bool, err error) {
	if token == "" {
		return "", "", false, nil
	}
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid base64: %w", err)
	}
	parts := strings.SplitN(string(raw), "\x00", 2)
	if len(parts) != 2 {
		return "", "", false, fmt.Errorf("malformed cursor")
	}
	return parts[0], parts[1], true, nil
}
