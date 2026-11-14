package imageengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrWaitBudgetExhausted reports a generation that ran past the time this
// scenario was prepared to wait for it. The job is cancelled before this is
// returned, so the error never describes work still running.
var ErrWaitBudgetExhausted = errors.New("image-tools inference: job outran its wait budget")

// WaitPolicy is how long a generation is waited for, and how.
//
// A generation used to be waited for with one WaitJob on the caller's context.
// Whatever cut that context short — in practice a 15 s proxy timeout in front
// of a CPU Stable Diffusion run that takes 18-26 minutes — ended the wait, and
// nothing told image-tools: every abandoned job ran to completion for nobody.
// The wait is now sized from the job's own estimate, taken in bounded chunks so
// a dropped connection or an image-tools restart is waited through rather than
// treated as the end, and every way of giving up cancels the job first.
type WaitPolicy struct {
	// Chunk bounds one WaitJob call. An unfinished job is checked and waited
	// on again; reaching the end of a chunk is not giving up.
	Chunk time.Duration
	// MinRun is the least run time any job is allowed, whatever its estimate.
	MinRun time.Duration
	// Unknown is the run budget when image-tools gives no estimate.
	Unknown time.Duration
	// MaxWait caps the whole wait, queueing included.
	MaxWait time.Duration
	// RetryBackoff is the pause before re-waiting after a transient failure.
	RetryBackoff time.Duration
	// MaxTransient is how many consecutive transient failures are ridden out.
	MaxTransient int
	// CancelTimeout bounds the cancel sent for an abandoned job.
	CancelTimeout time.Duration
}

// estimateFactor is the slack over image-tools' estimate. Estimates are made
// before a job runs, on a host whose load they cannot predict; a render at
// three times its estimate is slow, not wedged.
const estimateFactor = 3

func (p WaitPolicy) withDefaults() WaitPolicy {
	if p.Chunk <= 0 {
		p.Chunk = 2 * time.Minute
	}
	if p.MinRun <= 0 {
		p.MinRun = 10 * time.Minute
	}
	if p.Unknown <= 0 {
		p.Unknown = 45 * time.Minute
	}
	if p.MaxWait <= 0 {
		p.MaxWait = 2 * time.Hour
	}
	if p.RetryBackoff <= 0 {
		p.RetryBackoff = 2 * time.Second
	}
	if p.MaxTransient <= 0 {
		p.MaxTransient = 5
	}
	if p.CancelTimeout <= 0 {
		p.CancelTimeout = 10 * time.Second
	}
	return p
}

// runBudget is how long a running job with this estimate is waited for.
func (p WaitPolicy) runBudget(estimatedSeconds int) time.Duration {
	budget := p.Unknown
	if estimatedSeconds > 0 {
		budget = time.Duration(estimatedSeconds) * time.Second * estimateFactor
		if budget < p.MinRun {
			budget = p.MinRun
		}
	}
	if budget > p.MaxWait {
		budget = p.MaxWait
	}
	return budget
}

// errChunkElapsed marks a WaitJob that ended because its chunk did, with the
// job still unfinished and the caller still waiting.
var errChunkElapsed = errors.New("wait chunk elapsed")

// transientError is a wait failure worth waiting through: the connection
// dropped, or image-tools answered that it is briefly unavailable.
type transientError struct{ err error }

func (e *transientError) Error() string { return e.err.Error() }
func (e *transientError) Unwrap() error { return e.err }

// terminalError is a job that ended without a result. It needs no cancel.
type terminalError struct{ err error }

func (e *terminalError) Error() string { return e.err.Error() }
func (e *terminalError) Unwrap() error { return e.err }

const jobsServicePath = "/vrooli.image_tools.v1.jobs.JobsService/"

// awaitJob waits for a submitted generation to finish, for as long as the job
// can legitimately take, and cancels it on every path that stops waiting
// before it has finished.
func (c *Client) awaitJob(ctx context.Context, client *http.Client, base, jobID string, estimatedSeconds int) (waitGenerationResponse, error) {
	policy := c.Wait.withDefaults()
	base = strings.TrimRight(base, "/")
	start := time.Now()
	ceiling := start.Add(policy.MaxWait)
	deadline := earliest(start.Add(policy.runBudget(estimatedSeconds)), ceiling)
	transient := 0

	abandon := func(cause error) (waitGenerationResponse, error) {
		err := fmt.Errorf("image-tools inference job %s abandoned: %w", jobID, cause)
		if cancelErr := c.cancelJob(ctx, client, base, jobID, policy.CancelTimeout); cancelErr != nil {
			err = errors.Join(err, fmt.Errorf("cancel abandoned job %s: %w", jobID, cancelErr))
		}
		return waitGenerationResponse{}, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return abandon(err)
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return abandon(fmt.Errorf("%w (waited %s)", ErrWaitBudgetExhausted, time.Since(start).Round(time.Second)))
		}
		waited, err := c.waitChunk(ctx, client, base, jobID, min(policy.Chunk, remaining))
		var terminal *terminalError
		var flaky *transientError
		switch {
		case err == nil:
			return waited, nil
		case ctx.Err() != nil:
			return abandon(ctx.Err())
		case errors.As(err, &terminal):
			return waitGenerationResponse{}, err
		case errors.Is(err, errChunkElapsed):
			transient = 0
			// The job's own record moves the deadline: a job still queued
			// has not started spending its run budget, and a running one is
			// measured from when it actually started.
			if job, probeErr := c.getJob(ctx, client, base, jobID, policy.CancelTimeout); probeErr == nil {
				deadline = earliest(nextDeadline(policy, job, estimatedSeconds, deadline), ceiling)
			}
		case errors.As(err, &flaky):
			transient++
			if transient > policy.MaxTransient {
				return abandon(err)
			}
			select {
			case <-ctx.Done():
				return abandon(ctx.Err())
			case <-time.After(policy.RetryBackoff):
			}
		default:
			return abandon(err)
		}
	}
}

// nextDeadline re-derives the wait deadline from what image-tools reports.
func nextDeadline(policy WaitPolicy, job jobRecord, estimatedSeconds int, current time.Time) time.Time {
	if job.EstimatedSeconds > 0 {
		estimatedSeconds = job.EstimatedSeconds
	}
	switch job.State {
	case "JOB_STATE_QUEUED", "JOB_STATE_ADMITTING":
		return time.Now().Add(policy.runBudget(estimatedSeconds))
	case "JOB_STATE_RUNNING":
		if !job.StartedAt.IsZero() {
			return job.StartedAt.Add(policy.runBudget(estimatedSeconds))
		}
	}
	return current
}

// waitChunk is one bounded WaitJob.
func (c *Client) waitChunk(ctx context.Context, client *http.Client, base, jobID string, chunk time.Duration) (waitGenerationResponse, error) {
	chunkCtx, cancel := context.WithTimeout(ctx, chunk)
	defer cancel()
	body, status, err := callJobs(chunkCtx, client, base, "WaitJob", jobID)
	if err != nil {
		if ctx.Err() == nil && errors.Is(chunkCtx.Err(), context.DeadlineExceeded) {
			return waitGenerationResponse{}, errChunkElapsed
		}
		if ctx.Err() != nil {
			return waitGenerationResponse{}, ctx.Err()
		}
		return waitGenerationResponse{}, &transientError{fmt.Errorf("image-tools inference wait: %w", err)}
	}
	if status < 200 || status >= 300 {
		failure := fmt.Errorf("image-tools inference wait failed: %d: %s", status, strings.TrimSpace(string(body)))
		if status >= 500 {
			return waitGenerationResponse{}, &transientError{failure}
		}
		return waitGenerationResponse{}, failure
	}
	var waited waitGenerationResponse
	if err := json.Unmarshal(body, &waited); err != nil {
		return waitGenerationResponse{}, fmt.Errorf("image-tools inference wait response: %w", err)
	}
	switch waited.Job.State {
	case "JOB_STATE_SUCCEEDED", "":
		// An empty state is an older image-tools that did not echo it; the
		// result reference is checked by the caller either way.
		return waited, nil
	case "JOB_STATE_FAILED":
		return waitGenerationResponse{}, &terminalError{fmt.Errorf("image-tools inference failed: %s", waited.Job.Error)}
	case "JOB_STATE_CANCELED":
		return waitGenerationResponse{}, &terminalError{fmt.Errorf("image-tools inference job %s was cancelled", jobID)}
	default:
		// WaitJob answers only at a terminal state; anything else is a job
		// still in flight, so it is waited on again.
		return waitGenerationResponse{}, errChunkElapsed
	}
}

// getJob reads the job's current record on a short context of its own.
func (c *Client) getJob(ctx context.Context, client *http.Client, base, jobID string, timeout time.Duration) (jobRecord, error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	body, status, err := callJobs(probeCtx, client, base, "GetJob", jobID)
	if err != nil {
		return jobRecord{}, err
	}
	if status < 200 || status >= 300 {
		return jobRecord{}, fmt.Errorf("image-tools GetJob: %d", status)
	}
	var got waitGenerationResponse
	if err := json.Unmarshal(body, &got); err != nil {
		return jobRecord{}, err
	}
	return got.Job, nil
}

// cancelJob tells image-tools to stop a job nobody will collect.
//
// It runs on a fresh context: the usual reason to cancel is that the caller's
// context has just ended, and a cancel sent on that context is refused by the
// HTTP client before it leaves the process. WithoutCancel keeps the caller's
// values (trace and test-mode markers) without inheriting its cancellation.
func (c *Client) cancelJob(ctx context.Context, client *http.Client, base, jobID string, timeout time.Duration) error {
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	body, status, err := callJobs(cancelCtx, client, base, "CancelJob", jobID)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("%d: %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// callJobs is one unary JSON call to image-tools' JobsService.
func callJobs(ctx context.Context, client *http.Client, base, procedure, jobID string) ([]byte, int, error) {
	payload, _ := json.Marshal(map[string]string{"id": jobID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+jobsServicePath+procedure, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func earliest(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
