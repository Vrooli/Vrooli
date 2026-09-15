// Package redemption owns requests, reservations and exactly-once settlement.
package redemption

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Debit struct {
	ID             string
	TokenTypeID    string
	HolderID       string
	Amount         int64
	CauseReference string
	ActorIdentity  string
	CreatedAt      time.Time
}

type (
	InventoryReserveFunc func(context.Context, *sql.Tx, string, time.Time) (CatalogEntry, error)
	InventoryReleaseFunc func(context.Context, *sql.Tx, string, time.Time) error
	BalanceFunc          func(context.Context, *sql.Tx, string, string) (int64, error)
	AppendDebitFunc      func(context.Context, *sql.Tx, Debit) error
	FailureInjector      func(string) error
)

type Repository interface {
	FindByIdempotency(context.Context, string) (Redemption, bool, error)
	AvailableBalance(context.Context, string, string) (int64, error)
	Request(context.Context, Redemption, CatalogEntry) (Redemption, error)
	ListForHolder(context.Context, string) ([]Redemption, error)
	ListPending(context.Context) ([]Redemption, error)
	Approve(context.Context, DecisionInput, time.Time) (Redemption, error)
	Deny(context.Context, DecisionInput, time.Time) (Redemption, error)
}

func (r *sqliteRepository) ListForHolder(ctx context.Context, holderID string) ([]Redemption, error) {
	rows, err := r.db.QueryContext(ctx, redemptionSelect+` WHERE holder_id = ? ORDER BY requested_at DESC, id ASC`, holderID)
	if err != nil {
		return nil, fmt.Errorf("list holder redemptions: %w", err)
	}
	defer rows.Close()
	return scanRedemptions(rows, "holder redemptions")
}

func (r *sqliteRepository) FindByIdempotency(ctx context.Context, key string) (Redemption, bool, error) {
	return getByIdempotency(ctx, r.db, key)
}

type sqliteRepository struct {
	db               SQLExecutor
	reserveInventory InventoryReserveFunc
	releaseInventory InventoryReleaseFunc
	balance          BalanceFunc
	appendDebit      AppendDebitFunc
	injectFailure    FailureInjector
}

func NewSQLiteRepository(db SQLExecutor, reserve InventoryReserveFunc, release InventoryReleaseFunc, balance BalanceFunc, appendDebit AppendDebitFunc) Repository {
	return newSQLiteRepository(db, reserve, release, balance, appendDebit, nil)
}

func NewSQLiteRepositoryWithFailureInjector(db SQLExecutor, reserve InventoryReserveFunc, release InventoryReleaseFunc, balance BalanceFunc, appendDebit AppendDebitFunc, inject FailureInjector) Repository {
	return newSQLiteRepository(db, reserve, release, balance, appendDebit, inject)
}

func newSQLiteRepository(db SQLExecutor, reserve InventoryReserveFunc, release InventoryReleaseFunc, balance BalanceFunc, appendDebit AppendDebitFunc, inject FailureInjector) Repository {
	return &sqliteRepository{db: db, reserveInventory: reserve, releaseInventory: release, balance: balance, appendDebit: appendDebit, injectFailure: inject}
}

func (r *sqliteRepository) AvailableBalance(ctx context.Context, holderID, tokenTypeID string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, fmt.Errorf("begin available balance read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	amount, err := r.availableBalance(ctx, tx, holderID, tokenTypeID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit available balance read: %w", err)
	}
	return amount, nil
}

func (r *sqliteRepository) availableBalance(ctx context.Context, tx *sql.Tx, holderID, tokenTypeID string) (int64, error) {
	if r.balance == nil {
		return 0, errors.New("redemption repository requires a journal balance projector")
	}
	journalBalance, err := r.balance(ctx, tx, holderID, tokenTypeID)
	if err != nil {
		return 0, fmt.Errorf("read journal balance: %w", err)
	}
	var reserved int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM reservations
		WHERE holder_id = ? AND token_type_id = ? AND state = ?`,
		holderID, tokenTypeID, ReservationStateActive).Scan(&reserved); err != nil {
		return 0, fmt.Errorf("read active reservations: %w", err)
	}
	return journalBalance - reserved, nil
}

func (r *sqliteRepository) Request(ctx context.Context, redemption Redemption, expected CatalogEntry) (Redemption, error) {
	if r.reserveInventory == nil || r.appendDebit == nil {
		return Redemption{}, errors.New("redemption repository requires inventory and debit ports")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Redemption{}, fmt.Errorf("begin redemption request: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if existing, found, err := getByIdempotency(ctx, tx, redemption.IdempotencyKey); err != nil {
		return Redemption{}, err
	} else if found {
		return existing, nil
	}
	// The inventory update is the first write and therefore acquires SQLite's
	// transaction write lock before balance and reservation state are checked.
	entry, err := r.reserveInventory(ctx, tx, redemption.CatalogEntryID, redemption.RequestedAt)
	if err != nil {
		return Redemption{}, err
	}
	if entry != expected || entry.ID != redemption.CatalogEntryID || entry.TokenTypeID != redemption.TokenTypeID || entry.CostAmount != redemption.Amount || entry.ApprovalPosture != redemption.ApprovalPosture {
		return Redemption{}, ErrCatalogChanged
	}
	available, err := r.availableBalance(ctx, tx, redemption.HolderID, redemption.TokenTypeID)
	if err != nil {
		return Redemption{}, err
	}
	if available < redemption.Amount {
		return Redemption{}, fmt.Errorf("%w: requested %d, available %d", ErrInsufficientBalance, redemption.Amount, available)
	}

	redemption.State = StatePendingApproval
	reservationState := ReservationStateActive
	if redemption.ApprovalPosture == ApprovalPostureImmediate {
		redemption.State = StateSettled
		reservationState = ReservationStateSettled
		settledAt := redemption.RequestedAt
		redemption.SettledAt = &settledAt
	}
	if err := insertRedemption(ctx, tx, redemption); err != nil {
		return Redemption{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO reservations (id, redemption_id, holder_id, token_type_id, amount, state, created_at, released_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`, uuid.NewString(), redemption.ID, redemption.HolderID,
		redemption.TokenTypeID, redemption.Amount, reservationState, formatTime(redemption.RequestedAt))
	if err != nil {
		return Redemption{}, fmt.Errorf("insert reservation: %w", err)
	}
	if err := appendRedemptionEvent(ctx, tx, redemption.ID, "requested", redemption.HolderID, "", redemption.RequestedAt); err != nil {
		return Redemption{}, err
	}
	if redemption.State == StateSettled {
		if err := r.fail("before_journal_append"); err != nil {
			return Redemption{}, err
		}
		if err := r.appendDebit(ctx, tx, debitFor(redemption, redemption.HolderID, redemption.RequestedAt)); err != nil {
			return Redemption{}, fmt.Errorf("append redemption debit: %w", err)
		}
		if err := appendRedemptionEvent(ctx, tx, redemption.ID, "settled", redemption.HolderID, "", redemption.RequestedAt); err != nil {
			return Redemption{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Redemption{}, fmt.Errorf("commit redemption request: %w", err)
	}
	return redemption, nil
}

func (r *sqliteRepository) ListPending(ctx context.Context) ([]Redemption, error) {
	rows, err := r.db.QueryContext(ctx, redemptionSelect+` WHERE state = ? ORDER BY requested_at ASC, id ASC`, StatePendingApproval)
	if err != nil {
		return nil, fmt.Errorf("list pending redemptions: %w", err)
	}
	defer rows.Close()
	return scanRedemptions(rows, "pending redemptions")
}

func scanRedemptions(rows *sql.Rows, label string) ([]Redemption, error) {
	values := make([]Redemption, 0)
	for rows.Next() {
		value, err := scanRedemption(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", label, err)
	}
	return values, nil
}

func (r *sqliteRepository) Approve(ctx context.Context, input DecisionInput, at time.Time) (Redemption, error) {
	return r.decide(ctx, input, at, true)
}

func (r *sqliteRepository) Deny(ctx context.Context, input DecisionInput, at time.Time) (Redemption, error) {
	return r.decide(ctx, input, at, false)
}

func (r *sqliteRepository) decide(ctx context.Context, input DecisionInput, at time.Time, approve bool) (Redemption, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Redemption{}, fmt.Errorf("begin redemption decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if existing, found, err := mutationOutcome(ctx, tx, input.IdempotencyKey); err != nil {
		return Redemption{}, err
	} else if found {
		return existing, nil
	}
	value, err := getRedemption(ctx, tx, input.RedemptionID)
	if err != nil {
		return Redemption{}, err
	}
	if value.State != StatePendingApproval {
		return Redemption{}, fmt.Errorf("%w: redemption is %s", ErrRedemptionConflict, value.State)
	}
	value.DeciderSubject = input.DeciderSubject
	value.DecisionReason = input.Reason
	value.DecidedAt = timePointer(at)
	if approve {
		if err := r.fail("before_journal_append"); err != nil {
			return Redemption{}, err
		}
		if err := r.appendDebit(ctx, tx, debitFor(value, input.DeciderSubject, at)); err != nil {
			return Redemption{}, fmt.Errorf("append approved redemption debit: %w", err)
		}
		value.State = StateSettled
		value.SettledAt = timePointer(at)
		_, err = tx.ExecContext(ctx, `UPDATE reservations SET state = ? WHERE redemption_id = ? AND state = ?`, ReservationStateSettled, value.ID, ReservationStateActive)
	} else {
		if r.releaseInventory == nil {
			return Redemption{}, errors.New("redemption repository requires an inventory release port")
		}
		value.State = StateDenied
		_, err = tx.ExecContext(ctx, `UPDATE reservations SET state = ?, released_at = ? WHERE redemption_id = ? AND state = ?`, ReservationStateReleased, formatTime(at), value.ID, ReservationStateActive)
		if err == nil {
			err = r.releaseInventory(ctx, tx, value.CatalogEntryID, at)
		}
	}
	if err != nil {
		return Redemption{}, fmt.Errorf("update reservation: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE redemptions SET state = ?, decider_subject = ?, decision_reason = ?, decided_at = ?, settled_at = ?
		WHERE id = ?`, value.State, value.DeciderSubject, value.DecisionReason,
		formatTime(*value.DecidedAt), nullableTime(value.SettledAt), value.ID)
	if err != nil {
		return Redemption{}, fmt.Errorf("update redemption decision: %w", err)
	}
	eventKind := "denied"
	if approve {
		eventKind = "approved"
	}
	if err := appendRedemptionEvent(ctx, tx, value.ID, eventKind, input.DeciderSubject, input.Reason, at); err != nil {
		return Redemption{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO redemption_mutations (idempotency_key, redemption_id, operation, created_at) VALUES (?, ?, ?, ?)`, input.IdempotencyKey, value.ID, eventKind, formatTime(at)); err != nil {
		return Redemption{}, fmt.Errorf("record redemption decision idempotency: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Redemption{}, fmt.Errorf("commit redemption decision: %w", err)
	}
	return value, nil
}

func (r *sqliteRepository) fail(stage string) error {
	if r.injectFailure == nil {
		return nil
	}
	return r.injectFailure(stage)
}

func debitFor(value Redemption, actor string, at time.Time) Debit {
	return Debit{
		ID: uuid.NewString(), TokenTypeID: value.TokenTypeID, HolderID: value.HolderID,
		Amount: value.Amount, CauseReference: "redemption:" + value.ID, ActorIdentity: actor, CreatedAt: at,
	}
}

func insertRedemption(ctx context.Context, tx *sql.Tx, value Redemption) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO redemptions (
			id, holder_id, catalog_entry_id, token_type_id, grant_id, amount,
			idempotency_key, state, approval_posture, decider_subject,
			decision_reason, requested_at, decided_at, settled_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, NULL, ?)`,
		value.ID, value.HolderID, value.CatalogEntryID, value.TokenTypeID, value.GrantID,
		value.Amount, value.IdempotencyKey, value.State, value.ApprovalPosture,
		formatTime(value.RequestedAt), nullableTime(value.SettledAt))
	if err != nil {
		return fmt.Errorf("insert redemption: %w", err)
	}
	return nil
}

func appendRedemptionEvent(ctx context.Context, tx *sql.Tx, redemptionID, kind, actor, reason string, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO redemption_events (id, redemption_id, kind, actor_subject, reason, created_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), redemptionID, kind, actor, reason, formatTime(at))
	if err != nil {
		return fmt.Errorf("append redemption event: %w", err)
	}
	return nil
}

const redemptionSelect = `
	SELECT id, holder_id, catalog_entry_id, token_type_id, grant_id, amount,
	       idempotency_key, state, approval_posture, decider_subject,
	       decision_reason, requested_at, decided_at, settled_at
	FROM redemptions`

type (
	queryer interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	rowScanner interface{ Scan(...any) error }
)

func getRedemption(ctx context.Context, db queryer, id string) (Redemption, error) {
	value, err := scanRedemption(db.QueryRowContext(ctx, redemptionSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Redemption{}, ErrRedemptionNotFound
	}
	return value, err
}

func getByIdempotency(ctx context.Context, db queryer, key string) (Redemption, bool, error) {
	value, err := scanRedemption(db.QueryRowContext(ctx, redemptionSelect+` WHERE idempotency_key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return Redemption{}, false, nil
	}
	return value, err == nil, err
}

func mutationOutcome(ctx context.Context, db queryer, key string) (Redemption, bool, error) {
	var redemptionID string
	err := db.QueryRowContext(ctx, `SELECT redemption_id FROM redemption_mutations WHERE idempotency_key = ?`, key).Scan(&redemptionID)
	if errors.Is(err, sql.ErrNoRows) {
		return Redemption{}, false, nil
	}
	if err != nil {
		return Redemption{}, false, fmt.Errorf("read redemption mutation: %w", err)
	}
	value, err := getRedemption(ctx, db, redemptionID)
	return value, err == nil, err
}

func scanRedemption(row rowScanner) (Redemption, error) {
	var value Redemption
	var requestedAt string
	var decidedAt, settledAt sql.NullString
	err := row.Scan(&value.ID, &value.HolderID, &value.CatalogEntryID, &value.TokenTypeID,
		&value.GrantID, &value.Amount, &value.IdempotencyKey, &value.State,
		&value.ApprovalPosture, &value.DeciderSubject, &value.DecisionReason,
		&requestedAt, &decidedAt, &settledAt)
	if err != nil {
		return Redemption{}, err
	}
	value.RequestedAt, err = time.Parse(time.RFC3339Nano, requestedAt)
	if err != nil {
		return Redemption{}, fmt.Errorf("parse redemption request time: %w", err)
	}
	value.DecidedAt, err = parseNullableTime(decidedAt)
	if err != nil {
		return Redemption{}, fmt.Errorf("parse redemption decision time: %w", err)
	}
	value.SettledAt, err = parseNullableTime(settledAt)
	if err != nil {
		return Redemption{}, fmt.Errorf("parse redemption settlement time: %w", err)
	}
	return value, nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	return timePointer(parsed), err
}

func timePointer(value time.Time) *time.Time { return &value }
func formatTime(value time.Time) string      { return value.UTC().Format(time.RFC3339Nano) }
