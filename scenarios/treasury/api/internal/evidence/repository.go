package evidence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrAttemptNotFound = errors.New("spend attempt evidence not found")

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *SQLiteRecorder) AppendAttempt(ctx context.Context, value Attempt) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO spend_attempt_evidence(id,authorization_id,mandate_id,approval_id,settlement_id,instrument_id,agent_subject,outcome,basis,request_json,rail_response_json,receipt_json,recorded_at,retain_until) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(authorization_id) DO NOTHING`, value.ID, value.AuthorizationID, value.MandateID, value.ApprovalID, value.SettlementID, value.InstrumentID, value.AgentSubject, value.Outcome, value.Basis, value.RequestJSON, value.RailResponseJSON, value.ReceiptJSON, value.RecordedAt.UTC().Format(time.RFC3339Nano), value.RetainUntil.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("append spend attempt evidence: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 1 {
		return nil
	}
	existing, err := r.Replay(ctx, value.AuthorizationID)
	if err != nil {
		return err
	}
	if existing != value {
		return fmt.Errorf("append spend attempt evidence: authorization %q already has different immutable evidence", value.AuthorizationID)
	}
	return nil
}

func (r *SQLiteRecorder) Replay(ctx context.Context, authorizationID string) (Attempt, error) {
	var value Attempt
	var recordedAt, retainUntil string
	err := r.db.QueryRowContext(ctx, `SELECT id,authorization_id,mandate_id,approval_id,settlement_id,instrument_id,agent_subject,outcome,basis,request_json,rail_response_json,receipt_json,recorded_at,retain_until FROM spend_attempt_evidence WHERE authorization_id=?`, authorizationID).Scan(&value.ID, &value.AuthorizationID, &value.MandateID, &value.ApprovalID, &value.SettlementID, &value.InstrumentID, &value.AgentSubject, &value.Outcome, &value.Basis, &value.RequestJSON, &value.RailResponseJSON, &value.ReceiptJSON, &recordedAt, &retainUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return Attempt{}, ErrAttemptNotFound
	}
	if err != nil {
		return Attempt{}, err
	}
	if value.RecordedAt, err = time.Parse(time.RFC3339Nano, recordedAt); err != nil {
		return Attempt{}, err
	}
	value.RetainUntil, err = time.Parse(time.RFC3339Nano, retainUntil)
	return value, err
}

type SQLiteRecorder struct{ db DB }

func NewSQLiteRecorder(db DB) *SQLiteRecorder { return &SQLiteRecorder{db: db} }

func (r *SQLiteRecorder) Append(ctx context.Context, value Record) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO evidence_records(id, authorization_id, mandate_id, agent_subject, verdict, violated_constraint, detail, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`, value.ID, value.AuthorizationID, value.MandateID, value.AgentSubject, value.Verdict, value.ViolatedConstraint, value.Detail, value.CreatedAt)
	if err != nil {
		return fmt.Errorf("append evidence: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 1 {
		return err
	}
	var existing Record
	if err := r.db.QueryRowContext(ctx, `SELECT id,authorization_id,mandate_id,agent_subject,verdict,violated_constraint,detail,created_at FROM evidence_records WHERE id=?`, value.ID).Scan(&existing.ID, &existing.AuthorizationID, &existing.MandateID, &existing.AgentSubject, &existing.Verdict, &existing.ViolatedConstraint, &existing.Detail, &existing.CreatedAt); err != nil {
		return err
	}
	if existing != value {
		return fmt.Errorf("append evidence: id %q already has different immutable evidence", value.ID)
	}
	return nil
}
