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
)

type invocationReadModelRepository struct{ db *DB }

var (
	_ invocationreadmodel.Store            = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.RunStore         = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.GoalOutcomeStore = (*invocationReadModelRepository)(nil)
	_ invocationreadmodel.ProjectionStore  = (*invocationReadModelRepository)(nil)
)

func nullableSQLiteTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return SQLiteTime(*value)
}

func (r *invocationReadModelRepository) ReplaceRun(ctx context.Context, fact invocationreadmodel.RunFact) error {
	return replaceRunTx(ctx, r.db, fact)
}

func (r *invocationReadModelRepository) ReplaceGoalOutcome(ctx context.Context, goal invocationreadmodel.GoalOutcome) error {
	_, err := r.db.ExecContext(ctx, `UPDATE invocation_read_model_runs SET goal_id=?,goal_status=?,goal_token_budget=?,goal_tokens_used=?,goal_time_used_seconds=? WHERE run_id=?`, goal.GoalID, goal.Status, goal.TokenBudget, goal.TokensUsed, goal.TimeUsedSeconds, goal.RunID)
	if err != nil {
		return fmt.Errorf("persist goal outcome: %w", err)
	}
	return nil
}

func (r *invocationReadModelRepository) GoalOutcome(ctx context.Context, runID string) (*invocationreadmodel.GoalOutcome, error) {
	var row struct {
		RunID           string        `db:"run_id"`
		GoalID          string        `db:"goal_id"`
		Status          string        `db:"goal_status"`
		TokenBudget     sql.NullInt64 `db:"goal_token_budget"`
		TokensUsed      int64         `db:"goal_tokens_used"`
		TimeUsedSeconds int64         `db:"goal_time_used_seconds"`
	}
	err := r.db.GetContext(ctx, &row, `SELECT run_id,goal_id,goal_status,goal_token_budget,goal_tokens_used,goal_time_used_seconds FROM invocation_read_model_runs WHERE run_id=? AND goal_id<>''`, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &invocationreadmodel.GoalOutcome{RunID: row.RunID, GoalID: row.GoalID, Status: row.Status, TokensUsed: row.TokensUsed, TimeUsedSeconds: row.TimeUsedSeconds}
	if row.TokenBudget.Valid {
		value := row.TokenBudget.Int64
		out.TokenBudget = &value
	}
	return out, nil
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func replaceRunTx(ctx context.Context, executor sqlExecutor, fact invocationreadmodel.RunFact) error {
	if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_runs (run_id,occurred_at,created_at,started_at,ended_at,duration_ms,status,profile_id,runner_type,model,tag,total_cost_usd,authoritative_cost_usd,estimated_cost_usd,unknown_cost_usd,input_cost_usd,output_cost_usd,cache_read_cost_usd,cache_creation_cost_usd,total_tokens,input_tokens,output_tokens,cache_read_tokens,cache_creation_tokens,cost_time_basis,projected_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id) DO UPDATE SET occurred_at=excluded.occurred_at,created_at=excluded.created_at,started_at=excluded.started_at,ended_at=excluded.ended_at,duration_ms=excluded.duration_ms,status=excluded.status,profile_id=excluded.profile_id,runner_type=excluded.runner_type,model=excluded.model,tag=excluded.tag,total_cost_usd=excluded.total_cost_usd,authoritative_cost_usd=excluded.authoritative_cost_usd,estimated_cost_usd=excluded.estimated_cost_usd,unknown_cost_usd=excluded.unknown_cost_usd,input_cost_usd=excluded.input_cost_usd,output_cost_usd=excluded.output_cost_usd,cache_read_cost_usd=excluded.cache_read_cost_usd,cache_creation_cost_usd=excluded.cache_creation_cost_usd,total_tokens=excluded.total_tokens,input_tokens=excluded.input_tokens,output_tokens=excluded.output_tokens,cache_read_tokens=excluded.cache_read_tokens,cache_creation_tokens=excluded.cache_creation_tokens,cost_time_basis=excluded.cost_time_basis,projected_at=excluded.projected_at`, fact.RunID, SQLiteTime(fact.OccurredAt), SQLiteTime(fact.CreatedAt), nullableSQLiteTime(fact.StartedAt), nullableSQLiteTime(fact.EndedAt), fact.DurationMS, fact.Status, fact.ProfileID, fact.RunnerType, fact.Model, fact.Tag, fact.TotalCostUSD, fact.AuthoritativeCostUSD, fact.EstimatedCostUSD, fact.UnknownCostUSD, fact.InputCostUSD, fact.OutputCostUSD, fact.CacheReadCostUSD, fact.CacheCreationCostUSD, fact.TotalTokens, fact.InputTokens, fact.OutputTokens, fact.CacheReadTokens, fact.CacheCreationTokens, fact.CostTimeBasis, SQLiteTime(fact.ProjectedAt)); err != nil {
		return fmt.Errorf("upsert run read-model fact: %w", err)
	}
	if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_run_signals (run_id,read_calls,files_read_more_than_once) VALUES (?,?,?) ON CONFLICT(run_id) DO UPDATE SET read_calls=excluded.read_calls,files_read_more_than_once=excluded.files_read_more_than_once`, fact.RunID, fact.ReadCalls, fact.FileRereads); err != nil {
		return fmt.Errorf("upsert run read-model signals: %w", err)
	}
	if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_time_accounting (run_id,model_generating_ms,tool_executing_ms,idle_waiting_ms,awaiting_human_ms,unattributable_ms,model_tokens,tool_tokens,idle_tokens,human_tokens,unattributable_tokens) VALUES (?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id) DO UPDATE SET model_generating_ms=excluded.model_generating_ms,tool_executing_ms=excluded.tool_executing_ms,idle_waiting_ms=excluded.idle_waiting_ms,awaiting_human_ms=excluded.awaiting_human_ms,unattributable_ms=excluded.unattributable_ms,model_tokens=excluded.model_tokens,tool_tokens=excluded.tool_tokens,idle_tokens=excluded.idle_tokens,human_tokens=excluded.human_tokens,unattributable_tokens=excluded.unattributable_tokens`, fact.RunID, fact.TimeAccounting.ModelGeneratingMS, fact.TimeAccounting.ToolExecutingMS, fact.TimeAccounting.IdleWaitingMS, fact.TimeAccounting.AwaitingHumanMS, fact.TimeAccounting.UnattributableMS, fact.TimeAccounting.ModelTokens, fact.TimeAccounting.ToolTokens, fact.TimeAccounting.IdleTokens, fact.TimeAccounting.HumanTokens, fact.TimeAccounting.UnattributableTokens); err != nil {
		return fmt.Errorf("upsert run time accounting: %w", err)
	}
	return nil
}

func (r *invocationReadModelRepository) TimeAccountingForRun(ctx context.Context, runID string) (runsignal.TimeAccounting, bool, error) {
	var value runsignal.TimeAccounting
	err := r.db.GetContext(ctx, &value, `SELECT model_generating_ms,tool_executing_ms,idle_waiting_ms,awaiting_human_ms,unattributable_ms,model_tokens,tool_tokens,idle_tokens,human_tokens,unattributable_tokens FROM invocation_read_model_time_accounting WHERE run_id=?`, runID)
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
		if _, err := executor.ExecContext(ctx, `INSERT INTO invocation_read_model_self_report_spans (run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,profile_id,runner_type,model,tag,run_status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, span.RunID, span.EventID, span.RuleID, span.StartOffset, span.EndOffset, SQLiteTime(span.OccurredAt), span.TimeBasis, span.ClassifierVersion, span.CauseScope, span.Text, span.ProfileID, span.RunnerType, span.Model, span.Tag, span.RunStatus); err != nil {
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
		if _, err := tx.ExecContext(ctx, `INSERT INTO invocation_read_model_facts (run_id,call_event_id,result_event_id,tool_call_id,occurred_at,time_basis,tool_name,executable,command_path,ownership,catalog_snapshot,outcome,pairing_basis,failure_signature,signature_truncated,retry_of_call_event_id,help_recovery,fingerprint,availability,classifier_version,profile_id,runner_type,model,tag,run_status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, fact.RunID, fact.CallEventID, fact.ResultEventID, fact.ToolCallID, SQLiteTime(fact.OccurredAt), fact.TimeBasis, fact.ToolName, fact.Executable, fact.CommandPath, fact.Ownership, fact.CatalogSnapshot, fact.Outcome, fact.PairingBasis, fact.FailureSignature, fact.SignatureTruncated, fact.RetryOfCallEventID, fact.HelpRecovery, fact.Fingerprint, fact.Availability, fact.Version, fact.ProfileID, fact.RunnerType, fact.Model, fact.Tag, fact.RunStatus); err != nil {
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
	return out, rows.Err()
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
	rows, err := r.db.QueryxContext(ctx, `SELECT run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,profile_id,runner_type,model,tag,run_status FROM invocation_read_model_self_report_spans WHERE run_id=? ORDER BY occurred_at,event_id,start_offset`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.SelfReportSpan{}
	for rows.Next() {
		var s invocationreadmodel.SelfReportSpan
		var occurredAt SQLiteTime
		if err := rows.Scan(&s.RunID, &s.EventID, &s.RuleID, &s.StartOffset, &s.EndOffset, &occurredAt, &s.TimeBasis, &s.ClassifierVersion, &s.CauseScope, &s.Text, &s.ProfileID, &s.RunnerType, &s.Model, &s.Tag, &s.RunStatus); err != nil {
			return nil, err
		}
		s.OccurredAt = occurredAt.Time()
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *invocationReadModelRepository) Metrics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.Metrics, error) {
	where, args := invocationReadModelWhere(filter)
	query := `SELECT COUNT(*),
		SUM(CASE WHEN ownership <> 'unknown' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership = 'external' THEN 1 ELSE 0 END),
		SUM(CASE WHEN ownership = 'unknown' THEN 1 ELSE 0 END),
		SUM(CASE WHEN outcome = 'failure' THEN 1 ELSE 0 END),
		SUM(CASE WHEN retry_of_call_event_id <> '' THEN 1 ELSE 0 END),
		SUM(CASE WHEN help_recovery = 1 THEN 1 ELSE 0 END),
		SUM(CASE WHEN fingerprint IN (SELECT fingerprint FROM invocation_read_model_facts` + where + ` GROUP BY fingerprint HAVING COUNT(*) > 1) THEN 1 ELSE 0 END),
		COALESCE((SELECT MAX(bucket_count) FROM (SELECT COUNT(*) AS bucket_count FROM invocation_read_model_facts` + where + ` GROUP BY fingerprint)), 0)
		FROM invocation_read_model_facts` + where
	// The repeated-fingerprint subquery has the same predicate, so bind it
	// twice; it cannot accidentally measure a wider population than the rates.
	allArgs := append(append(append([]any{}, args...), args...), args...)
	var values [9]sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, allArgs...).Scan(&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7], &values[8]); err != nil {
		return invocationreadmodel.Metrics{}, err
	}
	m := invocationreadmodel.Metrics{TotalCalls: values[0].Int64, ResolvedCalls: values[1].Int64, ExternalCalls: values[2].Int64, UnknownCalls: values[3].Int64, FailedCalls: values[4].Int64, RetryCalls: values[5].Int64, HelpRecoveries: values[6].Int64, RepeatedCalls: values[7].Int64, LargestFingerprintBucket: values[8].Int64}
	if m.ResolvedCalls > 0 {
		m.ExternalToolShare = float64(m.ExternalCalls) / float64(m.ResolvedCalls)
	}
	if m.TotalCalls > 0 {
		m.FailureRate = float64(m.FailedCalls) / float64(m.TotalCalls)
		m.RetryRate = float64(m.RetryCalls) / float64(m.TotalCalls)
		m.HelpRecoveryRate = float64(m.HelpRecoveries) / float64(m.TotalCalls)
		m.RepeatedWorkRate = float64(m.RepeatedCalls) / float64(m.TotalCalls)
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
		COALESCE(SUM(authoritative_cost_usd), 0), COALESCE(SUM(estimated_cost_usd), 0), COALESCE(SUM(unknown_cost_usd), 0),
		COALESCE(AVG(total_cost_usd), 0),
		COALESCE(SUM(total_tokens), 0),
		COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_creation_tokens), 0),
		COALESCE(SUM(input_cost_usd), 0), COALESCE(SUM(output_cost_usd), 0), COALESCE(SUM(cache_read_cost_usd), 0), COALESCE(SUM(cache_creation_cost_usd), 0),
		COALESCE(SUM(signals.read_calls), 0),
		COALESCE(SUM(signals.files_read_more_than_once), 0)
		FROM invocation_read_model_runs LEFT JOIN invocation_read_model_run_signals signals ON signals.run_id = invocation_read_model_runs.run_id` + where
	var total, terminal, successful, durationCount, tokens, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens, readCalls, rereads sql.NullInt64
	var avgDuration, totalCost, authoritativeCost, estimatedCost, unknownCost, avgCost, inputCost, outputCost, cacheReadCost, cacheCreationCost sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total, &terminal, &successful, &durationCount, &avgDuration, &totalCost, &authoritativeCost, &estimatedCost, &unknownCost, &avgCost, &tokens, &inputTokens, &outputTokens, &cacheReadTokens, &cacheCreationTokens, &inputCost, &outputCost, &cacheReadCost, &cacheCreationCost, &readCalls, &rereads); err != nil {
		return invocationreadmodel.RunMetrics{}, err
	}
	m := invocationreadmodel.RunMetrics{TotalRuns: total.Int64, TerminalRuns: terminal.Int64, SuccessfulRuns: successful.Int64, CompletedDurationRuns: durationCount.Int64, AverageDurationMS: avgDuration.Float64, TotalCostUSD: totalCost.Float64, AuthoritativeCostUSD: authoritativeCost.Float64, EstimatedCostUSD: estimatedCost.Float64, UnknownCostUSD: unknownCost.Float64, AverageCostUSD: avgCost.Float64, TotalTokens: tokens.Int64, InputTokens: inputTokens.Int64, OutputTokens: outputTokens.Int64, CacheReadTokens: cacheReadTokens.Int64, CacheCreationTokens: cacheCreationTokens.Int64, InputCostUSD: inputCost.Float64, OutputCostUSD: outputCost.Float64, CacheReadCostUSD: cacheReadCost.Float64, CacheCreationCostUSD: cacheCreationCost.Float64, ReadCalls: readCalls.Int64, FileRereads: rereads.Int64}
	if m.TerminalRuns > 0 {
		m.SuccessRate = float64(m.SuccessfulRuns) / float64(m.TerminalRuns)
	}
	if m.ReadCalls > 0 {
		m.FileRereadRate = float64(m.FileRereads) / float64(m.ReadCalls)
	}
	return m, nil
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
	columns := map[string]string{"runner": "runner_type", "model": "model", "profile": "profile_id"}
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
		COALESCE(AVG(CASE WHEN started_at IS NOT NULL AND ended_at IS NOT NULL THEN duration_ms END), 0) AS avg_duration_ms
		FROM %s%s GROUP BY %s ORDER BY run_count DESC, value ASC LIMIT ?`, value, column, from, where, value)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.RunBreakdownRow{}
	for rows.Next() {
		var row invocationreadmodel.RunBreakdownRow
		if err := rows.Scan(&row.Value, &row.Key, &row.RunCount, &row.SuccessCount, &row.FailedCount, &row.TotalCostUSD, &row.TotalTokens, &row.AvgDurationMS); err != nil {
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
		SUM(CASE WHEN outcome = 'failure' THEN 1 ELSE 0 END) AS failed_count
		FROM invocation_read_model_facts`+where+` GROUP BY tool_name ORDER BY call_count DESC, tool_name ASC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.ToolUsageRow{}
	for rows.Next() {
		var row invocationreadmodel.ToolUsageRow
		if err := rows.Scan(&row.ToolName, &row.CallCount, &row.SuccessCount, &row.FailedCount); err != nil {
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
	add("profile_id", filter.ProfileID)
	add("runner_type", filter.RunnerType)
	add("model", filter.Model)
	add("run_status", filter.RunStatus)
	add("tool_name", filter.ToolName)
	if filter.GoalID != "" {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_runs irm WHERE irm.run_id=invocation_read_model_facts.run_id AND irm.goal_id=?)")
		args = append(args, filter.GoalID)
	}
	if filter.GoalStatus != "" {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_runs irm WHERE irm.run_id=invocation_read_model_facts.run_id AND irm.goal_status=?)")
		args = append(args, filter.GoalStatus)
	}
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
	add("profile_id", filter.ProfileID)
	add("runner_type", filter.RunnerType)
	add("model", filter.Model)
	add("status", filter.RunStatus)
	add("goal_status", filter.GoalStatus)
	add("goal_id", filter.GoalID)
	if filter.TagPrefix != "" {
		clauses, args = append(clauses, "tag LIKE ?"), append(args, filter.TagPrefix+"%")
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
	if filter.GoalID != "" {
		clauses, args = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_runs irm WHERE irm.run_id=invocation_read_model_episodes.run_id AND irm.goal_id=?)"), append(args, filter.GoalID)
	}
	if filter.GoalStatus != "" {
		clauses, args = append(clauses, "EXISTS (SELECT 1 FROM invocation_read_model_runs irm WHERE irm.run_id=invocation_read_model_episodes.run_id AND irm.goal_status=?)"), append(args, filter.GoalStatus)
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
	rows, err := r.db.QueryxContext(ctx, `SELECT run_id,classifier_version,call_event_id,COALESCE(result_event_id,''),COALESCE(tool_call_id,''),occurred_at,time_basis,tool_name,COALESCE(executable,''),COALESCE(command_path,''),ownership,COALESCE(catalog_snapshot,''),outcome,COALESCE(pairing_basis,'unpaired'),COALESCE(failure_signature,''),COALESCE(signature_truncated,0),COALESCE(retry_of_call_event_id,''),help_recovery,fingerprint,availability,profile_id,runner_type,model,tag,run_status FROM invocation_read_model_facts WHERE run_id=? ORDER BY occurred_at, call_event_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invocationreadmodel.Fact{}
	for rows.Next() {
		var f invocationreadmodel.Fact
		var occurredAt SQLiteTime
		if err := rows.Scan(&f.RunID, &f.Version, &f.CallEventID, &f.ResultEventID, &f.ToolCallID, &occurredAt, &f.TimeBasis, &f.ToolName, &f.Executable, &f.CommandPath, &f.Ownership, &f.CatalogSnapshot, &f.Outcome, &f.PairingBasis, &f.FailureSignature, &f.SignatureTruncated, &f.RetryOfCallEventID, &f.HelpRecovery, &f.Fingerprint, &f.Availability, &f.ProfileID, &f.RunnerType, &f.Model, &f.Tag, &f.RunStatus); err != nil {
			return nil, err
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
