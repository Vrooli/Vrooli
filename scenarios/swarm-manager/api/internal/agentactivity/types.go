package agentactivity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/identity"
)

// ErrBacklogItemBusy is returned when a spawn is attempted for a backlog item
// that already has an active agent (pending, starting, running, or needs_review).
var ErrBacklogItemBusy = errors.New("another agent is already active for this backlog item")

// pendingSpawnTTL is how long a pending record without a RunID is considered
// valid before being auto-failed. The spawn HTTP call typically has a 30-60s
// timeout, so 5 minutes is generous.
const pendingSpawnTTL = 5 * time.Minute

type OwnerType string

const (
	OwnerBacklog    OwnerType = "backlog"
	OwnerCapture    OwnerType = "capture"
	OwnerScenario   OwnerType = "scenario"
	OwnerInitiative OwnerType = "initiative"
)

type Purpose string

const (
	PurposeInitialize       Purpose = "initialize"
	PurposeWorkshop         Purpose = "workshop"
	PurposeFinalize         Purpose = "finalize"
	PurposeResearch         Purpose = "research"
	PurposeProcess          Purpose = "process"
	PurposeFixup            Purpose = "fixup"
	PurposeFollowUp         Purpose = "followup"
	PurposeSpecSync         Purpose = "spec_sync"
	PurposeClassify         Purpose = "classify"
	PurposeClarify          Purpose = "clarify"
	PurposeReview           Purpose = "review"
	PurposeFeedback         Purpose = "feedback"
	PurposeFeedbackContinue Purpose = "feedback_continue"
	PurposeInitiativeReview Purpose = "initiative_review"

	PurposeHolisticLoopInvestigate Purpose = "holistic_loop_investigate"
	PurposeHolisticLoopPlan        Purpose = "holistic_loop_plan"
	PurposeHolisticLoopExecute     Purpose = "holistic_loop_execute"
	PurposeHolisticLoopReview      Purpose = "holistic_loop_review"

	PurposePhasedPlanPrepare          Purpose = "phased_plan_prepare"
	PurposePhasedPlanExecuteNext      Purpose = "phased_plan_execute_next"
	PurposePhasedPlanClassifyProgress Purpose = "phased_plan_classify_progress"
	PurposePhasedPlanReview           Purpose = "phased_plan_review"
)

type InteractionType string

const (
	InteractionSpawn    InteractionType = "spawn"
	InteractionContinue InteractionType = "continue"
)

type Status string

const (
	StatusPending     Status = "pending"
	StatusStarting    Status = "starting"
	StatusRunning     Status = "running"
	StatusNeedsReview Status = "needs_review"
	StatusComplete    Status = "complete"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
	StatusUnspecified Status = "unspecified"
)

type Record struct {
	ActivityID      string            `json:"activity_id"`
	OwnerType       OwnerType         `json:"owner_type"`
	OwnerKind       string            `json:"owner_kind,omitempty"`
	OwnerName       string            `json:"owner_name"`
	OwnerTitle      string            `json:"owner_title,omitempty"`
	ExecutionID     string            `json:"execution_id,omitempty"`
	Purpose         Purpose           `json:"purpose"`
	InteractionType InteractionType   `json:"interaction_type"`
	TaskID          string            `json:"task_id,omitempty"`
	RunID           string            `json:"run_id,omitempty"`
	Status          Status            `json:"status"`
	RequestedAt     string            `json:"requested_at"`
	StartedAt       string            `json:"started_at,omitempty"`
	FinishedAt      string            `json:"finished_at,omitempty"`
	FailureReason   string            `json:"failure_reason,omitempty"`
	RequestedBy     string            `json:"requested_by,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	UpdatedAt       string            `json:"updated_at"`
}

type Spec struct {
	OwnerType   OwnerType
	OwnerKind   string
	OwnerName   string
	OwnerTitle  string
	ExecutionID string
	Purpose     Purpose
	RequestedBy string
	Metadata    map[string]string
}

type ListFilters struct {
	OwnerType   string
	OwnerKind   string
	OwnerName   string
	ExecutionID string
	Purpose     string
	Status      string
	RunID       string
	ActiveOnly  bool
}

func (s Spec) normalized() (Spec, error) {
	s.OwnerType = OwnerType(strings.ToLower(strings.TrimSpace(string(s.OwnerType))))
	s.OwnerKind = strings.ToLower(strings.TrimSpace(s.OwnerKind))
	s.OwnerName = strings.TrimSpace(s.OwnerName)
	s.OwnerTitle = strings.TrimSpace(s.OwnerTitle)
	s.ExecutionID = strings.TrimSpace(s.ExecutionID)
	s.Purpose = Purpose(strings.ToLower(strings.TrimSpace(string(s.Purpose))))
	s.RequestedBy = strings.TrimSpace(s.RequestedBy)
	if s.Metadata == nil {
		s.Metadata = map[string]string{}
	}

	switch s.OwnerType {
	case OwnerBacklog, OwnerCapture, OwnerScenario, OwnerInitiative:
	default:
		return Spec{}, fmt.Errorf("owner_type must be backlog, capture, scenario, or initiative")
	}

	switch s.Purpose {
	case PurposeInitialize, PurposeWorkshop, PurposeFinalize, PurposeResearch, PurposeProcess,
		PurposeFixup, PurposeFollowUp, PurposeSpecSync, PurposeClassify, PurposeClarify, PurposeReview,
		PurposeFeedback, PurposeFeedbackContinue, PurposeInitiativeReview,
		PurposeHolisticLoopInvestigate, PurposeHolisticLoopPlan, PurposeHolisticLoopExecute, PurposeHolisticLoopReview,
		PurposePhasedPlanPrepare, PurposePhasedPlanExecuteNext, PurposePhasedPlanClassifyProgress, PurposePhasedPlanReview:
	default:
		return Spec{}, fmt.Errorf("purpose is required")
	}

	if s.OwnerName == "" {
		return Spec{}, fmt.Errorf("owner_name is required")
	}
	if s.OwnerType == OwnerBacklog && s.OwnerKind == "" {
		return Spec{}, fmt.Errorf("owner_kind is required for backlog activity")
	}
	if s.RequestedBy == "" {
		s.RequestedBy = "swarm-manager"
	}

	copied := make(map[string]string, len(s.Metadata))
	for key, value := range s.Metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		copied[trimmedKey] = strings.TrimSpace(value)
	}
	s.Metadata = copied
	return s, nil
}

func isActiveStatus(status Status) bool {
	switch status {
	case StatusPending, StatusStarting, StatusRunning, StatusNeedsReview:
		return true
	default:
		return false
	}
}

// isPendingStale returns true when a pending record has no RunID and its
// RequestedAt timestamp is older than pendingSpawnTTL. Such records indicate
// a spawn HTTP call that hung or a process crash before the result was recorded.
func isPendingStale(rec Record) bool {
	if rec.Status != StatusPending || strings.TrimSpace(rec.RunID) != "" {
		return false
	}
	requested, err := time.Parse(time.RFC3339, rec.RequestedAt)
	if err != nil {
		return false
	}
	return time.Since(requested) > pendingSpawnTTL
}

type contextKey struct{}

func WithSpec(ctx context.Context, spec Spec) context.Context {
	return context.WithValue(ctx, contextKey{}, spec)
}

func specFromContext(ctx context.Context) (Spec, error) {
	spec, ok := ctx.Value(contextKey{}).(Spec)
	if !ok {
		return Spec{}, fmt.Errorf("agent activity spec missing from context")
	}
	normalized, err := spec.normalized()
	if err != nil {
		return Spec{}, err
	}
	// If RequestedBy was defaulted to "swarm-manager" but the request context
	// carries agent provenance, use the agent identity instead.
	if normalized.RequestedBy == "swarm-manager" {
		prov := identity.FromContext(ctx)
		if prov.IsAgent() {
			normalized.RequestedBy = prov.FormatStartedBy()
		}
	}
	return normalized, nil
}
