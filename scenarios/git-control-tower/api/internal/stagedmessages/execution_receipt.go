package stagedmessages

import (
	"context"
	"strings"

	"git-control-tower/internal/reviewjobs"
	"google.golang.org/protobuf/encoding/protojson"
)

// applyExecutionReceipt projects one authoritative owner receipt into the next
// durable job value. The caller keeps revision-checked persistence in reconcile.
func (s *Service) applyExecutionReceipt(ctx context.Context, job reviewjobs.Job, input *DraftInput, ex Execution) (reviewjobs.Job, error) {
	next := job
	maintenanceRefusal := ex.ErrorCode == "provider_admission_maintenance" && (!ex.ExecutionStartedKnown || !ex.ExecutionStarted)
	applyExecutionReceiptFields(&next, job, ex, maintenanceRefusal)
	if next.ExecutionID == "" {
		return transitionWithoutExecution(next, job, maintenanceRefusal)
	}

	target := executionReceiptState(job, ex, &next, maintenanceRefusal)
	if target == reviewjobs.Succeeded {
		valid, err := s.prepareSuccessfulExecutionResult(ctx, job, input, ex, &next)
		if err != nil {
			return reviewjobs.Job{}, err
		}
		if !valid {
			target = reviewjobs.Failed
		}
	}
	if err := next.Transition(target); err != nil {
		return reviewjobs.Job{}, err
	}
	return next, nil
}

func applyExecutionReceiptFields(next *reviewjobs.Job, job reviewjobs.Job, ex Execution, maintenanceRefusal bool) {
	if strings.TrimSpace(ex.Phase) != "" {
		next.Phase = ex.Phase
	}
	next.ExecutionID = ex.ID
	if next.ExecutionID == "" {
		next.ExecutionID = job.ExecutionID
	}
	next.Error = ex.Error
	// A later authoritative owner read supersedes a prior transient projection
	// error. Do not leave a recovered job carrying a stale failure category.
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
}

func transitionWithoutExecution(next reviewjobs.Job, job reviewjobs.Job, maintenanceRefusal bool) (reviewjobs.Job, error) {
	target := reviewjobs.Failed
	if maintenanceRefusal && !job.CancelRequested {
		target = reviewjobs.AwaitingProvider
	}
	if err := next.Transition(target); err != nil {
		return reviewjobs.Job{}, err
	}
	if target == reviewjobs.Failed {
		next.Error = "execution owner returned no handle"
	}
	return next, nil
}

func (s *Service) prepareSuccessfulExecutionResult(ctx context.Context, job reviewjobs.Job, input *DraftInput, ex Execution, next *reviewjobs.Job) (bool, error) {
	if input.supportsDeepEvidence() {
		current, err := s.store.Get(ctx, job.ID)
		if err != nil {
			return false, err
		}
		input.EvidenceEligibility = suppliedEvidence(current.Checks, input.Snapshot.Digest, ex.ID)
		input.InventoryEligibility = suppliedInventory(current.Checks, input.Snapshot.Digest, ex.ID)
	}
	ConstrainSynthesisScope(*input, ex.Result)
	if err := validateResult(*input, ex.Result); err != nil {
		next.ErrorCode = "invalid_structured_output"
		next.Retryable = true
		next.Error = err.Error()
		return false, nil
	}
	data, _ := protojson.Marshal(ex.Result)
	next.ResultJSON = string(data)
	return true, nil
}

func executionReceiptState(job reviewjobs.Job, ex Execution, next *reviewjobs.Job, maintenanceRefusal bool) reviewjobs.State {
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
			// Unknown owner status is not evidence of a failed run. Keep the last
			// durable state and expose the unsupported receipt.
			target = job.State
			next.ErrorCode = "unknown_owner_status"
			next.Error = "Agent Manager returned an unsupported execution state. The last confirmed job state is retained."
			next.Retryable = true
		}
	}
	if !job.CancelRequested && maintenanceRefusal {
		target = reviewjobs.AwaitingProvider
	}
	return target
}
