package operations

import (
	"encoding/json"
	"fmt"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// StandingSchemaVersion is the wire schema of Standing.
const StandingSchemaVersion = "1"

// Standing is the durable status contract every client consumes (REST,
// Connect and CLI render the same fields). It is a pure projection of the
// operation row: nothing here is inferred from elapsed time.
type Standing struct {
	SchemaVersion   string                  `json:"schema_version"`
	OperationID     string                  `json:"operation_id"`
	DeploymentID    string                  `json:"deployment_id"`
	RequestKey      string                  `json:"request_key"`
	PlanDigest      string                  `json:"plan_digest"`
	State           State                   `json:"state"`
	Terminal        bool                    `json:"terminal"`
	Fence           uint64                  `json:"fence"`
	WorkerID        string                  `json:"worker_id,omitempty"`
	LeaseExpiresAt  string                  `json:"lease_expires_at,omitempty"`
	HeartbeatAt     string                  `json:"heartbeat_at,omitempty"`
	CancelRequested bool                    `json:"cancel_requested"`
	ActiveStep      string                  `json:"active_step,omitempty"`
	CompletedSteps  []string                `json:"completed_steps"`
	StepReceipts    []domain.StepReceipt    `json:"step_receipts"`
	UnknownEffects  []domain.UnknownEffect  `json:"unknown_effects"`
	Result          *domain.OperationResult `json:"result,omitempty"`
	Error           *apierrors.Error        `json:"error,omitempty"`
	NextAction      *apierrors.NextAction   `json:"next_action,omitempty"`
	ReattachCommand string                  `json:"reattach_command"`
	// StillPending is set by wait when the observer's bound elapsed first.
	StillPending                bool   `json:"still_pending,omitempty"`
	RecommendedNextCheckSeconds int32  `json:"recommended_next_check_seconds,omitempty"`
	CreatedAt                   string `json:"created_at"`
	UpdatedAt                   string `json:"updated_at"`
	TerminalAt                  string `json:"terminal_at,omitempty"`
}

// StandingOf projects an operation row.
func StandingOf(op *domain.CloudOperation) Standing {
	s := Standing{
		SchemaVersion:   StandingSchemaVersion,
		OperationID:     op.ID,
		DeploymentID:    op.DeploymentID,
		RequestKey:      op.RequestKey,
		PlanDigest:      op.PlanDigest,
		State:           op.State,
		Terminal:        op.State.IsTerminal(),
		Fence:           op.Fence,
		WorkerID:        op.WorkerID,
		CancelRequested: op.CancelRequested,
		CompletedSteps:  []string{},
		StepReceipts:    []domain.StepReceipt{},
		UnknownEffects:  []domain.UnknownEffect{},
		ReattachCommand: fmt.Sprintf("scenario-to-cloud operation wait %s", op.ID),
		CreatedAt:       op.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:       op.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if op.LeaseExpiresAt != nil {
		s.LeaseExpiresAt = op.LeaseExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if op.HeartbeatAt != nil {
		s.HeartbeatAt = op.HeartbeatAt.UTC().Format(time.RFC3339Nano)
	}
	if op.TerminalAt != nil {
		s.TerminalAt = op.TerminalAt.UTC().Format(time.RFC3339Nano)
	}
	if marker := op.ActiveStepMarker(); marker != nil {
		s.ActiveStep = marker.Step
	}
	if receipts, err := op.Receipts(); err == nil && receipts != nil {
		s.StepReceipts = receipts
		for _, r := range receipts {
			if r.Outcome == StepSucceeded || r.Outcome == StepUnchanged || r.Outcome == StepSkipped {
				s.CompletedSteps = append(s.CompletedSteps, r.Step)
			}
		}
	}
	if effects, err := op.UnknownEffectList(); err == nil && effects != nil {
		s.UnknownEffects = effects
	}
	s.Result = op.ResultValue()
	if len(op.Error) > 0 && string(op.Error) != "null" {
		var typed apierrors.Error
		if err := json.Unmarshal(op.Error, &typed); err == nil && typed.Code != "" {
			s.Error = &typed
			if typed.NextAction != nil {
				s.NextAction = typed.NextAction
			}
		}
	}
	if s.NextAction == nil {
		s.NextAction = defaultNextAction(op, s)
	}
	return s
}

// defaultNextAction names the one thing a caller should do for a state.
func defaultNextAction(op *domain.CloudOperation, s Standing) *apierrors.NextAction {
	ref := "/api/v1/operations/" + op.ID
	switch op.State {
	case Admitted, Running, Verifying, CancelRequested:
		return &apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "wait", Reference: ref + "/wait", Label: "Wait once; the owner holds progress"}
	case WaitingInput:
		return &apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "resume", Reference: "/api/v1/deployments/" + op.DeploymentID + "/execute", Label: "Provide the missing input and resubmit under the same request key"}
	case Reconciling:
		label := "Restore reach to the target and run reconciliation"
		if len(s.UnknownEffects) > 0 {
			label = s.UnknownEffects[len(s.UnknownEffects)-1].NextAction
		}
		return &apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "reconcile", Reference: "/api/v1/operations/reconcile", Label: label}
	case Recovering, FailedRecovery:
		return &apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "recovery", Reference: "/api/v1/deployments/" + op.DeploymentID + "/recovery", Label: "Inspect the target; automatic recovery did not restore the change"}
	case Failed:
		return &apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "operation", Reference: ref, Label: "Inspect step receipts; replan and resubmit with a new request key"}
	default:
		return nil
	}
}

// Pending marks a standing returned by a timed-out observer.
func (s Standing) Pending(nextCheck time.Duration) Standing {
	s.StillPending = true
	secs := int32(nextCheck / time.Second)
	if secs < 1 {
		secs = 1
	}
	s.RecommendedNextCheckSeconds = secs
	return s
}
