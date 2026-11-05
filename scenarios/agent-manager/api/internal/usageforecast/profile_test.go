package usageforecast

import (
	"testing"
	"time"
)

// Without history the profile is uniform, so the projection is the linear
// burn; with it, the hours the operator works carry the weight, and a short
// history is only partly trusted.
func TestBurnProfileWeightsTheOperatorsActiveHours(t *testing.T) {
	from := at("2026-09-01T00:00:00Z")
	day := from.Add(24 * time.Hour)
	if mass := burnProfile(nil, from).Mass(day, day.Add(7*time.Hour)); !near(mass, 7, 1e-9) {
		t.Fatalf("an empty history must be uniform: mass=%v over 7h", mass)
	}
	windows := Windows(spans(shiftOperator(from, 35)...))
	asOf := from.Add(35 * 24 * time.Hour)
	full := burnProfile(windows, asOf)
	working, night := full.Mass(asOf.Add(12*time.Hour), asOf.Add(17*time.Hour)), full.Mass(asOf.Add(2*time.Hour), asOf.Add(7*time.Hour))
	if working < 5*night || !near(full.Mass(asOf, asOf.Add(7*24*time.Hour)), hoursPerWeek, 1e-6) {
		t.Fatalf("working-hours mass %.2f must dominate night %.2f, weights averaging 1", working, night)
	}
	// Two days of history: mostly uniform still.
	short := burnProfile(windows[:4], from.Add(2*24*time.Hour))
	if w, n := short.Mass(asOf.Add(12*time.Hour), asOf.Add(17*time.Hour)), short.Mass(asOf.Add(2*time.Hour), asOf.Add(7*time.Hour)); w > 2*n {
		t.Fatalf("a two-day history is mostly uniform: working %.2f night %.2f", w, n)
	}
	if got := full.until(asOf, full.Mass(asOf, asOf.Add(30*time.Hour))); got == nil || !near(got.Sub(asOf).Hours(), 30, 1e-6) {
		t.Fatalf("until must invert Mass: %v", got)
	}
}
