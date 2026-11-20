package utilizationhistory

import (
	"strings"
	"testing"
	"time"

	"agent-manager/internal/pricing"
)

func TestEstimateUsesProviderScopedBurnAndActualStableRunIDs(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	reset := now.Add(8 * time.Hour)
	input := estimateFixture(now, reset)
	got := Estimate(input)
	if len(got) != 4 {
		t.Fatalf("got %d tiers, want 4", len(got))
	}
	for _, tier := range got[:3] {
		if tier.Status != "available" || tier.Slots == nil {
			t.Fatalf("%s unavailable: %+v", tier.ID, tier)
		}
	}
	if *got[0].Slots != 29 || *got[0].Slots <= 10 {
		t.Fatalf("one-hour measured ceiling should exceed the former default of 10: slots=%d", *got[0].Slots)
	}
	if got[3].Slots != nil || got[3].Status != "unavailable" {
		t.Fatalf("extra allowance must remain unknown: %+v", got[3])
	}
	if len(got[0].ProviderSlots) != 1 || got[0].ProviderSlots[0].AccountScope != "verified-account" {
		t.Fatalf("scope provenance missing: %+v", got[0].ProviderSlots)
	}
	provider := got[0].ProviderSlots[0]
	if len(provider.RunIDs) != 1 || provider.RunIDs[0] != "123e4567-e89b-12d3-a456-426614174000" || provider.SampleCount != 11 || !provider.SampleStart.Equal(input.Quota[0].ObservedAt) || !provider.SampleEnd.Equal(input.Quota[1].ObservedAt) {
		t.Fatalf("physical run/window provenance missing: %+v", provider)
	}
}

// Missing coverage can hide a run's presence but never invent it, so it may
// only lower the ceiling: a truncated endpoint, a gap wider than the sampler
// supports, an unmapped neighbour or an unknown sample never raise slots.
func TestEstimateCoverageLossOnlyLowersTheCeiling(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	full := *Estimate(estimateFixture(now, now.Add(8*time.Hour)))[0].Slots
	for name, mutate := range map[string]func(*EstimateInput){
		"truncated endpoint": func(in *EstimateInput) { in.Samples = in.Samples[:len(in.Samples)-1] },
		"gap":                func(in *EstimateInput) { in.Samples = append(in.Samples[:5], in.Samples[8:]...) },
		"unmapped neighbour": func(in *EstimateInput) { in.Samples[2].UnmappedRunIDs = []string{"unmapped-7"} },
		"unknown sample": func(in *EstimateInput) {
			in.Samples[4].Complete = false
			in.Samples[4].Unknown = []string{"unclassified host process"}
		},
		"run unobserved": func(in *EstimateInput) { in.Samples[3].ProviderRunIDs = map[string][]string{} },
	} {
		t.Run(name, func(t *testing.T) {
			in := estimateFixture(now, now.Add(8*time.Hour))
			mutate(&in)
			got := Estimate(in)[0]
			if got.Slots != nil && *got.Slots > full {
				t.Fatalf("coverage loss raised the ceiling: %d > %d", *got.Slots, full)
			}
		})
	}
	in := estimateFixture(now, now.Add(8*time.Hour))
	in.Samples = nil
	if got := Estimate(in)[0]; got.Slots != nil || got.Status != "unavailable" || len(got.Unavailable) != 1 {
		t.Fatalf("no physical observation of the source run must stay unavailable: %+v", got)
	}
}

func TestEstimateKeepsScopeUnavailableWhenQuotaSourceRunIDMissing(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	in := estimateFixture(now, now.Add(8*time.Hour))
	other := in.Quota[1]
	other.AccountScope = "another-account"
	in.Quota = append(in.Quota, other)
	got := Estimate(in)
	if got[0].Slots != nil || got[0].Reason == "" {
		t.Fatalf("ambiguous account scope did not remain unavailable: %+v", got[0])
	}
}

func TestEstimateMapsMultipleQuotaAccountsToTheirExactSourceRuns(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	start, reset := now.Add(-10*time.Minute), now.Add(8*time.Hour)
	const runA = "123e4567-e89b-12d3-a456-426614174000"
	const runB = "123e4567-e89b-12d3-a456-426614174001"
	usedA0, usedA1, usedB0, usedB1 := 10.0, 10.5, 20.0, 21.0
	minutes := int64(60)
	in := estimateFixture(now, reset)
	in.Quota = []pricing.QuotaObservation{
		{Provider: "anthropic", AccountScope: "account-a", Pool: "five_hour", Window: "300m", ObservedAt: start, UsedPercent: &usedA0, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: runA},
		{Provider: "anthropic", AccountScope: "account-a", Pool: "five_hour", Window: "300m", ObservedAt: now, UsedPercent: &usedA1, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: runA},
		{Provider: "anthropic", AccountScope: "account-b", Pool: "five_hour", Window: "300m", ObservedAt: start, UsedPercent: &usedB0, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: runB},
		{Provider: "anthropic", AccountScope: "account-b", Pool: "five_hour", Window: "300m", ObservedAt: now, UsedPercent: &usedB1, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: runB},
	}
	for i := range in.Samples {
		in.Samples[i].ProviderRunIDs = map[string][]string{"anthropic": {runA, runB}}
	}
	got := Estimate(in)
	if got[0].Status != "available" || got[0].Slots == nil || len(got[0].ProviderSlots) != 2 {
		t.Fatalf("separate account curves with explicit run mapping should be usable: %+v", got[0])
	}
	byScope := map[string]ProviderEstimate{}
	for _, provider := range got[0].ProviderSlots {
		byScope[provider.AccountScope] = provider
	}
	for scope, wantRun := range map[string]string{"account-a": runA, "account-b": runB} {
		p, ok := byScope[scope]
		if !ok || len(p.RunIDs) != 1 || p.RunIDs[0] != wantRun || p.RunSeconds != 600 {
			t.Fatalf("scope %s exposure included wrong physical run IDs: %+v", scope, p)
		}
	}
}

// A reading without a valid source run ID still counts toward the account's
// burn, but only valid IDs establish which runs that burn is divided over. A
// window none of whose readings name a run cannot be measured.
func TestEstimateRequiresAtLeastOneValidSourceRunID(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for name, sourceID := range map[string]string{"missing": "", "malformed": "run-not-a-uuid"} {
		t.Run(name, func(t *testing.T) {
			in := estimateFixture(now, now.Add(8*time.Hour))
			in.Quota[0].SourceRunID = sourceID
			if got := Estimate(in)[0]; got.Status != "available" || len(got.ProviderSlots) != 1 || len(got.ProviderSlots[0].RunIDs) != 1 {
				t.Fatalf("one valid source run should measure the window: %+v", got)
			}
			in.Quota[1].SourceRunID = sourceID
			got := Estimate(in)[0]
			if got.Slots != nil || got.Status != "unavailable" || len(got.Unavailable) != 1 || !strings.Contains(got.Reason, "no source run IDs") {
				t.Fatalf("window with %s source run IDs must remain unavailable: %+v", name, got)
			}
		})
	}
}

// Live shape of 2026-11-20: the Claude statusline reported a scoped account
// without source runs while Codex reported unscoped readings from its runs.
// The unmeasurable Claude window is reported; it does not hide the Codex
// measurement.
func TestEstimateMeasuresEachWindowIndependently(t *testing.T) {
	now := time.Date(2026, 11, 20, 15, 10, 0, 0, time.UTC)
	in := estimateFixture(now, now.Add(8*time.Hour))
	for i := range in.Quota {
		in.Quota[i].AccountScope = "" // Codex frames carry no account scope
	}
	claudeReset := now.Add(5 * 24 * time.Hour)
	weekly := int64(10080)
	flat := 1.0
	for _, at := range []time.Time{now.Add(-30 * time.Minute), now.Add(-time.Minute)} {
		in.Quota = append(in.Quota, pricing.QuotaObservation{Provider: "anthropic", AccountScope: "anthropic:sha256:a", Pool: "seven_day", Window: "10080m", ObservedAt: at, UsedPercent: &flat, ResetAt: &claudeReset, WindowMinutes: &weekly})
	}
	got := Estimate(in)
	for _, tier := range got[:3] {
		if tier.Status != "available" || tier.Slots == nil || len(tier.ProviderSlots) != 1 || tier.ProviderSlots[0].Provider != "openai" {
			t.Fatalf("%s: measurable Codex window hidden by Claude: %+v", tier.ID, tier)
		}
		if len(tier.Unavailable) != 1 || tier.Unavailable[0].Provider != "anthropic" || tier.Unavailable[0].Reason == "" {
			t.Fatalf("%s: unmeasured Claude window not reported: %+v", tier.ID, tier.Unavailable)
		}
		if !strings.Contains(tier.ProviderSlots[0].ScopeEvidence, "account scope not reported") {
			t.Fatalf("unscoped membership evidence not disclosed: %+v", tier.ProviderSlots[0])
		}
	}
}

// Concurrent sessions re-render older cached percents. Within one reset chain
// a lower later reading is not a refund: burn and remaining allowance use the
// running maximum.
func TestEstimateUsesRunningMaximumAcrossCachedReadings(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	in := estimateFixture(now, now.Add(8*time.Hour))
	peak, cached := 12.0, 10.0
	mid := in.Quota[0]
	mid.ObservedAt, mid.UsedPercent = now.Add(-5*time.Minute), &peak
	last := in.Quota[1]
	last.UsedPercent = &cached
	in.Quota = []pricing.QuotaObservation{in.Quota[0], mid, last}
	got := Estimate(in)[0]
	if got.Status != "available" || len(got.ProviderSlots) != 1 || got.ProviderSlots[0].BurnPercent != 2 {
		t.Fatalf("burn must follow the running maximum: %+v", got)
	}
}

func estimateFixture(now, reset time.Time) EstimateInput {
	start := now.Add(-10 * time.Minute)
	used0, used1 := 10.0, 10.5
	minutes := int64(60)
	quota := []pricing.QuotaObservation{
		{Provider: "openai", AccountScope: "verified-account", Pool: "codex", Window: "60m", ObservedAt: start, UsedPercent: &used0, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: "123e4567-e89b-12d3-a456-426614174000"},
		{Provider: "openai", AccountScope: "verified-account", Pool: "codex", Window: "60m", ObservedAt: now, UsedPercent: &used1, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: "123e4567-e89b-12d3-a456-426614174000"},
	}
	samples := make([]Sample, 0, 11)
	for i := 0; i <= 10; i++ {
		at := start.Add(time.Duration(i) * time.Minute)
		samples = append(samples, Sample{ObservedAt: at, Complete: true, ProviderRunIDs: map[string][]string{"openai": {"123e4567-e89b-12d3-a456-426614174000"}}})
	}
	return EstimateInput{Now: now, Targets: []Target{{ID: TierLastReset, At: now.Add(time.Hour)}, {ID: TierUseByThursday, At: now.Add(2 * time.Hour)}, {ID: TierUseByWednesday, At: now.Add(3 * time.Hour)}, {ID: TierExtraAllowance, At: now.Add(4 * time.Hour)}}, Quota: quota, Samples: samples}
}

func TestEstimateAlwaysReturnsUnavailableExtraAllowance(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	input := estimateFixture(now, now.Add(8*time.Hour))
	input.Targets = input.Targets[:3]
	got := Estimate(input)
	if len(got) != 4 || got[3].ID != TierExtraAllowance || got[3].Status != "unavailable" || got[3].Slots != nil {
		t.Fatalf("missing explicit extra-allowance unknown: %+v", got)
	}
}

func TestEstimateIgnoresExpiredQuotaChainsBeforeScopeAmbiguityChecks(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	input := estimateFixture(now, now.Add(8*time.Hour))
	oldReset := now.Add(-time.Hour)
	oldAt := now.Add(-2 * time.Hour)
	oldUsedA, oldUsedB := 30.0, 35.0
	input.Quota = append(input.Quota,
		pricing.QuotaObservation{Provider: "openai", AccountScope: "expired-account", Pool: "codex", Window: "60m", ObservedAt: oldAt, UsedPercent: &oldUsedA, ResetAt: &oldReset},
		pricing.QuotaObservation{Provider: "openai", AccountScope: "expired-account", Pool: "codex", Window: "60m", ObservedAt: oldAt.Add(10 * time.Minute), UsedPercent: &oldUsedB, ResetAt: &oldReset},
	)
	got := Estimate(input)
	if got[0].Status != "available" || got[0].Slots == nil {
		t.Fatalf("expired reset chain invalidated current evidence: %+v", got[0])
	}
	if len(got[0].ProviderSlots) != 1 || got[0].ProviderSlots[0].AccountScope != "verified-account" {
		t.Fatalf("expired scope leaked into current provenance: %+v", got[0].ProviderSlots)
	}
}
