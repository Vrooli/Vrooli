package dochealing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/sqlitetime"

	"knowledge-observatory/internal/services/dochealing"
)

// SQLite is the dochealing domain's job store. It satisfies
// dochealing.JobStore.
type SQLite struct {
	DB *apidb.RoutedDB
}

var _ dochealing.JobStore = (*SQLite)(nil)

// NewSQLite returns a job store backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

func (s *SQLite) CreateJob(ctx context.Context, req dochealing.HealRequest, healthBefore *float64) (string, error) {
	if s == nil || s.DB == nil {
		return "", fmt.Errorf("job store not configured")
	}
	issuesJSON, err := json.Marshal(req.Issues)
	if err != nil {
		return "", fmt.Errorf("marshal issues: %w", err)
	}
	id := uuid.NewString()
	_, err = s.DB.ExecContext(ctx, `
INSERT INTO doc_heal_jobs
  (id, scenario_name, issues, auto_approve, dry_run, health_before, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, 'pending', CURRENT_TIMESTAMP)
`, id, strings.TrimSpace(req.ScenarioName), string(issuesJSON), req.AutoApprove, req.DryRun, healthBefore)
	if err != nil {
		return "", fmt.Errorf("create doc heal job failed: %w", err)
	}
	return id, nil
}

func (s *SQLite) GetJob(ctx context.Context, jobID string) (dochealing.HealJob, bool, error) {
	if s == nil || s.DB == nil {
		return dochealing.HealJob{}, false, fmt.Errorf("job store not configured")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return dochealing.HealJob{}, false, fmt.Errorf("job_id is required")
	}

	var (
		id, scenario, status, progress string
		agentRunID, errText, diffJSON  string
		healthBefore, healthAfter      sql.NullFloat64
		autoApprove, dryRun            bool
		createdAt                      time.Time
		startedAt, completedAt         sql.NullTime
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT id, scenario_name, status, COALESCE(progress, ''), COALESCE(diff, '{}'),
       COALESCE(agent_run_id, ''), COALESCE(error, ''), health_before, health_after,
       auto_approve, dry_run, created_at, started_at, completed_at
FROM doc_heal_jobs
WHERE id = ?
`, jobID).Scan(&id, &scenario, &status, &progress, &diffJSON, &agentRunID, &errText,
		&healthBefore, &healthAfter, &autoApprove, &dryRun, &createdAt, &startedAt, &completedAt)
	if err == sql.ErrNoRows {
		return dochealing.HealJob{}, false, nil
	}
	if err != nil {
		return dochealing.HealJob{}, false, fmt.Errorf("get doc heal job failed: %w", err)
	}

	var diff *dochealing.DiffPreview
	if trimmed := strings.TrimSpace(diffJSON); trimmed != "" && trimmed != "{}" {
		var parsed dochealing.DiffPreview
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			diff = &parsed
		}
	}

	job := dochealing.HealJob{
		JobID:        id,
		ScenarioName: scenario,
		Status:       status,
		Progress:     strings.TrimSpace(progress),
		Diff:         diff,
		Error:        strings.TrimSpace(errText),
		AgentRunID:   strings.TrimSpace(agentRunID),
		AutoApprove:  autoApprove,
		DryRun:       dryRun,
	}
	if healthBefore.Valid {
		v := healthBefore.Float64
		job.HealthBefore = &v
	}
	if healthAfter.Valid {
		v := healthAfter.Float64
		job.HealthAfter = &v
	}
	if startedAt.Valid {
		t := startedAt.Time.UTC()
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time.UTC()
		job.CompletedAt = &t
	}
	return job, true, nil
}

func (s *SQLite) MarkRunning(ctx context.Context, jobID, runID string, startedAt time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE doc_heal_jobs
SET status = 'running', agent_run_id = NULLIF(?, ''), started_at = ?
WHERE id = ?
`, strings.TrimSpace(runID), sqlitetime.Format(startedAt), jobID)
	if err != nil {
		return fmt.Errorf("mark running failed: %w", err)
	}
	return nil
}

func (s *SQLite) UpdateProgress(ctx context.Context, jobID, progress string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE doc_heal_jobs SET progress = NULLIF(?, '') WHERE id = ?`,
		strings.TrimSpace(progress), jobID)
	if err != nil {
		return fmt.Errorf("update progress failed: %w", err)
	}
	return nil
}

func (s *SQLite) UpdateReview(ctx context.Context, jobID string, diff *dochealing.DiffPreview, healthAfter *float64, status string, completedAt time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	blob := []byte("{}")
	if diff != nil {
		var err error
		blob, err = json.Marshal(diff)
		if err != nil {
			return fmt.Errorf("marshal diff failed: %w", err)
		}
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE doc_heal_jobs
SET status = ?, diff = ?, health_after = ?, completed_at = ?, error = NULL
WHERE id = ?
`, status, string(blob), healthAfter, sqlitetime.Format(completedAt), jobID)
	if err != nil {
		return fmt.Errorf("update review failed: %w", err)
	}
	return nil
}

func (s *SQLite) UpdateError(ctx context.Context, jobID, message string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE doc_heal_jobs SET error = NULLIF(?, '') WHERE id = ?`,
		strings.TrimSpace(message), jobID)
	if err != nil {
		return fmt.Errorf("update error failed: %w", err)
	}
	return nil
}

func (s *SQLite) MarkApproved(ctx context.Context, jobID, actor string, approvedAt time.Time, healthAfter *float64) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE doc_heal_jobs
SET status = 'approved', approved_by = NULLIF(?, ''), approved_at = ?, health_after = ?, error = NULL
WHERE id = ?
`, strings.TrimSpace(actor), sqlitetime.Format(approvedAt), healthAfter, jobID)
	if err != nil {
		return fmt.Errorf("mark approved failed: %w", err)
	}
	return nil
}

func (s *SQLite) MarkRejected(ctx context.Context, jobID, actor, reason string, completedAt time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE doc_heal_jobs
SET status = 'rejected', rejected_by = NULLIF(?, ''), reject_reason = NULLIF(?, ''),
    rejected_at = ?, error = NULL
WHERE id = ?
`, strings.TrimSpace(actor), strings.TrimSpace(reason), sqlitetime.Format(completedAt), jobID)
	if err != nil {
		return fmt.Errorf("mark rejected failed: %w", err)
	}
	return nil
}

func (s *SQLite) FailJob(ctx context.Context, jobID, message string, completedAt time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE doc_heal_jobs
SET status = 'failed', error = NULLIF(?, ''), completed_at = ?
WHERE id = ?
`, strings.TrimSpace(message), sqlitetime.Format(completedAt), jobID)
	if err != nil {
		return fmt.Errorf("fail job failed: %w", err)
	}
	return nil
}
