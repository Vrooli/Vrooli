package findings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
)

type SQLiteRepository struct{ db sqlcompat.DB }

type findingRow struct {
	Finding
	CreatedAtRaw string `db:"created_at"`
}

func NewSQLiteRepository(db sqlcompat.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, finding *Finding) error {
	if finding == nil {
		return fmt.Errorf("finding is required")
	}
	if finding.ID == uuid.Nil {
		finding.ID = uuid.New()
	}
	if finding.CreatedAt.IsZero() {
		finding.CreatedAt = time.Now().UTC()
	}
	if finding.Fingerprint == "" {
		finding.Fingerprint = Fingerprint(finding.Recommendation, finding.TargetPath)
	}
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO run_findings (id, run_id, investigation_run_id, category, severity, recommendation_text, evidence, target_path, fingerprint, operator_decision, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, finding.ID.String(), finding.RunID.String(), finding.InvestigationRunID.String(), finding.Category, finding.Severity, finding.Recommendation, finding.Evidence, finding.TargetPath, finding.Fingerprint, finding.Decision, finding.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) List(ctx context.Context, filter Filter) ([]Finding, error) {
	query, args := `SELECT id, run_id, investigation_run_id, category, severity, recommendation_text, evidence, target_path, fingerprint, operator_decision, created_at, COUNT(*) OVER (PARTITION BY fingerprint) AS occurrences FROM run_findings`, []any{}
	var where []string
	if filter.RunID != nil {
		where, args = append(where, "run_id = ?"), append(args, filter.RunID.String())
	}
	if filter.Since != nil {
		where, args = append(where, "created_at >= ?"), append(args, filter.Since.UTC().Format(time.RFC3339Nano))
	}
	if filter.Fingerprint != "" {
		where, args = append(where, "fingerprint = ?"), append(args, filter.Fingerprint)
	}
	if filter.Severity != "" {
		where, args = append(where, "severity = ?"), append(args, filter.Severity)
	}
	if filter.Decision != "" {
		where, args = append(where, "operator_decision = ?"), append(args, filter.Decision)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY occurrences DESC, created_at DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	var rows []findingRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	items := make([]Finding, 0, len(rows))
	for _, row := range rows {
		item := row.Finding
		parsed, err := time.Parse(time.RFC3339Nano, row.CreatedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse finding timestamp: %w", err)
		}
		item.CreatedAt = parsed
		items = append(items, item)
	}
	return items, nil
}

func (r *SQLiteRepository) SetDecision(ctx context.Context, investigationRunID uuid.UUID, decision string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE run_findings SET operator_decision = ? WHERE investigation_run_id = ?`, strings.TrimSpace(decision), investigationRunID.String())
	return err
}

func (r *SQLiteRepository) RecurrenceCount(ctx context.Context, fingerprint string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_findings WHERE fingerprint = ?`, fingerprint).Scan(&count)
	return count, err
}
