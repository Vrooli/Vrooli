package bindings

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"program-runtime/internal/sessions"
)

type Invocation struct {
	InvocationID      string
	BindingID         string
	TargetScenario    string
	SessionID         string
	ProgramID         string
	Provenance        string
	Outcome           string
	Reason            string
	LatencyMS         int64
	UsageInputTokens  int64
	UsageOutputTokens int64
	UsageCostMicros   int64
	Origin            string
	InvocationClass   string
	OccurredAt        time.Time
}

type InvocationRecorder interface {
	RecordInvocation(context.Context, Invocation) error
	ListInvocations(context.Context, time.Time, string, string) ([]Invocation, error)
}

type invocationRepository struct{ db sessions.SQLExecutor }

func NewInvocationRepository(db sessions.SQLExecutor) InvocationRecorder {
	return &invocationRepository{db: db}
}

func (r *invocationRepository) RecordInvocation(ctx context.Context, in Invocation) error {
	if in.InvocationID == "" {
		in.InvocationID = "inv_" + uuid.NewString()
	}
	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}
	if in.Origin == "" {
		in.Origin = invocationOrigin(in)
	}
	if in.InvocationClass == "" {
		in.InvocationClass = classifyInvocation(in)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO binding_invocations
 (invocation_id, binding_id, target_scenario, session_id, program_id, provenance, outcome, reason, latency_ms, usage_input_tokens, usage_output_tokens, usage_cost_micros, origin, invocation_class, occurred_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.InvocationID, in.BindingID, in.TargetScenario, in.SessionID, in.ProgramID, in.Provenance, in.Outcome, in.Reason, in.LatencyMS, in.UsageInputTokens, in.UsageOutputTokens, in.UsageCostMicros, in.Origin, in.InvocationClass, in.OccurredAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record binding invocation: %w", err)
	}
	return nil
}

func (r *invocationRepository) ListInvocations(ctx context.Context, since time.Time, bindingID, scenario string) ([]Invocation, error) {
	query := `SELECT invocation_id, binding_id, target_scenario, session_id, program_id, provenance, outcome, reason, latency_ms, usage_input_tokens, usage_output_tokens, usage_cost_micros, origin, invocation_class, occurred_at FROM binding_invocations WHERE occurred_at >= ?`
	args := []any{since.UTC().Format(time.RFC3339Nano)}
	if bindingID != "" {
		query += " AND binding_id = ?"
		args = append(args, bindingID)
	}
	if scenario != "" {
		query += " AND target_scenario = ?"
		args = append(args, scenario)
	}
	query += " ORDER BY occurred_at, invocation_id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list binding invocations: %w", err)
	}
	defer rows.Close()
	var out []Invocation
	for rows.Next() {
		var in Invocation
		var occurred string
		if err := rows.Scan(&in.InvocationID, &in.BindingID, &in.TargetScenario, &in.SessionID, &in.ProgramID, &in.Provenance, &in.Outcome, &in.Reason, &in.LatencyMS, &in.UsageInputTokens, &in.UsageOutputTokens, &in.UsageCostMicros, &in.Origin, &in.InvocationClass, &occurred); err != nil {
			return nil, fmt.Errorf("scan binding invocation: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, occurred)
		if err != nil {
			return nil, fmt.Errorf("parse binding invocation time: %w", err)
		}
		in.OccurredAt = parsed
		out = append(out, in)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate binding invocations: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].OccurredAt.Before(out[j].OccurredAt) })
	return out, nil
}

func invocationOrigin(in Invocation) string {
	if in.ProgramID == "sweep" || in.Provenance == "PROVENANCE_OPERATOR" || in.Provenance == "" {
		return "synthetic"
	}
	return "organic"
}

func classifyInvocation(in Invocation) string {
	if in.Outcome == "success" {
		return "success"
	}
	if in.Outcome == "refused" {
		return "refused"
	}
	reason := strings.ToLower(in.Reason)
	if strings.Contains(reason, "invalid arguments") || strings.Contains(reason, "missing required field") || strings.Contains(reason, "requires an explicit grant") || strings.Contains(reason, "requires explicit confirmation") {
		return "probe_invalid_argument"
	}
	if strings.Contains(reason, "deadline") || strings.Contains(reason, "timeout") {
		return "probe_timeout"
	}
	if strings.Contains(reason, "404") || strings.Contains(reason, "page not found") || strings.Contains(reason, "connection refused") || strings.Contains(reason, "not running") {
		return "target_unavailable"
	}
	return "target_failed"
}
