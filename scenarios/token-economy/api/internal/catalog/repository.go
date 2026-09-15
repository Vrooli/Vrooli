// Package catalog owns minter-declared redeemables and approval posture.
package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Repository interface {
	Create(context.Context, Entry, string) (Entry, error)
	Update(context.Context, Entry, string) (Entry, error)
	Get(context.Context, string) (Entry, error)
	List(context.Context, bool) ([]Entry, error)
	Retire(context.Context, string, time.Time, string) (Entry, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

// ReserveInventory revalidates availability and consumes one declared unit in
// a caller-owned transaction. Redemption uses it before checking spendable
// balance, so SQLite's write lock covers stock, reservations, and settlement.
func ReserveInventory(ctx context.Context, tx *sql.Tx, id string, at time.Time) (Entry, error) {
	entry, err := getEntry(ctx, tx, id)
	if err != nil {
		return Entry{}, err
	}
	if err := availabilityError(entry, at.UTC()); err != nil {
		return Entry{}, err
	}
	if entry.Availability.RemainingQuantity != nil {
		result, err := tx.ExecContext(ctx, `
			UPDATE catalog_entries
			SET remaining_quantity = remaining_quantity - 1, updated_at = ?
			WHERE id = ? AND remaining_quantity > 0`, formatTime(at), id)
		if err != nil {
			return Entry{}, fmt.Errorf("reserve catalog inventory: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return Entry{}, fmt.Errorf("read catalog inventory reservation: %w", err)
		}
		if rows != 1 {
			return Entry{}, &UnavailableCatalogError{Reason: "entry is out of stock"}
		}
	}
	return entry, nil
}

// ReleaseInventory returns a unit reserved by a denied redemption. Entries
// with unlimited quantity require no update.
func ReleaseInventory(ctx context.Context, tx *sql.Tx, id string, at time.Time) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE catalog_entries
		SET remaining_quantity = remaining_quantity + 1, updated_at = ?
		WHERE id = ? AND remaining_quantity IS NOT NULL`, formatTime(at), id)
	if err != nil {
		return fmt.Errorf("release catalog inventory: %w", err)
	}
	return nil
}

func (r *sqliteRepository) Create(ctx context.Context, entry Entry, idempotencyKey string) (Entry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("begin catalog create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if existing, found, err := mutationOutcome(ctx, tx, idempotencyKey); err != nil {
		return Entry{}, err
	} else if found {
		return existing, nil
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_entries (
			id, token_type_id, title, description, cost_amount, available_from,
			available_until, remaining_quantity, approval_posture, retired,
			created_at, updated_at, retired_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.TokenTypeID, entry.Title, entry.Description, entry.CostAmount,
		nullableTime(entry.Availability.AvailableFrom), nullableTime(entry.Availability.AvailableUntil),
		nullableInt(entry.Availability.RemainingQuantity), entry.ApprovalPosture, entry.Retired,
		formatTime(entry.CreatedAt), formatTime(entry.UpdatedAt), nullableTime(entry.RetiredAt))
	if err != nil {
		return Entry{}, fmt.Errorf("insert catalog entry: %w", err)
	}
	if err := recordMutation(ctx, tx, idempotencyKey, entry.ID, "create", entry.UpdatedAt); err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("commit catalog create: %w", err)
	}
	return entry, nil
}

func (r *sqliteRepository) Update(ctx context.Context, entry Entry, idempotencyKey string) (Entry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("begin catalog update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if existing, found, err := mutationOutcome(ctx, tx, idempotencyKey); err != nil {
		return Entry{}, err
	} else if found {
		return existing, nil
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE catalog_entries
		SET token_type_id = ?, title = ?, description = ?, cost_amount = ?,
		    available_from = ?, available_until = ?, remaining_quantity = ?,
		    approval_posture = ?, updated_at = ?
		WHERE id = ? AND retired = 0`,
		entry.TokenTypeID, entry.Title, entry.Description, entry.CostAmount,
		nullableTime(entry.Availability.AvailableFrom), nullableTime(entry.Availability.AvailableUntil),
		nullableInt(entry.Availability.RemainingQuantity), entry.ApprovalPosture,
		formatTime(entry.UpdatedAt), entry.ID)
	if err != nil {
		return Entry{}, fmt.Errorf("update catalog entry: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Entry{}, fmt.Errorf("read catalog update result: %w", err)
	}
	if rows == 0 {
		return Entry{}, ErrEntryNotFound
	}
	if err := recordMutation(ctx, tx, idempotencyKey, entry.ID, "update", entry.UpdatedAt); err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("commit catalog update: %w", err)
	}
	return entry, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Entry, error) {
	return getEntry(ctx, r.db, id)
}

func (r *sqliteRepository) List(ctx context.Context, includeRetired bool) ([]Entry, error) {
	query := catalogSelect
	if !includeRetired {
		query += " WHERE retired = 0"
	}
	query += " ORDER BY title ASC, id ASC"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list catalog entries: %w", err)
	}
	defer rows.Close()
	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog entries: %w", err)
	}
	return entries, nil
}

func (r *sqliteRepository) Retire(ctx context.Context, id string, retiredAt time.Time, idempotencyKey string) (Entry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("begin catalog retire: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if existing, found, err := mutationOutcome(ctx, tx, idempotencyKey); err != nil {
		return Entry{}, err
	} else if found {
		return existing, nil
	}
	entry, err := getEntry(ctx, tx, id)
	if err != nil {
		return Entry{}, err
	}
	if !entry.Retired {
		_, err = tx.ExecContext(ctx, `
			UPDATE catalog_entries SET retired = 1, retired_at = ?, updated_at = ? WHERE id = ?`,
			formatTime(retiredAt), formatTime(retiredAt), id)
		if err != nil {
			return Entry{}, fmt.Errorf("retire catalog entry: %w", err)
		}
		entry.Retired = true
		entry.RetiredAt = timePointer(retiredAt)
		entry.UpdatedAt = retiredAt
	}
	if err := recordMutation(ctx, tx, idempotencyKey, id, "retire", retiredAt); err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("commit catalog retire: %w", err)
	}
	return entry, nil
}

const catalogSelect = `
	SELECT id, token_type_id, title, description, cost_amount, available_from,
	       available_until, remaining_quantity, approval_posture, retired,
	       created_at, updated_at, retired_at
	FROM catalog_entries`

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type rowScanner interface{ Scan(...any) error }

func getEntry(ctx context.Context, db queryer, id string) (Entry, error) {
	entry, err := scanEntry(db.QueryRowContext(ctx, catalogSelect+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrEntryNotFound
	}
	return entry, err
}

func scanEntry(row rowScanner) (Entry, error) {
	var entry Entry
	var availableFrom, availableUntil, retiredAt sql.NullString
	var remaining sql.NullInt64
	var createdAt, updatedAt string
	err := row.Scan(
		&entry.ID, &entry.TokenTypeID, &entry.Title, &entry.Description, &entry.CostAmount,
		&availableFrom, &availableUntil, &remaining, &entry.ApprovalPosture, &entry.Retired,
		&createdAt, &updatedAt, &retiredAt,
	)
	if err != nil {
		return Entry{}, err
	}
	entry.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Entry{}, fmt.Errorf("parse catalog creation time: %w", err)
	}
	entry.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Entry{}, fmt.Errorf("parse catalog update time: %w", err)
	}
	entry.Availability.AvailableFrom, err = parseNullableTime(availableFrom)
	if err != nil {
		return Entry{}, fmt.Errorf("parse catalog availability start: %w", err)
	}
	entry.Availability.AvailableUntil, err = parseNullableTime(availableUntil)
	if err != nil {
		return Entry{}, fmt.Errorf("parse catalog availability end: %w", err)
	}
	entry.Availability.RemainingQuantity, err = parseNullableInt(remaining)
	if err != nil {
		return Entry{}, fmt.Errorf("parse catalog remaining quantity: %w", err)
	}
	entry.RetiredAt, err = parseNullableTime(retiredAt)
	if err != nil {
		return Entry{}, fmt.Errorf("parse catalog retirement time: %w", err)
	}
	return entry, nil
}

func mutationOutcome(ctx context.Context, tx *sql.Tx, key string) (Entry, bool, error) {
	var entryID string
	err := tx.QueryRowContext(ctx, "SELECT entry_id FROM catalog_mutations WHERE idempotency_key = ?", key).Scan(&entryID)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("read catalog mutation: %w", err)
	}
	entry, err := getEntry(ctx, tx, entryID)
	return entry, true, err
}

func recordMutation(ctx context.Context, tx *sql.Tx, key, entryID, operation string, createdAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_mutations (idempotency_key, entry_id, operation, created_at)
		VALUES (?, ?, ?, ?)`, key, entryID, operation, formatTime(createdAt))
	if err != nil {
		return fmt.Errorf("record catalog mutation: %w", err)
	}
	return nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func nullableInt(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	return timePointer(parsed), err
}

func parseNullableInt(value sql.NullInt64) (*int64, error) {
	if !value.Valid {
		return nil, nil
	}
	return &value.Int64, nil
}

func timePointer(value time.Time) *time.Time { return &value }

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
