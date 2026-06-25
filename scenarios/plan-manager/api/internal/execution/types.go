// Package execution is the guided-runner domain. It links a run to a plan,
// walks the plan's phases via typed status transitions (delegating the phase
// mutation to the plans SSOT), and acts as a just-in-time context server:
// GetStatus / GetNext assemble the current phase, what is next, the phase-scoped
// required-reading + reminders, the last validation and staleness, the COMPUTED
// resume point (earliest non-done phase) and completeness (full iff every phase
// is done) — so the agent never carries that knowledge. It captures
// decisions/findings in-flow (findings are always CANDIDATE, never auto-promoted),
// runs a thin guided completion process (nudges), assembles the canonical
// structured handoff from captured state, and captures a velocity point LOCAL ONLY.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {Repository, PlanStore, Validator, VelocitySink}
//	            ↑            ↑ home store    ↑ plans      ↑ validation  ↑ MoM (stub)
//	        (proto edge)   (all faked in tests; every cross-domain call degrades
//	                        gracefully — staleness/last_validation become UNKNOWN,
//	                        never a false PASS)
//
// The structured Plan/Phase/Reference Go types are owned by the plans domain;
// execution imports them (internal/plans) as the shared model. The in-flow
// records it OWNS (Decision/Finding/Handoff/VelocityPoint/Execution) are defined
// here, mirroring the shared proto messages, and persisted in the execution home
// store. The proto wire types live one floor up (handlers/execution) and never
// import this package; the handler is the only translation point (api-steer §7).
package execution

import internalplans "plan-manager/internal/plans"

// Completeness distinguishes a full run from a partial one. COMPUTED from the
// phase-status set (full iff every phase is done); never narrated.
type Completeness string

const (
	CompletenessUnspecified Completeness = ""
	CompletenessFull        Completeness = "full"
	CompletenessPartial     Completeness = "partial"
)

// FindingTriage is the operator-triage state of a candidate finding. Findings
// are filed as CANDIDATE (unvalidated) and never auto-promoted to real bugs.
type FindingTriage string

const (
	TriageUnspecified FindingTriage = ""
	TriageCandidate   FindingTriage = "candidate"
	TriagePromoted    FindingTriage = "promoted"
	TriageDismissed   FindingTriage = "dismissed"
)

// Execution is a run↔plan linkage with the runner's current-phase pointer.
type Execution struct {
	ID             string
	PlanID         string
	RunID          string
	CurrentPhaseID string
	Complete       bool
	StartedAt      string
	UpdatedAt      string
}

// Decision is an in-flow recorded design decision, captured during execution and
// folded into the handoff.
type Decision struct {
	ID          string
	ExecutionID string
	PhaseID     string
	Summary     string
	Detail      string
	RecordedAt  string
}

// Finding is an in-flow recorded candidate finding (a possible bug). Always
// filed as CANDIDATE; an operator triages it before it becomes a real bug.
// AttributionRunID powers attribution-keyed dedup (same run_id + title is not
// double-filed).
type Finding struct {
	ID               string
	ExecutionID      string
	PhaseID          string
	Title            string
	Detail           string
	Triage           FindingTriage
	AttributionRunID string
	RecordedAt       string
}

// Handoff is the canonical, structured handoff assembled from state captured
// in-flow during a run. plan-manager owns ONLY this structured layer; the prose
// final-message catch-all is owned by the orchestration layer and linked here by
// reference (ProseHandoffRef — a pass-through link, never read by plan-manager).
type Handoff struct {
	ID                string
	ExecutionID       string
	PlanID            string
	Completeness      Completeness
	ResumePhaseID     string
	Decisions         []Decision
	CandidateFindings []Finding
	LastValidation    ValidationResult
	HasValidation     bool
	Staleness         internalplans.StalenessTier
	ProseHandoffRef   string
	AssembledAt       string
}

// VelocityPoint is a per-plan/run velocity sample. Captured LOCAL ONLY in v1;
// the meta-optimization-manager emit seam is stubbed/deferred (see VelocitySink).
type VelocityPoint struct {
	ID              string
	PlanID          string
	RunID           string
	WallTimeSeconds int64
	Tokens          int64
	Iterations      int32
	Completeness    Completeness
	RecordedAt      string
}

// ValidationResult is the most recent validation/baseline outcome for a phase,
// surfaced into PhaseContext and the handoff. Mirrors the shared proto message;
// execution receives it from the Validator seam and never recomputes it.
type ValidationResult struct {
	ID          string
	PlanID      string
	PhaseID     string
	Verdict     string
	Staleness   internalplans.StalenessTier
	CommandsRun []string
	Detail      string
	RanAt       string
}

// PhaseContext is the just-in-time context assembled for a phase — everything an
// agent would otherwise carry in its head. COMPUTED at request time.
type PhaseContext struct {
	CurrentPhase    internalplans.Phase
	NextPhase       internalplans.Phase
	HasCurrent      bool
	HasNext         bool
	RequiredReading []string
	Reminders       []string
	LastValidation  ValidationResult
	HasValidation   bool
	Staleness       internalplans.StalenessTier
	ResumePhaseID   string
	Completeness    Completeness
}

// CompletionNudge is one item in the thin guided completion process.
type CompletionNudge struct {
	// "record_finding" | "file_bugs" | "confirm_phase_status".
	Kind      string
	Message   string
	Satisfied bool
}

// CompletionInputs are the velocity inputs the agent supplies on Complete;
// wall-time is computed by the service from the execution's started_at.
type CompletionInputs struct {
	Tokens     int64
	Iterations int32
}
