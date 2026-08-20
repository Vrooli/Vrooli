package journal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"source-ledger/internal/policy"
	vectorcodec "source-ledger/internal/vector"

	"github.com/google/uuid"
)

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func (r *SQLiteRepository) Append(ctx context.Context, e Entry, retries []string) (Entry, error) {
	if e.Scope == "" {
		e.Scope = "agent-memory"
	}
	if e.ImportKey != "" {
		var id string
		err := r.db.QueryRowContext(ctx, `SELECT id FROM entries WHERE scope=? AND import_key=?`, e.Scope, e.ImportKey).Scan(&id)
		if err == nil {
			existing, err := r.Get(ctx, id)
			existing.Existing = true
			return existing, err
		}
		if err != sql.ErrNoRows {
			return Entry{}, err
		}
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var importKey any
	if e.ImportKey != "" {
		importKey = e.ImportKey
	}
	importedAt := ""
	if !e.Import.ImportedAt.IsZero() {
		importedAt = e.Import.ImportedAt.Format(time.RFC3339Nano)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO entries (id,scope,body,facet_id,kind,actor_id,actor_kind,source_runtime,verification_status,harness_session_id,harness_kind,run_id,workflow_execution_id,import_key,source_harness,source_path,imported_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.Scope, e.Body, e.FacetID, e.Kind, e.Attribution.ActorID, e.Attribution.ActorKind, e.Attribution.SourceRuntime, e.Attribution.VerificationStatus, e.Attribution.HarnessSessionID, e.Attribution.HarnessKind, e.Correlation.RunID, e.Correlation.WorkflowExecutionID, importKey, e.Import.Harness, e.Import.Path, importedAt, e.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Entry{}, fmt.Errorf("append entry: %w", err)
	}
	var rowID int64
	if err := tx.QueryRowContext(ctx, `SELECT rowid FROM entries WHERE id=?`, e.ID).Scan(&rowID); err != nil {
		return Entry{}, fmt.Errorf("read appended journal rowid: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO journal_high_water_mark(id,max_rowid,recorded_at) VALUES(1,?,?)
ON CONFLICT(id) DO UPDATE SET max_rowid=MAX(journal_high_water_mark.max_rowid,excluded.max_rowid), recorded_at=excluded.recorded_at`, rowID, nowUTC()); err != nil {
		return Entry{}, fmt.Errorf("advance journal high-water mark: %w", err)
	}
	for i := range e.FacetTexts {
		f := &e.FacetTexts[i]
		if f.ID == "" {
			f.ID = uuid.NewString()
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO facet_texts (id,entry_id,kind,text,embedding_ref) VALUES (?,?,?,?,?)`, f.ID, e.ID, f.Kind, f.Text, f.EmbeddingRef)
		if err != nil {
			return Entry{}, err
		}
		if len(f.Vector) > 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO embeddings (id,facet_text_id,vector_json,vector_blob,created_at) VALUES (?,?,?,?,?)`, uuid.NewString(), f.ID, "", vectorcodec.Encode(f.Vector), e.CreatedAt.Format(time.RFC3339Nano))
			if err != nil {
				return Entry{}, err
			}
		}
	}
	for _, reason := range retries {
		_, err = tx.ExecContext(ctx, `INSERT INTO journal_retry_queue (id,entry_id,reason,created_at) VALUES (?,?,?,?)`, uuid.NewString(), e.ID, reason, e.CreatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return Entry{}, err
		}
	}
	return e, tx.Commit()
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Entry, error) {
	entries, err := r.list(ctx, `WHERE e.id = ? AND e.scope = ?`, 1, id, policy.ScopeFromContext(ctx))
	if err != nil {
		return Entry{}, err
	}
	if len(entries) == 0 {
		return Entry{}, sql.ErrNoRows
	}
	return entries[0], nil
}

func (r *SQLiteRepository) List(ctx context.Context, limit int) ([]Entry, error) {
	return r.list(ctx, "WHERE e.scope = ?", limit, policy.ScopeFromContext(ctx))
}

func (r *SQLiteRepository) ListByRun(ctx context.Context, runID string, limit int) ([]Entry, error) {
	if runID == "" {
		return nil, nil
	}
	return r.list(ctx, "WHERE e.run_id = ? AND e.scope = ?", limit, runID, policy.ScopeFromContext(ctx))
}

func (r *SQLiteRepository) FindByImportKey(ctx context.Context, key string) (Entry, bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM entries WHERE scope=? AND import_key=?`, policy.ScopeFromContext(ctx), key).Scan(&id)
	if err == sql.ErrNoRows {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, err
	}
	e, err := r.Get(ctx, id)
	return e, err == nil, err
}

func (r *SQLiteRepository) ClassificationRetries(ctx context.Context, limit int) ([]RetryItem, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT q.id,e.id,e.scope,e.body,e.facet_id,e.kind,e.actor_id,e.actor_kind,e.source_runtime,e.verification_status,e.harness_session_id,e.harness_kind,e.run_id,e.workflow_execution_id,e.import_key,e.source_harness,e.source_path,e.imported_at,e.created_at
FROM journal_retry_queue q JOIN entries e ON e.id=q.entry_id
WHERE q.reason='classify' AND e.scope=? AND NOT EXISTS (SELECT 1 FROM facet_assignments a WHERE a.entry_id=e.id AND a.facet_id<>?)
ORDER BY q.created_at,q.id LIMIT ?`, policy.ScopeFromContext(ctx), UnclassifiedFacet, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RetryItem
	for rows.Next() {
		var item RetryItem
		var created string
		var importKey, importedAt sql.NullString
		if err := rows.Scan(&item.ID, &item.Entry.ID, &item.Entry.Scope, &item.Entry.Body, &item.Entry.FacetID, &item.Entry.Kind, &item.Entry.Attribution.ActorID, &item.Entry.Attribution.ActorKind, &item.Entry.Attribution.SourceRuntime, &item.Entry.Attribution.VerificationStatus, &item.Entry.Attribution.HarnessSessionID, &item.Entry.Attribution.HarnessKind, &item.Entry.Correlation.RunID, &item.Entry.Correlation.WorkflowExecutionID, &importKey, &item.Entry.Import.Harness, &item.Entry.Import.Path, &importedAt, &created); err != nil {
			return nil, err
		}
		item.Reason = "classify"
		item.Entry.ImportKey = importKey.String
		if importedAt.Valid && importedAt.String != "" {
			item.Entry.Import.ImportedAt, _ = time.Parse(time.RFC3339Nano, importedAt.String)
		}
		item.Entry.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) EnqueueUnclassified(ctx context.Context) (int, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO journal_retry_queue(id,entry_id,reason,created_at)
SELECT lower(hex(randomblob(16))),e.id,'classify',?
FROM entries e
WHERE e.scope=? AND NOT EXISTS (SELECT 1 FROM facet_assignments a WHERE a.entry_id=e.id AND a.facet_id<>?)
  AND NOT EXISTS (SELECT 1 FROM journal_retry_queue q WHERE q.entry_id=e.id AND q.reason='classify')`,
		time.Now().UTC().Format(time.RFC3339Nano), policy.ScopeFromContext(ctx), UnclassifiedFacet)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return int(n), err
}

func (r *SQLiteRepository) AcknowledgeRetry(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM journal_retry_queue WHERE id=?`, id)
	return err
}

func (r *SQLiteRepository) PruneResolvedClassificationRetries(ctx context.Context) (int, error) {
	// A sentinel assignment records that classification was attempted and failed,
	// not that it succeeded. Treating it as resolved retired the retry row and
	// stranded the entry outside the facet vocabulary permanently.
	result, err := r.db.ExecContext(ctx, `DELETE FROM journal_retry_queue WHERE reason='classify' AND EXISTS (SELECT 1 FROM entries e WHERE e.id=journal_retry_queue.entry_id AND e.scope=?) AND EXISTS (SELECT 1 FROM facet_assignments a WHERE a.entry_id=journal_retry_queue.entry_id AND a.facet_id<>?)`, policy.ScopeFromContext(ctx), UnclassifiedFacet)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return int(n), err
}

func (r *SQLiteRepository) EmbeddingRetries(ctx context.Context, limit int) ([]RetryItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT q.entry_id FROM journal_retry_queue q JOIN entries je ON je.id=q.entry_id WHERE q.reason='embed' AND je.scope=? AND EXISTS (SELECT 1 FROM facet_texts ft WHERE ft.entry_id=q.entry_id AND NOT EXISTS (SELECT 1 FROM embeddings e WHERE e.facet_text_id=ft.id)) ORDER BY q.entry_id LIMIT ?`, policy.ScopeFromContext(ctx), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entryIDs []string
	for rows.Next() {
		var entryID string
		if err := rows.Scan(&entryID); err != nil {
			return nil, err
		}
		entryIDs = append(entryIDs, entryID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	items := make([]RetryItem, 0, len(entryIDs))
	for _, entryID := range entryIDs {
		entry, err := r.Get(ctx, entryID)
		if err != nil {
			return nil, err
		}
		items = append(items, RetryItem{Reason: "embed", Entry: entry})
	}
	return items, nil
}

func (r *SQLiteRepository) StoreFacetEmbedding(ctx context.Context, facetTextID string, vector []float64) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at) SELECT ?,?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM embeddings WHERE facet_text_id=?)`, uuid.NewString(), facetTextID, "", vectorcodec.Encode(vector), time.Now().UTC().Format(time.RFC3339Nano), facetTextID)
	return err
}

func (r *SQLiteRepository) AcknowledgeEmbeddingRetries(ctx context.Context, entryID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM journal_retry_queue WHERE entry_id=? AND reason='embed'`, entryID)
	return err
}

func (r *SQLiteRepository) PruneResolvedEmbeddingRetries(ctx context.Context) (int, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM journal_retry_queue WHERE reason='embed' AND EXISTS (SELECT 1 FROM entries e WHERE e.id=journal_retry_queue.entry_id AND e.scope=?) AND NOT EXISTS (SELECT 1 FROM facet_texts ft WHERE ft.entry_id=journal_retry_queue.entry_id AND NOT EXISTS (SELECT 1 FROM embeddings e WHERE e.facet_text_id=ft.id))`, policy.ScopeFromContext(ctx))
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return int(n), err
}

func (r *SQLiteRepository) list(ctx context.Context, where string, limit int, args ...any) ([]Entry, error) {
	if limit <= 0 {
		return nil, nil
	}
	q := `SELECT e.id,e.scope,e.body,e.facet_id,e.kind,e.actor_id,e.actor_kind,e.source_runtime,e.verification_status,e.harness_session_id,e.harness_kind,e.run_id,e.workflow_execution_id,e.import_key,e.source_harness,e.source_path,e.imported_at,e.created_at FROM entries e ` + where + ` ORDER BY e.created_at ASC,e.id ASC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var created string
		var importKey, importedAt sql.NullString
		if err = rows.Scan(&e.ID, &e.Scope, &e.Body, &e.FacetID, &e.Kind, &e.Attribution.ActorID, &e.Attribution.ActorKind, &e.Attribution.SourceRuntime, &e.Attribution.VerificationStatus, &e.Attribution.HarnessSessionID, &e.Attribution.HarnessKind, &e.Correlation.RunID, &e.Correlation.WorkflowExecutionID, &importKey, &e.Import.Harness, &e.Import.Path, &importedAt, &created); err != nil {
			return nil, err
		}
		e.ImportKey = importKey.String
		if importedAt.Valid && importedAt.String != "" {
			e.Import.ImportedAt, _ = time.Parse(time.RFC3339Nano, importedAt.String)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		facetTexts, err := r.listFacetTexts(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].FacetTexts = facetTexts
	}
	return out, nil
}

func (r *SQLiteRepository) listFacetTexts(ctx context.Context, entryID string) ([]FacetText, error) {
	fs, err := r.db.QueryContext(ctx, `SELECT id,kind,text,embedding_ref FROM facet_texts WHERE entry_id=? ORDER BY id`, entryID)
	if err != nil {
		return nil, err
	}
	defer fs.Close()

	var facetTexts []FacetText
	for fs.Next() {
		var facetText FacetText
		if err := fs.Scan(&facetText.ID, &facetText.Kind, &facetText.Text, &facetText.EmbeddingRef); err != nil {
			return nil, err
		}
		facetTexts = append(facetTexts, facetText)
	}
	if err := fs.Err(); err != nil {
		return nil, err
	}
	return facetTexts, nil
}
