package utilizationhistory

import (
	"fmt"
	"math"
	"sort"
	"time"

	"agent-manager/internal/pricing"
	"github.com/google/uuid"
)

const (
	TierLastReset       = "at_reset"
	TierUseByThursday   = "use_by_thursday"
	TierUseByWednesday  = "use_by_wednesday"
	TierExtraAllowance  = "extra_allowance"
	defaultMaxSampleGap = 3 * time.Minute
	minBurnHistory      = 10 * time.Minute
	maxAdmissionSlots   = 9999
)

// EstimateInput is the pure estimator's complete evidence boundary. Samples
// are global physical snapshots; quota observations are provider scoped.
// Callers must pass all observations for the intended provider scopes, not a
// pre-aggregated account-wide burn rate.
type EstimateInput struct {
	Now          time.Time
	Targets      []Target
	Quota        []pricing.QuotaObservation
	Samples      []Sample
	MaxSampleGap time.Duration
	MaxQuotaAge  time.Duration
}

// Target is an operator-local deadline resolved by the caller to an instant.
type Target struct {
	ID string
	At time.Time
}

// TierEstimate reports a defensible admission ceiling. Slots is nil whenever
// the evidence cannot establish a conservative per-run burn rate.
type TierEstimate struct {
	ID     string
	Target time.Time
	Slots  *int
	Status string
	Reason string
	Basis  string
	// ProviderSlots preserves each quota window's independent constraint.
	ProviderSlots []ProviderEstimate
}

type ProviderEstimate struct {
	Provider      string
	Pool          string
	Window        string
	AccountScope  string
	Slots         int
	QuotaStart    time.Time
	QuotaEnd      time.Time
	ScopeEvidence string
	RunIDs        []string
	SampleStart   time.Time
	SampleEnd     time.Time
	SampleCount   int
	RunSeconds    float64
	BurnPercent   float64
}

type runExposure struct {
	seconds     float64
	runIDs      []string
	sampleStart time.Time
	sampleEnd   time.Time
	sampleCount int
}

func quotaSourceRunIDs(points []pricing.QuotaObservation) (map[string]bool, bool) {
	ids := make(map[string]bool, len(points))
	for _, point := range points {
		parsed, err := uuid.Parse(point.SourceRunID)
		if err != nil || parsed == uuid.Nil {
			return nil, false
		}
		ids[parsed.String()] = true
	}
	return ids, len(ids) > 0
}

type quotaKey struct {
	provider, pool, window, account string
	reset                           time.Time
}

// Estimate computes integer ceilings from observed quota-percent burn divided
// by conservative physical run-seconds. A run contributes exposure only when
// its actual ID is present in both adjacent complete samples; IDs appearing
// at only one endpoint contribute no assumed duration. Unmapped physical runs,
// unknown snapshot coverage, missing account-to-run evidence, reset crossings,
// stale points, or gaps wider than MaxSampleGap make the affected estimate
// unavailable.
func Estimate(in EstimateInput) []TierEstimate {
	now := in.Now.UTC()
	gap := in.MaxSampleGap
	if gap <= 0 {
		gap = defaultMaxSampleGap
	}
	maxAge := in.MaxQuotaAge
	if maxAge <= 0 {
		maxAge = 10 * time.Minute
	}
	out := make([]TierEstimate, 0, len(in.Targets)+1)
	extraAllowanceIncluded := false
	for _, target := range in.Targets {
		if target.ID == TierExtraAllowance {
			extraAllowanceIncluded = true
		}
		row := TierEstimate{ID: target.ID, Target: target.At.UTC(), Status: "unavailable"}
		if target.ID == TierExtraAllowance {
			row.Reason = "no separate extra-allowance quota evidence is available"
			out = append(out, row)
			continue
		}
		if now.IsZero() || target.At.IsZero() || !target.At.After(now) {
			row.Reason = "target deadline must be after the observation time"
			out = append(out, row)
			continue
		}
		estimates, reason := estimateTarget(in, now, target.At.UTC(), gap, maxAge)
		if reason != "" {
			row.Reason = reason
			out = append(out, row)
			continue
		}
		ceiling := estimates[0].Slots
		for _, p := range estimates[1:] {
			if p.Slots < ceiling {
				ceiling = p.Slots
			}
		}
		if ceiling < 1 {
			row.Reason = "measured quota budget does not support one concurrent run through the target"
			out = append(out, row)
			continue
		}
		if ceiling > maxAdmissionSlots {
			ceiling = maxAdmissionSlots
		}
		row.Slots = &ceiling
		row.Status = "available"
		row.Basis = "provider-scoped quota percent burn divided by stable run-ID exposure using quota source-run IDs and complete, coextensive physical snapshots"
		row.ProviderSlots = estimates
		out = append(out, row)
	}
	if !extraAllowanceIncluded {
		out = append(out, TierEstimate{ID: TierExtraAllowance, Status: "unavailable", Reason: "no separate extra-allowance quota evidence is available"})
	}
	return out
}

func estimateTarget(in EstimateInput, now, target time.Time, maxGap, maxAge time.Duration) ([]ProviderEstimate, string) {
	groups := map[quotaKey][]pricing.QuotaObservation{}
	for _, q := range in.Quota {
		if q.Provider == "" || q.Pool == "" || q.Window == "" || q.ResetAt == nil || q.ResetAt.IsZero() || q.UsedPercent == nil {
			continue
		}
		// Durable quota history includes completed reset chains. They are not
		// evidence for the current quota allowance and must not create false
		// ambiguity or invalidate an otherwise usable current chain.
		if !q.ResetAt.After(now) {
			continue
		}
		if *q.UsedPercent < 0 || *q.UsedPercent > 100 {
			continue
		}
		key := quotaKey{q.Provider, q.Pool, q.Window, q.AccountScope, q.ResetAt.UTC()}
		groups[key] = append(groups[key], q)
	}
	if len(groups) == 0 {
		return nil, "no provider-scoped quota observations have usable percent and reset evidence"
	}
	// Keep account curves separate. A quota point is tied to physical exposure
	// only through its validated source run ID; provider identity alone is not
	// sufficient when several accounts are active.
	providers := map[string]bool{}
	for key := range groups {
		providers[key.provider] = true
	}
	accountRunOwner := map[string]string{}
	for key, points := range groups {
		for _, point := range points {
			parsed, err := uuid.Parse(point.SourceRunID)
			if err != nil || parsed == uuid.Nil {
				continue // the scoped estimate below will report this as unavailable
			}
			ownerKey := key.provider + "\x00" + parsed.String()
			if owner, exists := accountRunOwner[ownerKey]; exists && owner != key.account {
				return nil, fmt.Sprintf("%s quota source run is associated with multiple account scopes", key.provider)
			}
			accountRunOwner[ownerKey] = key.account
		}
	}
	result := make([]ProviderEstimate, 0, len(groups))
	for key, points := range groups {
		if key.account == "" {
			return nil, fmt.Sprintf("%s quota account scope is unknown; account-wide burn cannot be tied to provider runs", key.provider)
		}
		sort.Slice(points, func(i, j int) bool { return points[i].ObservedAt.Before(points[j].ObservedAt) })
		if !key.reset.After(now) {
			return nil, fmt.Sprintf("%s quota reset is not a current future reset", key.provider)
		}
		// Need a fresh final observation, a distinct burn delta, and ten minutes
		// of actual quota history in the same provider/window/reset chain.
		first, last := points[0], points[len(points)-1]
		if len(points) < 2 || last.ObservedAt.Sub(first.ObservedAt) < minBurnHistory {
			return nil, fmt.Sprintf("%s quota history is shorter than ten minutes or lacks multiple points", key.provider)
		}
		if last.ObservedAt.After(now) || now.Sub(last.ObservedAt) > maxAge {
			return nil, fmt.Sprintf("%s latest quota observation is stale", key.provider)
		}
		burn := *last.UsedPercent - *first.UsedPercent
		monotone := true
		for i := 1; i < len(points); i++ {
			if points[i].UsedPercent == nil || *points[i].UsedPercent < *points[i-1].UsedPercent {
				monotone = false
				break
			}
		}
		if !monotone {
			return nil, fmt.Sprintf("%s quota percent readings are not monotone within one reset window", key.provider)
		}
		if burn <= 0 {
			return nil, fmt.Sprintf("%s quota observations do not show positive measurable burn", key.provider)
		}
		start, end := first.ObservedAt.UTC(), last.ObservedAt.UTC()
		if end.Sub(start) < minBurnHistory {
			return nil, fmt.Sprintf("%s quota burn interval is too short", key.provider)
		}
		scopeRunIDs, validRunIDs := quotaSourceRunIDs(points)
		if !validRunIDs {
			return nil, fmt.Sprintf("%s quota account scope lacks valid source run IDs", key.provider)
		}
		exposure, reason := stableRunExposure(in.Samples, key.provider, scopeRunIDs, start, end, maxGap)
		if reason != "" {
			return nil, fmt.Sprintf("%s %s", key.provider, reason)
		}
		if exposure.seconds <= 0 {
			return nil, fmt.Sprintf("%s physical samples contain no stable actual run-ID exposure", key.provider)
		}
		percentPerRunSecond := burn / exposure.seconds
		budget := quotaBudgetUntil(key.reset, target, now, points, key.window)
		if budget < 0 {
			return nil, fmt.Sprintf("%s quota reset cadence or current remaining allowance is unknown", key.provider)
		}
		available := int(math.Floor(budget / (percentPerRunSecond * target.Sub(now).Seconds())))
		if available < 0 {
			available = 0
		}
		result = append(result, ProviderEstimate{Provider: key.provider, Pool: key.pool, Window: key.window, AccountScope: key.account, Slots: available, QuotaStart: start, QuotaEnd: end, ScopeEvidence: "provider-reported account scope mapped only to validated quota source run IDs present in complete physical samples", RunIDs: exposure.runIDs, SampleStart: exposure.sampleStart, SampleEnd: exposure.sampleEnd, SampleCount: exposure.sampleCount, RunSeconds: exposure.seconds, BurnPercent: burn})
	}
	if len(result) == 0 || len(providers) == 0 {
		return nil, "no usable provider quota burn evidence"
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Provider < result[j].Provider })
	return result, ""
}

func stableRunExposure(samples []Sample, provider string, allowedRunIDs map[string]bool, start, end time.Time, maxGap time.Duration) (runExposure, string) {
	ordered := make([]Sample, 0, len(samples))
	for _, s := range samples {
		if !s.ObservedAt.Before(start.Add(-maxGap)) && !s.ObservedAt.After(end.Add(maxGap)) {
			ordered = append(ordered, s)
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ObservedAt.Before(ordered[j].ObservedAt) })
	if len(ordered) < 2 || ordered[0].ObservedAt.After(start) || ordered[len(ordered)-1].ObservedAt.Before(end) {
		return runExposure{}, "physical snapshots do not bracket the complete quota-burn interval"
	}
	exposure := runExposure{sampleStart: ordered[0].ObservedAt.UTC(), sampleEnd: ordered[len(ordered)-1].ObservedAt.UTC(), sampleCount: len(ordered)}
	contributing := map[string]bool{}
	for i := 1; i < len(ordered); i++ {
		prev, cur := ordered[i-1], ordered[i]
		span := cur.ObservedAt.Sub(prev.ObservedAt)
		if span <= 0 {
			return runExposure{}, "physical snapshot timestamps are not strictly increasing"
		}
		if span > maxGap {
			return runExposure{}, "physical snapshot coverage has a gap wider than the supported maximum"
		}
		if !prev.Complete || !cur.Complete || len(prev.Unknown) > 0 || len(cur.Unknown) > 0 {
			return runExposure{}, "physical snapshot coverage is incomplete or carries unknowns"
		}
		if len(prev.UnmappedRunIDs) > 0 || len(cur.UnmappedRunIDs) > 0 {
			return runExposure{}, "unmapped physical run IDs prevent provider-specific attribution"
		}
		left, right := asSet(prev.ProviderRunIDs[provider]), asSet(cur.ProviderRunIDs[provider])
		from, to := prev.ObservedAt, cur.ObservedAt
		if from.Before(start) {
			from = start
		}
		if to.After(end) {
			to = end
		}
		if to.After(from) {
			for id := range left {
				if right[id] && allowedRunIDs[id] {
					exposure.seconds += to.Sub(from).Seconds()
					contributing[id] = true
				}
			}
		}
	}
	for id := range contributing {
		exposure.runIDs = append(exposure.runIDs, id)
	}
	for id := range allowedRunIDs {
		if !contributing[id] {
			return runExposure{}, "quota source run IDs are not stably present in provider-specific physical snapshots"
		}
	}
	sort.Strings(exposure.runIDs)
	return exposure, ""
}

func asSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != "" {
			set[id] = true
		}
	}
	return set
}

// quotaBudgetUntil counts only the current observed remainder before reset and
// full replenishments afterward. It requires provider-reported window length;
// an assumed reset cadence would create fictional capacity.
func quotaBudgetUntil(reset, target, now time.Time, points []pricing.QuotaObservation, window string) float64 {
	latest := points[len(points)-1]
	if latest.UsedPercent == nil || reset.IsZero() {
		return -1
	}
	remaining := 100 - *latest.UsedPercent
	if target.Before(reset) || target.Equal(reset) {
		return remaining
	}
	var minutes int64
	for _, p := range points {
		if p.WindowMinutes != nil && *p.WindowMinutes > 0 {
			minutes = *p.WindowMinutes
			break
		}
	}
	if minutes <= 0 {
		return -1
	}
	period := time.Duration(minutes) * time.Minute
	if reset.Sub(now) <= 0 {
		return -1
	}
	budget := remaining
	for boundary := reset; boundary.Before(target); boundary = boundary.Add(period) {
		end := boundary.Add(period)
		if end.After(target) {
			budget += 100 * target.Sub(boundary).Seconds() / period.Seconds()
		} else {
			budget += 100
		}
	}
	return budget
}
