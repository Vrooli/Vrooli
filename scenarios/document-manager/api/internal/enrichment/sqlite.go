package enrichment

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db queryer }

func NewSQLiteRepository(db queryer) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) SaveEnrichment(ctx context.Context, e Enrichment) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO enrichments (document_hash, status, summary, suggested_privacy_class, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(document_hash) DO UPDATE SET status=excluded.status, summary=excluded.summary, suggested_privacy_class=excluded.suggested_privacy_class, created_at=excluded.created_at`, e.DocumentHash, e.Status, e.Summary, e.SuggestedPrivacyClass, e.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) GetEnrichment(ctx context.Context, hash string) (Enrichment, error) {
	var e Enrichment
	var created string
	err := r.db.QueryRowContext(ctx, `SELECT document_hash, status, summary, suggested_privacy_class, created_at FROM enrichments WHERE document_hash = ?`, hash).Scan(&e.DocumentHash, &e.Status, &e.Summary, &e.SuggestedPrivacyClass, &created)
	if err != nil {
		return Enrichment{}, err
	}
	e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return e, err
}

func (r *SQLiteRepository) SaveEmbedding(ctx context.Context, e Embedding) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(e.Vector)*4))
	for _, value := range e.Vector {
		if err := binary.Write(buf, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO embeddings (id, document_hash, unit_id, role, model, dimension, content_version, retarget_strategy, vector, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, e.DocumentHash, e.UnitID, e.Role, e.Model, e.Dimension, e.ContentVersion, e.RetargetStrategy, buf.Bytes(), e.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) ListEmbeddings(ctx context.Context, hash string) ([]Embedding, error) {
	return r.listEmbeddings(ctx, `WHERE document_hash = ?`, hash)
}

func (r *SQLiteRepository) ListAllEmbeddings(ctx context.Context) ([]Embedding, error) {
	return r.listEmbeddings(ctx, "")
}

func (r *SQLiteRepository) listEmbeddings(ctx context.Context, where string, args ...any) ([]Embedding, error) {
	query := `SELECT id, document_hash, unit_id, role, model, dimension, content_version, retarget_strategy, vector, created_at FROM embeddings ` + where + ` ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Embedding
	for rows.Next() {
		var e Embedding
		var raw []byte
		var created string
		if err := rows.Scan(&e.ID, &e.DocumentHash, &e.UnitID, &e.Role, &e.Model, &e.Dimension, &e.ContentVersion, &e.RetargetStrategy, &raw, &created); err != nil {
			return nil, err
		}
		if len(raw)%4 != 0 {
			return nil, fmt.Errorf("invalid embedding vector blob")
		}
		e.Vector = make([]float32, len(raw)/4)
		for i := range e.Vector {
			if err := binary.Read(bytes.NewReader(raw[i*4:(i+1)*4]), binary.LittleEndian, &e.Vector[i]); err != nil {
				return nil, err
			}
		}
		e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
