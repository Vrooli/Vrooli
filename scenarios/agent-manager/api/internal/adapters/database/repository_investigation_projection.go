package database

import (
	"context"
	"fmt"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

type invocationFactRepository struct{ db *DB }

func (r *invocationFactRepository) ReplaceInvocationFacts(ctx context.Context, runID uuid.UUID, facts []runreport.InvocationFact) error {
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
	if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_invocation_facts WHERE run_id = ?`, runID.String()); err != nil {
		return fmt.Errorf("clear invocation facts: %w", err)
	}
	for _, fact := range facts {
		_, err = tx.ExecContext(ctx, `INSERT INTO investigation_invocation_facts (run_id,call_event_id,result_event_id,tool_call_id,tool_name,executable,command_path,ownership,catalog_snapshot,outcome,retry_of_call_event_id,help_recovery,fingerprint,availability,classifier_version) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, runID.String(), fact.CallEventID, fact.ResultEventID, fact.ToolCallID, fact.ToolName, fact.Executable, fact.CommandPath, fact.Ownership, fact.CatalogSnapshot, fact.Outcome, fact.RetryOfCallEventID, fact.HelpRecovery, fact.Fingerprint, fact.Availability, fact.Version)
		if err != nil {
			return fmt.Errorf("insert invocation fact: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *invocationFactRepository) InvocationFacts(ctx context.Context, runID uuid.UUID) ([]runreport.InvocationFact, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT classifier_version,call_event_id,COALESCE(result_event_id,''),COALESCE(tool_call_id,''),tool_name,COALESCE(executable,''),COALESCE(command_path,''),ownership,COALESCE(catalog_snapshot,''),outcome,COALESCE(retry_of_call_event_id,''),help_recovery,fingerprint,availability FROM investigation_invocation_facts WHERE run_id=? ORDER BY call_event_id`, runID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []runreport.InvocationFact{}
	for rows.Next() {
		var fact runreport.InvocationFact
		if err := rows.Scan(&fact.Version, &fact.CallEventID, &fact.ResultEventID, &fact.ToolCallID, &fact.ToolName, &fact.Executable, &fact.CommandPath, &fact.Ownership, &fact.CatalogSnapshot, &fact.Outcome, &fact.RetryOfCallEventID, &fact.HelpRecovery, &fact.Fingerprint, &fact.Availability); err != nil {
			return nil, err
		}
		out = append(out, fact)
	}
	return out, rows.Err()
}

func (r *invocationFactRepository) ReplaceReceiptEvidence(ctx context.Context, runID uuid.UUID, availability string, eventIDs []string) error {
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
		return fmt.Errorf("clear receipt evidence: %w", err)
	}
	for _, eventID := range eventIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_receipt_evidence (run_id,event_id,availability) VALUES (?,?,?)`, runID.String(), eventID, availability); err != nil {
			return fmt.Errorf("insert receipt evidence: %w", err)
		}
	}
	// Preserve explicit unavailable/unobserved state even when there is no
	// verified receipt identifier; empty identifiers are never returned.
	if len(eventIDs) == 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_receipt_evidence (run_id,event_id,availability) VALUES (?, '', ?)`, runID.String(), availability); err != nil {
			return fmt.Errorf("insert receipt evidence state: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *invocationFactRepository) ReceiptEvidence(ctx context.Context, runID uuid.UUID) ([]string, error) {
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
