// Package remediation turns completed Test Genie execution evidence into
// durable, reviewable remediation jobs. It deliberately owns intent and
// verification history, not sandbox or agent policy.
package remediation

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	JobStatusCreated             = "created"
	JobStatusRunning             = "running"
	JobStatusAgentCompleted      = "agent_completed"
	JobStatusVerificationRunning = "verification_running"
	JobStatusVerified            = "verified"
	JobStatusFailed              = "failed"
	JobStatusCancelled           = "cancelled"
	JobStatusDegraded            = "degraded"
)

var (
	ErrInvalidSelector = errors.New("invalid remediation selector")
	ErrActiveJob       = errors.New("an active remediation job already exists for this scenario")
	ErrInvalidState    = errors.New("invalid remediation job lifecycle transition")
	ErrNotFound        = errors.New("remediation job not found")
)

// Phase describes the immutable, descriptor-backed context of a source run.
type Phase struct {
	Name               string         `json:"name"`
	DisplayName        string         `json:"displayName,omitempty"`
	Provider           string         `json:"provider,omitempty"`
	DocsPath           string         `json:"docsPath,omitempty"`
	Status             string         `json:"status"`
	RunnabilityVerdict string         `json:"runnabilityVerdict,omitempty"`
	RunnabilityReason  string         `json:"runnabilityReason,omitempty"`
	Remediation        string         `json:"remediation,omitempty"`
	MaturityStanding   string         `json:"maturityStanding,omitempty"`
	ResultGating       string         `json:"resultGating,omitempty"`
	FindingsSummary    FindingSummary `json:"findingsSummary,omitempty"`
}

type FindingSummary struct {
	Total    int `json:"total,omitempty"`
	Blockers int `json:"blockers,omitempty"`
	Errors   int `json:"errors,omitempty"`
	Warnings int `json:"warnings,omitempty"`
	Infos    int `json:"infos,omitempty"`
}

// Finding is a portable copy of a normalized ArchitectureFinding. StableID is
// the sole identity used for selection and verification.
type Finding struct {
	StableID    string       `json:"stableId"`
	Code        string       `json:"code"`
	Source      string       `json:"source,omitempty"`
	Severity    string       `json:"severity"`
	Class       string       `json:"class"`
	Locations   []string     `json:"locations,omitempty"`
	Domains     []string     `json:"domains,omitempty"`
	Message     string       `json:"message,omitempty"`
	Suggestion  string       `json:"suggestion,omitempty"`
	Effort      string       `json:"effort,omitempty"`
	Phase       string       `json:"phase"`
	Gating      bool         `json:"gating"`
	Occurrences []Occurrence `json:"occurrences"`
}

type Occurrence struct {
	Phase     string   `json:"phase"`
	Locations []string `json:"locations,omitempty"`
}

type Bundle struct {
	ID         string   `json:"id"`
	Reason     string   `json:"reason"`
	FindingIDs []string `json:"findingIds"`
	PhaseNames []string `json:"phaseNames"`
	Rank       int      `json:"rank"`
	Gating     bool     `json:"gating"`
}

// RequirementEvidence keeps requirement status and validation references in
// the same immutable source snapshot as execution findings.
type RequirementEvidence struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	LiveStatus  string   `json:"liveStatus,omitempty"`
	Criticality string   `json:"criticality,omitempty"`
	Validations []string `json:"validations,omitempty"`
}

// Plan is an immutable server-built view of one completed execution.
type Plan struct {
	SourceExecutionID string                `json:"sourceExecutionId"`
	SourceRunID       string                `json:"sourceRunId"`
	Scenario          string                `json:"scenario"`
	CreatedAt         time.Time             `json:"createdAt"`
	Phases            []Phase               `json:"phases"`
	Findings          []Finding             `json:"findings"`
	Bundles           []Bundle              `json:"bundles"`
	Requirements      []RequirementEvidence `json:"requirements,omitempty"`
	Degraded          bool                  `json:"degraded"`
	DegradedReasons   []string              `json:"degradedReasons,omitempty"`
}

type Attribution struct {
	TaskID          string `json:"taskId,omitempty"`
	RunID           string `json:"runId,omitempty"`
	RoleRef         string `json:"roleRef,omitempty"`
	ResolvedProfile string `json:"resolvedProfile,omitempty"`
	OutputReference string `json:"outputReference,omitempty"`
}

type Verification struct {
	ExecutionID string       `json:"executionId,omitempty"`
	RunID       string       `json:"runId,omitempty"`
	StartedAt   time.Time    `json:"startedAt,omitempty"`
	CompletedAt time.Time    `json:"completedAt,omitempty"`
	Delta       FindingDelta `json:"delta,omitempty"`
	Degraded    string       `json:"degraded,omitempty"`
}

type FindingDelta struct {
	Resolved        []string `json:"resolved,omitempty"`
	Remaining       []string `json:"remaining,omitempty"`
	New             []string `json:"new,omitempty"`
	ChangedSeverity []string `json:"changedSeverity,omitempty"`
	Skipped         []string `json:"skipped,omitempty"`
	Unverifiable    []string `json:"unverifiable,omitempty"`
}

type Job struct {
	ID                     string       `json:"id"`
	Scenario               string       `json:"scenario"`
	Status                 string       `json:"status"`
	Source                 Plan         `json:"source"`
	SelectedFindingIDs     []string     `json:"selectedFindingIds"`
	SelectedRequirementIDs []string     `json:"selectedRequirementIds,omitempty"`
	AdditionalContext      string       `json:"additionalContext,omitempty"`
	Attribution            Attribution  `json:"attribution,omitempty"`
	Verification           Verification `json:"verification,omitempty"`
	CreatedAt              time.Time    `json:"createdAt"`
	UpdatedAt              time.Time    `json:"updatedAt"`
	CancelledAt            time.Time    `json:"cancelledAt,omitempty"`
	Failure                string       `json:"failure,omitempty"`
}

func NewJob(plan Plan, selected, requirements []string, context string, now time.Time) Job {
	return Job{ID: uuid.NewString(), Scenario: plan.Scenario, Status: JobStatusCreated, Source: plan,
		SelectedFindingIDs: normalizedIDs(selected), SelectedRequirementIDs: normalizedIDs(requirements), AdditionalContext: strings.TrimSpace(context), CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
}

func IsActiveStatus(status string) bool {
	switch status {
	case JobStatusCreated, JobStatusRunning, JobStatusAgentCompleted, JobStatusVerificationRunning:
		return true
	default:
		return false
	}
}

func normalizedIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
