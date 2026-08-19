package settlement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type DB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Record, error) {
	return scanRecord(r.db.QueryRowContext(ctx, selectSettlement+` WHERE id=?`, id))
}

func (r *SQLiteRepository) GetByIdempotencyKey(ctx context.Context, key string) (Record, error) {
	return scanRecord(r.db.QueryRowContext(ctx, selectSettlement+` WHERE idempotency_key=?`, key))
}

// Claim performs the unique-key insert and ready->calling transition in one
// transaction. The committed calling row is the durable fence before the
// external side effect; no second process can acquire the same key.
func (r *SQLiteRepository) Claim(ctx context.Context, value Record) (ClaimResult, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return ClaimResult{}, fmt.Errorf("begin settlement claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `INSERT INTO settlements(id,authorization_id,mandate_id,instrument_id,rail,idempotency_key,amount_minor,currency,counterparty,outcome,external_id,receipt_reference,basis,detail,occurred_at,created_at,updated_at,retain_until) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, value.ID, value.AuthorizationID, value.MandateID, value.InstrumentID, value.Rail, value.IdempotencyKey, value.AmountMinor, value.Currency, value.Counterparty, string(OutcomeReady), "", "", "", "", "", value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano), value.RetainUntil.Format(time.RFC3339Nano))
	if err != nil {
		return ClaimResult{}, fmt.Errorf("insert settlement claim: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return ClaimResult{}, err
	}
	claimed := inserted == 1
	if claimed {
		result, err = tx.ExecContext(ctx, `UPDATE settlements SET outcome='calling' WHERE id=? AND outcome='ready'`, value.ID)
		if err != nil {
			return ClaimResult{}, fmt.Errorf("lock settlement call: %w", err)
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil || rows != 1 {
			if rowsErr != nil {
				return ClaimResult{}, rowsErr
			}
			return ClaimResult{}, fmt.Errorf("lock settlement call: expected one ready row, updated %d", rows)
		}
	}
	stored, err := scanRecord(tx.QueryRowContext(ctx, selectSettlement+` WHERE idempotency_key=?`, value.IdempotencyKey))
	if err != nil {
		return ClaimResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ClaimResult{}, fmt.Errorf("commit settlement claim: %w", err)
	}
	return ClaimResult{Record: stored, Claimed: claimed}, nil
}

func (r *SQLiteRepository) Complete(ctx context.Context, id string, outcome Outcome, result RailResult, updatedAt, retainUntil string) (Record, error) {
	if outcome != OutcomeSettled && outcome != OutcomeFailed && outcome != OutcomeUnknown {
		return Record{}, fmt.Errorf("complete settlement: invalid terminal outcome %q", outcome)
	}
	query := `UPDATE settlements SET outcome=?,external_id=?,receipt_reference=?,basis=?,detail=?,occurred_at=?,updated_at=?,retain_until=? WHERE id=? AND outcome='calling'`
	if result.FromQuery {
		query = `UPDATE settlements SET outcome=?,external_id=?,receipt_reference=?,basis=?,detail=?,occurred_at=?,updated_at=?,retain_until=? WHERE id=? AND outcome='unknown'`
	}
	execResult, err := r.exec(ctx, query, string(outcome), result.ExternalID, result.ReceiptReference, result.Basis, result.Detail, formatOptionalTime(result.OccurredAt), updatedAt, retainUntil, id)
	if err != nil {
		return Record{}, err
	}
	rows, err := execResult.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if rows == 0 {
		current, getErr := r.Get(ctx, id)
		if getErr == nil && current.Outcome == outcome {
			return current, nil
		}
		return Record{}, fmt.Errorf("complete settlement: transition rejected from current state")
	}
	return r.Get(ctx, id)
}

func (r *SQLiteRepository) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

const selectSettlement = `SELECT id,authorization_id,mandate_id,instrument_id,rail,idempotency_key,amount_minor,currency,counterparty,outcome,external_id,receipt_reference,basis,detail,occurred_at,created_at,updated_at,retain_until FROM settlements`

type scanner interface{ Scan(...any) error }

func scanRecord(row scanner) (Record, error) {
	var value Record
	var outcome, occurredAt, createdAt, updatedAt, retainUntil string
	err := row.Scan(&value.ID, &value.AuthorizationID, &value.MandateID, &value.InstrumentID, &value.Rail, &value.IdempotencyKey, &value.AmountMinor, &value.Currency, &value.Counterparty, &outcome, &value.ExternalID, &value.ReceiptReference, &value.Basis, &value.Detail, &occurredAt, &createdAt, &updatedAt, &retainUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, err
	}
	value.Outcome = Outcome(outcome)
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Record{}, err
	}
	if value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return Record{}, err
	}
	if value.RetainUntil, err = time.Parse(time.RFC3339Nano, retainUntil); err != nil {
		return Record{}, err
	}
	if occurredAt != "" {
		value.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
	}
	return value, err
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

var _ Repository = (*SQLiteRepository)(nil)
