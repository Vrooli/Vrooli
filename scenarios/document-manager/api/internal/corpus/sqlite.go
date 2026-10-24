package corpus

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db queryer }

func NewSQLiteRepository(db queryer) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) CreateCollection(ctx context.Context, c Collection) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO collections (id, name, default_privacy_class, federated, created_at) VALUES (?, ?, ?, ?, ?)`, c.ID, c.Name, c.DefaultPrivacyClass, c.Federated, c.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetCollection(ctx context.Context, id string) (Collection, error) {
	var c Collection
	var created string
	var federated bool
	err := r.db.QueryRowContext(ctx, `SELECT id, name, default_privacy_class, federated, created_at FROM collections WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.DefaultPrivacyClass, &federated, &created)
	if err != nil {
		return Collection{}, err
	}
	c.Federated = federated
	c.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return c, err
}

func (r *SQLiteRepository) ListCollections(ctx context.Context, limit int) ([]Collection, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, default_privacy_class, federated, created_at FROM collections ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collection
	for rows.Next() {
		var c Collection
		var created string
		if err := rows.Scan(&c.ID, &c.Name, &c.DefaultPrivacyClass, &c.Federated, &created); err != nil {
			return nil, err
		}
		c.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) AddDocument(ctx context.Context, membership Membership) error {
	c, err := r.GetCollection(ctx, membership.CollectionID)
	if err != nil {
		return err
	}
	if err := ValidatePrivacyInheritance(c.DefaultPrivacyClass, membership.PrivacyClass); err != nil {
		return err
	}
	if membership.CreatedAt.IsZero() {
		membership.CreatedAt = time.Now().UTC()
	}
	_, err = r.db.ExecContext(ctx, `INSERT OR IGNORE INTO collection_documents (collection_id, document_hash, privacy_class, created_at) VALUES (?, ?, ?, ?)`, membership.CollectionID, membership.DocumentHash, membership.PrivacyClass, membership.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) ListDocuments(ctx context.Context, collectionID string, limit int) ([]Membership, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT collection_id, document_hash, privacy_class, created_at FROM collection_documents WHERE collection_id = ? ORDER BY created_at DESC, document_hash DESC LIMIT ?`, collectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Membership
	for rows.Next() {
		var m Membership
		var created string
		if err := rows.Scan(&m.CollectionID, &m.DocumentHash, &m.PrivacyClass, &created); err != nil {
			return nil, err
		}
		m.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) ListAnchors(ctx context.Context, collectionID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT anchor_uri FROM collection_anchors WHERE collection_id = ? ORDER BY anchor_uri`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err != nil {
			return nil, err
		}
		out = append(out, uri)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) AddAnchor(ctx context.Context, collectionID, uri string) error {
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO collection_anchors (collection_id, anchor_uri) VALUES (?, ?)`, collectionID, uri)
	return err
}

func (r *SQLiteRepository) CanRead(ctx context.Context, collectionID string, class documentpb.PrivacyClass) (bool, error) {
	c, err := r.GetCollection(ctx, collectionID)
	if err != nil {
		return false, err
	}
	// Federation is anonymous in the current search-hub contract, so only
	// public/internal collections may opt in. Confidential and secret remain
	// excluded regardless of the flag.
	return c.Federated && class <= documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL, nil
}
