// Outcome classification + handler dispatch.
//
// HandleResult is the post-execute seam: it classifies the runner's
// verdict into a domain.RunOutcome, dispatches to the appropriate
// handler (success / failure / cancel), and updates the run record.
// Each handler may invoke ApplyAtRunEnd via the caller (it's wired in
// for handler convenience but lives in finalize.go because it's load-
// bearing for terminal teardown).
//
// Sandbox teardown is centralized in Finalize (the deferred terminal
// seam); HandleResult does not call ApplySandboxLifecycle directly.
//
// Typed runner errors flow through HandleResult: when [core.Runner]
// stores a [*domain.RunnerError] on [runner.ExecuteResult.TerminalError]
// (populated by the codec's [codecs.Codec.ClassifyTerminalError]), the
// failure path lifts it into ExecErr so [EmitFailureEvent] surfaces the
// typed ErrorCode on the run timeline rather than a bare INTERNAL.
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md
// (Codec Terminal-Error Classification).
// DOC: scenarios/agent-manager/docs/internal/INVARIANTS.md
// (I3 — codec-side error classifier).

package phases

import (
	"context"
	"errors"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// HandleResultInput is the explicit input to HandleResult.
type HandleResultInput struct {
	Deps      Deps
	Run       *domain.Run
	Result    *runner.ExecuteResult
	ExecErr   error
	Sandbox   sandbox.Provider
	SandboxID *uuid.UUID
}

// HandleResultOutput is the explicit output of HandleResult.
type HandleResultOutput struct {
	Outcome domain.RunOutcome
}

// HandleResult is the post-execute seam: classify the outcome, dispatch
// to the matching handler, persist the result, broadcast terminal status.
func HandleResult(ctx context.Context, in HandleResultInput) HandleResultOutput {
	if in.Result != nil && in.Result.Result != nil && in.Run != nil && in.Run.ResolvedConfig != nil && in.Deps.StructuredResults != nil {
		in.Result.Result.Structured = in.Deps.StructuredResults.Resolve(ctx, in.Run.ResolvedConfig.ResultSpec, in.Result.Result)
	}
	outcome := classifyOutcome(in.ExecErr, in.Result)
	out := HandleResultOutput{Outcome: outcome}

	// Phase 3 durable lifecycle: reconcile a delayed process exit against
	// durable owner state exactly once. A stop request can persist a terminal
	// status (for example cancelled) while the runner process is still alive;
	// its later exit carries a stale in-memory snapshot and must not resurrect
	// the run to a different terminal state.
	if in.Run != nil && in.Deps.Runs != nil {
		if durable, readErr := in.Deps.Runs.Get(ctx, in.Run.ID); readErr == nil && durable != nil {
			if durable.LifecycleVersion != in.Run.LifecycleVersion {
				return out
			}
			if durable.Status.IsTerminal() {
				*in.Run = *durable
				if durable.Status == domain.RunStatusCancelled {
					out.Outcome = domain.RunOutcomeCancelled
				}
				if in.Deps.Broadcaster != nil {
					in.Deps.Broadcaster.BroadcastRunStatus(in.Run)
				}
				return out
			}
			// Durable cancellation intent recorded before termination outranks
			// the runner's own exit. Carry the intent onto the in-memory
			// snapshot and reconcile the delayed exit to cancelled, so the
			// cancellation handler owns terminal finalization exactly once.
			if durable.CancelRequestedAt != nil {
				in.Run.CancelRequestedAt = durable.CancelRequestedAt
				outcome = domain.RunOutcomeCancelled
				out.Outcome = domain.RunOutcomeCancelled
			}
		}
	}

	now := in.Deps.Now()
	in.Run.EndedAt = &now
	in.Run.UpdatedAt = now

	switch {
	case outcome.RequiresReview():
		HandleSuccessfulCompletion(ctx, in)
	case outcome.IsTerminalFailure():
		HandleFailure(ctx, in)
	case outcome == domain.RunOutcomeCancelled:
		HandleCancellation(ctx, in)
	default:
		HandleFailure(ctx, in)
	}

	// Qualification receipt: a bounded run's terminal seam is where its live
	// evidence exists. Derive and persist the reusable route-keyed receipt from
	// the run's admission record and this run's observed outcome, so a later
	// dependent delegation is admitted against real evidence instead of a
	// requested field. Provider acknowledgment stays empty unless the runner
	// exposed provider-native evidence; a limitation records that gap rather
	// than inventing a value.
	captureQualificationReceipt(in, outcome)

	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"failed to persist run result: "+err.Error())
			return out
		}
	}
	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(in.Run)
	}
	return out
}

// captureQualificationReceipt derives and persists the route-keyed
// qualification receipt for a bounded run at its terminal seam. It binds the
// run's immutable admission layers to the live evidence this run produced: the
// runtime version observed at admission, the accepted-output verdict, and the
// provider-reported usage. A run without an admission record captures nothing.
// An unobserved provider acknowledgment is recorded as a limitation instead of
// being inferred from the requested or effective layers.
func captureQualificationReceipt(in HandleResultInput, outcome domain.RunOutcome) {
	if in.Run == nil || in.Run.ResolvedConfig == nil {
		return
	}
	admission := in.Run.ResolvedConfig.Admission
	if admission == nil {
		return
	}
	ack := append([]string(nil), admission.ProviderAcknowledgment...)
	var limitations []string
	if len(ack) == 0 {
		limitations = append(limitations,
			"provider acknowledgment not exposed by the selected runner; omitted rather than inferred")
	}
	// The route identity is the role reference that actually resolved the run.
	// An explicit caller-supplied request wins; profile-based runs carry no
	// requested role, so fall back to the resolved role persisted on the run
	// config rather than emitting a receipt with an empty route, which would
	// keep the dependent-delegation gate permanently closed.
	route := strings.TrimSpace(admission.RequestedRoleRef)
	if route == "" {
		route = strings.TrimSpace(in.Run.ResolvedConfig.RoleRef)
	}
	receipt := domain.CaptureQualificationReceipt(admission, domain.QualificationEvidence{
		Route:                  route,
		RuntimeVersion:         admission.RuntimeVersion,
		ProviderAcknowledgment: ack,
		AcceptedOutput:         outcome.RequiresReview(),
		Usage:                  qualificationUsage(in.Result),
		Limitations:            limitations,
		RunID:                  in.Run.ID.String(),
		OperationID:            in.Run.IdempotencyKey,
		CapturedAt:             in.Deps.Now(),
	})
	if receipt == nil {
		return
	}
	admission.Receipt = receipt
}

// qualificationUsage converts a bounded run's execution metrics into the
// receipt's usage record. Provider-reported tokens or cost are recorded as
// measured; an all-zero result is retained as unknown and reserved, never as a
// measured zero.
func qualificationUsage(result *runner.ExecuteResult) domain.QualificationUsage {
	usage := domain.QualificationUsage{}
	if result == nil {
		usage.State = domain.QualificationUsageUnknownReserved
		usage.ReservedUnknown = true
		return usage
	}
	usage.InputTokens = int64(result.Metrics.TokensInput)
	usage.OutputTokens = int64(result.Metrics.TokensOutput)
	usage.CostUSD = result.Metrics.CostEstimateUSD
	if usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.CostUSD == 0 {
		usage.State = domain.QualificationUsageUnknownReserved
		usage.ReservedUnknown = true
		return usage
	}
	usage.State = domain.QualificationUsageMeasured
	return usage
}

// HandleSuccessfulCompletion mutates the run record for the success path:
// records summary + exit code + session id, queues recommendation
// extraction (when applicable), and runs ApplyAtRunEnd for sandboxed runs.
func HandleSuccessfulCompletion(ctx context.Context, in HandleResultInput) {
	if in.Result != nil {
		in.Run.Result = in.Result.Result
		in.Run.Summary = in.Result.Summary
		in.Run.ExitCode = &in.Result.ExitCode
		if in.Result.SessionID != "" {
			in.Run.SessionID = in.Result.SessionID
		}
	}

	if in.Run.RunMode == domain.RunModeInPlace {
		in.Run.Status = domain.RunStatusComplete
		in.Run.ApprovalState = domain.ApprovalStateNone
		domain.MarkFinalizationSkipped(in.Run, "in-place run has no sandbox to finalize", in.Deps.Now())
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
			"in-place run completed — skipping apply (no sandbox to diff)")
	} else {
		in.Run.Status = domain.RunStatusComplete
		in.Run.ApprovalState = domain.ApprovalStateNone
		cost := 0.0
		if in.Result != nil {
			cost = in.Result.Metrics.CostEstimateUSD
		}
		ApplyAtRunEnd(ctx, ApplyAtRunEndInput{
			Deps:      in.Deps,
			Run:       in.Run,
			SandboxID: in.SandboxID,
			Sandbox:   in.Sandbox,
			Outcome:   domain.ContractRunOutcomeSuccess,
			Cost:      cost,
		})
	}

	RevokeIdentityToken(in.Run)
}

// HandleFailure mutates the run record for the failure path: status →
// Failed, error message captured, failure event emitted, ApplyAtRunEnd
// invoked for sandboxed runs (which may flip to Complete on successful
// apply).
func HandleFailure(ctx context.Context, in HandleResultInput) {
	in.Run.Status = domain.RunStatusFailed
	if in.Result != nil {
		in.Run.Result = in.Result.Result
		in.Run.Summary = in.Result.Summary
	}

	if in.ExecErr != nil {
		in.Run.ErrorMsg = in.ExecErr.Error()
	} else if in.Result != nil && in.Result.ErrorMessage != "" {
		in.Run.ErrorMsg = in.Result.ErrorMessage
		in.Run.ExitCode = &in.Result.ExitCode
	}

	if in.Run.ErrorMsg != "" {
		if in.ExecErr != nil {
			if domainErr, ok := in.ExecErr.(domain.DomainError); ok {
				EmitFailureEvent(ctx, in.Deps, in.Run.ID, domainErr)
			} else {
				EmitGenericFailureEvent(ctx, in.Deps, in.Run.ID, in.ExecErr)
			}
		} else if in.Result != nil && in.Result.ErrorMessage != "" {
			EmitGenericFailureEvent(ctx, in.Deps, in.Run.ID, errors.New(in.Result.ErrorMessage))
		}
	}

	if in.Run.RunMode == domain.RunModeSandboxed {
		cost := 0.0
		if in.Result != nil {
			cost = in.Result.Metrics.CostEstimateUSD
		}
		outcome := classifyOutcome(in.ExecErr, in.Result).ToContract()
		ApplyAtRunEnd(ctx, ApplyAtRunEndInput{
			Deps:      in.Deps,
			Run:       in.Run,
			SandboxID: in.SandboxID,
			Sandbox:   in.Sandbox,
			Outcome:   outcome,
			Cost:      cost,
		})
	} else {
		domain.MarkFinalizationSkipped(in.Run, "in-place run has no sandbox to finalize", in.Deps.Now())
	}

	RevokeIdentityToken(in.Run)
}

// HandleCancellation mutates the run record for the cancel path.
func HandleCancellation(ctx context.Context, in HandleResultInput) {
	in.Run.Status = domain.RunStatusCancelled
	if in.Run.RunMode == domain.RunModeSandboxed {
		cost := 0.0
		if in.Result != nil {
			cost = in.Result.Metrics.CostEstimateUSD
		}
		ApplyAtRunEnd(ctx, ApplyAtRunEndInput{
			Deps:      in.Deps,
			Run:       in.Run,
			SandboxID: in.SandboxID,
			Sandbox:   in.Sandbox,
			Outcome:   domain.ContractRunOutcomeCancelled,
			Cost:      cost,
		})
	} else {
		domain.MarkFinalizationSkipped(in.Run, "in-place run has no sandbox to finalize", in.Deps.Now())
	}
	RevokeIdentityToken(in.Run)
}

// RevokeIdentityToken marks the run's identity token as revoked.
func RevokeIdentityToken(run *domain.Run) {
	if run == nil || run.IdentityTokenHash == "" {
		return
	}
	now := systemClock()
	run.IdentityTokenRevokedAt = &now
}

// classifyOutcome returns the domain.RunOutcome for the given execErr+result
// pair. The wasCancelled / timedOut signals are surfaced separately when the
// caller already detected those conditions; this helper handles the
// common "post-runner" case.
func classifyOutcome(execErr error, result *runner.ExecuteResult) domain.RunOutcome {
	var exitCode *int
	if result != nil {
		exitCode = &result.ExitCode
	}
	return domain.ClassifyRunOutcome(execErr, exitCode, false, false)
}

// ClassifyErrorOutcome maps setup-phase errors to RunOutcome for the
// failWithError path.
func ClassifyErrorOutcome(err error) domain.RunOutcome {
	switch err := err.(type) {
	case *domain.SandboxError:
		return domain.RunOutcomeSandboxFail
	case *domain.ConfigError:
		if err.Missing && err.Setting == "sandbox" {
			return domain.RunOutcomeSandboxFail
		}
		return domain.RunOutcomeException
	case *domain.RunnerError:
		if err.Operation == "timeout" {
			return domain.RunOutcomeTimeout
		}
		return domain.RunOutcomeRunnerFail
	default:
		return domain.RunOutcomeException
	}
}
