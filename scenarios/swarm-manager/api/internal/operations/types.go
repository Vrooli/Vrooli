package operations

import "time"

// DefaultWindow is the time window used when callers omit the `window`
// query parameter. Three hours mirrors the "last 3 hours" framing the
// Operations Center stats header presents.
const DefaultWindow = 3 * time.Hour

// MaxWindow caps the operations window to prevent unbounded payload
// growth. Callers needing older history use the per-activity / per-round
// detail pages, not this aggregate.
const MaxWindow = 24 * time.Hour

// MinWindow is the smallest accepted window. Anything below this risks an
// empty view for slow-firing lanes; raising it to a small floor surfaces
// "operator typed PT0S" as a 400 instead of a confusing empty payload.
const MinWindow = time.Minute

// OperationsView is the wire shape returned from GET /api/v1/operations.
//
// Lanes is always populated with all four canonical lanes — Investigate,
// Execute, Review, Reconcile — even when active counts are zero, because
// the UI renders four utilization bars unconditionally.
//
// Activities carries records that are currently active (pending, starting,
// running, or needs_review). RecentlyFinished carries records that
// completed within the requested window. Both lists are sorted newest
// first by RequestedAt for stable cursor semantics.
type OperationsView struct {
	Lanes            []LaneStatus  `json:"lanes"`
	Queue            QueueStatus   `json:"queue"`
	Activities       []ActivityRow `json:"activities"`
	RecentlyFinished []ActivityRow `json:"recently_finished"`
	GeneratedAt      time.Time     `json:"generated_at"`
	// Window is the effective query window the aggregator used after
	// defaults / clamping, echoed back so clients can confirm what they got.
	WindowSeconds int `json:"window_seconds"`
}

// LaneStatus mirrors execution.LaneStatus but is materialized via the
// agentactivity lane reader so the operations aggregator and governance
// status agree without an extra round-trip. Capacity comes from the
// governance lane-limits map; Queue is non-zero only for the Execute lane
// today (backlog-item processing is the only enqueueing lane).
type LaneStatus struct {
	Lane     string `json:"lane"`
	Active   int    `json:"active"`
	Capacity int    `json:"capacity"`
	Queue    int    `json:"queue"`
}

// QueueStatus reports backlog-execution queue depth. The Operations Center
// chip combines this with per-lane queue counts; today only Execute carries
// a queue, but the shape is plural so future lane-specific queues can land
// without a wire break.
type QueueStatus struct {
	Depth    int `json:"depth"`
	MaxDepth int `json:"max_depth"`
}

// ActivityRow is the canonical per-run summary the Operations Center
// renders in both the by-initiative view (nested under the initiative
// card) and the by-phase view (placed in its lane column).
//
// Row identity is RunID when set and ActivityID otherwise — runID is
// stable across status transitions and is what cancellation / detail
// links anchor on.
type ActivityRow struct {
	ActivityID      string `json:"activity_id"`
	RunID           string `json:"run_id,omitempty"`
	OwnerType       string `json:"owner_type"`
	OwnerKind       string `json:"owner_kind,omitempty"`
	OwnerName       string `json:"owner_name"`
	OwnerTitle      string `json:"owner_title,omitempty"`
	Purpose         string `json:"purpose"`
	PhaseKind       string `json:"phase_kind,omitempty"`
	Lane            string `json:"lane,omitempty"`
	Status          string `json:"status"`
	Mode            string `json:"mode,omitempty"`
	Phase           string `json:"phase,omitempty"`
	Round           int    `json:"round,omitempty"`
	InitiativeName  string `json:"initiative_name,omitempty"`
	RequestedAt     string `json:"requested_at"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
	RuntimeSeconds  int64  `json:"runtime_seconds,omitempty"`
	FailureReason   string `json:"failure_reason,omitempty"`
	RequestedBy     string `json:"requested_by,omitempty"`
	InteractionType string `json:"interaction_type,omitempty"`
}

// Filters carries the optional query-string filters passed through to the
// aggregator. Empty slices and zero values mean "no filter applied"; the
// aggregator never injects defaults beyond Window. Search (Q) is a
// case-insensitive substring match against OwnerTitle, OwnerName, and
// RunID — the three identifiers operators are likely to type.
type Filters struct {
	Window     time.Duration
	Statuses   []string
	Lanes      []string
	Modes      []string
	OwnerTypes []string
	Q          string
}

// IsActiveStatus reports whether the given activity status counts as
// "currently doing work." Mirrors agentactivity.isActiveStatus but exposed
// as a free function so the aggregator (and tests) can partition records
// without reaching into the package's unexported helpers.
func IsActiveStatus(status string) bool {
	switch status {
	case "pending", "starting", "running", "needs_review":
		return true
	default:
		return false
	}
}
