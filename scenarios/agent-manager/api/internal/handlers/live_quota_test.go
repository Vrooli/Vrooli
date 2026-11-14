package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/pricing"
)

func TestLiveQuotaCoalescesAndDoesNotExposeAccountData(t *testing.T) {
	var calls atomic.Int32
	gate := make(chan struct{})
	cache := &liveQuotaCache{read: func(ctx context.Context) ([]codecs.CodexLiveQuotaWindow, error) {
		calls.Add(1)
		select {
		case <-gate:
			return []codecs.CodexLiveQuotaWindow{{Pool: "codex", WindowMinutes: 10080, UsedPercent: 0}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	h := &PricingHandler{liveQuota: cache}
	var wg sync.WaitGroup
	responses := make([]liveQuota, 8)
	for i := range responses {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			rw := httptest.NewRecorder()
			h.LiveQuota(rw, httptest.NewRequest("GET", "/?agent=codex", nil))
			if err := json.Unmarshal(rw.Body.Bytes(), &responses[index]); err != nil {
				t.Error(err)
			}
		}(i)
	}
	for calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(gate)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("provider reads=%d", calls.Load())
	}
	for _, got := range responses {
		if got.Match != "likely" || len(got.Windows) != 1 || got.Windows[0].UsedPercent != 0 || got.ObservedAt.IsZero() {
			t.Fatalf("response=%+v", got)
		}
	}
	rw := httptest.NewRecorder()
	h.LiveQuota(rw, httptest.NewRequest("GET", "/?agent=unknown", nil))
	var unsupported liveQuota
	if err := json.Unmarshal(rw.Body.Bytes(), &unsupported); err != nil || unsupported.Match != "unavailable" || len(unsupported.Windows) != 0 {
		t.Fatalf("unsupported=%+v err=%v", unsupported, err)
	}
}

func TestLiveQuotaSuppressesProviderErrorDetails(t *testing.T) {
	h := &PricingHandler{liveQuota: &liveQuotaCache{read: func(context.Context) ([]codecs.CodexLiveQuotaWindow, error) {
		return nil, errors.New("secret-account-id provider detail")
	}}}
	rw := httptest.NewRecorder()
	h.LiveQuota(rw, httptest.NewRequest("GET", "/?agent=codex", nil))
	if strings.Contains(rw.Body.String(), "secret-account-id") {
		t.Fatalf("provider error leaked into response")
	}
	var result liveQuota
	if err := json.Unmarshal(rw.Body.Bytes(), &result); err != nil || result.Match != "unavailable" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestClaudeLiveQuotaUsesOneSampleAndRejectsConflicts(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	weekly, five := int64(10080), int64(300)
	p1, p2, conflict := 22.0, 55.0, 85.0
	reset := now.Add(time.Hour)
	rows := []pricing.QuotaObservation{
		{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: now, WindowMinutes: &five, UsedPercent: &p1, ResetAt: &reset},
		{Provider: "anthropic", Pool: "seven_day", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: now, WindowMinutes: &weekly, UsedPercent: &p2, ResetAt: &reset},
	}
	source := &quotaObservationHandlerFake{rows: rows}
	h := &PricingHandler{quotaStore: source}
	got := h.claudeLiveQuota(context.Background(), now)
	if got.Match != "likely" || len(got.Windows) != 2 || got.Windows[0].Pool != "subscription" || got.Windows[1].Pool != "subscription" || !got.ObservedAt.Equal(now) {
		t.Fatalf("sample=%+v", got)
	}
	source.rows = append(source.rows, pricing.QuotaObservation{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-b", ObservedAt: now.Add(-time.Minute), WindowMinutes: &five, UsedPercent: &conflict, ResetAt: &reset})
	if got = h.claudeLiveQuota(context.Background(), now); got.Match != "unavailable" || len(got.Windows) != 0 {
		t.Fatalf("conflicting sessions=%+v", got)
	}
	if got = h.claudeLiveQuota(context.Background(), now.Add(7*time.Hour)); got.Match != "unavailable" {
		t.Fatalf("stale=%+v", got)
	}
}

// The store suppresses a window's repeated reading, so windows of one account
// routinely have different observation times.
func TestClaudeLiveQuotaKeepsWindowsWhoseReadingDidNotChange(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	weekly, five := int64(10080), int64(300)
	fivePercent, weeklyPercent := 3.0, 87.0
	fiveReset, weeklyReset := now.Add(4*time.Hour), now.Add(96*time.Hour)
	weeklyObserved := now.Add(-2 * time.Minute)
	source := &quotaObservationHandlerFake{rows: []pricing.QuotaObservation{
		{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: now, WindowMinutes: &five, UsedPercent: &fivePercent, ResetAt: &fiveReset},
		{Provider: "anthropic", Pool: "seven_day", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: weeklyObserved, WindowMinutes: &weekly, UsedPercent: &weeklyPercent, ResetAt: &weeklyReset},
	}}
	got := (&PricingHandler{quotaStore: source}).claudeLiveQuota(context.Background(), now)
	if got.Match != "likely" || len(got.Windows) != 2 {
		t.Fatalf("both windows should be reported: %+v", got)
	}
	if got.Windows[0].WindowMinutes != 300 || got.Windows[0].UsedPercent != 3 || got.Windows[1].WindowMinutes != 10080 || got.Windows[1].UsedPercent != 87 {
		t.Fatalf("windows=%+v", got.Windows)
	}
	if got.Windows[1].ObservedAt == nil || !got.Windows[1].ObservedAt.Equal(weeklyObserved) || !got.ObservedAt.Equal(now) {
		t.Fatalf("observation ages lost: %+v", got)
	}
}

func TestClaudeLiveQuotaOnlyReportsWindowsTheProviderReports(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	weekly := int64(10080)
	percent := 40.0
	reset := now.Add(96 * time.Hour)
	source := &quotaObservationHandlerFake{rows: []pricing.QuotaObservation{
		{Provider: "anthropic", Pool: "seven_day", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: now, WindowMinutes: &weekly, UsedPercent: &percent, ResetAt: &reset},
	}}
	got := (&PricingHandler{quotaStore: source}).claudeLiveQuota(context.Background(), now)
	if got.Match != "likely" || len(got.Windows) != 1 || got.Windows[0].WindowMinutes != 10080 {
		t.Fatalf("weekly-only account=%+v", got)
	}
}

// Sessions of one account render at different moments, so their readings can
// differ by a tick. That is not evidence of a second account.
func TestClaudeLiveQuotaToleratesOneAccountsSessionsDisagreeingByATick(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	five := int64(300)
	older, newer := 2.0, 3.0
	reset := now.Add(4 * time.Hour)
	source := &quotaObservationHandlerFake{rows: []pricing.QuotaObservation{
		{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: now, WindowMinutes: &five, UsedPercent: &newer, ResetAt: &reset},
		{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-b", ObservedAt: now.Add(-time.Minute), WindowMinutes: &five, UsedPercent: &older, ResetAt: &reset},
	}}
	got := (&PricingHandler{quotaStore: source}).claudeLiveQuota(context.Background(), now)
	if got.Match != "likely" || len(got.Windows) != 1 || got.Windows[0].UsedPercent != 3 {
		t.Fatalf("same-account sessions=%+v", got)
	}
}

func TestClaudeLiveQuotaIgnoresAReadingFromAnElapsedWindow(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	five := int64(300)
	previous, current := 39.0, 0.0
	elapsed, reset := now.Add(-time.Second), now.Add(5*time.Hour)
	source := &quotaObservationHandlerFake{rows: []pricing.QuotaObservation{
		{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-a", ObservedAt: now, WindowMinutes: &five, UsedPercent: &current, ResetAt: &reset},
		{Provider: "anthropic", Pool: "five_hour", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: "session-b", ObservedAt: now.Add(-time.Minute), WindowMinutes: &five, UsedPercent: &previous, ResetAt: &elapsed},
	}}
	got := (&PricingHandler{quotaStore: source}).claudeLiveQuota(context.Background(), now)
	if got.Match != "likely" || len(got.Windows) != 1 || got.Windows[0].UsedPercent != 0 {
		t.Fatalf("reset boundary=%+v", got)
	}
}

func claudeReading(ref string, minutes int64, percent float64, observed, reset time.Time) pricing.QuotaObservation {
	return pricing.QuotaObservation{Provider: "anthropic", Provenance: pricing.ClaudeStatuslineProvenance, EvidenceRef: ref, ObservedAt: observed, WindowMinutes: &minutes, UsedPercent: &percent, ResetAt: &reset}
}

func TestClaudeLiveQuotaConflictKeepsItsSource(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	reset := now.Add(time.Hour)
	got := (&PricingHandler{quotaStore: &quotaObservationHandlerFake{rows: []pricing.QuotaObservation{
		claudeReading("session-a", 300, 22, now, reset),
		claudeReading("session-b", 300, 85, now.Add(-time.Minute), reset),
	}}}).claudeLiveQuota(context.Background(), now)
	if got.Match != "unavailable" || got.Source != "Claude Code statusline" {
		t.Fatalf("conflict=%+v", got)
	}
}

// Windows of one account can come from different sessions, because the store's
// repeat check ignores the session. Only a live reset that differs proves a
// second account: a window has one reset per account, however old the reading.
func TestClaudeLiveQuotaCombinesSessionsUnlessAResetProvesTwoAccounts(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	fiveReset, weekReset := now.Add(4*time.Hour), now.Add(96*time.Hour)
	for _, tc := range []struct {
		name  string
		rows  []pricing.QuotaObservation
		match string
	}{
		{"weekly deduplicated into another session", []pricing.QuotaObservation{
			claudeReading("session-a", 300, 5, now, fiveReset),
			claudeReading("session-b", 10080, 87, now.Add(-30*time.Minute), weekReset),
		}, "likely"},
		{"the other session's reading is older and lower, not a second account", []pricing.QuotaObservation{
			claudeReading("session-a", 300, 40, now, fiveReset),
			claudeReading("session-b", 10080, 87, now.Add(-30*time.Minute), weekReset),
			claudeReading("session-b", 300, 10, now.Add(-30*time.Minute), fiveReset),
		}, "likely"},
		{"an idle session's old reading is higher, not a second account", []pricing.QuotaObservation{
			claudeReading("session-a", 300, 5, now, fiveReset),
			claudeReading("session-b", 300, 60, now.Add(-30*time.Minute), fiveReset),
		}, "likely"},
		{"a session's later reading supersedes its own older reset", []pricing.QuotaObservation{
			claudeReading("session-a", 300, 5, now, fiveReset),
			claudeReading("session-b", 300, 5, now.Add(-20*time.Minute), fiveReset),
			claudeReading("session-b", 300, 5, now.Add(-time.Hour), fiveReset.Add(time.Hour)),
		}, "likely"},
		{"the weekly's session reported another five-hour window", []pricing.QuotaObservation{
			claudeReading("session-a", 300, 5, now, fiveReset),
			claudeReading("session-b", 10080, 87, now.Add(-30*time.Minute), weekReset),
			claudeReading("session-b", 300, 10, now.Add(-30*time.Minute), fiveReset.Add(time.Hour)),
		}, "unavailable"},
		{"the five-hour's session reported another weekly window", []pricing.QuotaObservation{
			claudeReading("session-a", 300, 5, now, fiveReset),
			claudeReading("session-a", 10080, 20, now.Add(-time.Hour), weekReset.Add(24*time.Hour)),
			claudeReading("session-b", 10080, 87, now.Add(-30*time.Minute), weekReset),
		}, "unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := (&PricingHandler{quotaStore: &quotaObservationHandlerFake{rows: tc.rows}}).claudeLiveQuota(context.Background(), now)
			if got.Match != tc.match {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestCodexLiveQuotaCacheDropsAWindowThatResetsWhileCached(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	reads := 0
	cache := &liveQuotaCache{
		now: func() time.Time { return now },
		read: func(context.Context) ([]codecs.CodexLiveQuotaWindow, error) {
			reads++
			soon, later := now.Add(20*time.Second), now.Add(40*time.Second)
			return []codecs.CodexLiveQuotaWindow{{Pool: "codex", WindowMinutes: 300, ResetsAt: &soon}, {Pool: "codex", WindowMinutes: 10080, ResetsAt: &later}}, nil
		},
	}
	if got := cache.cachedCodexLiveQuota(context.Background()); len(got.Windows) != 2 {
		t.Fatalf("first read=%+v", got)
	}
	now = now.Add(30 * time.Second)
	got := cache.cachedCodexLiveQuota(context.Background())
	if len(got.Windows) != 1 || got.Windows[0].WindowMinutes != 10080 || reads != 1 {
		t.Fatalf("the cached window whose reset passed must go, without a new read: reads=%d %+v", reads, got)
	}
	// With nothing current left the cached answer says nothing, so read again.
	now = now.Add(20 * time.Second)
	if got = cache.cachedCodexLiveQuota(context.Background()); reads != 2 || len(got.Windows) != 2 {
		t.Fatalf("reads=%d %+v", reads, got)
	}
}
