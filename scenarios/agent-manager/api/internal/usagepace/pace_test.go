package usagepace

import (
	"testing"
	"time"

	"agent-manager/internal/usageforecast"
)

func TestForecastScalesRealProviderRunoutIntoEachPaceTier(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	projected := now.Add(4 * 24 * time.Hour)
	reset := now.Add(7 * 24 * time.Hour)
	forecasts := []usageforecast.Forecast{{
		Window:    &usageforecast.Window{Provider: "codex", Pool: "weekly", Window: "weekly", ResetAt: reset, Start: now.Add(-time.Hour)},
		Status:    usageforecast.StatusWillExhaust,
		AsOf:      now.Add(-time.Minute),
		Projected: &projected,
	}}

	got := Forecast(forecasts, 10, now)
	if len(got) != 4 {
		t.Fatalf("got %d tiers, want 4", len(got))
	}
	wantIDs := []string{TierReset, TierUseByThursday, TierUseByWednesday, TierExtraAllowance}
	wantSlots := []int{6, 11, 16}
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
		if tier.Status != "available" || tier.AvailableSlots == nil || *tier.AvailableSlots != wantSlots[i] {
			t.Errorf("tier %s = %+v, want %d available slots", tier.ID, tier, wantSlots[i])
		}
		if tier.Basis == "" || len(tier.ScopeSlots) != 1 {
			t.Errorf("tier %s missing model basis or real scope slots: %+v", tier.ID, tier)
		}
	}
}

func TestForecastLeavesSlotsUnavailableWithoutProjectedRunout(t *testing.T) {
	now := time.Now().UTC()
	got := Forecast([]usageforecast.Forecast{{
		Window: &usageforecast.Window{Provider: "codex", Pool: "weekly", Window: "weekly", ResetAt: now.Add(7 * 24 * time.Hour)},
		Status: usageforecast.StatusUnknown,
	}}, 10, now)
	for _, tier := range got {
		if tier.Status != "unavailable" || tier.AvailableSlots != nil {
			t.Errorf("tier %s fabricated slot count: %+v", tier.ID, tier)
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
