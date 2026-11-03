package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("ledger emission not found")

type Status string

const (
	StatusQueued   Status = "queued"
	StatusAccepted Status = "accepted"
)

type Emission struct {
	ID, SettlementID, ExternalID, AdapterID, AccountID, BookID string
	AmountMinor                                                int64
	Currency, Basis, Description, LastError                    string
	Status                                                     Status
	Attempts                                                   int
	OccurredAt, FetchedAt, CreatedAt, AcceptedAt               time.Time
}

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository interface {
	Pending(context.Context, int) ([]Emission, error)
	List(context.Context, string, string, int) ([]Emission, error)
	GetBySettlement(context.Context, string) (Emission, error)
	MarkFailure(context.Context, string, string) error
	MarkAccepted(context.Context, string, time.Time) error
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Pending(ctx context.Context, limit int) ([]Emission, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, selectEmission+` WHERE status='queued' ORDER BY created_at,id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []Emission
	for rows.Next() {
		value, err := scanEmission(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

// List returns a bounded operator read of durable emission state. Exact external
// identity filtering lets an operator correlate one source occurrence across
// Treasury and Money Ledger without reading the whole journal.
func (r *SQLiteRepository) List(ctx context.Context, externalID, status string, limit int) ([]Emission, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && status != string(StatusQueued) && status != string(StatusAccepted) {
		return nil, fmt.Errorf("unsupported ledger emission status %q", status)
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := selectEmission
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if externalID = strings.TrimSpace(externalID); externalID != "" {
		conditions = append(conditions, "external_id=?")
		args = append(args, externalID)
	}
	if status != "" {
		conditions = append(conditions, "status=?")
		args = append(args, status)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC,id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Emission, 0)
	for rows.Next() {
		value, err := scanEmission(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *SQLiteRepository) GetBySettlement(ctx context.Context, settlementID string) (Emission, error) {
	return scanEmission(r.db.QueryRowContext(ctx, selectEmission+` WHERE settlement_id=?`, settlementID))
}

func (r *SQLiteRepository) MarkFailure(ctx context.Context, id, detail string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE ledger_emissions SET attempts=attempts+1,last_error=? WHERE id=? AND status='queued'`, detail, id)
	return err
}

func (r *SQLiteRepository) MarkAccepted(ctx context.Context, id string, acceptedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE ledger_emissions SET status='accepted',attempts=attempts+1,last_error='',accepted_at=? WHERE id=? AND status='queued'`, acceptedAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		current, getErr := r.GetByID(ctx, id)
		if getErr == nil && current.Status == StatusAccepted {
			return nil
		}
		return fmt.Errorf("mark ledger emission accepted: queued emission not found")
	}
	return nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (Emission, error) {
	return scanEmission(r.db.QueryRowContext(ctx, selectEmission+` WHERE id=?`, id))
}

const selectEmission = `SELECT id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at FROM ledger_emissions`

type scanner interface{ Scan(...any) error }

func scanEmission(row scanner) (Emission, error) {
	var value Emission
	var occurredAt, fetchedAt, createdAt, acceptedAt, status string
	err := row.Scan(&value.ID, &value.SettlementID, &value.ExternalID, &value.AdapterID, &value.AccountID, &value.BookID, &value.AmountMinor, &value.Currency, &value.Basis, &occurredAt, &fetchedAt, &value.Description, &status, &value.Attempts, &value.LastError, &createdAt, &acceptedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Emission{}, ErrNotFound
	}
	if err != nil {
		return Emission{}, err
	}
	value.Status = Status(status)
	if value.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt); err != nil {
		return Emission{}, err
	}
	if value.FetchedAt, err = time.Parse(time.RFC3339Nano, fetchedAt); err != nil {
		return Emission{}, err
	}
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Emission{}, err
	}
	if acceptedAt != "" {
		value.AcceptedAt, err = time.Parse(time.RFC3339Nano, acceptedAt)
	}
	return value, err
}

var _ Repository = (*SQLiteRepository)(nil)
