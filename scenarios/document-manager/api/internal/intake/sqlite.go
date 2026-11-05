package intake

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// queryer is kept separate because RoutedDB and *sql.DB expose the same
// context-aware query surface used by this repository.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct{ db queryer }

func NewSQLiteRepository(db queryer) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(d Document) (Document, error) {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(context.Background(), `INSERT INTO documents (id, content_sha256, source_name, detected_mime, pdf_type, pdf_confidence, privacy_class, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, d.ID, d.ContentSHA256, d.SourceName, d.DetectedMIME, d.PDFType, d.PDFConfidence, d.PrivacyClass, d.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Document{}, fmt.Errorf("create document: %w", err)
	}
	return d, nil
}

func (r *sqliteRepository) FindByHash(hash string) (Document, error) {
	return r.get(`SELECT id, content_sha256, source_name, detected_mime, pdf_type, pdf_confidence, privacy_class, created_at FROM documents WHERE content_sha256 = ?`, hash)
}

func (r *sqliteRepository) Get(id string) (Document, error) {
	return r.get(`SELECT id, content_sha256, source_name, detected_mime, pdf_type, pdf_confidence, privacy_class, created_at FROM documents WHERE id = ?`, id)
}

func (r *sqliteRepository) get(query string, arg string) (Document, error) {
	var d Document
	var created string
	err := r.db.QueryRowContext(context.Background(), query, arg).Scan(&d.ID, &d.ContentSHA256, &d.SourceName, &d.DetectedMIME, &d.PDFType, &d.PDFConfidence, &d.PrivacyClass, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{}, ErrNotFound{Key: arg}
	}
	if err != nil {
		return Document{}, err
	}
	d.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Document{}, err
	}
	return d, nil
}

func (r *sqliteRepository) List(limit int) ([]Document, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(context.Background(), `SELECT id, content_sha256, source_name, detected_mime, pdf_type, pdf_confidence, privacy_class, created_at FROM documents ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Document
	for rows.Next() {
		var d Document
		var created string
		if err := rows.Scan(&d.ID, &d.ContentSHA256, &d.SourceName, &d.DetectedMIME, &d.PDFType, &d.PDFConfidence, &d.PrivacyClass, &created); err != nil {
			return nil, err
		}
		d.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) ListSources() ([]string, error) {
	rows, err := r.db.QueryContext(context.Background(), `SELECT DISTINCT source_name FROM documents WHERE source_name <> '' ORDER BY source_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, rows.Err()
}
