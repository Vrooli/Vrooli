package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const paymentReconciliationScope = "stripe-events"

type paymentReconciliationDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// SQLPaymentReconciliationStateStore persists a single fenced scheduler lease
// and monotone provider-event cursor in LPBS's PostgreSQL commerce schema.
type SQLPaymentReconciliationStateStore struct {
	db paymentReconciliationDB
}

func NewSQLPaymentReconciliationStateStore(db paymentReconciliationDB) (*SQLPaymentReconciliationStateStore, error) {
	if db == nil {
		return nil, errors.New("payment reconciliation database is required")
	}
	return &SQLPaymentReconciliationStateStore{db: db}, nil
}

func (s *SQLPaymentReconciliationStateStore) Claim(ctx context.Context, owner string, now, expires time.Time) (PaymentReconciliationLease, error) {
	if owner == "" || !expires.After(now) {
		return PaymentReconciliationLease{}, errors.New("payment reconciliation lease owner and positive expiry are required")
	}
	ttlMicros := expires.Sub(now).Microseconds()
	if ttlMicros <= 0 {
		return PaymentReconciliationLease{}, errors.New("payment reconciliation lease expiry must be at least one microsecond")
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO payment_reconciliation_checkpoint(scope_id) VALUES($1) ON CONFLICT(scope_id) DO NOTHING`, paymentReconciliationScope); err != nil {
		return PaymentReconciliationLease{}, fmt.Errorf("initialize payment reconciliation checkpoint: %w", err)
	}
	return scanPaymentReconciliationLease(s.db.QueryRowContext(ctx, `
		UPDATE payment_reconciliation_checkpoint
		SET lease_owner=$2, lease_generation=lease_generation+1,
		    lease_expires_at=CURRENT_TIMESTAMP + ($3 * INTERVAL '1 microsecond'), updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND (lease_expires_at IS NULL OR lease_expires_at <= CURRENT_TIMESTAMP)
		RETURNING lease_generation, lease_expires_at, cursor_created_at, cursor_event_id
	`, paymentReconciliationScope, owner, ttlMicros), owner)
}

func (s *SQLPaymentReconciliationStateStore) Renew(ctx context.Context, lease PaymentReconciliationLease, now, expires time.Time) (PaymentReconciliationLease, error) {
	if lease.Owner == "" || lease.Generation <= 0 || !expires.After(now) {
		return PaymentReconciliationLease{}, ErrReconciliationLeaseLost
	}
	ttlMicros := expires.Sub(now).Microseconds()
	if ttlMicros <= 0 {
		return PaymentReconciliationLease{}, ErrReconciliationLeaseLost
	}
	updated, err := scanPaymentReconciliationLease(s.db.QueryRowContext(ctx, `
		UPDATE payment_reconciliation_checkpoint
		SET lease_expires_at=CURRENT_TIMESTAMP + ($4 * INTERVAL '1 microsecond'), updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3 AND lease_expires_at > CURRENT_TIMESTAMP
		RETURNING lease_generation, lease_expires_at, cursor_created_at, cursor_event_id
	`, paymentReconciliationScope, lease.Owner, lease.Generation, ttlMicros), lease.Owner)
	if errors.Is(err, ErrReconciliationLeaseHeld) {
		return PaymentReconciliationLease{}, ErrReconciliationLeaseLost
	}
	return updated, err
}

func (s *SQLPaymentReconciliationStateStore) Advance(ctx context.Context, lease PaymentReconciliationLease, cursor PaymentReconciliationCursor, _ time.Time) error {
	if !cursor.Valid() {
		return ErrInvalidReconciliationEvent
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE payment_reconciliation_checkpoint
		SET cursor_created_at=$4, cursor_event_id=$5, updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3 AND lease_expires_at > CURRENT_TIMESTAMP
		  AND (cursor_created_at IS NULL OR cursor_created_at < $4 OR (cursor_created_at=$4 AND cursor_event_id <= $5))
	`, paymentReconciliationScope, lease.Owner, lease.Generation, cursor.CreatedAt.UTC().Truncate(time.Microsecond), cursor.EventID)
	if err != nil {
		return fmt.Errorf("advance payment reconciliation checkpoint: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read payment reconciliation checkpoint result: %w", err)
	}
	if rows != 1 {
		return ErrReconciliationLeaseLost
	}
	return nil
}

func (s *SQLPaymentReconciliationStateStore) Release(ctx context.Context, lease PaymentReconciliationLease) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE payment_reconciliation_checkpoint
		SET lease_owner='', lease_expires_at=NULL, updated_at=CURRENT_TIMESTAMP
		WHERE scope_id=$1 AND lease_owner=$2 AND lease_generation=$3
	`, paymentReconciliationScope, lease.Owner, lease.Generation)
	if err != nil {
		return fmt.Errorf("release payment reconciliation lease: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read payment reconciliation release result: %w", err)
	}
	if rows != 1 {
		return ErrReconciliationLeaseLost
	}
	return nil
}

type paymentReconciliationScanner interface {
	Scan(...any) error
}

func scanPaymentReconciliationLease(row paymentReconciliationScanner, owner string) (PaymentReconciliationLease, error) {
	var lease PaymentReconciliationLease
	var cursorAt sql.NullTime
	var cursorID sql.NullString
	lease.Owner = owner
	if err := row.Scan(&lease.Generation, &lease.ExpiresAt, &cursorAt, &cursorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentReconciliationLease{}, ErrReconciliationLeaseHeld
		}
		return PaymentReconciliationLease{}, fmt.Errorf("read payment reconciliation lease: %w", err)
	}
	lease.ExpiresAt = lease.ExpiresAt.UTC()
	if cursorAt.Valid || cursorID.Valid {
		if !cursorAt.Valid || !cursorID.Valid || cursorID.String == "" {
			return PaymentReconciliationLease{}, errors.New("payment reconciliation checkpoint is incomplete")
		}
		lease.Cursor = PaymentReconciliationCursor{CreatedAt: cursorAt.Time.UTC(), EventID: cursorID.String}
		lease.HasCursor = true
	}
	return lease, nil
}
