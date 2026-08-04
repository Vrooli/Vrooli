// DOC: docs/guides/managing-policies.md
// DOC: docs/internal/INVARIANTS.md
package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/sqlutil"
	_ "modernc.org/sqlite"
)

const policySchema = `
CREATE TABLE IF NOT EXISTS policy_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  rule_type TEXT NOT NULL CHECK(rule_type IN ('access','rate_limit','circuit_breaker')),
  source_scenario TEXT NOT NULL,
  target_scenario TEXT NOT NULL,
  endpoint_pattern TEXT NOT NULL DEFAULT '',
  effect TEXT NOT NULL DEFAULT '' CHECK(effect IN ('','allow','deny')),
  priority INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1,
  max_requests INTEGER NOT NULL DEFAULT 0,
  window_seconds INTEGER NOT NULL DEFAULT 0,
  burst_allowance INTEGER NOT NULL DEFAULT 0,
  failure_threshold INTEGER NOT NULL DEFAULT 0,
  cooldown_seconds INTEGER NOT NULL DEFAULT 0,
  success_threshold INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
);
CREATE INDEX IF NOT EXISTS idx_policy_rules_type ON policy_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_policy_rules_source ON policy_rules(source_scenario);
CREATE INDEX IF NOT EXISTS idx_policy_rules_target ON policy_rules(target_scenario);

CREATE TABLE IF NOT EXISTS receipt_projection_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
	policy_id TEXT NOT NULL DEFAULT '',
  source_scenario TEXT NOT NULL,
  target_scenario TEXT NOT NULL,
  operation_pattern TEXT NOT NULL,
	protocol TEXT NOT NULL DEFAULT 'connect',
	event_type TEXT NOT NULL DEFAULT 'vrooli.events.receipt.v1',
	response_type TEXT NOT NULL DEFAULT '',
  response_fields_json TEXT NOT NULL DEFAULT '[]',
	read_principals_json TEXT NOT NULL DEFAULT '[]',
  redact_fields_json TEXT NOT NULL DEFAULT '[]',
  max_bytes INTEGER NOT NULL DEFAULT 65536,
  sample_per_ten_k INTEGER NOT NULL DEFAULT 10000,
  retention_days INTEGER NOT NULL DEFAULT 30,
  priority INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
);
CREATE INDEX IF NOT EXISTS idx_receipt_projection_match ON receipt_projection_rules(source_scenario, target_scenario, enabled, priority DESC, id ASC);

CREATE TABLE IF NOT EXISTS policy_violations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
  source_scenario TEXT NOT NULL,
  target_scenario TEXT NOT NULL,
  endpoint TEXT NOT NULL DEFAULT '',
  rule_id INTEGER NOT NULL,
  rule_type TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_violations_source ON policy_violations(source_scenario);
CREATE INDEX IF NOT EXISTS idx_violations_target ON policy_violations(target_scenario);
CREATE INDEX IF NOT EXISTS idx_violations_timestamp ON policy_violations(timestamp);

CREATE TABLE IF NOT EXISTS circuit_breaker_overrides (
  rule_id INTEGER PRIMARY KEY,
  state TEXT NOT NULL CHECK(state IN ('open','closed','half_open')),
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
);
`

// SQLiteStore implements policy.Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or reuses a SQLite database for policy storage.
// It accepts an existing *sql.DB so the policy tables can share the same
// database file as the event store.
func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	if _, err := db.Exec(policySchema); err != nil {
		return nil, fmt.Errorf("apply policy schema: %w", err)
	}
	// Existing greenfield databases may have been created during an earlier
	// development build. Make the hard-cut policy columns durable without a
	// compatibility read path; duplicate-column errors are expected on new DBs.
	for _, statement := range []string{
		`ALTER TABLE receipt_projection_rules ADD COLUMN policy_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipt_projection_rules ADD COLUMN protocol TEXT NOT NULL DEFAULT 'connect'`,
		`ALTER TABLE receipt_projection_rules ADD COLUMN event_type TEXT NOT NULL DEFAULT 'vrooli.events.receipt.v1'`,
		`ALTER TABLE receipt_projection_rules ADD COLUMN response_type TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE receipt_projection_rules ADD COLUMN read_principals_json TEXT NOT NULL DEFAULT '[]'`,
	} {
		if _, err := db.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, fmt.Errorf("migrate receipt capture policy: %w", err)
		}
	}
	// Vrooli Events has no supported legacy consumers. Rules without a stable
	// capture-policy identity are from the removed receipt-projection contract
	// and must never be surfaced or matched after the hard cut.
	if _, err := db.Exec(`DELETE FROM receipt_projection_rules WHERE policy_id = ''`); err != nil {
		return nil, fmt.Errorf("purge legacy receipt projection rules: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) CreateRule(ctx context.Context, r Rule) (int64, error) {
	if r.RuleType == RuleTypeAccess {
		r.Priority = ComputeSpecificity(r.SourceScenario, r.TargetScenario, r.EndpointPattern)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO policy_rules (rule_type, source_scenario, target_scenario, endpoint_pattern, effect, priority, enabled,
		  max_requests, window_seconds, burst_allowance, failure_threshold, cooldown_seconds, success_threshold)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(r.RuleType), r.SourceScenario, r.TargetScenario, r.EndpointPattern,
		string(r.Effect), r.Priority, sqlutil.BoolToInt(r.Enabled),
		r.MaxRequests, r.WindowSeconds, r.BurstAllowance,
		r.FailureThreshold, r.CooldownSeconds, r.SuccessThreshold)
	if err != nil {
		return 0, fmt.Errorf("insert policy rule: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetRule(ctx context.Context, id int64) (Rule, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, rule_type, source_scenario, target_scenario, endpoint_pattern, effect, priority, enabled,
		  max_requests, window_seconds, burst_allowance, failure_threshold, cooldown_seconds, success_threshold,
		  created_at, updated_at
		 FROM policy_rules WHERE id = ?`, id)
	return scanRuleFrom(row)
}

func (s *SQLiteStore) ListRules(ctx context.Context, f ListFilters) ([]Rule, error) {
	var clauses []string
	var args []any

	if f.RuleType != "" {
		clauses = append(clauses, "rule_type = ?")
		args = append(args, string(f.RuleType))
	}
	if f.Source != "" {
		clauses = append(clauses, "source_scenario = ?")
		args = append(args, f.Source)
	}
	if f.Target != "" {
		clauses = append(clauses, "target_scenario = ?")
		args = append(args, f.Target)
	}
	if f.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, sqlutil.BoolToInt(*f.Enabled))
	}

	query := `SELECT id, rule_type, source_scenario, target_scenario, endpoint_pattern, effect, priority, enabled,
	  max_requests, window_seconds, burst_allowance, failure_threshold, cooldown_seconds, success_threshold,
	  created_at, updated_at FROM policy_rules`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY priority DESC, id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list policy rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		r, err := scanRuleFrom(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *SQLiteStore) UpdateRule(ctx context.Context, r Rule) error {
	if r.RuleType == RuleTypeAccess {
		r.Priority = ComputeSpecificity(r.SourceScenario, r.TargetScenario, r.EndpointPattern)
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE policy_rules SET
		  rule_type=?, source_scenario=?, target_scenario=?, endpoint_pattern=?, effect=?, priority=?, enabled=?,
		  max_requests=?, window_seconds=?, burst_allowance=?, failure_threshold=?, cooldown_seconds=?, success_threshold=?,
		  updated_at=strftime('%Y-%m-%dT%H:%M:%f','now')
		 WHERE id=?`,
		string(r.RuleType), r.SourceScenario, r.TargetScenario, r.EndpointPattern,
		string(r.Effect), r.Priority, sqlutil.BoolToInt(r.Enabled),
		r.MaxRequests, r.WindowSeconds, r.BurstAllowance,
		r.FailureThreshold, r.CooldownSeconds, r.SuccessThreshold, r.ID)
	if err != nil {
		return fmt.Errorf("update policy rule: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteRule(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM policy_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete policy rule: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CreateReceiptProjection(ctx context.Context, r ReceiptProjectionRule) (int64, error) {
	fields, err := json.Marshal(r.ResponseFields)
	if err != nil {
		return 0, fmt.Errorf("marshal response fields: %w", err)
	}
	redactions, err := json.Marshal(r.RedactFields)
	if err != nil {
		return 0, fmt.Errorf("marshal redact fields: %w", err)
	}
	principals, err := json.Marshal(r.ReadPrincipals)
	if err != nil {
		return 0, fmt.Errorf("marshal read principals: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO receipt_projection_rules
		(policy_id, source_scenario, target_scenario, operation_pattern, protocol, event_type, response_type, response_fields_json, read_principals_json, redact_fields_json, max_bytes, sample_per_ten_k, retention_days, priority, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, r.PolicyID, r.SourceScenario, r.TargetScenario, r.OperationPattern, r.Protocol, r.EventType, r.ResponseType, string(fields), string(principals), string(redactions), r.MaxBytes, r.SamplePerTenK, r.RetentionDays, r.Priority, sqlutil.BoolToInt(r.Enabled))
	if err != nil {
		return 0, fmt.Errorf("insert receipt projection rule: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetReceiptProjection(ctx context.Context, id int64) (ReceiptProjectionRule, error) {
	return scanReceiptProjection(s.db.QueryRowContext(ctx, receiptProjectionSelect+` WHERE id = ?`, id))
}

func (s *SQLiteStore) ListReceiptProjections(ctx context.Context, f ReceiptProjectionFilters) ([]ReceiptProjectionRule, error) {
	clauses, args := []string{}, []any{}
	if f.Source != "" {
		clauses = append(clauses, "source_scenario = ?")
		args = append(args, f.Source)
	}
	if f.Target != "" {
		clauses = append(clauses, "target_scenario = ?")
		args = append(args, f.Target)
	}
	if f.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, sqlutil.BoolToInt(*f.Enabled))
	}
	query := receiptProjectionSelect
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY priority DESC, id ASC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list receipt projection rules: %w", err)
	}
	defer rows.Close()
	var result []ReceiptProjectionRule
	for rows.Next() {
		r, err := scanReceiptProjection(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) UpdateReceiptProjection(ctx context.Context, r ReceiptProjectionRule) error {
	fields, err := json.Marshal(r.ResponseFields)
	if err != nil {
		return fmt.Errorf("marshal response fields: %w", err)
	}
	redactions, err := json.Marshal(r.RedactFields)
	if err != nil {
		return fmt.Errorf("marshal redact fields: %w", err)
	}
	principals, err := json.Marshal(r.ReadPrincipals)
	if err != nil {
		return fmt.Errorf("marshal read principals: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE receipt_projection_rules SET policy_id=?, source_scenario=?, target_scenario=?, operation_pattern=?, protocol=?, event_type=?, response_type=?, response_fields_json=?, read_principals_json=?, redact_fields_json=?, max_bytes=?, sample_per_ten_k=?, retention_days=?, priority=?, enabled=?, updated_at=strftime('%Y-%m-%dT%H:%M:%f','now') WHERE id=?`, r.PolicyID, r.SourceScenario, r.TargetScenario, r.OperationPattern, r.Protocol, r.EventType, r.ResponseType, string(fields), string(principals), string(redactions), r.MaxBytes, r.SamplePerTenK, r.RetentionDays, r.Priority, sqlutil.BoolToInt(r.Enabled), r.ID)
	if err != nil {
		return fmt.Errorf("update receipt projection rule: %w", err)
	}
	return nil
}

// ReconcileReceiptProjections is intentionally one transaction: a capture
// declaration is useful only when its complete policy set is visible.
func (s *SQLiteStore) ReconcileReceiptProjections(ctx context.Context, rules []ReceiptProjectionRule) (ReceiptProjectionReconcileResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReceiptProjectionReconcileResult{}, fmt.Errorf("begin receipt projection reconcile: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, receiptProjectionSelect+" ORDER BY priority DESC, id ASC")
	if err != nil {
		return ReceiptProjectionReconcileResult{}, fmt.Errorf("list receipt projection rules: %w", err)
	}
	byPolicyID := map[string]ReceiptProjectionRule{}
	for rows.Next() {
		rule, scanErr := scanReceiptProjection(rows)
		if scanErr != nil {
			rows.Close()
			return ReceiptProjectionReconcileResult{}, scanErr
		}
		byPolicyID[rule.PolicyID] = rule
	}
	if err := rows.Close(); err != nil {
		return ReceiptProjectionReconcileResult{}, err
	}
	if err := rows.Err(); err != nil {
		return ReceiptProjectionReconcileResult{}, err
	}
	result := ReceiptProjectionReconcileResult{}
	for _, rule := range rules {
		fields, err := json.Marshal(rule.ResponseFields)
		if err != nil {
			return ReceiptProjectionReconcileResult{}, fmt.Errorf("marshal response fields: %w", err)
		}
		redactions, err := json.Marshal(rule.RedactFields)
		if err != nil {
			return ReceiptProjectionReconcileResult{}, fmt.Errorf("marshal redact fields: %w", err)
		}
		principals, err := json.Marshal(rule.ReadPrincipals)
		if err != nil {
			return ReceiptProjectionReconcileResult{}, fmt.Errorf("marshal read principals: %w", err)
		}
		if existing, ok := byPolicyID[rule.PolicyID]; ok {
			if _, err := tx.ExecContext(ctx, `UPDATE receipt_projection_rules SET policy_id=?, source_scenario=?, target_scenario=?, operation_pattern=?, protocol=?, event_type=?, response_type=?, response_fields_json=?, read_principals_json=?, redact_fields_json=?, max_bytes=?, sample_per_ten_k=?, retention_days=?, priority=?, enabled=?, updated_at=strftime('%Y-%m-%dT%H:%M:%f','now') WHERE id=?`, rule.PolicyID, rule.SourceScenario, rule.TargetScenario, rule.OperationPattern, rule.Protocol, rule.EventType, rule.ResponseType, string(fields), string(principals), string(redactions), rule.MaxBytes, rule.SamplePerTenK, rule.RetentionDays, rule.Priority, sqlutil.BoolToInt(rule.Enabled), existing.ID); err != nil {
				return ReceiptProjectionReconcileResult{}, fmt.Errorf("update receipt projection rule: %w", err)
			}
			result.Updated++
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO receipt_projection_rules (policy_id, source_scenario, target_scenario, operation_pattern, protocol, event_type, response_type, response_fields_json, read_principals_json, redact_fields_json, max_bytes, sample_per_ten_k, retention_days, priority, enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, rule.PolicyID, rule.SourceScenario, rule.TargetScenario, rule.OperationPattern, rule.Protocol, rule.EventType, rule.ResponseType, string(fields), string(principals), string(redactions), rule.MaxBytes, rule.SamplePerTenK, rule.RetentionDays, rule.Priority, sqlutil.BoolToInt(rule.Enabled)); err != nil {
			return ReceiptProjectionReconcileResult{}, fmt.Errorf("insert receipt projection rule: %w", err)
		}
		result.Created++
	}
	if err := tx.Commit(); err != nil {
		return ReceiptProjectionReconcileResult{}, fmt.Errorf("commit receipt projection reconcile: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) DeleteReceiptProjection(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM receipt_projection_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete receipt projection rule: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteReceiptProjectionByPolicyID(ctx context.Context, policyID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM receipt_projection_rules WHERE policy_id = ?`, policyID)
	if err != nil {
		return fmt.Errorf("delete receipt projection rule by policy id: %w", err)
	}
	return nil
}

func (s *SQLiteStore) MatchReceiptProjection(ctx context.Context, source, target, operation string) (*ReceiptProjectionRule, error) {
	enabled := true
	rules, err := s.ListReceiptProjections(ctx, ReceiptProjectionFilters{Enabled: &enabled})
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if patternMatches(rule.SourceScenario, source) && patternMatches(rule.TargetScenario, target) && patternMatches(rule.OperationPattern, operation) {
			return &rule, nil
		}
	}
	return nil, nil
}

const receiptProjectionSelect = `SELECT id, policy_id, source_scenario, target_scenario, operation_pattern, protocol, event_type, response_type, response_fields_json, read_principals_json, redact_fields_json, max_bytes, sample_per_ten_k, retention_days, priority, enabled, created_at, updated_at FROM receipt_projection_rules`

type receiptProjectionScanner interface{ Scan(...any) error }

func scanReceiptProjection(row receiptProjectionScanner) (ReceiptProjectionRule, error) {
	var r ReceiptProjectionRule
	var responseFields, readPrincipals, redactFields, createdAt, updatedAt string
	var enabled int
	if err := row.Scan(&r.ID, &r.PolicyID, &r.SourceScenario, &r.TargetScenario, &r.OperationPattern, &r.Protocol, &r.EventType, &r.ResponseType, &responseFields, &readPrincipals, &redactFields, &r.MaxBytes, &r.SamplePerTenK, &r.RetentionDays, &r.Priority, &enabled, &createdAt, &updatedAt); err != nil {
		return r, err
	}
	if err := json.Unmarshal([]byte(responseFields), &r.ResponseFields); err != nil {
		return r, fmt.Errorf("decode response fields: %w", err)
	}
	if err := json.Unmarshal([]byte(readPrincipals), &r.ReadPrincipals); err != nil {
		return r, fmt.Errorf("decode read principals: %w", err)
	}
	if err := json.Unmarshal([]byte(redactFields), &r.RedactFields); err != nil {
		return r, fmt.Errorf("decode redact fields: %w", err)
	}
	r.Enabled = enabled != 0
	var err error
	if r.CreatedAt, err = time.Parse(sqlutil.TimestampFormat, createdAt); err != nil {
		return r, err
	}
	if r.UpdatedAt, err = time.Parse(sqlutil.TimestampFormat, updatedAt); err != nil {
		return r, err
	}
	return r, nil
}

func patternMatches(pattern, value string) bool {
	if pattern == "" || pattern == "*" || pattern == "**" {
		return true
	}
	ok, err := path.Match(pattern, value)
	return err == nil && ok
}

func (s *SQLiteStore) LogViolation(ctx context.Context, v Violation) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO policy_violations (source_scenario, target_scenario, endpoint, rule_id, rule_type, reason)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		v.SourceScenario, v.TargetScenario, v.Endpoint, v.RuleID, string(v.RuleType), v.Reason)
	return err
}

func (s *SQLiteStore) ListViolations(ctx context.Context, f ViolationFilters) ([]Violation, error) {
	var clauses []string
	var args []any

	if f.Source != "" {
		clauses = append(clauses, "source_scenario = ?")
		args = append(args, f.Source)
	}
	if f.Target != "" {
		clauses = append(clauses, "target_scenario = ?")
		args = append(args, f.Target)
	}
	if f.RuleType != "" {
		clauses = append(clauses, "rule_type = ?")
		args = append(args, string(f.RuleType))
	}
	if f.Since != "" {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, f.Since)
	}
	if f.Until != "" {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, f.Until)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, timestamp, source_scenario, target_scenario, endpoint, rule_id, rule_type, reason FROM policy_violations`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list violations: %w", err)
	}
	defer rows.Close()

	var violations []Violation
	for rows.Next() {
		var v Violation
		if err := rows.Scan(&v.ID, &v.Timestamp, &v.SourceScenario, &v.TargetScenario,
			&v.Endpoint, &v.RuleID, &v.RuleType, &v.Reason); err != nil {
			return nil, err
		}
		violations = append(violations, v)
	}
	return violations, rows.Err()
}

func (s *SQLiteStore) SetCircuitBreakerOverride(ctx context.Context, ruleID int64, state CircuitState, ttlSeconds int) error {
	expiresAt := time.Now().Add(time.Duration(ttlSeconds) * time.Second).UTC().Format(sqlutil.TimestampFormat)
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO circuit_breaker_overrides (rule_id, state, expires_at)
		 VALUES (?, ?, ?)`,
		ruleID, string(state), expiresAt)
	if err != nil {
		return fmt.Errorf("set circuit breaker override: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetCircuitBreakerOverride(ctx context.Context, ruleID int64) (*CircuitBreakerOverride, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT rule_id, state, expires_at, created_at
		 FROM circuit_breaker_overrides
		 WHERE rule_id = ? AND expires_at > strftime('%Y-%m-%dT%H:%M:%f','now')`, ruleID)

	var o CircuitBreakerOverride
	var state, expiresAt, createdAt string
	err := row.Scan(&o.RuleID, &state, &expiresAt, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get circuit breaker override: %w", err)
	}
	o.State = CircuitState(state)
	o.ExpiresAt = sqlutil.ParseTime(expiresAt)
	o.CreatedAt = sqlutil.ParseTime(createdAt)
	return &o, nil
}

func (s *SQLiteStore) Close() error {
	return nil // DB lifecycle managed externally
}

// ruleScanner abstracts *sql.Row and *sql.Rows so rule scanning logic is shared.
type ruleScanner interface {
	Scan(dest ...any) error
}

// scanRuleFrom scans a single rule from any scanner (*sql.Row or *sql.Rows).
func scanRuleFrom(sc ruleScanner) (Rule, error) {
	var r Rule
	var ruleType, effect, createdAt, updatedAt string
	var enabled int
	err := sc.Scan(&r.ID, &ruleType, &r.SourceScenario, &r.TargetScenario,
		&r.EndpointPattern, &effect, &r.Priority, &enabled,
		&r.MaxRequests, &r.WindowSeconds, &r.BurstAllowance,
		&r.FailureThreshold, &r.CooldownSeconds, &r.SuccessThreshold,
		&createdAt, &updatedAt)
	if err != nil {
		return r, err
	}
	r.RuleType = RuleType(ruleType)
	r.Effect = Effect(effect)
	r.Enabled = enabled != 0
	r.CreatedAt = sqlutil.ParseTime(createdAt)
	r.UpdatedAt = sqlutil.ParseTime(updatedAt)
	return r, nil
}
