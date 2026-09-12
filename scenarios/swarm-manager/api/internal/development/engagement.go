package development

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Snapshot struct {
	Digest      string            `json:"digest"`
	Proposal    Proposal          `json:"proposal"`
	Artifacts   []Artifact        `json:"artifacts"`
	Contents    []ArtifactContent `json:"contents"`
	GoalMessage string            `json:"goal_message"`
}

type Usage struct {
	Tokens      int64 `json:"tokens"`
	WallSeconds int64 `json:"wall_seconds"`
}

func (u Usage) valid() bool { return u.Tokens >= 0 && u.WallSeconds >= 0 }
func (u Usage) fits(ceiling Usage) bool {
	return u.valid() && u.Tokens <= ceiling.Tokens && u.WallSeconds <= ceiling.WallSeconds
}

type Approval struct {
	Digest string    `json:"digest"`
	Actor  string    `json:"actor"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

// Attempt is reserved before dispatch. A lost start response must be reconciled
// with the same owner idempotency key, never released on a network timeout.
type Attempt struct {
	Key                string     `json:"key"`
	Digest             string     `json:"digest"`
	WorkflowDigest     string     `json:"workflow_digest,omitempty"`
	GrantDigest        string     `json:"grant_digest,omitempty"`
	Mode               string     `json:"mode"`
	Reserved           Usage      `json:"reserved"`
	ExecutionID        string     `json:"execution_id,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	SettledAt          *time.Time `json:"settled_at,omitempty"`
	Used               Usage      `json:"used"`
	Checkpoint         string     `json:"checkpoint,omitempty"`
	CapabilityRevision string     `json:"capability_revision,omitempty"`
	SelectionReason    string     `json:"selection_reason,omitempty"`
}

type Evidence struct {
	OutcomeID       string    `json:"outcome_id"`
	Source          string    `json:"source"`
	ResolverID      string    `json:"resolver_id"`
	ReceiptSchema   string    `json:"receipt_schema"`
	ReceiptID       string    `json:"receipt_id"`
	Digest          string    `json:"digest"`
	ExecutionID     string    `json:"execution_id"`
	SubjectRevision string    `json:"subject_revision"`
	Cohort          string    `json:"cohort"`
	Passed          bool      `json:"passed"`
	Status          string    `json:"status"`
	ObservedAt      time.Time `json:"observed_at"`
	FreshUntil      time.Time `json:"fresh_until"`
}

// CancellationIntent is persisted before the owner transport is contacted.
// Requested and acknowledged are separate from terminal usage settlement.
type CancellationIntent struct {
	OperationID string     `json:"operation_id"`
	Reason      string     `json:"reason"`
	State       string     `json:"state"`
	RequestedAt time.Time  `json:"requested_at"`
	AckAt       *time.Time `json:"ack_at,omitempty"`
}

// CampaignCheckpoint is the reconstructable continuation contract for one
// approved engagement. It records owner identity and outcome inventory while
// remaining separate from final evidence acceptance.
type CampaignCheckpoint struct {
	Digest              string               `json:"digest"`
	ApprovalDigest      string               `json:"approval_digest"`
	AttemptKey          string               `json:"attempt_key,omitempty"`
	OwnerExecutionID    string               `json:"owner_execution_id,omitempty"`
	Pending             bool                 `json:"pending"`
	RequiredOutcomeIDs  []string             `json:"required_outcome_ids"`
	CompletedOutcomeIDs []string             `json:"completed_outcome_ids"`
	RemainingOutcomeIDs []string             `json:"remaining_outcome_ids"`
	LastOutcome         string               `json:"last_outcome,omitempty"`
	LastCheckpoint      *CheckpointReference `json:"last_checkpoint,omitempty"`
	NoProgressCycles    int                  `json:"no_progress_cycles"`
}

// CheckpointReference keeps an owner checkpoint addressable without treating
// its text as an acceptance verdict. The owner may replace it on every useful
// verification; Swarm retains the typed reference for fresh continuation.
type CheckpointReference struct {
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Digest string `json:"digest"`
}

func NewCampaignCheckpoint(approvalDigest string, outcomes []Outcome, attemptKey string) CampaignCheckpoint {
	required := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		required = append(required, outcome.ID)
	}
	sort.Strings(required)
	checkpoint := CampaignCheckpoint{ApprovalDigest: approvalDigest, AttemptKey: attemptKey, Pending: true, RequiredOutcomeIDs: required, RemainingOutcomeIDs: append([]string(nil), required...)}
	checkpoint.Digest = checkpointDigest(checkpoint)
	return checkpoint
}

func (c CampaignCheckpoint) withOwner(executionID string) CampaignCheckpoint {
	c.OwnerExecutionID, c.Pending = executionID, true
	c.Digest = checkpointDigest(c)
	return c
}

func (c CampaignCheckpoint) settled(executionID, outcome string) CampaignCheckpoint {
	c.OwnerExecutionID, c.Pending, c.LastOutcome = executionID, false, outcome
	if reference := newCheckpointReference(outcome); reference != nil {
		if c.LastCheckpoint != nil && c.LastCheckpoint.Digest == reference.Digest {
			c.NoProgressCycles++
		} else {
			c.NoProgressCycles = 0
		}
		c.LastCheckpoint = reference
	}
	c.Digest = checkpointDigest(c)
	return c
}

func newCheckpointReference(value string) *CheckpointReference {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(value))
	return &CheckpointReference{Kind: "owner-verification", Value: value, Digest: "sha256:" + hex.EncodeToString(sum[:])}
}

func checkpointDigest(checkpoint CampaignCheckpoint) string {
	checkpoint.Digest = ""
	encoded, err := json.Marshal(checkpoint)
	if err != nil {
		panic(fmt.Sprintf("marshal campaign checkpoint: %v", err))
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type Engagement struct {
	WorkItem string `json:"work_item"`
	// PlanRef is the canonical work package. WorkShape remains a compatibility
	// discriminator for the adaptive runner and is not a second planning model.
	PlanRef      *PlanReference      `json:"plan_ref,omitempty"`
	WorkShape    string              `json:"work_shape"`
	Version      int64               `json:"version"`
	Digest       string              `json:"digest"`
	Status       string              `json:"status"`
	Approvals    []Approval          `json:"approvals"`
	Attempts     []Attempt           `json:"attempts"`
	Used         Usage               `json:"used"`
	Reserved     Usage               `json:"reserved"`
	Checkpoint   string              `json:"checkpoint,omitempty"`
	Campaign     CampaignCheckpoint  `json:"campaign_checkpoint"`
	Evidence     []Evidence          `json:"evidence"`
	AcceptedBy   string              `json:"accepted_by,omitempty"`
	AcceptedAt   *time.Time          `json:"accepted_at,omitempty"`
	StopReason   string              `json:"stop_reason,omitempty"`
	Cancellation *CancellationIntent `json:"cancellation,omitempty"`
}

// EvidenceResolver is an owner boundary, not an agent-supplied verdict. Resolve
// must fetch an immutable receipt from Source; CurrentRevision must independently
// identify the artifact revision being accepted (not merely the target contract).
type EvidenceResolver interface {
	Resolve(ctx context.Context, source, receiptID string) (Evidence, error)
	CurrentRevision(ctx context.Context, scenario string) (string, error)
}
