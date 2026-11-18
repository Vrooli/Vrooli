package utilizationhistory

import (
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

func TestEstimateUnavailableWhenSampleCoverageIsNotCoextensive(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	input := estimateFixture(now, now.Add(8*time.Hour))
	input.Samples = input.Samples[:len(input.Samples)-1]
	got := Estimate(input)
	if got[0].Slots != nil || got[0].Status != "unavailable" {
		t.Fatalf("incomplete endpoint unexpectedly supported capacity: %+v", got[0])
	}
}

func TestEstimateUnavailableOnSnapshotGapAndUnmappedRun(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for name, mutate := range map[string]func(*EstimateInput){
		"gap":      func(in *EstimateInput) { in.Samples = append(in.Samples[:5], in.Samples[8:]...) },
		"unmapped": func(in *EstimateInput) { in.Samples[2].UnmappedRunIDs = []string{"unmapped-7"} },
	} {
		t.Run(name, func(t *testing.T) {
			in := estimateFixture(now, now.Add(8*time.Hour))
			mutate(&in)
			got := Estimate(in)
			if got[0].Slots != nil {
				t.Fatalf("unsupported evidence returned %d slots", *got[0].Slots)
			}
		})
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

func TestEstimateRejectsMissingOrMalformedScopedSourceRunIDs(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for name, sourceID := range map[string]string{"missing": "", "malformed": "run-not-a-uuid"} {
		t.Run(name, func(t *testing.T) {
			in := estimateFixture(now, now.Add(8*time.Hour))
			in.Quota[0].SourceRunID = sourceID
			got := Estimate(in)
			if got[0].Slots != nil || got[0].Status != "unavailable" {
				t.Fatalf("scope with %s source run ID must remain unavailable: %+v", name, got[0])
			}
		})
	}
}

func TestEstimatePreservesUnknownPhysicalCoverageGuardForScopedRuns(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	in := estimateFixture(now, now.Add(8*time.Hour))
	in.Samples[4].Unknown = []string{"unclassified host process"}
	got := Estimate(in)
	if got[0].Slots != nil || got[0].Status != "unavailable" {
		t.Fatalf("unknown physical coverage must invalidate scoped exposure: %+v", got[0])
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
