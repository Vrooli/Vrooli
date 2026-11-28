package usageattribution

import (
	"fmt"
	"math"
	"sort"
	"time"

	"agent-manager/internal/domain"
)

// Calibration methods.
const (
	MethodFitted = "fitted" // scale fitted on the window's blocks; unexplained increase is unattributed
	MethodScaled = "scaled" // too few blocks to fit: the lowest block ratio scales, no R²
	MethodNone   = "none"   // no usage to calibrate
)

const (
	// blockPoints merges one-point steps into blocks of at least this many
	// points before calibrating: integer rounding moves a single step's
	// usage by up to a whole point.
	blockPoints = 3
	// minFitBlocks is the fewest blocks with usage the scale is fitted on:
	// max(8, 3 × fitted parameters), and the scale is the only parameter (model
	// and category ratios are priors, not fitted). Below it the scale is the
	// lowest block ratio, no R² is reported and the rest stays off-disk.
	minFitBlocks = 8
	// scaleQuantile is the increase-weighted quantile of block ratios taken
	// as the window's scale.
	scaleQuantile = 0.25
)

// Slice is one step of the window's series: Points is the observed increase
// in (From, To].
type Slice struct {
	From, To time.Time
	Points   float64
}

// ModelWeight is one model's token-to-quota conversion in a window. The
// ratios between categories and models come from the price book; the scale
// is the window's own calibration.
type ModelWeight struct {
	Runner, Model string
	// CachedPerPoint is the cached-input tokens that cost one point; other
	// categories cost their weight times a cached-input token.
	CachedPerPoint                         float64
	InputWeight, CreationWeight, OutWeight float64
	// Source is where the ratio comes from: price_book | tier_prior | fallback.
	// Every source is a prior: no window fits model ratios.
	Source string
	Slices int // slices the model has usage in
}

// Calibration records how one window converted tokens to points.
type Calibration struct {
	Method  string
	Weights []ModelWeight
	// RSquared is the centered R² of the fit; zero when Method is not fitted
	// (too few blocks to report one).
	RSquared       float64
	Slices         int
	OffDiskPerHour float64
	Note           string
}

// Result is one window's attribution. Points[i] is Turns[i]'s share of the
// observed increase; sum(Points) + Unattributed == Observed.
type Result struct {
	Window       Window
	From, To     time.Time
	Observed     float64
	Unattributed float64
	Calibration  Calibration
	Turns        []Turn
	Points       []float64
	// Weighted[i] is Turns[i] in cached-input-equivalent tokens of its model.
	Weighted []float64
	// Pending is usage after the newest step, not yet in any reading.
	Pending domain.UsageCounters
	// proj is the projection context Recommend sets for its own copy.
	proj projCtx
	// pendingAt is each run's newest request after the newest step: it is in
	// no consumer's points yet but shows the run is still going.
	pendingAt map[string]time.Time
}

// Attributed is the observed increase explained by the turns.
func (r Result) Attributed() float64 { return r.Observed - r.Unattributed }

// Slices builds the steps of w's series after from: each new percent is an
// increase over the previous one. The first slice starts at from with the
// percent in effect then.
func Slices(w Window, from time.Time) []Slice {
	if from.Before(w.Start) {
		from = w.Start
	}
	var slices []Slice
	previous, at, seeded := 0.0, from, false
	for _, point := range w.Series {
		if !point.At.After(from) {
			previous, seeded = point.Percent, true
			continue
		}
		if !seeded {
			// The window has no reading at or before from: its value there is
			// unknown, so the first reading after it only anchors the series.
			previous, at, seeded = point.Percent, point.At, true
			continue
		}
		if point.Percent > previous {
			slices = append(slices, Slice{From: at, To: point.At, Points: point.Percent - previous})
			previous, at = point.Percent, point.At
		}
	}
	return slices
}

// Attribute splits w's observed increase after from across turns. Only turns
// of the provider's runner are considered. Each block's increase is
// explained by its usage up to the window's calibrated scale and shared by
// cost; what the usage does not explain (off-disk use, uncaptured
// transcripts) is the unattributed remainder.
func Attribute(w Window, from time.Time, turns []Turn, prices Prices) Result {
	slices := blocks(Slices(w, from))
	result := Result{Window: w, From: from, To: w.LastAt, Calibration: Calibration{Method: MethodNone, Slices: len(slices)}}
	if from.Before(w.Start) {
		result.From = w.Start
	}
	if len(slices) > 0 {
		result.From = slices[0].From
	}
	runner := RunnerFor(w.Provider)
	var last time.Time
	if len(slices) > 0 {
		last = slices[len(slices)-1].To
	}
	sliceOf := []int{}
	priceOf := map[string]Price{}
	costs := []float64{}
	for _, turn := range turns {
		if turn.Runner != runner || !turn.At.After(result.From) {
			continue
		}
		if len(slices) == 0 || turn.At.After(last) {
			result.Pending = addCounters(result.Pending, turn.Tokens)
			if result.pendingAt == nil {
				result.pendingAt = map[string]time.Time{}
			}
			if turn.At.After(result.pendingAt[turn.RunID]) {
				result.pendingAt[turn.RunID] = turn.At
			}
			continue
		}
		index := sort.Search(len(slices), func(i int) bool { return !slices[i].To.Before(turn.At) })
		class := classKey(turn)
		price, ok := priceOf[class]
		if !ok {
			price = prices(turn.Runner, turn.Model).normalized()
			priceOf[class] = price
		}
		result.Turns = append(result.Turns, turn)
		sliceOf = append(sliceOf, index)
		costs = append(costs, price.cost(turn.Tokens))
		result.Weighted = append(result.Weighted, price.cost(turn.Tokens)/price.CacheRead)
	}
	for _, slice := range slices {
		result.Observed += slice.Points
	}
	result.Points = make([]float64, len(result.Turns))
	if len(result.Turns) == 0 {
		result.Unattributed = result.Observed
		if len(slices) > 0 {
			result.Calibration.Note = "no on-disk usage in this window: the whole increase is unattributed"
		}
		return result
	}

	scale, calibration := calibrate(slices, sliceOf, costs)
	calibration.Slices = len(slices)
	present := map[string]map[int]bool{}
	for i, turn := range result.Turns {
		class := classKey(turn)
		if present[class] == nil {
			present[class] = map[int]bool{}
		}
		present[class][sliceOf[i]] = true
	}
	for class, price := range priceOf {
		weight := ModelWeight{Runner: classRunner(class), Model: classModel(class), InputWeight: price.Input / price.CacheRead, CreationWeight: price.CacheCreation / price.CacheRead, OutWeight: price.Output / price.CacheRead, Source: "price_book", Slices: len(present[class])}
		switch tier := TierPrice(weight.Model); {
		case price == fallbackPrice:
			weight.Source = "fallback"
		case tier.Known && price == tier.normalized():
			weight.Source = "tier_prior"
		}
		if scale > 0 {
			weight.CachedPerPoint = 1 / (scale * price.CacheRead)
		}
		calibration.Weights = append(calibration.Weights, weight)
	}
	sort.Slice(calibration.Weights, func(i, j int) bool {
		return calibration.Weights[i].Runner+calibration.Weights[i].Model < calibration.Weights[j].Runner+calibration.Weights[j].Model
	})
	result.Calibration = calibration

	cost := make([]float64, len(slices))
	for i := range result.Turns {
		cost[sliceOf[i]] += costs[i]
	}
	explained := make([]float64, len(slices))
	for b, block := range slices {
		switch {
		case cost[b] <= 0:
		default:
			explained[b] = min(block.Points, scale*cost[b])
		}
		result.Unattributed += block.Points - explained[b]
	}
	for i := range result.Turns {
		if b := sliceOf[i]; cost[b] > 0 {
			result.Points[i] = explained[b] * costs[i] / cost[b]
		}
	}
	if hours := result.To.Sub(result.From).Hours(); hours > 0 {
		result.Calibration.OffDiskPerHour = result.Unattributed / hours
	}
	return result
}

// blocks merges consecutive steps until each holds at least blockPoints.
func blocks(steps []Slice) []Slice {
	var out []Slice
	for _, step := range steps {
		if n := len(out); n > 0 && out[n-1].Points < blockPoints {
			out[n-1].To, out[n-1].Points = step.To, out[n-1].Points+step.Points
			continue
		}
		out = append(out, step)
	}
	return out
}

// calibrate returns the window's points per unit of API-price cost. Model
// and category ratios come from the price book (they match E-2's per-model
// fit: sol, terra and luna cost about 10:5:1 per cached token). Off-disk use
// only ever adds to a block's increase, so the scale is a low quantile
// (scaleQuantile, increase-weighted) of each block's increase over its usage
// cost: blocks where off-disk use dominates neither set the scale nor
// inflate on-disk shares. With too few blocks the scale is the increase of
// the blocks with usage over their cost.
func calibrate(blocks []Slice, blockOf []int, costs []float64) (float64, Calibration) {
	var calibration Calibration
	cost := make([]float64, len(blocks))
	for i, c := range costs {
		cost[blockOf[i]] += c
	}
	type ratio struct{ value, weight float64 }
	var ratios []ratio
	var observed, total float64
	for b, block := range blocks {
		if cost[b] > 0 {
			ratios = append(ratios, ratio{block.Points / cost[b], block.Points})
			observed += block.Points
			total += cost[b]
		}
	}
	if total == 0 {
		calibration.Method = MethodScaled
		return 0, calibration
	}
	if len(ratios) < minFitBlocks {
		calibration.Method = MethodScaled
		calibration.Note = fmt.Sprintf("only %d blocks with usage (need %d): the lowest block ratio scales, the rest stays off-disk, no R² reported; %s", len(ratios), minFitBlocks, priorNote)
		lowest := ratios[0].value
		for _, r := range ratios {
			lowest = math.Min(lowest, r.value)
		}
		return lowest, calibration
	}
	sort.Slice(ratios, func(i, j int) bool { return ratios[i].value < ratios[j].value })
	rank, scale := observed*scaleQuantile, ratios[len(ratios)-1].value
	for _, r := range ratios {
		if rank -= r.weight; rank <= 0 {
			scale = r.value
			break
		}
	}
	// R² is centered on the blocks' mean increase: against zero, any fit that
	// tracks the increase's size reads near 1.
	var mean, squares, residuals float64
	for _, block := range blocks {
		mean += block.Points / float64(len(blocks))
	}
	for b, block := range blocks {
		squares += (block.Points - mean) * (block.Points - mean)
		residuals += (block.Points - scale*cost[b]) * (block.Points - scale*cost[b])
	}
	calibration.Method, calibration.Note = MethodFitted, priorNote
	if squares > 0 {
		calibration.RSquared = math.Max(0, 1-residuals/squares)
	}
	return scale, calibration
}

// priorNote states, in every calibration, that model ratios are not fitted.
const priorNote = "per-model weights are priors (price book or tier ratios), not fitted; models that share blocks cannot be told apart"

func classKey(turn Turn) string {
	model := turn.Model
	if model == "" {
		model = "unknown"
	}
	return turn.Runner + "\x00" + model
}

func classRunner(class string) string {
	for i := 0; i < len(class); i++ {
		if class[i] == 0 {
			return class[:i]
		}
	}
	return class
}

func classModel(class string) string {
	for i := 0; i < len(class); i++ {
		if class[i] == 0 {
			return class[i+1:]
		}
	}
	return ""
}

func addCounters(a, b domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{InputTokens: a.InputTokens + b.InputTokens, OutputTokens: a.OutputTokens + b.OutputTokens, CacheReadTokens: a.CacheReadTokens + b.CacheReadTokens, CacheCreationTokens: a.CacheCreationTokens + b.CacheCreationTokens}
}
