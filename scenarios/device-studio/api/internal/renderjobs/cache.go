package renderjobs

import (
	"context"
	"errors"
	"io"

	"device-studio/internal/jobs"
)

const (
	cacheProjectPageSize = 101
	cacheProjectMaxPages = 500
)

func (c *Coordinator) submitInputWithCache(ctx context.Context, route routePin, req SubmitRequest, input jobs.Input) (jobs.Job, bool, *CacheReuseReceipt, error) {
	if req.ReuseCachedSuccess {
		cached, receipt, err := c.findReusableSuccess(ctx, route, input)
		if err != nil {
			return jobs.Job{}, false, nil, err
		}
		if receipt != nil {
			if err := c.requireRoute(ctx, route); err != nil {
				return jobs.Job{}, false, nil, err
			}
			_, fingerprint, err := jobs.CanonicalInput(input)
			if err != nil {
				return jobs.Job{}, false, nil, err
			}
			if err := c.jobs.BindCacheIdempotencyAlias(ctx, req.IdempotencyKey, cached.ID, fingerprint); err != nil {
				return jobs.Job{}, false, nil, err
			}
			if err := c.requireRoute(ctx, route); err != nil {
				return jobs.Job{}, false, nil, err
			}
			return cached, false, receipt, nil
		}
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, false, nil, err
	}
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: req.IdempotencyKey, Input: input})
	if err != nil {
		if errors.Is(err, jobs.ErrIdempotencyConflict) {
			if existing, found, lookupErr := c.findIdempotentSubmit(ctx, route, req); lookupErr != nil || found {
				return existing, false, nil, lookupErr
			}
		}
		return jobs.Job{}, false, nil, err
	}
	return job, created, nil, nil
}

// findReusableSuccess searches only the same project for the exact canonical
// immutable input. A cache hit is returned only after the ordinary output
// owner verifies the succeeded artifact bytes and their completion receipt.
func (c *Coordinator) findReusableSuccess(ctx context.Context, route routePin, input jobs.Input) (jobs.Job, *CacheReuseReceipt, error) {
	if input.ProjectID == "" || input.WorkflowScopeID != "" || input.AccountingRequired {
		return jobs.Job{}, nil, nil
	}
	canonical, fingerprint, err := jobs.CanonicalInput(input)
	if err != nil {
		return jobs.Job{}, nil, err
	}
	cursor := jobs.Cursor{}
	for pageIndex := 0; pageIndex < cacheProjectMaxPages; pageIndex++ {
		page, err := c.jobs.ListProject(ctx, canonical.ProjectID, cursor, cacheProjectPageSize)
		if err != nil {
			return jobs.Job{}, nil, err
		}
		for _, candidate := range page {
			cursor = jobs.Cursor{CreatedAt: candidate.CreatedAt, ID: candidate.ID}
			if candidate.InputFingerprint != fingerprint || candidate.Input.ProjectID != canonical.ProjectID ||
				candidate.Input.SubjectKind != canonical.SubjectKind || candidate.Input.SubjectID != canonical.SubjectID ||
				candidate.Input.ShotRevisionID != canonical.ShotRevisionID || candidate.Input.ShotRevision != canonical.ShotRevision ||
				candidate.Input.OutputProfile != canonical.OutputProfile || candidate.State != jobs.Succeeded ||
				candidate.Settlement == jobs.SettlementPending || candidate.Settlement == jobs.SettlementReconciliation {
				continue
			}
			attempt, ok := currentAttempt(candidate)
			if !ok || attempt.State != jobs.AttemptSucceeded || attempt.Completion == nil ||
				attempt.Completion.InputFingerprint != fingerprint || attempt.Completion.AttemptID != attempt.ID ||
				(attempt.Execution != nil && attempt.ExecutionReleasedAt == nil) {
				continue
			}
			receipt, err := c.verifyReusableOutput(ctx, canonical.ProjectID, candidate, attempt)
			if err != nil {
				// An unavailable or corrupt old artifact is a miss. It cannot be
				// returned as a cache hit, but does not prevent a fresh render.
				continue
			}
			if err := c.requireRoute(ctx, route); err != nil {
				return jobs.Job{}, nil, err
			}
			return candidate, receipt, nil
		}
		if len(page) < cacheProjectPageSize {
			return jobs.Job{}, nil, nil
		}
	}
	// The scan is deliberately bounded. A miss after the bound is safe; it
	// means only that an older cached result was not considered.
	return jobs.Job{}, nil, nil
}

func (c *Coordinator) verifyReusableOutput(ctx context.Context, projectID string, job jobs.Job, attempt jobs.Attempt) (*CacheReuseReceipt, error) {
	wantKind := "still_png"
	if job.Input.SubjectKind == jobs.SubjectPhysicalRender {
		wantKind = physicalOutputKind(job.Input.OutputProfile)
	}
	var output *jobs.OutputRef
	for i := range attempt.Completion.Outputs {
		if attempt.Completion.Outputs[i].Kind == wantKind {
			output = &attempt.Completion.Outputs[i]
			break
		}
	}
	if output == nil {
		return nil, ErrOutputUnavailable
	}
	if output.MIMEType == "image/png" {
		verified, _, err := c.OpenOutput(ctx, projectID, job.ID)
		if err != nil {
			return nil, err
		}
		verifiedAttempt, ok := currentAttempt(verified)
		if !ok || verifiedAttempt.ID != attempt.ID {
			return nil, ErrOutputUnavailable
		}
		return &CacheReuseReceipt{SourceRenderID: job.ID, SourceAttemptID: attempt.ID, SHA256: output.SHA256, OutputKind: output.Kind}, nil
	}
	if output.MIMEType != "video/mp4" {
		return nil, ErrOutputUnavailable
	}
	verified, ref, reader, err := c.OpenVideoOutput(ctx, projectID, job.ID)
	if err != nil {
		return nil, err
	}
	readBytes, readErr := io.Copy(io.Discard, io.LimitReader(reader, maxVideoOutputBytes+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || readBytes != ref.SizeBytes || ref.SHA256 != output.SHA256 || verified.ID != job.ID {
		return nil, ErrOutputUnavailable
	}
	return &CacheReuseReceipt{SourceRenderID: job.ID, SourceAttemptID: attempt.ID, SHA256: output.SHA256, OutputKind: output.Kind}, nil
}
