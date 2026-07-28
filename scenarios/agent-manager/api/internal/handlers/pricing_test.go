package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/pricing"

	"github.com/gorilla/mux"
)

// Embedding the production interface keeps this test fake intentionally
// narrow: only the handler methods under test need behavior.
type pricingHandlerFake struct {
	pricing.Service
	models     []*pricing.ModelPricingListItem
	listErr    error
	refresh    string
	refreshErr error
	settings   *pricing.PricingSettings
	updated    bool
	overrides  []*pricing.ManualPriceOverride
	aliases    []*pricing.ModelAlias
	cache      *pricing.CacheStatus
	refreshed  bool
	deleted    bool
}

func (f *pricingHandlerFake) ListModelsWithPricing(context.Context) ([]*pricing.ModelPricingListItem, error) {
	return f.models, f.listErr
}

func (f *pricingHandlerFake) RefreshModelPricing(_ context.Context, model string) error {
	f.refresh = model
	return f.refreshErr
}

func (f *pricingHandlerFake) GetSettings(context.Context) (*pricing.PricingSettings, error) {
	return f.settings, nil
}

func (f *pricingHandlerFake) UpdateSettings(_ context.Context, settings *pricing.PricingSettings) error {
	f.updated = true
	f.settings = settings
	return nil
}

func (f *pricingHandlerFake) GetOverrides(context.Context, string) ([]*pricing.ManualPriceOverride, error) {
	return f.overrides, nil
}

func (f *pricingHandlerFake) ListAliases(context.Context, string) ([]*pricing.ModelAlias, error) {
	return f.aliases, nil
}

func (f *pricingHandlerFake) GetCacheStatus(context.Context) (*pricing.CacheStatus, error) {
	return f.cache, nil
}

func (f *pricingHandlerFake) RefreshPricing(context.Context) error {
	f.refreshed = true
	return nil
}

func (f *pricingHandlerFake) DeleteAlias(context.Context, string, string) error {
	f.deleted = true
	return nil
}

func TestPricingHandlerListModelsAndRecalculateValidation(t *testing.T) {
	fake := &pricingHandlerFake{models: []*pricing.ModelPricingListItem{}}
	h := NewPricingHandler(fake, mocks.NewFakeStatsRepository())
	rw := httptest.NewRecorder()
	h.ListModels(rw, httptest.NewRequest(http.MethodGet, "/models", nil))
	if rw.Code != http.StatusOK {
		t.Fatalf("list status=%d", rw.Code)
	}
	rw = httptest.NewRecorder()
	h.RecalculateModel(rw, httptest.NewRequest(http.MethodPost, "/", nil))
	if rw.Code != http.StatusBadRequest {
		t.Fatalf("missing model status=%d", rw.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = mux.SetURLVars(req, map[string]string{"model": "anthropic%2Fclaude"})
	rw = httptest.NewRecorder()
	h.RecalculateModel(rw, req)
	if rw.Code != http.StatusOK || fake.refresh != "anthropic/claude" {
		t.Fatalf("recalc status=%d model=%q", rw.Code, fake.refresh)
	}
	fake.listErr = errors.New("down")
	rw = httptest.NewRecorder()
	h.ListModels(rw, httptest.NewRequest(http.MethodGet, "/models", nil))
	if rw.Code < 400 {
		t.Fatalf("list failure status=%d", rw.Code)
	}
}

func TestPricingHandlerUpdateSettingsValidatesAndPersists(t *testing.T) {
	fake := &pricingHandlerFake{settings: &pricing.PricingSettings{HistoricalAverageDays: 30, ProviderCacheTTL: time.Hour}}
	h := NewPricingHandler(fake, mocks.NewFakeStatsRepository())
	for _, body := range []string{`{"historicalAverageDays":0}`, `{"providerCacheTtlSeconds":1}`} {
		rw := httptest.NewRecorder()
		h.UpdateSettings(rw, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(body)))
		if rw.Code != http.StatusBadRequest {
			t.Fatalf("invalid settings %s status=%d", body, rw.Code)
		}
	}
	rw := httptest.NewRecorder()
	h.UpdateSettings(rw, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"historicalAverageDays":31,"providerCacheTtlSeconds":120}`)))
	if rw.Code != http.StatusOK || !fake.updated || fake.settings.HistoricalAverageDays != 31 || fake.settings.ProviderCacheTTL != 120*time.Second {
		t.Fatalf("updated status=%d settings=%+v", rw.Code, fake.settings)
	}
}

func TestPricingHandlerWriteValidationBranches(t *testing.T) {
	fake := &pricingHandlerFake{settings: &pricing.PricingSettings{HistoricalAverageDays: 30, ProviderCacheTTL: time.Hour}}
	h := NewPricingHandler(fake, mocks.NewFakeStatsRepository())
	cases := []struct {
		name string
		h    func(http.ResponseWriter, *http.Request)
		path string
		body string
		vars map[string]string
	}{
		{"set-missing-model", h.SetOverride, "/", `{}`, nil},
		{"set-invalid-json", h.SetOverride, "/", `{`, map[string]string{"model": "m"}},
		{"set-invalid-component", h.SetOverride, "/", `{"component":"bad"}`, map[string]string{"model": "m"}},
		{"delete-missing-model", h.DeleteOverride, "/", ``, nil},
		{"delete-invalid-component", h.DeleteOverride, "/", ``, map[string]string{"model": "m", "component": "bad"}},
		{"alias-invalid-json", h.CreateAlias, "/", `{`, nil},
		{"alias-missing-required", h.CreateAlias, "/", `{}`, nil},
		{"delete-alias-missing-runner", h.DeleteAlias, "/", ``, map[string]string{"model": "m"}},
		{"compare-no-tokens", h.CompareModels, "/", `{}`, nil},
		{"compare-invalid-list", h.CompareModels, "/", `{"inputTokens":1,"modelList":"bad"}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			if tc.vars != nil {
				req = mux.SetURLVars(req, tc.vars)
			}
			rw := httptest.NewRecorder()
			tc.h(rw, req)
			if rw.Code < http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rw.Code, rw.Body.String())
			}
		})
	}
}

func TestPricingHandlerDeleteAliasUsesService(t *testing.T) {
	fake := &pricingHandlerFake{}
	h := NewPricingHandler(fake, mocks.NewFakeStatsRepository())
	req := mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/", nil), map[string]string{"runner_type": "codex", "model": "m"})
	rw := httptest.NewRecorder()
	h.DeleteAlias(rw, req)
	if rw.Code != http.StatusOK || !fake.deleted {
		t.Fatalf("status=%d deleted=%v body=%s", rw.Code, fake.deleted, rw.Body.String())
	}
}

func TestPricingHandlerReadSideEndpointsProjectOperationalData(t *testing.T) {
	now := time.Now().UTC()
	fake := &pricingHandlerFake{
		settings:  &pricing.PricingSettings{HistoricalAverageDays: 14, ProviderCacheTTL: 90 * time.Second},
		overrides: []*pricing.ManualPriceOverride{{Component: pricing.ComponentInputTokens, PriceUSD: 1.5, CreatedAt: now}},
		aliases:   []*pricing.ModelAlias{{RunnerModel: "model", RunnerType: "codex", CanonicalModel: "canonical", Provider: "openrouter", CreatedAt: now, UpdatedAt: now}},
		cache:     &pricing.CacheStatus{TotalModels: 2, ExpiredCount: 1, Providers: []pricing.ProviderCacheStatus{{Provider: "openrouter", ModelCount: 2, LastFetchedAt: now, ExpiresAt: now.Add(time.Hour)}}},
	}
	h := NewPricingHandler(fake, mocks.NewFakeStatsRepository())
	cases := []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
		vars map[string]string
	}{
		{"overrides", h.GetOverrides, map[string]string{"model": "canonical"}}, {"aliases", h.ListAliases, nil}, {"settings", h.GetSettings, nil}, {"cache", h.GetCacheStatus, nil}, {"refresh", h.RefreshAll, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.name == "refresh" {
				req = httptest.NewRequest(http.MethodPost, "/", nil)
			}
			if tc.vars != nil {
				req = mux.SetURLVars(req, tc.vars)
			}
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			if rr.Code != http.StatusOK || rr.Body.Len() == 0 {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
	if !fake.refreshed {
		t.Fatal("refresh was not invoked")
	}
}
