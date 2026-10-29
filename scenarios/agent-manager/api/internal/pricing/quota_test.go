package pricing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
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

func quotaEvent(runID uuid.UUID, at time.Time, data domain.RateLimitEventData) *domain.RunEvent {
	event := domain.NewRateLimitEvent(runID, data.LimitType, data.Message, data.ResetTime, 0)
	event.Timestamp, event.Data = at, &data
	return event
}

func TestObservationFromEventIdentifiesTheProviderReadingNotTheCarrier(t *testing.T) {
	at := time.Date(2026, 10, 29, 4, 30, 0, 0, time.UTC)
	percent := 33.0
	frame := domain.RateLimitEventData{LimitType: "10080m", Provider: "openai", Pool: "primary", UsedPercent: &percent, WindowMinutes: 10080, Provenance: "codex:event_msg.token_count.rate_limits"}
	live, err := ObservationFromEvent(quotaEvent(uuid.New(), at, frame), at.Add(time.Hour))
	if err != nil || live == nil {
		t.Fatal(err)
	}
	imported, err := ObservationFromEvent(quotaEvent(uuid.New(), at, frame), at.Add(7*time.Hour))
	if err != nil || imported == nil {
		t.Fatal(err)
	}
	if live.ID != imported.ID || live.SourceRunID == imported.SourceRunID {
		t.Fatalf("one rollout frame through two runs must share identity: %s vs %s", live.ID, imported.ID)
	}
	if live.Freshness != ObservationFresh || imported.Freshness != ObservationStale || live.Window != "10080m" || live.Standing != QuotaStandingAvailable {
		t.Fatalf("live=%+v imported=%+v", live, imported)
	}
	later, _ := ObservationFromEvent(quotaEvent(uuid.New(), at.Add(time.Second), frame), at)
	if later.ID == live.ID {
		t.Fatal("a later frame must be a new observation")
	}
	if none, err := ObservationFromEvent(quotaEvent(uuid.New(), at, domain.RateLimitEventData{LimitType: "5_hour"}), at); none != nil || err != nil {
		t.Fatalf("an unattributed rate-limit event is not an observation: %+v %v", none, err)
	}
	if none, _ := ObservationFromEvent(domain.NewErrorEvent(uuid.New(), "x", "y", false), at); none != nil {
		t.Fatal("an error event is not a quota observation")
	}
}

func TestObservationStandingPrefersProviderStatus(t *testing.T) {
	at := time.Now().UTC()
	full := 100.0
	for status, want := range map[string]QuotaStanding{"rejected": QuotaStandingExhausted, "allowed_warning": QuotaStandingLimited, "allowed": QuotaStandingAvailable, "": QuotaStandingExhausted} {
		o, err := ObservationFromEvent(quotaEvent(uuid.New(), at, domain.RateLimitEventData{LimitType: "five_hour", Provider: "anthropic", Pool: "five_hour", WindowMinutes: 300, Status: status}), at)
		if err != nil || o.Standing != want {
			t.Fatalf("status %q standing = %+v, want %s (%v)", status, o, want, err)
		}
	}
	o, _ := ObservationFromEvent(quotaEvent(uuid.New(), at, domain.RateLimitEventData{LimitType: "10080m", Provider: "openai", Pool: "primary", UsedPercent: &full, WindowMinutes: 10080}), at)
	if o.Standing != QuotaStandingExhausted {
		t.Fatalf("100%% standing = %s", o.Standing)
	}
}

func TestQuotaReadingWhoseResetHadPassedAtObservationIsRejectedAtEveryIngestPath(t *testing.T) {
	observedAt := time.Date(2026, 10, 22, 6, 41, 0, 0, time.UTC)
	percent := 40.0
	for name, reset := range map[string]time.Time{"hours before": observedAt.Add(-12 * time.Hour), "at the observation": observedAt} {
		event := &domain.RateLimitEventData{LimitType: "300m", UsedPercent: &percent, ResetTime: &reset}
		if _, err := FromRateLimitEvent("anthropic", "five_hour", "test", observedAt, event); !errors.Is(err, ErrResetElapsed) {
			t.Errorf("%s: event path err=%v, want ErrResetElapsed", name, err)
		}
		repo := NewMemoryRepository()
		o := &QuotaObservation{Provider: "anthropic", Pool: "five_hour", Window: "300m", ObservedAt: observedAt, Standing: QuotaStandingAvailable, Provenance: "test", UsedPercent: &percent, ResetAt: &reset}
		if err := repo.RecordQuotaObservation(context.Background(), o); !errors.Is(err, ErrResetElapsed) {
			t.Errorf("%s: store path err=%v, want ErrResetElapsed", name, err)
		}
	}
	future := observedAt.Add(time.Hour)
	if _, err := FromRateLimitEvent("anthropic", "five_hour", "test", observedAt, &domain.RateLimitEventData{LimitType: "300m", UsedPercent: &percent, ResetTime: &future}); err != nil {
		t.Fatalf("a reset still ahead is valid: %v", err)
	}
}

func TestStatuslineSampleWithAPastResetYieldsNoObservation(t *testing.T) {
	observed := time.Date(2026, 10, 22, 6, 41, 0, 0, time.UTC)
	used := 71.0
	sample := claudeStatuslineSample{ObservedAt: observed, FiveHour: &claudeStatuslineWindow{UsedPercentage: &used, ResetsAt: observed.Add(-12 * time.Hour).Unix()}}
	if got := claudeStatuslineObservations(sample, "/tmp/samples", observed); len(got) != 0 {
		t.Fatalf("observations=%+v", got)
	}
}
