package findings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/sqlcompat"
	"agent-manager/internal/usageattribution"

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
	if finding.QualitySignal == "" {
		finding.QualitySignal = qualitySignal(finding)
	}
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO run_findings (id, run_id, investigation_run_id, category, severity, recommendation_text, evidence, target_path, fingerprint, friction_topic, cites_resolved_commands, cites_real_outcome, cites_attributed_owner, quality_signal, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, finding.ID.String(), finding.RunID.String(), finding.InvestigationRunID.String(), finding.Category, finding.Severity, finding.Recommendation, finding.Evidence, finding.TargetPath, finding.Fingerprint, finding.FrictionTopic, finding.CitesResolvedCommands, finding.CitesRealOutcome, finding.CitesAttributedOwner, finding.QualitySignal, finding.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) List(ctx context.Context, filter Filter) ([]Finding, error) {
	query, args := `SELECT id, run_id, investigation_run_id, category, severity, recommendation_text, evidence, target_path, fingerprint, friction_topic, cites_resolved_commands, cites_real_outcome, cites_attributed_owner, quality_signal, created_at, COUNT(*) OVER (PARTITION BY fingerprint) AS occurrences FROM run_findings`, []any{}
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

func qualitySignal(finding *Finding) string {
	if finding == nil || strings.TrimSpace(finding.Evidence) == "" {
		return "unavailable"
	}
	flags := []bool{finding.CitesResolvedCommands, finding.CitesRealOutcome, finding.CitesAttributedOwner}
	count := 0
	for _, flag := range flags {
		if flag {
			count++
		}
	}
	if count == len(flags) {
		return "complete"
	}
	if count > 0 {
		return "partial"
	}
	return "unavailable"
}

func (r *SQLiteRepository) SetTopic(ctx context.Context, id uuid.UUID, topic string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE run_findings SET friction_topic=? WHERE id=?`, strings.TrimSpace(topic), id.String())
	return err
}

// FrictionStatusStore keeps friction_problem_status.
type FrictionStatusStore struct{ db sqlcompat.DB }

func NewFrictionStatusStore(db sqlcompat.DB) *FrictionStatusStore {
	return &FrictionStatusStore{db: db}
}

type statusRow struct {
	Fingerprint string `db:"fingerprint"`
	Status      string `db:"status"`
	BugID       string `db:"bug_id"`
	UpdatedAt   string `db:"updated_at"`
}

// FrictionStatuses returns every recorded status by fingerprint.
func (s *FrictionStatusStore) FrictionStatuses(ctx context.Context) (map[string]usageattribution.ProblemStatus, error) {
	var rows []statusRow
	if err := s.db.SelectContext(ctx, &rows, `SELECT fingerprint, status, bug_id, updated_at FROM friction_problem_status`); err != nil {
		return nil, err
	}
	out := make(map[string]usageattribution.ProblemStatus, len(rows))
	for _, row := range rows {
		at, _ := time.Parse(time.RFC3339Nano, row.UpdatedAt)
		out[row.Fingerprint] = usageattribution.ProblemStatus{Fingerprint: row.Fingerprint, Status: row.Status, BugID: row.BugID, At: at}
	}
	return out, nil
}

// SetFrictionStatus records what the operator did with a problem.
func (s *FrictionStatusStore) SetFrictionStatus(ctx context.Context, status usageattribution.ProblemStatus) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO friction_problem_status (fingerprint, status, bug_id, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(fingerprint) DO UPDATE SET status = excluded.status, bug_id = excluded.bug_id, updated_at = excluded.updated_at`,
		status.Fingerprint, status.Status, status.BugID, status.At.UTC().Format(time.RFC3339Nano))
	return err
}
