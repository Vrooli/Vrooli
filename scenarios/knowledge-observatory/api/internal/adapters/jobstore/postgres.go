package jobstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge-observatory/internal/ports"
)

type Postgres struct {
	DB *sql.DB
}

type jobRow struct {
	ID             string
	Status         string
	ErrorMessage   sql.NullString
	CreatedAt      time.Time
	StartedAt      sql.NullTime
	FinishedAt     sql.NullTime
	TotalChunks    int
	CompletedChunks int
	RequestJSON    []byte
}

func (p *Postgres) EnqueueDocumentIngest(ctx context.Context, req ports.DocumentIngestJobRequest) (string, error) {
	if p == nil || p.DB == nil {
		return "", fmt.Errorf("job store not configured")
	}
	if strings.TrimSpace(req.Namespace) == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if strings.TrimSpace(req.Collection) == "" {
		return "", fmt.Errorf("collection is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return "", fmt.Errorf("content is required")
	}

	if strings.TrimSpace(req.DocumentID) == "" {
		req.DocumentID = newUUIDv4ForJob()
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	var id string
	err = p.DB.QueryRowContext(ctx, `
INSERT INTO knowledge_observatory.ingest_jobs
  (request_json, status, created_at, total_chunks, completed_chunks)
VALUES
  ($1, 'pending', NOW(), 0, 0)
RETURNING id
`, raw).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("enqueue failed: %w", err)
	}
	return id, nil
}

func (p *Postgres) GetJob(ctx context.Context, jobID string) (ports.JobStatus, bool, error) {
	if p == nil || p.DB == nil {
		return ports.JobStatus{}, false, fmt.Errorf("job store not configured")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return ports.JobStatus{}, false, fmt.Errorf("jobID is required")
	}

	var row jobRow
	err := p.DB.QueryRowContext(ctx, `
SELECT id, status, COALESCE(error_message, ''), created_at, started_at, finished_at, total_chunks, completed_chunks, request_json
FROM knowledge_observatory.ingest_jobs
WHERE id = $1
`, jobID).Scan(&row.ID, &row.Status, &row.ErrorMessage, &row.CreatedAt, &row.StartedAt, &row.FinishedAt, &row.TotalChunks, &row.CompletedChunks, &row.RequestJSON)
	if err == sql.ErrNoRows {
		return ports.JobStatus{}, false, nil
	}
	if err != nil {
		return ports.JobStatus{}, false, fmt.Errorf("get job failed: %w", err)
	}

	createdAt := row.CreatedAt.UTC().Format(time.RFC3339)
	status := ports.JobStatus{
		JobID:           row.ID,
		Status:          row.Status,
		ErrorMessage:    strings.TrimSpace(row.ErrorMessage.String),
		CreatedAt:       &createdAt,
		TotalChunks:     row.TotalChunks,
		CompletedChunks: row.CompletedChunks,
	}
	if row.StartedAt.Valid {
		v := row.StartedAt.Time.UTC().Format(time.RFC3339)
		status.StartedAt = &v
	}
	if row.FinishedAt.Valid {
		v := row.FinishedAt.Time.UTC().Format(time.RFC3339)
		status.FinishedAt = &v
	}

	return status, true, nil
}

func (p *Postgres) ClaimNextPendingJob(ctx context.Context) (ports.DocumentIngestJob, bool, error) {
	if p == nil || p.DB == nil {
		return ports.DocumentIngestJob{}, false, fmt.Errorf("job store not configured")
	}

	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return ports.DocumentIngestJob{}, false, fmt.Errorf("begin tx failed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var row jobRow
	err = tx.QueryRowContext(ctx, `
SELECT id, status, COALESCE(error_message, ''), created_at, started_at, finished_at, total_chunks, completed_chunks, request_json
FROM knowledge_observatory.ingest_jobs
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
`).Scan(&row.ID, &row.Status, &row.ErrorMessage, &row.CreatedAt, &row.StartedAt, &row.FinishedAt, &row.TotalChunks, &row.CompletedChunks, &row.RequestJSON)
	if err == sql.ErrNoRows {
		return ports.DocumentIngestJob{}, false, nil
	}
	if err != nil {
		return ports.DocumentIngestJob{}, false, fmt.Errorf("claim select failed: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
UPDATE knowledge_observatory.ingest_jobs
SET status = 'running', started_at = NOW()
WHERE id = $1
`, row.ID)
	if err != nil {
		return ports.DocumentIngestJob{}, false, fmt.Errorf("claim update failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return ports.DocumentIngestJob{}, false, fmt.Errorf("commit failed: %w", err)
	}

	var req ports.DocumentIngestJobRequest
	if err := json.Unmarshal(row.RequestJSON, &req); err != nil {
		return ports.DocumentIngestJob{}, false, fmt.Errorf("failed to decode request_json: %w", err)
	}

	return ports.DocumentIngestJob{
		JobID: row.ID,
		Req:   req,
	}, true, nil
}

func (p *Postgres) UpdateJobProgress(ctx context.Context, jobID string, completedChunks, totalChunks int) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.ingest_jobs
SET completed_chunks = $2, total_chunks = $3
WHERE id = $1
`, jobID, completedChunks, totalChunks)
	if err != nil {
		return fmt.Errorf("update progress failed: %w", err)
	}
	return nil
}

func (p *Postgres) CompleteJob(ctx context.Context, jobID string, status string, errorMessage string) error {
	if p == nil || p.DB == nil {
		return fmt.Errorf("job store not configured")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = "success"
	}
	_, err := p.DB.ExecContext(ctx, `
UPDATE knowledge_observatory.ingest_jobs
SET status = $2, error_message = NULLIF($3, ''), finished_at = NOW()
WHERE id = $1
`, jobID, status, errorMessage)
	if err != nil {
		return fmt.Errorf("complete failed: %w", err)
	}
	return nil
}

func newUUIDv4ForJob() string {
	var buf [16]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	hexValue := hex.EncodeToString(buf[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}
