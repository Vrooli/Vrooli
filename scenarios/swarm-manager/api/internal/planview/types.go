// Package planview composes topology data, dependency-wave peeling
// (depgraph.Waves), and server-owned next-action markers into the Plan-lens board
// projection: Now / Next / Later / Done.
//
// Column semantics (plan decision D2) are actionability, not agent-vs-human:
// Now is in flight (summary counts only — cards come from /api/v1/operations),
// Next is actionable immediately by agent (run/workshop) or human
// (decide/review/classify), Later is not yet actionable grouped by nearest
// blocker, Done is recent outcomes. Wave membership is ordinal (dependency
// layers from runnable), never fabricated clock time (D3).
package planview

import (
	"context"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/operations"
)

// Card types.
const (
	CardItem    = "item"
	CardGate    = "gate"
	CardOutcome = "outcome"
)

// Card actions.
const (
	ActionRun      = "run"
	ActionWorkshop = "workshop"
	ActionFinalize = "finalize"
	ActionDecide   = "decide"
	ActionReview   = "review"
	ActionClassify = "classify"
	ActionNone     = "none"
)

// Blocker kinds for Later groups.
const (
	BlockerNone  = "none"
	BlockerGate  = "gate"
	BlockerItems = "items"
	BlockerCycle = "cycle"
)

// Done outcomes.
const (
	OutcomeOK            = "ok"
	OutcomeFailed        = "failed"
	OutcomeNeedsReview   = "needs_review"
	OutcomeNeedsFollowup = "needs_followup"
	// OutcomeDropped marks work closed by operator decision rather than by a
	// run. It is distinct from OutcomeOK so the Done column does not read a
	// dropped item as something that shipped.
	OutcomeDropped = "dropped"
)

// Window bounds for the Done column (seconds).
const (
	DefaultWindowSeconds = 24 * 60 * 60
	MinWindowSeconds     = 60
	MaxWindowSeconds     = 24 * 60 * 60
	// DoneCap bounds the Done column payload.
	DoneCap = 50
)

// NowSummary carries Now-column header counts; cards come from the
// operations endpoint.
type NowSummary struct {
	ActiveCount   int          `json:"active_count"`
	QueueDepth    int          `json:"queue_depth"`
	MaxQueueDepth int          `json:"max_queue_depth"`
	Lanes         []LaneStatus `json:"lanes"`
}

// LaneStatus mirrors operations lane utilization for the Now header.
type LaneStatus struct {
	Lane     string `json:"lane"`
	Active   int    `json:"active"`
	Capacity int    `json:"capacity"`
}

// Card is one board card.
type Card struct {
	ID          string `json:"id"`
	CardType    string `json:"card_type"`
	Action      string `json:"action"`
	ItemKind    string `json:"item_kind,omitempty"`
	ItemName    string `json:"item_name,omitempty"`
	Title       string `json:"title"`
	Status      string `json:"status,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Wave        int    `json:"wave"`
	Milestone   string `json:"milestone,omitempty"`
	Effort      string `json:"effort,omitempty"`
	Gate        *Gate  `json:"gate,omitempty"`
	Outcome     string `json:"outcome,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
	Unblocks    int    `json:"unblocks"`
}

// CardGroup is an ordered group of cards within a column.
type CardGroup struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	BlockerKind string   `json:"blocker_kind"`
	GateID      string   `json:"gate_id,omitempty"`
	BlockerKeys []string `json:"blocker_keys,omitempty"`
	Cards       []Card   `json:"cards"`
}

// Column is one board column's groups.
type Column struct {
	Groups    []CardGroup `json:"groups"`
	CardCount int         `json:"card_count"`
}

// Meta carries projection metadata and cycle diagnostics.
type Meta struct {
	GeneratedAt   string   `json:"generated_at"`
	WindowSeconds int      `json:"window_seconds"`
	MaxWave       int      `json:"max_wave"`
	Cycles        []string `json:"cycles"`
	// ETA is the completion estimate for the board's remaining work, present
	// only when an estimator is wired and the board has pending items.
	ETA *eta.Band `json:"eta,omitempty"`
}

// Board is the full plan-board projection.
type Board struct {
	Now   NowSummary `json:"now"`
	Next  Column     `json:"next"`
	Later Column     `json:"later"`
	Done  Column     `json:"done"`
	Meta  Meta       `json:"meta"`
}

// Params are the projection request parameters.
type Params struct {
	// WindowSeconds bounds the Done column; clamped to [MinWindowSeconds,
	// MaxWindowSeconds], defaulting to DefaultWindowSeconds when zero.
	WindowSeconds int
	// Goal, when non-empty, scopes the board to that goal's transitive
	// prerequisite closure: only closure items (and the gates/executions that
	// belong to them) appear, and the ETA band covers the scoped work. When
	// empty the projection is identical to the unscoped board.
	Goal string
}

// BacklogLister loads backlog items.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// GateEnumerator provides the attention read-model.
type GateEnumerator interface {
	Enumerate(ctx context.Context) []Gate
}

// NextActionResolver is the server-owned action authority, keyed by item ref.
// It is optional during the gates-to-projection transition, but production
// board wiring supplies the shared projection so ordinary cards cannot drift
// from the decision stream's action semantics.
//
// The contract is whole-projection rather than per-item: the board and the
// operator inbox read the same answer, so the board consumes that answer
// instead of driving its own resolution loop over it.
type NextActionResolver interface {
	ResolveNextActions(ctx context.Context) (map[string]backlog.NextActionProjection, error)
}

// GoalAction is a goal-owned entry already resolved by the cross-entity
// next-action authority. Planview consumes it as data and never reimplements
// the goal funnel.
type GoalAction struct {
	Name     string
	Title    string
	Action   string
	Priority int
}

type GoalActionLister interface {
	ListGoalActions(ctx context.Context) ([]GoalAction, error)
}

// ExecutionLister lists execution records.
type ExecutionLister interface {
	List(ctx context.Context, filters execution.ListFilters) ([]execution.Record, error)
}

// OpsSummarizer provides the operations aggregate the Now summary reads
// (operations.Aggregator satisfies it).
type OpsSummarizer interface {
	Aggregate(ctx context.Context, f operations.Filters) (*operations.OperationsView, error)
}

// GoalScoper resolves a goal name to the item refs ("<kind>/<name>") in its
// transitive prerequisite closure, used to scope the board to a goal. Optional:
// when the Config field is nil, a goal-scoped request is rejected. *goals.Service
// satisfies it.
type GoalScoper interface {
	ClosureRefs(name string) ([]string, error)
}

// ETAEstimatorFactory builds a fresh ETA estimator for the board-wide
// completion band. It may return (nil, nil) when estimation is unavailable, in
// which case the ETA is omitted. Optional: when the Config field is nil, no ETA
// is computed.
type ETAEstimatorFactory func() (*eta.Estimator, error)

// Config wires Service dependencies. Backlog and Gates are required;
// Executions and Ops degrade gracefully when nil (empty Done executions / zero
// Now counts). Now defaults to time.Now.
type Config struct {
	Backlog     BacklogLister
	Gates       GateEnumerator
	NextActions NextActionResolver
	GoalActions GoalActionLister
	Executions  ExecutionLister
	Ops         OpsSummarizer
	Goals       GoalScoper
	ETA         ETAEstimatorFactory
	Now         func() time.Time
}
