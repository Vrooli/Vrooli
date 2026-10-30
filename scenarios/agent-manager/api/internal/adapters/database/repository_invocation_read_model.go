package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
)

type invocationReadModelRepository struct{ db *DB }

var (
	_ invocationreadmodel.Store                 = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.RunStore              = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.WorkloadStore         = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.ProjectionStore       = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.CohortDefinitionStore = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.FreshnessStore        = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.ProjectionQueue       = (*invocationReadModelRepository)(nil)
)

func (r *invocationReadModelRepository) MarkProjectionPending(ctx context.Context, runID string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO invocation_read_model_pending (run_id,generation,attempts,next_attempt_unix) VALUES (?,1,0,?)
		ON CONFLICT(run_id) DO UPDATE SET generation=generation+1,attempts=0,next_attempt_unix=excluded.next_attempt_unix`, runID, at.Unix())
	return err
}

// EnqueueStaleProjections marks ended runs that have no watermark or one from
// another fact classifier. Existing marks keep their backoff.
func (r *invocationReadModelRepository) EnqueueStaleProjections(ctx context.Context, factVersion string, at time.Time, limit int) (int, error) {
	result, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_pending (run_id,generation,attempts,next_attempt_unix)
		SELECT r.id,1,0,? FROM runs r
		LEFT JOIN invocation_read_model_watermarks w ON w.run_id = r.id
		LEFT JOIN invocation_read_model_pending p ON p.run_id = r.id
		WHERE r.ended_at IS NOT NULL AND p.run_id IS NULL AND (w.run_id IS NULL OR w.classifier_version <> ?)
		ORDER BY r.ended_at DESC LIMIT ?`, at.Unix(), factVersion, limit)
	if err != nil {
		return 0, err
	}
	added, err := result.RowsAffected()
	return int(added), err
}

func (r *invocationReadModelRepository) DueProjections(ctx context.Context, at time.Time, limit int) ([]invocationreadmodel.PendingProjection, error) {
	var rows []struct {
		RunID      string `db:"run_id"`
		Generation int64  `db:"generation"`
		Attempts   int    `db:"attempts"`
	}
	// Live runs, then the newest ended run, first among due marks: a backlog
	// left by an earlier classifier change must not hold the current week
	// behind it.
	if err := r.db.SelectContext(ctx, &rows, `SELECT p.run_id,p.generation,p.attempts FROM invocation_read_model_pending p
		LEFT JOIN runs r ON r.id = p.run_id
		WHERE p.next_attempt_unix <= ? ORDER BY r.ended_at IS NOT NULL, r.ended_at DESC, p.run_id LIMIT ?`, at.Unix(), limit); err != nil {
		return nil, err
	}
	out := make([]invocationreadmodel.PendingProjection, 0, len(rows))
	for _, row := range rows {
		out = append(out, invocationreadmodel.PendingProjection{RunID: row.RunID, Generation: row.Generation, Attempts: row.Attempts})
	}
	return out, nil
}

func (r *invocationReadModelRepository) CompleteProjection(ctx context.Context, runID string, generation int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM invocation_read_model_pending WHERE run_id=? AND generation=?`, runID, generation)
	return err
}

func (r *invocationReadModelRepository) DeferProjection(ctx context.Context, runID string, until time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE invocation_read_model_pending SET attempts=attempts+1,next_attempt_unix=? WHERE run_id=?`, until.Unix(), runID)
	return err
}

func (r *invocationReadModelRepository) LatestProjection(ctx context.Context) (time.Time, error) {
	var projected SQLiteTime
	if err := r.db.GetContext(ctx, &projected, `SELECT MAX(projected_at) FROM invocation_read_model_runs`); err != nil {
		return time.Time{}, err
	}
	return projected.Time(), nil
}

func (r *invocationReadModelRepository) DefineCohort(ctx context.Context, definition invocationreadmodel.CohortDefinition) error {
	if strings.TrimSpace(definition.Name) == "" || strings.TrimSpace(definition.FilterJSON) == "" || strings.TrimSpace(definition.ClassifierVersion) == "" {
		return errors.New("cohort name, filter, and classifier version are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO invocation_cohort_definitions (name,filter_json,classifier_version,created_at,change_binding) VALUES (?,?,?,?,?) ON CONFLICT(name) DO UPDATE SET filter_json=excluded.filter_json,classifier_version=excluded.classifier_version,change_binding=excluded.change_binding`, definition.Name, definition.FilterJSON, definition.ClassifierVersion, SQLiteTime(definition.CreatedAt), definition.ChangeBinding)
	return err
}

func (r *invocationReadModelRepository) ListCohorts(ctx context.Context) ([]invocationreadmodel.CohortDefinition, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT name,filter_json,classifier_version,created_at,change_binding FROM invocation_cohort_definitions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []invocationreadmodel.CohortDefinition{}
	for rows.Next() {
		var item invocationreadmodel.CohortDefinition
		var created SQLiteTime
		if err := rows.Scan(&item.Name, &item.FilterJSON, &item.ClassifierVersion, &created, &item.ChangeBinding); err != nil {
			return nil, err
		}
		item.CreatedAt = created.Time()
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *invocationReadModelRepository) GetCohortDefinition(ctx context.Context, name string) (*invocationreadmodel.CohortDefinition, error) {
	var item invocationreadmodel.CohortDefinition
	var created SQLiteTime
	if err := r.db.QueryRowContext(ctx, `SELECT name,filter_json,classifier_version,created_at,change_binding FROM invocation_cohort_definitions WHERE name=?`, name).Scan(&item.Name, &item.FilterJSON, &item.ClassifierVersion, &created, &item.ChangeBinding); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item.CreatedAt = created.Time()
	return &item, nil
}

func (r *invocationReadModelRepository) DeleteCohort(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM invocation_cohort_definitions WHERE name=?`, name)
	return err
}

func nullableSQLiteTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return SQLiteTime(*value)
}

func (r *invocationReadModelRepository) ReplaceRun(ctx context.Context, fact invocationreadmodel.RunFact) error {
	return replaceRunTx(ctx, r.db, fact)
}

func (r *invocationReadModelRepository) BackfillWorkload(ctx context.Context, runID, kind, key, instance string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE invocation_read_model_runs SET workload_kind=?,workload_key=?,workload_instance=? WHERE run_id=?`, kind, key, instance, runID)
	return err
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// runProjectionTables are the read-model tables keyed by run id. A deleted run
// takes its rows with it, or run statistics keep counting it.
var runProjectionTables = []string{
	"invocation_read_model_facts", "invocation_read_model_self_report_spans", "invocation_read_model_runs",
	"invocation_read_model_time_accounting", "invocation_read_model_watermarks", "invocation_read_model_pending",
}

// deleteRunProjectionsTx removes everything the read model derived from runIDs
// inside the caller's transaction, so the run and its projection go together.
func deleteRunProjectionsTx(ctx context.Context, executor sqlExecutor, runIDs ...string) error {
	for _, runID := range runIDs {
		for _, table := range runProjectionTables {
			if _, err := executor.ExecContext(ctx, `DELETE FROM `+table+` WHERE run_id = ?`, runID); err != nil {
				return err
			}
		}
	}
	return nil
}

func replaceRunTx(ctx context.Context, executor sqlExecutor, fact invocationreadmodel.RunFact) error {
	values := strings.Repeat("?,", 40) + "?"
	query := `INSERT INTO invocation_read_model_runs (
		run_id,goal_id,goal_status,occurred_at,created_at,started_at,ended_at,duration_ms,status,profile_id,runner_type,model,tag,
		workload_kind,workload_key,workload_instance,total_cost_usd,cost_source,charge_reason,input_cost_usd,output_cost_usd,cache_read_cost_usd,
		cache_creation_cost_usd,total_tokens,input_tokens,output_tokens,cache_read_tokens,cache_creation_tokens,turns,
		tool_calls,total_charge_micro_usd,metered_charge_micro_usd,unpriced_token_count,preamble_injected_tokens,preamble_fixed_tokens,preamble_token_basis,unattributed_tokens,unattributed_reason,cost_time_basis,time_basis,projected_at
	) VALUES (` + values + `)
	ON CONFLICT(run_id) DO UPDATE SET
		goal_id=excluded.goal_id,goal_status=excluded.goal_status,occurred_at=excluded.occurred_at,created_at=excluded.created_at,started_at=excluded.started_at,ended_at=excluded.ended_at,
		duration_ms=excluded.duration_ms,status=excluded.status,profile_id=excluded.profile_id,runner_type=excluded.runner_type,
		model=excluded.model,tag=excluded.tag,workload_kind=excluded.workload_kind,workload_key=excluded.workload_key,
		workload_instance=excluded.workload_instance,total_cost_usd=excluded.total_cost_usd,cost_source=excluded.cost_source,charge_reason=excluded.charge_reason,input_cost_usd=excluded.input_cost_usd,
		output_cost_usd=excluded.output_cost_usd,cache_read_cost_usd=excluded.cache_read_cost_usd,
		cache_creation_cost_usd=excluded.cache_creation_cost_usd,total_tokens=excluded.total_tokens,input_tokens=excluded.input_tokens,
		output_tokens=excluded.output_tokens,cache_read_tokens=excluded.cache_read_tokens,cache_creation_tokens=excluded.cache_creation_tokens,
		turns=excluded.turns,tool_calls=excluded.tool_calls,total_charge_micro_usd=excluded.total_charge_micro_usd,
		metered_charge_micro_usd=excluded.metered_charge_micro_usd,unpriced_token_count=excluded.unpriced_token_count,
		preamble_injected_tokens=excluded.preamble_injected_tokens,preamble_fixed_tokens=excluded.preamble_fixed_tokens,preamble_token_basis=excluded.preamble_token_basis,
		unattributed_tokens=excluded.unattributed_tokens,unattributed_reason=excluded.unattributed_reason,
		cost_time_basis=excluded.cost_time_basis,time_basis=excluded.time_basis,projected_at=excluded.projected_at`
	if _, err := executor.ExecContext(ctx, query, fact.RunID, fact.GoalID, fact.GoalStatus, SQLiteTime(fact.OccurredAt), SQLiteTime(fact.CreatedAt), nullableSQLiteTime(fact.StartedAt), nullableSQLiteTime(fact.EndedAt), fact.DurationMS, fact.Status, fact.ProfileID, fact.RunnerType, fact.Model, fact.Tag, fact.WorkloadKind, fact.WorkloadKey, fact.WorkloadInstance, fact.TotalCostUSD, fact.CostSource, fact.ChargeReason, fact.InputCostUSD, fact.OutputCostUSD, fact.CacheReadCostUSD, fact.CacheCreationCostUSD, fact.TotalTokens, fact.InputTokens, fact.OutputTokens, fact.CacheReadTokens, fact.CacheCreationTokens, fact.Turns, fact.ToolCalls, fact.TotalChargeMicroUSD, fact.MeteredChargeMicroUSD, fact.UnpricedTokenCount, fact.PreambleInjectedTokens, fact.PreambleFixedTokens, fact.PreambleTokenBasis, fact.UnattributedTokens, fact.UnattributedReason, fact.CostTimeBasis, fact.TimeBasis, SQLiteTime(fact.ProjectedAt)); err != nil {
		return fmt.Errorf("upsert run read-model fact: %w", err)
	}
	if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_time_accounting (run_id,model_generating_ms,tool_executing_ms,idle_waiting_ms,awaiting_human_ms,unattributable_ms,model_tokens,tool_tokens,idle_tokens,human_tokens,unattributable_tokens,unattributable_reason) VALUES (?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id) DO UPDATE SET model_generating_ms=excluded.model_generating_ms,tool_executing_ms=excluded.tool_executing_ms,idle_waiting_ms=excluded.idle_waiting_ms,awaiting_human_ms=excluded.awaiting_human_ms,unattributable_ms=excluded.unattributable_ms,model_tokens=excluded.model_tokens,tool_tokens=excluded.tool_tokens,idle_tokens=excluded.idle_tokens,human_tokens=excluded.human_tokens,unattributable_tokens=excluded.unattributable_tokens,unattributable_reason=excluded.unattributable_reason`, fact.RunID, fact.TimeAccounting.ModelGeneratingMS, fact.TimeAccounting.ToolExecutingMS, fact.TimeAccounting.IdleWaitingMS, fact.TimeAccounting.AwaitingHumanMS, fact.TimeAccounting.UnattributableMS, fact.TimeAccounting.ModelTokens, fact.TimeAccounting.ToolTokens, fact.TimeAccounting.IdleTokens, fact.TimeAccounting.HumanTokens, fact.TimeAccounting.UnattributableTokens, fact.TimeAccounting.UnattributableReason); err != nil {
		return fmt.Errorf("upsert run time accounting: %w", err)
	}
	return nil
}

func (r *invocationReadModelRepository) TimeAccountingForRun(ctx context.Context, runID string) (runsignal.TimeAccounting, bool, error) {
	var value runsignal.TimeAccounting
	err := r.db.GetContext(ctx, &value, `SELECT model_generating_ms,tool_executing_ms,idle_waiting_ms,awaiting_human_ms,unattributable_ms,model_tokens,tool_tokens,idle_tokens,human_tokens,unattributable_tokens,unattributable_reason FROM invocation_read_model_time_accounting WHERE run_id=?`, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return value, false, nil
	}
	return value, err == nil, err
}

func (r *invocationReadModelRepository) Replace(ctx context.Context, facts []invocationreadmodel.Fact, watermark invocationreadmodel.Watermark) error {
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
	if err = r.replaceFactsTx(ctx, tx, facts, watermark); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *invocationReadModelRepository) AdvanceWatermark(ctx context.Context, watermark invocationreadmodel.Watermark) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,self_report_classifier_version,projection_complete,projected_at) VALUES (?,?,?,?,?,1,?) ON CONFLICT(run_id) DO UPDATE SET classifier_version=excluded.classifier_version,self_report_classifier_version=excluded.self_report_classifier_version,projection_complete=1,projected_at=excluded.projected_at`,
		watermark.RunID, watermark.LastEventID, SQLiteTime(watermark.LastEventAt), watermark.ClassifierVersion, watermark.SelfReportClassifierVersion, SQLiteTime(watermark.ProjectedAt))
	return err
}

func (r *invocationReadModelRepository) ReplaceProjection(ctx context.Context, facts []invocationreadmodel.Fact, spans []invocationreadmodel.SelfReportSpan, watermark invocationreadmodel.Watermark, run invocationreadmodel.RunFact) error {
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
	if err = r.replaceFactsTx(ctx, tx, facts, watermark); err != nil {
		return err
	}
	if err = replaceRunTx(ctx, tx, run); err != nil {
		return err
	}
	if err = replaceSelfReportSpansTx(ctx, tx, spans, watermark.RunID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func replaceSelfReportSpansTx(ctx context.Context, executor sqlExecutor, spans []invocationreadmodel.SelfReportSpan, runID string) error {
	if _, err := executor.ExecContext(ctx, `DELETE FROM invocation_read_model_self_report_spans WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("clear read-model self-report spans: %w", err)
	}
	for _, span := range spans {
		if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_self_report_spans (run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,role,span_capped,profile_id,runner_type,model,tag,run_status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, span.RunID, span.EventID, span.RuleID, span.StartOffset, span.EndOffset, SQLiteTime(span.OccurredAt), span.TimeBasis, span.ClassifierVersion, span.CauseScope, span.Text, span.Role, span.SpanCapped, span.ProfileID, span.RunnerType, span.Model, span.Tag, span.RunStatus); err != nil {
			return fmt.Errorf("insert read-model self-report span: %w", err)
		}
	}
	return nil
}

func (r *invocationReadModelRepository) replaceFactsTx(ctx context.Context, tx sqlExecutor, facts []invocationreadmodel.Fact, watermark invocationreadmodel.Watermark) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM invocation_read_model_facts WHERE run_id = ?`, watermark.RunID); err != nil {
		return fmt.Errorf("clear read-model facts: %w", err)
	}
	for _, fact := range facts {
		values := strings.Repeat("?,", 47+len(toolColumns)) + "?"
		query := `INSERT INTO invocation_read_model_facts (run_id,call_event_id,result_event_id,tool_call_id,occurred_at,time_basis,tool_name,capability,intent_class,executable,command_path,program_id,skill_id,ownership,ownership_reason,segment_index,segment_count,catalog_snapshot,outcome,pairing_basis,failure_signature,signature_truncated,retry_of_call_event_id,help_recovery,fingerprint,availability,classifier_version,profile_id,runner_type,model,tag,run_status,semantics_kind,semantics_verdict,wrapper,exit_code,duration_ms,arg_tokens,result_tokens,output_event_id,output_bytes,token_basis,residency_requests,residency_segment,incurred_input_tokens,incurred_output_tokens,incurred_cache_read_tokens,incurred_cache_creation_tokens,` + strings.Join(toolColumns, ",") + `) VALUES (` + values + `)`
		if _, err := tx.ExecContext(ctx, query, append([]any{fact.RunID, fact.CallEventID, fact.ResultEventID, fact.ToolCallID, SQLiteTime(fact.OccurredAt), fact.TimeBasis, fact.ToolName, fact.Capability, fact.IntentClass, fact.Executable, fact.CommandPath, fact.ProgramID, fact.SkillID, fact.Ownership, fact.OwnershipReason, fact.SegmentIndex, fact.SegmentCount, fact.CatalogSnapshot, fact.Outcome, fact.PairingBasis, fact.FailureSignature, fact.SignatureTruncated, fact.RetryOfCallEventID, fact.HelpRecovery, fact.Fingerprint, fact.Availability, fact.Version, fact.ProfileID, fact.RunnerType, fact.Model, fact.Tag, fact.RunStatus, fact.SemanticsKind, fact.SemanticsVerdict, fact.Wrapper, fact.ExitCode, fact.DurationMS, fact.ArgTokens, fact.ResultTokens, fact.OutputEventID, fact.OutputBytes, fact.TokenBasis, fact.ResidencyRequests, fact.ResidencySegment, fact.IncurredInputTokens, fact.IncurredOutputTokens, fact.IncurredCacheReadTokens, fact.IncurredCacheCreationTokens}, toolValues(fact.Tool)...)...); err != nil {
			return fmt.Errorf("insert read-model fact: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,self_report_classifier_version,projection_complete,projected_at) VALUES (?,?,?,?,?,1,?) ON CONFLICT(run_id) DO UPDATE SET last_event_id=excluded.last_event_id,last_event_at=excluded.last_event_at,classifier_version=excluded.classifier_version,self_report_classifier_version=excluded.self_report_classifier_version,projection_complete=1,projected_at=excluded.projected_at`, watermark.RunID, watermark.LastEventID, SQLiteTime(watermark.LastEventAt), watermark.ClassifierVersion, watermark.SelfReportClassifierVersion, SQLiteTime(watermark.ProjectedAt)); err != nil {
		return fmt.Errorf("upsert read-model watermark: %w", err)
	}
	return nil
}

func (r *invocationReadModelRepository) Cohort(ctx context.Context, filter invocationreadmodel.Filter, limit int) (invocationreadmodel.Cohort, error) {
	if limit <= 0 {
		limit = 100
	}
	where, args := invocationReadModelWhere(filter)
	var matched int
	if err := r.db.GetContext(ctx, &matched, "SELECT COUNT(DISTINCT run_id) FROM invocation_read_model_facts"+where, args...); err != nil {
		return invocationreadmodel.Cohort{}, err
	}
	args = append(args, limit+1)
	rows, err := r.db.QueryxContext(ctx, "SELECT DISTINCT run_id FROM invocation_read_model_facts"+where+" ORDER BY run_id LIMIT ?", args...)
	if err != nil {
		return invocationreadmodel.Cohort{}, err
	}
	defer rows.Close()
	out := invocationreadmodel.Cohort{RunIDs: []string{}, MatchedRuns: matched}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return out, err
		}
		out.RunIDs = append(out.RunIDs, id)
	}
	if len(out.RunIDs) > limit {
		out.RunIDs, out.Truncated = out.RunIDs[:limit], true
	}
	if matched > len(out.RunIDs) {
		out.Truncated = true
		out.DroppedRuns = matched - len(out.RunIDs)
	}
	return out, rows.Err()
}

// CohortRows hydrates the selected durable run ids in one bounded query. The
// optional fields remain nil when their source did not provide a value; the
// transport can therefore distinguish unavailable billing from a zero charge.
func (r *invocationReadModelRepository) CohortRows(ctx context.Context, runIDs []string, toolName string) ([]invocationreadmodel.CohortRun, error) {
	if len(runIDs) == 0 {
		return []invocationreadmodel.CohortRun{}, nil
	}
	placeholders := make([]string, len(runIDs))
	args := make([]any, 0, len(runIDs)+1)
	for i, runID := range runIDs {
		placeholders[i] = "?"
		args = append(args, runID)
	}
	toolSelect := "NULL"
	if toolName != "" {
		toolSelect = "(SELECT COUNT(*) FROM invocation_read_model_facts tf WHERE tf.run_id = irm.run_id AND tf.tool_name = ?)"
		args = append(args, toolName)
	}
	query := `SELECT irm.run_id, COALESCE(t.title, ''), irm.profile_id, COALESCE(p.name, ''), irm.status,
		irm.created_at, irm.model, irm.runner_type, irm.workload_key, irm.total_tokens,
		CASE WHEN COALESCE(json_extract(r.billing_snapshot, '$.mode'), '') <> '' THEN irm.total_charge_micro_usd ELSE NULL END,
		NULLIF(json_extract(r.billing_snapshot, '$.basis'), ''), ` + toolSelect + `
		FROM invocation_read_model_runs irm
		LEFT JOIN runs r ON r.id = irm.run_id
		LEFT JOIN tasks t ON t.id = r.task_id
		LEFT JOIN agent_profiles p ON p.id = irm.profile_id
		WHERE irm.run_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY irm.created_at DESC`
	// SQLite binds positional parameters in textual order. The tool-count
	// parameter occurs before the run-id predicates in the SELECT, so rebuild
	// the argument list accordingly when that optional expression is present.
	if toolName != "" {
		args = append([]any{toolName}, args[:len(args)-1]...)
	}
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]invocationreadmodel.CohortRun, 0, len(runIDs))
	for rows.Next() {
		var row invocationreadmodel.CohortRun
		var createdAt SQLiteTime
		var charge sql.NullInt64
		var basis sql.NullString
		var toolCalls sql.NullInt64
		if err := rows.Scan(&row.RunID, &row.TaskTitle, &row.ProfileID, &row.ProfileName, &row.Status, &createdAt, &row.Model, &row.RunnerType, &row.WorkloadKey, &row.TotalTokens, &charge, &basis, &toolCalls); err != nil {
			return nil, err
		}
		row.CreatedAt = createdAt.Time()
		if charge.Valid {
			value := charge.Int64
			row.TotalChargeMicroUSD = &value
		}
		if basis.Valid {
			value := basis.String
			row.ChargeBasis = &value
		}
		if toolCalls.Valid {
			value := toolCalls.Int64
			row.ToolCallCount = &value
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SelfReportSpansForRun returns only spans written by the current classifier
// version; rows from a retired classifier are never served.
func (r *invocationReadModelRepository) SelfReportSpansForRun(ctx context.Context, runID string) ([]invocationreadmodel.SelfReportSpan, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,role,span_capped,profile_id,runner_type,model,tag,run_status FROM invocation_read_model_self_report_spans WHERE run_id=? AND classifier_version=? ORDER BY occurred_at,event_id,start_offset`, runID, runsignal.SelfReportClassifierVersion)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.SelfReportSpan{}
	for rows.Next() {
		var s invocationreadmodel.SelfReportSpan
		var occurredAt SQLiteTime
		if err := rows.Scan(&s.RunID, &s.EventID, &s.RuleID, &s.StartOffset, &s.EndOffset, &occurredAt, &s.TimeBasis, &s.ClassifierVersion, &s.CauseScope, &s.Text, &s.Role, &s.SpanCapped, &s.ProfileID, &s.RunnerType, &s.Model, &s.Tag, &s.RunStatus); err != nil {
			return nil, err
		}
		s.OccurredAt = occurredAt.Time()
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) RunMetrics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error) {
	where, args := invocationReadModelRunWhere(filter)
	query := `SELECT COUNT(*),
		SUM(CASE WHEN status IN ('complete','failed','cancelled') THEN 1 ELSE 0 END),
		SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END),
		SUM(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN 1 ELSE 0 END),
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0)
		FROM invocation_read_model_runs` + where
	var total, terminal, successful, durationCount sql.NullInt64
	var avgDuration sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total, &terminal, &successful, &durationCount, &avgDuration); err != nil {
		return invocationreadmodel.RunMetrics{}, err
	}
	durations, err := r.RunDurationStatistics(ctx, filter)
	if err != nil {
		return invocationreadmodel.RunMetrics{}, err
	}
	m := invocationreadmodel.RunMetrics{TotalRuns: total.Int64, TerminalRuns: terminal.Int64, SuccessfulRuns: successful.Int64, CompletedDurationRuns: durationCount.Int64, AverageDurationMS: avgDuration.Float64, P50DurationMS: durations.P50DurationMS, P90DurationMS: durations.P90DurationMS}
	if m.TerminalRuns > 0 {
		m.SuccessRate = float64(m.SuccessfulRuns) / float64(m.TerminalRuns)
	}
	return m, nil
}

func (r *invocationReadModelRepository) HistoryCoverage(ctx context.Context) (time.Time, int64, error) {
	var floor SQLiteTime
	if err := r.db.GetContext(ctx, &floor, `SELECT MIN(created_at) FROM invocation_read_model_runs`); err != nil {
		return time.Time{}, 0, err
	}
	if floor.Time().IsZero() {
		return time.Time{}, 0, nil
	}
	var outside int64
	if err := r.db.GetContext(ctx, &outside, `SELECT COUNT(*) FROM runs WHERE created_at < ? AND NOT EXISTS (SELECT 1 FROM invocation_read_model_runs rm WHERE rm.run_id = runs.id)`, floor); err != nil {
		return time.Time{}, 0, err
	}
	return floor.Time(), outside, nil
}

func (r *invocationReadModelRepository) RunDurationStatistics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error) {
	where, args := invocationReadModelRunWhere(filter)
	query := `SELECT duration_ms
		FROM invocation_read_model_runs` + where + `
		 AND started_at IS NOT NULL AND ended_at IS NOT NULL
		ORDER BY duration_ms`
	var durations []int64
	if err := r.db.SelectContext(ctx, &durations, query, args...); err != nil {
		return invocationreadmodel.RunDurationStatistics{}, err
	}
	if len(durations) == 0 {
		return invocationreadmodel.RunDurationStatistics{}, nil
	}
	var total int64
	for _, duration := range durations {
		total += duration
	}
	return invocationreadmodel.RunDurationStatistics{
		AverageDurationMS: float64(total) / float64(len(durations)),
		P50DurationMS:     float64(nearestRank(durations, 0.50)),
		P90DurationMS:     float64(nearestRank(durations, 0.90)),
		P95DurationMS:     float64(nearestRank(durations, 0.95)),
		P99DurationMS:     float64(nearestRank(durations, 0.99)),
		MinDurationMS:     durations[0],
		MaxDurationMS:     durations[len(durations)-1],
		Count:             int64(len(durations)),
	}, nil
}

func (r *invocationReadModelRepository) RunStatusCounts(ctx context.Context, filter invocationreadmodel.Filter) ([]invocationreadmodel.RunStatusCount, error) {
	where, args := invocationReadModelRunWhere(filter)
	rows, err := r.db.QueryxContext(ctx, `SELECT status, COUNT(*) AS count FROM invocation_read_model_runs`+where+` GROUP BY status ORDER BY count DESC, status ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.RunStatusCount{}
	for rows.Next() {
		var row invocationreadmodel.RunStatusCount
		if err := rows.Scan(&row.Status, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func terminalRunBucketSQL(bucketDuration time.Duration) string {
	switch {
	case bucketDuration >= 24*time.Hour:
		return "strftime('%Y-%m-%dT00:00:00Z', occurred_at)"
	case bucketDuration >= 6*time.Hour:
		return "strftime('%Y-%m-%dT', occurred_at) || printf('%02d:00:00Z', (CAST(strftime('%H', occurred_at) AS INTEGER) / 6) * 6)"
	case bucketDuration >= time.Hour:
		return "strftime('%Y-%m-%dT%H:00:00Z', occurred_at)"
	default:
		return "strftime('%Y-%m-%dT%H:%M:00Z', occurred_at)"
	}
}

func (r *invocationReadModelRepository) RunTimeSeries(ctx context.Context, filter invocationreadmodel.Filter, bucketDuration time.Duration) ([]invocationreadmodel.RunTimeSeriesBucket, error) {
	if bucketDuration <= 0 {
		bucketDuration = time.Hour
	}
	where, args := invocationReadModelRunWhere(filter)
	bucket := terminalRunBucketSQL(bucketDuration)
	query := fmt.Sprintf(`SELECT %s AS bucket,
		COUNT(*) AS terminal_runs,
		SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) AS completed_runs,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_runs,
		SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) AS cancelled_runs,
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0) AS avg_duration_ms
		FROM invocation_read_model_runs%s GROUP BY %s ORDER BY bucket ASC`, bucket, where, bucket)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.RunTimeSeriesBucket{}
	for rows.Next() {
		var row invocationreadmodel.RunTimeSeriesBucket
		var value SQLiteTime
		if err := rows.Scan(&value, &row.TerminalRuns, &row.CompletedRuns, &row.FailedRuns, &row.CancelledRuns, &row.AvgDurationMS); err != nil {
			return nil, err
		}
		row.Bucket = value.Time()
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) ToolUsage(ctx context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.ToolUsageRow, error) {
	if limit <= 0 {
		limit = 20
	}
	where, args := invocationReadModelWhere(filter)
	args = append(args, limit)
	rows, err := r.db.QueryxContext(ctx, `SELECT tool_name, COUNT(*) AS call_count,
		SUM(CASE WHEN outcome = 'success' THEN 1 ELSE 0 END) AS success_count,
		SUM(CASE WHEN outcome = 'failure' THEN 1 ELSE 0 END) AS failed_count,
		COALESCE(SUM(arg_tokens + result_tokens), 0) AS total_tokens,
		COALESCE(SUM(arg_tokens + CASE WHEN token_basis = 'estimated' THEN result_tokens ELSE 0 END), 0) AS estimated_tokens
		FROM invocation_read_model_facts`+where+` GROUP BY tool_name ORDER BY call_count DESC, tool_name ASC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.ToolUsageRow{}
	for rows.Next() {
		var row invocationreadmodel.ToolUsageRow
		var estimated int64
		if err := rows.Scan(&row.ToolName, &row.CallCount, &row.SuccessCount, &row.FailedCount, &row.TotalTokens, &estimated); err != nil {
			return nil, err
		}
		if row.TotalTokens > 0 {
			row.EstimatedTokenShare = float64(estimated) / float64(row.TotalTokens)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) ToolCommandBreakdown(ctx context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.ToolCommandRow, error) {
	if limit <= 0 {
		limit = 20
	}
	where, args := invocationReadModelWhere(filter)
	args = append(args, limit)
	query := `SELECT COALESCE(executable, ''), COALESCE(command_path, ''), COUNT(*),
		SUM(CASE WHEN outcome = 'success' THEN 1 ELSE 0 END),
		SUM(CASE WHEN outcome = 'failure' THEN 1 ELSE 0 END),
		COUNT(DISTINCT run_id),
		COALESCE(SUM(arg_tokens + result_tokens), 0),
		COALESCE(SUM(arg_tokens + CASE WHEN token_basis = 'estimated' THEN result_tokens ELSE 0 END), 0),
		COALESCE(CAST(AVG(arg_tokens + result_tokens) AS INTEGER), 0),
		COALESCE(MAX(arg_tokens + result_tokens), 0)
		FROM invocation_read_model_facts` + where + `
		GROUP BY executable, command_path ORDER BY COUNT(*) DESC, executable ASC, command_path ASC LIMIT ?`
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]invocationreadmodel.ToolCommandRow, 0, limit)
	for rows.Next() {
		var row invocationreadmodel.ToolCommandRow
		var executable, commandPath string
		var estimated, averageFootprint, maxFootprint int64
		if err := rows.Scan(&executable, &commandPath, &row.CallCount, &row.SuccessCount, &row.FailedCount, &row.RunCount, &row.TotalTokens, &estimated, &averageFootprint, &maxFootprint); err != nil {
			return nil, err
		}
		row.P50FootprintTokens, row.P95FootprintTokens, row.MaxFootprintTokens = averageFootprint, averageFootprint, maxFootprint
		if row.TotalTokens > 0 {
			row.EstimatedTokenShare = float64(estimated) / float64(row.TotalTokens)
		}
		row.Executable, row.CommandPath = truncateCommandField(executable), truncateCommandField(commandPath)
		row.Truncated = row.Executable != executable || row.CommandPath != commandPath
		out = append(out, row)
	}
	return out, rows.Err()
}

func truncateCommandField(value string) string {
	const maxBytes = 256
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}

// TokenAttribution aggregates durable token factors without storing a
// pre-multiplied residency product. SQLite uses the same mean-as-percentile
// convention as RunDurationStatistics until a native percentile aggregate is
// available.
func (r *invocationReadModelRepository) TokenAttribution(ctx context.Context, filter invocationreadmodel.Filter, groupBy, view string, limit int) ([]invocationreadmodel.TokenAttributionRow, error) {
	groups := map[string]string{
		"capability":                "COALESCE(NULLIF(f.capability, ''), 'unknown')",
		"executable":                "COALESCE(NULLIF(f.executable, ''), 'unknown')",
		"command_path":              "COALESCE(NULLIF(f.command_path, ''), 'unknown')",
		"target_scenario_operation": "COALESCE(NULLIF(c.target_scenario || '::' || c.operation, '::'), 'unknown')",
	}
	if _, ok := groups[groupBy]; !ok {
		return nil, fmt.Errorf("unsupported token attribution group_by %q; accepted values: capability, executable, command_path, target_scenario_operation", groupBy)
	}
	if view != string(tokenaccounting.Footprint) && view != string(tokenaccounting.Residency) && view != string(tokenaccounting.Incurred) {
		return nil, fmt.Errorf("unsupported token attribution view %q; accepted values: footprint, residency, incurred", view)
	}
	if limit <= 0 {
		limit = 20
	}
	where, args := invocationReadModelWhere(filter)
	base := "(SELECT * FROM invocation_read_model_facts" + where + ") f LEFT JOIN investigation_cross_scenario_calls c ON c.run_id = f.run_id AND c.receipt_event_id = f.call_event_id"
	outerClauses := []string{}
	if filter.TargetScenario != "" {
		outerClauses = append(outerClauses, "c.target_scenario = ?")
		args = append(args, filter.TargetScenario)
	}
	if filter.Operation != "" {
		outerClauses = append(outerClauses, "c.operation = ?")
		args = append(args, filter.Operation)
	}
	query := `SELECT ` + groups[groupBy] + ` AS value, COUNT(*) AS call_count,
		COALESCE(SUM(f.arg_tokens + f.result_tokens), 0) AS footprint_tokens,
		COALESCE(SUM(f.result_tokens * f.residency_requests), 0) AS residency_tokens,
		COALESCE(SUM(f.incurred_input_tokens + f.incurred_output_tokens + f.incurred_cache_read_tokens + f.incurred_cache_creation_tokens), 0) AS incurred_tokens,
		COALESCE(SUM(f.arg_tokens + CASE WHEN f.token_basis = 'estimated' THEN f.result_tokens ELSE 0 END), 0) AS estimated_footprint_tokens,
		COALESCE(SUM(CASE WHEN f.token_basis = 'estimated' THEN f.result_tokens * f.residency_requests ELSE 0 END), 0) AS estimated_residency_tokens,
		COALESCE(CAST(AVG(f.arg_tokens + f.result_tokens) AS INTEGER), 0) AS p50_footprint_tokens,
		COALESCE(MAX(f.arg_tokens + f.result_tokens), 0) AS max_footprint_tokens
		FROM ` + base
	if len(outerClauses) > 0 {
		query += " WHERE " + strings.Join(outerClauses, " AND ")
	}
	query += " GROUP BY value ORDER BY " + map[string]string{"footprint": "footprint_tokens", "residency": "residency_tokens", "incurred": "incurred_tokens"}[view] + " DESC, value ASC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]invocationreadmodel.TokenAttributionRow, 0, limit)
	for rows.Next() {
		var row invocationreadmodel.TokenAttributionRow
		var footprint, residency, incurred, estimatedFootprint, estimatedResidency, p50, max int64
		if err := rows.Scan(&row.Value, &row.CallCount, &footprint, &residency, &incurred, &estimatedFootprint, &estimatedResidency, &p50, &max); err != nil {
			return nil, err
		}
		row.GroupBy = groupBy
		switch view {
		case string(tokenaccounting.Footprint):
			row.TotalTokens, row.EstimatedTokens = footprint, estimatedFootprint
		case string(tokenaccounting.Residency):
			row.TotalTokens, row.EstimatedTokens = residency, estimatedResidency
		case string(tokenaccounting.Incurred):
			row.TotalTokens = incurred
		}
		if row.TotalTokens > 0 {
			row.EstimatedTokenShare = float64(row.EstimatedTokens) / float64(row.TotalTokens)
		}
		row.P50FootprintTokens, row.P95FootprintTokens, row.MaxFootprintTokens = p50, p50, max
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// The part of a run's unattributed tokens that falls in the window is its
	// lifetime figure scaled by the share of the run's tokens requested in it.
	cte, residualArgs := windowedUsageCTE(filter)
	residualWhere, whereArgs := runWhereClauses(filter, false, true)
	residualArgs = append(residualArgs, whereArgs...)
	residualClauses := []string{}
	if filter.TargetScenario != "" {
		residualClauses = append(residualClauses, "EXISTS (SELECT 1 FROM investigation_cross_scenario_calls c WHERE c.run_id = invocation_read_model_runs.run_id AND c.target_scenario = ?)")
		residualArgs = append(residualArgs, filter.TargetScenario)
	}
	if filter.Operation != "" {
		residualClauses = append(residualClauses, "EXISTS (SELECT 1 FROM investigation_cross_scenario_calls c WHERE c.run_id = invocation_read_model_runs.run_id AND c.operation = ?)")
		residualArgs = append(residualArgs, filter.Operation)
	}
	if len(residualClauses) > 0 {
		if residualWhere == "" {
			residualWhere = " WHERE " + strings.Join(residualClauses, " AND ")
		} else {
			residualWhere += " AND " + strings.Join(residualClauses, " AND ")
		}
	}
	var residualRuns, residualTokens int64
	if err := r.db.QueryRowContext(ctx, "WITH "+cte+` SELECT COUNT(DISTINCT uw_run_id), CAST(COALESCE(SUM(invocation_read_model_runs.unattributed_tokens * `+windowShare+`), 0) AS INTEGER)
		FROM uw JOIN invocation_read_model_runs ON invocation_read_model_runs.run_id = uw.uw_run_id`+residualWhere, residualArgs...).Scan(&residualRuns, &residualTokens); err != nil {
		return nil, err
	}
	if residualRuns > 0 {
		out = append(out, invocationreadmodel.TokenAttributionRow{GroupBy: groupBy, Value: "unattributed", CallCount: residualRuns, TotalTokens: residualTokens})
	}
	return out, nil
}

// runScopeMembership restricts a run-keyed read-model table to the filter's
// run scope by delegating to runScopeClause, so every builder shares one
// definition of scope. It adds nothing when no scope is set.
func runScopeMembership(filter invocationreadmodel.Filter, column string) string {
	scope := runScopeClause(filter.Scope, true)
	if scope == "" {
		return ""
	}
	return column + " IN (SELECT invocation_read_model_runs.run_id FROM invocation_read_model_runs WHERE " + scope + ")"
}

func invocationReadModelWhere(filter invocationreadmodel.Filter) (string, []any) {
	clauses, args := []string{}, []any{}
	add := func(column, value string) {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	if filter.From != nil {
		clauses = append(clauses, "occurred_at >= ?")
		args = append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses = append(clauses, "occurred_at < ?")
		args = append(args, SQLiteTime(*filter.To))
	}
	add("ownership", filter.Ownership)
	add("outcome", filter.Outcome)
	add("executable", filter.Executable)
	add("fingerprint", filter.Fingerprint)
	add("run_id", filter.RunID)
	add("profile_id", filter.ProfileID)
	add("runner_type", filter.RunnerType)
	add("model", filter.Model)
	add("workload_kind", filter.WorkloadKind)
	add("workload_key", filter.WorkloadKey)
	add("run_status", filter.RunStatus)
	add("tool_name", filter.ToolName)
	add("classifier_version", filter.ClassifierVersion)
	if filter.TagPrefix != "" {
		clauses = append(clauses, "tag LIKE ?")
		args = append(args, filter.TagPrefix+"%")
	}
	if scope := runScopeMembership(filter, "run_id"); scope != "" {
		clauses = append(clauses, scope)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// invocationReadModelRunWhere applies the shared corpus filter to terminal
// run summaries in the filter's window: runs that ended in it. Run dimensions
// bind directly to the run summary; invocation-only predicates select runs
// that have at least one matching durable invocation fact, preserving the
// cohort meaning without raw events.
func invocationReadModelRunWhere(filter invocationreadmodel.Filter) (string, []any) {
	return runWhereClauses(filter, true, false)
}

// runWhereClauses is the run-summary predicate. windowed adds the ended-in-
// window bound (run statistics); token queries bound by request time instead
// and pass tokens so an unspecified scope counts every run.
func runWhereClauses(filter invocationreadmodel.Filter, windowed, tokens bool) (string, []any) {
	const table = "invocation_read_model_runs."
	clauses, args := []string{}, []any{}
	if windowed && filter.From != nil {
		clauses, args = append(clauses, table+"occurred_at >= ?"), append(args, SQLiteTime(*filter.From))
	}
	if windowed && filter.To != nil {
		clauses, args = append(clauses, table+"occurred_at < ?"), append(args, SQLiteTime(*filter.To))
	}
	add := func(column, value string) {
		if value != "" {
			clauses, args = append(clauses, table+column+" = ?"), append(args, value)
		}
	}
	add("run_id", filter.RunID)
	add("profile_id", filter.ProfileID)
	add("runner_type", filter.RunnerType)
	add("model", filter.Model)
	add("status", filter.RunStatus)
	if scope := runScopeClause(filter.Scope, tokens); scope != "" {
		clauses = append(clauses, scope)
	}
	if filter.TagPrefix != "" {
		clauses, args = append(clauses, table+"tag LIKE ?"), append(args, filter.TagPrefix+"%")
	}
	invocationClauses, invocationArgs := []string{"f.run_id = invocation_read_model_runs.run_id"}, []any{}
	for _, predicate := range []struct{ column, value string }{{"ownership", filter.Ownership}, {"outcome", filter.Outcome}, {"executable", filter.Executable}, {"fingerprint", filter.Fingerprint}, {"tool_name", filter.ToolName}} {
		if predicate.value != "" {
			invocationClauses, invocationArgs = append(invocationClauses, "f."+predicate.column+" = ?"), append(invocationArgs, predicate.value)
		}
	}
	if len(invocationClauses) > 1 {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_facts f WHERE "+strings.Join(invocationClauses, " AND ")+")")
		args = append(args, invocationArgs...)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (r *invocationReadModelRepository) Facts(ctx context.Context, runID string) ([]invocationreadmodel.Fact, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT run_id,classifier_version,call_event_id,COALESCE(result_event_id,''),COALESCE(tool_call_id,''),occurred_at,time_basis,tool_name,COALESCE(capability,'other'),COALESCE(intent_class,''),COALESCE(executable,''),COALESCE(command_path,''),ownership,COALESCE(ownership_reason,''),COALESCE(segment_index,0),COALESCE(segment_count,1),COALESCE(catalog_snapshot,''),outcome,COALESCE(pairing_basis,'unpaired'),COALESCE(failure_signature,''),COALESCE(signature_truncated,0),COALESCE(retry_of_call_event_id,''),help_recovery,fingerprint,availability,profile_id,runner_type,model,tag,run_status,COALESCE(semantics_kind,''),COALESCE(semantics_verdict,''),COALESCE(wrapper,''),exit_code,duration_ms,COALESCE(arg_tokens,0),COALESCE(result_tokens,0),COALESCE(output_event_id,''),COALESCE(output_bytes,0),COALESCE(token_basis,'unknown'),COALESCE(residency_requests,0),COALESCE(residency_segment,0),COALESCE(incurred_input_tokens,0),COALESCE(incurred_output_tokens,0),COALESCE(incurred_cache_read_tokens,0),COALESCE(incurred_cache_creation_tokens,0) FROM invocation_read_model_facts WHERE run_id=? ORDER BY occurred_at, call_event_id, segment_index`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.Fact{}
	for rows.Next() {
		var f invocationreadmodel.Fact
		var occurredAt SQLiteTime
		var exitCode, durationMS sql.NullInt64
		var tokenBasis string
		if err := rows.Scan(&f.RunID, &f.Version, &f.CallEventID, &f.ResultEventID, &f.ToolCallID, &occurredAt, &f.TimeBasis, &f.ToolName, &f.Capability, &f.IntentClass, &f.Executable, &f.CommandPath, &f.Ownership, &f.OwnershipReason, &f.SegmentIndex, &f.SegmentCount, &f.CatalogSnapshot, &f.Outcome, &f.PairingBasis, &f.FailureSignature, &f.SignatureTruncated, &f.RetryOfCallEventID, &f.HelpRecovery, &f.Fingerprint, &f.Availability, &f.ProfileID, &f.RunnerType, &f.Model, &f.Tag, &f.RunStatus, &f.SemanticsKind, &f.SemanticsVerdict, &f.Wrapper, &exitCode, &durationMS, &f.ArgTokens, &f.ResultTokens, &f.OutputEventID, &f.OutputBytes, &tokenBasis, &f.ResidencyRequests, &f.ResidencySegment, &f.IncurredInputTokens, &f.IncurredOutputTokens, &f.IncurredCacheReadTokens, &f.IncurredCacheCreationTokens); err != nil {
			return nil, err
		}
		f.TokenBasis = tokenaccounting.Basis(tokenBasis)
		if exitCode.Valid {
			value := int(exitCode.Int64)
			f.ExitCode = &value
		}
		if durationMS.Valid {
			value := durationMS.Int64
			f.DurationMS = &value
		}
		f.OccurredAt = occurredAt.Time()
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) Watermark(ctx context.Context, runID string) (*invocationreadmodel.Watermark, error) {
	var row struct {
		RunID                       string     `db:"run_id"`
		LastEventID                 string     `db:"last_event_id"`
		LastEventAt                 SQLiteTime `db:"last_event_at"`
		ClassifierVersion           string     `db:"classifier_version"`
		SelfReportClassifierVersion string     `db:"self_report_classifier_version"`
		ProjectedAt                 SQLiteTime `db:"projected_at"`
	}
	err := r.db.GetContext(ctx, &row, `SELECT run_id,last_event_id,last_event_at,classifier_version,self_report_classifier_version,projected_at FROM invocation_read_model_watermarks WHERE run_id=?`, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &invocationreadmodel.Watermark{RunID: row.RunID, LastEventID: row.LastEventID, LastEventAt: row.LastEventAt.Time(), ClassifierVersion: row.ClassifierVersion, SelfReportClassifierVersion: row.SelfReportClassifierVersion, ProjectedAt: row.ProjectedAt.Time()}, nil
}
