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
	UpdatePhase(ctx context.Context, planID string, phase planmodel.Phase) (planmodel.Plan, error)
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
