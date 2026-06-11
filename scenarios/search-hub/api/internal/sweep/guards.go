package sweep

import (
	"hash/fnv"
	"math"
	"math/rand"
	"sort"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// guards.go implements the four MANDATORY overfit guards (plan §7 Phase 6). They
// are pure functions so each is independently unit-testable — the test plan
// asserts each one blocks promotion on its own. A winner is promoted only when
// it clears ALL four:
//
//  1. Bootstrap CI of the PAIRED per-case recall margin (winner − incumbent)
//     excludes 0 — the win is statistically significant, not sampling noise.
//  2. Held-out validation: the winner, SELECTED on the tuning fold, must not
//     regress on an independent held-out fold (and the gibberish/negative
//     constraint must still hold — handled by the constraint guard on the full
//     run). Auto-`generated` cases are always held out (never select on them).
//  3. Multi-objective constraints: recall is maximized SUBJECT TO a gibberish
//     ceiling and a latency budget — an arm that leaks more junk or blows the
//     budget is infeasible regardless of recall.
//  4. Complexity / incumbent tie-break: among significant, feasible candidates,
//     prefer the simpler config (dense over hybrid, rerank-off over on, …) and
//     the incumbent — switch only past the noise band (which guard #1 defines).

// --- Guard 1: paired bootstrap CI ------------------------------------------

// pairedMarginCI bootstraps the per-case recall margin (winner[i] − incumbent[i])
// over count resamples-with-replacement of the aligned case vectors, returning
// the point-estimate mean and the 95% CI [lo, hi]. The PAIRED form (same cases
// in both arms — guaranteed, both run the same suite) is far more powerful than
// two independent CIs because it cancels per-case difficulty. Promotion requires
// lo > 0. Empty/mismatched inputs yield a zero, non-significant CI.
func pairedMarginCI(winner, incumbent []float64, count int, r *rand.Rand) (mean, lo, hi float64) {
	n := len(winner)
	if n == 0 || len(incumbent) != n {
		return 0, 0, 0
	}
	delta := make([]float64, n)
	var sum float64
	for i := range winner {
		delta[i] = winner[i] - incumbent[i]
		sum += delta[i]
	}
	mean = sum / float64(n)
	if count <= 0 {
		count = 2000
	}
	if r == nil {
		r = rand.New(rand.NewSource(1))
	}
	means := make([]float64, count)
	for b := 0; b < count; b++ {
		var s float64
		for i := 0; i < n; i++ {
			s += delta[r.Intn(n)]
		}
		means[b] = s / float64(n)
	}
	sort.Float64s(means)
	return mean, percentile(means, 2.5), percentile(means, 97.5)
}

// percentile returns the p-th percentile (0..100) of a sorted slice by
// nearest-rank, clamped to the slice bounds.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// --- Guard 2: held-out split ------------------------------------------------

// splitCases deterministically partitions positive case ids into a tuning fold
// (used to SELECT the winner) and a held-out fold (used to VALIDATE it). The
// split is stable across runs (a hash order, not RNG) so a sweep is
// reproducible. `generated` cases are forced into the held-out fold first; the
// remainder are split so the held-out fold is ceil(fraction · N) of them. With
// fraction ≤ 0 or ≥ 1 the split degrades sensibly (all-tuning / all-heldout).
func splitCases(positive []string, generated map[string]bool, fraction float64) (tuning, heldout []string) {
	// generated → always held out.
	var rest []string
	for _, id := range positive {
		if generated[id] {
			heldout = append(heldout, id)
		} else {
			rest = append(rest, id)
		}
	}
	// Stable pseudo-random order by hash so the held-out choice is deterministic
	// but not correlated with id lexical order (which often tracks difficulty).
	sort.Slice(rest, func(i, j int) bool { return hashID(rest[i]) < hashID(rest[j]) })
	k := 0
	if fraction > 0 {
		k = int(math.Ceil(fraction * float64(len(rest))))
	}
	if k > len(rest) {
		k = len(rest)
	}
	heldout = append(heldout, rest[:k]...)
	tuning = append(tuning, rest[k:]...)
	sort.Strings(tuning)
	sort.Strings(heldout)
	return tuning, heldout
}

func hashID(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// heldoutHolds reports whether a candidate validates on the held-out fold: the
// fold must be large enough to mean anything (≥ minHeldout positive cases) and
// the candidate's held-out recall must not regress below the incumbent's. A fold
// too small to validate FAILS the guard (conservative: an unvalidatable win is
// not promoted), with a reason for the verdict.
func heldoutHolds(candRecall, incRecall map[string]float64, heldout []string, minHeldout int) (ok bool, reason string) {
	if len(heldout) < minHeldout {
		return false, "held-out fold too small to validate"
	}
	cand := meanOver(candRecall, heldout)
	inc := meanOver(incRecall, heldout)
	if cand+1e-9 < inc {
		return false, "regresses on held-out fold"
	}
	return true, ""
}

// --- Guard 3: multi-objective constraints -----------------------------------

// Constraints bounds the non-objective metrics a winner must respect. MaxP95Ms
// of 0 disables the latency budget (no measurement to bound).
type Constraints struct {
	// MaxGibberish caps the worst gibberish (junk-leakage) score an arm may
	// produce; an arm above it is infeasible.
	MaxGibberish float64
	// MaxP95Ms caps the p95 latency; 0 disables the budget.
	MaxP95Ms int32
}

// feasible reports whether a run satisfies the constraints, with a reason when
// it does not (surfaced on the arm).
func (c Constraints) feasible(run *evalv1.EvalRun) (bool, string) {
	agg := run.GetAggregate()
	if agg.GetMaxGibberishScore() > c.MaxGibberish+1e-9 {
		return false, "gibberish leakage above ceiling"
	}
	if c.MaxP95Ms > 0 && agg.GetLatencyP95Ms() > c.MaxP95Ms {
		return false, "p95 latency above budget"
	}
	return true, ""
}

// deriveConstraints builds the feasibility bounds RELATIVE to the incumbent: a
// winner may not leak materially more junk than the incumbent (ceiling =
// max(incumbent gibberish, absolute floor)) nor blow a latency budget set as a
// multiple of the incumbent's p95. Anchoring to the incumbent keeps the
// constraint meaningful per-corpus instead of a magic absolute.
func deriveConstraints(incumbent *evalv1.EvalRun, absGibberishFloor, latencyMult float64) Constraints {
	agg := incumbent.GetAggregate()
	ceil := agg.GetMaxGibberishScore()
	if absGibberishFloor > ceil {
		ceil = absGibberishFloor
	}
	var budget int32
	if p := agg.GetLatencyP95Ms(); p > 0 && latencyMult > 0 {
		budget = int32(math.Ceil(float64(p) * latencyMult))
	}
	return Constraints{MaxGibberish: ceil, MaxP95Ms: budget}
}

// --- Guard 4: complexity / incumbent tie-break ------------------------------

// complexity is the cost rank of a tuning — lower is simpler and preferred. It
// encodes the plan's tie-break order: dense over hybrid, rerank-off over on,
// no-blend over blend, no task-prefix over prefix. Used to order equally-good
// candidates so the sweep switches to a more complex config only when it is
// genuinely (significantly) better, never on a coin-flip.
func complexity(t aisearch.TuningConfig) int {
	c := 0
	if t.Engine == aisearch.EngineHybrid {
		c += 4
	}
	if t.RerankEnabled {
		c += 2
	}
	if t.RerankBlend {
		c++
	}
	if t.EmbedTaskPrefix {
		c++
	}
	return c
}
