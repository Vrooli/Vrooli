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
