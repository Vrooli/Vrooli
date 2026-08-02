package database

import (
	"context"
	"encoding/json"
	"time"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

// receiptEvidenceRepository owns the receipt ledger, which is independent of
// the durable invocation projection and remains available to report rendering.
type receiptEvidenceRepository struct{ db *DB }

var (
	_ runreport.ReceiptJoinStore = (*receiptEvidenceRepository)(nil)
	_ runreport.LedgerStore      = (*receiptEvidenceRepository)(nil)
)

func (r *receiptEvidenceRepository) ReplaceCrossScenarioCalls(ctx context.Context, runID uuid.UUID, availability string, calls []runreport.CrossScenarioCall) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_cross_scenario_calls WHERE run_id=?`, runID.String()); err != nil {
		return err
	}
	for _, call := range calls {
		// The ledger is an efficacy projection, so only Vrooli Events-verified
		// receipts with a concrete target are durable rows. Unverified or empty
		// observations remain available through the diagnostic report but cannot
		// manufacture a capability population.
		if !call.Verified || call.TargetScenario == "" || call.Operation == "" || call.ReceiptEventID == "" {
			continue
		}
		projection, err := json.Marshal(call.Projection)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_cross_scenario_call_projections WHERE receipt_event_id=?`, call.ReceiptEventID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_cross_scenario_calls (run_id,receipt_event_id,occurred_at,target_scenario,operation,outcome,status_code,duration_ms,verified,projection,ledger_availability) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, runID.String(), call.ReceiptEventID, SQLiteTime(call.OccurredAt), call.TargetScenario, call.Operation, call.Outcome, call.StatusCode, call.DurationMS, call.Verified, string(projection), availability); err != nil {
			return err
		}
		for key, value := range call.Projection {
			encoded, err := json.Marshal(value)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_cross_scenario_call_projections (receipt_event_id,key,value_json) VALUES (?,?,?)`, call.ReceiptEventID, key, string(encoded)); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *receiptEvidenceRepository) CrossScenarioCalls(ctx context.Context, runID uuid.UUID) ([]runreport.CrossScenarioCall, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT receipt_event_id,occurred_at,target_scenario,operation,outcome,status_code,duration_ms,verified,projection FROM investigation_cross_scenario_calls WHERE run_id=? AND receipt_event_id<>'' ORDER BY receipt_event_id`, runID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []runreport.CrossScenarioCall{}
	for rows.Next() {
		var call runreport.CrossScenarioCall
		var projection string
		var occurredAt *time.Time
		if err := rows.Scan(&call.ReceiptEventID, &occurredAt, &call.TargetScenario, &call.Operation, &call.Outcome, &call.StatusCode, &call.DurationMS, &call.Verified, &projection); err != nil {
			return nil, err
		}
		if occurredAt != nil {
			call.OccurredAt = *occurredAt
		}
		if err := json.Unmarshal([]byte(projection), &call.Projection); err != nil {
			return nil, err
		}
		out = append(out, call)
	}
	return out, rows.Err()
}

func (r *receiptEvidenceRepository) ReplaceReceiptEvidence(ctx context.Context, runID uuid.UUID, availability string, eventIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_receipt_evidence WHERE run_id = ?`, runID.String()); err != nil {
		return err
	}
	for _, eventID := range eventIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_receipt_evidence (run_id,event_id,availability) VALUES (?,?,?)`, runID.String(), eventID, availability); err != nil {
			return err
		}
	}
	if len(eventIDs) == 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_receipt_evidence (run_id,event_id,availability) VALUES (?, '', ?)`, runID.String(), availability); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *receiptEvidenceRepository) ReceiptEvidence(ctx context.Context, runID uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT event_id FROM investigation_receipt_evidence WHERE run_id=? AND event_id<>'' ORDER BY event_id`, runID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
