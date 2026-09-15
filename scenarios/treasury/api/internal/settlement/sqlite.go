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

func (r *SQLiteRepository) Complete(ctx context.Context, id string, outcome Outcome, result RailResult, updatedAt, retainUntil string, artifactValues ...CompletionArtifacts) (Record, error) {
	if outcome != OutcomeSettled && outcome != OutcomeFailed && outcome != OutcomeUnknown {
		return Record{}, fmt.Errorf("complete settlement: invalid terminal outcome %q", outcome)
	}
	query := `UPDATE settlements SET outcome=?,external_id=?,receipt_reference=?,basis=?,detail=?,occurred_at=?,updated_at=?,retain_until=? WHERE id=? AND outcome='calling'`
	if result.FromQuery {
		query = `UPDATE settlements SET outcome=?,external_id=?,receipt_reference=?,basis=?,detail=?,occurred_at=?,updated_at=?,retain_until=? WHERE id=? AND outcome='unknown'`
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Record{}, err
	}
	defer func() { _ = tx.Rollback() }()
	execResult, err := tx.ExecContext(ctx, query, string(outcome), result.ExternalID, result.ReceiptReference, result.Basis, result.Detail, formatOptionalTime(result.OccurredAt), updatedAt, retainUntil, id)
	if err != nil {
		return Record{}, err
	}
	rows, err := execResult.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if rows == 0 {
		current, getErr := scanRecord(tx.QueryRowContext(ctx, selectSettlement+` WHERE id=?`, id))
		if getErr == nil && current.Outcome == outcome {
			return current, nil
		}
		return Record{}, fmt.Errorf("complete settlement: transition rejected from current state")
	}
	stored, err := scanRecord(tx.QueryRowContext(ctx, selectSettlement+` WHERE id=?`, id))
	if err != nil {
		return Record{}, err
	}
	if outcome == OutcomeSettled || outcome == OutcomeFailed {
		if len(artifactValues) != 1 {
			return Record{}, errors.New("complete settlement: exactly one immutable artifact snapshot is required")
		}
		artifacts := artifactValues[0]
		if artifacts.AgentSubject == "" || artifacts.RequestJSON == "" || artifacts.RailResponseJSON == "" || artifacts.ReceiptJSON == "" {
			return Record{}, errors.New("complete settlement: complete immutable artifact snapshot is required")
		}
		var approvalID string
		_ = tx.QueryRowContext(ctx, `SELECT id FROM approval_requests WHERE authorization_id=?`, stored.AuthorizationID).Scan(&approvalID)
		if _, err := tx.ExecContext(ctx, `INSERT INTO spend_attempt_evidence(id,authorization_id,mandate_id,approval_id,settlement_id,instrument_id,agent_subject,outcome,basis,request_json,rail_response_json,receipt_json,recorded_at,retain_until) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, stored.AuthorizationID+":attempt", stored.AuthorizationID, stored.MandateID, approvalID, stored.ID, stored.InstrumentID, artifacts.AgentSubject, string(outcome), result.Basis, artifacts.RequestJSON, artifacts.RailResponseJSON, artifacts.ReceiptJSON, updatedAt, retainUntil); err != nil {
			return Record{}, fmt.Errorf("append settlement evidence: %w", err)
		}
		if outcome == OutcomeSettled {
			basis := "authoritative"
			if result.Basis == "operator_attestation" {
				basis = "operator_asserted"
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_emissions(id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,0,'',?,'')`, stored.ID+":ledger", stored.ID, "treasury:"+stored.ID, "treasury", "", "", -stored.AmountMinor, stored.Currency, basis, formatOptionalTime(result.OccurredAt), updatedAt, "Treasury settlement "+stored.ID+" at "+stored.Counterparty, "queued", updatedAt); err != nil {
				return Record{}, fmt.Errorf("queue money-ledger emission: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return Record{}, err
	}
	return stored, nil
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
