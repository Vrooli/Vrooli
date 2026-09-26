package pricing

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

func TestQuotaObservationFreshnessPreservesMissingAndUnknown(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	base := &QuotaObservation{Provider: "openai", Pool: "codex", Window: "5h", ObservedAt: now.Add(-time.Minute), Standing: QuotaStandingAvailable, Provenance: "codex-native-rate-limit"}
	tests := []struct {
		name        string
		observation *QuotaObservation
		want        ObservationFreshness
	}{
		{name: "fresh", observation: base, want: ObservationFresh},
		{name: "stale", observation: func() *QuotaObservation { o := *base; o.ObservedAt = now.Add(-2 * time.Hour); return &o }(), want: ObservationStale},
		{name: "missing", observation: nil, want: ObservationMissing},
		{name: "unknown timestamp", observation: func() *QuotaObservation { o := *base; o.ObservedAt = time.Time{}; return &o }(), want: ObservationUnknown},
		{name: "future clock skew", observation: func() *QuotaObservation { o := *base; o.ObservedAt = now.Add(time.Minute); return &o }(), want: ObservationUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Freshness(tc.observation, now, time.Hour); got != tc.want {
				t.Fatalf("freshness=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestFromRateLimitEventDoesNotInventQuotaAmounts(t *testing.T) {
	observedAt := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	o, err := FromRateLimitEvent("openai", "codex", "codex-native-rate-limit", observedAt, &domain.RateLimitEventData{LimitType: "5h", Message: "limit reached"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Standing != QuotaStandingExhausted || o.Used != nil || o.Limit != nil || o.Remaining != nil {
		t.Fatalf("provider event invented quota amounts: %+v", o)
	}
	if o.Provider != "openai" || o.Pool != "codex" || o.Window != "5h" || o.Provenance != "codex-native-rate-limit" {
		t.Fatalf("provider attribution was not retained: %+v", o)
	}
	if o.Uncertainty == "" || !strings.Contains(o.Uncertainty, "account scope unavailable") {
		t.Fatalf("missing credential/account scope limitation: %+v", o)
	}
}

func TestFromRateLimitEventRequiresAttribution(t *testing.T) {
	_, err := FromRateLimitEvent("", "codex", "", time.Now(), &domain.RateLimitEventData{LimitType: "5h"})
	if err == nil {
		t.Fatal("missing provider/provenance accepted")
	}
}

func TestMemoryQuotaObservationRepositoryRetainsImmutableObservation(t *testing.T) {
	repo := NewMemoryRepository()
	used := int64(12)
	o := &QuotaObservation{ID: "q-1", Provider: "openai", Pool: "codex", Window: "5h", ObservedAt: time.Unix(10, 0).UTC(), Standing: QuotaStandingLimited, Used: &used, Provenance: "native"}
	if err := repo.RecordQuotaObservation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordQuotaObservation(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	used = 99
	rows, err := repo.ListQuotaObservations(context.Background(), "openai", "codex", "5h", 10)
	if err != nil || len(rows) != 1 || rows[0].Used == nil || *rows[0].Used != 12 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	*rows[0].Used = 77
	rows, err = repo.ListQuotaObservations(context.Background(), "openai", "codex", "5h", 10)
	if err != nil || len(rows) != 1 || rows[0].Used == nil || *rows[0].Used != 12 {
		t.Fatalf("list returned mutable repository state: rows=%+v err=%v", rows, err)
	}
}
