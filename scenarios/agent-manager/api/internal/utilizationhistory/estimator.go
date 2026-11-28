package utilizationhistory

import (
	"math"
	"sort"
	"strings"
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
	// ProviderSlots preserves each measured quota window's constraint.
	ProviderSlots []ProviderEstimate
	// Unavailable lists the current quota windows that could not be measured.
	Unavailable []ScopeUnavailable
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

// ScopeUnavailable explains why one current quota window contributes no
// ceiling. It never blocks another window's measured estimate.
type ScopeUnavailable struct {
	Provider     string
	Pool         string
	Window       string
	AccountScope string
	Reason       string
}

type runExposure struct {
	seconds     float64
	runIDs      []string
	sampleStart time.Time
	sampleEnd   time.Time
	sampleCount int
}

type quotaKey struct {
	provider, pool, window, account string
	reset                           time.Time
}

// burnHorizon bounds the trailing interval a burn rate is measured over, so
// the ceiling follows the current pace rather than the whole reset window.
const burnHorizon = 24 * time.Hour

// Estimate computes integer ceilings from observed quota-percent burn divided
// by physical run-seconds of the runs that reported that quota.
//
// Every bias in the evidence is resolved toward fewer slots. Account burn
// includes usage no run reported (standalone sessions, unmapped runs), and
// exposure counts a source run only between adjacent samples that both
// positively observed it, never across a gap wider than MaxSampleGap; an
// incomplete or unknown sample can hide a run but never invent one. Within one
// reset chain the used percent is the running maximum, because concurrent
// sessions keep rendering older cached percents.
//
// Each current quota window (provider, pool, window, account, reset) is
// estimated on its own. A window that cannot be measured (stale, no positive
// burn, no source runs, ambiguous account membership) is reported in
// Unavailable and does not hide another window's measurement; a tier is
// available when at least one window is measured, and its ceiling is the
// smallest measured window ceiling.
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
		estimates, unavailable, reason := estimateTarget(in, now, target.At.UTC(), gap, maxAge)
		row.Unavailable = unavailable
		if len(estimates) == 0 {
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
		row.ProviderSlots = estimates
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
		row.Basis = "smallest per-window ceiling: provider quota-percent burn (running maximum, trailing 24h of the current reset window) divided by physical run-seconds of the quota's source runs; unreported usage and unobserved runs only lower the ceiling"
		if len(unavailable) > 0 {
			row.Basis += "; unmeasured windows are listed separately and are not constrained by this ceiling"
		}
		out = append(out, row)
	}
	if !extraAllowanceIncluded {
		out = append(out, TierEstimate{ID: TierExtraAllowance, Status: "unavailable", Reason: "no separate extra-allowance quota evidence is available"})
	}
	return out
}

func estimateTarget(in EstimateInput, now, target time.Time, maxGap, maxAge time.Duration) ([]ProviderEstimate, []ScopeUnavailable, string) {
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
		return nil, nil, "no provider-scoped quota observations have usable percent and reset evidence"
	}
	// A source run reports the account it actually uses. One run reporting two
	// account scopes of a provider makes that provider's membership ambiguous.
	ambiguous := map[string]bool{}
	accountRunOwner := map[string]string{}
	for key, points := range groups {
		for _, point := range points {
			id, ok := validRunID(point.SourceRunID)
			if !ok {
				continue
			}
			ownerKey := key.provider + "\x00" + id
			if owner, exists := accountRunOwner[ownerKey]; exists && owner != key.account {
				ambiguous[key.provider] = true
			}
			accountRunOwner[ownerKey] = key.account
		}
	}
	keys := make([]quotaKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.provider != b.provider {
			return a.provider < b.provider
		}
		if a.pool != b.pool {
			return a.pool < b.pool
		}
		if a.account != b.account {
			return a.account < b.account
		}
		return a.reset.Before(b.reset)
	})
	result := make([]ProviderEstimate, 0, len(groups))
	unavailable := []ScopeUnavailable{}
	for _, key := range keys {
		estimate, reason := estimateWindow(key, groups[key], in.Samples, ambiguous[key.provider], now, target, maxGap, maxAge)
		if reason != "" {
			unavailable = append(unavailable, ScopeUnavailable{Provider: key.provider, Pool: key.pool, Window: key.window, AccountScope: key.account, Reason: reason})
			continue
		}
		result = append(result, estimate)
	}
	if len(result) == 0 {
		reasons := make([]string, 0, len(unavailable))
		for _, u := range unavailable {
			reasons = append(reasons, u.Reason)
		}
		return nil, unavailable, strings.Join(reasons, "; ")
	}
	return result, unavailable, ""
}

func estimateWindow(key quotaKey, points []pricing.QuotaObservation, samples []Sample, ambiguous bool, now, target time.Time, maxGap, maxAge time.Duration) (ProviderEstimate, string) {
	label := key.provider + " " + key.pool
	if ambiguous {
		return ProviderEstimate{}, label + ": a quota source run is associated with multiple account scopes"
	}
	sort.Slice(points, func(i, j int) bool { return points[i].ObservedAt.Before(points[j].ObservedAt) })
	first, last := points[0], points[len(points)-1]
	if last.ObservedAt.After(now) || now.Sub(last.ObservedAt) > maxAge {
		return ProviderEstimate{}, label + ": latest quota observation is stale"
	}
	end := last.ObservedAt.UTC()
	start := first.ObservedAt.UTC()
	if horizon := end.Add(-burnHorizon); horizon.After(start) {
		start = horizon
	}
	if len(points) < 2 || end.Sub(start) < minBurnHistory {
		return ProviderEstimate{}, label + ": quota history is shorter than ten minutes or lacks multiple points"
	}
	// Running maximum: an idle session re-renders an older cached percent, so
	// a lower later reading is not a refund.
	usedAt := func(at time.Time) float64 {
		value := 0.0
		for _, p := range points {
			if p.ObservedAt.After(at) {
				break
			}
			if *p.UsedPercent > value {
				value = *p.UsedPercent
			}
		}
		return value
	}
	used := usedAt(end)
	burn := used - usedAt(start)
	if burn <= 0 {
		return ProviderEstimate{}, label + ": quota observations do not show positive measurable burn"
	}
	sources := map[string]bool{}
	for _, p := range points {
		if p.ObservedAt.Before(start) {
			continue
		}
		if id, ok := validRunID(p.SourceRunID); ok {
			sources[id] = true
		}
	}
	if len(sources) == 0 {
		return ProviderEstimate{}, label + ": quota readings in the burn interval carry no source run IDs, so the burn cannot be tied to runs"
	}
	exposure := sourceRunExposure(samples, sources, start, end, maxGap)
	if exposure.seconds <= 0 {
		return ProviderEstimate{}, label + ": no quota source run was positively observed in adjacent physical samples during the burn interval"
	}
	percentPerRunSecond := burn / exposure.seconds
	budget := quotaBudgetUntil(key.reset, target, now, 100-used, points)
	if budget < 0 {
		return ProviderEstimate{}, label + ": quota reset cadence or current remaining allowance is unknown"
	}
	available := int(math.Floor(budget / (percentPerRunSecond * target.Sub(now).Seconds())))
	if available < 0 {
		available = 0
	}
	evidence := "quota source run IDs mapped to physically observed run-seconds"
	if key.account != "" {
		evidence = "provider-reported account scope; " + evidence
	} else {
		evidence = "account scope not reported; membership established only by " + evidence
	}
	return ProviderEstimate{Provider: key.provider, Pool: key.pool, Window: key.window, AccountScope: key.account, Slots: available, QuotaStart: start, QuotaEnd: end, ScopeEvidence: evidence, RunIDs: exposure.runIDs, SampleStart: exposure.sampleStart, SampleEnd: exposure.sampleEnd, SampleCount: exposure.sampleCount, RunSeconds: exposure.seconds, BurnPercent: burn}, ""
}

func validRunID(value string) (string, bool) {
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return "", false
	}
	return parsed.String(), true
}

// sourceRunExposure sums, for each pair of adjacent samples no further apart
// than maxGap, the clipped interval during which a source run was positively
// observed at both ends. Provider mapping is not required: the run reported
// the quota itself.
func sourceRunExposure(samples []Sample, sources map[string]bool, start, end time.Time, maxGap time.Duration) runExposure {
	ordered := make([]Sample, 0, len(samples))
	for _, s := range samples {
		if !s.ObservedAt.Before(start.Add(-maxGap)) && !s.ObservedAt.After(end.Add(maxGap)) {
			ordered = append(ordered, s)
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ObservedAt.Before(ordered[j].ObservedAt) })
	exposure := runExposure{sampleCount: len(ordered)}
	if len(ordered) > 0 {
		exposure.sampleStart, exposure.sampleEnd = ordered[0].ObservedAt.UTC(), ordered[len(ordered)-1].ObservedAt.UTC()
	}
	observed := func(s Sample) map[string]bool {
		set := asSet(s.UnmappedRunIDs)
		for _, ids := range s.ProviderRunIDs {
			for _, id := range ids {
				if id != "" {
					set[id] = true
				}
			}
		}
		return set
	}
	contributing := map[string]bool{}
	for i := 1; i < len(ordered); i++ {
		prev, cur := ordered[i-1], ordered[i]
		span := cur.ObservedAt.Sub(prev.ObservedAt)
		if span <= 0 || span > maxGap {
			continue
		}
		from, to := prev.ObservedAt, cur.ObservedAt
		if from.Before(start) {
			from = start
		}
		if to.After(end) {
			to = end
		}
		if !to.After(from) {
			continue
		}
		left, right := observed(prev), observed(cur)
		for id := range sources {
			if left[id] && right[id] {
				exposure.seconds += to.Sub(from).Seconds()
				contributing[id] = true
			}
		}
	}
	for id := range contributing {
		exposure.runIDs = append(exposure.runIDs, id)
	}
	sort.Strings(exposure.runIDs)
	return exposure
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
func quotaBudgetUntil(reset, target, now time.Time, remaining float64, points []pricing.QuotaObservation) float64 {
	if reset.IsZero() || remaining < 0 {
		return -1
	}
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
