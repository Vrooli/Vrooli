package agentactivity

import (
	"context"
	"fmt"
	"strings"
)

type OwnerType string

const (
	OwnerBacklog  OwnerType = "backlog"
	OwnerCapture  OwnerType = "capture"
	OwnerScenario OwnerType = "scenario"
)

type Purpose string

const (
	PurposeInitialize Purpose = "initialize"
	PurposeWorkshop   Purpose = "workshop"
	PurposeFinalize   Purpose = "finalize"
	PurposeResearch   Purpose = "research"
	PurposeProcess    Purpose = "process"
	PurposeFixup      Purpose = "fixup"
	PurposeFollowUp   Purpose = "followup"
	PurposeSpecSync   Purpose = "spec_sync"
	PurposeClassify   Purpose = "classify"
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
	case OwnerBacklog, OwnerCapture, OwnerScenario:
	default:
		return Spec{}, fmt.Errorf("owner_type must be backlog, capture, or scenario")
	}

	switch s.Purpose {
	case PurposeInitialize, PurposeWorkshop, PurposeFinalize, PurposeResearch, PurposeProcess,
		PurposeFixup, PurposeFollowUp, PurposeSpecSync, PurposeClassify:
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

type contextKey struct{}

func WithSpec(ctx context.Context, spec Spec) context.Context {
	return context.WithValue(ctx, contextKey{}, spec)
}

func specFromContext(ctx context.Context) (Spec, error) {
	spec, ok := ctx.Value(contextKey{}).(Spec)
	if !ok {
		return Spec{}, fmt.Errorf("agent activity spec missing from context")
	}
	return spec.normalized()
}

