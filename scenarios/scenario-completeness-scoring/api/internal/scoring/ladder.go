package scoring

import (
	"sort"

	"github.com/vrooli/maturity-go/dimensions"
	"github.com/vrooli/maturity-go/ladder"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"scenario-completeness-scoring/internal/signals"
)

// otTargetPercent is the operational-target bar the R4 gate is evaluated
// against. The ecosystem-manager control loop reads this from its active
// profile; this scenario has no profile concept, so the cached answer uses
// the strictest interpretation (all declared targets passing). The headline
// is labeled "as of digest" either way — it never claims to be EM's state.
const otTargetPercent = 100

// deriveMaturity evaluates the shared ladder gates over the cached snapshot
// and the requirements-derived operational-target metric.
func deriveMaturity(snap signals.Snapshot) Maturity {
	errorPlus := map[dimensions.Dimension]int{}
	totals := map[dimensions.Dimension]int{}
	approx := map[dimensions.Dimension]bool{}

	for phaseName, pr := range snap.Phases.Phases {
		if pr.HasFindings {
			for _, f := range pr.Findings {
				dim := findingDimension(f, phaseName)
				if dim == "" {
					continue
				}
				totals[dim]++
				if f.GetSeverity() >= architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
					errorPlus[dim]++
				}
			}
			continue
		}
		// Older writer: no per-finding detail. A failed phase is conservatively
		// at least one error in the phase's dimension; a passed phase proves
		// nothing about counts (left at zero).
		if pr.Status == "failed" {
			if dim, ok := dimensions.ForPhase(phaseName); ok {
				totals[dim]++
				errorPlus[dim]++
				approx[dim] = true
			}
		}
	}

	sig := ladder.Signals{
		ErrorPlusByDimension: errorPlus,
		CountByDimension:     totals,
		BuildPassing:         snap.Phases.Phases["unit"].Status == "passed",
		OTKnown:              snap.Requirements.Collected,
		OTHasTargets:         snap.Requirements.TargetsTotal > 0,
		OTTarget:             otTargetPercent,
	}
	if snap.Requirements.TargetsTotal > 0 {
		sig.OTPercentage = 100 * float64(snap.Requirements.TargetsPassing) / float64(snap.Requirements.TargetsTotal)
	}

	out := Maturity{BuildPassing: sig.BuildPassing}

	th := ladder.DefaultThresholds()
	if lowest, unsatisfied := ladder.Lowest(sig, th, ""); unsatisfied {
		out.WorkingRung = string(lowest.ID)
	} else {
		out.LadderClean = true
	}
	for _, r := range ladder.Rungs() {
		if !r.Satisfied(sig, th) {
			break
		}
		out.SatisfiedThrough = string(r.ID)
	}

	dims := make([]dimensions.Dimension, 0, len(totals))
	for d := range totals {
		dims = append(dims, d)
	}
	sort.Slice(dims, func(i, j int) bool { return dims[i] < dims[j] })
	for _, d := range dims {
		out.Dimensions = append(out.Dimensions, DimensionCount{
			Dimension:   string(d),
			ErrorPlus:   errorPlus[d],
			Total:       totals[d],
			Approximate: approx[d],
		})
	}
	return out
}

// findingDimension maps a finding onto the shared dimension vocabulary:
// by its declared source first, falling back to the phase that produced it.
func findingDimension(f *architecturev1.ArchitectureFinding, phaseName string) dimensions.Dimension {
	if d, ok := dimensions.ForSource(f.GetSource()); ok {
		return d
	}
	if d, ok := dimensions.ForPhase(phaseName); ok {
		return d
	}
	return ""
}
