// Package initiatives provides CRUD operations and rollup computation for
// initiative groupings of backlog items.
package initiatives

import "swarm-manager/internal/operatingmode"

// Initiative represents a named grouping of backlog items into a coherent
// work stream. Stored as individual JSON files under .vrooli/initiatives/.
type Initiative struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	Status             string   `json:"status"`               // lifecycle/result state
	Mode               string   `json:"mode,omitempty"`       // item-level, holistic-loop, phased-plan-drain
	Priority           int      `json:"priority,omitempty"`   // 1-10, optional (0 = unprioritized)
	DependsOn          []string `json:"depends_on,omitempty"` // initiative name refs
	Items              []string `json:"items"`                // "kind/name" references
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Created            string   `json:"created"`
	Updated            string   `json:"updated"`
	Note               string   `json:"note,omitempty"`
	ArchivedAt         *string  `json:"archived_at,omitempty"`
}

// RollupStatus provides aggregated status counts for an initiative's items.
type RollupStatus struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
	Archived   int `json:"archived"`
}

// InitiativeWithRollup pairs an initiative with its computed rollup status and
// the deduped scenarios its member items target.
type InitiativeWithRollup struct {
	Initiative      Initiative   `json:"initiative"`
	Rollup          RollupStatus `json:"rollup"`
	TargetScenarios []string     `json:"target_scenarios,omitempty"`
}

// CreateRequest holds validated fields for creating a new initiative.
type CreateRequest struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	Status             string   `json:"status,omitempty"`
	Mode               string   `json:"mode,omitempty"`
	Priority           int      `json:"priority,omitempty"`
	DependsOn          []string `json:"depends_on,omitempty"`
	Items              []string `json:"items,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
}

// UpdateRequest holds validated fields for updating an existing initiative.
type UpdateRequest struct {
	Title              *string   `json:"title,omitempty"`
	Description        *string   `json:"description,omitempty"`
	Status             *string   `json:"status,omitempty"`
	Mode               *string   `json:"mode,omitempty"`
	Priority           *int      `json:"priority,omitempty"`
	DependsOn          *[]string `json:"depends_on,omitempty"`
	Items              *[]string `json:"items,omitempty"`
	AcceptanceCriteria *[]string `json:"acceptance_criteria,omitempty"`
	Note               *string   `json:"note,omitempty"`
}

// HasChanges reports whether the update request contains at least one field.
func (r UpdateRequest) HasChanges() bool {
	return r.Title != nil || r.Description != nil || r.Status != nil ||
		r.Mode != nil || r.Priority != nil || r.DependsOn != nil ||
		r.Items != nil || r.AcceptanceCriteria != nil || r.Note != nil
}

// Initiative status constants. The lifecycle mirrors backlog items:
//
//	active → in_review → review_pending → completed | failed | needs_followup
const (
	InitiativeStatusActive        = "active"
	InitiativeStatusInReview      = "in_review"
	InitiativeStatusReviewPending = "review_pending"
	InitiativeStatusCompleted     = "completed"
	InitiativeStatusFailed        = "failed"
	InitiativeStatusNeedsFollowup = "needs_followup"
)

// ValidateStatus returns true if the status string is valid.
func ValidateStatus(status string) bool {
	switch status {
	case InitiativeStatusActive,
		InitiativeStatusInReview,
		InitiativeStatusReviewPending,
		InitiativeStatusCompleted,
		InitiativeStatusFailed,
		InitiativeStatusNeedsFollowup:
		return true
	default:
		return false
	}
}

// IsUserSettableInitiativeStatus reports whether a user may set this status
// directly via Create or PATCH. Only `active` is user-settable — every other
// status is owned by the review pipeline and must flow through
// internal/initiativereview so the decision is audited.
//
// New statuses added to the enum default to NOT user-settable; a deliberate
// choice must be made to include them here.
func IsUserSettableInitiativeStatus(s string) bool {
	return s == InitiativeStatusActive
}

// UserSettableInitiativeStatusList returns the human-readable list of
// statuses a user may set via Create or PATCH. Kept in one place so error
// messages stay in sync with the whitelist.
func UserSettableInitiativeStatusList() string {
	return InitiativeStatusActive
}

// IsTerminalInitiativeStatus reports whether the status is a user-decided
// terminal state (set only through the initiative review-decide endpoint).
func IsTerminalInitiativeStatus(s string) bool {
	switch s {
	case InitiativeStatusCompleted, InitiativeStatusFailed, InitiativeStatusNeedsFollowup:
		return true
	}
	return false
}

// IsReviewInitiativeStatus reports whether the initiative is in an active
// review phase.
func IsReviewInitiativeStatus(s string) bool {
	switch s {
	case InitiativeStatusInReview, InitiativeStatusReviewPending:
		return true
	}
	return false
}

// ValidatePriority returns true if the priority is zero (unprioritized) or
// falls within the allowed 1-10 range.
func ValidatePriority(p int) bool {
	return p == 0 || (p >= 1 && p <= 10)
}

// NormalizeMode returns the canonical initiative operating mode. Blank
// historical metadata is treated as the default item-level mode.
func NormalizeMode(mode string) string {
	return string(operatingmode.NormalizeMode(mode))
}

// ValidateMode returns true if the mode string identifies a registered
// operating mode. Blank is valid because it normalizes to item-level.
func ValidateMode(mode string) bool {
	return operatingmode.ValidateMode(mode)
}

// OperatingModeList returns the human-readable list of registered initiative
// operating modes for API validation errors.
func OperatingModeList() string {
	return operatingmode.ModeList()
}

// ContextItem is the compact view of a member item inside an initiative
// context response. Full item bodies are excluded to bound payload size.
type ContextItem struct {
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Priority   int      `json:"priority"`
	DependsOn  []string `json:"depends_on,omitempty"`
	Initiative string   `json:"initiative,omitempty"`
	ArchivedAt *string  `json:"archived_at,omitempty"`
}

// InitiativeContext aggregates an initiative with the immediately relevant
// neighborhood: its member items, its direct upstream initiatives (targets
// of depends_on), and direct downstream initiatives (ones that depend_on it).
// Transitive neighbors are deliberately excluded to keep the payload bounded.
// TargetScenarios is the deduped union of all member items' acceptance_allow
// globs, resolved to scenario names.
type InitiativeContext struct {
	Initiative            Initiative    `json:"initiative"`
	Rollup                RollupStatus  `json:"rollup"`
	Items                 []ContextItem `json:"items"`
	UpstreamInitiatives   []Initiative  `json:"upstream_initiatives"`
	DownstreamInitiatives []Initiative  `json:"downstream_initiatives"`
	TargetScenarios       []string      `json:"target_scenarios,omitempty"`
}
