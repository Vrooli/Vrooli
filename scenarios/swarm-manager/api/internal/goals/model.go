// Package goals provides the goal-scope domain for swarm-manager: a goal is an
// explicit set of end-state backlog item targets whose
// transitive prerequisite closure defines the work tracked toward it. Goals
// scope the graph/board views and goal-directed execution, and are the anchor
// the ETA engine estimates against. The store mirrors the initiatives store
// pattern ({dataRoot}/goals/{name}/goal.json).
package goals

import (
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eta"
)

// Status values for a goal.
const (
	StatusActive = "active"
	// StatusAchieved is an operator-only terminal claim: it means every
	// non-archived milestone has been independently verified delivered.
	StatusAchieved = "achieved"
	StatusArchived = "archived"
)

// Goal is a first-class scope entity. Targets are end-state refs; the scope is
// derived (targets + transitive prerequisite closure) rather than stored.
type Goal struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	// Priority uses the same polarity as backlog priority: lower values are
	// more urgent and sort first (0 highest, 10 lowest).
	Priority int `json:"priority,omitempty"`
	// Targets are end-state item refs: "<kind>/<name>".
	Targets []string `json:"targets"`
	// Seeded marks goals auto-created from de-facto goal tags so the UI can
	// distinguish them from operator-authored goals.
	Seeded      bool   `json:"seeded,omitempty"`
	SpawnedFrom string `json:"spawned_from,omitempty"`
	// ScopeHistory records closure-size snapshots over time so scope growth
	// (creep) is surfaced, not hidden. The first entry is the baseline.
	ScopeHistory []ScopeSnapshot `json:"scope_history,omitempty"`
	// Milestones are an optional owned partition of this goal's derived scope.
	// They never alter the target set or dependency closure.
	Milestones []Milestone `json:"milestones,omitempty"`
	Created    string      `json:"created"`
	Updated    string      `json:"updated"`
	ArchivedAt *string     `json:"archived_at,omitempty"`
}

// Milestone is an owned, non-nestable subdivision of a goal. It is serialized
// with its goal so membership and acceptance criteria update atomically.
type Milestone struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	Items              []string `json:"items,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	DependsOn          []string `json:"depends_on,omitempty"`
	SpawnedFrom        string   `json:"spawned_from,omitempty"`
	ArchivedAt         *string  `json:"archived_at,omitempty"`
	// VerifiedDeliveredAt is written only when the milestone-review workflow
	// returns the delivered verdict. Item terminal statuses are not evidence.
	VerifiedDeliveredAt *string            `json:"verified_delivered_at,omitempty"`
	CriterionVerdicts   []CriterionVerdict `json:"criterion_verdicts,omitempty"`
}

// CriterionVerdict is the durable proof summary from milestone review.
type CriterionVerdict struct {
	Criterion string   `json:"criterion"`
	Verdict   string   `json:"verdict"`
	Evidence  []string `json:"evidence,omitempty"`
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

// CreateRequest holds fields for creating a goal.
type CreateRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Targets     []string `json:"targets,omitempty"`
	Seeded      bool     `json:"seeded,omitempty"`
	SpawnedFrom string   `json:"spawned_from,omitempty"`
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
	// ScopeEntities hydrates the refs the goal detail view renders. Attached
	// by Get (the detail read) only, so List stays light.
	ScopeEntities *ScopeEntities `json:"scope_entities,omitempty"`
}

// ScopeEntities is read-time hydration for a goal's rendered item refs.
// The detail UI reuses its standard cards from this instead of joining the
// list endpoints, which window and filter items out. Derived, never stored —
// the goal itself keeps only refs so nothing here can go stale on disk.
type ScopeEntities struct {
	Items map[string]backlog.BacklogItem `json:"items,omitempty"`
}
