package stagedmessages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
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
			next := job
			next.ErrorCode, next.Error, next.Retryable = classifyDispatchError(err, firstDispatch)
			if job.ExecutionID == "" && job.DispatchAttempted {
				latest, readErr := s.store.Get(ctx, job.ID)
				if readErr != nil {
					return reviewjobs.Job{}, readErr
				}
				if latest.State.Terminal() {
					return latest, nil
				}
				if latest.CancelRequested {
					job = latest
					next = latest
					next.ErrorCode, next.Error, next.Retryable = classifyDispatchError(err, firstDispatch)
					if next.ErrorCode == "dispatch_unconfirmed" || !firstDispatch {
						next.ErrorCode = "dispatch_unconfirmed"
						next.Error = "Cancellation is waiting for Agent Manager to confirm whether the request was accepted; the same owner key will be reconciled."
						next.Retryable = false
					}
				}
			}
			var evidenceReadErr EvidenceReadError
			if errors.As(err, &evidenceReadErr) && ex.ID != "" && !job.CancelRequested && (job.ExecutionID == "" || job.ExecutionID == ex.ID) {
				// The owner admitted and completed this execution, but its result
				// evidence is not readable yet. Persist the exact handle and any
				// measured receipts so reconnect can read this execution instead of
				// redispatching or pretending no work happened.
				next.ExecutionID = ex.ID
				if ex.Phase != "" {
					next.Phase = ex.Phase
				}
				if ex.ExecutionStartedKnown {
					next.ExecutionStarted, next.ExecutionStartedKnown = ex.ExecutionStarted, true
				}
				if ex.TokensKnown {
					next.TokensUsed, next.TokensKnown = ex.TokensUsed, true
				}
				if ex.ChargeKnown {
					next.ChargeMicroUSD, next.ChargeKnown = ex.ChargeMicroUSD, true
				}
				if job.ExecutionID == "" && ex.Status == "succeeded" && next.State == reviewjobs.Dispatching {
					if e := next.Transition(reviewjobs.Running); e != nil {
						return reviewjobs.Job{}, e
					}
				}
			}
			// A typed owner NotFound for a previously saved handle is affirmative
			// evidence that this execution no longer exists. Transport failures and
			// handle-less admission errors remain unresolved instead of stale.
			if job.ExecutionID != "" && connect.CodeOf(err) == connect.CodeNotFound {
				next.ErrorCode = "execution_missing"
				next.Error = "Agent Manager no longer has the saved execution. This job is stale; no replacement was started."
				next.Retryable = false
				if e := next.Transition(reviewjobs.Stale); e != nil {
					return reviewjobs.Job{}, e
				}
			}
			switch connect.CodeOf(err) {
			case connect.CodeInvalidArgument, connect.CodePermissionDenied, connect.CodeUnauthenticated, connect.CodeFailedPrecondition, connect.CodeUnimplemented:
				if firstDispatch && job.ExecutionID == "" {
					target := reviewjobs.Failed
					if job.CancelRequested {
						target = reviewjobs.Cancelled
					}
					if e := next.Transition(target); e != nil {
						return reviewjobs.Job{}, e
					}
				}
			}
			// Only the first dispatch can establish a definitive admission refusal.
			// Once Running is durable, a prior lost acknowledgement may conceal an
			// accepted owner execution. Later auth/outage errors retain its key so
			// restored credentials can recover that execution rather than replace it.
			if action == "wait" && !next.State.Terminal() {
				if e := waitAttachment(ctx, attachedAt, timeout); e != nil {
					return reviewjobs.Job{}, e
				}
			}
			if _, e := s.store.UpdateAdvisory(ctx, job, next); e != nil {
				return reviewjobs.Job{}, e
			}
		} else {
			next := job
			if job.ExecutionID == "" && job.DispatchAttempted {
				latest, readErr := s.store.Get(ctx, job.ID)
				if readErr != nil {
					return reviewjobs.Job{}, readErr
				}
				if latest.State.Terminal() {
					return latest, nil
				}
				if latest.CancelRequested {
					job = latest
					next = latest
					if ex.ID == "" && job.ExecutionID == "" {
						next.ErrorCode = "dispatch_unconfirmed"
						next.Error = "Cancellation is waiting for Agent Manager to confirm whether the request was accepted; the same owner key will be reconciled."
						next.Retryable = false
						if _, updateErr := s.store.UpdateAdvisory(ctx, job, next); updateErr != nil {
							return reviewjobs.Job{}, updateErr
						}
						return s.store.Get(ctx, job.ID)
					}
					if ex.ID != "" && ex.Status != "succeeded" && ex.Status != "failed" && ex.Status != "cancelled" {
						var cancelled Execution
						var cancelErr error
						if latest.ExecutionID == ex.ID {
							// The owner-side key fence already adopted and cancelled this
							// execution. Observe its latest status; a second Cancel can race
							// terminal completion and turn a valid receipt into an error.
							cancelled, cancelErr = s.provider.Get(ctx, ex.ID, input)
						} else {
							cancelled, cancelErr = s.provider.Cancel(ctx, ex.ID, input)
						}
						if cancelErr != nil {
							next.ExecutionID = ex.ID
							next.ErrorCode = "execution_status_unavailable"
							next.Error = "Agent Manager could not confirm cancellation; the saved execution identity is retained for recovery."
							next.Retryable = true
							if _, updateErr := s.store.UpdateAdvisory(ctx, job, next); updateErr != nil {
								return reviewjobs.Job{}, updateErr
							}
							return s.store.Get(ctx, job.ID)
						}
						if cancelled.ID != "" && cancelled.ID != ex.ID {
							return reviewjobs.Job{}, fmt.Errorf("execution owner returned a different handle during cancellation")
						}
						if cancelled.ID == "" {
							cancelled.ID = ex.ID
						}
						ex = cancelled
					}
				}
			}
			maintenanceRefusal := ex.ErrorCode == "provider_admission_maintenance" && (!ex.ExecutionStartedKnown || !ex.ExecutionStarted)
			if strings.TrimSpace(ex.Phase) != "" {
				next.Phase = ex.Phase
			}
			next.ExecutionID = ex.ID
			if next.ExecutionID == "" {
				next.ExecutionID = job.ExecutionID
			}
			next.Error = ex.Error
			// A later authoritative owner read supersedes a prior transient
			// projection error. Do not leave a recovered job carrying a stale
			// failure category.
			next.ErrorCode = ex.ErrorCode
			next.Retryable = ex.Retryable
			if ex.ExecutionStartedKnown {
				next.ExecutionStarted, next.ExecutionStartedKnown = ex.ExecutionStarted, true
			} else if ex.ErrorCode == "provider_admission_maintenance" {
				// Backward-compatible workflow projections may carry the stable
				// maintenance terminal receipt without the new evidence bit.
				next.ExecutionStarted = false
				next.ExecutionStartedKnown = true
			} else if ex.Status == "succeeded" {
				// A completed drafting workflow proves that its child runner ran.
				next.ExecutionStarted, next.ExecutionStartedKnown = true, true
			}
			if ex.TokensKnown {
				next.TokensUsed, next.TokensKnown = ex.TokensUsed, true
			}
			if ex.ChargeKnown {
				next.ChargeMicroUSD, next.ChargeKnown = ex.ChargeMicroUSD, true
			}
			if maintenanceRefusal {
				// A typed prelaunch refusal proves that no child work or charge occurred.
				next.ChargeMicroUSD, next.ChargeKnown = 0, true
			}
			if next.ExecutionID == "" {
				target := reviewjobs.Failed
				if maintenanceRefusal && !job.CancelRequested {
					target = reviewjobs.AwaitingProvider
				}
				if e := next.Transition(target); e != nil {
					return reviewjobs.Job{}, e
				}
				if target == reviewjobs.Failed {
					next.Error = "execution owner returned no handle"
				}
			} else {
				var target reviewjobs.State
				if job.CancelRequested {
					switch ex.Status {
					case "succeeded":
						target = reviewjobs.Cancelled
						next.Error = "Execution completed after cancellation request; result discarded"
						next.ResultJSON = ""
					case "failed":
						target = reviewjobs.Failed
					case "cancelled":
						target = reviewjobs.Cancelled
					default:
						target = reviewjobs.Cancelling
					}
				} else {
					switch ex.Status {
					case "queued", "running", "awaiting_input", string(reviewjobs.Cancelling), "succeeded", "failed", "cancelled":
						target = reviewjobs.State(ex.Status)
					default:
						// Unknown owner status is not evidence of a failed run. Keep
						// the last durable state and expose the unsupported receipt.
						target = job.State
						next.ErrorCode = "unknown_owner_status"
						next.Error = "Agent Manager returned an unsupported execution state. The last confirmed job state is retained."
						next.Retryable = true
					}
				}
				if !job.CancelRequested && maintenanceRefusal {
					target = reviewjobs.AwaitingProvider
				}
				if target == reviewjobs.Succeeded {
					if input.supportsDeepEvidence() {
						current, e := s.store.Get(ctx, job.ID)
						if e != nil {
							return reviewjobs.Job{}, e
						}
						input.EvidenceEligibility = suppliedEvidence(current.Checks, input.Snapshot.Digest, ex.ID)
						input.InventoryEligibility = suppliedInventory(current.Checks, input.Snapshot.Digest, ex.ID)
					}
					ConstrainSynthesisScope(input, ex.Result)
					if e := validateResult(input, ex.Result); e != nil {
						target = reviewjobs.Failed
						next.ErrorCode = "invalid_structured_output"
						next.Retryable = true
						next.Error = e.Error()
					} else {
						data, _ := protojson.Marshal(ex.Result)
						next.ResultJSON = string(data)
					}
				}
				if e := next.Transition(target); e != nil {
					return reviewjobs.Job{}, e
				}
			}
			if _, e := s.store.UpdateAdvisory(ctx, job, next); e != nil {
				return reviewjobs.Job{}, e
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

// resolveCancelledDispatch never retries Start after cancellation. It first
// asks the owner for an execution already admitted under the durable key. An
// authoritative miss proves Start was not accepted; a found execution is
// attached durably and observed from the owner's post-cancellation receipt.
// Lookup outages preserve Cancelling and uncertainty without creating work.
func (s *Service) resolveCancelledDispatch(ctx context.Context, repo string, job reviewjobs.Job, input DraftInput, timeout time.Duration) (reviewjobs.Job, error) {
	fencer, ok := s.provider.(DispatchCancellationProvider)
	if !ok {
		return s.recordDispatchUnconfirmed(ctx, job)
	}
	result, err := fencer.CancelByKey(ctx, job.IdempotencyKey, input)
	if err != nil || !result.Fenced {
		// A failed or non-fenced owner response cannot prove that a Start RPC
		// still in flight will not be accepted later.
		return s.recordDispatchUnconfirmed(ctx, job)
	}
	if result.Execution == nil {
		// The owner committed its cancellation fence and found no admitted
		// execution in the same serialized boundary. A late Start is now refused.
		next := job
		if err := next.Transition(reviewjobs.Cancelled); err != nil {
			return reviewjobs.Job{}, err
		}
		next.ErrorCode = "cancelled_before_start"
		next.Error = "Cancellation fenced the request before Agent Manager accepted it. No agent work started."
		next.Retryable = false
		next.ExecutionStarted, next.ExecutionStartedKnown = false, true
		next.TokensUsed, next.TokensKnown = 0, true
		next.ChargeMicroUSD, next.ChargeKnown = 0, true
		if _, updateErr := s.store.UpdateAdvisory(ctx, job, next); updateErr != nil {
			return reviewjobs.Job{}, updateErr
		}
		return s.store.Get(ctx, job.ID)
	}
	execution := *result.Execution
	if execution.ID == "" {
		return s.recordDispatchUnconfirmed(ctx, job)
	}
	next := job
	next.ExecutionID = execution.ID
	next.ErrorCode = ""
	next.Error = ""
	next.Retryable = false
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return reviewjobs.Job{}, err
	}
	attached, err := s.store.Get(ctx, job.ID)
	if err != nil {
		return reviewjobs.Job{}, err
	}
	// The owner fence endpoint already requested cancellation when the execution
	// was nonterminal. Observe its exact receipt instead of issuing another
	// cancellation, which can be invalid if the execution completed in between.
	return s.reconcile(ctx, repo, attached, input, ownerReceiptAction, timeout)
}

func (s *Service) recordDispatchUnconfirmed(ctx context.Context, job reviewjobs.Job) (reviewjobs.Job, error) {
	if job.ErrorCode == "dispatch_unconfirmed" {
		return job, nil
	}
	next := job
	next.ErrorCode = "dispatch_unconfirmed"
	next.Error = "Cancellation is waiting for Agent Manager to confirm whether the request was accepted; no new Start will be sent after cancellation."
	next.Retryable = false
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return reviewjobs.Job{}, err
	}
	return s.store.Get(ctx, job.ID)
}

// classifyDispatchError persists a bounded, operator-safe receipt. Transport
// details can contain implementation or credential context, so they stay out
// of the durable job/UI. Ambiguous owner acknowledgements remain nonterminal.
func classifyDispatchError(err error, firstDispatch bool) (code, message string, retryable bool) {
	var evidenceErr EvidenceReadError
	if errors.As(err, &evidenceErr) {
		return "evidence_unavailable", "Agent Manager finished the execution, but its saved result evidence could not be read. The job remains attached; reconnect to retrieve the same result.", true
	}
	// Authentication and policy errors remain actionable after dispatch too:
	// the durable execution identity is retained, and reconnecting after the
	// access issue is fixed must observe that same run rather than start another.
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated:
		return "authentication_required", "Agent Manager authentication is required to reconnect to this saved execution. Restore access, then reconnect to the same job; no replacement was started.", true
	case connect.CodePermissionDenied:
		if firstDispatch {
			return "policy_denied", "Agent Manager policy denied the request. Review the authorization policy before starting a new job.", false
		}
		return "policy_denied", "Agent Manager policy denied access to this saved execution. Resolve the policy, then reconnect to the same job; no replacement was started.", true
	}
	if !firstDispatch {
		return "execution_status_unavailable", "Agent Manager could not confirm the current execution status; the saved execution identity is retained.", true
	}
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument:
		return "invalid_request", "Agent Manager rejected the request as invalid.", false
	case connect.CodeUnauthenticated:
		return "authentication_required", "Agent Manager authentication is required before this request can start.", true
	case connect.CodePermissionDenied:
		return "policy_denied", "Agent Manager policy denied the request.", false
	case connect.CodeFailedPrecondition:
		return "workflow_precondition_failed", "Agent Manager refused this workflow request before confirming an execution.", false
	case connect.CodeUnimplemented:
		return "unsupported_operation", "Agent Manager does not support this request.", false
	default:
		return "dispatch_unconfirmed", "Could not confirm whether Agent Manager accepted the request; the existing request will be reconciled with the same owner key.", false
	}
}

func waitAttachment(ctx context.Context, started time.Time, timeout time.Duration) error {
	remaining := timeout - time.Since(started)
	if remaining <= 0 {
		return nil
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
