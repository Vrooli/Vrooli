package journal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func (r *SQLiteRepository) Append(ctx context.Context, e Entry, retries []string) (Entry, error) {
	if e.ImportKey != "" {
		var id string
		err := r.db.QueryRowContext(ctx, `SELECT id FROM entries WHERE import_key=?`, e.ImportKey).Scan(&id)
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
	_, err = tx.ExecContext(ctx, `INSERT INTO entries (id,body,facet_id,kind,actor_id,actor_kind,source_runtime,run_id,workflow_execution_id,import_key,source_harness,source_path,imported_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.Body, e.FacetID, e.Kind, e.Attribution.ActorID, e.Attribution.ActorKind, e.Attribution.SourceRuntime, e.Correlation.RunID, e.Correlation.WorkflowExecutionID, importKey, e.Import.Harness, e.Import.Path, importedAt, e.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Entry{}, fmt.Errorf("append entry: %w", err)
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
			b, _ := json.Marshal(f.Vector)
			_, err = tx.ExecContext(ctx, `INSERT INTO embeddings (id,facet_text_id,vector_json,created_at) VALUES (?,?,?,?)`, uuid.NewString(), f.ID, string(b), e.CreatedAt.Format(time.RFC3339Nano))
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
	entries, err := r.list(ctx, `WHERE e.id = ?`, 1, id)
	if err != nil {
		return Entry{}, err
	}
	if len(entries) == 0 {
		return Entry{}, sql.ErrNoRows
	}
	return entries[0], nil
}

func (r *SQLiteRepository) List(ctx context.Context, limit int) ([]Entry, error) {
	return r.list(ctx, "", limit)
}

func (r *SQLiteRepository) FindByImportKey(ctx context.Context, key string) (Entry, bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM entries WHERE import_key=?`, key).Scan(&id)
	if err == sql.ErrNoRows {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, err
	}
	e, err := r.Get(ctx, id)
	return e, err == nil, err
}

func (r *SQLiteRepository) list(ctx context.Context, where string, limit int, args ...any) ([]Entry, error) {
	if limit <= 0 {
		return nil, nil
	}
	q := `SELECT e.id,e.body,e.facet_id,e.kind,e.actor_id,e.actor_kind,e.source_runtime,e.run_id,e.workflow_execution_id,e.import_key,e.source_harness,e.source_path,e.imported_at,e.created_at FROM entries e ` + where + ` ORDER BY e.created_at ASC,e.id ASC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for rows.Next() {
		var e Entry
		var created string
		var importKey, importedAt sql.NullString
		if err = rows.Scan(&e.ID, &e.Body, &e.FacetID, &e.Kind, &e.Attribution.ActorID, &e.Attribution.ActorKind, &e.Attribution.SourceRuntime, &e.Correlation.RunID, &e.Correlation.WorkflowExecutionID, &importKey, &e.Import.Harness, &e.Import.Path, &importedAt, &created); err != nil {
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
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		fs, err := r.db.QueryContext(ctx, `SELECT id,kind,text,embedding_ref FROM facet_texts WHERE entry_id=? ORDER BY id`, out[i].ID)
		if err != nil {
			return nil, err
		}
		for fs.Next() {
			var f FacetText
			if err = fs.Scan(&f.ID, &f.Kind, &f.Text, &f.EmbeddingRef); err != nil {
				fs.Close()
				return nil, err
			}
			out[i].FacetTexts = append(out[i].FacetTexts, f)
		}
		if err := fs.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
