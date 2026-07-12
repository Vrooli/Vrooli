// Package remediation turns completed Test Genie execution evidence into
// durable, reviewable remediation jobs. It deliberately owns intent and
// verification history, not sandbox or agent policy.
package remediation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	JobStatusCreated             = "created"
	JobStatusLaunchPending       = "launch_pending"
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
	PhasePresentation  string         `json:"phasePresentation,omitempty"`
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
	ExecutionID      string           `json:"executionId,omitempty"`
	RunID            string           `json:"runId,omitempty"`
	StartedAt        time.Time        `json:"startedAt,omitempty"`
	CompletedAt      time.Time        `json:"completedAt,omitempty"`
	Delta            FindingDelta     `json:"delta,omitempty"`
	RequirementDelta RequirementDelta `json:"requirementDelta,omitempty"`
	Degraded         string           `json:"degraded,omitempty"`
}

// RequirementDelta is deliberately separate from finding changes. Requirement
// success requires fresh requirement evidence; an empty findings scan is never
// treated as proof that a requirement was satisfied.
type RequirementDelta struct {
	Resolved     []string `json:"resolved,omitempty"`
	Remaining    []string `json:"remaining,omitempty"`
	Skipped      []string `json:"skipped,omitempty"`
	Unverifiable []string `json:"unverifiable,omitempty"`
}

type FindingDelta struct {
	Resolved        []string `json:"resolved,omitempty"`
	Remaining       []string `json:"remaining,omitempty"`
	New             []string `json:"new,omitempty"`
	ChangedSeverity []string `json:"changedSeverity,omitempty"`
	Skipped         []string `json:"skipped,omitempty"`
	Unverifiable    []string `json:"unverifiable,omitempty"`
}

// Attempt is an immutable lifecycle event. Repeated entries for one
// idempotency key document recovery without overwriting what happened before.
type Attempt struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind"`
	State          string    `json:"state"`
	IdempotencyKey string    `json:"idempotencyKey"`
	RoleRef        string    `json:"roleRef,omitempty"`
	TaskID         string    `json:"taskId,omitempty"`
	RunID          string    `json:"runId,omitempty"`
	Detail         string    `json:"detail,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Job struct {
	ID                     string       `json:"id"`
	Scenario               string       `json:"scenario"`
	Status                 string       `json:"status"`
	Source                 Plan         `json:"source"`
	SourceHash             string       `json:"sourceHash"`
	SelectedFindingIDs     []string     `json:"selectedFindingIds"`
	SelectedRequirementIDs []string     `json:"selectedRequirementIds,omitempty"`
	SelectionHash          string       `json:"selectionHash"`
	LaunchAttempt          int          `json:"launchAttempt"`
	AdditionalContext      string       `json:"additionalContext,omitempty"`
	Attribution            Attribution  `json:"attribution,omitempty"`
	Verification           Verification `json:"verification,omitempty"`
	Attempts               []Attempt    `json:"attempts,omitempty"`
	CreatedAt              time.Time    `json:"createdAt"`
	UpdatedAt              time.Time    `json:"updatedAt"`
	CancelledAt            time.Time    `json:"cancelledAt,omitempty"`
	Failure                string       `json:"failure,omitempty"`
}

func launchIdempotencyKey(job Job) string {
	return fmt.Sprintf("test-genie/remediation/%s/launch/%d", job.ID, job.LaunchAttempt)
}

func newAttempt(kind, state, key, roleRef, detail string, now time.Time) Attempt {
	return Attempt{ID: uuid.NewString(), Kind: kind, State: state, IdempotencyKey: key, RoleRef: strings.TrimSpace(roleRef), Detail: strings.TrimSpace(detail), CreatedAt: now.UTC()}
}

func NewJob(plan Plan, selected, requirements []string, context string, now time.Time) Job {
	job := Job{ID: uuid.NewString(), Scenario: plan.Scenario, Status: JobStatusCreated, Source: plan,
		SelectedFindingIDs: normalizedIDs(selected), SelectedRequirementIDs: normalizedIDs(requirements), AdditionalContext: strings.TrimSpace(context), CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
	job.SourceHash = sourceHash(job.Source)
	job.SelectionHash = selectionHash(job.SelectedFindingIDs, job.SelectedRequirementIDs)
	return job
}

func sourceHash(source Plan) string { return stableHash(source) }

func selectionHash(findings, requirements []string) string {
	return stableHash(struct {
		Findings     []string `json:"findings"`
		Requirements []string `json:"requirements"`
	}{Findings: normalizedIDs(findings), Requirements: normalizedIDs(requirements)})
}

func stableHash(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("remediation immutable evidence cannot be encoded: " + err.Error())
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func IsActiveStatus(status string) bool {
	switch status {
	case JobStatusCreated, JobStatusLaunchPending, JobStatusRunning, JobStatusAgentCompleted, JobStatusVerificationRunning:
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
