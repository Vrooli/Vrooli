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
	"github.com/gorilla/mux"
)

type paceReadings struct {
	spans []pricing.QuotaReadingSpan
	err   error
}

func (p paceReadings) QuotaReadingSpans(context.Context, string, time.Time) ([]pricing.QuotaReadingSpan, error) {
	return p.spans, p.err
}

type paceSettings struct{ value config.OrchestrationSettings }

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

func TestPaceHTTPGetPutAndBaselineReset(t *testing.T) {
	t.Setenv("AGENT_MANAGER_TIMEZONE", "UTC")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	settings := config.DefaultOrchestrationSettings()
	settings.RunExecution.MaxConcurrentRuns = 10
	settings.UsagePace.BaseMaxConcurrentRuns = 10
	store := &paceSettings{value: settings}
	handler := NewHandler(paceReadings{spans: quotaSpans(now)}, store)
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
	if len(before.Tiers) != 4 || before.Tiers[0].AvailableSlots == nil || before.Tiers[3].Status != "unavailable" {
		t.Fatalf("GET tiers=%+v, want three quota forecasts and unavailable extra allowance", before.Tiers)
	}

	put := httptest.NewRecorder()
	router.ServeHTTP(put, httptest.NewRequest(http.MethodPut, APIPath, strings.NewReader(`{"selected_tier":"use_by_wednesday"}`)))
	if put.Code != http.StatusOK {
		t.Fatalf("PUT selection status=%d body=%s", put.Code, put.Body.String())
	}
	if store.value.UsagePace.SelectedTier != TierUseByWednesday || store.value.UsagePace.TargetRunOutAt == "" || store.value.RunExecution.MaxConcurrentRuns == 10 {
		t.Fatalf("selected pace settings=%+v runExecution=%+v", store.value.UsagePace, store.value.RunExecution)
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
	handler := NewHandler(paceReadings{}, store)
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
