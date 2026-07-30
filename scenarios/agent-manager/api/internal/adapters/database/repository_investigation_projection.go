package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

func (r *invocationFactRepository) ReplaceCrossScenarioCalls(ctx context.Context, runID uuid.UUID, availability string, calls []runreport.CrossScenarioCall) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_cross_scenario_calls WHERE run_id=?`, runID.String()); err != nil {
		return err
	}
	for _, call := range calls {
		projection, _ := json.Marshal(call.Projection)
		if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_cross_scenario_call_projections WHERE receipt_event_id=?`, call.ReceiptEventID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_cross_scenario_calls (run_id,receipt_event_id,occurred_at,target_scenario,operation,outcome,status_code,duration_ms,verified,projection,ledger_availability) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, runID.String(), call.ReceiptEventID, SQLiteTime(call.OccurredAt), call.TargetScenario, call.Operation, call.Outcome, call.StatusCode, call.DurationMS, call.Verified, string(projection), availability); err != nil {
			return err
		}
		for key, value := range call.Projection {
			encoded, _ := json.Marshal(value)
			if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_cross_scenario_call_projections (receipt_event_id,key,value_json) VALUES (?,?,?)`, call.ReceiptEventID, key, string(encoded)); err != nil {
				return err
			}
		}
	}
	if len(calls) == 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO investigation_cross_scenario_calls (run_id,receipt_event_id,occurred_at,target_scenario,operation,outcome,status_code,duration_ms,verified,projection,ledger_availability) VALUES (?, '', NULL, '', '', '', 0, 0, 0, '{}', ?)`, runID.String(), availability)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *invocationFactRepository) CrossScenarioCalls(ctx context.Context, runID uuid.UUID) ([]runreport.CrossScenarioCall, error) {
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
		_ = json.Unmarshal([]byte(projection), &call.Projection)
		out = append(out, call)
	}
	return out, rows.Err()
}

func (r *invocationFactRepository) ReplaceEpisodes(ctx context.Context, runID uuid.UUID, episodes []runreport.FrictionEpisode) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_friction_episodes WHERE run_id=?`, runID.String()); err != nil {
		return fmt.Errorf("clear episodes: %w", err)
	}
	for _, episode := range episodes {
		flags, _ := json.Marshal(episode.HonestyFlags)
		evidence, _ := json.Marshal(episode.EvidenceEventIDs)
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_friction_episodes (run_id,episode_id,classifier_version,pattern,cause_scope,severity,honesty_flags,start_event_id,end_event_id,evidence_event_ids,turns,tokens,wall_clock_ms,suspected_owner_scenario,suspected_owner_command,owner_confidence,failed_joined_calls,fingerprint) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, runID.String(), episode.EpisodeID, episode.ClassifierVersion, episode.Pattern, episode.CauseScope, episode.Severity, string(flags), episode.StartEventID, episode.EndEventID, string(evidence), episode.Turns, episode.Tokens, episode.WallClockMS, episode.SuspectedOwnerScenario, episode.SuspectedOwnerCommand, episode.OwnerConfidence, episode.FailedJoinedCalls, episode.Fingerprint); err != nil {
			return fmt.Errorf("insert episode: %w", err)
		}
	}
	return tx.Commit()
}

func (r *invocationFactRepository) Episodes(ctx context.Context, runID uuid.UUID) ([]runreport.FrictionEpisode, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT episode_id,classifier_version,pattern,cause_scope,severity,honesty_flags,start_event_id,end_event_id,evidence_event_ids,turns,tokens,wall_clock_ms,suspected_owner_scenario,suspected_owner_command,owner_confidence,failed_joined_calls,fingerprint FROM investigation_friction_episodes WHERE run_id=? ORDER BY start_event_id`, runID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []runreport.FrictionEpisode{}
	for rows.Next() {
		var e runreport.FrictionEpisode
		var flags, evidence string
		if err := rows.Scan(&e.EpisodeID, &e.ClassifierVersion, &e.Pattern, &e.CauseScope, &e.Severity, &flags, &e.StartEventID, &e.EndEventID, &evidence, &e.Turns, &e.Tokens, &e.WallClockMS, &e.SuspectedOwnerScenario, &e.SuspectedOwnerCommand, &e.OwnerConfidence, &e.FailedJoinedCalls, &e.Fingerprint); err != nil {
			return nil, err
		}
		e.RunID = runID.String()
		_ = json.Unmarshal([]byte(flags), &e.HonestyFlags)
		_ = json.Unmarshal([]byte(evidence), &e.EvidenceEventIDs)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *invocationFactRepository) ReplaceSelfReportSpans(ctx context.Context, runID uuid.UUID, spans []runreport.SelfReportSpan) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM investigation_self_report_spans WHERE run_id=?`, runID.String()); err != nil {
		return err
	}
	for _, span := range spans {
		if _, err = tx.ExecContext(ctx, `INSERT INTO investigation_self_report_spans (run_id,event_id,rule_id,classifier_version,cause_scope,start_offset,end_offset,text) VALUES (?,?,?,?,?,?,?,?)`, runID.String(), span.EventID, span.RuleID, span.ClassifierVersion, span.CauseScope, span.StartOffset, span.EndOffset, span.Text); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *invocationFactRepository) SelfReportSpans(ctx context.Context, runID uuid.UUID) ([]runreport.SelfReportSpan, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT classifier_version,event_id,rule_id,cause_scope,start_offset,end_offset,text FROM investigation_self_report_spans WHERE run_id=? ORDER BY event_id,start_offset`, runID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	spans := []runreport.SelfReportSpan{}
	for rows.Next() {
		var span runreport.SelfReportSpan
		if err := rows.Scan(&span.ClassifierVersion, &span.EventID, &span.RuleID, &span.CauseScope, &span.StartOffset, &span.EndOffset, &span.Text); err != nil {
			return nil, err
		}
		spans = append(spans, span)
	}
	return spans, rows.Err()
}

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
