package database

import (
	"context"
	"database/sql"
	"encoding/json"
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
)

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
	if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_run_signals (run_id,read_calls,files_read_more_than_once) VALUES (?,?,?) ON CONFLICT(run_id) DO UPDATE SET read_calls=excluded.read_calls,files_read_more_than_once=excluded.files_read_more_than_once`, fact.RunID, fact.ReadCalls, fact.FileRereads); err != nil {
		return fmt.Errorf("upsert run read-model signals: %w", err)
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

func (r *invocationReadModelRepository) ReplaceProjection(ctx context.Context, facts []invocationreadmodel.Fact, errors []invocationreadmodel.ErrorFact, episodes []invocationreadmodel.Episode, spans []invocationreadmodel.SelfReportSpan, watermark invocationreadmodel.Watermark, run invocationreadmodel.RunFact) error {
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
	if err = replaceErrorsTx(ctx, tx, errors, watermark.RunID); err != nil {
		return err
	}
	if err = replaceEpisodesTx(ctx, tx, episodes, watermark.RunID); err != nil {
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

func replaceEpisodesTx(ctx context.Context, executor sqlExecutor, episodes []invocationreadmodel.Episode, runID string) error {
	if _, err := executor.ExecContext(ctx, `DELETE FROM invocation_read_model_episodes WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("clear read-model episodes: %w", err)
	}
	for _, episode := range episodes {
		honesty, err := json.Marshal(episode.HonestyFlags)
		if err != nil {
			return fmt.Errorf("encode episode honesty flags: %w", err)
		}
		evidence, err := json.Marshal(episode.EvidenceEventIDs)
		if err != nil {
			return fmt.Errorf("encode episode evidence ids: %w", err)
		}
		if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_episodes (run_id,episode_id,occurred_at,time_basis,classifier_version,pattern,cause_scope,severity,honesty_flags_json,start_event_id,end_event_id,evidence_event_ids_json,turns,cycle_count,repeated_element,tokens,wall_clock_ms,suspected_owner_scenario,suspected_owner_command,owner_confidence,failed_joined_calls,fingerprint,profile_id,runner_type,model,tag,run_status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, episode.RunID, episode.EpisodeID, SQLiteTime(episode.OccurredAt), episode.TimeBasis, episode.ClassifierVersion, episode.Pattern, episode.CauseScope, episode.Severity, string(honesty), episode.StartEventID, episode.EndEventID, string(evidence), episode.Turns, episode.CycleCount, episode.RepeatedElement, episode.Tokens, episode.WallClockMS, episode.SuspectedOwnerScenario, episode.SuspectedOwnerCommand, episode.OwnerConfidence, episode.FailedJoinedCalls, episode.Fingerprint, episode.ProfileID, episode.RunnerType, episode.Model, episode.Tag, episode.RunStatus); err != nil {
			return fmt.Errorf("insert read-model episode: %w", err)
		}
	}
	return nil
}

func replaceSelfReportSpansTx(ctx context.Context, executor sqlExecutor, spans []invocationreadmodel.SelfReportSpan, runID string) error {
	if _, err := executor.ExecContext(ctx, `DELETE FROM invocation_read_model_self_report_spans WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("clear read-model self-report spans: %w", err)
	}
	for _, span := range spans {
		if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_self_report_spans (run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,role,operator_correction,span_capped,profile_id,runner_type,model,tag,run_status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, span.RunID, span.EventID, span.RuleID, span.StartOffset, span.EndOffset, SQLiteTime(span.OccurredAt), span.TimeBasis, span.ClassifierVersion, span.CauseScope, span.Text, span.Role, span.OperatorCorrection, span.SpanCapped, span.ProfileID, span.RunnerType, span.Model, span.Tag, span.RunStatus); err != nil {
			return fmt.Errorf("insert read-model self-report span: %w", err)
		}
	}
	return nil
}

func replaceErrorsTx(ctx context.Context, executor sqlExecutor, facts []invocationreadmodel.ErrorFact, runID string) error {
	if _, err := executor.ExecContext(ctx, `DELETE FROM invocation_read_model_errors WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("clear read-model errors: %w", err)
	}
	for _, fact := range facts {
		if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_errors (run_id,event_id,occurred_at,time_basis,error_code,profile_id,runner_type,model,tag) VALUES (?,?,?,?,?,?,?,?,?)`, fact.RunID, fact.EventID, SQLiteTime(fact.OccurredAt), fact.TimeBasis, fact.ErrorCode, fact.ProfileID, fact.RunnerType, fact.Model, fact.Tag); err != nil {
			return fmt.Errorf("insert read-model error: %w", err)
		}
	}
	return nil
}

func (r *invocationReadModelRepository) replaceFactsTx(ctx context.Context, tx sqlExecutor, facts []invocationreadmodel.Fact, watermark invocationreadmodel.Watermark) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM invocation_read_model_facts WHERE run_id = ?`, watermark.RunID); err != nil {
		return fmt.Errorf("clear read-model facts: %w", err)
	}
	for _, fact := range facts {
		values := strings.Repeat("?,", 45) + "?"
		query := `INSERT INTO invocation_read_model_facts (run_id,call_event_id,result_event_id,tool_call_id,occurred_at,time_basis,tool_name,capability,intent_class,executable,command_path,program_id,skill_id,ownership,ownership_reason,segment_index,segment_count,catalog_snapshot,outcome,pairing_basis,failure_signature,signature_truncated,retry_of_call_event_id,help_recovery,fingerprint,availability,classifier_version,profile_id,runner_type,model,tag,run_status,semantics_kind,semantics_verdict,wrapper,exit_code,duration_ms,arg_tokens,result_tokens,token_basis,residency_turns,residency_segment,incurred_input_tokens,incurred_output_tokens,incurred_cache_read_tokens,incurred_cache_creation_tokens) VALUES (` + values + `)`
		if _, err := tx.ExecContext(ctx, query, fact.RunID, fact.CallEventID, fact.ResultEventID, fact.ToolCallID, SQLiteTime(fact.OccurredAt), fact.TimeBasis, fact.ToolName, fact.Capability, fact.IntentClass, fact.Executable, fact.CommandPath, fact.ProgramID, fact.SkillID, fact.Ownership, fact.OwnershipReason, fact.SegmentIndex, fact.SegmentCount, fact.CatalogSnapshot, fact.Outcome, fact.PairingBasis, fact.FailureSignature, fact.SignatureTruncated, fact.RetryOfCallEventID, fact.HelpRecovery, fact.Fingerprint, fact.Availability, fact.Version, fact.ProfileID, fact.RunnerType, fact.Model, fact.Tag, fact.RunStatus, fact.SemanticsKind, fact.SemanticsVerdict, fact.Wrapper, fact.ExitCode, fact.DurationMS, fact.ArgTokens, fact.ResultTokens, fact.TokenBasis, fact.ResidencyTurns, fact.ResidencySegment, fact.IncurredInputTokens, fact.IncurredOutputTokens, fact.IncurredCacheReadTokens, fact.IncurredCacheCreationTokens); err != nil {
			return fmt.Errorf("insert read-model fact: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,episode_classifier_version,self_report_classifier_version,projection_complete,projected_at) VALUES (?,?,?,?,?,?,1,?) ON CONFLICT(run_id) DO UPDATE SET last_event_id=excluded.last_event_id,last_event_at=excluded.last_event_at,classifier_version=excluded.classifier_version,episode_classifier_version=excluded.episode_classifier_version,self_report_classifier_version=excluded.self_report_classifier_version,projection_complete=1,projected_at=excluded.projected_at`, watermark.RunID, watermark.LastEventID, SQLiteTime(watermark.LastEventAt), watermark.ClassifierVersion, watermark.EpisodeClassifierVersion, watermark.SelfReportClassifierVersion, SQLiteTime(watermark.ProjectedAt)); err != nil {
		return fmt.Errorf("upsert read-model watermark: %w", err)
	}
	return nil
}

func (r *invocationReadModelRepository) Aggregate(ctx context.Context, filter invocationreadmodel.Filter, dimension string, limit int) ([]invocationreadmodel.AggregateRow, error) {
	columns := map[string]string{"ownership": "ownership", "outcome": "outcome", "executable": "executable", "fingerprint": "fingerprint", "tool": "tool_name", "runner": "runner_type", "model": "model", "profile": "profile_id", "status": "run_status"}
	column, ok := columns[dimension]
	if !ok {
		return nil, fmt.Errorf("unsupported aggregate dimension %q", dimension)
	}
	where, args := invocationReadModelWhere(filter)
	if limit <= 0 {
		limit = 100
	}
	query := fmt.Sprintf("SELECT %s AS value, COUNT(*) AS count FROM invocation_read_model_facts%s GROUP BY %s ORDER BY count DESC, value ASC LIMIT ?", column, where, column)
	args = append(args, limit)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.AggregateRow{}
	for rows.Next() {
		var row invocationreadmodel.AggregateRow
		row.Dimension = dimension
		if err := rows.Scan(&row.Value, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *invocationReadModelRepository) Cohort(ctx context.Context, filter invocationreadmodel.Filter, limit int) (invocationreadmodel.Cohort, error) {
	if limit <= 0 {
		limit = 100
	}
	where, args := invocationReadModelWhere(filter)
	table := "invocation_read_model_facts"
	if filter.EpisodePattern != "" || filter.EpisodeCauseScope != "" || filter.EpisodeFingerprint != "" || filter.SelfReportRuleID != "" || filter.SelfReportCauseScope != "" {
		where, args = invocationReadModelEpisodeWhere(filter)
		table = "invocation_read_model_episodes"
	}
	var matched int
	if err := r.db.GetContext(ctx, &matched, "SELECT COUNT(DISTINCT run_id) FROM "+table+where, args...); err != nil {
		return invocationreadmodel.Cohort{}, err
	}
	args = append(args, limit+1)
	rows, err := r.db.QueryxContext(ctx, "SELECT DISTINCT run_id FROM "+table+where+" ORDER BY run_id LIMIT ?", args...)
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

func (r *invocationReadModelRepository) Episodes(ctx context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.Episode, error) {
	if limit <= 0 {
		limit = 1000
	}
	where, args := invocationReadModelEpisodeWhere(filter)
	args = append(args, limit)
	return r.queryEpisodes(ctx, `SELECT run_id,episode_id,occurred_at,time_basis,classifier_version,pattern,cause_scope,severity,honesty_flags_json,start_event_id,end_event_id,evidence_event_ids_json,turns,cycle_count,repeated_element,tokens,wall_clock_ms,suspected_owner_scenario,suspected_owner_command,owner_confidence,failed_joined_calls,fingerprint,profile_id,runner_type,model,tag,run_status FROM invocation_read_model_episodes`+where+` ORDER BY occurred_at, episode_id LIMIT ?`, args...)
}

func (r *invocationReadModelRepository) EpisodesForRun(ctx context.Context, runID string) ([]invocationreadmodel.Episode, error) {
	return r.queryEpisodes(ctx, `SELECT run_id,episode_id,occurred_at,time_basis,classifier_version,pattern,cause_scope,severity,honesty_flags_json,start_event_id,end_event_id,evidence_event_ids_json,turns,cycle_count,repeated_element,tokens,wall_clock_ms,suspected_owner_scenario,suspected_owner_command,owner_confidence,failed_joined_calls,fingerprint,profile_id,runner_type,model,tag,run_status FROM invocation_read_model_episodes WHERE run_id=? ORDER BY occurred_at, episode_id`, runID)
}

func (r *invocationReadModelRepository) queryEpisodes(ctx context.Context, query string, args ...any) ([]invocationreadmodel.Episode, error) {
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.Episode{}
	for rows.Next() {
		var e invocationreadmodel.Episode
		var occurredAt SQLiteTime
		var honesty, evidence string
		if err := rows.Scan(&e.RunID, &e.EpisodeID, &occurredAt, &e.TimeBasis, &e.ClassifierVersion, &e.Pattern, &e.CauseScope, &e.Severity, &honesty, &e.StartEventID, &e.EndEventID, &evidence, &e.Turns, &e.CycleCount, &e.RepeatedElement, &e.Tokens, &e.WallClockMS, &e.SuspectedOwnerScenario, &e.SuspectedOwnerCommand, &e.OwnerConfidence, &e.FailedJoinedCalls, &e.Fingerprint, &e.ProfileID, &e.RunnerType, &e.Model, &e.Tag, &e.RunStatus); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(honesty), &e.HonestyFlags); err != nil {
			return nil, fmt.Errorf("decode episode honesty flags: %w", err)
		}
		if err := json.Unmarshal([]byte(evidence), &e.EvidenceEventIDs); err != nil {
			return nil, fmt.Errorf("decode episode evidence ids: %w", err)
		}
		e.OccurredAt = occurredAt.Time()
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) SelfReportSpansForRun(ctx context.Context, runID string) ([]invocationreadmodel.SelfReportSpan, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,role,operator_correction,span_capped,profile_id,runner_type,model,tag,run_status FROM invocation_read_model_self_report_spans WHERE run_id=? ORDER BY occurred_at,event_id,start_offset`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.SelfReportSpan{}
	for rows.Next() {
		var s invocationreadmodel.SelfReportSpan
		var occurredAt SQLiteTime
		if err := rows.Scan(&s.RunID, &s.EventID, &s.RuleID, &s.StartOffset, &s.EndOffset, &occurredAt, &s.TimeBasis, &s.ClassifierVersion, &s.CauseScope, &s.Text, &s.Role, &s.OperatorCorrection, &s.SpanCapped, &s.ProfileID, &s.RunnerType, &s.Model, &s.Tag, &s.RunStatus); err != nil {
			return nil, err
		}
		s.OccurredAt = occurredAt.Time()
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) Metrics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.Metrics, error) {
	where, args := invocationReadModelWhere(filter)
	fingerprintWhere := where + " AND fingerprint <> ''"
	if where == "" {
		fingerprintWhere = " WHERE fingerprint <> ''"
	}
	query := `SELECT COUNT(*),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership = 'external' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership = 'not-a-command' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership = 'compound-unresolved' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership = 'unparseable' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') AND outcome IN ('success', 'failure') THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') AND outcome NOT IN ('success', 'failure') THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') AND outcome = 'failure' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') AND retry_of_call_event_id <> '' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') AND help_recovery = 1 THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership IN ('resolved', 'project', 'external') AND fingerprint <> '' AND fingerprint IN (SELECT fingerprint FROM invocation_read_model_facts` + where + ` GROUP BY fingerprint HAVING COUNT(*) > 1) THEN 1 ELSE 0 END),
		COALESCE((SELECT MAX(bucket_count) FROM (SELECT COUNT(*) AS bucket_count FROM invocation_read_model_facts` + fingerprintWhere + ` GROUP BY fingerprint)), 0)
		FROM invocation_read_model_facts` + where
	// The repeated-fingerprint subquery has the same predicate, so bind it
	// twice; it cannot accidentally measure a wider population than the rates.
	allArgs := append(append(append([]any{}, args...), args...), args...)
	var values [13]sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, allArgs...).Scan(&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7], &values[8], &values[9], &values[10], &values[11], &values[12]); err != nil {
		return invocationreadmodel.Metrics{}, err
	}
	m := invocationreadmodel.Metrics{TotalCalls: values[0].Int64, ResolvedCalls: values[1].Int64, ExternalCalls: values[2].Int64, NotACommandCalls: values[3].Int64, CompoundUnresolvedCalls: values[4].Int64, UnparseableCalls: values[5].Int64, PairedCalls: values[6].Int64, UnpairedCalls: values[7].Int64, FailedCalls: values[8].Int64, RetryCalls: values[9].Int64, HelpRecoveries: values[10].Int64, RepeatedCalls: values[11].Int64, LargestFingerprintBucket: values[12].Int64, ClassifiedBase: values[1].Int64}
	// `unknown` is retained only for pre-v5 rows; count it as explicitly
	// unclassified while replay removes it from current projections.
	var legacyUnknown int64
	legacyWhere := " WHERE ownership = 'unknown'"
	if where != "" {
		legacyWhere = where + " AND ownership = 'unknown'"
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invocation_read_model_facts`+legacyWhere, args...).Scan(&legacyUnknown); err == nil {
		m.UnclassifiedCalls = m.CompoundUnresolvedCalls + m.UnparseableCalls + legacyUnknown
	} else {
		m.UnclassifiedCalls = m.CompoundUnresolvedCalls + m.UnparseableCalls
	}
	// not-a-command is an intentional terminal state for tools whose payload
	// contains no command (Read, Edit, planning, and similar tools). It is
	// published separately and must not be reported as extractor failure.
	m.UnknownCalls = m.UnclassifiedCalls
	m.UnclassifiedCount = m.UnclassifiedCalls
	if m.TotalCalls > 0 {
		m.UnclassifiedShare = float64(m.UnclassifiedCalls) / float64(m.TotalCalls)
	}
	if m.ClassifiedBase > 0 {
		m.ExternalToolShare = float64(m.ExternalCalls) / float64(m.ClassifiedBase)
	}
	if m.ClassifiedBase > 0 {
		m.FailureRate = float64(m.FailedCalls) / float64(m.ClassifiedBase)
		m.RetryRate = float64(m.RetryCalls) / float64(m.ClassifiedBase)
		m.HelpRecoveryRate = float64(m.HelpRecoveries) / float64(m.ClassifiedBase)
		m.RepeatedWorkRate = float64(m.RepeatedCalls) / float64(m.ClassifiedBase)
	}
	return m, nil
}

func (r *invocationReadModelRepository) RunMetrics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunMetrics, error) {
	where, args := invocationReadModelRunWhere(filter)
	query := `SELECT COUNT(*),
		SUM(CASE WHEN status IN ('complete','failed','cancelled') THEN 1 ELSE 0 END),
		SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END),
		SUM(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN 1 ELSE 0 END),
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0),
		COALESCE(SUM(total_cost_usd), 0),
		COALESCE(AVG(total_cost_usd), 0),
		COALESCE(SUM(total_tokens), 0),
		COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_creation_tokens), 0),
		COALESCE(SUM(input_cost_usd), 0), COALESCE(SUM(output_cost_usd), 0), COALESCE(SUM(cache_read_cost_usd), 0), COALESCE(SUM(cache_creation_cost_usd), 0),
		COALESCE(SUM(total_charge_micro_usd), 0), COALESCE(SUM(metered_charge_micro_usd), 0), COALESCE(SUM(unpriced_token_count), 0),
		COALESCE(SUM(signals.read_calls), 0),
		COALESCE(SUM(signals.files_read_more_than_once), 0)
		FROM invocation_read_model_runs LEFT JOIN invocation_read_model_run_signals signals ON signals.run_id = invocation_read_model_runs.run_id` + where
	var total, terminal, successful, durationCount, tokens, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens, totalCharge, meteredCharge, unpricedTokens, readCalls, rereads sql.NullInt64
	var avgDuration, totalCost, avgCost, inputCost, outputCost, cacheReadCost, cacheCreationCost sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total, &terminal, &successful, &durationCount, &avgDuration, &totalCost, &avgCost, &tokens, &inputTokens, &outputTokens, &cacheReadTokens, &cacheCreationTokens, &inputCost, &outputCost, &cacheReadCost, &cacheCreationCost, &totalCharge, &meteredCharge, &unpricedTokens, &readCalls, &rereads); err != nil {
		return invocationreadmodel.RunMetrics{}, err
	}
	m := invocationreadmodel.RunMetrics{TotalRuns: total.Int64, TerminalRuns: terminal.Int64, SuccessfulRuns: successful.Int64, CompletedDurationRuns: durationCount.Int64, AverageDurationMS: avgDuration.Float64, TotalCostUSD: totalCost.Float64, AverageCostUSD: avgCost.Float64, TotalTokens: tokens.Int64, InputTokens: inputTokens.Int64, OutputTokens: outputTokens.Int64, CacheReadTokens: cacheReadTokens.Int64, CacheCreationTokens: cacheCreationTokens.Int64, InputCostUSD: inputCost.Float64, OutputCostUSD: outputCost.Float64, CacheReadCostUSD: cacheReadCost.Float64, CacheCreationCostUSD: cacheCreationCost.Float64, TotalChargeMicroUSD: totalCharge.Int64, MeteredChargeMicroUSD: meteredCharge.Int64, UnpricedTokenCount: unpricedTokens.Int64, ReadCalls: readCalls.Int64, FileRereads: rereads.Int64}
	if m.TerminalRuns > 0 {
		m.SuccessRate = float64(m.SuccessfulRuns) / float64(m.TerminalRuns)
	}
	if m.SuccessfulRuns > 0 {
		m.ConsumptionPerSuccessfulCompletion = float64(m.TotalTokens) / float64(m.SuccessfulRuns)
	}
	if m.ReadCalls > 0 {
		m.FileRereadRate = float64(m.FileRereads) / float64(m.ReadCalls)
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

func (r *invocationReadModelRepository) ChargeByBasis(ctx context.Context, filter invocationreadmodel.Filter) ([]invocationreadmodel.ChargeByBasis, error) {
	where, args := invocationReadModelRunWhere(filter)
	query := `WITH filtered AS (SELECT invocation_read_model_runs.* FROM invocation_read_model_runs` + where + `)
		SELECT COALESCE(NULLIF(json_extract(r.billing_snapshot, '$.basis'), ''), 'unknown'),
		COUNT(*), COALESCE(SUM(filtered.total_charge_micro_usd), 0), COALESCE(SUM(filtered.total_tokens), 0),
		COALESCE(NULLIF(json_extract(r.billing_snapshot, '$.reason'), ''), '')
		FROM filtered LEFT JOIN runs r ON r.id = filtered.run_id
		GROUP BY 1, 5 ORDER BY 2 DESC, 1 ASC`
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []invocationreadmodel.ChargeByBasis{}
	for rows.Next() {
		var row invocationreadmodel.ChargeByBasis
		if err := rows.Scan(&row.Basis, &row.RunCount, &row.ChargeMicroUSD, &row.TokenCount, &row.ChargeReason); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *invocationReadModelRepository) RunDurationStatistics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.RunDurationStatistics, error) {
	where, args := invocationReadModelRunWhere(filter)
	query := `SELECT
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0),
		COALESCE(MIN(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0),
		COALESCE(MAX(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0),
		COUNT(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN 1 END)
		FROM invocation_read_model_runs` + where
	var average sql.NullFloat64
	var min, max, count sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&average, &min, &max, &count); err != nil {
		return invocationreadmodel.RunDurationStatistics{}, err
	}
	// The old SQLite endpoint used AVG for every percentile; preserve that
	// documented behavior exactly while reading from the durable projection.
	return invocationreadmodel.RunDurationStatistics{AverageDurationMS: average.Float64, P50DurationMS: average.Float64, P95DurationMS: average.Float64, P99DurationMS: average.Float64, MinDurationMS: min.Int64, MaxDurationMS: max.Int64, Count: count.Int64}, nil
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

func (r *invocationReadModelRepository) RunBreakdown(ctx context.Context, filter invocationreadmodel.Filter, dimension string, limit int) ([]invocationreadmodel.RunBreakdownRow, error) {
	columns := map[string]string{"runner": "runner_type", "model": "model", "profile": "profile_id", "workload": "workload_key", "workload_kind": "workload_kind"}
	column, ok := columns[dimension]
	if !ok {
		return nil, fmt.Errorf("unsupported durable run breakdown dimension %q", dimension)
	}
	if limit <= 0 {
		limit = 20
	}
	where, args := invocationReadModelRunWhere(filter)
	args = append(args, limit)
	value := column
	from := "invocation_read_model_runs"
	if dimension == "workload" {
		value = "COALESCE(workload_key, '')"
		column = value
	}
	if dimension == "profile" {
		// Profile display names are a retained product label, resolved from the
		// profile catalogue after selecting the durable run population. This is
		// deliberately not a reopened event-log interpretation.
		value = "COALESCE((SELECT name FROM agent_profiles WHERE id = invocation_read_model_runs.profile_id), invocation_read_model_runs.profile_id)"
	}
	query := fmt.Sprintf(`SELECT %s AS value, %s AS key, COUNT(*) AS run_count,
		SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) AS success_count,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_count,
		COALESCE(SUM(total_cost_usd), 0) AS total_cost_usd,
		COALESCE(SUM(total_tokens), 0) AS total_tokens,
		COALESCE(SUM(total_charge_micro_usd), 0) AS total_charge_micro_usd,
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0) AS avg_duration_ms,
		COALESCE(SUM(total_tokens) / NULLIF(SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END), 0), 0) AS consumption_per_successful_completion,
		COALESCE(SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(*), 0), 0) AS completion_rate
		FROM %s%s GROUP BY %s ORDER BY run_count DESC, value ASC LIMIT ?`, value, column, from, where, value)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.RunBreakdownRow{}
	for rows.Next() {
		var row invocationreadmodel.RunBreakdownRow
		if err := rows.Scan(&row.Value, &row.Key, &row.RunCount, &row.SuccessCount, &row.FailedCount, &row.TotalCostUSD, &row.TotalTokens, &row.TotalChargeMicroUSD, &row.AvgDurationMS, &row.ConsumptionPerSuccessfulCompletion, &row.CompletionRate); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) CanaryRuns(ctx context.Context, filter invocationreadmodel.Filter) ([]invocationreadmodel.CanaryRun, error) {
	clauses := []string{"irm.workload_kind NOT IN (?, ?)", "r.canary_arm IN (?, ?)"}
	args := []any{"imported", "interactive", "incumbent", "challenger"}
	if filter.From != nil {
		clauses = append(clauses, "irm.created_at >= ?")
		args = append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses = append(clauses, "irm.created_at < ?")
		args = append(args, SQLiteTime(*filter.To))
	}
	if filter.RunnerType != "" {
		clauses = append(clauses, "irm.runner_type = ?")
		args = append(args, filter.RunnerType)
	}
	query := `SELECT COALESCE(json_extract(r.resolved_config, '$.policySnapshot.roleRef'), ''), r.canary_arm, irm.status, COALESCE(irm.duration_ms, 0), COALESCE(irm.total_cost_usd, 0)
		FROM invocation_read_model_runs irm JOIN runs r ON r.id = irm.run_id WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY irm.created_at`
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.CanaryRun{}
	for rows.Next() {
		var item invocationreadmodel.CanaryRun
		if err := rows.Scan(&item.Role, &item.Arm, &item.Status, &item.DurationMS, &item.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, item)
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
		COALESCE(SUM(total_cost_usd), 0) AS total_cost_usd,
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
		if err := rows.Scan(&value, &row.TerminalRuns, &row.CompletedRuns, &row.FailedRuns, &row.CancelledRuns, &row.TotalCostUSD, &row.AvgDurationMS); err != nil {
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

func (r *invocationReadModelRepository) CapabilityUsage(ctx context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.CapabilityUsageRow, error) {
	if limit <= 0 {
		limit = 20
	}
	clauses := []string{"c.verified = 1", "c.target_scenario <> ''"}
	args := []any{}
	if filter.From != nil {
		clauses = append(clauses, "c.occurred_at >= ?")
		args = append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses = append(clauses, "c.occurred_at < ?")
		args = append(args, SQLiteTime(*filter.To))
	}
	if filter.TargetScenario != "" {
		clauses = append(clauses, "c.target_scenario = ?")
		args = append(args, filter.TargetScenario)
	}
	if filter.Operation != "" {
		clauses = append(clauses, "c.operation = ?")
		args = append(args, filter.Operation)
	}
	args = append(args, limit)
	rows, err := r.db.QueryxContext(ctx, `SELECT c.target_scenario, c.operation, COUNT(*) AS call_count,
		SUM(CASE WHEN c.outcome = 'success' THEN 1 ELSE 0 END) AS success_count,
		SUM(CASE WHEN c.outcome <> 'success' THEN 1 ELSE 0 END) AS failed_count,
		COALESCE(SUM(c.duration_ms), 0) AS total_duration_ms,
		COALESCE(SUM(COALESCE(f.arg_tokens, 0) + COALESCE(f.result_tokens, 0)), 0) AS total_tokens,
		COALESCE(SUM(COALESCE(f.arg_tokens, 0) + CASE WHEN f.token_basis = 'estimated' THEN COALESCE(f.result_tokens, 0) ELSE 0 END), 0) AS estimated_tokens
		FROM investigation_cross_scenario_calls c LEFT JOIN invocation_read_model_facts f ON f.run_id = c.run_id AND f.call_event_id = c.receipt_event_id WHERE `+strings.Join(clauses, " AND ")+`
		GROUP BY c.target_scenario, c.operation ORDER BY call_count DESC, c.target_scenario ASC, c.operation ASC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.CapabilityUsageRow{}
	for rows.Next() {
		var row invocationreadmodel.CapabilityUsageRow
		var estimated int64
		if err := rows.Scan(&row.TargetScenario, &row.Operation, &row.CallCount, &row.SuccessCount, &row.FailedCount, &row.TotalDurationMS, &row.TotalTokens, &estimated); err != nil {
			return nil, err
		}
		if row.TotalTokens > 0 {
			row.EstimatedTokenShare = float64(estimated) / float64(row.TotalTokens)
		}
		out = append(out, row)
	}
	return out, rows.Err()
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
		COALESCE(SUM(f.result_tokens * f.residency_turns), 0) AS residency_tokens,
		COALESCE(SUM(f.incurred_input_tokens + f.incurred_output_tokens + f.incurred_cache_read_tokens + f.incurred_cache_creation_tokens), 0) AS incurred_tokens,
		COALESCE(SUM(f.arg_tokens + CASE WHEN f.token_basis = 'estimated' THEN f.result_tokens ELSE 0 END), 0) AS estimated_footprint_tokens,
		COALESCE(SUM(CASE WHEN f.token_basis = 'estimated' THEN f.result_tokens * f.residency_turns ELSE 0 END), 0) AS estimated_residency_tokens,
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
	residualWhere, residualArgs := invocationReadModelRunWhere(filter)
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
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(unattributed_tokens), 0) FROM invocation_read_model_runs`+residualWhere, residualArgs...).Scan(&residualRuns, &residualTokens); err != nil {
		return nil, err
	}
	if residualRuns > 0 {
		out = append(out, invocationreadmodel.TokenAttributionRow{GroupBy: groupBy, Value: "unattributed", CallCount: residualRuns, TotalTokens: residualTokens})
	}
	return out, nil
}

func (r *invocationReadModelRepository) CapabilityEfficacy(ctx context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.CapabilityEfficacyRow, error) {
	if limit <= 0 {
		limit = 20
	}
	clauses := []string{"c.verified = 1", "c.target_scenario <> ''"}
	args := []any{}
	if filter.From != nil {
		clauses = append(clauses, "c.occurred_at >= ?")
		args = append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses = append(clauses, "c.occurred_at < ?")
		args = append(args, SQLiteTime(*filter.To))
	}
	if filter.TargetScenario != "" {
		clauses = append(clauses, "c.target_scenario = ?")
		args = append(args, filter.TargetScenario)
	}
	if filter.Operation != "" {
		clauses = append(clauses, "c.operation = ?")
		args = append(args, filter.Operation)
	}
	args = append(args, limit)
	query := `SELECT c.target_scenario, c.operation, COUNT(*) AS call_count,
		SUM(CASE WHEN c.outcome = 'success' THEN 1 ELSE 0 END) AS success_count,
		COALESCE(SUM(CASE WHEN EXISTS (SELECT 1 FROM invocation_read_model_episodes e WHERE e.run_id = c.run_id AND e.pattern = 'fallback-after-capability' AND e.suspected_owner_scenario = c.target_scenario) THEN 1 ELSE 0 END), 0) AS fallback_after_count,
		COALESCE(SUM(CASE WHEN EXISTS (SELECT 1 FROM invocation_read_model_episodes e WHERE e.run_id = c.run_id AND e.pattern = 'capability-abandoned' AND e.suspected_owner_scenario = c.target_scenario) THEN 1 ELSE 0 END), 0) AS abandoned_count
		FROM investigation_cross_scenario_calls c WHERE ` + strings.Join(clauses, " AND ") + ` GROUP BY c.target_scenario, c.operation ORDER BY call_count DESC, c.target_scenario ASC, c.operation ASC LIMIT ?`
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.CapabilityEfficacyRow{}
	for rows.Next() {
		var row invocationreadmodel.CapabilityEfficacyRow
		if err := rows.Scan(&row.TargetScenario, &row.Operation, &row.CallCount, &row.SuccessCount, &row.FallbackAfterCount, &row.AbandonedCount); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) ErrorPatterns(ctx context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.ErrorPattern, error) {
	if limit <= 0 {
		limit = 20
	}
	where, args := invocationReadModelErrorWhere(filter)
	args = append(args, limit)
	rows, err := r.db.QueryxContext(ctx, `SELECT error_code, COUNT(*) AS count, MAX(occurred_at) AS last_seen,
		(SELECT recent.run_id FROM invocation_read_model_errors recent WHERE recent.error_code = invocation_read_model_errors.error_code ORDER BY recent.occurred_at DESC, recent.run_id ASC LIMIT 1) AS sample_run_id
		FROM invocation_read_model_errors`+where+` GROUP BY error_code ORDER BY count DESC, error_code ASC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.ErrorPattern{}
	for rows.Next() {
		var row invocationreadmodel.ErrorPattern
		var lastSeen SQLiteTime
		if err := rows.Scan(&row.ErrorCode, &row.Count, &lastSeen, &row.SampleRunID); err != nil {
			return nil, err
		}
		row.LastSeen = lastSeen.Time()
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) FindingMetrics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.FindingMetrics, error) {
	where, args := invocationReadModelFindingWhere(filter)
	base := " FROM run_findings findings JOIN invocation_read_model_runs runs ON runs.run_id = findings.run_id" + where
	query := `SELECT COUNT(*),
		SUM(CASE WHEN findings.fingerprint IN (SELECT findings.fingerprint` + base + ` GROUP BY findings.fingerprint HAVING COUNT(*) > 1) THEN 1 ELSE 0 END),
		COUNT(DISTINCT CASE WHEN findings.fingerprint IN (SELECT findings.fingerprint` + base + ` GROUP BY findings.fingerprint HAVING COUNT(*) > 1) THEN findings.fingerprint END)` + base
	// Each subquery repeats the same filtered corpus, therefore its bind values
	// repeat in exactly the order the SQL text repeats it.
	allArgs := append(append(append([]any{}, args...), args...), args...)
	var total, recurring, fingerprints sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, allArgs...).Scan(&total, &recurring, &fingerprints); err != nil {
		return invocationreadmodel.FindingMetrics{}, err
	}
	m := invocationreadmodel.FindingMetrics{TotalFindings: total.Int64, RecurringFindings: recurring.Int64, RecurringFingerprints: fingerprints.Int64}
	if m.TotalFindings > 0 {
		m.RecurrenceRate = float64(m.RecurringFindings) / float64(m.TotalFindings)
	}
	return m, nil
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
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// invocationReadModelRunWhere applies the shared corpus filter to terminal
// summaries. Run dimensions and window bind directly to the run summary;
// invocation-only predicates select runs that have at least one matching
// durable invocation fact, preserving the cohort meaning without raw events.
func invocationReadModelRunWhere(filter invocationreadmodel.Filter) (string, []any) {
	clauses, args := []string{}, []any{}
	if filter.From != nil {
		clauses, args = append(clauses, "occurred_at >= ?"), append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses, args = append(clauses, "occurred_at < ?"), append(args, SQLiteTime(*filter.To))
	}
	add := func(column, value string) {
		if value != "" {
			clauses, args = append(clauses, column+" = ?"), append(args, value)
		}
	}
	add("run_id", filter.RunID)
	add("profile_id", filter.ProfileID)
	add("runner_type", filter.RunnerType)
	add("model", filter.Model)
	for _, excluded := range filter.ExcludedWorkloadKinds {
		if strings.TrimSpace(excluded) != "" {
			clauses = append(clauses, "workload_kind <> ?")
			args = append(args, excluded)
		}
	}
	add("status", filter.RunStatus)
	if filter.TagPrefix != "" {
		clauses, args = append(clauses, "tag LIKE ?"), append(args, filter.TagPrefix+"%")
	}
	if filter.ErrorCode != "" {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_errors e WHERE e.run_id = invocation_read_model_runs.run_id AND e.error_code = ?)")
		args = append(args, filter.ErrorCode)
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

// invocationReadModelEpisodeWhere applies the shared cohort predicate to the
// durable episode rows themselves. Invocation predicates remain explicit
// EXISTS clauses, so a cohort never needs to reopen raw events.
func invocationReadModelEpisodeWhere(filter invocationreadmodel.Filter) (string, []any) {
	clauses, args := []string{}, []any{}
	if filter.From != nil {
		clauses, args = append(clauses, "occurred_at >= ?"), append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses, args = append(clauses, "occurred_at < ?"), append(args, SQLiteTime(*filter.To))
	}
	for _, predicate := range []struct{ column, value string }{{"profile_id", filter.ProfileID}, {"runner_type", filter.RunnerType}, {"model", filter.Model}, {"run_status", filter.RunStatus}, {"pattern", filter.EpisodePattern}, {"cause_scope", filter.EpisodeCauseScope}, {"fingerprint", filter.EpisodeFingerprint}} {
		if predicate.value != "" {
			clauses, args = append(clauses, predicate.column+" = ?"), append(args, predicate.value)
		}
	}
	if filter.TagPrefix != "" {
		clauses, args = append(clauses, "tag LIKE ?"), append(args, filter.TagPrefix+"%")
	}
	invocationClauses, invocationArgs := []string{"f.run_id = invocation_read_model_episodes.run_id"}, []any{}
	for _, predicate := range []struct{ column, value string }{{"ownership", filter.Ownership}, {"outcome", filter.Outcome}, {"executable", filter.Executable}, {"fingerprint", filter.Fingerprint}, {"tool_name", filter.ToolName}} {
		if predicate.value != "" {
			invocationClauses, invocationArgs = append(invocationClauses, "f."+predicate.column+" = ?"), append(invocationArgs, predicate.value)
		}
	}
	if len(invocationClauses) > 1 {
		clauses, args = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_facts f WHERE "+strings.Join(invocationClauses, " AND ")+")"), append(args, invocationArgs...)
	}
	if filter.SelfReportRuleID != "" || filter.SelfReportCauseScope != "" {
		spanClauses, spanArgs := []string{"s.run_id = invocation_read_model_episodes.run_id"}, []any{}
		for _, predicate := range []struct{ column, value string }{{"rule_id", filter.SelfReportRuleID}, {"cause_scope", filter.SelfReportCauseScope}} {
			if predicate.value != "" {
				spanClauses, spanArgs = append(spanClauses, "s."+predicate.column+" = ?"), append(spanArgs, predicate.value)
			}
		}
		clauses, args = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_self_report_spans s WHERE "+strings.Join(spanClauses, " AND ")+")"), append(args, spanArgs...)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// invocationReadModelErrorWhere mirrors the legacy error-pattern scope: time
// and run dimensions only. Error events do not carry an invocation identity,
// so ownership/outcome/tool predicates must not silently imply one.
func invocationReadModelErrorWhere(filter invocationreadmodel.Filter) (string, []any) {
	clauses, args := []string{}, []any{}
	if filter.From != nil {
		clauses, args = append(clauses, "occurred_at >= ?"), append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses, args = append(clauses, "occurred_at < ?"), append(args, SQLiteTime(*filter.To))
	}
	for _, predicate := range []struct{ column, value string }{{"profile_id", filter.ProfileID}, {"runner_type", filter.RunnerType}, {"model", filter.Model}} {
		if predicate.value != "" {
			clauses, args = append(clauses, predicate.column+" = ?"), append(args, predicate.value)
		}
	}
	if filter.ErrorCode != "" {
		clauses, args = append(clauses, "error_code = ?"), append(args, filter.ErrorCode)
	}
	if filter.TagPrefix != "" {
		clauses, args = append(clauses, "tag LIKE ?"), append(args, filter.TagPrefix+"%")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func invocationReadModelFindingWhere(filter invocationreadmodel.Filter) (string, []any) {
	clauses, args := []string{}, []any{}
	if filter.From != nil {
		clauses, args = append(clauses, "findings.created_at >= ?"), append(args, SQLiteTime(*filter.From))
	}
	if filter.To != nil {
		clauses, args = append(clauses, "findings.created_at < ?"), append(args, SQLiteTime(*filter.To))
	}
	add := func(column, value string) {
		if value != "" {
			clauses, args = append(clauses, "runs."+column+" = ?"), append(args, value)
		}
	}
	add("run_id", filter.RunID)
	add("profile_id", filter.ProfileID)
	add("runner_type", filter.RunnerType)
	add("model", filter.Model)
	add("status", filter.RunStatus)
	if filter.TagPrefix != "" {
		clauses, args = append(clauses, "runs.tag LIKE ?"), append(args, filter.TagPrefix+"%")
	}
	invocationClauses, invocationArgs := []string{"invocations.run_id = runs.run_id"}, []any{}
	for _, predicate := range []struct{ column, value string }{{"ownership", filter.Ownership}, {"outcome", filter.Outcome}, {"executable", filter.Executable}, {"fingerprint", filter.Fingerprint}, {"tool_name", filter.ToolName}} {
		if predicate.value != "" {
			invocationClauses, invocationArgs = append(invocationClauses, "invocations."+predicate.column+" = ?"), append(invocationArgs, predicate.value)
		}
	}
	if len(invocationClauses) > 1 {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_facts invocations WHERE "+strings.Join(invocationClauses, " AND ")+")")
		args = append(args, invocationArgs...)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (r *invocationReadModelRepository) Facts(ctx context.Context, runID string) ([]invocationreadmodel.Fact, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT run_id,classifier_version,call_event_id,COALESCE(result_event_id,''),COALESCE(tool_call_id,''),occurred_at,time_basis,tool_name,COALESCE(capability,'other'),COALESCE(intent_class,''),COALESCE(executable,''),COALESCE(command_path,''),ownership,COALESCE(ownership_reason,''),COALESCE(segment_index,0),COALESCE(segment_count,1),COALESCE(catalog_snapshot,''),outcome,COALESCE(pairing_basis,'unpaired'),COALESCE(failure_signature,''),COALESCE(signature_truncated,0),COALESCE(retry_of_call_event_id,''),help_recovery,fingerprint,availability,profile_id,runner_type,model,tag,run_status,COALESCE(semantics_kind,''),COALESCE(semantics_verdict,''),COALESCE(wrapper,''),exit_code,duration_ms,COALESCE(arg_tokens,0),COALESCE(result_tokens,0),COALESCE(token_basis,'unknown'),COALESCE(residency_turns,0),COALESCE(residency_segment,0),COALESCE(incurred_input_tokens,0),COALESCE(incurred_output_tokens,0),COALESCE(incurred_cache_read_tokens,0),COALESCE(incurred_cache_creation_tokens,0) FROM invocation_read_model_facts WHERE run_id=? ORDER BY occurred_at, call_event_id, segment_index`, runID)
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
		if err := rows.Scan(&f.RunID, &f.Version, &f.CallEventID, &f.ResultEventID, &f.ToolCallID, &occurredAt, &f.TimeBasis, &f.ToolName, &f.Capability, &f.IntentClass, &f.Executable, &f.CommandPath, &f.Ownership, &f.OwnershipReason, &f.SegmentIndex, &f.SegmentCount, &f.CatalogSnapshot, &f.Outcome, &f.PairingBasis, &f.FailureSignature, &f.SignatureTruncated, &f.RetryOfCallEventID, &f.HelpRecovery, &f.Fingerprint, &f.Availability, &f.ProfileID, &f.RunnerType, &f.Model, &f.Tag, &f.RunStatus, &f.SemanticsKind, &f.SemanticsVerdict, &f.Wrapper, &exitCode, &durationMS, &f.ArgTokens, &f.ResultTokens, &tokenBasis, &f.ResidencyTurns, &f.ResidencySegment, &f.IncurredInputTokens, &f.IncurredOutputTokens, &f.IncurredCacheReadTokens, &f.IncurredCacheCreationTokens); err != nil {
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
		EpisodeClassifierVersion    string     `db:"episode_classifier_version"`
		SelfReportClassifierVersion string     `db:"self_report_classifier_version"`
		ProjectedAt                 SQLiteTime `db:"projected_at"`
	}
	err := r.db.GetContext(ctx, &row, `SELECT run_id,last_event_id,last_event_at,classifier_version,episode_classifier_version,self_report_classifier_version,projected_at FROM invocation_read_model_watermarks WHERE run_id=?`, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &invocationreadmodel.Watermark{RunID: row.RunID, LastEventID: row.LastEventID, LastEventAt: row.LastEventAt.Time(), ClassifierVersion: row.ClassifierVersion, EpisodeClassifierVersion: row.EpisodeClassifierVersion, SelfReportClassifierVersion: row.SelfReportClassifierVersion, ProjectedAt: row.ProjectedAt.Time()}, nil
}
