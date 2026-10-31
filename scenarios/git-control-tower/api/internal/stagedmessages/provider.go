// Package stagedmessages owns immutable index subjects and message-only advisory jobs.
package stagedmessages

import (
	"context"
	"time"

	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

// DraftInput contains only the bounded immutable input the operator selected.
type DraftInput struct {
	FormatVersion        int                                 `json:"format_version,omitempty"`
	Snapshot             *advisoryv1.StagedMessageSnapshot   `json:"snapshot"`
	Contexts             []*advisoryv1.StagedMessageContext  `json:"contexts"`
	OperatorNote         string                              `json:"operator_note"`
	JobID                string                              `json:"job_id,omitempty"`
	ContinuationOfJobID  string                              `json:"continuation_of_job_id,omitempty"`
	PriorResult          *advisoryv1.StagedMessageSuggestion `json:"prior_result,omitempty"`
	InventoryEligibility map[int]bool                        `json:"-"`
	EvidenceEligibility  map[int]bool                        `json:"-"`
}

// EvidenceAuthorizer verifies the exact AM execution when its child can request
// evidence before Start has returned its handle to GCT.
type EvidenceAuthorizer interface {
	AuthorizeEvidence(context.Context, DraftInput, string, string) error
}

// Execution is authoritative owner state, never a transcript-derived completion.
type Execution struct {
	ID     string
	Status string // queued, running, cancelling, succeeded, failed, cancelled
	Phase  string
	Result *advisoryv1.StagedMessageSuggestion
	Error  string
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
