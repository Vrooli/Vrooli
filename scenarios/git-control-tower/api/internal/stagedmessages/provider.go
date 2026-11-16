// Package stagedmessages owns immutable index subjects and message-only advisory jobs.
package stagedmessages

import (
	"context"
	"time"

	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

// DraftInput contains only the bounded immutable input the operator selected.
type DraftInput struct {
	FormatVersion         int                                 `json:"format_version,omitempty"`
	Mode                  advisoryv1.StagedMessageMode        `json:"mode,omitempty"`
	Snapshot              *advisoryv1.StagedMessageSnapshot   `json:"snapshot"`
	Contexts              []*advisoryv1.StagedMessageContext  `json:"contexts"`
	Omissions             []*advisoryv1.Omission              `json:"omissions,omitempty"`
	OperatorNote          string                              `json:"operator_note"`
	JobID                 string                              `json:"job_id,omitempty"`
	WorktreeCapture       bool                                `json:"worktree_capture,omitempty"`
	QueuedSelection       bool                                `json:"queued_selection,omitempty"`
	QueuedSelectionID     string                              `json:"queued_selection_id,omitempty"`
	QueuedSource          advisoryv1.WorktreeSelectionSource  `json:"queued_source,omitempty"`
	CapturedWorktree      map[string]FileBytes                `json:"captured_worktree,omitempty"`
	ReplaceQueueEntryID   string                              `json:"replace_queue_entry_id,omitempty"`
	ExpectedQueueRevision uint64                              `json:"expected_queue_revision,omitempty"`
	ContinuationOfJobID   string                              `json:"continuation_of_job_id,omitempty"`
	PriorResult           *advisoryv1.StagedMessageSuggestion `json:"prior_result,omitempty"`
	InventoryEligibility  map[int]bool                        `json:"-"`
	EvidenceEligibility   map[int]bool                        `json:"-"`
}

// draftInputSemantics is the single compatibility adapter for durable input
// formats. Stored inputs remain immutable; callers derive behavior from this
// view instead of rewriting old snapshots in place.
type draftInputSemantics struct {
	deepEvidence bool
	synthesis    bool
	legacyDraft  bool
}

func (input DraftInput) semantics() draftInputSemantics {
	switch input.FormatVersion {
	case 2:
		return draftInputSemantics{legacyDraft: true}
	case 3:
		return draftInputSemantics{deepEvidence: true}
	case 4:
		return draftInputSemantics{deepEvidence: true, synthesis: true}
	default:
		return draftInputSemantics{}
	}
}

func (input DraftInput) supportsDeepEvidence() bool { return input.semantics().deepEvidence }
func (input DraftInput) supportsSynthesis() bool    { return input.semantics().synthesis }

// EvidenceAuthorizer verifies the exact AM execution when its child can request
// evidence before Start has returned its handle to GCT.
type EvidenceAuthorizer interface {
	AuthorizeEvidence(context.Context, DraftInput, string, string) error
}

// ProviderReadiness is a no-store observation of the runner owner's admission
// state. Unknown is conservative and never authorizes a retry.
type ProviderReadiness struct {
	Provider   string     `json:"provider"`
	State      string     `json:"state"`
	Known      bool       `json:"known"`
	ObservedAt *time.Time `json:"observed_at,omitempty"`
	Message    string     `json:"message"`
}

type ReadinessProvider interface {
	Readiness(context.Context) ProviderReadiness
}

// EvidenceReadError means the owner execution is known but its retained result
// or the evidence needed to validate that result could not be read. Callers
// must preserve the execution identity and retry the read against the same job.
type EvidenceReadError struct {
	Cause error
}

func (e EvidenceReadError) Error() string {
	if e.Cause == nil {
		return "message execution evidence could not be read"
	}
	return "message execution evidence could not be read: " + e.Cause.Error()
}

func (e EvidenceReadError) Unwrap() error { return e.Cause }

// Execution is authoritative owner state, never a transcript-derived completion.
type Execution struct {
	ID                    string
	Status                string // queued, running, cancelling, succeeded, failed, cancelled
	Phase                 string
	Result                *advisoryv1.StagedMessageSuggestion
	Error                 string
	ErrorCode             string
	Retryable             bool
	ExecutionStarted      bool
	ExecutionStartedKnown bool
	TokensUsed            int64
	TokensKnown           bool
	ChargeMicroUSD        int64
	ChargeKnown           bool
}

// Provider is the policy-filtered Agent Manager boundary. Search MUST NOT start
// an execution. Start's request key must be honored by the execution owner.
type Provider interface {
	Search(context.Context, *advisoryv1.StagedMessageSnapshot) ([]*advisoryv1.StagedMessageContext, []*advisoryv1.Omission, error)
	Start(context.Context, DraftInput, string) (Execution, error)
	// The canonical durable input lets adapters expand compact owner results after
	// restart without retaining a second manifest store or process-local cache.
	Get(context.Context, string, DraftInput) (Execution, error)
	Wait(context.Context, string, DraftInput, time.Duration) (Execution, error)
	Cancel(context.Context, string, DraftInput) (Execution, error)
}

// DispatchCancellation is the owner-issued result of atomically fencing a
// start key and cancelling any execution already admitted under it.
type DispatchCancellation struct {
	Fenced    bool
	Execution *Execution
}

type DispatchCancellationProvider interface {
	CancelByKey(context.Context, string, DraftInput) (DispatchCancellation, error)
}
