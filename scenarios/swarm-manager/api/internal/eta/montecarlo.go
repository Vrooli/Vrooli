package eta

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// DefaultTrials is the Monte Carlo trial count. A few hundred trials give a
// stable p50/p80 without meaningful per-request cost on a backlog of this size.
const DefaultTrials = 400

// DefaultSeed makes the rollup deterministic: the same closure + samples yields
// the same band across requests, which is the honest behavior for an estimate
// (it should not flicker between reloads).
const DefaultSeed int64 = 1

// Estimator holds the per-effort-class duration distributions, the global and
// gate-wait pools, and the execute-lane capacity used to roll a goal's closure
// up into a p50/p80 completion band.
type Estimator struct {
	byClass  map[string]Distribution
	global   Distribution
	gateWait Distribution
	capacity int
	trials   int
	seed     int64
}

// NewEstimator builds an Estimator from the current duration samples, optional
// operator priors (per effort class; nil to use the built-in defaults), and the
// execute-lane capacity. Capacity is floored at 1.
func NewEstimator(samples []Sample, operatorPriors map[string]Prior, capacity, trials int, seed int64) *Estimator {
	if capacity < 1 {
		capacity = 1
	}
	if trials < 1 {
		trials = DefaultTrials
	}

	byClassSamples := make(map[string][]Sample)
	var all []Sample
	for _, s := range samples {
		class := NormalizeEffort(s.EffortClass)
		if class != "" {
			byClassSamples[class] = append(byClassSamples[class], s)
		}
		all = append(all, s)
	}

	defaults := DefaultPriors()
	byClass := make(map[string]Distribution, len(EffortClasses))
	for _, class := range EffortClasses {
		prior, isOperator := defaults[class], false
		if operatorPriors != nil {
			if p, ok := operatorPriors[class]; ok {
				prior, isOperator = p, true
			}
		}
		byClass[class] = BuildDistribution(class, byClassSamples[class], prior, isOperator)
	}

	globalPrior, globalIsOperator := DefaultGlobalPrior(), false
	if operatorPriors != nil {
		if p, ok := operatorPriors[""]; ok {
			globalPrior, globalIsOperator = p, true
		}
	}
	global := BuildDistribution("", all, globalPrior, globalIsOperator)
	gateWait := BuildDistribution("gate-wait", nil, DefaultGateWaitPrior(), false)

	return &Estimator{
		byClass:  byClass,
		global:   global,
		gateWait: gateWait,
		capacity: capacity,
		trials:   trials,
		seed:     seed,
	}
}

// distFor returns the distribution for an effort class, falling back to the
// global pool for unsized or unknown classes.
func (e *Estimator) distFor(class string) Distribution {
	if d, ok := e.byClass[NormalizeEffort(class)]; ok {
		return d
	}
	return e.global
}

// ClosureItem is one item in a goal's (or the board's) prerequisite closure.
type ClosureItem struct {
	Ref         string
	EffortClass string
	// Done marks an item that already counts as complete; it contributes zero
	// remaining duration but still sequences its dependents.
	Done bool
	// Gated marks an item currently blocked on a dependency/gate; it incurs an
	// additional gate-wait draw on top of its execution duration.
	Gated bool
}

// GoalClosureInput is the pure input to a rollup: the closure items plus the
// within-closure dependency edges (ref → prerequisite refs). Edges pointing
// outside the closure are ignored.
type GoalClosureInput struct {
	Items []ClosureItem
	Deps  map[string][]string
}

// Band is a p50/p80 completion estimate with an explicit basis/confidence
// label. Hours are the raw estimate; the *Label fields are humanized.
type Band struct {
	P50Hours       float64 `json:"p50_hours"`
	P80Hours       float64 `json:"p80_hours"`
	P50Label       string  `json:"p50_label"`
	P80Label       string  `json:"p80_label"`
	Basis          string  `json:"basis"`
	BasisLabel     string  `json:"basis_label"`
	Confidence     string  `json:"confidence"`
	RemainingItems int     `json:"remaining_items"`
	LaneCapacity   int     `json:"lane_capacity"`
}

// EstimateGoal rolls a closure up into a completion band. ok is false only when
// the closure is empty (no items at all). A closure whose items are all done
// yields a zero "done" band with RemainingItems 0.
//
// Each trial draws a remaining duration per pending item (execution + gate-wait
// when gated), then takes the max of the dependency critical path and the total
// remaining work divided by execute-lane capacity — the honest lower bound: a
// goal cannot finish faster than its longest prerequisite chain, nor faster
// than its total work spread across the lanes.
func (e *Estimator) EstimateGoal(in GoalClosureInput) (Band, bool) {
	if len(in.Items) == 0 {
		return Band{}, false
	}

	refs := make([]string, 0, len(in.Items))
	pending := make([]ClosureItem, 0, len(in.Items))
	for _, it := range in.Items {
		refs = append(refs, it.Ref)
		if !it.Done {
			pending = append(pending, it)
		}
	}

	basis, basisLabel, confidence := e.aggregate(pending)
	band := Band{
		Basis:          basis,
		BasisLabel:     basisLabel,
		Confidence:     confidence,
		RemainingItems: len(pending),
		LaneCapacity:   e.capacity,
	}
	if len(pending) == 0 {
		band.P50Label = HumanizeHours(0)
		band.P80Label = HumanizeHours(0)
		return band, true
	}

	rng := rand.New(rand.NewSource(e.seed)) //nolint:gosec // estimation, not crypto
	trials := make([]float64, e.trials)
	dur := make(map[string]float64, len(refs))
	for t := 0; t < e.trials; t++ {
		for _, it := range in.Items {
			if it.Done {
				dur[it.Ref] = 0
				continue
			}
			d := e.distFor(it.EffortClass).draw(rng.NormFloat64())
			if it.Gated {
				d += e.gateWait.draw(rng.NormFloat64())
			}
			dur[it.Ref] = d
		}
		critical := longestPath(refs, in.Deps, dur)
		total := 0.0
		for _, ref := range refs {
			total += dur[ref]
		}
		throughput := total / float64(e.capacity)
		trials[t] = math.Max(critical, throughput)
	}

	sort.Float64s(trials)
	band.P50Hours = percentileSorted(trials, 0.50)
	band.P80Hours = percentileSorted(trials, 0.80)
	band.P50Label = HumanizeHours(band.P50Hours)
	band.P80Label = HumanizeHours(band.P80Hours)
	return band, true
}

// aggregate summarizes the basis/label/confidence across the classes the
// pending items draw from: the weakest confidence, the summed sample count, and
// a single basis (or BasisMixed when they differ).
func (e *Estimator) aggregate(pending []ClosureItem) (basis, label, confidence string) {
	if len(pending) == 0 {
		return BasisDefault, "priors only", ConfidenceLow
	}
	confRank := map[string]int{ConfidenceHigh: 3, ConfidenceMedium: 2, ConfidenceLow: 1}
	bases := map[string]bool{}
	total := 0
	anySamples := false
	minConf := ""
	for _, it := range pending {
		d := e.distFor(it.EffortClass)
		bases[d.Basis] = true
		total += d.SampleCount
		if d.SampleCount > 0 {
			anySamples = true
		}
		if minConf == "" || confRank[d.Confidence] < confRank[minConf] {
			minConf = d.Confidence
		}
	}
	if len(bases) == 1 {
		for b := range bases {
			basis = b
		}
	} else {
		basis = BasisMixed
	}
	if anySamples {
		label = fmt.Sprintf("%d samples", total)
	} else {
		label = "priors only"
	}
	if minConf == "" {
		minConf = ConfidenceLow
	}
	return basis, label, minConf
}

// longestPath returns the maximum dependency-weighted path through the closure,
// where each node's weight is dur[ref]. It is cycle-safe: a back edge is
// treated as zero-length so a cyclic closure still terminates.
func longestPath(refs []string, deps map[string][]string, dur map[string]float64) float64 {
	memo := make(map[string]float64, len(refs))
	visiting := make(map[string]bool, len(refs))
	var dfs func(ref string) float64
	dfs = func(ref string) float64 {
		if v, ok := memo[ref]; ok {
			return v
		}
		if visiting[ref] {
			return 0 // cycle guard
		}
		visiting[ref] = true
		best := 0.0
		for _, d := range deps[ref] {
			if _, known := dur[d]; !known {
				continue // dependency outside the closure
			}
			if p := dfs(d); p > best {
				best = p
			}
		}
		visiting[ref] = false
		total := best + dur[ref]
		memo[ref] = total
		return total
	}
	maxPath := 0.0
	for _, ref := range refs {
		if p := dfs(ref); p > maxPath {
			maxPath = p
		}
	}
	return maxPath
}

// percentileSorted returns the p-quantile (0..1) of an ascending slice using
// nearest-rank. It assumes len(sorted) > 0.
func percentileSorted(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[n-1]
	}
	idx := int(math.Ceil(p*float64(n))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

// HumanizeHours renders an hour count as a coarse, honest duration phrase
// ("~3 days", "~2 weeks"). Zero or negative reads as "done".
func HumanizeHours(h float64) string {
	if h <= 0 {
		return "done"
	}
	if h < 36 {
		n := int(math.Round(h))
		if n < 1 {
			n = 1
		}
		return fmt.Sprintf("~%d %s", n, plural(n, "hour"))
	}
	if days := h / 24; days < 14 {
		n := int(math.Round(days))
		return fmt.Sprintf("~%d %s", n, plural(n, "day"))
	}
	n := int(math.Round(h / (24 * 7)))
	return fmt.Sprintf("~%d %s", n, plural(n, "week"))
}

func plural(n int, unit string) string {
	if n == 1 {
		return unit
	}
	return unit + "s"
}
