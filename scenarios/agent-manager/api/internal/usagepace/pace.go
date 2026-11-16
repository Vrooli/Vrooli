// Package usagepace turns provider quota forecasts into admission slot
// forecasts for explicit operator run-out targets.
package usagepace

import (
	"math"
	"sort"
	"time"

	"agent-manager/internal/usageforecast"
)

const (
	TierReset          = "at_reset"
	TierUseByThursday  = "use_by_thursday"
	TierUseByWednesday = "use_by_wednesday"
	TierExtraAllowance = "extra_allowance"
	MaxSlots           = 9999
)

// Tier is one precomputed operator pace choice. Slots is nil when the current
// provider readings cannot support an honest slot estimate.
type Tier struct {
	ID                     string      `json:"id"`
	Label                  string      `json:"label"`
	TargetRunOutAt         *time.Time  `json:"target_run_out_at,omitempty"`
	TargetRunOutSeconds    int64       `json:"target_run_out_seconds,omitempty"`
	AvailableSlots         *int        `json:"available_slots,omitempty"`
	EffectiveMaxConcurrent *int        `json:"effective_max_concurrent_runs,omitempty"`
	Status                 string      `json:"status"`
	Basis                  string      `json:"basis,omitempty"`
	Reason                 string      `json:"reason,omitempty"`
	SourceScopes           []string    `json:"source_scopes,omitempty"`
	SourceObservedAt       *time.Time  `json:"source_observed_at,omitempty"`
	ScopeSlots             []ScopeSlot `json:"scope_slots,omitempty"`
}

// ScopeSlot identifies the quota window that contributes a tier's global
// admission ceiling.
type ScopeSlot struct {
	Scope           string    `json:"scope"`
	AvailableSlots  int       `json:"available_slots"`
	ProjectedRunOut time.Time `json:"projected_run_out_at"`
	ResetAt         time.Time `json:"reset_at"`
}

// Forecast returns real slot projections for the supported pace tiers. It
// uses the current quota forecast's projected run-out under the existing
// admission ceiling as the observed-rate baseline. If any observed primary
// scope is stale, unknown, or missing a projected run-out, this returns an
// unavailable result rather than inventing a capacity number.
func Forecast(forecasts []usageforecast.Forecast, baselineSlots int, now time.Time) []Tier {
	now = now.UTC()
	valid := make([]usageforecast.Forecast, 0, len(forecasts))
	for _, f := range forecasts {
		if f.Window == nil || f.Window.ResetAt.IsZero() || f.Window.ResetAt.Before(now) || f.Dormant(now) {
			continue
		}
		valid = append(valid, f)
	}
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].Window.Scope() < valid[j].Window.Scope()
	})

	types := []struct {
		id, label string
	}{
		{TierReset, "Last until quota resets"},
		{TierUseByThursday, "Use by Thursday"},
		{TierUseByWednesday, "Use by Wednesday"},
		{TierExtraAllowance, "Use extra allowance"},
	}
	result := make([]Tier, 0, len(types))
	for _, tier := range types {
		row := Tier{ID: tier.id, Label: tier.label, Status: "unavailable"}
		if tier.id == TierExtraAllowance {
			row.Reason = "provider quota readings do not identify a separate extra-allowance budget"
			result = append(result, row)
			continue
		}
		if baselineSlots < 1 || len(valid) == 0 {
			row.Reason = "no fresh provider quota forecast is available to estimate slots"
			result = append(result, row)
			continue
		}
		minSlots := MaxSlots
		complete := true
		var targetAt time.Time
		for _, f := range valid {
			if f.Projected == nil || !f.Projected.After(now) || f.Status == usageforecast.StatusUnknown || f.Status == usageforecast.StatusIdle {
				complete = false
				break
			}
			target := f.Window.ResetAt
			if tierTarget := targetForDay(tier.id, now, usageforecast.DigestLocation()); !tierTarget.IsZero() && tierTarget.Before(target) {
				target = tierTarget
			}
			targetSeconds := int64(target.Sub(now).Seconds())
			if targetSeconds < 1 {
				complete = false
				break
			}
			currentSeconds := f.Projected.Sub(now).Seconds()
			// The same quota budget is expected to be consumed over the target
			// interval. Scaling current admitted slots by observed/target burn
			// preserves the measured provider rate as the only source of capacity.
			slots := int(math.Round(float64(baselineSlots) * currentSeconds / float64(targetSeconds)))
			if slots < 1 {
				slots = 1
			}
			if slots > MaxSlots {
				slots = MaxSlots
			}
			if slots < minSlots {
				minSlots = slots
			}
			row.SourceScopes = append(row.SourceScopes, f.Window.Scope())
			row.ScopeSlots = append(row.ScopeSlots, ScopeSlot{
				Scope:           f.Window.Scope(),
				AvailableSlots:  slots,
				ProjectedRunOut: *f.Projected,
				ResetAt:         f.Window.ResetAt,
			})
			if row.SourceObservedAt == nil || f.AsOf.After(*row.SourceObservedAt) {
				observed := f.AsOf
				row.SourceObservedAt = &observed
			}
			if targetAt.IsZero() || target.Before(targetAt) {
				targetAt = target
			}
		}
		if !complete {
			row.Reason = "one or more current provider quota windows lack a fresh projected run-out"
			result = append(result, row)
			continue
		}
		row.Status = "available"
		row.Basis = "relative estimate from provider-reported quota forecasts; assumes quota burn scales linearly with admitted concurrent runs"
		row.Basis = "relative estimate from the provider-reported quota forecast; assumes quota burn scales linearly with admitted concurrent runs"
		row.TargetRunOutAt = &targetAt
		row.TargetRunOutSeconds = int64(targetAt.Sub(now).Seconds())
		row.AvailableSlots = &minSlots
		row.EffectiveMaxConcurrent = &minSlots
		result = append(result, row)
	}
	return result
}

// targetForDay returns the end of the next matching operator-local calendar
// day. Its returned instant is absolute; JSON encoding keeps it in UTC.
func targetForDay(tier string, now time.Time, location *time.Location) time.Time {
	var day time.Weekday
	switch tier {
	case TierUseByThursday:
		day = time.Thursday
	case TierUseByWednesday:
		day = time.Wednesday
	default:
		return time.Time{}
	}
	if location == nil {
		location = time.UTC
	}
	localNow := now.In(location)
	date := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 23, 59, 59, 0, location)
	for date.Weekday() != day || !date.After(localNow) {
		date = date.AddDate(0, 0, 1)
	}
	return date
}
