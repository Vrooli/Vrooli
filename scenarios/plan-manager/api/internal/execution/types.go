// Package execution is the guided-runner domain. It links a run to a plan,
// walks the plan's phases via typed status transitions (delegating the phase
// mutation to the plans SSOT), and acts as a just-in-time context server:
// GetStatus / GetNext assemble the current phase, what is next, the phase-scoped
// required-reading + reminders, the last validation and staleness, the COMPUTED
// resume point (earliest non-done phase) and completeness (full iff every phase
// is done) — so the agent never carries that knowledge. Decisions, findings,
// bug reports, and records are owned by the log domain (LogService); execution
// reads compact log summaries through the LogLedger seam for its just-in-time
// context and handoff. It runs a thin guided completion process (nudges),
// assembles the canonical structured handoff from captured state, and captures a
// velocity point LOCAL ONLY.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {Repository, PlanStore, Validator, VelocitySink}
//	            ↑            ↑ home store    ↑ plans      ↑ validation  ↑ MoM (stub)
//	        (proto edge)   (all faked in tests; every cross-domain call degrades
//	                        gracefully — staleness/last_validation become UNKNOWN,
//	                        never a false PASS)
//
// The structured Plan/Phase/Reference Go types live in the neutral planmodel kernel;
// execution imports that kernel as the shared model. The records it OWNS
// (Handoff/VelocityPoint/Execution) are defined here, mirroring the shared proto
// messages, and persisted in the execution home store. The proto wire types live
// one floor up (handlers/execution) and never
// import this package; the handler is the only translation point (api-steer §7).
package execution

import planmodel "plan-manager/internal/planmodel"

// Completeness distinguishes a full run from a partial one. COMPUTED from the
// phase-status set (full iff every phase is done); never narrated.
type Completeness string

const (
	CompletenessUnspecified Completeness = ""
	CompletenessFull        Completeness = "full"
	CompletenessPartial     Completeness = "partial"
)

// Execution is a run↔plan linkage with the runner's current-phase pointer.
// InputsFreshenedAt/FreshenStatus/FreshenDetail record the one-time
// execution-start "freshen inputs" step (baseline snapshot capture + reference
// staleness recompute, delegated to the validation domain). They stay empty until
// the first start/resume freshens; a degraded attempt is re-tried on the next
// start/resume (status != "captured"), never on the per-poll status/next path.
type Execution struct {
	ID                string
	PlanID            string
	RunID             string
	CurrentPhaseID    string
	Complete          bool
	StartedAt         string
	UpdatedAt         string
	InputsFreshenedAt string
	FreshenStatus     string
	FreshenDetail     string
}

// Freshen status values recorded on an Execution after the execution-start
// freshen step. "captured" is terminal (never re-run); "degraded" is re-attempted
// on the next start/resume.
const (
	FreshenStatusCaptured = "captured"
	FreshenStatusDegraded = "degraded"
)

// PhaseTransitionInputs are the typed controls for a phase-status transition.
// ValidationOverrideReason is required only for done transitions that do not
// have a recent passing validation result.
type PhaseTransitionInputs struct {
	ToStatus                 planmodel.PhaseStatus
	ValidationOverrideReason string
}

// Handoff is the canonical, structured handoff assembled from state captured
// in-flow during a run. plan-manager owns ONLY this structured layer; the prose
// final-message catch-all is owned by the orchestration layer and linked here by
// reference (ProseHandoffRef — a pass-through link, never read by plan-manager).
//
// Decisions, findings, bug reports, and records are NOT owned here — they are
// typed entries in the log domain. The handoff snapshots a compact LogSummary
// plus the entries captured during the run, read through the LogLedger seam.
type Handoff struct {
	ID              string
	ExecutionID     string
	PlanID          string
	Completeness    Completeness
	ResumePhaseID   string
	LogSummary      planmodel.LogSummary
	LogEntries      []planmodel.LogEntry
	LastValidation  ValidationResult
	HasValidation   bool
	Staleness       planmodel.StalenessTier
	ProseHandoffRef string
	AssembledAt     string
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
	Staleness   planmodel.StalenessTier
	CommandsRun []string
	Detail      string
	RanAt       string
}

// PhaseContext is the just-in-time context assembled for a phase — everything an
// agent would otherwise carry in its head. COMPUTED at request time.
type PhaseContext struct {
	CurrentPhase    planmodel.Phase
	NextPhase       planmodel.Phase
	HasCurrent      bool
	HasNext         bool
	RequiredReading []string
	Reminders       []string
	RelevantContext []planmodel.RelevantContextItem
	LastValidation  ValidationResult
	HasValidation   bool
	Staleness       planmodel.StalenessTier
	ResumePhaseID   string
	Completeness    Completeness
	// LogSummary is a compact roll-up of the execution's log ledger so a resumed
	// agent sees decisions/findings/bugs/records without reading every entry.
	LogSummary planmodel.LogSummary
	// InputsFreshened/FreshenStatus/FreshenDetail surface the one-time
	// execution-start freshen step (baseline snapshot + staleness recompute) so the
	// agent sees whether the "before" anchor was captured fresh, or why it degraded.
	InputsFreshened bool
	FreshenStatus   string
	FreshenDetail   string
}

// CompletionNudge is one item in the thin guided completion process. Kinds are
// typed and Plan Manager-local; the messages point at `plan-manager log ...`
// commands, never external scenario CLIs.
type CompletionNudge struct {
	// "record_finding" | "file_bug" | "capture_record" | "confirm_phase_status".
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

type NextActionKind string

const (
	NextActionRecommended NextActionKind = "recommended"
	NextActionAlternative NextActionKind = "alternative"
	NextActionOptional    NextActionKind = "optional"
	NextActionRecovery    NextActionKind = "recovery"
)

type NextAction struct {
	ID                 string
	Kind               NextActionKind
	Label              string
	Reason             string
	Argv               []string
	ContentPlaceholder string
	BlockedBy          []string
}

type GuidedStep struct {
	StepKind       string
	Title          string
	Summary        string
	Instructions   []string
	RequiredInputs []string
	Examples       []string
	CommonMistakes []string
	NextActions    []NextAction
}
