// Package goals provides the goal-scope domain for swarm-manager: a goal is an
// explicit set of end-state targets (backlog items and/or initiatives) whose
// transitive prerequisite closure defines the work tracked toward it. Goals
// scope the graph/board views and goal-directed execution, and are the anchor
// the ETA engine estimates against. The store mirrors the initiatives store
// pattern ({dataRoot}/goals/{name}/goal.json).
package goals

import (
	"strings"

	"swarm-manager/internal/eta"
)

// Status values for a goal.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// InitiativeTargetPrefix marks a target ref as an initiative rather than a
// backlog item. Item refs are "<kind>/<name>"; initiative refs are
// "initiative/<name>".
const InitiativeTargetPrefix = "initiative/"

// Goal is a first-class scope entity. Targets are end-state refs; the scope is
// derived (targets + transitive prerequisite closure) rather than stored.
type Goal struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Priority    int    `json:"priority,omitempty"`
	// Targets are end-state refs: "<kind>/<name>" for items or
	// "initiative/<name>" for initiatives.
	Targets []string `json:"targets"`
	// Seeded marks goals auto-created from de-facto goal tags so the UI can
	// distinguish them from operator-authored goals.
	Seeded bool `json:"seeded,omitempty"`
	// ScopeHistory records closure-size snapshots over time so scope growth
	// (creep) is surfaced, not hidden. The first entry is the baseline.
	ScopeHistory []ScopeSnapshot `json:"scope_history,omitempty"`
	Created      string          `json:"created"`
	Updated      string          `json:"updated"`
	ArchivedAt   *string         `json:"archived_at,omitempty"`
}

// ReadyGoalItem is a ready-to-run backlog item in an active goal's closure,
// carried by the continuous auto-enqueue drain. GoalPriority is the highest
// priority among the goals that have it ready.
type ReadyGoalItem struct {
	Kind         string
	Name         string
	GoalPriority int
}

// ScopeSnapshot is a point-in-time record of a goal's closure size, used for
// scope-creep tracking.
type ScopeSnapshot struct {
	At          string `json:"at"`
	TargetCount int    `json:"target_count"`
	ClosureSize int    `json:"closure_size"`
	Completed   int    `json:"completed"`
}

// IsInitiativeTarget reports whether a target ref denotes an initiative.
func IsInitiativeTarget(ref string) bool {
	return strings.HasPrefix(ref, InitiativeTargetPrefix)
}

// InitiativeName extracts the initiative name from an "initiative/<name>" ref.
func InitiativeName(ref string) string {
	return strings.TrimPrefix(ref, InitiativeTargetPrefix)
}

// CreateRequest holds fields for creating a goal.
type CreateRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Targets     []string `json:"targets,omitempty"`
	Seeded      bool     `json:"seeded,omitempty"`
}

// UpdateRequest holds optional fields for updating a goal.
type UpdateRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	Targets     *[]string `json:"targets,omitempty"`
}

// HasChanges reports whether the update carries at least one field.
func (r UpdateRequest) HasChanges() bool {
	return r.Title != nil || r.Description != nil || r.Priority != nil || r.Targets != nil
}

// GoalWithScope pairs a goal with its computed scope for API responses. ETA is
// the p50/p80 completion band for the goal's remaining closure, present only
// when an estimator is wired and the closure has items.
type GoalWithScope struct {
	Goal  Goal      `json:"goal"`
	Scope Scope     `json:"scope"`
	ETA   *eta.Band `json:"eta,omitempty"`
}
