// Package usagepace exposes provider quota targets and enables admission tiers
// only when utilization evidence supports a trustworthy ceiling.
package usagepace

import (
	"sort"
	"time"

	"agent-manager/internal/usageforecast"
)

const (
	TierReset          = "at_reset"
	TierUseByThursday  = "use_by_thursday"
	TierUseByWednesday = "use_by_wednesday"
	TierExtraAllowance = "extra_allowance"
)

// Tier is one precomputed operator pace choice. Slots is nil when the current
// quota and utilization evidence cannot support an honest slot estimate.
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
	ObservedExecutingRuns  *uint32     `json:"observed_executing_runs,omitempty"`
	UtilizationObservedAt  *time.Time  `json:"utilization_observed_at,omitempty"`
	UtilizationComplete    bool        `json:"utilization_complete"`
}

// ScopeSlot identifies the quota window that contributes a tier's global
// admission ceiling when sufficient utilization evidence is available.
type ScopeSlot struct {
	Scope           string    `json:"scope"`
	Provider        string    `json:"provider"`
	Pool            string    `json:"pool"`
	Window          string    `json:"window"`
	AccountScope    string    `json:"account_scope,omitempty"`
	AvailableSlots  int       `json:"available_slots"`
	ProjectedRunOut time.Time `json:"projected_run_out_at"`
	ResetAt         time.Time `json:"reset_at"`
	RunIDs          []string  `json:"run_ids"`
	QuotaStart      time.Time `json:"quota_window_start"`
	QuotaEnd        time.Time `json:"quota_window_end"`
	SampleStart     time.Time `json:"utilization_sample_start"`
	SampleEnd       time.Time `json:"utilization_sample_end"`
	SampleCount     int       `json:"utilization_sample_count"`
	RunSeconds      float64   `json:"measured_run_seconds"`
	BurnPercent     float64   `json:"measured_quota_burn_percent"`
	ScopeEvidence   string    `json:"scope_evidence"`
}

// Utilization is a point-in-time physical execution observation. It is useful
// provenance but cannot normalize quota burn accumulated over a historical
// window; that requires coextensive, per-provider utilization history.
type Utilization struct {
	ExecutingRuns uint32
	ObservedAt    time.Time
	Complete      bool
}

// Forecast returns supported pace targets and explicit availability. Provider
// quota forecasts are account-wide; the available utilization source is only a
// current physical-execution snapshot, not history matched to the quota burn
// interval. Therefore it cannot support a per-run slot ceiling, and tiers stay
// unavailable until such evidence exists. The at-reset target uses the latest
// active reset, so a short quota window does not replace a longer horizon.
func Forecast(forecasts []usageforecast.Forecast, utilization Utilization, now time.Time) []Tier {
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
		{TierReset, "Last through the final active quota reset"},
		{TierUseByThursday, "Use by Thursday"},
		{TierUseByWednesday, "Use by Wednesday"},
		{TierExtraAllowance, "Use extra allowance"},
	}
	result := make([]Tier, 0, len(types))
	for _, tier := range types {
		row := Tier{ID: tier.id, Label: tier.label, Status: "unavailable", UtilizationComplete: utilization.Complete}
		if utilization.ExecutingRuns > 0 {
			observed := utilization.ExecutingRuns
			row.ObservedExecutingRuns = &observed
		}
		if !utilization.ObservedAt.IsZero() {
			observed := utilization.ObservedAt.UTC()
			row.UtilizationObservedAt = &observed
		}
		if tier.id == TierExtraAllowance {
			row.Reason = "provider quota readings do not identify a separate extra-allowance budget"
			result = append(result, row)
			continue
		}
		targetAt := targetForDay(tier.id, now, usageforecast.DigestLocation())
		if targetAt.IsZero() {
			for _, f := range valid {
				if targetAt.IsZero() || f.Window.ResetAt.After(targetAt) {
					targetAt = f.Window.ResetAt
				}
			}
		}
		if !targetAt.IsZero() {
			row.TargetRunOutAt = &targetAt
			row.TargetRunOutSeconds = int64(targetAt.Sub(now).Seconds())
		}
		if len(valid) == 0 {
			row.Reason = "no fresh provider quota forecast is available to estimate slots"
			result = append(result, row)
			continue
		}
		quotaComplete := true
		for _, f := range valid {
			// Record each active quota window before evaluating it. If a later
			// window blocks the tier, the response must identify that actual
			// blocker as well as any earlier windows that passed. Otherwise a
			// weekly exhaustion can be reported as if the five-hour window were
			// the only source considered.
			row.SourceScopes = append(row.SourceScopes, f.Window.Scope())
			if row.SourceObservedAt == nil || f.AsOf.After(*row.SourceObservedAt) {
				observed := f.AsOf
				row.SourceObservedAt = &observed
			}
			if f.Projected == nil || f.Status == usageforecast.StatusUnknown || f.Status == usageforecast.StatusIdle {
				quotaComplete = false
				row.Reason = f.Window.Scope() + ": one or more quota windows lack a fresh, usable forecast"
				break
			}
			if f.Used >= 100 {
				quotaComplete = false
				row.Reason = f.Window.Scope() + ": quota window is exhausted before reset"
				break
			}
		}
		if !quotaComplete {
			result = append(result, row)
			continue
		}
		row.Reason = "quota burn is account-wide, while physical utilization is only a point-in-time snapshot; no coextensive per-provider utilization history supports a trustworthy slot ceiling"
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
