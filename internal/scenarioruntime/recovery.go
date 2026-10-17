package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// UpsertRecoveryPolicy stores an explicit desired-state declaration. A
// workload is not recoverable until an operator enables a critical policy;
// zero-value policy fields therefore fail closed.
func (s *SQLiteStore) UpsertRecoveryPolicy(ctx context.Context, policy RecoveryPolicy) (RecoveryPolicy, error) {
	if strings.TrimSpace(policy.Scenario) == "" {
		return RecoveryPolicy{}, fmt.Errorf("upsert recovery policy: scenario is required")
	}
	policy.Variant = InstanceKey{Scenario: policy.Scenario, Variant: policy.Variant}.Normalize().Variant
	if policy.DependencyTier < 0 {
		return RecoveryPolicy{}, fmt.Errorf("upsert recovery policy: dependency tier must be non-negative")
	}
	if policy.RetryBudget < 0 {
		return RecoveryPolicy{}, fmt.Errorf("upsert recovery policy: retry budget must be non-negative")
	}
	policy.UpdatedAt = s.now()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_recovery_policies (
  scenario, variant, critical, dependency_tier, enabled, retry_budget, opt_out, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scenario, variant) DO UPDATE SET
  critical = excluded.critical,
  dependency_tier = excluded.dependency_tier,
  enabled = excluded.enabled,
  retry_budget = excluded.retry_budget,
  opt_out = excluded.opt_out,
  updated_at = excluded.updated_at`,
		policy.Scenario, policy.Variant, policy.Critical, policy.DependencyTier,
		policy.Enabled, policy.RetryBudget, policy.OptOut, formatTime(policy.UpdatedAt))
	if err != nil {
		return RecoveryPolicy{}, fmt.Errorf("upsert runtime recovery policy: %w", err)
	}
	return policy, nil
}

func (s *SQLiteStore) GetRecoveryPolicy(ctx context.Context, scenario, variant string) (RecoveryPolicy, error) {
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	return scanRecoveryPolicy(s.db.QueryRowContext(ctx, recoveryPolicySelectSQL+` WHERE scenario = ? AND variant = ?`, scenario, variant))
}

func (s *SQLiteStore) ListRecoveryPolicies(ctx context.Context, filter RecoveryPolicyFilter) ([]RecoveryPolicy, error) {
	query := recoveryPolicySelectSQL
	var clauses []string
	var args []any
	if strings.TrimSpace(filter.Scenario) != "" {
		clauses = append(clauses, "scenario = ?")
		args = append(args, filter.Scenario)
	}
	if strings.TrimSpace(filter.Variant) != "" {
		clauses = append(clauses, "variant = ?")
		args = append(args, InstanceKey{Scenario: filter.Scenario, Variant: filter.Variant}.Normalize().Variant)
	}
	if filter.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, *filter.Enabled)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY dependency_tier ASC, scenario ASC, variant ASC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runtime recovery policies: %w", err)
	}
	defer rows.Close()
	return scanRecoveryPolicies(rows)
}

func (s *SQLiteStore) CreatePressureEpoch(ctx context.Context, epoch PressureEpoch) (PressureEpoch, error) {
	if strings.TrimSpace(epoch.EpochID) == "" {
		epoch.EpochID = newID("pressure")
	}
	if strings.TrimSpace(epoch.Status) == "" {
		epoch.Status = PressureEpochDetected
	}
	now := s.now()
	if epoch.DetectedAt.IsZero() {
		epoch.DetectedAt = now
	}
	epoch.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_pressure_epochs (epoch_id, status, source, detected_at, cleared_at, updated_at, details_json)
VALUES (?, ?, ?, ?, ?, ?, ?)`, epoch.EpochID, epoch.Status, epoch.Source,
		formatTime(epoch.DetectedAt), formatOptionalTime(epoch.ClearedAt), formatTime(epoch.UpdatedAt), epoch.DetailsJSON)
	if err != nil {
		return PressureEpoch{}, fmt.Errorf("create runtime pressure epoch: %w", err)
	}
	return epoch, nil
}

func (s *SQLiteStore) UpdatePressureEpoch(ctx context.Context, epoch PressureEpoch) (PressureEpoch, error) {
	if strings.TrimSpace(epoch.EpochID) == "" {
		return PressureEpoch{}, fmt.Errorf("update pressure epoch: epoch id is required")
	}
	if strings.TrimSpace(epoch.Status) == "" {
		return PressureEpoch{}, fmt.Errorf("update pressure epoch: status is required")
	}
	epoch.UpdatedAt = s.now()
	result, err := s.db.ExecContext(ctx, `
UPDATE runtime_pressure_epochs
SET status = ?, source = ?, cleared_at = ?, updated_at = ?, details_json = ?
WHERE epoch_id = ?`, epoch.Status, epoch.Source, formatOptionalTime(epoch.ClearedAt),
		formatTime(epoch.UpdatedAt), epoch.DetailsJSON, epoch.EpochID)
	if err != nil {
		return PressureEpoch{}, fmt.Errorf("update runtime pressure epoch: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PressureEpoch{}, fmt.Errorf("inspect runtime pressure epoch update: %w", err)
	}
	if affected == 0 {
		return PressureEpoch{}, ErrNotFound
	}
	current, err := s.GetPressureEpoch(ctx, epoch.EpochID)
	if err != nil {
		return PressureEpoch{}, err
	}
	return current, nil
}

func (s *SQLiteStore) GetPressureEpoch(ctx context.Context, epochID string) (PressureEpoch, error) {
	return scanPressureEpoch(s.db.QueryRowContext(ctx, pressureEpochSelectSQL+` WHERE epoch_id = ?`, epochID))
}

func (s *SQLiteStore) ListPressureEpochs(ctx context.Context, limit int) ([]PressureEpoch, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, pressureEpochSelectSQL+` ORDER BY detected_at DESC, epoch_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list runtime pressure epochs: %w", err)
	}
	defer rows.Close()
	return scanPressureEpochs(rows)
}

func (s *SQLiteStore) RecordRecoveryDecision(ctx context.Context, decision RecoveryDecision) (RecoveryDecision, error) {
	if strings.TrimSpace(decision.EpochID) == "" || strings.TrimSpace(decision.Scenario) == "" {
		return RecoveryDecision{}, fmt.Errorf("record recovery decision: epoch id and scenario are required")
	}
	if strings.TrimSpace(decision.State) == "" || strings.TrimSpace(decision.IdempotencyKey) == "" {
		return RecoveryDecision{}, fmt.Errorf("record recovery decision: state and idempotency key are required")
	}
	if decision.Attempt < 0 {
		return RecoveryDecision{}, fmt.Errorf("record recovery decision: attempt must be non-negative")
	}
	if strings.TrimSpace(decision.DecisionID) == "" {
		decision.DecisionID = newID("recovery")
	}
	decision.Variant = InstanceKey{Scenario: decision.Scenario, Variant: decision.Variant}.Normalize().Variant
	if decision.CreatedAt.IsZero() {
		decision.CreatedAt = s.now()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_recovery_decisions (
  decision_id, epoch_id, scenario, variant, state, reason, attempt, cooldown_until,
  idempotency_key, created_at, details_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(idempotency_key) DO NOTHING`, decision.DecisionID, decision.EpochID,
		decision.Scenario, decision.Variant, decision.State, decision.Reason, decision.Attempt,
		formatOptionalTime(decision.CooldownUntil), decision.IdempotencyKey,
		formatTime(decision.CreatedAt), decision.DetailsJSON)
	if err != nil {
		return RecoveryDecision{}, fmt.Errorf("record runtime recovery decision: %w", err)
	}
	return s.recoveryDecisionByIdempotencyKey(ctx, decision.IdempotencyKey)
}

func (s *SQLiteStore) ListRecoveryDecisions(ctx context.Context, filter RecoveryDecisionFilter) ([]RecoveryDecision, error) {
	query := recoveryDecisionSelectSQL
	var clauses []string
	var args []any
	for column, value := range map[string]string{"epoch_id": filter.EpochID, "scenario": filter.Scenario} {
		if strings.TrimSpace(value) != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	if strings.TrimSpace(filter.Variant) != "" {
		clauses = append(clauses, "variant = ?")
		args = append(args, InstanceKey{Scenario: filter.Scenario, Variant: filter.Variant}.Normalize().Variant)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	query += " ORDER BY created_at DESC, decision_id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runtime recovery decisions: %w", err)
	}
	defer rows.Close()
	return scanRecoveryDecisions(rows)
}

func (s *SQLiteStore) recoveryDecisionByIdempotencyKey(ctx context.Context, key string) (RecoveryDecision, error) {
	return scanRecoveryDecision(s.db.QueryRowContext(ctx, recoveryDecisionSelectSQL+` WHERE idempotency_key = ?`, key))
}

const recoveryPolicySelectSQL = `
SELECT scenario, variant, critical, dependency_tier, enabled, retry_budget, opt_out, updated_at
FROM runtime_recovery_policies`

const pressureEpochSelectSQL = `
SELECT epoch_id, status, source, detected_at, cleared_at, updated_at, details_json
FROM runtime_pressure_epochs`

const recoveryDecisionSelectSQL = `
SELECT decision_id, epoch_id, scenario, variant, state, reason, attempt, cooldown_until,
       idempotency_key, created_at, details_json
FROM runtime_recovery_decisions`

func scanRecoveryPolicy(row scanner) (RecoveryPolicy, error) {
	var policy RecoveryPolicy
	var updatedAt string
	if err := row.Scan(&policy.Scenario, &policy.Variant, &policy.Critical, &policy.DependencyTier,
		&policy.Enabled, &policy.RetryBudget, &policy.OptOut, &updatedAt); err != nil {
		return RecoveryPolicy{}, mapRowErr(err)
	}
	parsed, err := parseRequiredTime(updatedAt)
	if err != nil {
		return RecoveryPolicy{}, fmt.Errorf("parse recovery policy updated_at: %w", err)
	}
	policy.UpdatedAt = parsed
	return policy, nil
}

func scanRecoveryPolicies(rows *sql.Rows) ([]RecoveryPolicy, error) {
	var policies []RecoveryPolicy
	for rows.Next() {
		policy, err := scanRecoveryPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func scanPressureEpoch(row scanner) (PressureEpoch, error) {
	var epoch PressureEpoch
	var detectedAt, updatedAt string
	var clearedAt sql.NullString
	if err := row.Scan(&epoch.EpochID, &epoch.Status, &epoch.Source, &detectedAt, &clearedAt, &updatedAt, &epoch.DetailsJSON); err != nil {
		return PressureEpoch{}, mapRowErr(err)
	}
	var err error
	if epoch.DetectedAt, err = parseRequiredTime(detectedAt); err != nil {
		return PressureEpoch{}, fmt.Errorf("parse pressure epoch detected_at: %w", err)
	}
	if epoch.ClearedAt, err = parseOptionalTime(clearedAt); err != nil {
		return PressureEpoch{}, fmt.Errorf("parse pressure epoch cleared_at: %w", err)
	}
	if epoch.UpdatedAt, err = parseRequiredTime(updatedAt); err != nil {
		return PressureEpoch{}, fmt.Errorf("parse pressure epoch updated_at: %w", err)
	}
	return epoch, nil
}

func scanPressureEpochs(rows *sql.Rows) ([]PressureEpoch, error) {
	var epochs []PressureEpoch
	for rows.Next() {
		epoch, err := scanPressureEpoch(rows)
		if err != nil {
			return nil, err
		}
		epochs = append(epochs, epoch)
	}
	return epochs, rows.Err()
}

func scanRecoveryDecision(row scanner) (RecoveryDecision, error) {
	var decision RecoveryDecision
	var cooldown sql.NullString
	var createdAt string
	if err := row.Scan(&decision.DecisionID, &decision.EpochID, &decision.Scenario, &decision.Variant,
		&decision.State, &decision.Reason, &decision.Attempt, &cooldown, &decision.IdempotencyKey,
		&createdAt, &decision.DetailsJSON); err != nil {
		return RecoveryDecision{}, mapRowErr(err)
	}
	var err error
	if decision.CooldownUntil, err = parseOptionalTime(cooldown); err != nil {
		return RecoveryDecision{}, fmt.Errorf("parse recovery decision cooldown_until: %w", err)
	}
	if decision.CreatedAt, err = parseRequiredTime(createdAt); err != nil {
		return RecoveryDecision{}, fmt.Errorf("parse recovery decision created_at: %w", err)
	}
	return decision, nil
}

func scanRecoveryDecisions(rows *sql.Rows) ([]RecoveryDecision, error) {
	var decisions []RecoveryDecision
	for rows.Next() {
		decision, err := scanRecoveryDecision(rows)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}
	return decisions, rows.Err()
}
