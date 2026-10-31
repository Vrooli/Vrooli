package usageattribution

import (
	"math"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageforecast"
)

const sumTolerance = 1e-9

func pointsSum(result Result) float64 {
	var sum float64
	for _, p := range result.Points {
		sum += p
	}
	return sum
}

// A5: every group-by's shares plus the unattributed remainder equal the
// observed increase, for the fitted and the scaled calibration alike.
func TestAttributionSharesPlusRemainderSumToObservedForEveryDimension(t *testing.T) {
	f := newWeekFixture()
	few := f.window
	few.Series = few.Series[:4] // too few slices to fit: one scale on the prior
	for name, window := range map[string]Window{"fitted": f.window, "scaled": few} {
		t.Run(name, func(t *testing.T) {
			result := Attribute(window, window.Start, f.turns, fixturePrices)
			if result.Calibration.Method != name {
				t.Fatalf("calibration method = %q, want %q (%s)", result.Calibration.Method, name, result.Calibration.Note)
			}
			if last := window.Series[len(window.Series)-1].Percent; result.Observed != last {
				t.Fatalf("observed = %v, want the newest step %v", result.Observed, last)
			}
			if got := pointsSum(result) + result.Unattributed; math.Abs(got-result.Observed) > sumTolerance {
				t.Fatalf("turn points + unattributed = %v, want observed %v", got, result.Observed)
			}
			for _, by := range Dimensions {
				var sum float64
				for _, burner := range Group(result, by, f.runs, "") {
					sum += burner.Points
				}
				if got := sum + result.Unattributed; math.Abs(got-result.Observed) > 1e-6 {
					t.Errorf("group by %s: burners + unattributed = %v, want %v", by, got, result.Observed)
				}
			}
		})
	}
}

// The window's scale is fitted against the observed increase and recorded
// per model with the price book's ratios; a window whose truth differs
// recalibrates rather than reusing the previous window's scale.
func TestAttributionRecalibratesModelWeightsPerWindow(t *testing.T) {
	f := newWeekFixture()
	first := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	doubled := truthWindow(f.turns, map[string]float64{modelTerra: 0.8, modelLuna: 1.2}, steady(0.02), 6*24)
	second := Attribute(doubled, doubled.Start, f.turns, fixturePrices)
	terra := func(r Result) ModelWeight {
		for _, w := range r.Calibration.Weights {
			if w.Model == modelTerra {
				return w
			}
		}
		t.Fatalf("no terra weight in %+v", r.Calibration.Weights)
		return ModelWeight{}
	}
	a, b := terra(first), terra(second)
	if first.Calibration.Method != MethodFitted || a.Source != "price_book" || a.Slices == 0 || a.CachedPerPoint <= 0 {
		t.Fatalf("terra weight not calibrated (%s): %+v", first.Calibration.Method, a)
	}
	// Truth: 0.4 points per USD at 5e-8 USD per cached token is 50M cached
	// tokens per point; the doubled window costs half that.
	if want := 1 / (0.4 * 5e-8); math.Abs(a.CachedPerPoint-want)/want > 0.25 {
		t.Errorf("terra cached tokens per point = %.3g, want ≈%.3g", a.CachedPerPoint, want)
	}
	if ratio := a.CachedPerPoint / b.CachedPerPoint; math.Abs(ratio-2) > 0.4 {
		t.Errorf("recalibrated ratio = %.2f, want ≈2 (first %.3g, second %.3g)", ratio, a.CachedPerPoint, b.CachedPerPoint)
	}
	if math.Abs(a.InputWeight-20) > 1e-9 || math.Abs(a.OutWeight-100) > 1e-9 {
		t.Errorf("terra category weights = in %v out %v, want the price ratios 20 and 100", a.InputWeight, a.OutWeight)
	}
	if first.Calibration.RSquared <= 0.8 {
		t.Errorf("fit R² = %.2f, want a fit that explains most slices", first.Calibration.RSquared)
	}
}

// Increases no on-disk usage explains are the unattributed remainder: a
// window with only off-disk burn is entirely unattributed, and an off-disk
// burst beside steady on-disk usage stays unattributed instead of inflating
// the sessions that happened to run during it.
func TestAttributionReportsOffDiskUseAsUnattributed(t *testing.T) {
	offDiskOnly := truthWindow(nil, nil, steady(0.5), 48)
	result := Attribute(offDiskOnly, offDiskOnly.Start, nil, fixturePrices)
	if result.Observed == 0 || result.Unattributed != result.Observed {
		t.Fatalf("off-disk window: observed %v unattributed %v, want all unattributed", result.Observed, result.Unattributed)
	}
	f := newWeekFixture()
	heavy := truthWindow(f.turns, map[string]float64{modelTerra: 0.4, modelLuna: 0.6}, burst(48, 60, 2.5), 6*24)
	fitted := Attribute(heavy, heavy.Start, f.turns, fixturePrices)
	// Truth: 2.5 pts/h for 12 hours = 30 points are off disk.
	if fitted.Unattributed < 22 || fitted.Unattributed > 36 {
		t.Errorf("unattributed = %.1f of %.0f, want ≈30 off-disk points", fitted.Unattributed, fitted.Observed)
	}
	if fitted.Calibration.OffDiskPerHour <= 0 {
		t.Errorf("off-disk rate = %v, want the fitted rate", fitted.Calibration.OffDiskPerHour)
	}
}

// since narrows the attributed period to the increase after it, and usage
// after the newest step is pending, not attributed.
func TestAttributionPeriodStartsAtSinceAndHoldsPendingUsage(t *testing.T) {
	f := newWeekFixture()
	since := f.window.Start.Add(72 * time.Hour)
	result := Attribute(f.window, since, f.turns, fixturePrices)
	before := 0.0
	for _, p := range f.window.Series {
		if !p.At.After(since) {
			before = p.Percent
		}
	}
	if want := f.window.UsedPercent - before; result.Observed != want {
		t.Fatalf("observed after since = %v, want %v", result.Observed, want)
	}
	for _, turn := range result.Turns {
		if !turn.At.After(since) {
			t.Fatalf("turn at %s precedes since %s", turn.At, since)
		}
	}
	late := append([]Turn(nil), f.turns...)
	late = append(late, Turn{At: f.window.LastAt.Add(time.Minute), RunID: "orchestrator", Runner: codex, Model: modelTerra, Reports: 1, Tokens: f.turns[0].Tokens})
	pending := Attribute(f.window, f.window.Start, late, fixturePrices)
	if pending.Pending.Total() < f.turns[0].Tokens.Total() {
		t.Fatalf("pending = %d tokens, want the usage after the newest step", pending.Pending.Total())
	}
}

// A window without a reading at or before the period start does not know the
// value there: the first reading after it anchors the series.
func TestSlicesAnchorAtFirstReadingWhenTheStartValueIsUnknown(t *testing.T) {
	start := fixtureStart
	w := Window{Start: start, Series: []usageforecast.Point{{At: start.Add(time.Hour), Percent: 20}, {At: start.Add(2 * time.Hour), Percent: 22}}}
	slices := Slices(w, start)
	if len(slices) != 1 || slices[0].Points != 2 || !slices[0].From.Equal(start.Add(time.Hour)) {
		t.Fatalf("slices = %+v, want one 2-point slice from the first reading", slices)
	}
	known := Window{Start: start, Series: append([]usageforecast.Point{{At: start}}, w.Series...)}
	if slices := Slices(known, start); len(slices) != 2 || slices[0].Points != 20 {
		t.Fatalf("slices with a known zero start = %+v, want 20 then 2", slices)
	}
}

func TestGroupSplitsToolKindsByCallsAndNamesOperatorDefaults(t *testing.T) {
	result := Result{
		Observed: 3,
		Turns: []Turn{
			{RunID: "a", Runner: codex, Reports: 1, Facts: transcriptindex.TurnFacts{ToolCalls: 4, ToolKinds: map[string]int{"shell": 3, "mcp:chrome.navigate": 1}}},
			{RunID: "b", Runner: codex, Reports: 1, Facts: transcriptindex.TurnFacts{ImagesIn: 2}},
		},
		Points:   []float64{2, 1},
		Weighted: []float64{200, 100},
	}
	runs := map[string]Run{"a": {ID: "a", Source: SourceIndexed}, "b": {ID: "b", Source: SourceAgentManager, ProfileID: "p1", ProfileName: "Delivery", Workload: "interactive:rcl/E1"}}
	kinds := map[string]float64{}
	for _, b := range Group(result, ByToolKind, runs, "") {
		kinds[b.Key] = b.Points
	}
	if kinds["shell"] != 1.5 || kinds["mcp:chrome.navigate"] != 0.5 || kinds["none"] != 1 {
		t.Fatalf("tool kind points = %v, want shell 1.5, mcp 0.5, none 1", kinds)
	}
	profiles := Group(result, ByProfile, runs, "")
	if profiles[0].Key != operatorProfile || profiles[1].Label != "Delivery" {
		t.Fatalf("profiles = %+v, want operator sessions then Delivery", profiles)
	}
	if modality := Group(result, ByModality, runs, ""); modality[0].Key != "text" || modality[1].Key != "image_in" {
		t.Fatalf("modality = %+v, want text then image_in", modality)
	}
	if only := Group(result, ByRun, runs, "b"); len(only) != 1 || only[0].Key != "b" || only[0].Source != SourceAgentManager {
		t.Fatalf("run filter = %+v, want only run b", only)
	}
}

// P3-F8: with enough blocks the fit reports a centered R² and says its model
// weights are priors; both models share every block, so their ratio cannot be
// fitted from the window.
func TestFittedCalibrationReportsCenteredRSquaredAndLabelsPriors(t *testing.T) {
	f := newWeekFixture()
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	c := result.Calibration
	if c.Method != MethodFitted || c.Slices < minFitBlocks {
		t.Fatalf("method %s over %d blocks, want fitted with at least %d", c.Method, c.Slices, minFitBlocks)
	}
	if c.RSquared <= 0 || c.RSquared > 1 || !strings.Contains(c.Note, "priors") {
		t.Errorf("R² %.3f note %q, want a centered R² in (0,1] and the weights labelled priors", c.RSquared, c.Note)
	}
	// Centered: a scale that predicts the mean increase explains nothing.
	flat := Window{Provider: "openai", Pool: "primary", Name: "10080m", Minutes: 10080, Start: fixtureStart, LastAt: f.window.LastAt, Series: f.window.Series}
	if r := Attribute(flat, flat.Start, f.turns, fixturePrices).Calibration.RSquared; r > 1 {
		t.Errorf("R² = %v", r)
	}
}

// Below the minimum block count no R² is reported and the part of an increase
// the lowest block ratio does not explain stays off-disk.
func TestTooFewBlocksReportNoRSquaredAndKeepAnOffDiskRemainder(t *testing.T) {
	f := newWeekFixture()
	few := f.window
	few.Series = few.Series[:5]
	result := Attribute(few, few.Start, f.turns, fixturePrices)
	c := result.Calibration
	if c.Method != MethodScaled || c.RSquared != 0 || !strings.Contains(c.Note, "no R²") || !strings.Contains(c.Note, "priors") {
		t.Fatalf("calibration = %+v, want scaled, no R², weights labelled priors", c)
	}
	if result.Unattributed <= 0 {
		t.Errorf("unattributed = %v, want an off-disk remainder kept below the minimum block count", result.Unattributed)
	}
}
