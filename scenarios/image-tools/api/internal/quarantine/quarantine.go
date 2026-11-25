// Package quarantine retains refused safety-review outputs for admin review.
//
// A refused output is copied into a private `quarantine/` namespace that the
// ordinary blob route refuses to serve. Only the safety review route can read
// it, and only until its expiry; a janitor purges expired blobs. This is an
// explicit policy choice: the bytes are not published, listed as a result, or
// reachable through `GET /api/v1/blobs/`, but they are recoverable for a bounded
// window so an administrator can decide whether a refusal was a false positive.
package quarantine

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Prefix namespaces every quarantined blob. The ordinary blob route rejects it.
const Prefix = "quarantine/"

// IsKey reports whether a storage key names a quarantined output.
func IsKey(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(filepath.ToSlash(key)), Prefix)
}

// Item is one retained refused output.
type Item struct {
	ID           int64
	Key          string
	JobID        string
	OriginalRef  string
	ReasonCode   string
	SafetyReview string
	Score        float64
	Evidence     string
	Finding      string
	MIME         string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Pending reports whether the item is still reviewable at now.
func (i Item) Pending(now time.Time) bool {
	return now.Before(i.ExpiresAt)
}

// SQLExecutor is the narrow DB surface the store needs.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// BlobStore is the narrow storage surface the store needs (satisfied by
// *storage.Store).
type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, mime string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, key string) error
}

// Store persists quarantined outputs and their blobs.
type Store struct {
	db        SQLExecutor
	blobs     BlobStore
	now       func() time.Time
	retention time.Duration
}

// New constructs the store. retention <= 0 disables quarantine.
func New(db SQLExecutor, blobs BlobStore, retention time.Duration) *Store {
	return &Store{db: db, blobs: blobs, now: time.Now, retention: retention}
}

// Enabled reports whether quarantine is configured.
func (s *Store) Enabled() bool {
	return s != nil && s.db != nil && s.blobs != nil && s.retention > 0
}

// Errors callers translate to a status.
var (
	ErrDisabled = errors.New("safety quarantine is not configured")
	ErrNotFound = errors.New("quarantined output not found")
	ErrExpired  = errors.New("quarantined output has expired")
)

// Quarantine copies data into a private key and records the review entry.
func (s *Store) Quarantine(ctx context.Context, item Item, data []byte) (string, error) {
	if !s.Enabled() {
		return "", ErrDisabled
	}
	if strings.TrimSpace(item.JobID) == "" {
		return "", errors.New("quarantine: job id is required")
	}
	key := quarantineKey(item.JobID, item.MIME)
	mime := strings.TrimSpace(item.MIME)
	if mime == "" {
		mime = "application/octet-stream"
	}
	if err := s.blobs.Put(ctx, key, bytes.NewReader(data), mime); err != nil {
		return "", fmt.Errorf("quarantine: store %q: %w", key, err)
	}
	now := s.now().UTC()
	expires := now.Add(s.retention)
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO safety_quarantine (storage_key, job_id, original_ref, reason_code, safety_review,
  score, evidence, finding, mime, created_at, expires_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		key, strings.TrimSpace(item.JobID), strings.TrimSpace(item.OriginalRef), item.ReasonCode,
		item.SafetyReview, item.Score, item.Evidence, item.Finding, mime,
		now.Format(time.RFC3339Nano), expires.Format(time.RFC3339Nano)); err != nil {
		_ = s.blobs.Delete(ctx, key)
		return "", fmt.Errorf("quarantine: record %q: %w", key, err)
	}
	return key, nil
}

func quarantineKey(jobID, mime string) string {
	segment := safeSegment(jobID)
	if segment == "" {
		segment = "job"
	}
	return Prefix + segment + "/" + uuid.NewString() + extensionFor(mime)
}

func safeSegment(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func extensionFor(mime string) string {
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(mime, ";", 2)[0])) {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/avif":
		return ".avif"
	default:
		return ".png"
	}
}

const itemColumns = `id, storage_key, job_id, original_ref, reason_code, safety_review, score,
  evidence, finding, mime, created_at, expires_at`

func scanItem(scan func(...any) error) (Item, error) {
	var out Item
	var created, expires string
	if err := scan(&out.ID, &out.Key, &out.JobID, &out.OriginalRef, &out.ReasonCode,
		&out.SafetyReview, &out.Score, &out.Evidence, &out.Finding, &out.MIME,
		&created, &expires); err != nil {
		return Item{}, err
	}
	out.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	out.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return out, nil
}

// List returns pending entries newest first and the total pending count.
func (s *Store) List(ctx context.Context, limit, offset int) ([]Item, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM safety_quarantine WHERE expires_at > ?`, now).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("quarantine: count: %w", err)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+itemColumns+` FROM safety_quarantine
		 WHERE expires_at > ?
		 ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, now, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("quarantine: list: %w", err)
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		item, err := scanItem(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("quarantine: scan: %w", err)
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

// Get returns one entry whether or not it has expired.
func (s *Store) Get(ctx context.Context, id int64) (Item, bool, error) {
	if s == nil || s.db == nil {
		return Item{}, false, nil
	}
	item, err := scanItem(s.db.QueryRowContext(ctx,
		`SELECT `+itemColumns+` FROM safety_quarantine WHERE id = ?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, false, nil
	}
	return item, err == nil, err
}

// Open returns the bytes of a pending entry. The caller closes the reader.
func (s *Store) Open(ctx context.Context, id int64) (io.ReadCloser, Item, error) {
	item, found, err := s.Get(ctx, id)
	if err != nil {
		return nil, Item{}, err
	}
	if !found {
		return nil, Item{}, ErrNotFound
	}
	if !item.Pending(s.now().UTC()) {
		return nil, item, ErrExpired
	}
	rc, mime, err := s.blobs.Get(ctx, item.Key)
	if err != nil {
		return nil, item, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if strings.TrimSpace(item.MIME) == "" && strings.TrimSpace(mime) != "" {
		item.MIME = mime
	}
	return rc, item, nil
}

// PurgeExpired deletes expired blobs and their rows. A blob that cannot be
// deleted keeps its row so the next pass retries rather than orphaning bytes.
func (s *Store) PurgeExpired(ctx context.Context, limit int) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, storage_key FROM safety_quarantine
		 WHERE expires_at <= ? ORDER BY expires_at LIMIT ?`,
		s.now().UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return 0, fmt.Errorf("quarantine: list expired: %w", err)
	}
	type expired struct {
		id  int64
		key string
	}
	var pending []expired
	for rows.Next() {
		var row expired
		if err := rows.Scan(&row.id, &row.key); err != nil {
			rows.Close()
			return 0, fmt.Errorf("quarantine: scan expired: %w", err)
		}
		pending = append(pending, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	var errs []error
	removed := 0
	for _, row := range pending {
		if err := s.blobs.Delete(ctx, row.key); err != nil {
			errs = append(errs, fmt.Errorf("quarantine: purge %q: %w", row.key, err))
			continue
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM safety_quarantine WHERE id = ?`, row.id); err != nil {
			errs = append(errs, fmt.Errorf("quarantine: delete row %d: %w", row.id, err))
			continue
		}
		removed++
	}
	return removed, errors.Join(errs...)
}

// RunJanitor purges expired entries now and then every interval until ctx ends.
func RunJanitor(ctx context.Context, store *Store, interval time.Duration, logger *log.Logger) {
	// A store whose retention was changed to 0 still purges rows written under
	// the earlier policy, so the janitor runs even when new quarantine is off.
	if store == nil || store.db == nil || store.blobs == nil {
		return
	}
	if interval <= 0 {
		interval = time.Hour
	}
	if logger == nil {
		logger = log.Default()
	}
	purge := func() {
		total := 0
		for {
			removed, err := store.PurgeExpired(ctx, 200)
			if err != nil {
				logger.Printf("safety quarantine: purge: %v", err)
			}
			total += removed
			// A short batch means the backlog is drained; a failed batch is
			// retried on the next tick rather than spun on.
			if removed < 200 {
				break
			}
		}
		if total > 0 {
			logger.Printf("safety quarantine: purged %d expired output(s)", total)
		}
	}
	purge()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}
