package stagedmessages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
)

const ownerReceiptAction = "owner_receipt"

func (s *Service) Job(ctx context.Context, repo, dir, jobID, requestKey, action string, timeout time.Duration) (*v1.StagedMessageJob, error) {
	var job reviewjobs.Job
	var e error
	if jobID != "" {
		job, e = s.store.Get(ctx, jobID)
	} else if requestKey != "" {
		job, e = s.store.GetByIdempotency(ctx, key(repo, requestKey))
	} else {
		return nil, fmt.Errorf("job id or request key is required")
	}
	if e != nil {
		return nil, e
	}
	if job.Operation != operation || job.RepositoryID != repo {
		return nil, ErrNotFound
	}
	if action == "get" {
		return s.project(ctx, repo, dir, job)
	}
	input, e := decodeDraftInput([]byte(job.InputJSON))
	if e != nil {
		return nil, e
	}
	job, e = s.reconcile(ctx, repo, job, input, action, timeout)
	if e != nil {
		return nil, e
	}
	return s.project(ctx, repo, dir, job)
}

func terminal(state reviewjobs.State) bool { return state.Terminal() }

// projectedLifecycleState repairs one legacy projection without rewriting the
// durable row. Older maintenance refusals were persisted as failed even when
// their typed cause and explicit zero-work receipts prove they never started.
func projectedLifecycleState(job reviewjobs.Job) reviewjobs.State {
	if job.State == reviewjobs.Failed && job.ErrorCode == "provider_admission_maintenance" &&
		job.ExecutionStartedKnown && !job.ExecutionStarted && job.TokensKnown && job.TokensUsed == 0 &&
		(!job.ChargeKnown || job.ChargeMicroUSD == 0) {
		return reviewjobs.AwaitingProvider
	}
	return job.State
}

// advanceRead owns one explicit provider/CAS lifecycle step and then projects
// the resulting durable job. Public GETs call project directly.
func (s *Service) advanceRead(ctx context.Context, repo, dir string, job reviewjobs.Job, action string, timeout time.Duration) (*v1.StagedMessageJob, error) {
	input, e := decodeDraftInput([]byte(job.InputJSON))
	if e != nil {
		return nil, e
	}
	reconciled, e := s.reconcile(ctx, repo, job, input, action, timeout)
	if e != nil {
		return nil, e
	}
	return s.project(ctx, repo, dir, reconciled)
}

func (s *Service) project(ctx context.Context, repo, dir string, job reviewjobs.Job) (*v1.StagedMessageJob, error) {
	input, e := decodeDraftInput([]byte(job.InputJSON))
	if e != nil {
		return nil, e
	}
	errorCode, retryable := job.ErrorCode, job.Retryable
	// Older API builds persisted the workflow's successful terminal code
	// ("completed") when GCT rejected its structured draft. Preserve the
	// failure reason while restoring the retry policy documented for this case.
	if job.State == reviewjobs.Failed && strings.HasPrefix(job.Error, "Invalid structured message result:") {
		errorCode, retryable = "invalid_structured_output", true
	} else if job.State == reviewjobs.Failed && job.ErrorCode == "structured_result_invalid" {
		// Older Agent Manager revisions rejected raw XML before GCT could parse
		// it. Project those saved failures as retryable against the active
		// final-handoff revision without mutating the historical receipt.
		errorCode, retryable = "invalid_structured_output", true
	}
	phase := job.Phase
	if phase == "" {
		phase = string(projectedLifecycleState(job))
	}
	result := &v1.StagedMessageJob{Id: job.ID, RepositoryId: repo, SnapshotDigest: input.Snapshot.GetDigest(), Status: string(projectedLifecycleState(job)), CreatedAt: job.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: job.UpdatedAt.UTC().Format(time.RFC3339Nano), ExecutionId: job.ExecutionID, CancelRequested: job.CancelRequested, Error: job.Error, ErrorCode: errorCode, Retryable: retryable, ExecutionStarted: job.ExecutionStarted, ExecutionStartedKnown: job.ExecutionStartedKnown, TokensUsed: job.TokensUsed, TokensKnown: job.TokensKnown, ChargeMicroUsd: job.ChargeMicroUSD, ChargeKnown: job.ChargeKnown, Snapshot: input.Snapshot, Phase: phase, Omissions: input.Omissions}
	result.ContinuationOfJobId = input.ContinuationOfJobID
	if input.supportsDeepEvidence() {
		result.EvidenceFilesRead = uint32(len(suppliedEvidence(job.Checks, input.Snapshot.Digest, job.ExecutionID)))
		result.EvidenceFilesTotal = uint32(len(input.Snapshot.Files) - len(priorCovered(input)))
		if input.supportsSynthesis() {
			result.EvidenceFilesTotal = uint32(len(input.Snapshot.Files))
		}
	}
	if job.ResultJSON != "" {
		result.Result = &v1.StagedMessageSuggestion{}
		if e := protojson.Unmarshal([]byte(job.ResultJSON), result.Result); e != nil {
			return nil, e
		}
	}
	if job.State == reviewjobs.Succeeded && s.continuations != nil {
		// Projected against current history, so a commit made meanwhile counts.
		result.Continuations, _ = s.continuations.Continuations(ctx, dir, input.Snapshot)
	}
	var standing *v1.CheckStagedMessageResponse
	var applicabilityErr error
	if input.WorktreeCapture {
		applicable, checkErr := WorktreeSelectionApplicable(ctx, dir, input.Snapshot)
		standing = &v1.CheckStagedMessageResponse{Applicable: applicable}
		if !applicable {
			standing.Reason = "Selected worktree paths changed after capture"
		}
		applicabilityErr = checkErr
	} else {
		standing, applicabilityErr = s.Check(ctx, repo, dir, result.SnapshotDigest)
	}
	if applicabilityErr != nil {
		result.ApplicabilityReason = "Current staged selection unavailable: " + applicabilityErr.Error()
	} else {
		result.Applicable = standing.Applicable
		result.ApplicabilityReason = standing.Reason
	}
	return result, nil
}

// reconcile owns exactly one persisted/provider lifecycle step. Projection and
// applicability checks run once in read, even after a CAS or cancelled admission.
func (s *Service) reconcile(ctx context.Context, repo string, job reviewjobs.Job, input DraftInput, action string, timeout time.Duration) (reviewjobs.Job, error) {
	// A confirmed maintenance refusal is durable and quiet until the operator
	// explicitly retries from the saved input after provider readiness opens.
	if job.State == reviewjobs.AwaitingProvider {
		return job, nil
	}
	if !job.State.Terminal() {
		if action == "cancel" && !job.CancelRequested {
			if e := s.store.RequestAdvisoryCancel(ctx, job.ID, repo); e != nil {
				return reviewjobs.Job{}, e
			}
			var e error
			job, e = s.store.Get(ctx, job.ID)
			if e != nil {
				return reviewjobs.Job{}, e
			}
		}
		if job.State.Terminal() {
			return job, nil
		}
		if s.provider == nil {
			next := job
			if job.State == reviewjobs.Queued && job.ExecutionID == "" && (!job.DispatchAttemptedKnown || !job.DispatchAttempted) {
				if e := next.Transition(reviewjobs.Failed); e != nil {
					return reviewjobs.Job{}, e
				}
				next.ErrorCode = "provider_unavailable"
				next.Error = "Agent Manager message execution is unavailable; no request was dispatched."
				next.Retryable = true
				next.ExecutionStarted, next.ExecutionStartedKnown = false, true
				next.TokensUsed, next.TokensKnown = 0, true
				next.ChargeMicroUSD, next.ChargeKnown = 0, true
			} else if job.ExecutionID != "" {
				next.ErrorCode = "execution_status_unavailable"
				next.Error = "Agent Manager message execution is unavailable; the saved execution identity is retained."
				next.Retryable = true
			} else {
				next.ErrorCode = "dispatch_unconfirmed"
				next.Error = "Agent Manager message execution is unavailable; the saved request remains unresolved and will be reconciled with the same owner key."
				next.Retryable = false
			}
			if _, e := s.store.UpdateAdvisory(ctx, job, next); e != nil {
				return reviewjobs.Job{}, e
			}
			saved, e := s.store.Get(ctx, job.ID)
			if e != nil {
				return reviewjobs.Job{}, e
			}
			return saved, nil
		}
		if job.ExecutionID == "" && job.CancelRequested && job.DispatchAttempted {
			return s.resolveCancelledDispatch(ctx, repo, job, input, timeout)
		}
		attachedAt := s.now()
		var ex Execution
		var err error
		firstDispatch := false
		if job.ExecutionID == "" {
			// Admission is durable before dispatch. A crash/lost acknowledgement is
			// reconciled with the same owner idempotency key, never a fresh execution.
			firstDispatch = job.State == reviewjobs.Queued
			claimed, changed, e := s.store.ClaimAdvisoryDispatch(ctx, job)
			if e != nil {
				return reviewjobs.Job{}, e
			}
			current, e := s.store.Get(ctx, job.ID)
			if e != nil {
				return reviewjobs.Job{}, e
			}
			if !changed || current.State.Terminal() || current.ExecutionID != "" || !current.UpdatedAt.Equal(claimed.UpdatedAt) {
				return current, nil
			}
			if s.beforeDispatch != nil {
				s.beforeDispatch(ctx, current)
				current, e = s.store.Get(ctx, job.ID)
				if e != nil {
					return reviewjobs.Job{}, e
				}
				if current.State.Terminal() || current.ExecutionID != "" {
					return current, nil
				}
			}
			if current.CancelRequested && current.DispatchAttemptedKnown && !current.DispatchAttempted {
				if e := s.store.RequestAdvisoryCancel(ctx, current.ID, repo); e != nil {
					return reviewjobs.Job{}, e
				}
				return s.store.Get(ctx, current.ID)
			}
			// The caller that requested cancellation should not race a still-running
			// Start call with another Start. If the owner reply was ambiguous, a later
			// advance (or the settle worker) reuses the same idempotency key to adopt
			// any accepted handle before it cancels that execution.
			if current.CancelRequested && action == "cancel" && current.DispatchAttempted && current.ErrorCode != "dispatch_unconfirmed" {
				return current, nil
			}
			firstDispatch = current.DispatchAttemptedKnown && !current.DispatchAttempted
			if !current.DispatchAttemptedKnown || !current.DispatchAttempted {
				attempted, marked, markErr := s.store.MarkAdvisoryDispatchAttempt(ctx, current)
				if markErr != nil {
					return reviewjobs.Job{}, markErr
				}
				if !marked {
					return s.store.Get(ctx, current.ID)
				}
				current = attempted
			}
			// Retain this exact claim for the response CAS; a reload must never
			// lend a later retry's revision to this older owner response.
			job = current
			ex, err = s.provider.Start(ctx, input, job.IdempotencyKey)
		} else if job.CancelRequested && action != ownerReceiptAction {
			ex, err = s.provider.Cancel(ctx, job.ExecutionID, input)
		} else if action == "wait" {
			ex, err = s.provider.Wait(ctx, job.ExecutionID, input, timeout)
		} else {
			ex, err = s.provider.Get(ctx, job.ExecutionID, input)
		}
		if err == nil && job.ExecutionID != "" && ex.ID != "" && ex.ID != job.ExecutionID {
			err = fmt.Errorf("execution owner returned a different handle; original execution retained")
		}
		if err != nil {
			updated, done, recordErr := s.recordExecutionError(ctx, job, action, timeout, attachedAt, firstDispatch, ex, err)
			if recordErr != nil {
				return reviewjobs.Job{}, recordErr
			}
			if done {
				return updated, nil
			}
		} else {
			updated, done, recordErr := s.recordExecutionSuccess(ctx, job, input, ex)
			if recordErr != nil {
				return reviewjobs.Job{}, recordErr
			}
			if done {
				return updated, nil
			}

		}
		var e error
		job, e = s.store.Get(ctx, job.ID)
		if e != nil {
			return reviewjobs.Job{}, e
		}
	}
	return job, nil
}
