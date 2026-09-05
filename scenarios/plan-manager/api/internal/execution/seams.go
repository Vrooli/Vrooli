package execution

import (
	"context"

	planmodel "plan-manager/internal/planmodel"
)

// PlanStore is the read+phase-mutate seam onto the plans SSOT. Production wraps
// the plans domain Service; tests inject a fake. execution reads plans for
// context assembly (GetPlan) and delegates the phase-status change to the plans
// domain (UpdatePhase) — it never persists plan/phase structure itself, the plans
// domain stays the single source of truth for the record.
type PlanStore interface {
	GetPlan(ctx context.Context, idOrSlug string) (planmodel.Plan, error)
	// UpdatePhase applies an authored/status mutation to one phase and returns the
	// recomputed plan (plan status is derived from the phase-status set there).
	UpdatePhase(ctx context.Context, planID, workspaceID, workspaceRoot string, phase planmodel.Phase) (planmodel.Plan, error)
	// ExtendChangeBoundary appends allow globs to the plan's change boundary and
	// returns the recomputed plan plus the globs that were actually new. The
	// plans domain owns the append/deny/placeholder rules; execution only decides
	// WHEN a widening is legitimate and records the audit entry.
	ExtendChangeBoundary(ctx context.Context, planID, workspaceID, workspaceRoot string, globs []string) (planmodel.Plan, []string, error)
}

// Validator is the read seam onto the validation domain for the just-in-time
// context (last validation + staleness). Production wraps the validation Service;
// tests inject a fake. A nil Validator (or one returning an error) degrades the
// injected context to UNKNOWN — never a false PASS or fabricated freshness.
//
// The context server reads the LAST STORED validation result here — it does NOT
// trigger a live run. status/next are poll-style verbs; shelling git-control-tower
// on every poll would defeat the whole "cheap context for a local model" point.
// The agent runs validation explicitly (ValidationService.RunValidation), which
// persists the result this seam reads back.
type Validator interface {
	// LastValidation returns the most recent STORED validation result + its
	// staleness for a plan/phase. ok=false when none has been recorded yet (the
	// agent has not run validation) — an honest "not yet validated", never a guess.
	LastValidation(ctx context.Context, planID, phaseID string) (ValidationResult, bool, error)
}

// LogLedger is the read seam onto the log domain for the execution's
// just-in-time context summaries, completion nudges, and canonical handoff.
// Production wraps the log Service; tests inject a fake. A nil LogLedger (or one
// returning an error) degrades to an empty summary — the handoff/context still
// assemble, they just show no captured entries (never a fabricated count).
//
// Decisions, findings, bug reports, and records are OWNED by the log domain;
// execution only reads compact summaries here. The agent writes them through
// `plan-manager log ...` commands, never through the runner.
type LogLedger interface {
	// Summarize returns the compact roll-up plus the captured entries for an
	// execution (oldest-first).
	Summarize(ctx context.Context, executionID string) (planmodel.LogSummary, []planmodel.LogEntry, error)
	// SummarizePhase returns the compact roll-up plus captured entries for one
	// execution phase. It powers the phase-close feedback checkpoint.
	SummarizePhase(ctx context.Context, executionID, phaseID string) (planmodel.LogSummary, []planmodel.LogEntry, error)
}

// InputFreshener captures the regression-anchor's baseline snapshot fresh at
// execution start and recomputes reference staleness against current HEAD,
// delegated to the validation domain (which owns git-control-tower) so execution
// never imports git-control-tower directly. Production wraps the validation
// Service; tests inject a fake. A nil freshener (or one returning an error)
// degrades honestly — the freshen status is recorded and surfaced, never blocks
// phase work, and the agent can retry by resuming.
//
// This runs ONCE per execution start/resume (recorded on the Execution record),
// never on the per-poll status/next path: capturing the "before" is only valid
// immediately before edits begin, and shelling git-control-tower on every poll
// would defeat the cheap-context goal.
type BaselineSynchronizer interface {
	// FreshenInputs captures the baseline snapshot from the plan's anchor intent
	// and recomputes reference staleness. It reports the outcome; it never mutates
	// the authored plan/references (staleness is reported, not written back).
	// baselineName is the execution-owned ticket. It may differ from the
	// authored plan ticket after an explicit legacy recapture adoption.
	SyncBaseline(ctx context.Context, planID, baselineName string) (FreshenResult, error)
}

// SourceEvidencePreflighter is the optional, authoritative GCT estimate seam.
// It deliberately carries no filesystem behavior: Plan Manager records GCT's
// answer and never reconstructs selection policy locally.
type SourceEvidencePreflighter interface {
	EstimateSourceEvidence(ctx context.Context, repoPaths []string) (SourceEvidencePreflight, error)
}

type SourceEvidencePreflight struct {
	PolicyVersion          int
	IncludeIgnored         bool
	RetainContent          bool
	EligibleFiles          int
	EligibleBytes          int64
	ExcludedIgnoredFiles   int
	ExcludedIgnoredBytes   int64
	ExcludedSensitiveFiles int
	ExcludedBinaryFiles    int
	OversizedFiles         int
	RetainedContentBytes   int64
	RepairRequired         bool
	TopContributors        []SourceEvidenceContributor
	Issues                 []SourceEvidenceIssue
	Recommendations        []SourceEvidenceRecommendation
}

type (
	SourceEvidenceIssue          struct{ Code, Severity, Detail string }
	SourceEvidenceRecommendation struct{ Selection, Reason string }
	SourceEvidenceContributor    struct {
		Path  string
		Files int
		Bytes int64
	}
)

// FreshenResult reports the outcome of the execution-start freshen step.
// BaselineCaptured=false with a Detail is honest degradation (git-control-tower
// down or anchor intent still a placeholder) — never a fabricated capture.
type FreshenResult struct {
	BaselineCaptured bool
	BaselineName     string
	BaselineSet      BaselineSetState
	// StalenessSummary is a short human roll-up of the recomputed reference
	// staleness (reported only — authored references are never mutated).
	StalenessSummary string
	Detail           string
}

// VelocitySink is the future meta-optimization-manager emit seam. v1 captures
// velocity LOCAL ONLY (persisted regardless via the repository); this seam exists
// so the eventual remote emit lands behind an interface rather than a hard wire.
//
// TODO(meta-optimization-manager): wire a real emit here once the MoM ingest
// contract exists. Until then DefaultVelocitySink is a no-op — there is NO wire
// to MoM and velocity never leaves the local store. See
// docs/concepts/INTEGRATIONS.md.
type VelocitySink interface {
	Emit(ctx context.Context, point VelocityPoint) error
}

// noopVelocitySink is the stub VelocitySink: it accepts the point and does
// nothing (local persistence already happened at the call site). Wired by the
// handler module as the default; tests can assert against a recording fake.
type noopVelocitySink struct{}

// DefaultVelocitySink returns the no-op sink. The documented stub — see
// VelocitySink's TODO.
func DefaultVelocitySink() VelocitySink { return noopVelocitySink{} }

func (noopVelocitySink) Emit(context.Context, VelocityPoint) error { return nil }
