// Package backlog types defines the core domain types, constants, and
// conversion functions for backlog items. These are pure data definitions
// with no HTTP or filesystem dependencies.
package backlog

import (
	"fmt"
	"strings"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/identity"

	repocontract "github.com/vrooli/repo-contract-go"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// BacklogStatus represents the lifecycle state of a backlog item.
//
// Two valid post-execution shapes land an item in the terminal gate:
//
//  1. Finalization-eligible items (normal execute runs):
//     in_progress → in_review → review_pending → (user) → completed|failed|needs_followup
//     The review agent gathers evidence during in_review; the user decides
//     terminal via the review-decide endpoint.
//
//  2. Non-finalization items (execution types that skip post-run checks):
//     in_progress → review_pending → (user) → completed|failed|needs_followup
//     No review agent runs, so the item skips in_review and moves directly
//     to awaiting the user's decision.
//
// Terminal status writes from the execution system are forbidden; only
// review-decide should flip review_pending → terminal (see update_patch.go
// for the validator that enforces this for user PATCH requests).
type BacklogStatus string

const (
	StatusSuggested     BacklogStatus = backlogstatus.Suggested
	StatusBacklog       BacklogStatus = backlogstatus.Backlog
	StatusResearching   BacklogStatus = backlogstatus.Researching
	StatusReady         BacklogStatus = backlogstatus.Ready
	StatusQueued        BacklogStatus = backlogstatus.Queued
	StatusInProgress    BacklogStatus = backlogstatus.InProgress
	StatusInReview      BacklogStatus = backlogstatus.InReview
	StatusReviewPending BacklogStatus = backlogstatus.ReviewPending
	StatusCompleted     BacklogStatus = backlogstatus.Completed
	StatusFailed        BacklogStatus = backlogstatus.Failed
	// StatusNeedsFollowup is a user-decided terminal state set only by
	// review-decide. It means "delivered, but more work is needed — the
	// user should schedule it." Do NOT conflate with execution.StatusNeedsFixup,
	// which is a run-level state the execution system sets to drive
	// auto-fixup; the two exist on different enums for different reasons.
	StatusNeedsFollowup BacklogStatus = backlogstatus.NeedsFollowup
)

// IsTerminalStatus reports whether the given status is a user-decided terminal
// state. Only review-decide transitions should land in these.
func IsTerminalStatus(s BacklogStatus) bool {
	return backlogstatus.IsTerminal(string(s))
}

// IsReviewStatus reports whether the item is in an active review phase
// (agent gathering evidence or waiting for user decision).
func IsReviewStatus(s BacklogStatus) bool {
	return backlogstatus.IsReview(string(s))
}

// IsArchived reports whether the backlog item has a non-empty archived_at
// timestamp and therefore should be treated as terminal/non-actionable by
// rollup, dependency, and visibility callers.
func IsArchived(item BacklogItem) bool {
	return item.ArchivedAt != nil && strings.TrimSpace(*item.ArchivedAt) != ""
}

// IsValidTransition is a permissive safety net over the backlog state
// machine: it rejects nonsensical transitions regardless of caller (e.g.,
// Completed → Ready). Individual handlers stack tighter rules on top.
// See backlogstatus.IsValidTransition for the full rule set.
func IsValidTransition(from, to BacklogStatus) bool {
	return backlogstatus.IsValidTransition(string(from), string(to))
}

// BacklogKind represents a category of backlog work.
type BacklogKind string

const (
	KindIdea     BacklogKind = "idea"
	KindResearch BacklogKind = "research"
	KindFix      BacklogKind = "fix"
	KindExecute  BacklogKind = "execute"
	KindChore    BacklogKind = "chore"
)

// backlogKindDirs maps each BacklogKind to its on-disk directory name.
// Derived from KindConfig to maintain a single source of truth.
var backlogKindDirs = func() map[BacklogKind]string {
	m := make(map[BacklogKind]string, len(KindConfig))
	for k, meta := range KindConfig {
		m[k] = meta.Dir
	}
	return m
}()

// BacklogItem represents a unit of work stored on disk.
type BacklogItem struct {
	Name            string               `json:"name"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	Status          BacklogStatus        `json:"status"`
	Priority        int                  `json:"priority"`
	Tags            []string             `json:"tags"`
	Created         string               `json:"created"`
	Updated         string               `json:"updated"`
	Kind            BacklogKind          `json:"kind"`
	DependsOn       []string             `json:"depends_on,omitempty"`
	Initiative      string               `json:"initiative,omitempty"`
	Effort          string               `json:"effort,omitempty"`
	AcceptanceAllow []string             `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string             `json:"acceptance_deny,omitempty"`
	Creates         []string             `json:"creates,omitempty"`
	SpawnedFrom     string               `json:"spawned_from,omitempty"`
	PlanRef         *PlanRef             `json:"plan_ref,omitempty"`
	FindingRef      string               `json:"finding_ref,omitempty"`
	Note            string               `json:"note,omitempty"`
	SuggestedSkills []string             `json:"suggested_skills,omitempty"`
	CreatedBy       *identity.Provenance `json:"created_by,omitempty"`
	ArchivedAt      *string              `json:"archived_at,omitempty"`
}

// PlanRef links a swarm-manager work container to a canonical plan-manager plan.
type PlanRef struct {
	Provider string `json:"provider"`
	PlanID   string `json:"plan_id"`
	Slug     string `json:"slug"`
	Role     string `json:"role"`
}

const (
	PlanRefProviderPlanManager = "plan-manager"
	PlanRefRoleExecutionSpec   = "execution_spec"
	PlanRefRoleOperatingMode   = "operating_mode_plan"
)

// BacklogFile represents a file or directory within a backlog item folder.
type BacklogFile struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"` // "file" or "directory"
	Size     int64         `json:"size,omitempty,string"`
	Children []BacklogFile `json:"children,omitempty"`
}

// protectedBacklogFileName is the spec file that cannot be renamed or deleted
// through the file operation API.
const protectedBacklogFileName = "spec.json"

// ResearchMode describes the intent for backlog agent work.
type ResearchMode string

const (
	ResearchModeWorkshop   ResearchMode = "workshop"
	ResearchModeFinalize   ResearchMode = "finalize"
	ResearchModeInitialize ResearchMode = "initialize"
)

// promptSelection holds the resolved skill ID, variables, and rendered prompt
// text returned by prompt-manager.
type promptSelection struct {
	SkillID      string
	Variables    map[string]string
	Prompt       string
	ExperimentID string
	VariantID    string
}

// ParseBacklogKind validates and normalizes a raw kind string.
func ParseBacklogKind(raw string) (BacklogKind, error) {
	candidate := BacklogKind(strings.ToLower(strings.TrimSpace(raw)))
	if _, ok := backlogKindDirs[candidate]; ok {
		return candidate, nil
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidKind, raw)
}

// validateBacklogStatus returns true if the given status string is a known
// backlog status value.
func validateBacklogStatus(status string) bool {
	return backlogstatus.IsValid(status)
}

// validateEffort checks that an effort value is one of the valid t-shirt sizes.
// Returns the normalized (uppercased) value, or an error if invalid.
func validateEffort(raw string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}
	switch value {
	case "XS", "S", "M", "L", "XL":
		return value, nil
	default:
		return "", fmt.Errorf("effort must be XS, S, M, L, or XL")
	}
}

// validateGlobs checks that each glob pattern is non-empty, relative, and
// syntactically valid.
func validateGlobs(globs []string) error {
	for i, g := range globs {
		if err := repocontract.ValidateRepoGlob(g); err != nil {
			return fmt.Errorf("glob[%d]: invalid pattern %q: %w", i, g, err)
		}
	}
	return nil
}

// sanitizeName converts a name to a folder-safe format.
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// backlogToProto converts a BacklogItem to its protobuf representation.
func backlogToProto(item BacklogItem) *domainpb.BacklogItem {
	result := &domainpb.BacklogItem{
		Name:        item.Name,
		Title:       item.Title,
		Description: item.Description,
		Status:      string(item.Status),
		Priority:    int32(item.Priority),
		Tags:        item.Tags,
		Created:     item.Created,
		Updated:     item.Updated,
		Kind:        string(item.Kind),
	}
	if len(item.DependsOn) > 0 {
		result.DependsOn = item.DependsOn
	}
	if strings.TrimSpace(item.Initiative) != "" {
		result.Initiative = &item.Initiative
	}
	if strings.TrimSpace(item.Effort) != "" {
		result.Effort = &item.Effort
	}
	if len(item.AcceptanceAllow) > 0 {
		result.AcceptanceAllow = item.AcceptanceAllow
	}
	if len(item.AcceptanceDeny) > 0 {
		result.AcceptanceDeny = item.AcceptanceDeny
	}
	if len(item.Creates) > 0 {
		result.Creates = item.Creates
	}
	if strings.TrimSpace(item.SpawnedFrom) != "" {
		result.SpawnedFrom = &item.SpawnedFrom
	}
	if item.PlanRef != nil {
		result.PlanRef = planRefToProto(item.PlanRef)
	}
	if strings.TrimSpace(item.FindingRef) != "" {
		result.FindingRef = &item.FindingRef
	}
	if strings.TrimSpace(item.Note) != "" {
		result.Note = &item.Note
	}
	if len(item.SuggestedSkills) > 0 {
		result.SuggestedSkills = item.SuggestedSkills
	}
	if item.CreatedBy != nil {
		attr := agentsessions.AttributionFromProvenance(*item.CreatedBy)
		result.CreatedBy = agentsessions.AttributionToProto(attr)
	}
	if item.ArchivedAt != nil {
		result.ArchivedAt = item.ArchivedAt
	}
	return result
}

// ToProto converts a domain BacklogItem to its protobuf representation for
// other typed service adapters that should not duplicate backlog wire mapping.
func ToProto(item BacklogItem) *domainpb.BacklogItem {
	return backlogToProto(item)
}

func planRefToProto(ref *PlanRef) *domainpb.PlanRef {
	if ref == nil {
		return nil
	}
	return &domainpb.PlanRef{
		Provider: ref.Provider,
		PlanId:   ref.PlanID,
		Slug:     ref.Slug,
		Role:     ref.Role,
	}
}

func planRefFromProto(ref *domainpb.PlanRef) *PlanRef {
	if ref == nil {
		return nil
	}
	return &PlanRef{
		Provider: strings.TrimSpace(ref.Provider),
		PlanID:   strings.TrimSpace(ref.PlanId),
		Slug:     strings.TrimSpace(ref.Slug),
		Role:     strings.TrimSpace(ref.Role),
	}
}

func normalizePlanRef(ref *PlanRef) *PlanRef {
	if ref == nil {
		return nil
	}
	normalized := &PlanRef{
		Provider: strings.TrimSpace(ref.Provider),
		PlanID:   strings.TrimSpace(ref.PlanID),
		Slug:     strings.TrimSpace(ref.Slug),
		Role:     strings.TrimSpace(ref.Role),
	}
	if normalized.Provider == "" && normalized.PlanID == "" && normalized.Slug == "" && normalized.Role == "" {
		return nil
	}
	return normalized
}

func validatePlanRef(ref *PlanRef, role string) error {
	if ref == nil {
		return nil
	}
	if ref.Provider != PlanRefProviderPlanManager {
		return fmt.Errorf("plan_ref.provider must be %q", PlanRefProviderPlanManager)
	}
	if ref.PlanID == "" {
		return fmt.Errorf("plan_ref.plan_id is required")
	}
	if ref.Slug == "" {
		return fmt.Errorf("plan_ref.slug is required")
	}
	if ref.Role != role {
		return fmt.Errorf("plan_ref.role must be %q", role)
	}
	return nil
}
