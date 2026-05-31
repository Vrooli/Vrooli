// Package dtv is ecosystem-manager's read seam to the development-toolchain-validator
// (DTV). The controller consumes DTV's per-skill *trust / cost / convergence*
// fitness — never *efficacy* (DTV validates against pristine goldens that carry
// zero findings, so it can never observe "skill X closed N findings"; efficacy
// is learned live by the P1 bandit). See docs/concepts/CONTROL-MODEL.md
// "Development Toolchain Validator As Gate And Prior".
//
// Fitness drives two seams: the Layer-1 eligibility gate (deny RED skills) and
// the cold-start trust/cost prior. Every read fails open: a DTV outage or an
// absent skill yields an UNKNOWN fitness, reproducing exact P1 behavior.
package dtv

import (
	"context"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
)

// Verdict is EM's view of DTV's cross-golden fitness classification. The zero
// value is VerdictUnknown so a missing/absent fitness fails open.
type Verdict int

const (
	// VerdictUnknown: DTV has no data for the skill (or was unreachable). Fails
	// open — never gated, prior degrades to uniform.
	VerdictUnknown Verdict = iota
	// VerdictGreen: latest record on every golden is PASS.
	VerdictGreen
	// VerdictYellow: incoherent-but-runnable (latest unexpected-mutation, no run
	// failures). Allowed, but trust-discounted via the prior.
	VerdictYellow
	// VerdictRed: latest record is a run failure (intrinsic non-convergence /
	// crash — Flavor-1 thrashing). Hard-gated out of selection.
	VerdictRed
)

// String renders the verdict for traces and CLI/UI surfaces.
func (v Verdict) String() string {
	switch v {
	case VerdictGreen:
		return "green"
	case VerdictYellow:
		return "yellow"
	case VerdictRed:
		return "red"
	default:
		return "unknown"
	}
}

// Fitness is the EM-local projection of DTV's SkillFitness aggregate — only the
// fields the eligibility gate and prior need. Decoupled from the proto so the
// selector never depends on DTV's generated types. The zero value is a valid
// UNKNOWN fitness.
type Fitness struct {
	SkillID string
	Verdict Verdict
	// PassRate is pass_count / total_runs across all goldens (0 when no runs).
	PassRate float64
	// TotalRuns backs the prior's min-runs guard (thin evidence ⇒ no prior).
	TotalRuns int64
	// AvgTokens is the mean tokens per run — the prior's cost denominator.
	AvgTokens float64
	// UniqueDiffHashes feeds the prior's convergence-confidence multiplier.
	UniqueDiffHashes int
	// AnyStale lowers convergence confidence (stale validation ⇒ less trust),
	// but never forces RED — staleness is reduced confidence, not a failure.
	AnyStale bool
}

// Known reports whether DTV actually has data for this skill (verdict resolved
// past UNKNOWN). Used by transparency surfaces to distinguish "no data" from a
// real verdict.
func (f Fitness) Known() bool { return f.Verdict != VerdictUnknown }

// SkillFitnessProvider is the controller's read boundary to DTV fitness.
//
// seam: SkillFitnessProvider — production wires dtv.Client (Connect-RPC to DTV
// resolved via discovery); tests wire FakeProvider. Implementations MUST fail
// open: on any error they return an UNKNOWN Fitness alongside the error.
type SkillFitnessProvider interface {
	Fitness(ctx context.Context, skillID string) (Fitness, error)
}

// fitnessFromProto maps DTV's SkillFitness onto the EM-local projection.
func fitnessFromProto(p *reportv1.SkillFitness) Fitness {
	if p == nil {
		return Fitness{}
	}
	return Fitness{
		SkillID:          p.SkillId,
		Verdict:          verdictFromProto(p.Verdict),
		PassRate:         p.PassRate,
		TotalRuns:        p.TotalRuns,
		AvgTokens:        p.AvgTokens,
		UniqueDiffHashes: int(p.UniqueDiffHashes),
		AnyStale:         p.AnyStale,
	}
}

func verdictFromProto(v reportv1.SkillFitnessVerdict) Verdict {
	switch v {
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_GREEN:
		return VerdictGreen
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_YELLOW:
		return VerdictYellow
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_RED:
		return VerdictRed
	default:
		return VerdictUnknown
	}
}
