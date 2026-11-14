package usageforecast

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/pricing"
)

func onlyForecast(t *testing.T, readings []pricing.QuotaReadingSpan, now time.Time) Forecast {
	t.Helper()
	forecasts := Current(readings, now)
	if len(forecasts) != 1 {
		t.Fatalf("forecasts=%d want 1: %+v", len(forecasts), forecasts)
	}
	return forecasts[0]
}

// The real Codex window on 2026-10-29 runs out ~Oct 30–31, days before its
// Nov 4 reset; Agent Manager must say will_exhaust with a band around that.
func TestForecastProjectsTheOct28CodexWindowToExhaustBeforeReset(t *testing.T) {
	now := at("2026-10-29T04:40:00Z")
	f := onlyForecast(t, spans(oct28Shape()), now)
	if f.Status != StatusWillExhaust {
		t.Fatalf("status=%s (%s) want will_exhaust", f.Status, f.Reason)
	}
	if f.Used != 33 || !f.Window.ResetAt.Equal(at("2026-11-04T06:57:00Z")) || !f.Window.Start.Equal(at("2026-10-28T06:57:00Z")) {
		t.Fatalf("used=%v reset=%v start=%v", f.Used, f.Window.ResetAt, f.Window.Start)
	}
	burn := map[time.Duration]Burn{}
	for _, b := range f.Burn {
		burn[b.Horizon] = b
	}
	if b := burn[6*time.Hour]; !b.Available || !near(b.Rate, 3.33, 0.1) {
		t.Fatalf("6h burn=%+v want ~3.3%%/h", b)
	}
	// Horizons end now (04:40Z), not at the newest reading (04:30Z).
	if b := burn[24*time.Hour]; !b.Available || !near(b.Rate, 33/21.72, 0.01) || !near(b.Span.Hours(), 21.72, 0.01) {
		t.Fatalf("24h burn=%+v want 33%% over the window's 21.7h to now", b)
	}
	if f.BandStart == nil || f.BandEnd == nil || f.Projected == nil {
		t.Fatalf("band missing: %+v", f)
	}
	if f.BandStart.Before(at("2026-10-29T18:00:00Z")) || f.BandStart.After(at("2026-10-30T03:00:00Z")) {
		t.Fatalf("band start=%v want around Oct 30 00:00Z", f.BandStart)
	}
	if f.BandEnd.Before(at("2026-10-30T20:00:00Z")) || f.BandEnd.After(at("2026-10-31T06:00:00Z")) {
		t.Fatalf("band end=%v want around Oct 31 00:00Z", f.BandEnd)
	}
	if f.Projected.Before(*f.BandStart) || f.Projected.After(*f.BandEnd) {
		t.Fatalf("projected %v outside band %v..%v", f.Projected, f.BandStart, f.BandEnd)
	}
	if f.Lasting == nil || !near(*f.Lasting, 0.46, 0.01) {
		t.Fatalf("lasting=%v want ≤0.46%%/h (67%% over the 146h to reset)", f.Lasting)
	}
	if len(f.Window.Series()) < 2 || f.Window.Series()[0].Percent != 0 || !f.Window.Series()[0].At.Equal(f.Window.Start) {
		t.Fatalf("series must start at the window start at 0%%: %v", f.Window.Series()[:1])
	}
}

func TestForecastDetectsEarlyResetsAndClassifiesStatus(t *testing.T) {
	history := append(earlyResetHistory(), oct28Shape())
	windows := Windows(spans(history...))
	if len(windows) != 5 {
		t.Fatalf("windows=%d want 5", len(windows))
	}
	for i, start := range []string{"2026-10-22T17:09:00Z", "2026-10-23T18:02:00Z", "2026-10-25T06:10:00Z", "2026-10-26T06:08:00Z", "2026-10-28T06:57:00Z"} {
		if !windows[i].Start.Equal(at(start)) {
			t.Errorf("window %d start=%v want %s", i, windows[i].Start, start)
		}
	}
	f := onlyForecast(t, spans(history...), at("2026-10-29T04:40:00Z"))
	if !f.EarlyReset || f.Previous == nil || f.Previous.Final != 27 || !f.Previous.ResetAt.Equal(at("2026-11-02T06:08:00Z")) || !f.Previous.EndedAt.Equal(at("2026-10-28T06:57:00Z")) {
		t.Fatalf("early reset note: early=%v previous=%+v", f.EarlyReset, f.Previous)
	}
	if f.Status != StatusWillExhaust {
		t.Fatalf("status=%s (%s)", f.Status, f.Reason)
	}

	reset := at("2026-11-04T06:57:00Z")
	start := reset.Add(-7 * 24 * time.Hour)
	for _, tc := range []struct {
		name   string
		window windowSpec
		now    time.Time
		want   string
	}{
		{"slow burn lasts to reset", codexWindow(reset, start.Add(48*time.Hour), ramp(start.Add(time.Hour), start.Add(48*time.Hour), 1, 10)), start.Add(48 * time.Hour), StatusOnTrack},
		{"band straddles reset", codexWindow(reset, start.Add(48*time.Hour), ramp(start.Add(time.Hour), start.Add(48*time.Hour), 1, 28)), start.Add(48 * time.Hour), StatusAtRisk},
		{"fast burn", codexWindow(reset, start.Add(24*time.Hour), ramp(start.Add(time.Hour), start.Add(24*time.Hour), 1, 40)), start.Add(24 * time.Hour), StatusWillExhaust},
		{"exhausted", codexWindow(reset, start.Add(30*time.Hour), ramp(start.Add(time.Hour), start.Add(30*time.Hour), 1, 100)), start.Add(30 * time.Hour), StatusWillExhaust},
		{"stale beyond 6h", codexWindow(reset, start.Add(24*time.Hour), ramp(start.Add(time.Hour), start.Add(24*time.Hour), 1, 40)), start.Add(31 * time.Hour), StatusUnknown},
		{"reset passed", codexWindow(reset, start.Add(24*time.Hour), ramp(start.Add(time.Hour), start.Add(24*time.Hour), 1, 40)), reset.Add(time.Minute), StatusUnknown},
		{"too little history", codexWindow(reset, start.Add(5*time.Minute), []step{{start.Add(time.Minute), 1}}), start.Add(5 * time.Minute), StatusUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := onlyForecast(t, spans(tc.window), tc.now)
			if got.Status != tc.want {
				t.Fatalf("status=%s (%s) want %s; band=%v..%v", got.Status, got.Reason, tc.want, got.BandStart, got.BandEnd)
			}
		})
	}
	if forecasts := Current(nil, at("2026-10-29T00:00:00Z")); len(forecasts) != 0 {
		t.Fatalf("no samples must produce no windows, got %d", len(forecasts))
	}
}

func TestForecastNeverSplitsAWindowOnAPercentDropUnderTheSameReset(t *testing.T) {
	reset := at("2026-11-04T06:57:00Z")
	start := reset.Add(-7 * 24 * time.Hour)
	// A lagging reader reports 0..9% while the window already stood at 20%:
	// the reset time is unchanged, so it is the same window at its maximum.
	readings := spans(codexWindow(reset, start.Add(30*time.Hour),
		ramp(start.Add(time.Hour), start.Add(20*time.Hour), 1, 20),
		ramp(start.Add(21*time.Hour), start.Add(30*time.Hour), 0, 9)))
	windows := Windows(readings)
	if len(windows) != 1 || windows[0].Max() != 20 || !windows[0].Start.Equal(start) {
		t.Fatalf("a lower percent under the same reset must not start a window: %d windows", len(windows))
	}
	if f := onlyForecast(t, readings, start.Add(30*time.Hour)); f.EarlyReset || f.Previous != nil || f.Used != 20 {
		t.Fatalf("no early reset expected: early=%v previous=%+v used=%v", f.EarlyReset, f.Previous, f.Used)
	}
}

// The live Claude Code statusline feed on 2026-10-29 06:41Z (F-ORCH-3): four
// sessions write samples. Three run an old model whose cached rate_limits
// carry resets_at long past; two current sessions report 46 and 63-64% for
// the same five-hour window.
func TestForecastUsesFreshClaudeWindowsAcrossConcurrentSessions(t *testing.T) {
	now := at("2026-10-29T06:42:00Z")
	var readings []pricing.QuotaReadingSpan
	for _, s := range []struct {
		observed string
		five     float64
		fiveAt   int64
		seven    float64
		sevenAt  int64
	}{
		{"2026-10-29T06:41:06Z", 46, 1793268600, 21, 1793804400},
		{"2026-10-29T06:41:08Z", 5, 1789775400, 24, 1790175600},  // expired: Sep 18 / Sep 23
		{"2026-10-29T06:41:09Z", 57, 1792692600, 18, 1793199600}, // expired: Oct 22 / Oct 28
		{"2026-10-29T06:41:09.5Z", 4, 1792021800, 4, 1792594800}, // expired: Oct 14 / Oct 21
		{"2026-10-29T06:41:11Z", 63, 1793268600, 23, 1793804400},
		{"2026-10-29T06:41:20Z", 46, 1793268600, 21, 1793804400}, // the lagging session again
		{"2026-10-29T06:41:41Z", 64, 1793268600, 23, 1793804400},
	} {
		observed := at(s.observed)
		readings = append(readings, claudeSpan("five_hour", observed, s.five, s.fiveAt), claudeSpan("seven_day", observed, s.seven, s.sevenAt))
	}
	forecasts := Current(readings, now)
	if len(forecasts) != 2 {
		t.Fatalf("forecasts=%d want five_hour and seven_day", len(forecasts))
	}
	want := map[string]struct {
		used  float64
		reset string
	}{"five_hour": {64, "2026-10-29T10:10:00Z"}, "seven_day": {23, "2026-11-04T15:00:00Z"}}
	for _, f := range forecasts {
		w, ok := want[f.Window.Pool]
		if !ok {
			t.Fatalf("unexpected scope %s", f.Window.Scope())
		}
		if f.Used != w.used || !f.Window.ResetAt.Equal(at(w.reset)) {
			t.Errorf("%s: used=%v reset=%v want %v%% resets %s", f.Window.Pool, f.Used, f.Window.ResetAt, w.used, w.reset)
		}
		if f.EarlyReset || f.Previous != nil {
			t.Errorf("%s: expired cached windows must not become history or an early reset: early=%v previous=%+v", f.Window.Pool, f.EarlyReset, f.Previous)
		}
		if strings.Contains(f.Reason, "no reading of the new window") {
			t.Errorf("%s: the current window has not reset: %q", f.Window.Pool, f.Reason)
		}
	}
	// Before the 64% reading, the lagging session's later 46% does not lower
	// the window: it stands at the 63% maximum.
	if f := Current(readings[:12], now); f[0].Window.Pool != "five_hour" || f[0].Used != 63 || f[1].Used != 23 {
		t.Fatalf("five_hour=%v seven_day=%v want 63/23", f[0].Used, f[1].Used)
	}
	if windows := Windows(readings); len(windows) != 2 {
		t.Fatalf("windows=%d want 2 (expired samples dropped)", len(windows))
	}
}

func TestForecastStartsANewClaudeWindowOnlyAtANewResetTime(t *testing.T) {
	// A fresh five-hour window followed by the next one: on time, not early.
	first, second := at("2026-10-29T10:10:00Z"), at("2026-10-29T15:10:00Z")
	readings := []pricing.QuotaReadingSpan{
		claudeSpan("five_hour", at("2026-10-29T06:00:00Z"), 40, first.Unix()),
		claudeSpan("five_hour", at("2026-10-29T10:00:00Z"), 90, first.Unix()),
		claudeSpan("five_hour", at("2026-10-29T10:20:00Z"), 3, second.Unix()),
		claudeSpan("five_hour", at("2026-10-29T10:21:00Z"), 91, first.Unix()), // expired cached sample
	}
	f := onlyForecast(t, readings, at("2026-10-29T10:30:00Z"))
	if !f.Window.ResetAt.Equal(second) || f.Used != 3 || f.EarlyReset {
		t.Fatalf("current=%v used=%v early=%v want the 15:10 window at 3%%, no early reset", f.Window.ResetAt, f.Used, f.EarlyReset)
	}
	if f.Previous == nil || f.Previous.Final != 90 || !f.Previous.Observed || !f.Previous.EndedAt.Equal(first) {
		t.Fatalf("previous=%+v want 90%% observed, ended at %v", f.Previous, first)
	}
}

// AM's own Codex readings of the window that reset early on 10-28 stop at
// 2% on 10-26 14:10; the window actually ended at 27%. The note must report
// the highest reading and mark it as not the observed outcome.
func TestForecastMarksASparselyReadPreviousWindowAsALastReading(t *testing.T) {
	prevReset, curReset := at("2026-11-02T06:08:00Z"), time.Unix(1793775438, 0).UTC()
	prevStart := prevReset.Add(-7 * 24 * time.Hour)
	readings := spans(
		codexWindow(prevReset, at("2026-10-26T14:10:00Z"), []step{{prevStart.Add(time.Minute), 0}, {at("2026-10-26T09:00:00Z"), 1}, {at("2026-10-26T12:00:00Z"), 2}}),
		codexWindow(curReset, at("2026-10-29T01:49:00Z"), ramp(at("2026-10-28T20:54:00Z"), at("2026-10-29T01:49:00Z"), 9, 27)))
	f := onlyForecast(t, readings, at("2026-10-29T02:00:00Z"))
	if !f.EarlyReset || f.Previous == nil || f.Previous.Final != 2 || f.Previous.Observed || !f.Previous.LastAt.Equal(at("2026-10-26T14:10:00Z")) {
		t.Fatalf("previous=%+v early=%v want last reading 2%% not observed as final", f.Previous, f.EarlyReset)
	}
	if !f.Window.ResetAt.Equal(curReset.Truncate(time.Minute)) || f.Used != 27 {
		t.Fatalf("current reset=%v used=%v", f.Window.ResetAt, f.Used)
	}
}

func TestForecastGroupsJitteredResetTimesIntoOneWindow(t *testing.T) {
	reset := at("2026-11-04T06:57:59Z")
	readings := spans(codexWindow(reset, at("2026-10-29T04:00:00Z"), ramp(at("2026-10-28T08:00:00Z"), at("2026-10-29T04:00:00Z"), 1, 20)))
	jittered := at("2026-11-04T06:58:00Z")
	for i := range readings {
		if i%2 == 1 {
			readings[i].ResetAt = &jittered
		}
	}
	if windows := Windows(readings); len(windows) != 1 || windows[0].Max() != 20 {
		t.Fatalf("jittered resets_at must group as one window, got %d", len(windows))
	}
}

func TestForecastReportsStandingOnlyWindowsFromStanding(t *testing.T) {
	now := at("2026-10-29T12:00:00Z")
	reset := now.Add(3 * time.Hour)
	for standing, want := range map[pricing.QuotaStanding]string{
		pricing.QuotaStandingExhausted: StatusWillExhaust,
		pricing.QuotaStandingLimited:   StatusAtRisk,
		pricing.QuotaStandingAvailable: StatusOnTrack,
		pricing.QuotaStandingUnknown:   StatusUnknown,
	} {
		readings := []pricing.QuotaReadingSpan{{Provider: "anthropic", Pool: "five_hour", Window: "five_hour", ResetAt: &reset, Standing: standing, FirstAt: now.Add(-time.Hour), LastAt: now.Add(-time.Minute), Samples: 2}}
		f := onlyForecast(t, readings, now)
		if f.Status != want || f.Window.HasPercent || f.Projected != nil || len(f.Burn) != 0 {
			t.Errorf("%s: status=%s hasPercent=%v projected=%v", standing, f.Status, f.Window.HasPercent, f.Projected)
		}
		if stale := onlyForecast(t, readings, now.Add(7*time.Hour)); stale.Status != StatusUnknown {
			t.Errorf("%s: stale standing must be unknown, got %s", standing, stale.Status)
		}
	}
}

// F-ORCH-5: a trailing horizon with no reading at or before its start is
// unknown, not "0% then". Live shape 2026-10-29: Claude five_hour window
// began 05:10Z, first reading 06:41Z, now 07:03Z.
func TestForecastLeavesTrailingHorizonsWithoutAStartingReadingUnknown(t *testing.T) {
	reset := at("2026-10-29T10:10:00Z")
	readings := []pricing.QuotaReadingSpan{
		claudeSpan("five_hour", at("2026-10-29T06:41:00Z"), 63, reset.Unix()),
		claudeSpan("five_hour", at("2026-10-29T06:49:00Z"), 67, reset.Unix()),
		claudeSpan("five_hour", at("2026-10-29T06:55:00Z"), 69, reset.Unix()),
	}
	f := onlyForecast(t, readings, at("2026-10-29T07:03:00Z"))
	burn := map[time.Duration]Burn{}
	for _, b := range f.Burn {
		burn[b.Horizon] = b
	}
	if b := burn[time.Hour]; b.Available {
		t.Fatalf("1h burn=%+v want unknown: no reading at or before 06:03Z", b)
	}
	for _, h := range []time.Duration{6 * time.Hour, 24 * time.Hour} {
		b := burn[h]
		if !b.Available || !near(b.Span.Hours(), 1.883, 0.01) || !near(b.Rate, 69/1.883, 0.1) {
			t.Fatalf("%v burn=%+v want defined from the implicit 05:10Z start", h, b)
		}
	}
	if !near(f.Central, 69/1.75, 0.1) {
		t.Fatalf("central=%v must come from a defined horizon", f.Central)
	}
}

func TestForecastWithOnlyUnknownHorizonsFallsBackToTheRateSinceWindowStart(t *testing.T) {
	// Claude seven_day window began 10-26 12:00Z; the first reading arrives
	// at 12:20Z on 10-29, so no horizon has a reading at its start.
	reset := at("2026-11-02T12:00:00Z")
	readings := []pricing.QuotaReadingSpan{
		claudeSpan("seven_day", at("2026-10-29T12:20:00Z"), 30, reset.Unix()),
		claudeSpan("seven_day", at("2026-10-29T12:40:00Z"), 33, reset.Unix()),
	}
	f := onlyForecast(t, readings, at("2026-10-29T12:45:00Z"))
	for _, b := range f.Burn {
		if b.Available {
			t.Fatalf("burn=%+v want every horizon unknown", b)
		}
	}
	if want := 33 / 72.667; !near(f.Central, want, 0.01) {
		t.Fatalf("central=%v want the since-start rate %v", f.Central, want)
	}
	if f.Status == StatusWillExhaust || !strings.Contains(f.Reason, "since the window began") {
		t.Fatalf("status=%s reason=%q want on_track-ish with the fallback named", f.Status, f.Reason)
	}
}

// Codex rollouts before 2025-11 carried resets_at null and window lengths of
// 299/10079 minutes. Their readings once grouped into one "5h" window
// spanning Oct 30 → Nov 23; a window must never outlast its nominal length,
// and a drop in used percent starts the next one.
func TestForecastSplitsWindowsWithoutAResetTimeByLengthAndPercentDrop(t *testing.T) {
	day := at("2025-10-30T12:00:00Z")
	var specs []windowSpec
	for d := 0; d < 3; d++ {
		start := day.Add(time.Duration(d) * 24 * time.Hour)
		// Three 5h windows on one day whose readings repeat the same
		// percents: only the percent drop or the length separates them.
		for h := 0; h < 3; h++ {
			s := start.Add(time.Duration(h) * 5 * time.Hour)
			specs = append(specs, windowSpec{provider: "openai", pool: "primary", minutes: 299, steps: ramp(s.Add(time.Minute), s.Add(4*time.Hour), 3, 30), last: s.Add(4*time.Hour + 30*time.Minute)})
		}
	}
	windows := Windows(spans(specs...))
	if len(windows) != 9 {
		t.Fatalf("windows=%d want 9", len(windows))
	}
	for _, w := range windows {
		if w.Window != "300m" || w.WindowMinutes != 300 {
			t.Fatalf("window length must round to the hour: %s %d", w.Window, w.WindowMinutes)
		}
		if length := w.LastAt.Sub(w.Start); length > 5*time.Hour+lengthTolerance || w.Max() != 30 {
			t.Fatalf("window %v..%v spans %v at %v%% (at most 5h, ending at 30%%)", w.Start, w.LastAt, length, w.Max())
		}
	}
	// One flat reading held for a day is cut at the nominal length.
	flat := Windows(spans(windowSpec{provider: "openai", pool: "primary", minutes: 300, steps: []step{{day, 7}}, last: day.Add(24 * time.Hour)}))
	if len(flat) != 1 || !flat[0].LastAt.Equal(day.Add(5*time.Hour)) {
		t.Fatalf("a held reading must end at the nominal length: %+v", flat[0])
	}
}

// Horizons and run-out are anchored at now. A horizon with no reading inside
// it is stale, not a current rate, and the last reading is reported.
func TestForecastAnchorsAtNowAndMarksStaleHorizons(t *testing.T) {
	shape := oct28Shape() // newest reading 04:30Z at 33%
	now := at("2026-10-29T06:00:00Z")
	f := onlyForecast(t, spans(shape), now)
	if !f.Anchor.Equal(now) || !f.AsOf.Equal(at("2026-10-29T04:30:00Z")) || f.Window.LastPercent != 33 {
		t.Fatalf("anchor=%v asOf=%v last=%v", f.Anchor, f.AsOf, f.Window.LastPercent)
	}
	burn := map[time.Duration]Burn{}
	for _, b := range f.Burn {
		burn[b.Horizon] = b
	}
	if b := burn[time.Hour]; !b.Stale || b.Available {
		t.Fatalf("1h burn=%+v: the newest reading is 1.5h old, so it is stale", b)
	}
	if b := burn[6*time.Hour]; b.Stale || !b.Available || !near(b.Rate, (33-18)/6.0, 0.05) {
		t.Fatalf("6h burn=%+v want 18→33%% over 00:00–06:00Z", b)
	}
	// The run-out moves with the anchor: the same window read 1.5h later
	// with no new reading projects 1.5h later.
	early := onlyForecast(t, spans(shape), at("2026-10-29T04:30:00Z"))
	if f.Projected == nil || early.Projected == nil || !near(f.Projected.Sub(*early.Projected).Hours(), 1.5, 0.01) {
		t.Fatalf("projected %v vs %v want 1.5h later", f.Projected, early.Projected)
	}
}

// A lagging reader after the maximum: the last reading is what it reported,
// the window stands at its maximum.
func TestForecastReportsTheLastReadingBesideTheMaximum(t *testing.T) {
	reset := at("2026-10-29T10:10:00Z")
	readings := []pricing.QuotaReadingSpan{
		claudeSpan("five_hour", at("2026-10-29T06:41:00Z"), 63, reset.Unix()),
		claudeSpan("five_hour", at("2026-10-29T06:49:00Z"), 46, reset.Unix()),
	}
	f := onlyForecast(t, readings, at("2026-10-29T06:50:00Z"))
	if f.Used != 63 || f.Window.LastPercent != 46 || !f.AsOf.Equal(at("2026-10-29T06:49:00Z")) {
		t.Fatalf("used=%v last=%v at %v", f.Used, f.Window.LastPercent, f.AsOf)
	}
}

func TestForecastDropsScopesNoLongerRead(t *testing.T) {
	// F-ORCH-7: the Codex primary 5h window that reset 2026-09-09 and the
	// secondary weekly window that reset 2026-09-16 are scopes no longer read;
	// on 2026-10-29 they are not current windows. A window whose reset passed
	// less than one nominal length ago is still current ("no reading of the
	// new window yet").
	now := at("2026-10-29T11:00:00Z")
	fiveHourReset := at("2026-09-09T14:00:00Z")
	deadFive := windowSpec{provider: "openai", pool: "primary", minutes: 300, reset: fiveHourReset, steps: ramp(fiveHourReset.Add(-4*time.Hour), fiveHourReset.Add(-time.Hour), 1, 30), last: fiveHourReset.Add(-time.Hour)}
	weekReset := at("2026-09-16T08:00:00Z")
	deadWeek := windowSpec{provider: "openai", pool: "secondary", minutes: 10080, reset: weekReset, steps: ramp(weekReset.Add(-48*time.Hour), weekReset.Add(-time.Hour), 1, 40), last: weekReset.Add(-time.Hour)}
	recentReset := now.Add(-3 * time.Hour)
	recent := windowSpec{provider: "anthropic", pool: "five_hour", minutes: 300, reset: recentReset, steps: ramp(recentReset.Add(-4*time.Hour), recentReset.Add(-time.Hour), 1, 20), last: recentReset.Add(-time.Hour)}
	live := codexWindow(now.Add(5*24*time.Hour), now.Add(-10*time.Minute), ramp(now.Add(-48*time.Hour), now.Add(-10*time.Minute), 1, 20))

	forecasts := Current(spans(deadFive, deadWeek, recent, live), now)
	scopes := map[string]Forecast{}
	for _, f := range forecasts {
		scopes[f.Window.Scope()] = f
	}
	if len(scopes) != 2 {
		t.Fatalf("scopes=%v want only the live weekly and the just-reset five-hour window", keys(scopes))
	}
	if f, ok := scopes["anthropic|five_hour|300m"]; !ok || !strings.Contains(f.Reason, "no reading of the new window") {
		t.Fatalf("just-reset scope must stay current and unknown: %+v", f)
	}
	if _, ok := scopes["openai|primary|10080m"]; !ok {
		t.Fatalf("live scope missing: %v", keys(scopes))
	}
}

func keys(m map[string]Forecast) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestStaleWindowIsIdleOnlyWhenNoWorkSinceTheLastReading(t *testing.T) {
	reset := at("2026-11-04T06:57:00Z")
	start := reset.Add(-7 * 24 * time.Hour)
	last := start.Add(24 * time.Hour)
	readings := spans(codexWindow(reset, last, ramp(start.Add(time.Hour), last, 1, 40)))
	now := last.Add(7 * time.Hour)
	cases := []struct {
		name   string
		source ActivitySource
		want   string
	}{
		{"no source cannot rule out unmetered use", nil, StatusUnknown},
		{"source failed", stubActivity{err: context.DeadlineExceeded}, StatusUnknown},
		{"work since the reading", stubActivity{last: last.Add(time.Hour)}, StatusUnknown},
		{"no work since the reading", stubActivity{last: last.Add(-time.Hour)}, StatusIdle},
		{"never any work", stubActivity{}, StatusIdle},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CurrentWithActivity(context.Background(), readings, now, c.source)
			if len(got) != 1 || got[0].Status != c.want {
				t.Fatalf("status=%v, want %s", got, c.want)
			}
			if c.want == StatusIdle && (got[0].Lasting == nil || !strings.Contains(got[0].Reason, "lasts at this pace")) {
				t.Fatalf("idle must say it lasts at this pace: %+v", got[0])
			}
		})
	}
	// A fresh reading is never idle, and a reset window stays unknown.
	if got := CurrentWithActivity(context.Background(), readings, last.Add(time.Hour), stubActivity{}); got[0].Status == StatusIdle {
		t.Fatalf("fresh reading: %s", got[0].Status)
	}
	if got := CurrentWithActivity(context.Background(), readings, reset.Add(time.Minute), stubActivity{}); got[0].Status != StatusUnknown {
		t.Fatalf("reset passed: %s", got[0].Status)
	}
}

func TestWindowsSeparateVerifiedAccountsAndUnknownHistory(t *testing.T) {
	now := time.Date(2026, 10, 29, 1, 0, 0, 0, time.UTC)
	reset := now.Add(5 * time.Hour)
	percent := 25.0
	var spans []pricing.QuotaReadingSpan
	for _, account := range []string{"", "account-a", "account-b"} {
		spans = append(spans, pricing.QuotaReadingSpan{AccountScope: account, Provider: "openai", Pool: "primary", Window: "300m", ResetAt: &reset, UsedPercent: &percent, Standing: pricing.QuotaStandingAvailable, FirstAt: now, LastAt: now})
	}
	windows := Windows(spans)
	if len(windows) != 3 {
		t.Fatalf("windows=%+v; identical observations from separate scopes must not merge", windows)
	}
	seen := make(map[string]bool)
	for _, window := range windows {
		seen[window.AccountScope] = true
	}
	if !seen[""] || !seen["account-a"] || !seen["account-b"] {
		t.Fatalf("missing scope: %+v", seen)
	}
}

// A saving comes out of the window's remaining burn in the shape the burn is
// projected in: none leaves the forecast's own run-out; enough to bring the
// end under 100% lasts until the reset; in between the run-out moves later
// but still short of the reset; the cut needed is against the measured 24 h
// rate. A window not at risk has no projected outcome.
func TestWithSavingProjectsTheWindowWithTheLeversApplied(t *testing.T) {
	f := onlyForecast(t, spans(oct28Shape()), at("2026-10-29T04:40:00Z"))
	reset := f.Window.ResetAt
	none := f.WithSaving(0)
	if !none.Projected || none.ReachesReset || none.RunOut == nil || !none.RunOut.Equal(*f.Projected) || none.Shortfall != reset.Sub(*f.Projected) {
		t.Fatalf("no saving = %+v, want the forecast's own run-out %v", none, f.Projected)
	}
	if none.CutNeeded == nil || !near(*none.CutNeeded, 1-*f.Lasting/(33/21.72), 0.01) {
		t.Fatalf("cut needed = %v, want against the 24h rate", none.CutNeeded)
	}
	hot := f
	hot.Burn = append([]Burn(nil), f.Burn...)
	for i := range hot.Burn {
		if hot.Burn[i].Horizon == 24*time.Hour {
			hot.Burn[i].Rate = 3 // twice the central burn: the cut follows what is burning now
		}
	}
	if cut := hot.WithSaving(0).CutNeeded; cut == nil || !near(*cut, 1-*f.Lasting/3, 0.001) {
		t.Fatalf("cut needed at a 3%%/h 24h rate = %v, want %v", cut, 1-*f.Lasting/3)
	}
	over := none.EndPercent - 100 // the points that must go to last until the reset
	partial := f.WithSaving(over / 2)
	if !partial.Projected || partial.ReachesReset || partial.RunOut == nil || !partial.RunOut.After(*none.RunOut) || !partial.RunOut.Before(reset) || partial.Shortfall >= none.Shortfall || !near(partial.EndPercent, 100+over/2, 0.001) {
		t.Fatalf("partial saving = %+v, want a later run-out still before %v", partial, reset)
	}
	if enough := f.WithSaving(over + 1); !enough.ReachesReset || enough.RunOut != nil || !near(enough.EndPercent, 99, 0.001) {
		t.Fatalf("enough saving = %+v, want it to last at 99%%", enough)
	}
	if all := f.WithSaving(1e6); all.EndPercent != f.Used {
		t.Fatalf("a saving larger than the burn ends at %v, want the used %v", all.EndPercent, f.Used)
	}
	f.Status = StatusOnTrack
	if calm := f.WithSaving(5); calm.Projected || calm.CutNeeded != nil {
		t.Fatalf("on-track window = %+v, want no projected outcome", calm)
	}
}

// The usage surfaces read the windows as the API marks them: a window whose
// newest reading predates its period is dormant (no reading at all is not);
// each provider pool's longest window is primary, so a provider's second pool
// keeps its own; the most urgent window with a reading in its period is the
// focus.
func TestMarkNamesDormantPrimaryAndFocusWindows(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	at := func(h int) *time.Time { t := now.Add(time.Duration(h) * time.Hour); return &t }
	window := func(provider, pool string, minutes int64, start, reset time.Time) *Window {
		return &Window{Provider: provider, Pool: pool, Window: fmt.Sprint(minutes), WindowMinutes: minutes, Start: start, ResetAt: reset}
	}
	week, day := now.Add(-48*time.Hour), now.Add(-2*time.Hour)
	forecasts := []Forecast{
		{Window: window("openai", "codex", 300, day, now.Add(3*time.Hour)), AsOf: now.Add(-time.Hour), Status: StatusOnTrack, Used: 80},
		{Window: window("openai", "codex", 10080, week, now.Add(5*24*time.Hour)), AsOf: now.Add(-time.Hour), Status: StatusAtRisk, Used: 40, Projected: at(30)},
		{Window: window("openai", "spark", 10080, week, now.Add(5*24*time.Hour)), AsOf: now.Add(-time.Hour), Status: StatusWillExhaust, Used: 70, Projected: at(20)},
		{Window: window("anthropic", "", 10080, week, now.Add(-time.Hour)), AsOf: now.Add(-3 * time.Hour), Status: StatusWillExhaust, Used: 99, Projected: at(1)},   // reset passed: dormant
		{Window: window("anthropic", "opus", 300, day, now.Add(3*time.Hour)), AsOf: now.Add(-5 * time.Hour), Status: StatusWillExhaust, Used: 90, Projected: at(2)}, // reading before the period: dormant
		{Window: window("google", "", 1440, day, now.Add(20*time.Hour)), Status: StatusUnknown},                                                                     // no reading yet
	}
	marks := Mark(forecasts, now)
	want := []Marks{{Primary: false}, {Primary: true}, {Primary: true, Focus: true}, {Dormant: true, Primary: true}, {Dormant: true, Primary: true}, {Primary: true}}
	for i := range want {
		if marks[i] != want[i] {
			t.Errorf("window %d (%s %s %dm): marks %+v, want %+v", i, forecasts[i].Window.Provider, forecasts[i].Window.Pool, forecasts[i].Window.WindowMinutes, marks[i], want[i])
		}
	}
	if got := Mark(forecasts[:2], now); !got[1].Focus || got[0].Focus {
		t.Errorf("among live windows the one that may run out is the focus before a quieter one more used: %+v", got)
	}
	if got := Mark([]Forecast{forecasts[1], forecasts[0]}, now); !got[0].Focus {
		t.Errorf("a window that may run out stays the focus whatever is listed after it: %+v", got)
	}
	later, sooner := forecasts[2], forecasts[2]
	later.Projected, sooner.Projected = at(40), at(10)
	if got := Mark([]Forecast{later, sooner}, now); !got[1].Focus {
		t.Errorf("of two windows that will run out, the sooner is the focus: %+v", got)
	}
}
