// Package plans is the structured-plan SSOT domain. It owns the first-class
// Plan + Phase record (see docs/concepts/PLAN-MODEL.md): CRUD, the deterministic
// markdown *view* (rendered from the record, never parsed back), lifecycle
// status (COMPUTED from the phase-status set), the supersession/dependency graph
// (via content_hash + plan_edges), per-surface templates, and the plan-source
// resolver (canonical write under the ~/.vrooli home store + import/migrate from
// the hygiene-blessed fallback locations).
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → Repository
//	            ↑           ↑ (faked in tests)
//	        (proto edge)  owned home store (SQLite)
//
// The proto wire types live one floor up (handlers/plans) and never import this
// package; the handler is the only translation point (api-steer §7). Authoring,
// execution and validation operate *on* plans but live in their own domains.
package plans

// PlanStatus is the lifecycle state of a plan. COMPUTED from the phase-status
// set plus lifecycle actions; never free-text edited.
type PlanStatus string

const (
	PlanStatusDraft    PlanStatus = "draft"
	PlanStatusActive   PlanStatus = "active"
	PlanStatusComplete PlanStatus = "complete"
	PlanStatusArchived PlanStatus = "archived"
)

// PhaseStatus is the state of a single first-class phase.
type PhaseStatus string

const (
	PhaseStatusTodo    PhaseStatus = "todo"
	PhaseStatusActive  PhaseStatus = "active"
	PhaseStatusDone    PhaseStatus = "done"
	PhaseStatusBlocked PhaseStatus = "blocked"
)

// StalenessTier is the computed validity of referenced code since authoring.
type StalenessTier string

const (
	StalenessFresh           StalenessTier = "fresh"
	StalenessLightlyStale    StalenessTier = "lightly_stale"
	StalenessDefinitelyStale StalenessTier = "definitely_stale"
	StalenessUnknown         StalenessTier = "" // unset / degraded
)

// ReferenceKind is the machine-readable locator family.
type ReferenceKind string

const (
	ReferenceCode ReferenceKind = "code"
	ReferenceReq  ReferenceKind = "req"
	ReferenceDoc  ReferenceKind = "doc"
)

// ReferenceResolution is whether a reference resolves against code-facts.
type ReferenceResolution string

const (
	ResolutionUnspecified ReferenceResolution = ""
	ResolutionResolved    ReferenceResolution = "resolved"
	ResolutionUnresolved  ReferenceResolution = "unresolved"
	ResolutionFuture      ReferenceResolution = "future"
	ResolutionMissing     ReferenceResolution = "missing"
)

// Reference is one connected-code locator on a plan or phase. kind/target/future
// are AUTHORED; resolution/staleness/change_factor are filled by the validation
// domain.
type Reference struct {
	ID           string
	Kind         ReferenceKind
	Target       string
	Future       bool
	Resolution   ReferenceResolution
	Staleness    StalenessTier
	ChangeFactor float64
	Note         string
}

// RegressionAnchor is the "before" anchor captured prior to changes. AUTO-FILLED
// by the authoring wizard (delegating to git-control-tower).
type RegressionAnchor struct {
	Strategy       string
	Scenario       string
	BaselineName   string
	HeadSha        string
	AllowlistPaths []string
	Commands       []string
	CapturedAt     string
	Unavailable    bool
}

// Phase is a first-class phase. order/title/intent/required_reading/reminders/
// acceptance/references are AUTHORED; baseline_scope/status are COMPUTED.
// decisions/findings/last_validation are owned by other domains and are joined
// in at their read paths (this domain stores only the structural fields).
type Phase struct {
	ID              string
	Order           int
	Title           string
	Intent          string
	RequiredReading []string
	Reminders       []string
	BaselineScope   []string
	Acceptance      string
	Status          PhaseStatus
	References      []Reference
}

// Plan is the top-level structured record.
type Plan struct {
	ID               string
	Slug             string
	Title            string
	Status           PlanStatus
	ContentHash      string
	CreatedAt        string
	UpdatedAt        string
	Purpose          string
	Scope            string
	Constraints      string
	NonGoals         string
	References       []Reference
	RegressionAnchor RegressionAnchor
	DefinitionOfDone string
	Phases           []Phase
	Supersedes       []string
	SupersededBy     []string
}

// PlanEdge is one supersession/dependency edge between two plans (the plan
// graph).
type PlanEdge struct {
	FromPlanID string
	ToPlanID   string
	Kind       string // "supersedes" | "depends_on"
}

// PlanTemplate is a per-surface starter plan that pre-scaffolds the phase
// skeleton so a plan starts with the right shape.
type PlanTemplate struct {
	ID          string
	Name        string
	Description string
	Surface     string // "cli" | "proto" | "ui" | "generic"
}

// ListFilter narrows ListPlans. A zero value matches all non-archived plans.
type ListFilter struct {
	Status          PlanStatus
	IncludeArchived bool
}
