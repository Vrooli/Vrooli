package usagepace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runtimeactivity"
	"agent-manager/internal/utilizationhistory"
	"github.com/gorilla/mux"
)

type paceReadings struct {
	spans []pricing.QuotaReadingSpan
	quota []pricing.QuotaObservation
	err   error
}

func (p paceReadings) QuotaReadingSpans(context.Context, string, time.Time) ([]pricing.QuotaReadingSpan, error) {
	return p.spans, p.err
}

func (p paceReadings) RecordQuotaObservation(context.Context, *pricing.QuotaObservation) error {
	return nil
}
func (p paceReadings) ListQuotaObservations(context.Context, string, string, string, int) ([]pricing.QuotaObservation, error) {
	return p.quota, p.err
}

type paceHistory []utilizationhistory.Sample

func (p paceHistory) Read(context.Context, time.Time, time.Time) ([]utilizationhistory.Sample, error) {
	return p, nil
}

type paceSettings struct{ value config.OrchestrationSettings }

type paceActivity struct{ snapshot runtimeactivity.Snapshot }

func (p paceActivity) Read(context.Context) runtimeactivity.Snapshot { return p.snapshot }

func (p *paceSettings) GetOrchestrationSettings(context.Context) (*config.OrchestrationSettings, error) {
	copy := p.value
	return &copy, nil
}

func (p *paceSettings) UpdateOrchestrationSettings(_ context.Context, value *config.OrchestrationSettings) error {
	if err := value.Validate(); err != nil {
		return err
	}
	p.value = *value
	return nil
}

func TestPaceHTTPPreservesUnavailableEstimateAndBaselineReset(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	settings := config.DefaultOrchestrationSettings()
	settings.RunExecution.MaxConcurrentRuns = 10
	settings.UsagePace.BaseMaxConcurrentRuns = 10
	store := &paceSettings{value: settings}
	handler := NewHandler(paceReadings{spans: quotaSpans(now)}, store, paceActivity{snapshot: observedActivity(2, now)})
	handler.now = func() time.Time { return now }
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, APIPath, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", get.Code, get.Body.String())
	}
	var before response
	if err := json.Unmarshal(get.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	if len(before.Tiers) != 4 || before.Tiers[0].Status != "unavailable" || before.Tiers[0].AvailableSlots != nil || before.Tiers[3].Status != "unavailable" {
		t.Fatalf("GET tiers=%+v, want unavailable estimates without historical utilization", before.Tiers)
	}
	if before.Tiers[0].ObservedExecutingRuns == nil || *before.Tiers[0].ObservedExecutingRuns != 2 || !strings.Contains(before.Tiers[0].Reason, "point-in-time snapshot") {
		t.Fatalf("GET did not retain the honest current-utilization limitation: %+v", before.Tiers[0])
	}

	put := httptest.NewRecorder()
	router.ServeHTTP(put, httptest.NewRequest(http.MethodPut, APIPath, strings.NewReader(`{"selected_tier":"use_by_wednesday"}`)))
	if put.Code != http.StatusConflict || store.value.UsagePace.SelectedTier != "" || store.value.RunExecution.MaxConcurrentRuns != 10 {
		t.Fatalf("PUT accepted unsupported tier: status=%d settings=%+v body=%s", put.Code, store.value, put.Body.String())
	}

	reset := httptest.NewRecorder()
	router.ServeHTTP(reset, httptest.NewRequest(http.MethodPut, APIPath, strings.NewReader(`{"selected_tier":"baseline"}`)))
	if reset.Code != http.StatusOK {
		t.Fatalf("PUT baseline status=%d body=%s", reset.Code, reset.Body.String())
	}
	if store.value.UsagePace.SelectedTier != "" || store.value.UsagePace.TargetRunOutAt != "" || store.value.RunExecution.MaxConcurrentRuns != 10 {
		t.Fatalf("baseline reset left pace active: pace=%+v runExecution=%+v", store.value.UsagePace, store.value.RunExecution)
	}
}

func TestPaceHTTPRefusesTierWithoutRealAllowanceSource(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	settings := config.DefaultOrchestrationSettings()
	store := &paceSettings{value: settings}
	handler := NewHandler(paceReadings{}, store, paceActivity{})
	handler.now = func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, APIPath, nil))
	if get.Code != http.StatusOK || strings.Contains(get.Body.String(), `"available_slots":0`) {
		t.Fatalf("GET fabricated slots for missing quota source: status=%d body=%s", get.Code, get.Body.String())
	}
	put := httptest.NewRecorder()
	router.ServeHTTP(put, httptest.NewRequest(http.MethodPut, APIPath, strings.NewReader(`{"selected_tier":"extra_allowance"}`)))
	if put.Code != http.StatusConflict || store.value.UsagePace.SelectedTier != "" {
		t.Fatalf("unavailable extra allowance selection status=%d settings=%+v body=%s", put.Code, store.value.UsagePace, put.Body.String())
	}
}

func TestPaceHTTPRefusesQuotaTierWithoutMeasuredUtilization(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	settings := config.DefaultOrchestrationSettings()
	store := &paceSettings{value: settings}
	handler := NewHandler(paceReadings{spans: quotaSpans(now)}, store, nil)
	handler.now = func() time.Time { return now }
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	var before response
	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, APIPath, nil))
	if get.Code != http.StatusOK || json.Unmarshal(get.Body.Bytes(), &before) != nil {
		t.Fatalf("GET status=%d body=%s", get.Code, get.Body.String())
	}
	for _, tier := range before.Tiers[:3] {
		if tier.Status != "unavailable" || tier.AvailableSlots != nil {
			t.Errorf("tier %s estimated capacity without utilization evidence: %+v", tier.ID, tier)
		}
	}
	put := httptest.NewRecorder()
	router.ServeHTTP(put, httptest.NewRequest(http.MethodPut, APIPath, strings.NewReader(`{"selected_tier":"use_by_wednesday"}`)))
	if put.Code != http.StatusConflict || store.value.UsagePace.SelectedTier != "" || store.value.RunExecution.MaxConcurrentRuns != settings.RunExecution.MaxConcurrentRuns {
		t.Fatalf("selection without measured utilization status=%d settings=%+v body=%s", put.Code, store.value, put.Body.String())
	}
}

func TestPaceHTTPMeasuredTierUpdatesAdmissionAndReturnsRunEvidence(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	reset := now.Add(8 * time.Hour)
	minutes := int64(7 * 24 * 60)
	used0, used1 := 10.0, 10.005
	runID := "11111111-1111-4111-8111-111111111111"
	quota := []pricing.QuotaObservation{
		{Provider: "openai", AccountScope: "verified-account", Pool: "codex", Window: "10080m", ObservedAt: now.Add(-10 * time.Minute), UsedPercent: &used0, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: runID},
		{Provider: "openai", AccountScope: "verified-account", Pool: "codex", Window: "10080m", ObservedAt: now, UsedPercent: &used1, ResetAt: &reset, WindowMinutes: &minutes, SourceRunID: runID},
		// A Claude window with no measurable burn is reported, not blocking.
		{Provider: "anthropic", AccountScope: "anthropic:sha256:a", Pool: "seven_day", Window: "10080m", ObservedAt: now.Add(-20 * time.Minute), UsedPercent: &used0, ResetAt: &reset, WindowMinutes: &minutes},
		{Provider: "anthropic", AccountScope: "anthropic:sha256:a", Pool: "seven_day", Window: "10080m", ObservedAt: now, UsedPercent: &used0, ResetAt: &reset, WindowMinutes: &minutes},
	}
	spans := []pricing.QuotaReadingSpan{
		{Provider: "openai", AccountScope: "verified-account", Pool: "codex", Window: "10080m", WindowMinutes: &minutes, ResetAt: &reset, UsedPercent: &used0, Standing: pricing.QuotaStandingAvailable, FirstAt: now.Add(-10 * time.Minute), LastAt: now.Add(-10 * time.Minute), Samples: 1},
		{Provider: "openai", AccountScope: "verified-account", Pool: "codex", Window: "10080m", WindowMinutes: &minutes, ResetAt: &reset, UsedPercent: &used1, Standing: pricing.QuotaStandingAvailable, FirstAt: now, LastAt: now, Samples: 1},
	}
	samples := make(paceHistory, 0, 11)
	for i := 0; i <= 10; i++ {
		samples = append(samples, utilizationhistory.Sample{ObservedAt: now.Add(-10*time.Minute + time.Duration(i)*time.Minute), Complete: true, ProviderRunIDs: map[string][]string{"openai": {runID}}})
	}
	settings := config.DefaultOrchestrationSettings()
	settings.RunExecution.MaxConcurrentRuns = 10
	settings.UsagePace.BaseMaxConcurrentRuns = 10
	store := &paceSettings{value: settings}
	handler := NewHandler(paceReadings{spans: spans, quota: quota}, store, paceActivity{snapshot: observedActivity(1, now)}, samples)
	handler.now = func() time.Time { return now }
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, APIPath, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", get.Code, get.Body.String())
	}
	var before response
	if err := json.Unmarshal(get.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	tier := before.Tiers[0]
	if tier.Label == "" || tier.Status != "available" || tier.AvailableSlots == nil || *tier.AvailableSlots <= 10 || len(tier.ScopeSlots) != 1 || len(tier.ScopeSlots[0].RunIDs) != 1 || tier.ScopeSlots[0].RunIDs[0] != runID {
		t.Fatalf("measured tier or provenance missing: %+v", tier)
	}
	if len(tier.UnavailableScopes) != 1 || tier.UnavailableScopes[0].Provider != "anthropic" || tier.UnavailableScopes[0].Reason == "" {
		t.Fatalf("unmeasured window not reported beside the measured ceiling: %+v", tier.UnavailableScopes)
	}
	if before.UtilizationHistory.Samples != 11 || before.UtilizationHistory.Complete != 11 || before.UtilizationHistory.ProviderRunIDs["openai"][0] != runID {
		t.Fatalf("durable sample coverage summary missing: %+v", before.UtilizationHistory)
	}
	put := httptest.NewRecorder()
	router.ServeHTTP(put, httptest.NewRequest(http.MethodPut, APIPath, strings.NewReader(`{"selected_tier":"at_reset"}`)))
	if put.Code != http.StatusOK || store.value.UsagePace.SelectedTier != "at_reset" || store.value.RunExecution.MaxConcurrentRuns != *tier.AvailableSlots {
		t.Fatalf("measured tier failed to update admission: status=%d settings=%+v body=%s", put.Code, store.value, put.Body.String())
	}
}

func TestSummarizeHistoryDistinguishesMissingFromCompleteZero(t *testing.T) {
	missing := summarizeHistory(nil)
	if missing.Samples != 0 || len(missing.Unknown) == 0 {
		t.Fatalf("missing history must be explicit, got %+v", missing)
	}
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	zero := summarizeHistory([]utilizationhistory.Sample{{ObservedAt: at, Complete: true, ProviderRunIDs: map[string][]string{}}})
	if zero.Samples != 1 || zero.Complete != 1 || len(zero.Unknown) != 0 {
		t.Fatalf("complete zero activity must remain distinct from missing history: %+v", zero)
	}
	partial := summarizeHistory([]utilizationhistory.Sample{{ObservedAt: at, Complete: false, Unknown: []string{"scope evidence incomplete"}}})
	if partial.Complete != 0 || len(partial.Unknown) != 1 || partial.Unknown[0] != "scope evidence incomplete" {
		t.Fatalf("partial activity must retain the source unknown: %+v", partial)
	}
}

func quotaSpans(now time.Time) []pricing.QuotaReadingSpan {
	minutes := int64(7 * 24 * 60)
	reset := now.Add(3 * 24 * time.Hour)
	firstPercent, lastPercent := 20.0, 40.0
	start := now.Add(-4 * 24 * time.Hour)
	return []pricing.QuotaReadingSpan{
		{Provider: "codex", Pool: "primary", Window: "weekly", WindowMinutes: &minutes, ResetAt: &reset, UsedPercent: &firstPercent, Standing: pricing.QuotaStandingAvailable, FirstAt: start, LastAt: now.Add(-3 * time.Hour), Samples: 1},
		{Provider: "codex", Pool: "primary", Window: "weekly", WindowMinutes: &minutes, ResetAt: &reset, UsedPercent: &lastPercent, Standing: pricing.QuotaStandingAvailable, FirstAt: now.Add(-3 * time.Hour), LastAt: now.Add(-time.Hour), Samples: 1},
	}
}

func observedActivity(runs uint32, at time.Time) runtimeactivity.Snapshot {
	count := runs
	return runtimeactivity.Snapshot{ObservedAt: at, Complete: true, ExecutingRuns: &count, ObservedExecutingRuns: runs}
}
