package outcomeledger

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// Thresholds below which a row reads "not enough data".
const (
	MinAcceptedOutcomes = 5
	MinRuns             = 10
	// ReadingMargin is the pre-stated relative difference in cost that counts as
	// one: "about the same" needs the whole range inside plus or minus this.
	ReadingMargin      = 0.30
	bootstrapResamples = 1000
	rangeLow           = 0.05
	longRunHours       = 24.0
)

// Row statuses and verdicts.
const (
	StatusEnough         = "enough"
	StatusNotEnough      = "not_enough_data"
	VerdictCheaper       = "cheaper"
	VerdictMoreExpensive = "more_expensive"
	VerdictAboutSame     = "about_same"
	VerdictInconclusive  = "inconclusive"
	VerdictBaseline      = "baseline"
)

// Strata a row belongs to. An operator can correct only work they watch, so a
// headless unit can never read corrected and rework rates differ by path for
// reasons that have nothing to do with the model. Rows never pool the two.
const (
	StratumHeadless = "headless"
	StratumOperator = "operator"
)

// StratumOf names the stratum of a start path.
func StratumOf(path string) string {
	if path == PathHeadless {
		return StratumHeadless
	}
	return StratumOperator
}

// finished reports a unit whose work ended well as far as anything shows:
// accepted, or lifecycle-complete with acceptance unknown.
func finished(u Unit) bool { return u.Outcome == OutcomeAccepted || u.Outcome == OutcomeCompleted }

// Unit is one work unit as Analyze reads it.
type Unit struct {
	ID, Key, Runner string
	Outcome, Source string
	Path            string
	Attempts        int
	Points          float64
	Consumption     float64
	LinkConfidence  float64
	// Start is when the unit's first attempt began: the order the planned look
	// reads units in.
	Start time.Time
	// Arm and ArmMode are the root attempt's assigned arm of a Learn
	// comparison; empty outside one. Work stays in its assigned arm. ArmBasis
	// says how that arm is known.
	Arm, ArmMode, ArmBasis string
	// SpansArms flags runs of the unit in different arms for a reason other than
	// a retry. The unit measures neither arm and is excluded and counted.
	SpansArms bool
}

// Run is one run's use as Analyze reads it. Key can differ from its unit's: a
// Sonnet sub-agent of an Opus session counts under Sonnet.
type Run struct {
	ID, Key, Runner string
	UnitID, Path    string
	Points          float64
	Consumption     float64
	ActiveMS        int64
	WallMS          int64
	Arm, ArmMode    string
}

// Range is a statistic with its bootstrap range.
type Range struct {
	Available        bool
	Value, Low, High float64
	Samples          int
}

// Row is one group's measures.
type Row struct {
	Key, Runner, Status, Reason string
	// Stratum is headless or operator; empty when Config.PoolPaths pooled them.
	Stratum                     string
	Units, Runs                 int
	Accepted, Corrected, Failed int
	Abandoned, Unknown          int
	// Completed counts units that finished with acceptance unknown.
	Completed                int
	UnknownRate              float64
	ReworkRate               Range
	PointsTotal, PointsKnown float64
	PointsPerAccepted        Range
	// PointsPerCompleted is a weaker unit: all known points over accepted plus
	// completed units. It gives a reading, never a verdict.
	PointsPerCompleted     Range
	ConsumptionPerAccepted Range
	PointsPerRun           Range
	ConsumptionPerRun      Range
	ActiveMinutesPerRun    Range
	WallHoursMean          float64
	WallHoursMedian        float64
	RunsOver24h            int
	IdleShare              float64
	Baseline               string
	Verdict                string
	CheaperBy              Range
	ConsumptionRatio       Range
	RunLengthRatio         Range
	// BreakEvenRatio is the baseline's realized points per reference token over
	// this row's, on the same known units and fixed weights as ConsumptionRatio:
	// CheaperBy.Value equals BreakEvenRatio / ConsumptionRatio.Value.
	BreakEvenRatio float64
	// PriceSource is what the pair's points inherit: both rows' price source, or mixed.
	PriceSource string
	// ModelPriceRatio is the baseline's model price over this row's at one fixed
	// token mix of the runner, and ModelPriceSource where those prices come from.
	ModelPriceRatio  float64
	ModelPriceSource string
	// The weaker comparison: per completed outcome against CompletedBaseline.
	CompletedBaseline  string
	CheaperByCompleted Range
	CompletedReading   string
	Evidence           map[string]int
}

// Report is Analyze's result.
type Report struct {
	Rows            []Row
	Evidence, Paths map[string]int
	Units, Unknown  int
	Completed       int
	LinkConfidence  float64
	LinkedUnits     int
	Notes           []string
}

// Config tunes Analyze.
type Config struct {
	// Baseline names the row each runner's rows are compared with; empty picks
	// the priciest row per consumption unit that has enough data.
	Baseline string
	// PriceSources names what each key's weights come from.
	PriceSources map[string]string
	// ModelPrices are the price book's per-token prices by key, and ReferenceMix
	// the one token mix per runner they are compared at, so the model price stands
	// apart from the mix each model realized.
	ModelPrices  map[string]ModelPrice
	ReferenceMix map[string]TokenMix
	// PoolPaths keeps one row per key across paths. Only a randomized comparison
	// sets it: there assignment, not the path, decides who ran what.
	PoolPaths bool
}

// TokenMix is token use by category.
type TokenMix struct {
	Input, CacheRead, CacheCreation, Output float64
}

// ModelPrice is a model's price per token by category and where it comes from.
type ModelPrice struct {
	Input, CacheRead, CacheCreation, Output float64
	Source                                  string
}

func (p ModelPrice) at(mix TokenMix) float64 {
	return p.Input*mix.Input + p.CacheRead*mix.CacheRead + p.CacheCreation*mix.CacheCreation + p.Output*mix.Output
}

// Analyze computes the outcome measures. It is pure and deterministic: the
// bootstrap is seeded from each row's key.
func Analyze(units []Unit, runs []Run, cfg Config) Report {
	type key struct{ runner, stratum, key string }
	stratum := func(path string) string {
		if cfg.PoolPaths {
			return ""
		}
		return StratumOf(path)
	}
	unitGroups, runGroups := map[key][]Unit{}, map[key][]Run{}
	report := Report{Evidence: map[string]int{}, Paths: map[string]int{}}
	var confidence float64
	for _, unit := range units {
		k := key{unit.Runner, stratum(unit.Path), unit.Key}
		unitGroups[k] = append(unitGroups[k], unit)
		report.Units++
		report.Evidence[unit.Source]++
		report.Paths[unit.Path]++
		switch unit.Outcome {
		case OutcomeUnknown:
			report.Unknown++
		case OutcomeCompleted:
			report.Completed++
		}
		if unit.Attempts > 1 {
			report.LinkedUnits++
			confidence += unit.LinkConfidence
		}
	}
	if report.LinkedUnits > 0 {
		report.LinkConfidence = confidence / float64(report.LinkedUnits)
	}
	for _, run := range runs {
		k := key{run.Runner, stratum(run.Path), run.Key}
		runGroups[k] = append(runGroups[k], run)
	}
	keys := map[key]bool{}
	for k := range unitGroups {
		keys[k] = true
	}
	for k := range runGroups {
		keys[k] = true
	}
	for k := range keys {
		row := buildRow(k.key, k.runner, unitGroups[k], runGroups[k], cfg)
		row.Stratum = k.stratum
		report.Rows = append(report.Rows, row)
	}
	compareRows(report.Rows, func(r Row) []Unit { return unitGroups[key{r.Runner, r.Stratum, r.Key}] }, func(r Row) []Run { return runGroups[key{r.Runner, r.Stratum, r.Key}] }, cfg)
	sort.Slice(report.Rows, func(i, j int) bool {
		a, b := report.Rows[i], report.Rows[j]
		if a.Runner != b.Runner {
			return a.Runner < b.Runner
		}
		if a.Stratum != b.Stratum {
			return a.Stratum < b.Stratum
		}
		if a.PointsTotal != b.PointsTotal {
			return a.PointsTotal > b.PointsTotal
		}
		return a.Key < b.Key
	})
	for _, row := range report.Rows {
		if row.RunsOver24h > 0 && row.WallHoursMean > 3*row.WallHoursMedian {
			report.Notes = append(report.Notes, fmt.Sprintf("%s: mean wall %.1f h is %.0fx the median %.1f h because %d of %d runs stayed open over 24 h; median active %.1f min. Use the medians.",
				row.Key, row.WallHoursMean, row.WallHoursMean/row.WallHoursMedian, row.WallHoursMedian, row.RunsOver24h, row.Runs, row.ActiveMinutesPerRun.Value))
		}
	}
	return report
}

func seedFor(parts ...string) int64 {
	h := fnv.New64a()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return int64(h.Sum64() & (1<<62 - 1))
}

func buildRow(key, runner string, units []Unit, runs []Run, cfg Config) Row {
	row := Row{Key: key, Runner: runner, Units: len(units), Runs: len(runs), Evidence: map[string]int{}, Verdict: StatusNotEnough, PriceSource: cfg.PriceSources[key]}
	for _, unit := range units {
		row.Evidence[unit.Source]++
		row.PointsTotal += unit.Points
		switch unit.Outcome {
		case OutcomeAccepted:
			row.Accepted++
		case OutcomeCompleted:
			row.Completed++
		case OutcomeCorrected:
			row.Corrected++
		case OutcomeFailed:
			row.Failed++
		case OutcomeAbandoned:
			row.Abandoned++
		default:
			row.Unknown++
		}
		if unit.Outcome != OutcomeUnknown {
			row.PointsKnown += unit.Points
		}
	}
	if len(units) > 0 {
		row.UnknownRate = float64(row.Unknown) / float64(len(units))
	}
	rng := rand.New(rand.NewSource(seedFor("row", runner, key)))
	known := knownUnits(units)
	row.ReworkRate = bootstrap(known, rng, 1, reworkRate)
	row.PointsPerAccepted = bootstrap(known, rng, MinAcceptedOutcomes, pointsPerAccepted)
	row.ConsumptionPerAccepted = bootstrap(known, rng, MinAcceptedOutcomes, consumptionPerAccepted)
	row.PointsPerCompleted = bootstrapFinished(known, rng, pointsPerCompleted)
	row.PointsPerRun = medianRange(runs, rng, func(r Run) float64 { return r.Points })
	row.ConsumptionPerRun = medianRange(runs, rng, func(r Run) float64 { return r.Consumption })
	row.ActiveMinutesPerRun = medianRange(runs, rng, func(r Run) float64 { return float64(r.ActiveMS) / 60000 })
	var wallSum, activeSum float64
	wall := make([]float64, 0, len(runs))
	for _, run := range runs {
		hours := float64(run.WallMS) / 3.6e6
		wall = append(wall, hours)
		wallSum += hours
		activeSum += float64(run.ActiveMS) / 3.6e6
		if hours > longRunHours {
			row.RunsOver24h++
		}
	}
	if len(runs) > 0 {
		row.WallHoursMean = wallSum / float64(len(runs))
		row.WallHoursMedian = median(wall)
	}
	if wallSum > 0 {
		row.IdleShare = math.Max(0, 1-activeSum/wallSum)
	}
	if row.Accepted < MinAcceptedOutcomes {
		row.Status, row.Reason = StatusNotEnough, fmt.Sprintf("%d accepted outcomes; need %d", row.Accepted, MinAcceptedOutcomes)
	} else {
		// A row stays inconclusive unless a comparison settles it.
		row.Status, row.Verdict = StatusEnough, VerdictInconclusive
	}
	return row
}

func knownUnits(units []Unit) []Unit {
	out := make([]Unit, 0, len(units))
	for _, unit := range units {
		if unit.Outcome != OutcomeUnknown {
			out = append(out, unit)
		}
	}
	return out
}

func reworkRate(sample []Unit) (float64, bool) {
	if len(sample) == 0 {
		return 0, false
	}
	var rework int
	for _, unit := range sample {
		if unit.Attempts > 1 || unit.Outcome == OutcomeCorrected {
			rework++
		}
	}
	return float64(rework) / float64(len(sample)), true
}

// perOutcome divides all spend in the sample by the units counted as outcomes.
// A completed unit's spend stays in the numerator of a per-accepted figure: the
// money was spent whether or not anyone checked the result.
func perOutcome(sample []Unit, value func(Unit) float64, counts func(Unit) bool) (float64, bool) {
	var total float64
	outcomes := 0
	for _, unit := range sample {
		total += value(unit)
		if counts(unit) {
			outcomes++
		}
	}
	if outcomes == 0 {
		return 0, false
	}
	return total / float64(outcomes), true
}

func isAccepted(u Unit) bool { return u.Outcome == OutcomeAccepted }

func perAccepted(sample []Unit, value func(Unit) float64) (float64, bool) {
	return perOutcome(sample, value, isAccepted)
}

func pointsPerAccepted(sample []Unit) (float64, bool) {
	return perAccepted(sample, func(u Unit) float64 { return u.Points })
}

func consumptionPerAccepted(sample []Unit) (float64, bool) {
	return perAccepted(sample, func(u Unit) float64 { return u.Consumption })
}

func pointsPerCompleted(sample []Unit) (float64, bool) {
	return perOutcome(sample, func(u Unit) float64 { return u.Points }, finished)
}

// bootstrapFinished is bootstrap's floor read on accepted plus completed units.
func bootstrapFinished(sample []Unit, rng *rand.Rand, stat func([]Unit) (float64, bool)) Range {
	count := 0
	for _, unit := range sample {
		if finished(unit) {
			count++
		}
	}
	if count < MinAcceptedOutcomes {
		return Range{Samples: len(sample)}
	}
	return bootstrap(sample, rng, 1, stat)
}

// bootstrap computes stat on sample and its range over resamples. It needs at
// least minAccepted accepted units (or any sample when minAccepted is 1).
func bootstrap(sample []Unit, rng *rand.Rand, minAccepted int, stat func([]Unit) (float64, bool)) Range {
	accepted := 0
	for _, unit := range sample {
		if unit.Outcome == OutcomeAccepted {
			accepted++
		}
	}
	if len(sample) == 0 || (minAccepted > 1 && accepted < minAccepted) {
		return Range{Samples: len(sample)}
	}
	value, ok := stat(sample)
	if !ok {
		return Range{Samples: len(sample)}
	}
	out := Range{Available: true, Value: value, Samples: len(sample)}
	draws := make([]float64, 0, bootstrapResamples)
	resample := make([]Unit, len(sample))
	for i := 0; i < bootstrapResamples; i++ {
		for j := range resample {
			resample[j] = sample[rng.Intn(len(sample))]
		}
		if v, ok := stat(resample); ok {
			draws = append(draws, v)
		}
	}
	out.Low, out.High = percentiles(draws)
	return out
}

// medianRange is a median over runs with its bootstrap range; unavailable below MinRuns.
func medianRange(runs []Run, rng *rand.Rand, value func(Run) float64) Range {
	if len(runs) < MinRuns {
		return Range{Samples: len(runs)}
	}
	values := make([]float64, len(runs))
	for i, run := range runs {
		values[i] = value(run)
	}
	out := Range{Available: true, Value: median(values), Samples: len(runs)}
	draws := make([]float64, 0, bootstrapResamples)
	resample := make([]float64, len(values))
	for i := 0; i < bootstrapResamples; i++ {
		for j := range resample {
			resample[j] = values[rng.Intn(len(values))]
		}
		draws = append(draws, median(resample))
	}
	out.Low, out.High = percentiles(draws)
	return out
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func percentiles(draws []float64) (low, high float64) {
	sort.Float64s(draws)
	return percentilesAt(draws, rangeLow)
}

// readRange names what a "cheaper by" range supports. "About the same" needs the
// whole range inside the pre-stated margin; a range that only crosses 1 says
// nothing either way and is inconclusive.
func readRange(cheaper Range) string {
	switch {
	case !cheaper.Available:
		return StatusNotEnough
	case cheaper.Low > 0 && 1/cheaper.High >= 1-ReadingMargin && 1/cheaper.Low <= 1+ReadingMargin:
		return VerdictAboutSame
	case cheaper.Low > 1:
		return VerdictCheaper
	case cheaper.High < 1:
		return VerdictMoreExpensive
	}
	return VerdictInconclusive
}

// compareRows fills each enough-data row's comparison with the baseline of its
// runner and stratum, and the weaker per-completed reading beside it.
func compareRows(rows []Row, units func(Row) []Unit, runs func(Row) []Run, cfg Config) {
	type group struct{ runner, stratum string }
	type baseline struct {
		index int
		price float64
	}
	pick := func(eligible func(Row) bool) map[group]baseline {
		best := map[group]baseline{}
		for i, row := range rows {
			if !eligible(row) {
				continue
			}
			price := pricePerConsumption(knownUnits(units(row)))
			if price <= 0 {
				continue
			}
			g := group{row.Runner, row.Stratum}
			current, seen := best[g]
			named := cfg.Baseline != "" && strings.EqualFold(row.Key, cfg.Baseline)
			if named || !seen || (cfg.Baseline == "" && price > current.price) {
				best[g] = baseline{i, price}
			}
		}
		return best
	}
	accepted := pick(func(r Row) bool { return r.Status == StatusEnough })
	completed := pick(func(r Row) bool { return r.Accepted+r.Completed >= MinAcceptedOutcomes })
	for i := range rows {
		row := &rows[i]
		g := group{row.Runner, row.Stratum}
		myKnown := knownUnits(units(*row))
		if base, ok := completed[g]; ok && row.Accepted+row.Completed >= MinAcceptedOutcomes {
			reference := rows[base.index]
			row.CompletedBaseline = reference.Key
			if base.index == i {
				row.CompletedReading = VerdictBaseline
			} else {
				rng := rand.New(rand.NewSource(seedFor("completed", row.Runner, row.Stratum, row.Key, reference.Key)))
				row.CheaperByCompleted = ratioRange(knownUnits(units(reference)), myKnown, rng, pointsPerCompleted)
				row.CompletedReading = readRange(row.CheaperByCompleted)
			}
		}
		base, ok := accepted[g]
		if row.Status != StatusEnough || !ok {
			continue
		}
		reference := rows[base.index]
		row.Baseline = reference.Key
		if base.index == i {
			row.Verdict = VerdictBaseline
			continue
		}
		theirKnown := knownUnits(units(reference))
		rng := rand.New(rand.NewSource(seedFor("compare", row.Runner, row.Stratum, row.Key, reference.Key)))
		row.CheaperBy = ratioRange(theirKnown, myKnown, rng, pointsPerAccepted)
		row.ConsumptionRatio = ratioRange(myKnown, theirKnown, rng, consumptionPerAccepted)
		row.RunLengthRatio = runLengthRatio(runs(*row), runs(reference), rng)
		if price := pricePerConsumption(myKnown); price > 0 {
			row.BreakEvenRatio = base.price / price
		}
		row.PriceSource = pairSource(cfg.PriceSources[row.Key], cfg.PriceSources[reference.Key])
		mine, theirs := cfg.ModelPrices[row.Key], cfg.ModelPrices[reference.Key]
		if mix, ok := cfg.ReferenceMix[row.Runner]; ok && mine.at(mix) > 0 && theirs.at(mix) > 0 {
			row.ModelPriceRatio = theirs.at(mix) / mine.at(mix)
			row.ModelPriceSource = pairSource(mine.Source, theirs.Source)
		}
		row.Verdict = readRange(row.CheaperBy)
	}
}

// pairSource names where a pair's prices come from.
func pairSource(a, b string) string {
	if a == b {
		return a
	}
	return "mixed"
}

func pricePerConsumption(units []Unit) float64 {
	var points, consumption float64
	for _, unit := range units {
		points += unit.Points
		consumption += unit.Consumption
	}
	if consumption <= 0 {
		return 0
	}
	return points / consumption
}

// ratioRange is stat(top)/stat(bottom) with a range from independent resamples.
func ratioRange(top, bottom []Unit, rng *rand.Rand, stat func([]Unit) (float64, bool)) Range {
	if len(top) == 0 || len(bottom) == 0 {
		return Range{Samples: len(top) + len(bottom)}
	}
	a, okA := stat(top)
	b, okB := stat(bottom)
	if !okA || !okB || b == 0 {
		return Range{Samples: len(top) + len(bottom)}
	}
	out := Range{Available: true, Value: a / b, Samples: len(top) + len(bottom)}
	draws := make([]float64, 0, bootstrapResamples)
	first, second := make([]Unit, len(top)), make([]Unit, len(bottom))
	for i := 0; i < bootstrapResamples; i++ {
		for j := range first {
			first[j] = top[rng.Intn(len(top))]
		}
		for j := range second {
			second[j] = bottom[rng.Intn(len(bottom))]
		}
		x, okX := stat(first)
		y, okY := stat(second)
		if okX && okY && y != 0 {
			draws = append(draws, x/y)
		}
	}
	out.Low, out.High = percentiles(draws)
	return out
}

func runLengthRatio(mine, theirs []Run, rng *rand.Rand) Range {
	if len(mine) < MinRuns || len(theirs) < MinRuns {
		return Range{Samples: len(mine) + len(theirs)}
	}
	minutes := func(list []Run) []float64 {
		out := make([]float64, len(list))
		for i, run := range list {
			out[i] = float64(run.ActiveMS)
		}
		return out
	}
	a, b := minutes(mine), minutes(theirs)
	if median(b) == 0 {
		return Range{Samples: len(mine) + len(theirs)}
	}
	out := Range{Available: true, Value: median(a) / median(b), Samples: len(mine) + len(theirs)}
	draws := make([]float64, 0, bootstrapResamples)
	x, y := make([]float64, len(a)), make([]float64, len(b))
	for i := 0; i < bootstrapResamples; i++ {
		for j := range x {
			x[j] = a[rng.Intn(len(a))]
		}
		for j := range y {
			y[j] = b[rng.Intn(len(b))]
		}
		if m := median(y); m != 0 {
			draws = append(draws, median(x)/m)
		}
	}
	out.Low, out.High = percentiles(draws)
	return out
}

// Use is one run's points and consumption on one model.
type Use struct {
	Points, Consumption float64
}

// Dimensions Analyze groups by.
const (
	ByModel = "model"
	ByRole  = "role"
	ByMode  = "mode"
	ByArm   = "arm"
)

// Assemble turns ledger rows and per-run use into Analyze's inputs. A unit's
// key is its dominant value by points; each run keeps its own. Attached rows
// and runs with no requests are not work and stay out.
func Assemble(runs []RunFacts, use map[string]map[string]Use, by string) ([]Unit, []Run) {
	byUnit := map[string][]RunFacts{}
	for _, run := range runs {
		if run.Path == PathAttached || !run.HasRequests() || run.WorkUnitID == "" {
			continue
		}
		byUnit[run.WorkUnitID] = append(byUnit[run.WorkUnitID], run)
	}
	ids := make([]string, 0, len(byUnit))
	for id := range byUnit {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	units := make([]Unit, 0, len(ids))
	var out []Run
	for _, id := range ids {
		members := byUnit[id]
		unit := Unit{ID: id, Outcome: members[0].UnitOutcome, Source: members[0].UnitSource, LinkConfidence: 1}
		if unit.Outcome == "" {
			unit.Outcome, unit.Source = OutcomeUnknown, SourceNoSignal
		}
		list := attempts(members)
		root := list[0]
		unit.Runner, unit.Path = root.Harness, root.Path
		unit.Arm, unit.ArmMode, unit.ArmBasis = root.Arm, root.ArmMode, root.ArmBasis
		unit.Start = runTime(root)
		unit.Attempts = len(list)
		assignArms(&unit, members)
		weights := map[string]float64{}
		for _, member := range members {
			if !unit.SpansArms {
				// A retry and a child with no arm of its own stay in the unit's arm.
				member.Arm, member.ArmMode = unit.Arm, unit.ArmMode
			}
			models := use[member.RunID]
			var runPoints, runConsumption float64
			runWeights := map[string]float64{}
			for model, u := range models {
				runPoints += u.Points
				runConsumption += u.Consumption
				runWeights[runKey(member, model, by)] += weight(u)
				weights[runKey(member, model, by)] += weight(u)
			}
			if len(models) == 0 {
				weights[runKey(member, member.ActualModel, by)] += 0
				runWeights[runKey(member, member.ActualModel, by)] += 0
			}
			unit.Points += runPoints
			unit.Consumption += runConsumption
			if member.LinkConfidence > 0 && member.LinkConfidence < unit.LinkConfidence {
				unit.LinkConfidence = member.LinkConfidence
			}
			out = append(out, Run{ID: member.RunID, UnitID: id, Path: member.Path, Key: dominant(runWeights), Runner: member.Harness, Points: runPoints, Consumption: runConsumption, ActiveMS: member.ActiveMS, WallMS: member.WallMS, Arm: member.Arm, ArmMode: member.ArmMode})
		}
		unit.Key = dominant(weights)
		units = append(units, unit)
	}
	return units, out
}

// assignArms flags a unit whose runs fell in different arms. A retry stays in the
// arm the unit started in, and a run with no arm follows its unit. Anything else
// measures neither arm: a run assigned elsewhere, or assigned work joined to a
// unit that started outside the comparison.
func assignArms(unit *Unit, members []RunFacts) {
	for _, member := range members {
		switch {
		case member.Arm == unit.Arm && member.ArmMode == unit.ArmMode, member.Arm == "":
		case unit.Arm != "" && !member.Child() && retryLink(member.LinkKind):
		default:
			unit.SpansArms = true
			if unit.ArmMode == "" {
				unit.ArmMode = member.ArmMode
			}
		}
	}
}

func weight(u Use) float64 {
	if u.Points > 0 {
		return u.Points
	}
	return u.Consumption * 1e-9
}

func runKey(run RunFacts, model, by string) string {
	switch by {
	case ByRole:
		switch {
		case run.Role != "":
			return run.Role
		case run.Child():
			return "sub_agent"
		}
		return "operator_session"
	case ByArm:
		if run.Arm == "" {
			return "unassigned"
		}
		return run.Arm
	case ByMode:
		switch run.ModeRelation {
		case ModeApplied, ModeOnTarget:
			return run.ModeID
		case ModeNotChanged, ModePredates, ModeSkipped:
			return "not_changed"
		}
		return "no_mode"
	}
	if model == "" {
		return "unknown"
	}
	return model
}

func dominant(weights map[string]float64) string {
	best, value := "", -1.0
	for key, w := range weights {
		if w > value || (w == value && key < best) {
			best, value = key, w
		}
	}
	if best == "" {
		return "unknown"
	}
	return best
}
