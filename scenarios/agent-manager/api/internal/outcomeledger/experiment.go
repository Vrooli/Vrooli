package outcomeledger

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

// Experiment verdicts. They read the evidence, not the mode: "looks better" is
// what the assigned arms show, not a proof.
const (
	ExperimentCheaperBetter = "cheaper_looks_better"
	ExperimentAboutSame     = "about_the_same"
	ExperimentHigherBetter  = "higher_looks_better"
	// ExperimentInconclusive is the final reading of a look that settled nothing.
	ExperimentInconclusive = "inconclusive"
	// ExperimentWithheld is a look whose arms lost different shares of their
	// outcomes, so the known units are not comparable.
	ExperimentWithheld  = "withheld"
	ExperimentNotEnough = "not_enough_data"

	// DesignObservational is work nothing assigned, or whose assignment is not on record.
	DesignObservational = "observational"
	// DesignRandomized is managed runs each assigned by chance, on record per run.
	DesignRandomized = "randomized"
	// DesignDayBlockRandomized includes operator sessions: whole day blocks were
	// assigned by chance, the block is the independent unit, and the operator saw
	// the label when launching.
	DesignDayBlockRandomized = "day_block_randomized"
)

// ExperimentConfig states, before any data is read, how much evidence a
// verdict needs. DefaultExperimentConfig holds the values the power simulation
// in docs/reference/operational-stats.md chose.
type ExperimentConfig struct {
	// MinAcceptedPerArm is the planned look: the verdict is read once, on the
	// units up to the moment both arms hold this many accepted outcomes.
	MinAcceptedPerArm int
	// MinEffect is the relative difference in points per accepted outcome that
	// counts as a difference; an interval inside plus or minus MinEffect reads
	// "about the same".
	MinEffect float64
	// ReworkMargin is the absolute rise in the rework rate that counts as worse
	// quality.
	ReworkMargin float64
	// Alpha is the family-wise two-sided level, split over the cost and rework
	// tests and over the comparisons started by the look.
	Alpha float64
	// Permutations is the number of label shuffles.
	Permutations int
	// UnknownMargin is the gap between the arms' unknown rates that withholds a verdict.
	UnknownMargin float64
	// BlockDays is the session block length the comparison randomized by.
	BlockDays int
	// ComparisonStarts lists when every learn comparison started: the family.
	ComparisonStarts []time.Time
}

// DefaultExperimentConfig is the pre-stated rule. MinAcceptedPerArm and
// MinEffect come from the power simulation (operational-stats.md, "Learn mode
// power"); ReworkMargin is a quality floor, not a power target.
func DefaultExperimentConfig() ExperimentConfig {
	return ExperimentConfig{MinAcceptedPerArm: PlannedAcceptedPerArm, MinEffect: ReadingMargin, ReworkMargin: 0.20, Alpha: 0.05, Permutations: 10000, UnknownMargin: 0.10, BlockDays: usagemode.DefaultBlockDays}
}

// PlannedAcceptedPerArm is where a halving at the pooled CV reaches about 80%
// power at the corrected level of one comparison.
const PlannedAcceptedPerArm = 80

// testsPerComparison is the cost test and the rework test.
const testsPerComparison = 2

// experimentResamples is the bootstrap size of the decision range: its tails can
// be as thin as the corrected level.
const experimentResamples = 4000

// ArmResult is one arm's measures, from Analyze, on the look's sample once the
// look is reached.
type ArmResult struct {
	Arm      string
	Row      Row
	Units    int
	Known    int
	Accepted int
	// AssumedAccepted is the part of Accepted that rests only on the agent's own
	// verdict: no approval and no validation backs it.
	AssumedAccepted int
	Completed       int
	Unknown         int
	Exceptions      int
	// TotalPoints is every unit's spend; UnknownPoints the unknown units' part of it.
	TotalPoints, UnknownPoints float64
	// Clusters is the independent units read: a whole day block counts once.
	Clusters int
}

// ExperimentReport is the analysis of one Learn comparison, by assigned arm
// (intention to treat): a run stays in the arm it was given whatever model it
// ended up on. Unknown outcomes stay out of the ratio, and are compared across
// arms, bounded and costed.
type ExperimentReport struct {
	ModeID                string
	Design                string
	Incumbent, Challenger ArmResult
	// Cost is challenger over incumbent points per accepted outcome, 90% range.
	Cost Range
	// DecisionRange is Cost at the corrected level the verdict reads, DecisionLevel.
	DecisionRange Range
	DecisionLevel float64
	// CostP and ReworkP are two-sided permutation p-values.
	CostP, ReworkP float64
	// ReworkDiff is challenger minus incumbent rework rate.
	ReworkDiff Range
	// reworkDecision is ReworkDiff at the decision level.
	reworkDecision Range
	Verdict        string
	Reason         string
	// NeedAccepted is how many more accepted outcomes the thinner arm needs to
	// reach the planned look; zero when it has them.
	NeedAccepted int
	// LookReached and LookAt say whether and when both arms reached the planned sample.
	LookReached bool
	LookAt      time.Time
	// UnitsAfterLook started after the look and cannot move the verdict.
	UnitsAfterLook int
	// Comparisons is the family at the look; AlphaPerTest the level each test reads.
	Comparisons  int
	AlphaPerTest float64
	// SpanningUnits were flagged as spanning arms and excluded.
	SpanningUnits int
	// UnknownRateGap is the larger gap between the arms' unknown and
	// acceptance-unknown rates.
	UnknownRateGap float64
	// CostBound is the cost ratio if every outcome not known to be accepted went
	// the best and the worst way, on total spend.
	CostBound Range
	Config    ExperimentConfig
}

// AnalyzeExperiment compares the arms of one mode. Analyze is reused for each
// arm's rows, so intervals and unknown handling are the ones the rest of the
// evidence view uses. Everything is deterministic: the seed comes from the mode.
func AnalyzeExperiment(modeID string, units []Unit, runs []Run, exceptions map[string]int, cfg ExperimentConfig) ExperimentReport {
	if cfg.Permutations <= 0 || cfg.MinAcceptedPerArm <= 0 || cfg.Alpha <= 0 {
		cfg = mergeConfig(cfg)
	}
	report := ExperimentReport{ModeID: modeID, Config: cfg, Verdict: ExperimentNotEnough, AlphaPerTest: cfg.Alpha / testsPerComparison, Comparisons: 1, DecisionLevel: 1 - cfg.Alpha}
	var armUnits []Unit
	dropped := map[string]bool{}
	for _, unit := range units {
		if unit.ArmMode != modeID {
			continue
		}
		if unit.SpansArms {
			report.SpanningUnits++
			dropped[unit.ID] = true
			continue
		}
		if unit.Arm != usagemode.ArmIncumbent && unit.Arm != usagemode.ArmChallenger {
			continue
		}
		unit.Key, unit.Runner = unit.Arm, ""
		armUnits = append(armUnits, unit)
	}
	report.Design = designOf(armUnits)
	sample := armUnits
	if look, ok := plannedLook(armUnits, cfg.MinAcceptedPerArm); ok {
		report.LookReached, report.LookAt = true, look
		sample = unitsAtLook(armUnits, look, cfg.BlockDays)
		report.UnitsAfterLook = len(armUnits) - len(sample)
		report.Comparisons = familyAt(cfg.ComparisonStarts, look)
		report.AlphaPerTest = cfg.Alpha / float64(testsPerComparison*report.Comparisons)
		report.DecisionLevel = 1 - 2*report.AlphaPerTest
	}
	inSample := make(map[string]bool, len(sample))
	for _, unit := range sample {
		inSample[unit.ID] = true
	}
	var armRuns []Run
	for _, run := range runs {
		if run.ArmMode != modeID || (run.Arm != usagemode.ArmIncumbent && run.Arm != usagemode.ArmChallenger) || dropped[run.UnitID] {
			continue
		}
		if run.UnitID != "" && !inSample[run.UnitID] {
			continue
		}
		run.Key, run.Runner = run.Arm, ""
		armRuns = append(armRuns, run)
	}
	analysis := Analyze(sample, armRuns, Config{PoolPaths: true})
	byArm := map[string]Row{}
	for _, row := range analysis.Rows {
		byArm[row.Key] = row
	}
	var inc, chal []Unit
	for _, unit := range knownUnits(sample) {
		if unit.Arm == usagemode.ArmChallenger {
			chal = append(chal, unit)
		} else {
			inc = append(inc, unit)
		}
	}
	report.Incumbent = armResult(usagemode.ArmIncumbent, byArm, sample, inc, exceptions, cfg.BlockDays)
	report.Challenger = armResult(usagemode.ArmChallenger, byArm, sample, chal, exceptions, cfg.BlockDays)
	report.UnknownRateGap = unknownGap(report.Incumbent, report.Challenger)
	report.CostBound = costBound(report.Incumbent, report.Challenger)

	if !report.LookReached {
		thin := min(report.Incumbent.Accepted, report.Challenger.Accepted)
		report.NeedAccepted = max(cfg.MinAcceptedPerArm-thin, 0)
		report.Reason = fmt.Sprintf("%d accepted outcomes on the incumbent and %d on the challenger; the verdict is read once, when each arm holds %d. Completed and unknown outcomes do not count, and reading this earlier cannot produce a verdict.",
			report.Incumbent.Accepted, report.Challenger.Accepted, cfg.MinAcceptedPerArm)
		return report
	}
	seed := seedFor("experiment", modeID)
	report.Cost, report.DecisionRange = clusterRanges(chal, inc, cfg.BlockDays, report.DecisionLevel, seed, func(c, i armTotals) (float64, bool) {
		if c.accepted == 0 || i.accepted == 0 || i.points <= 0 {
			return 0, false
		}
		return (c.points / float64(c.accepted)) / (i.points / float64(i.accepted)), true
	})
	report.ReworkDiff, report.reworkDecision = clusterRanges(chal, inc, cfg.BlockDays, report.DecisionLevel, seed+1, func(c, i armTotals) (float64, bool) {
		if c.units == 0 || i.units == 0 {
			return 0, false
		}
		return float64(c.redo)/float64(c.units) - float64(i.redo)/float64(i.units), true
	})
	report.CostP, report.ReworkP = permutationTests(chal, inc, cfg.BlockDays, cfg.Permutations, seedFor("permutation", modeID))
	return decide(report)
}

// designOf reads the design from the units' recorded assignment basis: nothing
// is called randomized unless the chance assignment is on record.
func designOf(units []Unit) string {
	design := DesignObservational
	for _, unit := range units {
		switch {
		case unit.ArmBasis == ArmBasisCanary:
			if design == DesignObservational {
				design = DesignRandomized
			}
		case sessionBasis(unit.ArmBasis):
			design = DesignDayBlockRandomized
		default:
			return DesignObservational
		}
	}
	return design
}

// DesignOf is the design of a set of assigned units.
func DesignOf(units []Unit) string { return designOf(units) }

// plannedLook is the start of the first unit at which both arms hold planned
// accepted outcomes, in the order the units began.
func plannedLook(units []Unit, planned int) (time.Time, bool) {
	ordered := append([]Unit(nil), units...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if !ordered[i].Start.Equal(ordered[j].Start) {
			return ordered[i].Start.Before(ordered[j].Start)
		}
		return ordered[i].ID < ordered[j].ID
	})
	var inc, chal int
	for _, unit := range ordered {
		if unit.Outcome == OutcomeAccepted {
			if unit.Arm == usagemode.ArmChallenger {
				chal++
			} else {
				inc++
			}
		}
		if inc >= planned && chal >= planned {
			return unit.Start, true
		}
	}
	return time.Time{}, false
}

// unitsAtLook is the fixed sample of a look: units started by it, and whole day
// blocks, since a block's sessions were randomized together.
func unitsAtLook(units []Unit, look time.Time, blockDays int) []Unit {
	out := make([]Unit, 0, len(units))
	for _, unit := range units {
		if !unit.Start.After(look) || (sessionBasis(unit.ArmBasis) && DayBlock(unit.Start, blockDays) <= DayBlock(look, blockDays)) {
			out = append(out, unit)
		}
	}
	return out
}

// familyAt counts the comparisons started by the look, at least this one. Later
// comparisons do not change the level, so a settled verdict stays settled.
func familyAt(starts []time.Time, look time.Time) int {
	count := 0
	for _, start := range starts {
		if !start.After(look) {
			count++
		}
	}
	return max(count, 1)
}

func armResult(arm string, rows map[string]Row, all, known []Unit, exceptions map[string]int, blockDays int) ArmResult {
	result := ArmResult{Arm: arm, Row: rows[arm], Known: len(known), Exceptions: exceptions[arm], Clusters: len(clustersOf(known, blockDays))}
	for _, unit := range all {
		if unit.Arm != arm {
			continue
		}
		result.Units++
		result.TotalPoints += unit.Points
		switch unit.Outcome {
		case OutcomeAccepted:
			result.Accepted++
			if unit.Source == SourceAgentVerdict {
				result.AssumedAccepted++
			}
		case OutcomeCompleted:
			result.Completed++
		case OutcomeUnknown:
			result.Unknown++
			result.UnknownPoints += unit.Points
		}
	}
	return result
}

// unknownGap is the larger gap between the arms in the rate of unknown outcomes
// and of outcomes whose acceptance is unknown (unknown plus completed).
func unknownGap(a, b ArmResult) float64 {
	if a.Units == 0 || b.Units == 0 {
		return 0
	}
	rate := func(count, units int) float64 { return float64(count) / float64(units) }
	unknown := math.Abs(rate(a.Unknown, a.Units) - rate(b.Unknown, b.Units))
	indeterminate := math.Abs(rate(a.Unknown+a.Completed, a.Units) - rate(b.Unknown+b.Completed, b.Units))
	return math.Max(unknown, indeterminate)
}

// costBound is the challenger's cost ratio on total spend at the two extremes:
// every outcome not known to be accepted was accepted on one side and on neither
// on the other. Whatever the unknown outcomes were, the true ratio lies inside.
func costBound(inc, chal ArmResult) Range {
	if inc.Accepted == 0 || chal.Accepted == 0 || inc.TotalPoints <= 0 {
		return Range{}
	}
	perAccepted := func(arm ArmResult, extra int) float64 { return arm.TotalPoints / float64(arm.Accepted+extra) }
	low := perAccepted(chal, chal.Unknown+chal.Completed) / perAccepted(inc, 0)
	high := perAccepted(chal, 0) / perAccepted(inc, inc.Unknown+inc.Completed)
	return Range{Available: true, Value: perAccepted(chal, 0) / perAccepted(inc, 0), Low: low, High: high, Samples: inc.Units + chal.Units}
}

func mergeConfig(cfg ExperimentConfig) ExperimentConfig {
	def := DefaultExperimentConfig()
	if cfg.MinAcceptedPerArm <= 0 {
		cfg.MinAcceptedPerArm = def.MinAcceptedPerArm
	}
	if cfg.MinEffect <= 0 {
		cfg.MinEffect = def.MinEffect
	}
	if cfg.ReworkMargin <= 0 {
		cfg.ReworkMargin = def.ReworkMargin
	}
	if cfg.Alpha <= 0 {
		cfg.Alpha = def.Alpha
	}
	if cfg.UnknownMargin <= 0 {
		cfg.UnknownMargin = def.UnknownMargin
	}
	if cfg.Permutations <= 0 {
		cfg.Permutations = def.Permutations
	}
	return cfg
}

// decide applies the pre-stated rule at the planned look. It runs once, on a
// fixed sample, so reading the status again cannot move the result.
func decide(report ExperimentReport) ExperimentReport {
	cfg := report.Config
	cost, decision, rework, reworkHigh := report.Cost, report.DecisionRange, report.ReworkDiff, report.reworkDecision.High
	level := report.DecisionLevel * 100
	spend := fmt.Sprintf(" Total spend, unknown outcomes included: incumbent %.2f points (%.2f on unknown outcomes), challenger %.2f (%.2f). If every outcome not known to be accepted went the best or worst way, the ratio is %.2f to %.2f.",
		report.Incumbent.TotalPoints, report.Incumbent.UnknownPoints, report.Challenger.TotalPoints, report.Challenger.UnknownPoints, report.CostBound.Low, report.CostBound.High)
	if report.UnknownRateGap > cfg.UnknownMargin {
		report.Verdict = ExperimentWithheld
		report.Reason = fmt.Sprintf("the arms lost different shares of their outcomes: unknown or acceptance-unknown rates differ by %.0f points, over the %.0f allowed, so the known units are not comparable.%s", report.UnknownRateGap*100, cfg.UnknownMargin*100, spend)
		return report
	}
	if !cost.Available {
		report.Verdict = ExperimentInconclusive
		report.Reason = "points per accepted outcome could not be estimated for both arms." + spend
		return report
	}
	alpha := report.AlphaPerTest
	reworkOK := rework.Available && reworkHigh < cfg.ReworkMargin
	switch {
	case rework.Available && rework.Value >= cfg.ReworkMargin && report.ReworkP <= alpha:
		report.Verdict = ExperimentHigherBetter
		report.Reason = fmt.Sprintf("the challenger redid work %.0f points more often (%.0f%% against %.0f%%, p=%.3f against %.4f)", rework.Value*100,
			report.Challenger.Row.ReworkRate.Value*100, report.Incumbent.Row.ReworkRate.Value*100, report.ReworkP, alpha)
	case report.CostP <= alpha && decision.High < 1 && cost.Value <= 1-cfg.MinEffect && reworkOK:
		report.Verdict = ExperimentCheaperBetter
		report.Reason = fmt.Sprintf("the challenger cost %.0f%% less per accepted outcome (ratio %.2f, %.1f%% range %.2f to %.2f, p=%.3f against %.4f) with no sign of more rework", (1-cost.Value)*100, cost.Value, level, decision.Low, decision.High, report.CostP, alpha)
	case report.CostP <= alpha && decision.Low > 1 && cost.Value >= 1+cfg.MinEffect:
		report.Verdict = ExperimentHigherBetter
		report.Reason = fmt.Sprintf("the challenger cost %.0f%% more per accepted outcome (ratio %.2f, %.1f%% range %.2f to %.2f, p=%.3f against %.4f)", (cost.Value-1)*100, cost.Value, level, decision.Low, decision.High, report.CostP, alpha)
	case decision.Low >= 1-cfg.MinEffect && decision.High <= 1+cfg.MinEffect && reworkOK:
		report.Verdict = ExperimentAboutSame
		report.Reason = fmt.Sprintf("cost per accepted outcome differs by no more than %.0f%% either way (ratio %.2f, %.1f%% range %.2f to %.2f) and rework is not higher", cfg.MinEffect*100, cost.Value, level, decision.Low, decision.High)
	default:
		report.Verdict = ExperimentInconclusive
		report.Reason = fmt.Sprintf("the planned look settled nothing: ratio %.2f, %.1f%% range %.2f to %.2f, p=%.3f against %.4f; it needed a significant difference beyond %.0f%%, or a range inside plus or minus %.0f%% with rework not higher",
			cost.Value, level, decision.Low, decision.High, report.CostP, alpha, cfg.MinEffect*100, cfg.MinEffect*100)
	}
	report.Reason += "." + spend
	return report
}

// armCluster is one independently randomized piece of an arm: a managed unit, or
// a whole session day block. Analysis resamples and permutes these, never the
// sessions inside a block.
type armCluster struct {
	block bool
	armTotals
}

type armTotals struct {
	points                float64
	accepted, units, redo int
}

func (t *armTotals) add(other armTotals) {
	t.points += other.points
	t.accepted += other.accepted
	t.units += other.units
	t.redo += other.redo
}

func clustersOf(units []Unit, blockDays int) []armCluster {
	blocks := map[int64]int{}
	var out []armCluster
	for _, unit := range units {
		totals := armTotals{points: unit.Points, units: 1}
		if unit.Outcome == OutcomeAccepted {
			totals.accepted = 1
		}
		if unit.Attempts > 1 || unit.Outcome == OutcomeCorrected {
			totals.redo = 1
		}
		if !sessionBasis(unit.ArmBasis) {
			out = append(out, armCluster{armTotals: totals})
			continue
		}
		block := DayBlock(unit.Start, blockDays)
		index, seen := blocks[block]
		if !seen {
			index = len(out)
			blocks[block] = index
			out = append(out, armCluster{block: true})
		}
		out[index].add(totals)
	}
	return out
}

// splitClusters groups an arm's clusters by how they were randomized: managed
// units and session blocks.
func splitClusters(clusters []armCluster) [2][]armTotals {
	var groups [2][]armTotals
	for _, cluster := range clusters {
		if cluster.block {
			groups[1] = append(groups[1], cluster.armTotals)
		} else {
			groups[0] = append(groups[0], cluster.armTotals)
		}
	}
	return groups
}

func totalsOf(groups [2][]armTotals) armTotals {
	var total armTotals
	for _, group := range groups {
		for _, totals := range group {
			total.add(totals)
		}
	}
	return total
}

// clusterRanges is a statistic's 90% range and its range at level, from
// resampling each arm's managed units and session blocks separately.
func clusterRanges(chal, inc []Unit, blockDays int, level float64, seed int64, stat func(c, i armTotals) (float64, bool)) (Range, Range) {
	a, b := splitClusters(clustersOf(chal, blockDays)), splitClusters(clustersOf(inc, blockDays))
	value, ok := stat(totalsOf(a), totalsOf(b))
	samples := len(chal) + len(inc)
	if !ok {
		return Range{Samples: samples}, Range{Samples: samples}
	}
	rng := rand.New(rand.NewSource(seed))
	draws := make([]float64, 0, experimentResamples)
	for i := 0; i < experimentResamples; i++ {
		if v, ok := stat(resampleClusters(a, rng), resampleClusters(b, rng)); ok {
			draws = append(draws, v)
		}
	}
	sort.Float64s(draws)
	shown := Range{Available: true, Value: value, Samples: samples}
	shown.Low, shown.High = percentilesAt(draws, rangeLow)
	decision := shown
	decision.Low, decision.High = percentilesAt(draws, (1-level)/2)
	return shown, decision
}

func resampleClusters(groups [2][]armTotals, rng *rand.Rand) armTotals {
	var total armTotals
	for _, group := range groups {
		for range group {
			total.add(group[rng.Intn(len(group))])
		}
	}
	return total
}

// percentilesAt reads a sorted list at tail and one minus tail.
func percentilesAt(sorted []float64, tail float64) (low, high float64) {
	if len(sorted) == 0 {
		return 0, 0
	}
	at := func(q float64) float64 {
		return sorted[int(math.Min(float64(len(sorted)-1), math.Floor(q*float64(len(sorted)))))]
	}
	return at(tail), at(1 - tail)
}

// permutationTests shuffles arm labels over what was randomized and returns the
// two-sided p-values of the cost ratio (on the log scale) and of the rework
// difference. A managed run keeps its own label and a day block's sessions share
// one, and the two are shuffled apart, so a session never moves without its
// block. The seed is fixed, so the same data always gives the same p.
func permutationTests(chal, inc []Unit, blockDays, permutations int, seed int64) (costP, reworkP float64) {
	a, b := clustersOf(chal, blockDays), clustersOf(inc, blockDays)
	all := append(append([]armCluster(nil), a...), b...)
	isChal := make([]bool, len(all))
	for i := range a {
		isChal[i] = true
	}
	observedCost, observedRework, ok := armStats(all, isChal)
	if !ok {
		return 1, 1
	}
	var managed, blocks []int
	for i, cluster := range all {
		if cluster.block {
			blocks = append(blocks, i)
		} else {
			managed = append(managed, i)
		}
	}
	rng := rand.New(rand.NewSource(seed))
	shuffled := append([]bool(nil), isChal...)
	costHits, reworkHits, valid := 1, 1, 1
	const slack = 1e-12
	for i := 0; i < permutations; i++ {
		for _, group := range [][]int{managed, blocks} {
			rng.Shuffle(len(group), func(x, y int) {
				shuffled[group[x]], shuffled[group[y]] = shuffled[group[y]], shuffled[group[x]]
			})
		}
		c, r, ok := armStats(all, shuffled)
		if !ok {
			continue
		}
		valid++
		if math.Abs(c) >= math.Abs(observedCost)-slack {
			costHits++
		}
		if math.Abs(r) >= math.Abs(observedRework)-slack {
			reworkHits++
		}
	}
	return float64(costHits) / float64(valid), float64(reworkHits) / float64(valid)
}

// armStats returns the log cost ratio (challenger over incumbent points per
// accepted outcome) and the rework difference for one labelling.
func armStats(clusters []armCluster, isChal []bool) (logRatio, reworkDiff float64, ok bool) {
	var c, i armTotals
	for index, cluster := range clusters {
		if isChal[index] {
			c.add(cluster.armTotals)
		} else {
			i.add(cluster.armTotals)
		}
	}
	if c.accepted == 0 || i.accepted == 0 || c.points <= 0 || i.points <= 0 || c.units == 0 || i.units == 0 {
		return 0, 0, false
	}
	return math.Log((c.points / float64(c.accepted)) / (i.points / float64(i.accepted))), float64(c.redo)/float64(c.units) - float64(i.redo)/float64(i.units), true
}
