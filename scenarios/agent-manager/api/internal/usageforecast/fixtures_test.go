package usageforecast

import (
	"fmt"
	"math"
	"time"

	"agent-manager/internal/pricing"
)

// step is one provider reading change: from at, the provider reports percent.
type step struct {
	at      time.Time
	percent float64
}

func at(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}

// ramp emits the integer steps a provider reports while used percent climbs
// linearly from p0 at t0 to p1 at t1.
func ramp(t0, t1 time.Time, p0, p1 int) []step {
	if p1 <= p0 {
		return []step{{t0, float64(p0)}}
	}
	out := make([]step, 0, p1-p0+1)
	for p := p0; p <= p1; p++ {
		frac := float64(p-p0) / float64(p1-p0)
		out = append(out, step{t0.Add(time.Duration(frac * float64(t1.Sub(t0)))), float64(p)})
	}
	return out
}

// window describes one provider window's readings for the generator.
type windowSpec struct {
	provider, pool string
	minutes        int64
	reset          time.Time
	steps          []step
	// last is the newest reading; the final step is held until then.
	last time.Time
}

func codexWindow(reset time.Time, last time.Time, steps ...[]step) windowSpec {
	var all []step
	for _, s := range steps {
		all = append(all, s...)
	}
	return windowSpec{provider: "openai", pool: "primary", minutes: 10080, reset: reset, steps: all, last: last}
}

// spans renders windows the way QuotaReadingSpans returns them: one span per
// distinct reading, the reset truncated to the minute, oldest first.
func spans(windows ...windowSpec) []pricing.QuotaReadingSpan {
	var out []pricing.QuotaReadingSpan
	for _, w := range windows {
		reset := w.reset.Truncate(time.Minute)
		minutes := w.minutes
		for i, s := range w.steps {
			last := w.last
			if i+1 < len(w.steps) {
				last = w.steps[i+1].at.Add(-time.Second)
			}
			percent := s.percent
			standing := pricing.QuotaStandingAvailable
			if percent >= 100 {
				standing = pricing.QuotaStandingExhausted
			}
			r := reset
			span := pricing.QuotaReadingSpan{Provider: w.provider, Pool: w.pool, Window: fmt.Sprintf("%dm", minutes), WindowMinutes: &minutes, UsedPercent: &percent, Standing: standing, FirstAt: s.at, LastAt: last, Samples: 1}
			if !w.reset.IsZero() {
				span.ResetAt = &r
			}
			out = append(out, span)
		}
	}
	return out
}

// claudeSpan is one Claude Code statusline sample as QuotaReadingSpans
// returns it: the reset truncated to the minute.
func claudeSpan(pool string, observed time.Time, percent float64, resetsAt int64) pricing.QuotaReadingSpan {
	minutes := pricing.ClaudeWindowMinutes(pool)
	reset := time.Unix(resetsAt, 0).UTC().Truncate(time.Minute)
	return pricing.QuotaReadingSpan{Provider: "anthropic", Pool: pool, Window: fmt.Sprintf("%dm", minutes), WindowMinutes: &minutes, ResetAt: &reset, UsedPercent: &percent, Standing: pricing.QuotaStandingAvailable, FirstAt: observed, LastAt: observed, Samples: 1}
}

func near(got, want, tolerance float64) bool { return math.Abs(got-want) <= tolerance }

// oct28Shape is the Codex weekly window measured on 2026-10-29 (EVIDENCE
// E-1): opened 10-28 06:57:18Z, resets_at 1793775438 (2026-11-04T06:57:18Z),
// 0→33% over ~21.5h with ~3.4%/h over the last 6h.
func oct28Shape() windowSpec {
	start, reset := at("2026-10-28T06:57:18Z"), time.Unix(1793775438, 0).UTC()
	return codexWindow(reset, at("2026-10-29T04:30:00Z"),
		ramp(start.Add(10*time.Minute), at("2026-10-28T22:30:00Z"), 1, 13),
		ramp(at("2026-10-28T22:48:00Z"), at("2026-10-29T04:30:00Z"), 14, 33))
}

// earlyResetHistory is the 10-22..10-28 sequence of early resets (EVIDENCE
// E-1): new 7-day windows began 10-22 17:09, 10-23 18:02, 10-25 06:10,
// 10-26 06:08 and 10-28 06:57; the previous windows ended at 15/8/13/27%.
func earlyResetHistory() []windowSpec {
	week := 7 * 24 * time.Hour
	type w struct {
		start string
		end   string
		final int
	}
	var out []windowSpec
	for _, item := range []w{{"2026-10-22T17:09:00Z", "2026-10-23T18:01:00Z", 15}, {"2026-10-23T18:02:00Z", "2026-10-25T06:09:00Z", 8}, {"2026-10-25T06:10:00Z", "2026-10-26T06:07:00Z", 13}, {"2026-10-26T06:08:00Z", "2026-10-28T06:56:00Z", 27}} {
		start, end := at(item.start), at(item.end)
		out = append(out, codexWindow(start.Add(week), end, ramp(start.Add(5*time.Minute), end.Add(-30*time.Minute), 1, item.final)))
	}
	return out
}

// shiftOperator is an operator who works two shifts a day on Codex five-hour
// windows: 12:00–17:00 UTC at a steady 8%/h (a window that ends at 40%), and
// 20:00–22:00 UTC at 10%/h before stopping for the night (a window that ends
// at 20%). A naive linear projection made early in the evening window
// expects the burn to continue to its 01:00 reset.
func shiftOperator(from time.Time, days int) []windowSpec {
	var out []windowSpec
	for d := 0; d < days; d++ {
		day := from.Add(time.Duration(d) * 24 * time.Hour)
		noon, evening := day.Add(12*time.Hour), day.Add(20*time.Hour)
		out = append(out,
			windowSpec{provider: "openai", pool: "primary", minutes: 300, reset: noon.Add(5 * time.Hour), steps: ramp(noon.Add(5*time.Minute), noon.Add(5*time.Hour-5*time.Minute), 1, 40), last: noon.Add(5*time.Hour - 5*time.Minute)},
			windowSpec{provider: "openai", pool: "primary", minutes: 300, reset: evening.Add(5 * time.Hour), steps: ramp(evening.Add(5*time.Minute), evening.Add(2*time.Hour), 1, 20), last: evening.Add(2 * time.Hour)})
	}
	return out
}
