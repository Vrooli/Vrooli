package ingestjobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"knowledge-observatory/internal/services/ingest"

	"knowledge-observatory/internal/ports"
)

type Runner struct {
	Jobs   ports.JobStore
	Ingest *ingest.Service

	Now       func() time.Time
	Sleep     func(time.Duration)
	Interval  time.Duration
	MaxChunks int
}

func (r *Runner) Run(ctx context.Context) {
	interval := r.Interval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	sleep := r.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		didWork, err := r.processOne(ctx)
		if err != nil {
			sleep(interval)
			continue
		}
		if !didWork {
			sleep(interval)
		}
	}
}

func (r *Runner) processOne(ctx context.Context) (bool, error) {
	if r == nil || r.Jobs == nil || r.Ingest == nil {
		return false, fmt.Errorf("runner not configured")
	}

	job, ok, err := r.Jobs.ClaimNextPendingJob(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	req := job.Req
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Collection = strings.TrimSpace(req.Collection)
	req.DocumentID = strings.TrimSpace(req.DocumentID)
	req.Content = strings.TrimSpace(req.Content)

	chunks := ingest.ChunkText(req.Content, req.ChunkSize, req.ChunkOverlap, r.MaxChunks)
	_ = r.Jobs.UpdateJobProgress(ctx, job.JobID, 0, len(chunks))

	for i, chunk := range chunks {
		idx := i
		recordID := ingest.RecordIDForChunk(req.Namespace, req.DocumentID, i, chunk)
		_, err := r.Ingest.UpsertRecord(ctx, ingest.UpsertRecordRequest{
			Namespace:  req.Namespace,
			Collection: req.Collection,
			RecordID:   recordID,
			DocumentID: req.DocumentID,
			ChunkIndex: &idx,
			Content:    chunk,
			Metadata:   req.Metadata,
			Visibility: req.Visibility,
			Source:     req.Source,
			SourceType: req.SourceType,
		})
		if err != nil {
			_ = r.Jobs.CompleteJob(ctx, job.JobID, "failure", err.Error())
			return true, nil
		}
		_ = r.Jobs.UpdateJobProgress(ctx, job.JobID, i+1, len(chunks))
	}

	_ = r.Jobs.CompleteJob(ctx, job.JobID, "success", "")
	return true, nil
}

