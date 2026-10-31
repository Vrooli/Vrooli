package outcomeledger

import (
	"math"
	"math/rand"
	"sort"
)

// Pilot is what a role's recent work says about a comparison's variance and
// pace. Everything is measured from real units except when Assumed is set.
type Pilot struct {
	// UnitsPerDay counts units with a known outcome: unknown ones never join.
	UnitsPerDay float64
	// CostCV is the coefficient of variation of points per unit.
	CostCV float64
	// AcceptedShare is the share of known units with evidence of acceptance.
	AcceptedShare float64
	// CompletedShare adds units that only finished: what lifecycle status alone
	// would count if it stood in for acceptance.
	CompletedShare float64
	// Units is how many known units the pilot saw.
	Units int
	// Assumed is true when the pilot was too thin to measure the variance and
	// the default was used instead.
	Assumed bool
}

// DefaultPlanEffect is the true cost reduction a plan targets when none is
// named: a halving. A target at or below the verdict's MinEffect could never be
// read reliably, because the verdict also needs the estimate to reach MinEffect.
const DefaultPlanEffect = 0.5

// DefaultCostCV is the variance assumed when a role has fewer than
// MinPilotUnits known units. It is the pooled value from the 30 days to
// 2026-10-30 (operational-stats.md), not a guess.
const (
	DefaultCostCV = 1.5
	MinPilotUnits = 8
)

// PilotFromUnits measures a pilot from units over a number of days.
func PilotFromUnits(units []Unit, days float64) Pilot {
	known := knownUnits(units)
	pilot := Pilot{Units: len(known), CostCV: DefaultCostCV, Assumed: true}
	if days > 0 {
		pilot.UnitsPerDay = float64(len(known)) / days
	}
	if len(known) == 0 {
		return pilot
	}
	accepted, ended := 0, 0
	var sum, sumSq float64
	for _, unit := range known {
		if unit.Outcome == OutcomeAccepted {
			accepted++
		}
		if finished(unit) {
			ended++
		}
		sum += unit.Points
		sumSq += unit.Points * unit.Points
	}
	pilot.AcceptedShare = float64(accepted) / float64(len(known))
	pilot.CompletedShare = float64(ended) / float64(len(known))
	if len(known) >= MinPilotUnits && sum > 0 {
		n := float64(len(known))
		mean := sum / n
		variance := math.Max(0, (sumSq-n*mean*mean)/(n-1))
		pilot.CostCV, pilot.Assumed = math.Sqrt(variance)/mean, false
	}
	return pilot
}

// PowerOptions bound a simulation.
type PowerOptions struct {
	Sims         int
	Permutations int
	Resamples    int
	// Alpha is the per-test level the verdict reads; the default is one comparison's.
	Alpha float64
	Seed  int64
	// ChallengerAcceptance multiplies the challenger's accepted share: 0.8 means it
	// is accepted 20% less often than the incumbent. Zero means the same.
	ChallengerAcceptance float64
}

func (o PowerOptions) withDefaults() PowerOptions {
	if o.Sims <= 0 {
		o.Sims = 400
	}
	if o.Permutations <= 0 {
		o.Permutations = 399
	}
	if o.Resamples <= 0 {
		o.Resamples = 399
	}
	if o.Alpha <= 0 {
		o.Alpha = DefaultExperimentConfig().Alpha / testsPerComparison
	}
	if o.Seed == 0 {
		o.Seed = 20261030
	}
	if o.ChallengerAcceptance <= 0 {
		o.ChallengerAcceptance = 1
	}
	return o
}

// maxSimulatedUnits caps one arm of a simulation: past it a plan reports the power
// there, which understates the power at the larger sample.
const maxSimulatedUnits = 700

// simUnit is a synthetic unit.
type simUnit struct {
	cost     float64
	accepted bool
}

// drawArm draws n units: lognormal cost with mean 1 and the pilot's CV
// (scaled by costScale), accepted at the given share.
func drawArm(rng *rand.Rand, n int, pilot Pilot, costScale, acceptedShare float64) []simUnit {
	sigma2 := math.Log(1 + pilot.CostCV*pilot.CostCV)
	sigma, mu := math.Sqrt(sigma2), -sigma2/2
	out := make([]simUnit, n)
	for i := range out {
		out[i] = simUnit{cost: costScale * math.Exp(mu+sigma*rng.NormFloat64()), accepted: rng.Float64() < acceptedShare}
	}
	return out
}

func armRatio(chal, inc []simUnit) (float64, bool) {
	var cs, is float64
	var ca, ia int
	for _, u := range chal {
		cs += u.cost
		if u.accepted {
			ca++
		}
	}
	for _, u := range inc {
		is += u.cost
		if u.accepted {
			ia++
		}
	}
	if ca == 0 || ia == 0 || is <= 0 {
		return 0, false
	}
	return (cs / float64(ca)) / (is / float64(ia)), true
}

// simPermutationP is the two-sided permutation p-value of the log ratio.
func simPermutationP(rng *rand.Rand, chal, inc []simUnit, permutations int) (float64, bool) {
	observed, ok := armRatio(chal, inc)
	if !ok {
		return 1, false
	}
	pool := append(append([]simUnit(nil), chal...), inc...)
	obs := math.Abs(math.Log(observed))
	hits, valid := 1, 1
	for i := 0; i < permutations; i++ {
		rng.Shuffle(len(pool), func(a, b int) { pool[a], pool[b] = pool[b], pool[a] })
		r, ok := armRatio(pool[:len(chal)], pool[len(chal):])
		if !ok {
			continue
		}
		valid++
		if math.Abs(math.Log(r)) >= obs-1e-12 {
			hits++
		}
	}
	return float64(hits) / float64(valid), true
}

// SimulateDetection is the share of simulated comparisons with perArm units per
// arm that read "cheaper looks better" under the verdict rule (significant at the
// corrected level and at least minEffect lower) when the challenger's cost per
// unit is truly lower by effect. With ChallengerAcceptance below 1 its true cost
// per accepted outcome is (1-effect)/ChallengerAcceptance of the incumbent's.
func SimulateDetection(pilot Pilot, perArm int, effect, minEffect float64, opts PowerOptions) float64 {
	opts = opts.withDefaults()
	rng := rand.New(rand.NewSource(opts.Seed + int64(perArm)*7919 + int64(effect*1e4)))
	hits := 0
	for s := 0; s < opts.Sims; s++ {
		inc := drawArm(rng, perArm, pilot, 1, pilot.AcceptedShare)
		chal := drawArm(rng, perArm, pilot, 1-effect, pilot.AcceptedShare*opts.ChallengerAcceptance)
		ratio, ok := armRatio(chal, inc)
		if !ok {
			continue
		}
		p, _ := simPermutationP(rng, chal, inc, opts.Permutations)
		if p <= opts.Alpha && ratio <= 1-minEffect {
			hits++
		}
	}
	return float64(hits) / float64(opts.Sims)
}

// SimulateEquivalence is the share of simulated null comparisons (no true
// difference) whose bootstrap range at the corrected level falls inside plus or
// minus minEffect: the chance of reading "about the same" when it is true.
func SimulateEquivalence(pilot Pilot, perArm int, minEffect float64, opts PowerOptions) float64 {
	opts = opts.withDefaults()
	rng := rand.New(rand.NewSource(opts.Seed + 31 + int64(perArm)*104729))
	hits := 0
	draws := make([]float64, 0, opts.Resamples)
	for s := 0; s < opts.Sims; s++ {
		inc := drawArm(rng, perArm, pilot, 1, pilot.AcceptedShare)
		chal := drawArm(rng, perArm, pilot, 1, pilot.AcceptedShare)
		draws = draws[:0]
		a, b := make([]simUnit, perArm), make([]simUnit, perArm)
		for r := 0; r < opts.Resamples; r++ {
			for i := range a {
				a[i] = chal[rng.Intn(perArm)]
				b[i] = inc[rng.Intn(perArm)]
			}
			if ratio, ok := armRatio(a, b); ok {
				draws = append(draws, ratio)
			}
		}
		if len(draws) == 0 {
			continue
		}
		sort.Float64s(draws)
		low, high := percentilesAt(draws, opts.Alpha)
		if low >= 1-minEffect && high <= 1+minEffect {
			hits++
		}
	}
	return float64(hits) / float64(opts.Sims)
}

// Horizon is a time with its range, in days. Reached false means the look cannot
// be reached at this pace.
type Horizon struct {
	Days, LowDays, HighDays float64
	PerArm                  int
	Reached                 bool
}

// Plan is the expected time to the planned look before a comparison starts, and
// what the look can then read.
type Plan struct {
	Pilot Pilot
	// Effect is the true difference the power figures target.
	Effect float64
	// Detect is the time to the planned look, counting only accepted outcomes.
	Detect Horizon
	// Same is the time to the look when it is likely to read "about the same" for
	// no true difference; not reached when it is not likely.
	Same Horizon
	// CompletedOnly is the time to the look if lifecycle completion stood in for
	// acceptance. It is a weaker reading and never gives a verdict.
	CompletedOnly Horizon
	// PowerAtLook and SameAtLook are the chances a true Effect reads cheaper and no
	// difference reads about the same at the look; CompletedPower the first on the
	// completed-only sample.
	PowerAtLook, SameAtLook, CompletedPower float64
	// DetectableIn90Days is the smallest relative reduction detectable at target
	// power at the look when the look falls within 90 days; 0 means none.
	DetectableIn90Days float64
	Power              float64
	AlphaPerTest       float64
	Notes              []string
}

// PlanExperiment turns a pilot into the expected time to the planned look. split
// is the challenger share; the thinner arm sets the pace. The range is the pace's
// Poisson uncertainty. A role whose work records no accepted outcome never reaches
// the look, and the plan says so.
func PlanExperiment(pilot Pilot, split, effect, minEffect, target float64, minAccepted int, opts PowerOptions) Plan {
	if split <= 0 || split >= 1 {
		split = 0.5
	}
	if effect <= 0 {
		effect = DefaultPlanEffect
	}
	opts = opts.withDefaults()
	plan := Plan{Pilot: pilot, Effect: effect, Power: target, AlphaPerTest: opts.Alpha}
	thin := math.Min(split, 1-split)
	lookAt := func(share float64) Horizon {
		if share <= 0 || pilot.UnitsPerDay <= 0 {
			return Horizon{}
		}
		perArm := int(math.Ceil(float64(minAccepted) / share))
		days := func(rate float64) float64 { return float64(perArm) / thin / rate }
		spread := 1.28 * math.Sqrt(math.Max(pilot.UnitsPerDay*30, 1)) / 30
		slow := math.Max(pilot.UnitsPerDay-spread, pilot.UnitsPerDay*0.25/30)
		return Horizon{PerArm: perArm, Days: days(pilot.UnitsPerDay), LowDays: days(pilot.UnitsPerDay + spread), HighDays: days(slow), Reached: true}
	}
	plan.Detect = lookAt(pilot.AcceptedShare)
	plan.CompletedOnly = lookAt(pilot.CompletedShare)
	if plan.Detect.Reached {
		n := min(plan.Detect.PerArm, maxSimulatedUnits)
		plan.PowerAtLook = SimulateDetection(pilot, n, effect, minEffect, opts)
		plan.SameAtLook = SimulateEquivalence(pilot, n, minEffect, opts)
		if plan.SameAtLook >= target {
			plan.Same = plan.Detect
		}
		if plan.Detect.Days <= 90 {
			for _, e := range []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9} {
				if SimulateDetection(pilot, n, e, math.Min(minEffect, e), opts) >= target {
					plan.DetectableIn90Days = e
					break
				}
			}
		}
		if plan.Detect.PerArm > maxSimulatedUnits {
			plan.Notes = append(plan.Notes, "power is simulated at 700 units per side; the look needs more, so it is a lower bound")
		}
	} else if pilot.Units > 0 {
		plan.Notes = append(plan.Notes, "none of this role's finished work records an accepted outcome (an approval, a passing validation or a positive verdict), so no verdict is reachable from it until some does")
	}
	if plan.CompletedOnly.Reached {
		completed := pilot
		completed.AcceptedShare = pilot.CompletedShare
		plan.CompletedPower = SimulateDetection(completed, min(plan.CompletedOnly.PerArm, maxSimulatedUnits), effect, minEffect, opts)
	}
	if pilot.Assumed {
		plan.Notes = append(plan.Notes, "the role has too little finished work to measure its variance, so the pooled value was assumed")
	}
	return plan
}
