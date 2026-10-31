package usageforecast

import (
	"testing"
	"time"
)

func summaryOf(t *testing.T, report Report, population string, fraction float64) Summary {
	t.Helper()
	for _, s := range report.Summaries {
		if s.Population == population && s.Fraction == fraction {
			return s
		}
	}
	t.Fatalf("no %s summary at %.2f", population, fraction)
	return Summary{}
}

func summaryAt(t *testing.T, report Report, fraction float64) Summary {
	t.Helper()
	return summaryOf(t, report, PopulationAll, fraction)
}

// SPEC A4: the calibrated hour-of-week projection beats the naive linear
// baseline on an operator whose burn follows a daily rhythm, both on the last
// five windows per scope and on the whole history.
func TestBacktestBeatsLinearBaselineOnTheOperatorsBurnProfile(t *testing.T) {
	// Five weeks: the profile is trusted in full after four.
	const days = 35
	from := at("2026-09-01T00:00:00Z")
	report := Backtest(spans(shiftOperator(from, days)...), from.Add((days+1)*24*time.Hour))
	if len(report.Windows) != 2*days || report.Unobserved != 0 {
		t.Fatalf("windows=%d unobserved=%d want %d observed", len(report.Windows), report.Unobserved, 2*days)
	}
	for _, fraction := range Checkpoints {
		for _, population := range []string{PopulationRecent, PopulationAll} {
			s := summaryOf(t, report, population, fraction)
			if !s.BeatsBaseline || s.MedianAbsErrorPoints >= s.BaselineMedianAbsErrorPts {
				t.Errorf("%s @%.2f: model %.1f vs baseline %.1f pts must beat it", population, fraction, s.MedianAbsErrorPoints, s.BaselineMedianAbsErrorPts)
			}
		}
		if s := summaryOf(t, report, PopulationRecent, fraction); s.Windows != ScoredWindowsPerScope {
			t.Errorf("recent population @%.2f scored %d windows want the last %d of the one scope", fraction, s.Windows, ScoredWindowsPerScope)
		}
		if s := summaryOf(t, report, PopulationAll, fraction); s.Windows != 2*days {
			t.Errorf("all population @%.2f scored %d windows want %d", fraction, s.Windows, 2*days)
		}
	}
	// The newest evening window at 25%: linear expects the 10%/h burn to run
	// to 01:00; the profile knows the operator stops at 22:00.
	evening := report.Windows[len(report.Windows)-1].Checkpoints[0]
	if evening.Calibration.Windows < minCalibration || evening.BaselineAbsError < 15 || evening.AbsErrorPoints > 5 {
		t.Fatalf("evening checkpoint=%+v want a calibrated projection within 5 pts while linear is off by >15", evening)
	}
}

// The backtest is time-ordered: adding later windows never changes what an
// earlier checkpoint projected.
func TestBacktestNeverLooksAhead(t *testing.T) {
	from := at("2026-10-01T00:00:00Z")
	short := Backtest(spans(shiftOperator(from, 8)...), from.Add(9*24*time.Hour))
	long := Backtest(spans(shiftOperator(from, 14)...), from.Add(15*24*time.Hour))
	for i, w := range short.Windows {
		for j, c := range w.Checkpoints {
			if got := long.Windows[i].Checkpoints[j]; got.PredictedFinal != c.PredictedFinal || got.Calibration != c.Calibration {
				t.Fatalf("window %d checkpoint %d changed with later history: %+v vs %+v", i, j, got, c)
			}
		}
	}
}

func TestBacktestUsesEarlyResetsAsWindowEndsAndSkipsUnobservedOutcomes(t *testing.T) {
	history := earlyResetHistory()
	// The last completed window lost its readings 20h before its early reset,
	// so its final percent is only a lower bound.
	history[3].last = history[3].last.Add(-20 * time.Hour)
	history[3].steps = ramp(history[3].steps[0].at, history[3].last, 1, 12)
	report := Backtest(spans(append(history, oct28Shape())...), at("2026-10-29T04:40:00Z"))
	if len(report.Windows) != 4 {
		t.Fatalf("completed windows=%d want 4 (the current window is not complete)", len(report.Windows))
	}
	for i, final := range []float64{15, 8, 13, 12} {
		if w := report.Windows[i]; !w.EarlyReset || w.Final != final {
			t.Errorf("window %d early=%v final=%v want early reset at %v%%", i, w.EarlyReset, w.Final, final)
		}
	}
	if report.Unobserved != 1 || report.Windows[3].OutcomeObserved {
		t.Fatalf("unobserved=%d window3 observed=%v", report.Unobserved, report.Windows[3].OutcomeObserved)
	}
	if !report.Windows[0].EndedAt.Equal(at("2026-10-23T18:02:00Z")) {
		t.Fatalf("an early reset ends the window at the next window's start, got %v", report.Windows[0].EndedAt)
	}
	if s := summaryAt(t, report, 0.25); s.Windows != 3 {
		t.Fatalf("only observed outcomes are scored: %+v", s)
	}
}

func TestBacktestMeasuresExhaustionTimeErrorWhenAWindowRunsOut(t *testing.T) {
	reset := at("2026-10-21T00:00:00Z")
	start := reset.Add(-7 * 24 * time.Hour)
	exhausting := codexWindow(reset, start.Add(100*time.Hour), ramp(start.Add(time.Hour), start.Add(100*time.Hour), 1, 100))
	report := Backtest(spans(exhausting), reset.Add(time.Hour))
	w := report.Windows[0]
	if w.ActualExhaustion == nil || !w.ActualExhaustion.Equal(start.Add(100*time.Hour)) {
		t.Fatalf("actual exhaustion=%v", w.ActualExhaustion)
	}
	half := summaryAt(t, report, 0.5)
	if half.HourErrorWindows != 1 || half.MedianAbsErrorHours > 6 || half.OutcomeAccuracy != 1 {
		t.Fatalf("a steady burn to 100%% must be projected within hours: %+v", half)
	}
}
