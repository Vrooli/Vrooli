package dochealingstore

// DOC: docs/concepts/ARCHITECTURE.md#integrations
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge-observatory/internal/services/dochealing"
)

type Postgres struct {
	DB *sql.DB
}

type jobRow struct {
	ID           string
	Scenario     string
	Status       string
	Progress     sql.NullString
	DiffJSON     []byte
	AgentRunID   sql.NullString
	Error        sql.NullString
	HealthBefore sql.NullFloat64
	HealthAfter  sql.NullFloat64
	AutoApprove  bool
	DryRun       bool
	CreatedAt    time.Time
	StartedAt    sql.NullTime
	CompletedAt  sql.NullTime
}

func (p *Postgres) CreateJob(ctx context.Context, req dochealing.HealRequest, healthBefore *float64) (string, error) {
	if p == nil || p.DB == nil {
		return "", fmt.Errorf("job store not configured")
	}
	var id string
	issuesJSON, err := json.Marshal(req.Issues)
	if err != nil {
		return "", fmt.Errorf("marshal issues: %w", err)
	}
	var healthValue interface{} = nil
	if healthBefore != nil {
		healthValue = *healthBefore
	}
	err = p.DB.QueryRowContext(ctx, `
INSERT INTO knowledge_observatory.doc_heal_jobs
  (scenario_name, issues, auto_approve, dry_run, health_before, status, created_at)
VALUES
  ($1, $2, $3, $4, $5, 'pending', NOW())
RETURNING id
`, strings.TrimSpace(req.ScenarioName), issuesJSON, req.AutoApprove, req.DryRun, healthValue).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create doc heal job failed: %w", err)
	}
	return id, nil
}

func (p *Postgres) GetJob(ctx context.Context, jobID string) (dochealing.HealJob, bool, error) {
	if p == nil || p.DB == nil {
		return dochealing.HealJob{}, false, fmt.Errorf("job store not configured")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return dochealing.HealJob{}, false, fmt.Errorf("job_id is required")
	}

	var row jobRow
	err := p.DB.QueryRowContext(ctx, `
SELECT id, scenario_name, status, COALESCE(progress, ''), COALESCE(diff, '{}'::jsonb), COALESCE(agent_run_id::text, ''), COALESCE(error, ''),
       health_before, health_after, auto_approve, dry_run, created_at, started_at, completed_at
FROM knowledge_observatory.doc_heal_jobs
WHERE id = $1
`, jobID).Scan(&row.ID, &row.Scenario, &row.Status, &row.Progress, &row.DiffJSON, &row.AgentRunID, &row.Error,
		&row.HealthBefore, &row.HealthAfter, &row.AutoApprove, &row.DryRun, &row.CreatedAt, &row.StartedAt, &row.CompletedAt)
	if err == sql.ErrNoRows {
		return dochealing.HealJob{}, false, nil
	}
	if err != nil {
		return dochealing.HealJob{}, false, fmt.Errorf("get doc heal job failed: %w", err)
	}

	var diff *dochealing.DiffPreview
	if len(row.DiffJSON) > 0 {
		if string(row.DiffJSON) != "{}" {
			var parsed dochealing.DiffPreview
			if err := json.Unmarshal(row.DiffJSON, &parsed); err == nil {
				diff = &parsed
			}
		}
	}

	job := dochealing.HealJob{
		JobID:        row.ID,
		ScenarioName: row.Scenario,
		Status:       row.Status,
		Progress:     strings.TrimSpace(row.Progress.String),
		Diff:         diff,
		Error:        strings.TrimSpace(row.Error.String),
		AgentRunID:   strings.TrimSpace(row.AgentRunID.String),
		AutoApprove:  row.AutoApprove,
		DryRun:       row.DryRun,
	}
	if row.HealthBefore.Valid {
		value := row.HealthBefore.Float64
		job.HealthBefore = &value
	}
	if row.HealthAfter.Valid {
		value := row.HealthAfter.Float64
		job.HealthAfter = &value
	}
	if row.StartedAt.Valid {
		t := row.StartedAt.Time.UTC()
		job.StartedAt = &t
	}
	if row.CompletedAt.Valid {
		t := row.CompletedAt.Time.UTC()
		job.CompletedAt = &t
	}

	return job, true, nil
}

func (p *Postgres) MarkRunning(ctx context.Context, jobID, runID string, startedAt time.Time) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET status = 'running', agent_run_id = NULLIF($2, '')::uuid, started_at = $3
WHERE id = $1
`, jobID, runID, startedAt.UTC())
	if err != nil {
		return fmt.Errorf("mark running failed: %w", err)
	}
	return nil
}

func (p *Postgres) UpdateProgress(ctx context.Context, jobID, progress string) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET progress = NULLIF($2, '')
WHERE id = $1
`, jobID, strings.TrimSpace(progress))
	if err != nil {
		return fmt.Errorf("update progress failed: %w", err)
	}
	return nil
}

func (p *Postgres) UpdateReview(ctx context.Context, jobID string, diff *dochealing.DiffPreview, healthAfter *float64, status string, completedAt time.Time) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	var blob []byte
	var err error
	if diff == nil {
		blob = []byte("{}")
	} else {
		blob, err = json.Marshal(diff)
		if err != nil {
			return fmt.Errorf("marshal diff failed: %w", err)
		}
	}
	var healthValue interface{} = nil
	if healthAfter != nil {
		healthValue = *healthAfter
	}
	_, err = p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET status = $2, diff = $3, health_after = $4, completed_at = $5, error = NULL
WHERE id = $1
`, jobID, status, blob, healthValue, completedAt.UTC())
	if err != nil {
		return fmt.Errorf("update review failed: %w", err)
	}
	return nil
}

func (p *Postgres) UpdateError(ctx context.Context, jobID, message string) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET error = NULLIF($2, '')
WHERE id = $1
`, jobID, strings.TrimSpace(message))
	if err != nil {
		return fmt.Errorf("update error failed: %w", err)
	}
	return nil
}

func (p *Postgres) MarkApproved(ctx context.Context, jobID, actor string, approvedAt time.Time, healthAfter *float64) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	var healthValue interface{} = nil
	if healthAfter != nil {
		healthValue = *healthAfter
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET status = 'approved', approved_by = NULLIF($2, ''), approved_at = $3, health_after = $4, error = NULL
WHERE id = $1
`, jobID, strings.TrimSpace(actor), approvedAt.UTC(), healthValue)
	if err != nil {
		return fmt.Errorf("mark approved failed: %w", err)
	}
	return nil
}

func (p *Postgres) MarkRejected(ctx context.Context, jobID, actor, reason string, completedAt time.Time) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET status = 'rejected', rejected_by = NULLIF($2, ''), reject_reason = NULLIF($3, ''), rejected_at = $4, error = NULL
WHERE id = $1
`, jobID, strings.TrimSpace(actor), strings.TrimSpace(reason), completedAt.UTC())
	if err != nil {
		return fmt.Errorf("mark rejected failed: %w", err)
	}
	return nil
}

func (p *Postgres) FailJob(ctx context.Context, jobID, message string, completedAt time.Time) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.doc_heal_jobs
SET status = 'failed', error = NULLIF($2, ''), completed_at = $3
WHERE id = $1
`, jobID, strings.TrimSpace(message), completedAt.UTC())
	if err != nil {
		return fmt.Errorf("fail job failed: %w", err)
	}
	return nil
}
