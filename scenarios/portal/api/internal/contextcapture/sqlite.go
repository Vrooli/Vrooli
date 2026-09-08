package contextcapture

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Schema belongs beside the repository interpreting these lifecycle states.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS context_import_intents (
 owner TEXT NOT NULL, request_id TEXT NOT NULL, document_id TEXT NOT NULL,
 digest TEXT NOT NULL, created_at INTEGER NOT NULL, expires_at INTEGER NOT NULL,
 PRIMARY KEY(owner,request_id)
);
CREATE INDEX IF NOT EXISTS idx_context_import_expiry ON context_import_intents(expires_at);
CREATE TABLE IF NOT EXISTS context_capsules (
 id TEXT PRIMARY KEY,
 owner TEXT NOT NULL,
 document_json TEXT NOT NULL,
 image_bytes INTEGER NOT NULL CHECK(image_bytes>0 AND image_bytes<=33554432),
 state TEXT NOT NULL CHECK(state IN ('staging','ready')),
 expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_context_capsules_owner ON context_capsules(owner);
CREATE INDEX IF NOT EXISTS idx_context_capsules_expiry ON context_capsules(expires_at,id);
`
}

type sqliteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Reserve(ctx context.Context, doc Document, size int64) error {
	id, err := uuid.Parse(doc.ID)
	if err != nil || id == uuid.Nil || id.String() != doc.ID || doc.Owner == "" || len(doc.Owner) > 256 || size <= 0 || size > MaxImageBytes || doc.CreatedAt.IsZero() || !doc.ExpiresAt.After(doc.CreatedAt) || doc.ExpiresAt.Sub(doc.CreatedAt) > 24*time.Hour {
		return ErrInvalid
	}
	requestID, requestErr := uuid.Parse(doc.RequestID)
	if requestErr != nil || requestID == uuid.Nil || requestID.String() != doc.RequestID || len(doc.ImportDigest) != 64 {
		return ErrInvalid
	}
	encoded, err := json.Marshal(doc)
	if err != nil || len(encoded) > 512*1024 {
		return ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Acquire SQLite write exclusion before checking shared quotas. Expired and
	// staging rows count until physical blob cleanup permits metadata deletion.
	if _, err = tx.ExecContext(ctx, "UPDATE context_capsules SET state=state WHERE 0"); err != nil {
		return err
	}
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT document_id FROM context_import_intents WHERE owner=? AND request_id=?", doc.Owner, doc.RequestID).Scan(&existing)
	if err == nil {
		return ErrIntentExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var intents, ownedIntents int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(SUM(CASE WHEN owner=? THEN 1 ELSE 0 END),0) FROM context_import_intents", doc.Owner).Scan(&intents, &ownedIntents); err != nil {
		return err
	}
	if intents >= 8192 || ownedIntents >= 1024 {
		return ErrQuota
	}
	var count, ownedCount, used, ownedUsed int64
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(image_bytes),0),COALESCE(SUM(CASE WHEN owner=? THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN owner=? THEN image_bytes ELSE 0 END),0) FROM context_capsules`, doc.Owner, doc.Owner).Scan(&count, &used, &ownedCount, &ownedUsed); err != nil {
		return err
	}
	if count >= 256 || ownedCount >= 64 || used+size > 1024*1024*1024 || ownedUsed+size > 256*1024*1024 {
		return ErrQuota
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO context_capsules(id,owner,document_json,image_bytes,state,expires_at) VALUES(?,?,?,?,?,?)`, doc.ID, doc.Owner, string(encoded), size, Staging, doc.ExpiresAt.UnixNano())
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO context_import_intents(owner,request_id,document_id,digest,created_at,expires_at) VALUES(?,?,?,?,?,?)", doc.Owner, doc.RequestID, doc.ID, doc.ImportDigest, doc.CreatedAt.UnixNano(), doc.CreatedAt.Add(24*time.Hour).UnixNano()); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sqliteRepository) WithRecord(ctx context.Context, owner, id string, fn func(Record) (Action, error)) error {
	if owner == "" || id == "" || fn == nil {
		return ErrUnavailable
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE context_capsules SET state=state WHERE id=? AND owner=?", id, owner)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrUnavailable
	}
	var record Record
	var encoded string
	if err = tx.QueryRowContext(ctx, "SELECT document_json,image_bytes,state FROM context_capsules WHERE id=? AND owner=?", id, owner).Scan(&encoded, &record.Size, &record.State); err != nil {
		return err
	}
	if json.Unmarshal([]byte(encoded), &record.Document) != nil || record.Document.Owner != owner || record.Document.ID != id {
		return ErrUnavailable
	}
	action, err := fn(record)
	if err != nil {
		return err
	}
	switch action {
	case Keep:
	case Publish:
		if record.State != Staging {
			return ErrUnavailable
		}
		if _, err = tx.ExecContext(ctx, "UPDATE context_capsules SET state='ready' WHERE id=? AND owner=?", id, owner); err != nil {
			return err
		}
	case Remove:
		if _, err = tx.ExecContext(ctx, "DELETE FROM context_capsules WHERE id=? AND owner=?", id, owner); err != nil {
			return err
		}
	default:
		return ErrInvalid
	}
	return tx.Commit()
}

func (r *sqliteRepository) Expired(ctx context.Context, now time.Time, limit int) ([]OwnedID, error) {
	if now.IsZero() || limit < 1 || limit > 256 {
		return nil, ErrInvalid
	}
	rows, err := r.db.QueryContext(ctx, "SELECT owner,id FROM context_capsules WHERE expires_at<=? ORDER BY expires_at,id LIMIT ?", now.UnixNano(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []OwnedID{}
	for rows.Next() {
		var id OwnedID
		if err = rows.Scan(&id.Owner, &id.ID); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *sqliteRepository) LookupIntent(ctx context.Context, owner, requestID string) (ImportIntent, error) {
	var result ImportIntent
	var created int64
	err := r.db.QueryRowContext(ctx, "SELECT document_id,digest,created_at FROM context_import_intents WHERE owner=? AND request_id=?", owner, requestID).Scan(&result.DocumentID, &result.Digest, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrUnavailable
	}
	if err != nil {
		return result, err
	}
	result.CreatedAt = time.Unix(0, created).UTC()
	return result, nil
}

func (r *sqliteRepository) PruneIntents(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM context_import_intents WHERE expires_at<=? AND NOT EXISTS(SELECT 1 FROM context_capsules WHERE id=context_import_intents.document_id)", now.UnixNano())
	return err
}

// CancelIntent serializes cancellation against reservation and publication.
// An empty document_id is a bounded negative receipt for an unknown request.
// Existing receipts retain their identity and lifetime after blob removal.
func (r *sqliteRepository) CancelIntent(ctx context.Context, owner, requestID string, now time.Time, remove func(Record) error) error {
	id, err := uuid.Parse(requestID)
	if owner == "" || len(owner) > 256 || err != nil || id == uuid.Nil || id.String() != requestID || now.IsZero() || remove == nil {
		return ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "UPDATE context_capsules SET state=state WHERE 0"); err != nil {
		return err
	}
	var documentID string
	err = tx.QueryRowContext(ctx, "SELECT document_id FROM context_import_intents WHERE owner=? AND request_id=?", owner, requestID).Scan(&documentID)
	if errors.Is(err, sql.ErrNoRows) {
		var total, owned int
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(SUM(CASE WHEN owner=? THEN 1 ELSE 0 END),0) FROM context_import_intents", owner).Scan(&total, &owned); err != nil {
			return err
		}
		if total >= 8192 || owned >= 1024 {
			return ErrQuota
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO context_import_intents(owner,request_id,document_id,digest,created_at,expires_at) VALUES(?,?,'','',?,?)", owner, requestID, now.UnixNano(), now.Add(24*time.Hour).UnixNano()); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if documentID != "" {
		var record Record
		var encoded string
		err = tx.QueryRowContext(ctx, "SELECT document_json,image_bytes,state FROM context_capsules WHERE owner=? AND id=?", owner, documentID).Scan(&encoded, &record.Size, &record.State)
		if err == nil {
			if json.Unmarshal([]byte(encoded), &record.Document) != nil || record.Document.Owner != owner || record.Document.ID != documentID || record.Document.RequestID != requestID {
				return ErrUnavailable
			}
			if err = remove(record); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "DELETE FROM context_capsules WHERE owner=? AND id=?", owner, documentID); err != nil {
				return err
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	return tx.Commit()
}
