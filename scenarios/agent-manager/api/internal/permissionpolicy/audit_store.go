package permissionpolicy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agent-manager/internal/sqlcompat"
)

// SQLiteAuditStore is the production AuditStore implementation. It retains
// last-result metadata but deliberately never reads or writes native files.
type SQLiteAuditStore struct {
	db sqlcompat.DB
}

func NewSQLiteAuditStore(db sqlcompat.DB) *SQLiteAuditStore {
	return &SQLiteAuditStore{db: db}
}

func (s *SQLiteAuditStore) RecordReconcile(ctx context.Context, result ReconcileResult) error {
	if s == nil || s.db == nil {
		return nil
	}
	resources, err := json.Marshal(result.Resources)
	if err != nil {
		return fmt.Errorf("serialize permission reconcile resources: %w", err)
	}
	missing, err := json.Marshal(result.MissingHardEnforcementRuleIDs)
	if err != nil {
		return fmt.Errorf("serialize missing hard-enforcement rules: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO permission_policy_reconcile_audit (
			catalog_digest, started_at, finished_at, explicitly_authorized, success,
			hard_enforcement_satisfied, missing_hard_enforcement_rule_ids, resource_results
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, result.CatalogDigest, formatTimestamp(result.StartedAt), formatTimestamp(result.FinishedAt), result.ExplicitlyAuthorized,
		result.Success, result.HardEnforcementSatisfied, string(missing), string(resources))
	if err != nil {
		return fmt.Errorf("record permission reconcile audit: %w", err)
	}
	return nil
}

func (s *SQLiteAuditStore) LastReconcile(ctx context.Context) (*ReconcileResult, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT catalog_digest, started_at, finished_at, explicitly_authorized, success,
			hard_enforcement_satisfied, missing_hard_enforcement_rule_ids, resource_results
		FROM permission_policy_reconcile_audit
		ORDER BY id DESC LIMIT 1
	`)
	var (
		result                ReconcileResult
		startedAt, finishedAt string
		missing, resources    string
	)
	if err := row.Scan(&result.CatalogDigest, &startedAt, &finishedAt, &result.ExplicitlyAuthorized, &result.Success,
		&result.HardEnforcementSatisfied, &missing, &resources); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load last permission reconcile audit: %w", err)
	}
	var err error
	if result.StartedAt, err = time.Parse(time.RFC3339Nano, startedAt); err != nil {
		return nil, fmt.Errorf("parse permission reconcile start time: %w", err)
	}
	if result.FinishedAt, err = time.Parse(time.RFC3339Nano, finishedAt); err != nil {
		return nil, fmt.Errorf("parse permission reconcile finish time: %w", err)
	}
	if err := json.Unmarshal([]byte(missing), &result.MissingHardEnforcementRuleIDs); err != nil {
		return nil, fmt.Errorf("parse missing hard-enforcement rules: %w", err)
	}
	if err := json.Unmarshal([]byte(resources), &result.Resources); err != nil {
		return nil, fmt.Errorf("parse permission reconcile resources: %w", err)
	}
	return result.Clone(), nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

var _ AuditStore = (*SQLiteAuditStore)(nil)
