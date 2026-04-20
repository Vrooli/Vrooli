// DOC: docs/guides/managing-policies.md
// DOC: docs/internal/INVARIANTS.md
package policy

import (
	"context"
	"database/sql"
	"fmt"
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
