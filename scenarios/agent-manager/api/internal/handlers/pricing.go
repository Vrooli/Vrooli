// Package handlers provides HTTP handlers for pricing endpoints.
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"

	"github.com/gorilla/mux"
)

// PricingHandler provides HTTP handlers for pricing endpoints.
type PricingHandler struct {
	svc pricing.Service
}

// NewPricingHandler creates a new pricing handler.
func NewPricingHandler(svc pricing.Service) *PricingHandler {
	return &PricingHandler{svc: svc}
}

// RegisterRoutes registers pricing API routes on the given router.
func (h *PricingHandler) RegisterRoutes(r *mux.Router) {
	// Model pricing endpoints
	r.HandleFunc("/api/v1/pricing/models", h.ListModels).Methods("GET")
	r.HandleFunc("/api/v1/pricing/models/{model}/recalculate", h.RecalculateModel).Methods("POST")
	r.HandleFunc("/api/v1/pricing/models/{model}/overrides", h.GetOverrides).Methods("GET")
	r.HandleFunc("/api/v1/pricing/models/{model}/overrides", h.SetOverride).Methods("PUT")
	r.HandleFunc("/api/v1/pricing/models/{model}/overrides/{component}", h.DeleteOverride).Methods("DELETE")

	// Alias management endpoints
	r.HandleFunc("/api/v1/pricing/aliases", h.ListAliases).Methods("GET")
	r.HandleFunc("/api/v1/pricing/aliases", h.CreateAlias).Methods("POST")
	r.HandleFunc("/api/v1/pricing/aliases/{runner_type}/{model}", h.DeleteAlias).Methods("DELETE")

	// Settings endpoints
	r.HandleFunc("/api/v1/pricing/settings", h.GetSettings).Methods("GET")
	r.HandleFunc("/api/v1/pricing/settings", h.UpdateSettings).Methods("PUT")

	// Cache status endpoint
	r.HandleFunc("/api/v1/pricing/cache", h.GetCacheStatus).Methods("GET")

	// Refresh endpoint
	r.HandleFunc("/api/v1/pricing/refresh", h.RefreshAll).Methods("POST")
}

// =============================================================================
// Response Types
// =============================================================================

// ModelPricingListResponse is the HTTP response for listing model pricing.
type ModelPricingListResponse struct {
	Models []*pricing.ModelPricingListItem `json:"models"`
	Total  int                             `json:"total"`
}

// OverridesResponse is the HTTP response for getting model overrides.
type OverridesResponse struct {
	Overrides []*OverrideItem `json:"overrides"`
}

// OverrideItem represents a single override in the response.
type OverrideItem struct {
	Component string     `json:"component"`
	PriceUSD  float64    `json:"priceUsd"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// SetOverrideRequest is the HTTP request for setting an override.
type SetOverrideRequest struct {
	Component string     `json:"component"`
	PriceUSD  float64    `json:"priceUsd"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// AliasesResponse is the HTTP response for listing aliases.
type AliasesResponse struct {
	Aliases []*AliasItem `json:"aliases"`
	Total   int          `json:"total"`
}

// AliasItem represents a single alias in the response.
type AliasItem struct {
	RunnerModel    string    `json:"runnerModel"`
	RunnerType     string    `json:"runnerType"`
	CanonicalModel string    `json:"canonicalModel"`
	Provider       string    `json:"provider"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// CreateAliasRequest is the HTTP request for creating an alias.
type CreateAliasRequest struct {
	RunnerModel    string `json:"runnerModel"`
	RunnerType     string `json:"runnerType"`
	CanonicalModel string `json:"canonicalModel"`
	Provider       string `json:"provider"`
}

// SettingsResponse is the HTTP response for pricing settings.
type SettingsResponse struct {
	HistoricalAverageDays  int `json:"historicalAverageDays"`
	ProviderCacheTTLSeconds int `json:"providerCacheTtlSeconds"`
}

// UpdateSettingsRequest is the HTTP request for updating settings.
type UpdateSettingsRequest struct {
	HistoricalAverageDays  *int `json:"historicalAverageDays,omitempty"`
	ProviderCacheTTLSeconds *int `json:"providerCacheTtlSeconds,omitempty"`
}

// CacheStatusResponse is the HTTP response for cache status.
type CacheStatusResponse struct {
	TotalModels  int                         `json:"totalModels"`
	ExpiredCount int                         `json:"expiredCount"`
	Providers    []ProviderCacheStatusItem   `json:"providers"`
}

// ProviderCacheStatusItem represents a single provider's cache status.
type ProviderCacheStatusItem struct {
	Provider      string    `json:"provider"`
	ModelCount    int       `json:"modelCount"`
	LastFetchedAt time.Time `json:"lastFetchedAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
	IsStale       bool      `json:"isStale"`
}

// =============================================================================
// Handlers
// =============================================================================

// ListModels handles GET /api/v1/pricing/models
func (h *PricingHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.svc.ListModelsWithPricing(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, ModelPricingListResponse{
		Models: models,
		Total:  len(models),
	})
}

// RecalculateModel handles POST /api/v1/pricing/models/{model}/recalculate
func (h *PricingHandler) RecalculateModel(w http.ResponseWriter, r *http.Request) {
	model := decodePathParam(mux.Vars(r)["model"])
	if model == "" {
		writeSimpleError(w, r, "model", "model is required")
		return
	}

	if err := h.svc.RefreshModelPricing(r.Context(), model); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed", "model": model})
}

// GetOverrides handles GET /api/v1/pricing/models/{model}/overrides
func (h *PricingHandler) GetOverrides(w http.ResponseWriter, r *http.Request) {
	model := decodePathParam(mux.Vars(r)["model"])
	if model == "" {
		writeSimpleError(w, r, "model", "model is required")
		return
	}

	overrides, err := h.svc.GetOverrides(r.Context(), model)
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]*OverrideItem, len(overrides))
	for i, o := range overrides {
		items[i] = &OverrideItem{
			Component: string(o.Component),
			PriceUSD:  o.PriceUSD,
			ExpiresAt: o.ExpiresAt,
			CreatedAt: o.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, OverridesResponse{Overrides: items})
}

// SetOverride handles PUT /api/v1/pricing/models/{model}/overrides
func (h *PricingHandler) SetOverride(w http.ResponseWriter, r *http.Request) {
	model := decodePathParam(mux.Vars(r)["model"])
	if model == "" {
		writeSimpleError(w, r, "model", "model is required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req SetOverrideRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Validate component
	component := pricing.PricingComponent(req.Component)
	if !isValidComponent(component) {
		writeSimpleError(w, r, "component", "invalid pricing component")
		return
	}

	override := &pricing.ManualPriceOverride{
		CanonicalModelName: model,
		Component:          component,
		PriceUSD:           req.PriceUSD,
		ExpiresAt:          req.ExpiresAt,
	}

	if err := h.svc.SetOverride(r.Context(), override); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "created"})
}

// DeleteOverride handles DELETE /api/v1/pricing/models/{model}/overrides/{component}
func (h *PricingHandler) DeleteOverride(w http.ResponseWriter, r *http.Request) {
	model := decodePathParam(mux.Vars(r)["model"])
	if model == "" {
		writeSimpleError(w, r, "model", "model is required")
		return
	}

	componentStr := mux.Vars(r)["component"]
	component := pricing.PricingComponent(componentStr)
	if !isValidComponent(component) {
		writeSimpleError(w, r, "component", "invalid pricing component")
		return
	}

	if err := h.svc.DeleteOverride(r.Context(), model, component); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListAliases handles GET /api/v1/pricing/aliases
func (h *PricingHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	runnerType := queryFirst(r, "runner_type", "runnerType")

	aliases, err := h.svc.ListAliases(r.Context(), runnerType)
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]*AliasItem, len(aliases))
	for i, a := range aliases {
		items[i] = &AliasItem{
			RunnerModel:    a.RunnerModel,
			RunnerType:     a.RunnerType,
			CanonicalModel: a.CanonicalModel,
			Provider:       a.Provider,
			CreatedAt:      a.CreatedAt,
			UpdatedAt:      a.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, AliasesResponse{
		Aliases: items,
		Total:   len(items),
	})
}

// CreateAlias handles POST /api/v1/pricing/aliases
func (h *PricingHandler) CreateAlias(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req CreateAliasRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	if req.RunnerModel == "" {
		writeSimpleError(w, r, "runnerModel", "runnerModel is required")
		return
	}
	if req.RunnerType == "" {
		writeSimpleError(w, r, "runnerType", "runnerType is required")
		return
	}
	if req.CanonicalModel == "" {
		writeSimpleError(w, r, "canonicalModel", "canonicalModel is required")
		return
	}

	provider := req.Provider
	if provider == "" {
		provider = "openrouter"
	}

	alias := &pricing.ModelAlias{
		RunnerModel:    req.RunnerModel,
		RunnerType:     req.RunnerType,
		CanonicalModel: req.CanonicalModel,
		Provider:       provider,
	}

	if err := h.svc.UpsertAlias(r.Context(), alias); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// DeleteAlias handles DELETE /api/v1/pricing/aliases/{runner_type}/{model}
func (h *PricingHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	runnerType := mux.Vars(r)["runner_type"]
	model := decodePathParam(mux.Vars(r)["model"])

	if runnerType == "" {
		writeSimpleError(w, r, "runner_type", "runner_type is required")
		return
	}
	if model == "" {
		writeSimpleError(w, r, "model", "model is required")
		return
	}

	// Note: The service doesn't have a DeleteAlias method, but the repository does.
	// For now, return not implemented until the service interface is extended.
	writeError(w, r, domain.NewValidationError("endpoint", "delete alias not yet implemented"))
}

// GetSettings handles GET /api/v1/pricing/settings
func (h *PricingHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	ttlSeconds := int(settings.ProviderCacheTTL.Seconds())
	writeJSON(w, http.StatusOK, SettingsResponse{
		HistoricalAverageDays:  settings.HistoricalAverageDays,
		ProviderCacheTTLSeconds: ttlSeconds,
	})
}

// UpdateSettings handles PUT /api/v1/pricing/settings
func (h *PricingHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req UpdateSettingsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Get current settings
	settings, err := h.svc.GetSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Apply updates
	if req.HistoricalAverageDays != nil {
		if *req.HistoricalAverageDays < 1 || *req.HistoricalAverageDays > 365 {
			writeSimpleError(w, r, "historicalAverageDays", "must be between 1 and 365")
			return
		}
		settings.HistoricalAverageDays = *req.HistoricalAverageDays
	}
	if req.ProviderCacheTTLSeconds != nil {
		if *req.ProviderCacheTTLSeconds < 60 || *req.ProviderCacheTTLSeconds > 86400 {
			writeSimpleError(w, r, "providerCacheTtlSeconds", "must be between 60 and 86400")
			return
		}
		settings.ProviderCacheTTL = time.Duration(*req.ProviderCacheTTLSeconds) * time.Second
	}

	if err := h.svc.UpdateSettings(r.Context(), settings); err != nil {
		writeError(w, r, err)
		return
	}

	ttlSeconds := int(settings.ProviderCacheTTL.Seconds())
	writeJSON(w, http.StatusOK, SettingsResponse{
		HistoricalAverageDays:  settings.HistoricalAverageDays,
		ProviderCacheTTLSeconds: ttlSeconds,
	})
}

// GetCacheStatus handles GET /api/v1/pricing/cache
func (h *PricingHandler) GetCacheStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetCacheStatus(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	providers := make([]ProviderCacheStatusItem, len(status.Providers))
	for i, p := range status.Providers {
		providers[i] = ProviderCacheStatusItem{
			Provider:      p.Provider,
			ModelCount:    p.ModelCount,
			LastFetchedAt: p.LastFetchedAt,
			ExpiresAt:     p.ExpiresAt,
			IsStale:       p.IsStale,
		}
	}

	writeJSON(w, http.StatusOK, CacheStatusResponse{
		TotalModels:  status.TotalModels,
		ExpiredCount: status.ExpiredCount,
		Providers:    providers,
	})
}

// RefreshAll handles POST /api/v1/pricing/refresh
func (h *PricingHandler) RefreshAll(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RefreshPricing(r.Context()); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}

// =============================================================================
// Helpers
// =============================================================================

// decodePathParam URL-decodes a path parameter (for model names with slashes like "anthropic/claude-3-opus").
func decodePathParam(param string) string {
	decoded, err := url.PathUnescape(param)
	if err != nil {
		return param
	}
	return strings.TrimSpace(decoded)
}

// isValidComponent checks if a pricing component is valid.
func isValidComponent(c pricing.PricingComponent) bool {
	switch c {
	case pricing.ComponentInputTokens,
		pricing.ComponentOutputTokens,
		pricing.ComponentCacheRead,
		pricing.ComponentCacheCreation,
		pricing.ComponentWebSearch,
		pricing.ComponentServerToolUse:
		return true
	default:
		return false
	}
}
