package deepsearchstore

// DOC: docs/concepts/ARCHITECTURE.md#integrations
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge-observatory/internal/services/deepsearch"
)

type Postgres struct {
	DB *sql.DB
}

type jobRow struct {
	ID          string
	Status      string
	Progress    sql.NullString
	ResultsJSON []byte
	AgentRunID  sql.NullString
	Error       sql.NullString
	CreatedAt   time.Time
	StartedAt   sql.NullTime
	CompletedAt sql.NullTime
}

func (p *Postgres) CreateJob(ctx context.Context, req deepsearch.DeepSearchRequest) (string, error) {
	if p == nil || p.DB == nil {
		return "", fmt.Errorf("job store not configured")
	}
	var id string
	err := p.DB.QueryRowContext(ctx, `
INSERT INTO knowledge_observatory.deep_search_jobs
  (query, scope, scenario_name, base_path, status, created_at)
VALUES
  ($1, $2, NULLIF($3, ''), NULLIF($4, ''), 'pending', NOW())
RETURNING id
`, req.Query, req.Scope, req.Scenario, req.BasePath).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create deep search job failed: %w", err)
	}
	return id, nil
}

func (p *Postgres) GetJob(ctx context.Context, jobID string) (deepsearch.DeepSearchJob, bool, error) {
	if p == nil || p.DB == nil {
		return deepsearch.DeepSearchJob{}, false, fmt.Errorf("job store not configured")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return deepsearch.DeepSearchJob{}, false, fmt.Errorf("job_id is required")
	}

	var row jobRow
	err := p.DB.QueryRowContext(ctx, `
SELECT id, status, COALESCE(progress, ''), COALESCE(results, '{}'::jsonb), COALESCE(agent_run_id::text, ''), COALESCE(error, ''),
       created_at, started_at, completed_at
FROM knowledge_observatory.deep_search_jobs
WHERE id = $1
`, jobID).Scan(&row.ID, &row.Status, &row.Progress, &row.ResultsJSON, &row.AgentRunID, &row.Error, &row.CreatedAt, &row.StartedAt, &row.CompletedAt)
	if err == sql.ErrNoRows {
		return deepsearch.DeepSearchJob{}, false, nil
	}
	if err != nil {
		return deepsearch.DeepSearchJob{}, false, fmt.Errorf("get deep search job failed: %w", err)
	}

	results := []deepsearch.DeepSearchResult{}
	if len(row.ResultsJSON) > 0 {
		_ = json.Unmarshal(row.ResultsJSON, &results)
	}

	job := deepsearch.DeepSearchJob{
		JobID:      row.ID,
		Status:     row.Status,
		Progress:   strings.TrimSpace(row.Progress.String),
		Results:    results,
		Error:      strings.TrimSpace(row.Error.String),
		AgentRunID: strings.TrimSpace(row.AgentRunID.String),
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
UPDATE knowledge_observatory.deep_search_jobs
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
UPDATE knowledge_observatory.deep_search_jobs
SET progress = NULLIF($2, '')
WHERE id = $1
`, jobID, strings.TrimSpace(progress))
	if err != nil {
		return fmt.Errorf("update progress failed: %w", err)
	}
	return nil
}

func (p *Postgres) CompleteJob(ctx context.Context, jobID string, results []deepsearch.DeepSearchResult, completedAt time.Time) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	blob, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal results failed: %w", err)
	}
	_, err = p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.deep_search_jobs
SET status = 'completed', results = $2, completed_at = $3, error = NULL
WHERE id = $1
`, jobID, blob, completedAt.UTC())
	if err != nil {
		return fmt.Errorf("complete job failed: %w", err)
	}
	return nil
}

func (p *Postgres) FailJob(ctx context.Context, jobID, message string, completedAt time.Time) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.deep_search_jobs
SET status = 'failed', error = NULLIF($2, ''), completed_at = $3
WHERE id = $1
`, jobID, strings.TrimSpace(message), completedAt.UTC())
	if err != nil {
		return fmt.Errorf("fail job failed: %w", err)
	}
	return nil
}
