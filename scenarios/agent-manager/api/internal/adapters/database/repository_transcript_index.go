// Responsibility: persist the Transcript Indexer's file rows and durable usage summaries.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
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
	FirstSeenAt         string         `db:"first_seen_at"`
	IndexedAt           sql.NullString `db:"indexed_at"`
}

const transcriptFileColumns = `file_key, path, root_kind, root_path, harness, session_id, header_session_id, parent_session_id,
	parent_file_key, sidechain, web_console_session_id, run_id, owner, status, error, read_offset, size, mtime_unix_nano,
	head_hash, turns, usage_reports, active, recall_bytes, recall_evicted, pending_facts_json, first_seen_at, indexed_at`

func (r *TranscriptIndexRepository) ListTranscriptFiles(ctx context.Context) ([]transcriptindex.FileRow, error) {
	var rows []transcriptFileRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT `+transcriptFileColumns+` FROM transcript_files`); err != nil {
		return nil, wrapDBError("list_transcript_files", "TranscriptFile", "", err)
	}
	out := make([]transcriptindex.FileRow, 0, len(rows))
	for _, row := range rows {
		file := transcriptindex.FileRow{
			Key: row.Key, Path: row.Path, RootKind: row.RootKind, RootPath: row.RootPath, Harness: row.Harness,
			SessionID: row.SessionID, HeaderSessionID: row.HeaderSessionID, ParentSessionID: row.ParentSessionID,
			ParentFileKey: row.ParentFileKey, Sidechain: row.Sidechain, WebConsoleSessionID: row.WebConsoleSessionID,
			RunID: row.RunID, Owner: row.Owner, Status: row.Status, Error: row.Error, Offset: row.Offset, Size: row.Size,
			MtimeUnixNano: row.MtimeUnixNano, HeadHash: row.HeadHash, Turns: row.Turns, UsageReports: row.UsageReports,
			Active: row.Active, RecallBytes: row.RecallBytes, RecallEvicted: row.RecallEvicted, PendingFacts: row.PendingFacts,
		}
		file.FirstSeenAt, _ = time.Parse(time.RFC3339Nano, row.FirstSeenAt)
		if row.IndexedAt.Valid {
			if at, err := time.Parse(time.RFC3339Nano, row.IndexedAt.String); err == nil {
				file.IndexedAt = &at
			}
		}
		out = append(out, file)
	}
	return out, nil
}

func (r *TranscriptIndexRepository) SaveTranscriptFile(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("save_transcript_file", "TranscriptFile", file.Key, err)
	}
	defer func() { _ = tx.Rollback() }()
	var indexedAt any
	if file.IndexedAt != nil {
		indexedAt = file.IndexedAt.UTC().Format(time.RFC3339Nano)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO transcript_files (`+transcriptFileColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(file_key) DO UPDATE SET path=excluded.path, root_kind=excluded.root_kind, root_path=excluded.root_path,
			harness=excluded.harness, session_id=excluded.session_id, header_session_id=excluded.header_session_id,
			parent_session_id=excluded.parent_session_id, parent_file_key=excluded.parent_file_key, sidechain=excluded.sidechain,
			web_console_session_id=excluded.web_console_session_id, run_id=excluded.run_id, owner=excluded.owner,
			status=excluded.status, error=excluded.error, read_offset=excluded.read_offset, size=excluded.size,
			mtime_unix_nano=excluded.mtime_unix_nano, head_hash=excluded.head_hash, turns=excluded.turns,
			usage_reports=excluded.usage_reports, active=excluded.active, recall_bytes=excluded.recall_bytes,
			recall_evicted=excluded.recall_evicted, pending_facts_json=excluded.pending_facts_json, indexed_at=excluded.indexed_at`,
		file.Key, file.Path, file.RootKind, file.RootPath, file.Harness, file.SessionID, file.HeaderSessionID, file.ParentSessionID,
		file.ParentFileKey, boolInt(file.Sidechain), file.WebConsoleSessionID, file.RunID, file.Owner, file.Status, file.Error,
		file.Offset, file.Size, file.MtimeUnixNano, file.HeadHash, file.Turns, file.UsageReports, boolInt(file.Active),
		file.RecallBytes, boolInt(file.RecallEvicted), file.PendingFacts, file.FirstSeenAt.UTC().Format(time.RFC3339Nano), indexedAt); err != nil {
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
	if err := tx.Commit(); err != nil {
		return wrapDBError("save_transcript_file", "TranscriptFile", file.Key, err)
	}
	return nil
}

func (r *TranscriptIndexRepository) ResetTranscriptFile(ctx context.Context, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM transcript_usage WHERE file_key = ?`, key); err != nil {
		return wrapDBError("reset_transcript_file", "TranscriptFile", key, err)
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

// attachedExitReason is the error the reconciler wrote when it marked every
// normally exited attached run failed (D-7).
const attachedExitReason = "attached harness process is no longer present"

// migrateAttachedExitStatus corrects attached runs the reconciler marked
// failed only because their harness process exited: a normal exit is
// complete. It queues them for re-projection so the read model follows.
// Idempotent: corrected rows no longer match.
func (db *DB) migrateAttachedExitStatus(ctx context.Context) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	match := `execution_mode = 'attached' AND status = 'failed' AND error_msg = ?`
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_pending (run_id, generation, attempts, next_attempt_unix)
		SELECT id, 1, 0, 0 FROM runs WHERE `+match, attachedExitReason); err != nil {
		return fmt.Errorf("queue attached exit re-projection: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE runs SET status = 'complete', phase = 'completed', error_msg = '' WHERE `+match, attachedExitReason); err != nil {
		return fmt.Errorf("correct attached exit status: %w", err)
	}
	return tx.Commit()
}
