package transfer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer (seam-discovery): both *sql.DB (unit tests via
// testutil/db.NewSQLite) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteRepository is the production Repository impl. Unexported so callers
// depend on the Repository interface and tests substitute the fake.
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository. db is the pool from
// main.go; clk supplies timestamps so tests advance time deterministically.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// itemTimeFormat matches the wire format and the devices domain's round-trip:
// RFC3339Nano sorts lexicographically in time order for a fixed zone, so string
// range comparisons on expires_at are correct purge filters. Empty string sorts
// before any timestamp, which is why Pinned (” expires_at) never matches the
// `expires_at != ” AND expires_at <= now` purge predicate.
const itemTimeFormat = time.RFC3339Nano

const (
	insertItemSQL = `
INSERT INTO items (id, owner_id, origin_device_id, kind, name, mime, size_bytes,
                   text_content, blob_key, thumb_key, retention, target_device_id,
                   delivered, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	itemColumns = `id, owner_id, origin_device_id, kind, name, mime, size_bytes,
                   text_content, blob_key, thumb_key, retention, target_device_id,
                   delivered, expires_at, created_at`

	// visibilityClause is the delivery ACL: broadcast, directed-to-me, or mine.
	visibilityClause = `owner_id = ? AND (target_device_id = '' OR target_device_id = ? OR origin_device_id = ?)`

	selectVisibleItemSQL = `SELECT ` + itemColumns + ` FROM items WHERE ` + visibilityClause + ` AND id = ?`

	selectOwnerItemSQL = `SELECT ` + itemColumns + ` FROM items WHERE owner_id = ? AND id = ?`

	deleteItemSQL = `DELETE FROM items WHERE owner_id = ? AND id = ?`

	markDeliveredSQL = `UPDATE items SET delivered = 1 WHERE owner_id = ? AND id = ?`

	usageByOwnerSQL = `SELECT COALESCE(SUM(size_bytes), 0) FROM items WHERE owner_id = ?`

	usageByDeviceSQL = `SELECT COALESCE(SUM(size_bytes), 0) FROM items WHERE owner_id = ? AND origin_device_id = ?`

	dueForPurgeSQL = `SELECT ` + itemColumns + ` FROM items
WHERE (expires_at != '' AND expires_at <= ?) OR (retention = 'live' AND delivered = 1)`

	purgeByIDSQL = `DELETE FROM items WHERE id = ?`
)

func (r *sqliteRepository) now() time.Time { return r.clock.Now().UTC() }

func (r *sqliteRepository) Create(ctx context.Context, item Item) (Item, error) {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = r.now()
	}
	delivered := 0
	if item.Delivered {
		delivered = 1
	}
	_, err := r.db.ExecContext(ctx, insertItemSQL,
		item.ID, item.OwnerID, item.OriginDeviceID, string(item.Kind), item.Name,
		item.MIME, item.SizeBytes, item.Text, item.BlobKey, item.ThumbKey,
		string(item.Retention), item.TargetDeviceID, delivered,
		formatExpiry(item.ExpiresAt), item.CreatedAt.Format(itemTimeFormat),
	)
	if err != nil {
		return Item{}, fmt.Errorf("insert item %q: %w", item.ID, err)
	}
	return item, nil
}

func (r *sqliteRepository) GetVisible(ctx context.Context, ownerID, deviceID, id string) (Item, error) {
	row := r.db.QueryRowContext(ctx, selectVisibleItemSQL, ownerID, deviceID, deviceID, id)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrItemNotFound{ID: id}
	}
	if err != nil {
		return Item{}, fmt.Errorf("get item %q: %w", id, err)
	}
	return item, nil
}

func (r *sqliteRepository) GetByOwner(ctx context.Context, ownerID, id string) (Item, error) {
	row := r.db.QueryRowContext(ctx, selectOwnerItemSQL, ownerID, id)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrItemNotFound{ID: id}
	}
	if err != nil {
		return Item{}, fmt.Errorf("get item %q: %w", id, err)
	}
	return item, nil
}

func (r *sqliteRepository) ListVisible(ctx context.Context, ownerID, deviceID string, f ListFilter) ([]Item, error) {
	var b strings.Builder
	b.WriteString(`SELECT ` + itemColumns + ` FROM items WHERE ` + visibilityClause)
	args := []any{ownerID, deviceID, deviceID}
	if f.Kind == KindText || f.Kind == KindFile {
		b.WriteString(` AND kind = ?`)
		args = append(args, string(f.Kind))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		b.WriteString(` AND (LOWER(name) LIKE ? OR LOWER(text_content) LIKE ?)`)
		like := "%" + strings.ToLower(q) + "%"
		args = append(args, like, like)
	}
	b.WriteString(` ORDER BY created_at DESC, id DESC`)

	rows, err := r.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) Delete(ctx context.Context, ownerID, id string) (Item, error) {
	item, err := r.GetByOwner(ctx, ownerID, id)
	if err != nil {
		return Item{}, err
	}
	if _, err := r.db.ExecContext(ctx, deleteItemSQL, ownerID, id); err != nil {
		return Item{}, fmt.Errorf("delete item %q: %w", id, err)
	}
	return item, nil
}

func (r *sqliteRepository) MarkDelivered(ctx context.Context, ownerID, id string) error {
	if _, err := r.db.ExecContext(ctx, markDeliveredSQL, ownerID, id); err != nil {
		return fmt.Errorf("mark item %q delivered: %w", id, err)
	}
	return nil
}

func (r *sqliteRepository) UsageByOwner(ctx context.Context, ownerID string) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, usageByOwnerSQL, ownerID).Scan(&total); err != nil {
		return 0, fmt.Errorf("owner usage: %w", err)
	}
	return total, nil
}

func (r *sqliteRepository) UsageByDevice(ctx context.Context, ownerID, deviceID string) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, usageByDeviceSQL, ownerID, deviceID).Scan(&total); err != nil {
		return 0, fmt.Errorf("device usage: %w", err)
	}
	return total, nil
}

func (r *sqliteRepository) DueForPurge(ctx context.Context, now time.Time) ([]Item, error) {
	rows, err := r.db.QueryContext(ctx, dueForPurgeSQL, now.UTC().Format(itemTimeFormat))
	if err != nil {
		return nil, fmt.Errorf("scan purge-due items: %w", err)
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan purge item: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purge items: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) PurgeByID(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, purgeByIDSQL, id); err != nil {
		return fmt.Errorf("purge item %q: %w", id, err)
	}
	return nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(s rowScanner) (Item, error) {
	var (
		item       Item
		kind       string
		retention  string
		delivered  int
		expiresRaw string
		createdRaw string
	)
	if err := s.Scan(
		&item.ID, &item.OwnerID, &item.OriginDeviceID, &kind, &item.Name,
		&item.MIME, &item.SizeBytes, &item.Text, &item.BlobKey, &item.ThumbKey,
		&retention, &item.TargetDeviceID, &delivered, &expiresRaw, &createdRaw,
	); err != nil {
		return Item{}, err
	}
	item.Kind = Kind(kind)
	item.Retention = Retention(retention)
	item.Delivered = delivered != 0

	var err error
	if expiresRaw != "" {
		if item.ExpiresAt, err = time.Parse(itemTimeFormat, expiresRaw); err != nil {
			return Item{}, fmt.Errorf("parse expires_at %q: %w", expiresRaw, err)
		}
	}
	if item.CreatedAt, err = time.Parse(itemTimeFormat, createdRaw); err != nil {
		return Item{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	return item, nil
}

// formatExpiry renders a zero time as the empty string (the Pinned sentinel) so
// the purge predicate `expires_at != ”` correctly excludes Pinned items.
func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(itemTimeFormat)
}
