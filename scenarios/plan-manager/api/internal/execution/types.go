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

import (
	planmodel "plan-manager/internal/planmodel"
)

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
	// BaselineSet is the execution-owned checkpoint for a new-plan baseline
	// collection. It snapshots the resolved policy at capture time so resume and
	// phase validation never derive a different before-state from edited plan
	// prose. Legacy single-scenario anchors leave it zero-valued.
	BaselineSet BaselineSetState
	// PhaseValidationGenerations advances whenever execution-local scope changes.
	// A validation result is usable only when it was synchronized for this
	// execution and the current generation, so a prior ticket cannot certify
	// newly discovered work.
	PhaseValidationGenerations map[string]int
	ScopeAmendments            []ScopeAmendment
	DegradedReason             string
	LifecycleState             ExecutionLifecycleState
	AbandonedReason            string
	AbandonedAt                string
	AbandonedBy                string
}

type ExecutionLifecycleState string

const (
	ExecutionLifecycleActive    ExecutionLifecycleState = "active"
	ExecutionLifecycleCompleted ExecutionLifecycleState = "completed"
	ExecutionLifecycleAbandoned ExecutionLifecycleState = "abandoned"
)

func (e Execution) EffectiveLifecycleState() ExecutionLifecycleState {
	if e.LifecycleState != "" {
		return e.LifecycleState
	}
	if e.Complete {
		return ExecutionLifecycleCompleted
	}
	return ExecutionLifecycleActive
}

type BaselineSetStatus string

// BaselineSetStateSchemaVersion advances when the durable checkpoint gains new
// recovery/provenance fields. Existing JSON rows remain readable because every
// addition is optional and legacy rows simply have no member/path detail.
const BaselineSetStateSchemaVersion = 2

const (
	BaselineSetStatusRequired            BaselineSetStatus = "required"
	BaselineSetStatusScopeRepairRequired BaselineSetStatus = "scope_repair_required"
	BaselineSetStatusComplete            BaselineSetStatus = "complete"
	BaselineSetStatusPartial             BaselineSetStatus = "partial"
	BaselineSetStatusDegraded            BaselineSetStatus = "degraded"
)

// BaselineSetState is durable execution evidence, not authored plan policy.
// Path evidence remains separately informational; Required/Ready coverage is
// only the behavioral GCT collection gate.
type BaselineSetState struct {
	Version          int
	Name             string
	CollectionBranch string
	ScenarioTargets  []string
	RepoPaths        []string
	CapturedAt       string
	Status           BaselineSetStatus
	Required         int
	Ready            int
	Pending          int
	Failed           int
	Skipped          int
	Stale            int
	Members          []BaselineSetMember
	PathSnapshots    []BaselineSetPathSnapshot
	Detail           string
	CaptureArgv      []string
	WaitArgv         []string
	SyncArgv         []string
	// LegacyAdoptionRequired is set only for an explicitly marked historical
	// plan. It blocks normal work until the operator chooses an honest recapture
	// or degraded partial-handoff path.
	LegacyAdoptionRequired bool
	LastSyncedAt           string
	// SourcePreflight is a direct GCT estimate captured immediately before the
	// producer ticket is rendered. It is advisory source evidence only; member
	// coverage remains the behavioral gate.
	SourcePreflight      SourceEvidencePreflight
	PreflightUnavailable bool
}

// ScopeAmendment is an append-only explanation of a phase's real validation
// scope. It never changes historical before-state membership: newly affected
// scenarios must first be extended and captured by Git Control Tower.
type ScopeAmendment struct {
	ID                   string
	PhaseID              string
	Author               string
	Reason               string
	OldMinimum           []string
	NewMinimum           []string
	InvalidatedAt        string
	CreatedAt            string
	InvalidatedTicketIDs []string
}

// ScopeAmendmentRequest is the execution-owned, auditable control surface for
// broadening a phase validation selection within the captured inventory.
type ScopeAmendmentRequest struct {
	PhaseID string
	Members []string
	Author  string
	Reason  string
}

// SourceScopeRepairRequest replaces only the informational source selection
// for one execution ticket. Behavioral collection members are immutable.
type SourceScopeRepairRequest struct {
	Paths  []string
	Reason string
}

// BaselineAdoptionMode makes legacy execution repair explicit. Recapture only
// creates a new producer ticket; it never starts or waits for GCT. Degraded
// preserves readable history but permanently disallows normal completion.
type BaselineAdoptionMode string

const (
	BaselineAdoptionRecapture BaselineAdoptionMode = "recapture"
	BaselineAdoptionDegraded  BaselineAdoptionMode = "degraded"
)

type BaselineAdoptionRequest struct {
	Mode      BaselineAdoptionMode
	Name      string
	Members   []string
	RepoPaths []string
	Reason    string
}

// BaselineSetMember preserves the GCT member/run checkpoint used for recovery
// and drill-down. Its status is capture state, not a behavioral diff verdict.
type BaselineSetMember struct {
	Scenario     string
	BaselineName string
	Required     bool
	Status       string
	RunID        string
	GitSHA       string
	Error        string
}

// BaselineSetPathSnapshot preserves only safe source-evidence metadata. Bytes
// remain private to GCT and are never copied into Plan Manager state.
type BaselineSetPathSnapshot struct {
	Name      string
	Branch    string
	CreatedAt string
}

func (s BaselineSetState) Complete() bool {
	return s.Status == BaselineSetStatusComplete && s.Required > 0 && s.Ready == s.Required
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
	FeedbackOverrideReason   string
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
	// ChangeBoundary snapshots the plan's blast-radius contract so the next agent
	// sees what was allowed/denied and where validation coverage is informational.
	ChangeBoundary planmodel.ChangeBoundary
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
	ID              string
	PlanID          string
	PhaseID         string
	Verdict         string
	Staleness       planmodel.StalenessTier
	CommandsRun     []string
	Detail          string
	RanAt           string
	ExecutionID     string
	OperationID     string
	ScopeGeneration int
	FullInventory   bool
	RequiredMembers []string
	SelectedMembers []string
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
	BaselineSet     BaselineSetState
	ScopeGeneration int
	// ValidationMembers is non-empty only when an execution-local scope
	// amendment broadened this phase. It is rendered into the next ticket so the
	// agent does not have to reconstruct the amended selector from history.
	ValidationMembers []string
	// ChangeBoundary is the plan's (or current phase's narrowing) blast-radius
	// contract: the allowed/denied paths a fresh or resumed agent must respect,
	// surfaced without reading the full plan markdown.
	ChangeBoundary planmodel.ChangeBoundary
	// FeedbackCheckpoint is the phase-close capture gate. It is computed from
	// phase-scoped log entries or an explicit no-feedback note, never stored on the
	// phase itself.
	FeedbackCheckpoint PhaseFeedbackCheckpoint
}

// PhaseFeedbackCheckpoint reports whether a phase has durable feedback captured,
// or an explicit durable note that there was nothing to capture.
type PhaseFeedbackCheckpoint struct {
	PhaseID          string
	Reviewed         bool
	Satisfied        bool
	Summary          string
	Decisions        int
	Findings         int
	BugReports       int
	Records          int
	Notes            int
	PendingSync      int
	FailedSync       int
	NoFeedbackTitle  string
	NoFeedbackDetail string
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

type (
	NextActionKind = planmodel.NextActionKind
	NextAction     = planmodel.NextAction
	GuidedStep     = planmodel.GuidedStep
)

const (
	NextActionRecommended = planmodel.NextActionRecommended
	NextActionAlternative = planmodel.NextActionAlternative
	NextActionOptional    = planmodel.NextActionOptional
	NextActionRecovery    = planmodel.NextActionRecovery
)
