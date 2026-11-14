package usageforecast

import (
	"math"
	"sort"
	"time"

	"agent-manager/internal/pricing"
)

// Checkpoints are the fractions of a window's actual life at which the
// backtest projects (SPEC A4).
var Checkpoints = []float64{0.25, 0.5}

// Populations: SPEC A4 scores the newest ScoredWindowsPerScope windows of
// each provider/scope; the whole history is reported beside it.
const (
	PopulationRecent      = "last_5_windows_per_scope"
	PopulationAll         = "all_windows"
	ScoredWindowsPerScope = 5
)

// Checkpoint compares one projection with the window's outcome. A baseline
// projection is the naive linear rate since the window start.
type Checkpoint struct {
	Fraction              float64
	AsOf                  time.Time
	UsedAt                float64
	PredictedFinal        float64
	BaselineFinal         float64
	AbsErrorPoints        float64
	BaselineAbsError      float64
	PredictedExhaustion   *time.Time
	BaselineExhaustion    *time.Time
	AbsErrorHours         *float64
	BaselineAbsErrorHours *float64
	OutcomeCorrect        bool
	BaselineOutcomeOK     bool
	Calibration           Calibration
}

// BacktestWindow is one completed window and its checkpoints.
type BacktestWindow struct {
	Window           *Window
	EndedAt          time.Time
	EarlyReset       bool
	Final            float64
	ActualExhaustion *time.Time
	// OutcomeObserved is false when the window did not exhaust and its newest
	// reading is more than StaleAfter before the end: Final is then only a
	// lower bound, so the window is not scored.
	OutcomeObserved bool
	Checkpoints     []Checkpoint
}

// Summary aggregates one checkpoint fraction across one population.
type Summary struct {
	Population                string
	Fraction                  float64
	Windows                   int64
	MedianAbsErrorPoints      float64
	BaselineMedianAbsErrorPts float64
	HourErrorWindows          int64
	MedianAbsErrorHours       float64
	BaselineMedianAbsErrorHrs float64
	OutcomeAccuracy           float64
	BaselineOutcomeAccuracy   float64
	BeatsBaseline             bool
}

// Report is the backtest over every completed window in the readings.
type Report struct {
	Windows []BacktestWindow
	// Summaries holds, per checkpoint, the PopulationRecent summary then the
	// PopulationAll one.
	Summaries []Summary
	// Unobserved counts completed windows whose outcome was not observed.
	Unobserved int64
}

// Backtest replays the forecaster over every completed window that reported
// percentages and used any quota, in time order: each projection sees only
// the readings and completed windows that existed at its checkpoint.
func Backtest(spans []pricing.QuotaReadingSpan, now time.Time) Report {
	byScope := map[string][]*Window{}
	var scopes []string
	for _, w := range Windows(spans) {
		if _, seen := byScope[w.Scope()]; !seen {
			scopes = append(scopes, w.Scope())
		}
		byScope[w.Scope()] = append(byScope[w.Scope()], w)
	}
	sort.Strings(scopes)
	var report Report
	for _, scope := range scopes {
		windows := byScope[scope]
		for i, w := range windows {
			if !w.HasPercent || w.Max() <= 0 {
				continue
			}
			var next *Window
			if i+1 < len(windows) {
				next = windows[i+1]
			}
			end, done := w.end(next, now)
			if !done {
				continue
			}
			scored := backtestWindow(w, windows[:i], end)
			if !scored.OutcomeObserved {
				report.Unobserved++
			}
			report.Windows = append(report.Windows, scored)
		}
	}
	for _, fraction := range Checkpoints {
		report.Summaries = append(report.Summaries, summarize(recent(report.Windows, fraction), fraction, PopulationRecent), summarize(report.Windows, fraction, PopulationAll))
	}
	return report
}

func backtestWindow(w *Window, previous []*Window, end time.Time) BacktestWindow {
	out := BacktestWindow{Window: w, EndedAt: end, Final: w.Max(), EarlyReset: !w.ResetAt.IsZero() && end.Before(w.ResetAt.Add(-resetTolerance))}
	for _, p := range w.points {
		if p.Percent >= 100 {
			at := p.At
			out.ActualExhaustion = &at
			break
		}
	}
	// An exhausted window's outcome is known even though readings stop.
	out.OutcomeObserved = out.ActualExhaustion != nil || end.Sub(w.LastAt) <= StaleAfter
	for _, fraction := range Checkpoints {
		at := w.Start.Add(time.Duration(fraction * float64(end.Sub(w.Start))))
		// A forecaster at the checkpoint whose newest reading is stale
		// reports unknown, so it makes no projection to score.
		seen, ok := w.truncate(at)
		if !ok || at.Sub(seen.LastAt) > StaleAfter || seen.LastAt.Sub(seen.origin()) < minBurnSpan {
			continue
		}
		model := project(&seen, previous, seen.LastAt, end)
		used := seen.Max()
		if used >= 100 {
			continue
		}
		baselineRate := rateOver(&seen, seen.origin(), seen.LastAt, 0).Rate
		c := Checkpoint{Fraction: fraction, AsOf: seen.LastAt, UsedAt: used, Calibration: model.Calibration}
		c.PredictedFinal = math.Min(100, used+model.burnBetween(seen.LastAt, end))
		c.PredictedExhaustion, c.AbsErrorHours = exhaustion(model.Projected, end, out.ActualExhaustion)
		c.BaselineFinal = math.Min(100, used+baselineRate*end.Sub(seen.LastAt).Hours())
		c.BaselineExhaustion, c.BaselineAbsErrorHours = exhaustion(after(seen.LastAt, 100-used, baselineRate), end, out.ActualExhaustion)
		c.AbsErrorPoints, c.BaselineAbsError = math.Abs(c.PredictedFinal-out.Final), math.Abs(c.BaselineFinal-out.Final)
		c.OutcomeCorrect = exhaustsBy(c.PredictedExhaustion, end) == (out.ActualExhaustion != nil)
		c.BaselineOutcomeOK = exhaustsBy(c.BaselineExhaustion, end) == (out.ActualExhaustion != nil)
		out.Checkpoints = append(out.Checkpoints, c)
	}
	return out
}

// exhaustion reports a projected run-out when it falls on or before the
// window end; the hour error is defined only when the window actually
// exhausted.
func exhaustion(projected *time.Time, end time.Time, actual *time.Time) (*time.Time, *float64) {
	var hours *float64
	if actual != nil && projected != nil {
		v := math.Abs(projected.Sub(*actual).Hours())
		hours = &v
	}
	if projected != nil && projected.After(end) {
		projected = nil
	}
	return projected, hours
}

func exhaustsBy(at *time.Time, end time.Time) bool { return at != nil && !at.After(end) }

// recent keeps, per scope, the newest ScoredWindowsPerScope windows that are
// scored at fraction.
func recent(windows []BacktestWindow, fraction float64) []BacktestWindow {
	byScope := map[string][]BacktestWindow{}
	for _, w := range windows {
		if w.OutcomeObserved && w.checkpoint(fraction) != nil {
			byScope[w.Window.Scope()] = append(byScope[w.Window.Scope()], w)
		}
	}
	var out []BacktestWindow
	for _, scoped := range byScope {
		sort.SliceStable(scoped, func(i, j int) bool { return scoped[i].EndedAt.Before(scoped[j].EndedAt) })
		out = append(out, scoped[max(0, len(scoped)-ScoredWindowsPerScope):]...)
	}
	return out
}

func (w BacktestWindow) checkpoint(fraction float64) *Checkpoint {
	for i := range w.Checkpoints {
		if w.Checkpoints[i].Fraction == fraction {
			return &w.Checkpoints[i]
		}
	}
	return nil
}

func summarize(windows []BacktestWindow, fraction float64, population string) Summary {
	s := Summary{Population: population, Fraction: fraction}
	var model, baseline, modelHours, baselineHours []float64
	var modelOK, baselineOK int64
	for _, w := range windows {
		c := w.checkpoint(fraction)
		if !w.OutcomeObserved || c == nil {
			continue
		}
		s.Windows++
		model, baseline = append(model, c.AbsErrorPoints), append(baseline, c.BaselineAbsError)
		if c.AbsErrorHours != nil && c.BaselineAbsErrorHours != nil {
			modelHours, baselineHours = append(modelHours, *c.AbsErrorHours), append(baselineHours, *c.BaselineAbsErrorHours)
		}
		if c.OutcomeCorrect {
			modelOK++
		}
		if c.BaselineOutcomeOK {
			baselineOK++
		}
	}
	if s.Windows == 0 {
		return s
	}
	s.MedianAbsErrorPoints, s.BaselineMedianAbsErrorPts = median(model), median(baseline)
	s.HourErrorWindows = int64(len(modelHours))
	s.MedianAbsErrorHours, s.BaselineMedianAbsErrorHrs = median(modelHours), median(baselineHours)
	s.OutcomeAccuracy, s.BaselineOutcomeAccuracy = float64(modelOK)/float64(s.Windows), float64(baselineOK)/float64(s.Windows)
	s.BeatsBaseline = s.MedianAbsErrorPoints < s.BaselineMedianAbsErrorPts
	return s
}
