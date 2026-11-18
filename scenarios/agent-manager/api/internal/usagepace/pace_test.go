package usagepace

import (
	"strings"
	"testing"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"
)

func TestForecastKeepsTiersUnavailableWithoutHistoricalUtilization(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	projected := now.Add(4 * 24 * time.Hour)
	reset := now.Add(7 * 24 * time.Hour)
	forecasts := []usageforecast.Forecast{{
		Window:    &usageforecast.Window{Provider: "codex", Pool: "weekly", Window: "weekly", WindowMinutes: 10080, ResetAt: reset, Start: now.Add(-time.Hour), StartKnown: true, HasPercent: true},
		Status:    usageforecast.StatusWillExhaust,
		AsOf:      now.Add(-time.Minute),
		Projected: &projected,
	}}

	got := Forecast(forecasts, testUtilization(10, now), now)
	if len(got) != 4 {
		t.Fatalf("got %d tiers, want 4", len(got))
	}
	wantIDs := []string{TierReset, TierUseByThursday, TierUseByWednesday, TierExtraAllowance}
	for i, tier := range got {
		if tier.ID != wantIDs[i] {
			t.Errorf("tier[%d].ID = %q, want %q", i, tier.ID, wantIDs[i])
		}
		if i == len(got)-1 {
			if tier.Status != "unavailable" || tier.AvailableSlots != nil {
				t.Errorf("extra allowance tier = %+v, want unavailable without a source", tier)
			}
			continue
		}
		if tier.Status != "unavailable" || tier.AvailableSlots != nil || tier.EffectiveMaxConcurrent != nil {
			t.Errorf("tier %s claimed a slot ceiling without historical utilization evidence: %+v", tier.ID, tier)
		}
		if !strings.Contains(tier.Reason, "point-in-time snapshot") {
			t.Errorf("tier %s did not explain the utilization evidence gap: %+v", tier.ID, tier)
		}
		wantTarget := reset
		if tier.ID == TierUseByThursday || tier.ID == TierUseByWednesday {
			wantTarget = targetForDay(tier.ID, now, time.UTC)
		}
		if tier.TargetRunOutAt == nil || !tier.TargetRunOutAt.Equal(wantTarget) {
			t.Errorf("tier %s target = %v, want %v", tier.ID, tier.TargetRunOutAt, wantTarget)
		}
	}
}

func TestForecastSeparatesShortWindowResetFromWeekdayRunOutTargets(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	shortReset := now.Add(time.Hour)
	weekReset := now.Add(7 * 24 * time.Hour)
	shortRunOut := now.Add(60 * time.Hour) // The measured forecast lasts beyond this short reset.
	weeklyRunOut := now.Add(4 * 24 * time.Hour)
	forecasts := []usageforecast.Forecast{
		{
			Window: &usageforecast.Window{Provider: "anthropic", Pool: "five_hour", Window: "300m", WindowMinutes: 300, ResetAt: shortReset, Start: now.Add(-4 * time.Hour), StartKnown: true, HasPercent: true},
			Status: usageforecast.StatusOnTrack, Used: 10, AsOf: now, Projected: &shortRunOut,
		},
		{
			Window: &usageforecast.Window{Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: 10080, ResetAt: weekReset, Start: now, StartKnown: true, HasPercent: true},
			Status: usageforecast.StatusWillExhaust, Used: 20, AsOf: now, Projected: &weeklyRunOut,
		},
	}

	got := Forecast(forecasts, testUtilization(10, now), now)
	if len(got) != 4 {
		t.Fatalf("got %d tiers, want 4", len(got))
	}
	if got[0].Status != "unavailable" || got[0].TargetRunOutAt == nil || !got[0].TargetRunOutAt.Equal(weekReset) {
		t.Fatalf("at-reset tier = %+v", got[0])
	}
	if got[1].Status != "unavailable" || got[2].Status != "unavailable" {
		t.Fatalf("weekday tiers claimed unsupported slots: Thursday=%+v Wednesday=%+v", got[1], got[2])
	}
	if got[1].TargetRunOutAt == nil || !got[1].TargetRunOutAt.Equal(targetForDay(TierUseByThursday, now, time.UTC)) || got[2].TargetRunOutAt == nil || !got[2].TargetRunOutAt.Equal(targetForDay(TierUseByWednesday, now, time.UTC)) {
		t.Fatalf("weekday targets were capped to the short reset: Thursday=%+v Wednesday=%+v", got[1], got[2])
	}
	for _, tier := range got[:3] {
		if len(tier.SourceScopes) != 2 {
			t.Errorf("tier %s quota scopes = %v, want both short and weekly constraints", tier.ID, tier.SourceScopes)
		}
	}
}

func TestForecastLeavesGlobalCeilingUnavailableWhenAQuotaWindowIsExhausted(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	reset := now.Add(24 * time.Hour)
	projected := now.Add(-12 * time.Hour)
	got := Forecast([]usageforecast.Forecast{{
		Window: &usageforecast.Window{Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: 10080, ResetAt: reset, Start: now.Add(-5 * 24 * time.Hour), StartKnown: true, HasPercent: true},
		Status: usageforecast.StatusWillExhaust, Used: 100, AsOf: now, Projected: &projected,
	}}, testUtilization(10, now), now)
	for _, tier := range got[:3] {
		if tier.Status != "unavailable" || tier.AvailableSlots != nil || !strings.Contains(tier.Reason, "exhausted") {
			t.Errorf("tier %s claimed a positive global ceiling from an exhausted window: %+v", tier.ID, tier)
		}
	}
}

func TestForecastIdentifiesTheQuotaWindowThatBlocksTheGlobalCeiling(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	fiveHourReset := now.Add(time.Hour)
	weekReset := now.Add(48 * time.Hour)
	fiveHourRunOut := now.Add(24 * time.Hour)
	weekRunOut := now.Add(-time.Hour)
	got := Forecast([]usageforecast.Forecast{
		{
			Window: &usageforecast.Window{Provider: "anthropic", Pool: "five_hour", Window: "300m", WindowMinutes: 300, ResetAt: fiveHourReset, Start: now.Add(-4 * time.Hour), StartKnown: true, HasPercent: true},
			Status: usageforecast.StatusOnTrack, Used: 18, AsOf: now.Add(-time.Minute), Projected: &fiveHourRunOut,
		},
		{
			Window: &usageforecast.Window{Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: 10080, ResetAt: weekReset, Start: now.Add(-5 * 24 * time.Hour), StartKnown: true, HasPercent: true},
			Status: usageforecast.StatusWillExhaust, Used: 100, AsOf: now.Add(-time.Minute), Projected: &weekRunOut,
		},
	}, testUtilization(10, now), now)

	for _, tier := range got[:3] {
		if tier.Status != "unavailable" || !strings.Contains(tier.Reason, "exhausted") || !strings.HasPrefix(tier.Reason, "anthropic|seven_day|10080m:") {
			t.Fatalf("tier %s = %+v, want unavailable due to exhausted weekly window", tier.ID, tier)
		}
		if want := []string{"anthropic|five_hour|300m", "anthropic|seven_day|10080m"}; len(tier.SourceScopes) != len(want) || tier.SourceScopes[0] != want[0] || tier.SourceScopes[1] != want[1] {
			t.Errorf("tier %s source scopes = %v, want both evaluated quota windows %v", tier.ID, tier.SourceScopes, want)
		}
		if len(tier.ScopeSlots) != 0 {
			t.Errorf("tier %s published per-scope slots without historical utilization: %+v", tier.ID, tier.ScopeSlots)
		}
		if tier.SourceObservedAt == nil || !tier.SourceObservedAt.Equal(now.Add(-time.Minute)) {
			t.Errorf("tier %s source observed at = %v, want the latest evaluated quota timestamp", tier.ID, tier.SourceObservedAt)
		}
	}
}

func TestForecastUsesFreshWeeklyQuotaForLatestResetHorizonAndNamesBothScopes(t *testing.T) {
	const nowText = "2026-11-16T19:05:00Z"
	now, _ := time.Parse(time.RFC3339, nowText)
	fiveReset := now.Add(75 * time.Minute)
	weekReset, _ := time.Parse(time.RFC3339, "2026-11-18T15:00:00Z")
	fiveStart := fiveReset.Add(-5 * time.Hour)
	weekStart := weekReset.Add(-7 * 24 * time.Hour)
	percent := func(v float64) *float64 { return &v }
	minutes := func(v int64) *int64 { return &v }
	reset := func(v time.Time) *time.Time { return &v }
	readings := []pricing.QuotaReadingSpan{
		{Provider: "anthropic", Pool: "five_hour", Window: "300m", WindowMinutes: minutes(300), ResetAt: reset(fiveReset), UsedPercent: percent(1), FirstAt: fiveStart.Add(10 * time.Minute), LastAt: fiveStart.Add(10 * time.Minute), Samples: 1},
		{Provider: "anthropic", Pool: "five_hour", Window: "300m", WindowMinutes: minutes(300), ResetAt: reset(fiveReset), UsedPercent: percent(18), FirstAt: now.Add(-2 * time.Minute), LastAt: now.Add(-2 * time.Minute), Samples: 1},
		{Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: minutes(10080), ResetAt: reset(weekReset), UsedPercent: percent(1), FirstAt: weekStart.Add(10 * time.Minute), LastAt: weekStart.Add(10 * time.Minute), Samples: 1},
		{Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: minutes(10080), ResetAt: reset(weekReset), UsedPercent: percent(100), FirstAt: time.Date(2026, 11, 14, 15, 8, 42, 0, time.UTC), LastAt: time.Date(2026, 11, 14, 15, 8, 42, 0, time.UTC), Samples: 1},
		{Provider: "anthropic", Pool: "seven_day", Window: "10080m", WindowMinutes: minutes(10080), ResetAt: reset(weekReset), UsedPercent: percent(97), FirstAt: now.Add(-2 * time.Minute), LastAt: now.Add(-2 * time.Minute), Samples: 1},
	}
	got := Forecast(usageforecast.Current(readings, now), testUtilization(10, now), now)
	if got[0].Status != "unavailable" || got[0].AvailableSlots != nil || !strings.Contains(got[0].Reason, "point-in-time snapshot") {
		t.Fatalf("latest-reset tier = %+v, want unavailable without coextensive utilization history", got[0])
	}
	if len(got[0].SourceScopes) != 2 || got[0].SourceScopes[0] != "anthropic|five_hour|300m" || got[0].SourceScopes[1] != "anthropic|seven_day|10080m" {
		t.Fatalf("near-reset scopes = %v, want both live windows", got[0].SourceScopes)
	}
	if got[0].TargetRunOutAt == nil || !got[0].TargetRunOutAt.Equal(weekReset) {
		t.Fatalf("at-reset target = %v, want latest active reset %s", got[0].TargetRunOutAt, weekReset)
	}
	if len(got[0].ScopeSlots) != 0 {
		t.Fatalf("latest-reset tier published unsupported scope slots: %+v", got[0].ScopeSlots)
	}
}

func TestForecastLeavesSlotsUnavailableWithoutProjectedRunout(t *testing.T) {
	now := time.Now().UTC()
	got := Forecast([]usageforecast.Forecast{{
		Window: &usageforecast.Window{Provider: "codex", Pool: "weekly", Window: "weekly", ResetAt: now.Add(7 * 24 * time.Hour)},
		Status: usageforecast.StatusUnknown,
	}}, testUtilization(10, now), now)
	for _, tier := range got {
		if tier.Status != "unavailable" || tier.AvailableSlots != nil {
			t.Errorf("tier %s fabricated slot count: %+v", tier.ID, tier)
		}
	}
}

func TestForecastDoesNotTreatCurrentOccupancyAsHistoricalUtilization(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	projected := now.Add(4 * 24 * time.Hour)
	reset := now.Add(7 * 24 * time.Hour)
	forecast := []usageforecast.Forecast{{
		Window:    &usageforecast.Window{Provider: "codex", Pool: "weekly", Window: "weekly", WindowMinutes: 10080, ResetAt: reset, Start: now.Add(-time.Hour), StartKnown: true, HasPercent: true},
		Status:    usageforecast.StatusWillExhaust,
		Used:      50,
		AsOf:      now.Add(-time.Minute),
		Projected: &projected,
	}}
	for label, activity := range map[string]Utilization{
		"full current occupancy":    testUtilization(10, now),
		"low current occupancy":     testUtilization(2, now),
		"unknown current occupancy": {ObservedAt: now, Complete: false},
		"zero current occupancy":    testUtilization(0, now),
	} {
		got := Forecast(forecast, activity, now)[0]
		if got.Status != "unavailable" || got.AvailableSlots != nil || !strings.Contains(got.Reason, "point-in-time snapshot") {
			t.Errorf("%s activity was incorrectly treated as historical utilization: %+v", label, got)
		}
	}
}

func TestWeekdayTierUsesConfiguredOperatorLocation(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "America/Los_Angeles")
	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	target := targetForDay(TierUseByWednesday, now, usageforecast.DigestLocation())
	local := target.In(usageforecast.DigestLocation())
	if local.Weekday() != time.Wednesday || local.Hour() != 23 {
		t.Fatalf("operator-local Wednesday target = %s, want Wednesday at 23:59 local", local)
	}
	if target.Location().String() != usageforecast.DigestLocation().String() {
		t.Fatalf("target location = %s, want configured operator location", target.Location())
	}
}

func testUtilization(runs uint32, at time.Time) Utilization {
	return Utilization{ExecutingRuns: runs, ObservedAt: at, Complete: true}
}
