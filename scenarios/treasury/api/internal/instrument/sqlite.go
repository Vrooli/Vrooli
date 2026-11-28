package instrument

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type DB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Instrument) (Instrument, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO instruments(id,book_id,mandate_id,rail,credential_reference,cap_minor,currency,counterparty,expires_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, value.ID, value.BookID, value.MandateID, value.Rail, value.CredentialReference, value.CapMinor, value.Currency, value.Counterparty, value.ExpiresAt.Format(time.RFC3339Nano), value.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Instrument{}, fmt.Errorf("create instrument: %w", err)
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Instrument, error) {
	var value Instrument
	var expiresAt, createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT id,book_id,mandate_id,rail,credential_reference,cap_minor,currency,counterparty,expires_at,created_at FROM instruments WHERE id=?`, id).Scan(&value.ID, &value.BookID, &value.MandateID, &value.Rail, &value.CredentialReference, &value.CapMinor, &value.Currency, &value.Counterparty, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Instrument{}, ErrNotFound
	}
	if err != nil {
		return Instrument{}, err
	}
	if value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt); err != nil {
		return Instrument{}, err
	}
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Instrument{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) GetCardIssueAttempt(ctx context.Context, instrumentID string) (CardIssueAttempt, error) {
	return scanCardIssueAttempt(r.db.QueryRowContext(ctx, `SELECT fingerprint,status,created_at,updated_at FROM card_issuance_attempts WHERE instrument_id=?`, instrumentID))
}

func (r *SQLiteRepository) BeginCardIssue(ctx context.Context, instrumentID, fingerprint string, now time.Time) (CardIssueAttempt, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return CardIssueAttempt{}, err
	}
	defer func() { _ = tx.Rollback() }()
	current, err := scanCardIssueAttempt(tx.QueryRowContext(ctx, `SELECT fingerprint,status,created_at,updated_at FROM card_issuance_attempts WHERE instrument_id=?`, instrumentID))
	if errors.Is(err, ErrNotFound) {
		value := CardIssueAttempt{Fingerprint: fingerprint, Status: "in_flight", CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
		if _, err := tx.ExecContext(ctx, `INSERT INTO card_issuance_attempts(instrument_id,fingerprint,status,created_at,updated_at) VALUES(?,?,?,?,?)`, instrumentID, fingerprint, value.Status, value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
			return CardIssueAttempt{}, err
		}
		if err := tx.Commit(); err != nil {
			return CardIssueAttempt{}, err
		}
		return value, nil
	}
	if err != nil {
		return CardIssueAttempt{}, err
	}
	if current.Fingerprint != fingerprint {
		return CardIssueAttempt{}, fmt.Errorf("%w: instrument id has an unresolved card issue with different request terms", ErrInvalid)
	}
	if current.Status == "complete" {
		if err := tx.Commit(); err != nil {
			return CardIssueAttempt{}, err
		}
		return current, nil
	}
	if !now.Before(current.CreatedAt.Add(cardIssueIdempotencyRetention)) {
		return CardIssueAttempt{}, ErrCardIssueReconciliation
	}
	if current.Status == "in_flight" && now.Before(current.UpdatedAt.Add(cardIssueInFlightLease)) {
		return CardIssueAttempt{}, ErrCardIssueInProgress
	}
	current.Status = "in_flight"
	current.UpdatedAt = now.UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE card_issuance_attempts SET status='in_flight',updated_at=? WHERE instrument_id=? AND fingerprint=?`, current.UpdatedAt.Format(time.RFC3339Nano), instrumentID, fingerprint); err != nil {
		return CardIssueAttempt{}, err
	}
	if err := tx.Commit(); err != nil {
		return CardIssueAttempt{}, err
	}
	return current, nil
}

func (r *SQLiteRepository) SetCardIssueAttemptStatus(ctx context.Context, instrumentID, fingerprint, status string, now time.Time) error {
	if status != "in_flight" && status != "unknown" && status != "complete" {
		return fmt.Errorf("%w: invalid card issue attempt status", ErrInvalid)
	}
	result, err := r.db.ExecContext(ctx, `UPDATE card_issuance_attempts SET status=?,updated_at=? WHERE instrument_id=? AND fingerprint=? AND (status!='complete' OR ?='complete')`, status, now.UTC().Format(time.RFC3339Nano), instrumentID, fingerprint, status)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		current, getErr := r.GetCardIssueAttempt(ctx, instrumentID)
		if getErr != nil {
			return getErr
		}
		if current.Fingerprint != fingerprint {
			return fmt.Errorf("%w: card issue attempt fingerprint changed", ErrInvalid)
		}
		if current.Status == "complete" && status != "complete" {
			// A delayed caller may report an old timeout after another process
			// has already committed the instrument. Completion is monotonic.
			return nil
		}
		return ErrNotFound
	}
	return nil
}

type cardIssueRow interface {
	Scan(...any) error
}

func scanCardIssueAttempt(row cardIssueRow) (CardIssueAttempt, error) {
	var value CardIssueAttempt
	var createdAt, updatedAt string
	if err := row.Scan(&value.Fingerprint, &value.Status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CardIssueAttempt{}, ErrNotFound
		}
		return CardIssueAttempt{}, err
	}
	var err error
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return CardIssueAttempt{}, err
	}
	if value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return CardIssueAttempt{}, err
	}
	return value, nil
}

var (
	_ Repository       = (*SQLiteRepository)(nil)
	_ CardIssueJournal = (*SQLiteRepository)(nil)
)
