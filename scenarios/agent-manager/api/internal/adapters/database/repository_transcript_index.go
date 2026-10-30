// Responsibility: persist the Transcript Indexer's file rows and durable usage summaries.
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// TranscriptIndexRepository implements transcriptindex.Store.
type TranscriptIndexRepository struct{ db *DB }

// NewTranscriptIndexRepository builds the indexer's store.
func NewTranscriptIndexRepository(db *DB) *TranscriptIndexRepository {
	return &TranscriptIndexRepository{db: db}
}

var _ transcriptindex.Store = (*TranscriptIndexRepository)(nil)

type transcriptFileRow struct {
	Key                 string         `db:"file_key"`
	Path                string         `db:"path"`
	RootKind            string         `db:"root_kind"`
	RootPath            string         `db:"root_path"`
	Harness             string         `db:"harness"`
	SessionID           string         `db:"session_id"`
	HeaderSessionID     string         `db:"header_session_id"`
	ParentSessionID     string         `db:"parent_session_id"`
	ParentFileKey       string         `db:"parent_file_key"`
	Sidechain           bool           `db:"sidechain"`
	WebConsoleSessionID string         `db:"web_console_session_id"`
	RunID               string         `db:"run_id"`
	Owner               string         `db:"owner"`
	Status              string         `db:"status"`
	Error               string         `db:"error"`
	Offset              int64          `db:"read_offset"`
	Size                int64          `db:"size"`
	MtimeUnixNano       int64          `db:"mtime_unix_nano"`
	HeadHash            string         `db:"head_hash"`
	Turns               int64          `db:"turns"`
	UsageReports        int64          `db:"usage_reports"`
	Active              bool           `db:"active"`
	RecallBytes         int64          `db:"recall_bytes"`
	RecallEvicted       bool           `db:"recall_evicted"`
	PendingFacts        string         `db:"pending_facts_json"`
	ParserVersion       int            `db:"parser_version"`
	LastObservedEventAt sql.NullString `db:"last_observed_event_at"`
	FirstSeenAt         string         `db:"first_seen_at"`
	IndexedAt           sql.NullString `db:"indexed_at"`
}

const transcriptFileColumns = `file_key, path, root_kind, root_path, harness, session_id, header_session_id, parent_session_id,
	parent_file_key, sidechain, web_console_session_id, run_id, owner, status, error, read_offset, size, mtime_unix_nano,
	head_hash, turns, usage_reports, active, recall_bytes, recall_evicted, pending_facts_json, parser_version, last_observed_event_at, first_seen_at, indexed_at`

func (r *TranscriptIndexRepository) ListTranscriptFiles(ctx context.Context) ([]transcriptindex.FileRow, error) {
	var rows []transcriptFileRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT `+transcriptFileColumns+` FROM transcript_files`); err != nil {
		return nil, wrapDBError("list_transcript_files", "TranscriptFile", "", err)
	}
	out := make([]transcriptindex.FileRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, transcriptFileFromRow(row))
	}
	return out, nil
}

func transcriptFileFromRow(row transcriptFileRow) transcriptindex.FileRow {
	file := transcriptindex.FileRow{
		Key: row.Key, Path: row.Path, RootKind: row.RootKind, RootPath: row.RootPath, Harness: row.Harness,
		SessionID: row.SessionID, HeaderSessionID: row.HeaderSessionID, ParentSessionID: row.ParentSessionID,
		ParentFileKey: row.ParentFileKey, Sidechain: row.Sidechain, WebConsoleSessionID: row.WebConsoleSessionID,
		RunID: row.RunID, Owner: row.Owner, Status: row.Status, Error: row.Error, Offset: row.Offset, Size: row.Size,
		MtimeUnixNano: row.MtimeUnixNano, HeadHash: row.HeadHash, Turns: row.Turns, UsageReports: row.UsageReports,
		Active: row.Active, RecallBytes: row.RecallBytes, RecallEvicted: row.RecallEvicted, PendingFacts: row.PendingFacts,
		ParserVersion: row.ParserVersion,
	}
	file.FirstSeenAt, _ = time.Parse(time.RFC3339Nano, row.FirstSeenAt)
	if row.LastObservedEventAt.Valid {
		if at, err := time.Parse(time.RFC3339Nano, row.LastObservedEventAt.String); err == nil {
			file.LastObservedEventAt = &at
		}
	}
	if row.IndexedAt.Valid {
		if at, err := time.Parse(time.RFC3339Nano, row.IndexedAt.String); err == nil {
			file.IndexedAt = &at
		}
	}
	return file
}

func (r *TranscriptIndexRepository) transcriptFileTx(ctx context.Context, tx *Tx, key string) (*transcriptindex.FileRow, error) {
	rows, err := tx.QueryContext(ctx, `SELECT `+transcriptFileColumns+` FROM transcript_files WHERE file_key = ?`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mapper := r.db.mapper
	if mapper == nil {
		mapper = reflectx.NewMapper("db")
	}
	xrows := &sqlx.Rows{Rows: rows, Mapper: mapper}
	if !xrows.Next() {
		return nil, xrows.Err()
	}
	var raw transcriptFileRow
	if err := xrows.StructScan(&raw); err != nil {
		return nil, err
	}
	file := transcriptFileFromRow(raw)
	return &file, nil
}

// BeginTranscriptCapture conditionally claims the exact observed checkpoint.
// The unique intent string is retained until the matching owner commit.
func (r *TranscriptIndexRepository) BeginTranscriptCapture(ctx context.Context, expected *transcriptindex.FileRow, intent transcriptindex.FileRow) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := r.transcriptFileTx(ctx, tx, intent.Key)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, expected) {
		return fmt.Errorf("transcript checkpoint changed before capture intent")
	}
	if err := r.saveTranscriptFileTx(ctx, tx, intent, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TranscriptIndexRepository) SaveTranscriptIfUnchanged(ctx context.Context, expected *transcriptindex.FileRow, file transcriptindex.FileRow) error {
	return r.BeginTranscriptCapture(ctx, expected, file)
}

// AdmitLegacyRelink binds a source checkpoint only while its exact legacy run
// still has the expected provenance and no other indexed file or usage owner.
// Existing event and private-file evidence remains untouched.
func (r *TranscriptIndexRepository) AdmitLegacyRelink(ctx context.Context, expected *transcriptindex.FileRow, file transcriptindex.FileRow, legacyPath string) error {
	id, err := uuid.Parse(file.RunID)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := r.transcriptFileTx(ctx, tx, file.Key)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, expected) {
		return fmt.Errorf("legacy source checkpoint changed")
	}
	run, err := (&runRepository{db: r.db}).getTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeImported || run.TranscriptPath != legacyPath || run.ImportSourceHarness != file.Harness || run.ImportSourceSessionID != file.SessionID {
		return fmt.Errorf("legacy run identity changed")
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM transcript_files WHERE run_id = ? AND file_key <> ?) + (SELECT COUNT(*) FROM transcript_usage WHERE run_id = ?)`, file.RunID, file.Key, file.RunID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("legacy run has other transcript or usage evidence requiring reconciliation")
	}
	if err := r.saveTranscriptFileTx(ctx, tx, file, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// RecordLegacyCleanupInTx retains an explicit cleanup obligation and refuses
// promotion if another file or usage source became associated with the run.
func (r *TranscriptIndexRepository) RecordLegacyCleanupInTx(ctx context.Context, tx *sql.Tx, file transcriptindex.FileRow, privatePath string) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM transcript_files WHERE run_id = ? AND file_key <> ?) + (SELECT COUNT(*) FROM transcript_usage WHERE run_id = ?)`, file.RunID, file.Key, file.RunID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("legacy run acquired other transcript or usage evidence")
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, file.RunID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("legacy run retains events whose coverage by the new source is unverified")
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO transcript_legacy_cleanup(run_id, file_key, private_path, source_path) VALUES (?, ?, ?, ?)`, file.RunID, file.Key, privatePath, file.Path)
	return err
}

func (r *TranscriptIndexRepository) LegacyCleanupIssue(ctx context.Context, runID string) (*transcriptindex.ReplacementIssue, error) {
	var issue transcriptindex.ReplacementIssue
	err := r.db.QueryRowContext(ctx, `SELECT file_key, private_path FROM transcript_legacy_cleanup WHERE run_id = ?`, runID).Scan(&issue.FileKey, &issue.Path)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	issue.Reason = "legacy private transcript retained; cleanup requires identity-safe owner operation"
	return &issue, nil
}

// ResetEmptyOwnedTranscript permits the legacy run-owned backfill only when
// its exact checkpoint still exists and has no retained usage to discard.
func (r *TranscriptIndexRepository) ResetEmptyOwnedTranscript(ctx context.Context, expected, next transcriptindex.FileRow) error {
	if expected.Owner != transcriptindex.OwnerRun || expected.IndexedAt != nil || next.Owner != transcriptindex.OwnerRun {
		return fmt.Errorf("not an unindexed run-owned checkpoint")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := r.transcriptFileTx(ctx, tx, expected.Key)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, &expected) {
		return fmt.Errorf("run-owned checkpoint changed before backfill")
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM transcript_usage WHERE file_key = ?`, expected.Key).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("run-owned checkpoint retains usage; reconciliation required")
	}
	if err := r.saveTranscriptFileTx(ctx, tx, next, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// FailTranscriptCapture records a failed read only while the exact capture
// intent is still owned; a stale worker cannot overwrite a newer checkpoint.
func (r *TranscriptIndexRepository) FailTranscriptCapture(ctx context.Context, key, intent, status, reason string) error {
	if intent == "" {
		return fmt.Errorf("capture intent is required")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE transcript_files SET status = ?, error = ? WHERE file_key = ? AND status = ? AND error = ?`, status, reason, key, transcriptindex.StatusPending, intent)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("capture intent changed before failure could be recorded")
	}
	return nil
}

func (r *TranscriptIndexRepository) CommitOwnedTranscript(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, intent string) error {
	if file.Owner != transcriptindex.OwnerRun || intent == "" {
		return fmt.Errorf("run-owned transcript intent is required")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	checkpoint, err := r.transcriptFileTx(ctx, tx, file.Key)
	if err != nil {
		return err
	}
	if checkpoint == nil || checkpoint.Status != transcriptindex.StatusPending || checkpoint.Error != intent || checkpoint.RunID != file.RunID {
		return fmt.Errorf("run-owned transcript intent changed before commit")
	}
	if err := r.saveTranscriptFileTx(ctx, tx, file, usage); err != nil {
		return err
	}
	return tx.Commit()
}

// CommitRecallEviction fences the checkpoint and event owner together while
// preserving the durable usage ledger.
func (r *TranscriptIndexRepository) CommitRecallEviction(ctx context.Context, expected, file transcriptindex.FileRow, deleteEvents func(*sql.Tx) error) error {
	if expected.Owner != transcriptindex.OwnerIndexer || expected.Active || expected.Status != transcriptindex.StatusIndexed || expected.Error != "" || deleteEvents == nil {
		return fmt.Errorf("recall checkpoint is not eligible for eviction")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := r.transcriptFileTx(ctx, tx, expected.Key)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, &expected) {
		return fmt.Errorf("recall checkpoint changed before eviction")
	}
	var status, mode string
	if err := tx.QueryRowContext(ctx, `SELECT status, COALESCE(execution_mode, '') FROM runs WHERE id = ?`, expected.RunID).Scan(&status, &mode); err != nil {
		return err
	}
	if domain.ExecutionMode(mode).Normalized() != domain.ExecutionModeImported || domain.RunStatus(status) != domain.RunStatusComplete {
		return fmt.Errorf("recall run is not an idle imported run")
	}
	if err := deleteEvents(tx.Tx); err != nil {
		return err
	}
	if err := r.saveTranscriptFileTx(ctx, tx, file, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// CommitRecallRebuild checks the original checkpoint before publishing any
// staged events, and commits both the events and checkpoint in one transaction.
func (r *TranscriptIndexRepository) CommitRecallRebuild(ctx context.Context, expected, file transcriptindex.FileRow, appendEvents func(*sql.Tx) error) error {
	if expected.Owner != transcriptindex.OwnerIndexer || expected.Status != transcriptindex.StatusIndexed || expected.Error != "" || appendEvents == nil {
		return fmt.Errorf("recall checkpoint is not eligible for rebuild")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := r.transcriptFileTx(ctx, tx, expected.Key)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, &expected) {
		return fmt.Errorf("recall checkpoint changed during rebuild")
	}
	if err := appendEvents(tx.Tx); err != nil {
		return err
	}
	if err := r.saveTranscriptFileTx(ctx, tx, file, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TranscriptIndexRepository) SaveTranscriptFile(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("save_transcript_file", "TranscriptFile", file.Key, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.saveTranscriptFileTx(ctx, tx, file, usage); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrapDBError("save_transcript_file", "TranscriptFile", file.Key, err)
	}
	return nil
}

func (r *TranscriptIndexRepository) saveTranscriptFileTx(ctx context.Context, tx *Tx, file transcriptindex.FileRow, usage []transcriptindex.UsageRow) error {
	var indexedAt any
	var lastObservedEventAt any
	if file.LastObservedEventAt != nil {
		lastObservedEventAt = file.LastObservedEventAt.UTC().Format(time.RFC3339Nano)
	}
	if file.IndexedAt != nil {
		indexedAt = file.IndexedAt.UTC().Format(time.RFC3339Nano)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO transcript_files (`+transcriptFileColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(file_key) DO UPDATE SET path=excluded.path, root_kind=excluded.root_kind, root_path=excluded.root_path,
			harness=excluded.harness, session_id=excluded.session_id, header_session_id=excluded.header_session_id,
			parent_session_id=excluded.parent_session_id, parent_file_key=excluded.parent_file_key, sidechain=excluded.sidechain,
			web_console_session_id=excluded.web_console_session_id, run_id=excluded.run_id, owner=excluded.owner,
			status=excluded.status, error=excluded.error, read_offset=excluded.read_offset, size=excluded.size,
			mtime_unix_nano=excluded.mtime_unix_nano, head_hash=excluded.head_hash, turns=excluded.turns,
			usage_reports=excluded.usage_reports, active=excluded.active, recall_bytes=excluded.recall_bytes,
			recall_evicted=excluded.recall_evicted, pending_facts_json=excluded.pending_facts_json,
			parser_version=excluded.parser_version, last_observed_event_at=excluded.last_observed_event_at, indexed_at=excluded.indexed_at`,
		file.Key, file.Path, file.RootKind, file.RootPath, file.Harness, file.SessionID, file.HeaderSessionID, file.ParentSessionID,
		file.ParentFileKey, boolInt(file.Sidechain), file.WebConsoleSessionID, file.RunID, file.Owner, file.Status, file.Error,
		file.Offset, file.Size, file.MtimeUnixNano, file.HeadHash, file.Turns, file.UsageReports, boolInt(file.Active),
		file.RecallBytes, boolInt(file.RecallEvicted), file.PendingFacts, file.ParserVersion, lastObservedEventAt, file.FirstSeenAt.UTC().Format(time.RFC3339Nano), indexedAt); err != nil {
		return wrapDBError("save_transcript_file", "TranscriptFile", file.Key, err)
	}
	if len(usage) > 0 {
		stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at,
			session_id, message_id, parent_tool_use_id, model, turn_index, input_tokens, output_tokens, cache_read_tokens,
			cache_creation_tokens, has_cumulative, cum_input_tokens, cum_output_tokens, cum_cache_read_tokens,
			cum_cache_creation_tokens, reconciliation_authority, images_in, images_in_user, images_in_high_detail,
			image_bytes_in, images_out, web_searches, web_fetches, audio_in, tool_calls, tool_kinds_json, credits_balance)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return wrapDBError("save_transcript_usage", "TranscriptUsage", file.Key, err)
		}
		defer stmt.Close()
		for _, row := range usage {
			u, f := row.Usage, row.Facts
			var cumulative domain.UsageCounters
			if u.Cumulative != nil {
				cumulative = *u.Cumulative
			}
			kinds := "{}"
			if len(f.ToolKinds) > 0 {
				if encoded, err := json.Marshal(f.ToolKinds); err == nil {
					kinds = string(encoded)
				}
			}
			if _, err := stmt.ExecContext(ctx, row.FileKey, row.LineOffset, row.Ordinal, row.RunID, row.ObservedAt.UTC().Format(time.RFC3339Nano),
				u.SessionID, u.MessageID, u.ParentToolUseID, u.Model, u.TurnIndex, u.InputTokens, u.OutputTokens, u.CacheReadTokens,
				u.CacheCreationTokens, boolInt(u.Cumulative != nil), cumulative.InputTokens, cumulative.OutputTokens, cumulative.CacheReadTokens,
				cumulative.CacheCreationTokens, boolInt(u.ReconciliationAuthority), f.ImagesIn, f.ImagesInUser, f.ImagesInHighDetail,
				f.ImageBytesIn, f.ImagesOut, f.WebSearches+u.WebSearchRequests, f.WebFetches, f.AudioIn, f.ToolCalls, kinds, f.CreditsBalance); err != nil {
				return wrapDBError("save_transcript_usage", "TranscriptUsage", file.Key, err)
			}
		}
	}
	return nil
}

// CommitTranscriptAndRun preserves the canonical run update guards and task
// projection while committing the transcript checkpoint in the same owner transaction.
func (r *TranscriptIndexRepository) CommitTranscriptAndRun(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, expected *transcriptindex.FileRow, intent string, apply func(*domain.Run) error) error {
	if file.Owner != transcriptindex.OwnerIndexer || apply == nil {
		return fmt.Errorf("atomic transcript update requires the matching imported run")
	}
	id, err := uuid.Parse(file.RunID)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("commit_transcript_and_run", "TranscriptFile", file.Key, err)
	}
	defer tx.Rollback()
	checkpoint, err := r.transcriptFileTx(ctx, tx, file.Key)
	if err != nil {
		return err
	}
	if intent != "" {
		if checkpoint == nil || checkpoint.Status != transcriptindex.StatusPending || checkpoint.Error != intent || checkpoint.RunID != file.RunID {
			return fmt.Errorf("capture intent changed before commit")
		}
	} else if !reflect.DeepEqual(checkpoint, expected) {
		return fmt.Errorf("transcript checkpoint changed before idle commit")
	}
	runs := &runRepository{db: r.db}
	run, err := runs.getTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeImported {
		return fmt.Errorf("atomic transcript update requires the matching imported run")
	}
	if err := apply(run); err != nil {
		return err
	}
	var next int64
	if err := runs.updateTx(ctx, tx, run, nil, &next); err != nil {
		return err
	}
	if err := r.saveTranscriptFileTx(ctx, tx, file, usage); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrapDBError("commit_transcript_and_run", "TranscriptFile", file.Key, err)
	}
	run.LifecycleVersion = next
	run.EffortSignalReceipt = nil
	return nil
}

// replacementToken covers the old file row and every associated usage row.
// It fences concurrent modifications between staging and promotion.
func replacementToken(ctx context.Context, tx *Tx, key string) (string, error) {
	h := sha256.New()
	total := 0
	for _, query := range []string{`SELECT * FROM transcript_files WHERE file_key = ?`, `SELECT * FROM transcript_usage WHERE file_key = ? ORDER BY line_offset, ordinal`} {
		rows, err := tx.QueryContext(ctx, query, key)
		if err != nil {
			return "", err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return "", err
		}
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return "", err
			}
			encoded, err := json.Marshal(values)
			if err != nil {
				rows.Close()
				return "", err
			}
			total += len(encoded)
			if total > 32<<20 {
				rows.Close()
				return "", fmt.Errorf("transcript evidence exceeds replacement token limit")
			}
			fmt.Fprintf(h, "%d:", len(encoded))
			_, _ = h.Write(encoded)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return "", err
		}
		if err := rows.Close(); err != nil {
			return "", err
		}
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (r *TranscriptIndexRepository) TranscriptReplacementToken(ctx context.Context, key string) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	return replacementToken(ctx, tx, key)
}

// MarkReplacementIntent fences the old evidence and persists uncertainty
// before promotion. It returns the token of the marked checkpoint.
func (r *TranscriptIndexRepository) MarkReplacementIntent(ctx context.Context, key, token string) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	actual, err := replacementToken(ctx, tx, key)
	if err != nil {
		return "", err
	}
	if actual != token {
		return "", fmt.Errorf("transcript evidence changed during replacement staging")
	}
	result, err := tx.ExecContext(ctx, `UPDATE transcript_files SET status = ?, error = ? WHERE file_key = ? AND ((status IN (?, ?) AND error = '') OR (status = ? AND substr(error, 1, ?) = ?))`, transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"promotion interrupted or outcome uncertain", key, transcriptindex.StatusIndexed, transcriptindex.StatusPending, transcriptindex.StatusUnreadable, len(transcriptindex.ReplacementReconciliationRequired), transcriptindex.ReplacementReconciliationRequired)
	if err != nil {
		return "", err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return "", fmt.Errorf("replacement checkpoint is unavailable or has another active intent")
	}
	next, err := replacementToken(ctx, tx, key)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return next, nil
}

// CommitTranscriptReplacement swaps usage, checkpoint, run state, and the
// caller-supplied event-owner operation in one routed transaction.
func (r *TranscriptIndexRepository) CommitTranscriptReplacement(ctx context.Context, oldKey, token string, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, apply func(*domain.Run) error, replaceEvents func(*sql.Tx) error) error {
	if file.Owner != transcriptindex.OwnerIndexer || apply == nil || replaceEvents == nil {
		return fmt.Errorf("replacement requires the matching imported run and event owner")
	}
	id, err := uuid.Parse(file.RunID)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	actual, err := replacementToken(ctx, tx, oldKey)
	if err != nil {
		return err
	}
	if actual != token {
		return fmt.Errorf("transcript evidence changed during replacement staging")
	}
	var present int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM transcript_files WHERE file_key = ? AND run_id = ?`, oldKey, file.RunID).Scan(&present); err != nil {
		return err
	}
	if present != 1 {
		return fmt.Errorf("replacement checkpoint is no longer owned by the run")
	}
	runs := &runRepository{db: r.db}
	run, err := runs.getTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeImported {
		return fmt.Errorf("replacement requires the matching imported run")
	}
	if err := apply(run); err != nil {
		return err
	}
	var next int64
	if err := runs.updateTx(ctx, tx, run, nil, &next); err != nil {
		return err
	}
	if err := replaceEvents(tx.Tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM transcript_usage WHERE file_key = ?`, oldKey); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM transcript_files WHERE file_key = ?`, oldKey); err != nil {
		return err
	}
	if err := r.saveTranscriptFileTx(ctx, tx, file, usage); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	run.LifecycleVersion = next
	run.EffortSignalReceipt = nil
	return nil
}

// FinishReplacementEffects clears the durable marker only for the exact
// checkpoint whose derived effects were delivered.
func (r *TranscriptIndexRepository) FinishReplacementEffects(ctx context.Context, file transcriptindex.FileRow) error {
	result, err := r.db.ExecContext(ctx, `UPDATE transcript_files SET status = ?, error = '' WHERE file_key = ? AND run_id = ? AND read_offset = ? AND mtime_unix_nano = ? AND status = ? AND error = ?`, transcriptindex.StatusIndexed, file.Key, file.RunID, file.Offset, file.MtimeUnixNano, transcriptindex.StatusPending, transcriptindex.ReplacementEffectsPending)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("replacement effects checkpoint changed before acknowledgement")
	}
	return nil
}

func (r *TranscriptIndexRepository) ResetTranscriptFile(ctx context.Context, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM transcript_usage WHERE file_key = ?`, key); err != nil {
		return wrapDBError("reset_transcript_file", "TranscriptFile", key, err)
	}
	return nil
}

// deleteRunTranscriptFilesTx forgets the transcript files indexed into runIDs
// inside the caller's transaction, when those runs are deleted.
func deleteRunTranscriptFilesTx(ctx context.Context, executor sqlExecutor, runIDs ...string) error {
	for _, runID := range runIDs {
		if _, err := executor.ExecContext(ctx, `DELETE FROM transcript_files WHERE run_id = ?`, runID); err != nil {
			return err
		}
	}
	return nil
}

func (r *TranscriptIndexRepository) DeleteTranscriptFile(ctx context.Context, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM transcript_files WHERE file_key = ?`, key); err != nil {
		return wrapDBError("delete_transcript_file", "TranscriptFile", key, err)
	}
	return nil
}

type transcriptUsageRow struct {
	FileKey                 string `db:"file_key"`
	LineOffset              int64  `db:"line_offset"`
	Ordinal                 int    `db:"ordinal"`
	RunID                   string `db:"run_id"`
	ObservedAt              string `db:"observed_at"`
	SessionID               string `db:"session_id"`
	MessageID               string `db:"message_id"`
	ParentToolUseID         string `db:"parent_tool_use_id"`
	Model                   string `db:"model"`
	TurnIndex               int    `db:"turn_index"`
	InputTokens             int    `db:"input_tokens"`
	OutputTokens            int    `db:"output_tokens"`
	CacheReadTokens         int    `db:"cache_read_tokens"`
	CacheCreationTokens     int    `db:"cache_creation_tokens"`
	HasCumulative           bool   `db:"has_cumulative"`
	CumInputTokens          int    `db:"cum_input_tokens"`
	CumOutputTokens         int    `db:"cum_output_tokens"`
	CumCacheReadTokens      int    `db:"cum_cache_read_tokens"`
	CumCacheCreationTokens  int    `db:"cum_cache_creation_tokens"`
	ReconciliationAuthority bool   `db:"reconciliation_authority"`
	ImagesIn                int    `db:"images_in"`
	ImagesInUser            int    `db:"images_in_user"`
	ImagesInHighDetail      int    `db:"images_in_high_detail"`
	ImageBytesIn            int64  `db:"image_bytes_in"`
	ImagesOut               int    `db:"images_out"`
	WebSearches             int    `db:"web_searches"`
	WebFetches              int    `db:"web_fetches"`
	AudioIn                 int    `db:"audio_in"`
	ToolCalls               int    `db:"tool_calls"`
	ToolKindsJSON           string `db:"tool_kinds_json"`
	CreditsBalance          string `db:"credits_balance"`
}

func (r *TranscriptIndexRepository) TranscriptUsageForRun(ctx context.Context, runID string) ([]transcriptindex.UsageRow, error) {
	var rows []transcriptUsageRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT * FROM transcript_usage WHERE run_id = ? ORDER BY observed_at, file_key, line_offset, ordinal`, runID); err != nil {
		return nil, wrapDBError("transcript_usage_for_run", "TranscriptUsage", runID, err)
	}
	out := make([]transcriptindex.UsageRow, 0, len(rows))
	for _, row := range rows {
		usage := transcriptindex.UsageRow{FileKey: row.FileKey, LineOffset: row.LineOffset, Ordinal: row.Ordinal, RunID: row.RunID}
		usage.ObservedAt, _ = time.Parse(time.RFC3339Nano, row.ObservedAt)
		usage.Usage = domain.UsageEventData{
			PayloadKind: domain.PayloadKindUsage, SessionID: row.SessionID, MessageID: row.MessageID, ParentToolUseID: row.ParentToolUseID,
			Model: row.Model, TurnIndex: row.TurnIndex, InputTokens: row.InputTokens, OutputTokens: row.OutputTokens,
			CacheReadTokens: row.CacheReadTokens, CacheCreationTokens: row.CacheCreationTokens, ReconciliationAuthority: row.ReconciliationAuthority,
		}
		if row.HasCumulative {
			usage.Usage.Cumulative = &domain.UsageCounters{InputTokens: row.CumInputTokens, OutputTokens: row.CumOutputTokens, CacheReadTokens: row.CumCacheReadTokens, CacheCreationTokens: row.CumCacheCreationTokens}
		}
		usage.Facts = transcriptindex.TurnFacts{
			ImagesIn: row.ImagesIn, ImagesInUser: row.ImagesInUser, ImagesInHighDetail: row.ImagesInHighDetail, ImageBytesIn: row.ImageBytesIn,
			ImagesOut: row.ImagesOut, WebSearches: row.WebSearches, WebFetches: row.WebFetches, AudioIn: row.AudioIn, ToolCalls: row.ToolCalls,
			CreditsBalance: row.CreditsBalance,
		}
		_ = json.Unmarshal([]byte(row.ToolKindsJSON), &usage.Facts.ToolKinds)
		out = append(out, usage)
	}
	return out, nil
}

// RecallBytes measures the recall layer on its own rows: the run's retained
// event payloads plus its derived conversation-search documents. It never
// reads whole-database size, so native data cannot push recall out.
func (r *TranscriptIndexRepository) RecallBytes(ctx context.Context, runID string) (int64, error) {
	var bytes int64
	query := `SELECT COALESCE((SELECT SUM(LENGTH(data)) FROM run_events WHERE run_id = ?), 0)
		+ COALESCE((SELECT SUM(LENGTH(content)) FROM conversation_search_catalog WHERE source_run_id = ?), 0)`
	if err := r.db.GetContext(ctx, &bytes, query, runID, runID); err != nil {
		return 0, wrapDBError("recall_bytes", "TranscriptFile", runID, err)
	}
	return bytes, nil
}
