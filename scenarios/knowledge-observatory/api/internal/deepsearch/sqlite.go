package deepsearch

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

	"knowledge-observatory/internal/services/deepsearch"
)

// SQLite is the deepsearch domain's job store.
//
// It satisfies deepsearch.JobStore, which is declared by the service that owns
// the behaviour; this package owns only the storage.
type SQLite struct {
	DB *apidb.RoutedDB
}

var _ deepsearch.JobStore = (*SQLite)(nil)

// NewSQLite returns a job store backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

// CreateJob inserts a pending job and returns its id.
//
// Postgres generated the id with gen_random_uuid() and handed it back through
// RETURNING. SQLite has no UUID default, so the id is minted here.
func (s *SQLite) CreateJob(ctx context.Context, req deepsearch.DeepSearchRequest) (string, error) {
	if s == nil || s.DB == nil {
		return "", fmt.Errorf("job store not configured")
	}
	id := uuid.NewString()
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO deep_search_jobs
  (id, query, scope, scenario_name, base_path, max_results, status, created_at)
VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, 'pending', CURRENT_TIMESTAMP)
`, id, req.Query, req.Scope, strings.TrimSpace(req.Scenario), strings.TrimSpace(req.BasePath), req.MaxResults)
	if err != nil {
		return "", fmt.Errorf("create deep search job failed: %w", err)
	}
	return id, nil
}

func (s *SQLite) GetJob(ctx context.Context, jobID string) (deepsearch.DeepSearchJob, bool, error) {
	if s == nil || s.DB == nil {
		return deepsearch.DeepSearchJob{}, false, fmt.Errorf("job store not configured")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return deepsearch.DeepSearchJob{}, false, fmt.Errorf("job_id is required")
	}

	var (
		id, status, progress, agentRunID, errText string
		resultsJSON                               string
		maxResults                                sql.NullInt64
		createdAt                                 time.Time
		startedAt, completedAt                    sql.NullTime
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT id, status, COALESCE(progress, ''), COALESCE(results, '[]'), COALESCE(agent_run_id, ''),
       COALESCE(error, ''), max_results, created_at, started_at, completed_at
FROM deep_search_jobs
WHERE id = ?
`, jobID).Scan(&id, &status, &progress, &resultsJSON, &agentRunID, &errText,
		&maxResults, &createdAt, &startedAt, &completedAt)
	if err == sql.ErrNoRows {
		return deepsearch.DeepSearchJob{}, false, nil
	}
	if err != nil {
		return deepsearch.DeepSearchJob{}, false, fmt.Errorf("get deep search job failed: %w", err)
	}

	results := []deepsearch.DeepSearchResult{}
	if trimmed := strings.TrimSpace(resultsJSON); trimmed != "" && trimmed != "{}" {
		_ = json.Unmarshal([]byte(trimmed), &results)
	}

	job := deepsearch.DeepSearchJob{
		JobID:      id,
		Status:     status,
		Progress:   strings.TrimSpace(progress),
		Results:    results,
		Error:      strings.TrimSpace(errText),
		AgentRunID: strings.TrimSpace(agentRunID),
		MaxResults: int(maxResults.Int64),
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
UPDATE deep_search_jobs
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
		`UPDATE deep_search_jobs SET progress = NULLIF(?, '') WHERE id = ?`,
		strings.TrimSpace(progress), jobID)
	if err != nil {
		return fmt.Errorf("update progress failed: %w", err)
	}
	return nil
}

func (s *SQLite) CompleteJob(ctx context.Context, jobID string, results []deepsearch.DeepSearchResult, completedAt time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	blob, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("marshal results failed: %w", err)
	}
	_, err = s.DB.ExecContext(ctx, `
UPDATE deep_search_jobs
SET status = 'completed', results = ?, completed_at = ?, error = NULL
WHERE id = ?
`, string(blob), sqlitetime.Format(completedAt), jobID)
	if err != nil {
		return fmt.Errorf("complete job failed: %w", err)
	}
	return nil
}

func (s *SQLite) FailJob(ctx context.Context, jobID, message string, completedAt time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE deep_search_jobs
SET status = 'failed', error = NULLIF(?, ''), completed_at = ?
WHERE id = ?
`, strings.TrimSpace(message), sqlitetime.Format(completedAt), jobID)
	if err != nil {
		return fmt.Errorf("fail job failed: %w", err)
	}
	return nil
}
