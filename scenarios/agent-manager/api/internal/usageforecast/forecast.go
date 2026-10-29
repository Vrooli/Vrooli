// Package usageforecast projects, from the provider's own quota readings,
// whether each subscription window runs out before its reset (SPEC T-4). It
// reads pricing.QuotaReadingSpan summaries of quota_observations and never
// derives a reading from token counts.
package usageforecast

import (
	"fmt"
	"math"
	"sort"
	"time"

	"agent-manager/internal/pricing"
)

// Status values (docs/concepts/USAGE-INTELLIGENCE.md T-4).
const (
	StatusOnTrack     = "on_track"
	StatusAtRisk      = "at_risk"
	StatusWillExhaust = "will_exhaust"
	StatusUnknown     = "unknown"
)

const (
	// StaleAfter is the age after which the newest reading no longer
	// establishes the window's state.
	StaleAfter = pricing.QuotaFreshnessMaxAge
	// CurrentLookback bounds the read for current windows. It covers enough
	// completed windows of every scope to calibrate the projection (Codex
	// weekly windows reset early every one to three days; a weekly scope
	// without early resets needs CalibrationWindows weeks).
	CurrentLookback = 70 * 24 * time.Hour
	// CalibrationWindows is how many of the scope's newest completed windows
	// calibrate a projection; at least minCalibration are needed.
	CalibrationWindows = 10
	minCalibration     = 3
	// resetTolerance groups readings whose reset times differ by provider
	// jitter (1–5 s) or by the minute truncation of the span read.
	resetTolerance = 2 * time.Minute
	// lengthTolerance is how far a window without a reported reset time may
	// run past its nominal length before a reading starts the next window.
	lengthTolerance = 5 * time.Minute
	// resetDrop is the used-percent drop that ends a window without a reported
	// reset time (Codex rollouts before 2025-11 carried resets_at null).
	resetDrop = 2
	// minBurnSpan is the shortest history a burn rate is computed over.
	minBurnSpan = 10 * time.Minute
)

// Horizons are the trailing burn horizons.
var Horizons = []time.Duration{time.Hour, 6 * time.Hour, 24 * time.Hour}

// Point is one step of the monotone used-percent series: the first time the
// provider reported Percent.
type Point struct {
	At      time.Time
	Percent float64
}

type interval struct{ first, last time.Time }

// Window is one provider subscription window reconstructed from readings.
type Window struct {
	Provider      string
	Pool          string
	Window        string
	WindowMinutes int64
	// ResetAt is the provider's scheduled reset; zero when not reported.
	ResetAt time.Time
	Start   time.Time
	// StartKnown means Start is ResetAt − window length, where the provider
	// resets used percent to zero.
	StartKnown bool
	HasPercent bool
	Standing   pricing.QuotaStanding
	FirstAt    time.Time
	LastAt     time.Time
	Samples    int64
	// LastPercent is the used percent of the newest reading; it is lower
	// than Max when the newest reader lags (F-ORCH-3).
	LastPercent float64
	points      []Point
	readings    []interval
}

// Scope names the provider, pool and window length.
func (w *Window) Scope() string { return w.Provider + "|" + w.Pool + "|" + w.Window }

func (w *Window) nominal() time.Duration { return time.Duration(w.WindowMinutes) * time.Minute }

// Key identifies one window for alert deduplication.
func (w *Window) Key() string {
	anchor := w.Start
	if !w.ResetAt.IsZero() {
		anchor = w.ResetAt
	}
	return w.Scope() + "|" + anchor.UTC().Format("2006-01-02T15:04Z")
}

// Max is the highest used percent reported in the window.
func (w *Window) Max() float64 {
	if len(w.points) == 0 {
		return 0
	}
	return w.points[len(w.points)-1].Percent
}

// Series returns the step series, including the implicit zero at a known
// window start.
func (w *Window) Series() []Point {
	if !w.HasPercent {
		return nil
	}
	series := make([]Point, 0, len(w.points)+1)
	if w.StartKnown && (len(w.points) == 0 || w.points[0].At.After(w.Start)) {
		series = append(series, Point{At: w.Start})
	}
	return append(series, w.points...)
}

// origin is the earliest time the series has a value.
func (w *Window) origin() time.Time {
	if w.StartKnown || len(w.points) == 0 {
		return w.Start
	}
	return w.points[0].At
}

// valueAt is the step value at t (the last reported percent at or before t).
func (w *Window) valueAt(t time.Time) float64 {
	value := 0.0
	if !w.StartKnown && len(w.points) > 0 {
		value = w.points[0].Percent
	}
	for _, p := range w.points {
		if p.At.After(t) {
			break
		}
		value = p.Percent
	}
	return value
}

// truncate returns the window as a forecaster would have seen it at t, or
// false when no reading existed by then.
func (w *Window) truncate(t time.Time) (Window, bool) {
	out := *w
	out.points, out.readings, out.LastAt = nil, nil, time.Time{}
	for _, p := range w.points {
		if !p.At.After(t) {
			out.points = append(out.points, p)
		}
	}
	for _, r := range w.readings {
		if r.first.After(t) {
			continue
		}
		last := r.last
		if last.After(t) {
			last = t
		}
		out.readings = append(out.readings, interval{r.first, last})
		if last.After(out.LastAt) {
			out.LastAt = last
		}
	}
	return out, !out.LastAt.IsZero()
}

func (w *Window) add(span pricing.QuotaReadingSpan) {
	if w.ResetAt.IsZero() && w.WindowMinutes > 0 {
		// A window without a reset time lasts at most its nominal length.
		if end := w.Start.Add(w.nominal()); span.LastAt.After(end) {
			span.LastAt = end
		}
	}
	w.Samples += span.Samples
	if w.FirstAt.IsZero() || span.FirstAt.Before(w.FirstAt) {
		w.FirstAt = span.FirstAt
	}
	if !span.LastAt.Before(w.LastAt) {
		w.LastAt, w.Standing = span.LastAt, span.Standing
		if span.UsedPercent != nil {
			w.LastPercent = *span.UsedPercent
		}
	}
	if w.WindowMinutes == 0 && span.WindowMinutes != nil {
		w.WindowMinutes = *span.WindowMinutes
	}
	w.readings = append(w.readings, interval{span.FirstAt, span.LastAt})
	if span.UsedPercent == nil {
		return
	}
	w.HasPercent = true
	if percent := *span.UsedPercent; percent > w.Max() {
		w.points = append(w.points, Point{At: span.FirstAt, Percent: percent})
	}
}

func newWindow(span pricing.QuotaReadingSpan) *Window {
	w := &Window{Provider: span.Provider, Pool: span.Pool, Window: span.Window, Start: span.FirstAt}
	if span.WindowMinutes != nil {
		w.WindowMinutes = *span.WindowMinutes
	}
	if span.ResetAt != nil {
		w.ResetAt = span.ResetAt.UTC()
		if w.WindowMinutes > 0 {
			w.Start, w.StartKnown = w.ResetAt.Add(-time.Duration(w.WindowMinutes)*time.Minute), true
			if w.Start.After(span.FirstAt) {
				w.Start = span.FirstAt
			}
		}
	}
	return w
}

// Windows groups reading spans into windows: by provider, pool and window
// length, then by reset time within resetTolerance. The reset time alone
// identifies a window. Used percent never splits one: concurrent Claude Code
// sessions cache different percents for the same window (46 and 63 under one
// resets_at), so within a window the series is the running maximum. A span
// whose reset had already passed when it was first observed is a cached value
// from an expired window and is dropped; a span that outlives its reset is
// cut at the reset. Readings without a reset time (single-reader Codex
// rollouts) start a new window when used percent drops by resetDrop points or
// the window has run its nominal length, so no window spans more than its
// nominal length. Window lengths are rounded to the hour (Codex reported 299
// and 10079 minutes for the 5h and weekly windows). Windows are returned
// ordered by reset time (a window without a reset time by its first reading),
// so the last window of a scope is the newest.
func Windows(spans []pricing.QuotaReadingSpan) []*Window {
	ordered := make([]pricing.QuotaReadingSpan, 0, len(spans))
	for _, span := range spans {
		span = roundLength(span)
		if span.ResetAt != nil && !span.ResetAt.IsZero() {
			// ResetAt is truncated to the minute; the provider reset lies
			// within the following minute.
			end := span.ResetAt.Add(time.Minute)
			if !span.FirstAt.Before(end) {
				continue
			}
			if span.LastAt.After(end) {
				span.LastAt = end
			}
		}
		ordered = append(ordered, span)
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].FirstAt.Before(ordered[j].FirstAt) })
	var windows []*Window
	byScope := map[string][]*Window{}
	for _, span := range ordered {
		scope := span.Provider + "|" + span.Pool + "|" + span.Window
		var match *Window
		if span.ResetAt == nil || span.ResetAt.IsZero() {
			match = continues(byScope[scope], span)
		} else {
			for _, candidate := range byScope[scope] {
				if sameReset(candidate.ResetAt, span.ResetAt) {
					match = candidate
					break
				}
			}
		}
		if match == nil {
			match = newWindow(span)
			windows, byScope[scope] = append(windows, match), append(byScope[scope], match)
		}
		match.add(span)
	}
	sort.SliceStable(windows, func(i, j int) bool { return windows[i].order().Before(windows[j].order()) })
	return windows
}

// continues returns the newest window without a reset time when span
// belongs to it: within its nominal length and without a used-percent drop.
func continues(windows []*Window, span pricing.QuotaReadingSpan) *Window {
	var last *Window
	for _, w := range windows {
		if w.ResetAt.IsZero() && (last == nil || w.FirstAt.After(last.FirstAt)) {
			last = w
		}
	}
	switch {
	case last == nil:
		return nil
	case last.WindowMinutes > 0 && span.FirstAt.Sub(last.FirstAt) > last.nominal()+lengthTolerance:
		return nil
	case span.UsedPercent != nil && *span.UsedPercent <= last.Max()-resetDrop:
		return nil
	}
	return last
}

// roundLength rounds a window length to the hour and relabels a window named
// by its length.
func roundLength(span pricing.QuotaReadingSpan) pricing.QuotaReadingSpan {
	if span.WindowMinutes == nil || *span.WindowMinutes <= 60 {
		return span
	}
	minutes := (*span.WindowMinutes + 30) / 60 * 60
	if span.Window == fmt.Sprintf("%dm", *span.WindowMinutes) {
		span.Window = fmt.Sprintf("%dm", minutes)
	}
	span.WindowMinutes = &minutes
	return span
}

// end is when the window ended, given the next window of its scope (nil when
// none), and false while it is still open at now. An early reset ends it at
// the next window's start; it never ends before its newest reading.
func (w *Window) end(next *Window, now time.Time) (time.Time, bool) {
	var end time.Time
	switch {
	case next != nil:
		end = next.Start
	case !w.ResetAt.IsZero() && !now.Before(w.ResetAt):
		end = w.ResetAt
	case w.ResetAt.IsZero() && w.WindowMinutes > 0 && !now.Before(w.Start.Add(w.nominal())):
		end = w.Start.Add(w.nominal())
	default:
		return time.Time{}, false
	}
	if !w.ResetAt.IsZero() && end.After(w.ResetAt) {
		end = w.ResetAt
	}
	if w.ResetAt.IsZero() && w.WindowMinutes > 0 && end.After(w.Start.Add(w.nominal())) {
		end = w.Start.Add(w.nominal())
	}
	if end.Before(w.LastAt) {
		end = w.LastAt
	}
	return end, true
}

// scheduledEnd is the end a forecaster assumes for an open window.
func (w *Window) scheduledEnd() time.Time {
	switch {
	case !w.ResetAt.IsZero():
		return w.ResetAt
	case w.WindowMinutes > 0:
		return w.Start.Add(w.nominal())
	}
	return time.Time{}
}

// dead reports that the window's scheduled end passed more than one nominal
// window length before now: a reading after the end would have opened a newer
// window, so the scope is no longer observed.
func (w *Window) dead(now time.Time) bool {
	end := w.scheduledEnd()
	return w.WindowMinutes > 0 && !end.IsZero() && now.Sub(end) > w.nominal()
}

// order is the window's position in its scope's history.
func (w *Window) order() time.Time {
	if w.ResetAt.IsZero() {
		return w.FirstAt
	}
	return w.ResetAt
}

func sameReset(window time.Time, reset *time.Time) bool {
	if reset == nil || reset.IsZero() {
		return window.IsZero()
	}
	if window.IsZero() {
		return false
	}
	delta := window.Sub(*reset)
	return delta <= resetTolerance && delta >= -resetTolerance
}

// Burn is the observed burn over one trailing horizon ending now.
type Burn struct {
	Horizon   time.Duration
	Available bool
	// Stale is true when no reading lies inside the horizon: the newest
	// reading is older than the horizon, so there is no current rate.
	Stale bool
	Rate  float64
	// Lo and Hi bound Rate by the ±1 point integer rounding of both readings.
	Lo, Hi float64
	Span   time.Duration
}

// Previous summarizes the window before the current one.
type Previous struct {
	ResetAt time.Time
	EndedAt time.Time
	// Final is the highest used percent observed in the window.
	Final float64
	// LastAt is the newest reading of the window.
	LastAt time.Time
	// Observed is true when Final is the window's outcome: the window
	// exhausted or was read within StaleAfter of its end. Otherwise readings
	// are sparse and Final is only the last reading, a lower bound.
	Observed bool
}

// Calibration is how the scope's completed windows actually burned after
// the same point of their life, relative to the profile projection from
// their start: Factor is the median ratio of realized to projected remaining burn
// over the newest CalibrationWindows windows, Low and High the quartiles.
// With fewer than minCalibration windows it is 1 (the uncalibrated profile).
type Calibration struct {
	Factor, Low, High float64
	Windows           int
}

// Forecast is one window's projection.
type Forecast struct {
	Window *Window
	Status string
	Reason string
	Used   float64
	// AsOf is the newest reading; Anchor is when projections start (now, or
	// AsOf in a backtest).
	AsOf, Anchor time.Time
	Burn         []Burn
	// Central is the mean projected burn from Anchor to the window end: the
	// burn since the window start, shaped by the hour-of-week profile and
	// scaled by Calibration.Factor.
	Central     float64
	Calibration Calibration
	profile     Profile
	// intensity is used percent per profile-weighted hour since the start.
	intensity float64
	Projected *time.Time
	BandStart *time.Time
	BandEnd   *time.Time
	// Lasting is the burn that lasts exactly until the reset.
	Lasting    *float64
	EarlyReset bool
	Previous   *Previous
}

// Project forecasts one window as of now. previous holds the earlier windows
// of the same scope; the completed ones calibrate the projection.
func Project(w *Window, previous []*Window, now time.Time) Forecast {
	return project(w, previous, now, w.scheduledEnd()).withStaleness(now)
}

// project forecasts w at now, assuming it ends at end. The backtest calls it
// with the window's actual end, the same horizon its linear baseline gets.
func project(w *Window, previous []*Window, now, end time.Time) Forecast {
	f := Forecast{Window: w, AsOf: w.LastAt, Anchor: w.LastAt, Used: w.Max(), Status: StatusUnknown}
	if now.After(f.Anchor) {
		f.Anchor = now
	}
	if !w.HasPercent {
		f.Status, f.Reason = standingStatus(w.Standing), "the provider reports a standing only, no used percentage"
		return f
	}
	f.Burn = burns(w, f.Anchor)
	if !w.ResetAt.IsZero() && now.Before(w.ResetAt) {
		lasting := (100 - f.Used) / w.ResetAt.Sub(now).Hours()
		f.Lasting = &lasting
	}
	if f.Used >= 100 {
		at := w.points[len(w.points)-1].At
		f.Projected, f.BandStart, f.BandEnd = &at, &at, &at
		f.Status, f.Reason = StatusWillExhaust, "the provider reports the window exhausted"
		return f
	}
	origin := w.origin()
	if f.AsOf.Sub(origin) < minBurnSpan {
		f.Reason = "fewer than 10 minutes of readings in this window"
		return f
	}
	// Projections follow the operator's hour-of-week burn profile, learned
	// from the scope's readings before this window began (the window's own
	// readings would teach it that only the hours already burned are
	// active), scaled to the burn since the window began, and calibrated on
	// completed windows.
	f.profile = burnProfile(previous, w.Start)
	delta := w.Max() - w.valueAt(origin)
	f.intensity = delta / f.profile.Mass(origin, f.AsOf)
	fraction := 0.0
	if !end.IsZero() && end.After(w.Start) {
		fraction = float64(f.AsOf.Sub(w.Start)) / float64(end.Sub(w.Start))
	}
	f.Calibration = calibrate(previous, w, fraction, f.profile)
	c := f.Calibration
	if end.After(f.Anchor) {
		f.Central = f.burnBetween(f.Anchor, end) / end.Sub(f.Anchor).Hours()
	} else {
		f.Central = c.Factor * f.intensity
	}
	// The band scales the projection by the union of the calibration
	// quartiles, the ±1 point rounding of the burn since the start, and the
	// spread of the current horizons around what the profile expected over
	// them. A horizon widens it only where it differs beyond its own
	// rounding, and then by its nearer edge: one integer step in the last
	// hour is noise, not a 1%/h burn.
	lo, hi := c.Low, c.High
	if delta > 0 {
		lo, hi = math.Min(lo, c.Factor*(delta-1)/delta), math.Max(hi, c.Factor*(delta+1)/delta)
	}
	for _, b := range f.Burn {
		expected := f.intensity * f.profile.Mass(f.Anchor.Add(-b.Span), f.Anchor) / b.Span.Hours()
		switch {
		case !b.Available || expected <= 0:
		case b.Lo > c.Factor*expected:
			hi = math.Max(hi, b.Lo/expected)
		case b.Hi < c.Factor*expected:
			lo = math.Min(lo, b.Hi/expected)
		}
	}
	remaining := 100 - f.Used
	f.Projected = f.runOut(remaining, c.Factor)
	f.BandStart = f.runOut(math.Max(remaining-1, 0), hi)
	f.BandEnd = f.runOut(remaining+1, math.Max(0, lo))
	switch {
	case w.ResetAt.IsZero():
		f.Status, f.Reason = StatusUnknown, "the provider did not report a reset time"
	case f.BandEnd != nil && f.BandEnd.Before(w.ResetAt):
		f.Status, f.Reason = StatusWillExhaust, "the whole projection band ends before the reset"
	case f.BandStart != nil && f.BandStart.Before(w.ResetAt):
		f.Status, f.Reason = StatusAtRisk, "the projection band straddles the reset"
	default:
		f.Status, f.Reason = StatusOnTrack, "projected exhaustion is after the reset"
	}
	if c.Windows < minCalibration {
		f.Reason += fmt.Sprintf("; uncalibrated: %d completed window(s) of history, so this is the burn since the window began, shaped by the hour-of-week profile", c.Windows)
	}
	return f
}

// burnBetween is the projected used-percent increase from from to to.
func (f Forecast) burnBetween(from, to time.Time) float64 {
	return f.Calibration.Factor * f.intensity * f.profile.Mass(from, to)
}

// runOut is when remaining points are used at factor times the profile
// intensity, starting at the anchor; nil when that never happens.
func (f Forecast) runOut(remaining, factor float64) *time.Time {
	if factor <= 0 || f.intensity <= 0 {
		return nil
	}
	return f.profile.until(f.Anchor, remaining/(factor*f.intensity))
}

func (f Forecast) withStaleness(now time.Time) Forecast {
	w := f.Window
	switch {
	case w.LastAt.IsZero():
		f.Status, f.Reason = StatusUnknown, "no readings"
	case !w.ResetAt.IsZero() && !now.Before(w.ResetAt):
		f.Status, f.Reason, f.Lasting = StatusUnknown, "the window reset at "+w.ResetAt.UTC().Format(time.RFC3339)+"; no reading of the new window yet", nil
	case now.Sub(w.LastAt) > StaleAfter:
		f.Status, f.Reason = StatusUnknown, "the newest reading is older than 6 hours"
	}
	return f
}

func standingStatus(standing pricing.QuotaStanding) string {
	switch standing {
	case pricing.QuotaStandingExhausted:
		return StatusWillExhaust
	case pricing.QuotaStandingLimited:
		return StatusAtRisk
	case pricing.QuotaStandingAvailable:
		return StatusOnTrack
	}
	return StatusUnknown
}

// burns measures each trailing horizon ending at anchor. A horizon holding no
// reading is stale; one starting inside the window needs a real reading at or
// before its start (the percent then is unknown, not 0); one reaching back
// past the window start uses its implicit 0%.
func burns(w *Window, anchor time.Time) []Burn {
	origin := w.origin()
	out := make([]Burn, 0, len(Horizons))
	for _, horizon := range Horizons {
		from := anchor.Add(-horizon)
		if from.Before(origin) {
			from = origin
		}
		b := Burn{Horizon: horizon, Span: anchor.Sub(from)}
		switch {
		case w.LastAt.Before(anchor.Add(-horizon)):
			b.Stale = true
		case b.Span >= minBurnSpan && (from.Equal(origin) || w.readingAtOrBefore(from)):
			b = rateOver(w, from, anchor, horizon)
		}
		out = append(out, b)
	}
	return out
}

func (w *Window) readingAtOrBefore(t time.Time) bool {
	return len(w.points) > 0 && !w.points[0].At.After(t)
}

func rateOver(w *Window, from, to time.Time, horizon time.Duration) Burn {
	span := to.Sub(from)
	hours, delta := span.Hours(), w.valueAt(to)-w.valueAt(from)
	return Burn{Horizon: horizon, Span: span, Available: true, Rate: delta / hours, Lo: math.Max(0, (delta-1)/hours), Hi: (delta + 1) / hours}
}

// calibrate compares, for each of the scope's newest completed windows, the
// burn it actually had after the same fraction of its life with the profile
// projection from its start. Only windows that ended by current's newest
// reading count, and each is seen as a forecaster would have seen it then,
// so a backtest never looks ahead.
func calibrate(previous []*Window, current *Window, fraction float64, profile Profile) Calibration {
	var ratios []float64
	for i := len(previous) - 1; i >= 0 && len(ratios) < CalibrationWindows && fraction > 0; i-- {
		p, next := previous[i], current
		if i+1 < len(previous) {
			next = previous[i+1]
		}
		end, done := p.end(next, current.LastAt)
		if !done || end.After(current.LastAt) || !p.HasPercent || (p.Max() < 100 && end.Sub(p.LastAt) > StaleAfter) {
			continue
		}
		if ratio, ok := realizedRatio(p, end, fraction, profile); ok {
			ratios = append(ratios, ratio)
		}
	}
	if len(ratios) < minCalibration {
		return Calibration{Factor: 1, Low: 1, High: 1, Windows: len(ratios)}
	}
	sort.Float64s(ratios)
	return Calibration{Factor: quantile(ratios, 0.5), Low: quantile(ratios, 0.25), High: quantile(ratios, 0.75), Windows: len(ratios)}
}

// realizedRatio is a completed window's realized burn after fraction of its
// life divided by the profile projection made then, both capped at 100%.
func realizedRatio(p *Window, end time.Time, fraction float64, profile Profile) (float64, bool) {
	at := p.Start.Add(time.Duration(fraction * float64(end.Sub(p.Start))))
	seen, ok := p.truncate(at)
	origin := seen.origin()
	if !ok || at.Sub(seen.LastAt) > StaleAfter || seen.LastAt.Sub(origin) < minBurnSpan {
		return 0, false
	}
	used := seen.Max()
	intensity := (used - seen.valueAt(origin)) / profile.Mass(origin, seen.LastAt)
	projected := math.Min(100-used, intensity*profile.Mass(seen.LastAt, end))
	if projected <= 0 {
		return 0, false
	}
	return (p.Max() - used) / projected, true
}

// quantile interpolates the q-quantile of sorted values.
func quantile(sorted []float64, q float64) float64 {
	pos := q * float64(len(sorted)-1)
	lo := int(pos)
	if lo+1 >= len(sorted) {
		return sorted[lo]
	}
	return sorted[lo] + (pos-float64(lo))*(sorted[lo+1]-sorted[lo])
}

func after(from time.Time, remaining, rate float64) *time.Time {
	if rate <= 0 {
		return nil
	}
	at := from.Add(time.Duration(remaining / rate * float64(time.Hour)))
	return &at
}

// Current forecasts the newest window of every scope (the one with the latest
// reset time), with the previous window's outcome for the early-reset note.
// When even the newest window's reset has passed, it is reported as unknown
// ("no reading of the new window yet"); a scope whose newest window ended more
// than one nominal window length ago is no longer read at all and is left out
// (F-ORCH-7). Results are sorted by scope.
func Current(spans []pricing.QuotaReadingSpan, now time.Time) []Forecast {
	byScope := map[string][]*Window{}
	var scopes []string
	for _, w := range Windows(spans) {
		if _, seen := byScope[w.Scope()]; !seen {
			scopes = append(scopes, w.Scope())
		}
		byScope[w.Scope()] = append(byScope[w.Scope()], w)
	}
	sort.Strings(scopes)
	out := make([]Forecast, 0, len(scopes))
	for _, scope := range scopes {
		windows := byScope[scope]
		current := windows[len(windows)-1]
		if current.dead(now) {
			continue
		}
		f := Project(current, windows[:len(windows)-1], now)
		for i := len(windows) - 2; i >= 0; i-- {
			if prev := windows[i]; prev.Max() > 0 || !prev.HasPercent {
				ended := current.Start
				if !prev.ResetAt.IsZero() && prev.ResetAt.Before(ended) {
					ended = prev.ResetAt
				}
				f.Previous = &Previous{ResetAt: prev.ResetAt, EndedAt: ended, Final: prev.Max(), LastAt: prev.LastAt, Observed: prev.Max() >= 100 || ended.Sub(prev.LastAt) <= StaleAfter}
				f.EarlyReset = !prev.ResetAt.IsZero() && current.Start.Before(prev.ResetAt.Add(-resetTolerance))
				break
			}
		}
		out = append(out, f)
	}
	return out
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return quantile(sorted, 0.5)
}
