// Package validation is the plan-health domain. It resolves a plan's code
// references against code-facts, computes staleness tiers from referenced-code
// change, derives each phase's exact baseline/validation command set across all
// affected locations, runs those baselines on request (compute + run, the agent
// in the loop), and verifies Definition-of-Done against the regression anchor as
// an oracle. Every cross-scenario call is behind a seam and degrades to a marked
// gap (UNKNOWN), never a false PASS.
//
// Layering:
//
//	handler → Service → {PlanSource, ReferenceResolver, StalenessComputer, CommandRunner}
//	            ↑            ↑ plans domain   ↑ code-facts      ↑ freshness    ↑ git-control-tower
//	        (proto edge)   (all faked in tests; all degrade gracefully)
//
// The structured plan/phase/reference Go types are owned by the plans domain;
// validation imports them (internal/plans) as the shared model and annotates
// references with resolution/staleness. It does NOT own project-level validation
// of resources/packages — that is consumed from test-genie / scenario-validation.
package validation

import internalplans "plan-manager/internal/plans"

// Verdict is the outcome of a validation/DoD check. Unknown is the honest
// degraded result when a composed dependency is unavailable — never a false pass.
type Verdict string

const (
	VerdictUnspecified Verdict = ""
	VerdictPass        Verdict = "pass"
	VerdictFail        Verdict = "fail"
	VerdictUnknown     Verdict = "unknown"
)

// Result is a validation/baseline outcome for a plan or phase. Computed by the
// service (verdict from baseline diff exit-0 as oracle).
type Result struct {
	ID          string
	PlanID      string
	PhaseID     string
	Verdict     Verdict
	Staleness   internalplans.StalenessTier
	CommandsRun []string
	Detail      string
	RanAt       string
}

// ReferenceReport is the resolved-reference view returned by ResolveReferences /
// ComputeStaleness: the annotated references plus whether a dependency degraded.
type ReferenceReport struct {
	References []internalplans.Reference
	Overall    internalplans.StalenessTier
	Degraded   bool
}

// BaselineScope is the derived command set across all affected locations for a
// phase (or plan), plus the distinct locations the commands cover.
type BaselineScope struct {
	Commands  []string
	Locations []string
}
