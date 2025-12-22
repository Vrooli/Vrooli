package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// EntitlementHandler provides HTTP handlers for entitlement operations.
type EntitlementHandler struct {
	service      *entitlement.Service
	usageTracker *entitlement.UsageTracker
	settingsRepo UserSettingsRepository
}

// UserSettingsRepository defines the interface for user settings storage.
type UserSettingsRepository interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

// NewEntitlementHandler creates a new entitlement handler.
func NewEntitlementHandler(
	service *entitlement.Service,
	usageTracker *entitlement.UsageTracker,
	settingsRepo UserSettingsRepository,
) *EntitlementHandler {
	return &EntitlementHandler{
		service:      service,
		usageTracker: usageTracker,
		settingsRepo: settingsRepo,
	}
}

// EntitlementStatusResponse represents the entitlement status response.
type EntitlementStatusResponse struct {
	UserIdentity        string   `json:"user_identity"`
	Status              string   `json:"status"`
	Tier                string   `json:"tier"`
	IsActive            bool     `json:"is_active"`
	Features            []string `json:"features,omitempty"`
	FeatureAccess       []FeatureAccessSummary `json:"feature_access,omitempty"`
	MonthlyLimit        int      `json:"monthly_limit"` // -1 for unlimited
	MonthlyUsed         int      `json:"monthly_used"`
	MonthlyRemaining    int      `json:"monthly_remaining"` // -1 for unlimited
	RequiresWatermark   bool     `json:"requires_watermark"`
	CanUseAI            bool     `json:"can_use_ai"`
	CanUseRecording     bool     `json:"can_use_recording"`
	EntitlementsEnabled bool     `json:"entitlements_enabled"`
	OverrideTier        string   `json:"override_tier,omitempty"`
}

// FeatureAccessSummary describes a subscription feature and access state.
type FeatureAccessSummary struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	RequiredTier string `json:"required_tier,omitempty"`
	HasAccess    bool   `json:"has_access"`
}

// GetEntitlementStatus handles GET /api/v1/entitlement/status
// Returns the current user's entitlement status and usage.
func (h *EntitlementHandler) GetEntitlementStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get user identity from context (set by middleware) or query param
	userIdentity := entitlement.UserIdentityFromContext(r.Context())
	if userIdentity == "" {
		userIdentity = strings.TrimSpace(r.URL.Query().Get("user"))
	}

	// If still no user identity, try to load from settings
	if userIdentity == "" && h.settingsRepo != nil {
		if saved, err := h.settingsRepo.GetSetting(ctx, "user_identity"); err == nil {
			userIdentity = saved
		}
	}

	// Get entitlement
	overrideTier := h.getOverrideTier(ctx)
	overrideActive := overrideTier != ""

	var ent *entitlement.Entitlement
	var err error
	if overrideActive {
		ent = h.service.BuildOverrideEntitlement(userIdentity, overrideTier)
	} else {
		ent, err = h.service.GetEntitlement(ctx, userIdentity)
		if err != nil {
			// Return default status on error
			ent = &entitlement.Entitlement{
				UserIdentity: userIdentity,
				Status:       entitlement.StatusInactive,
				Tier:         entitlement.TierFree,
			}
		}
	}

	// Get usage count
	usedCount := 0
	if h.usageTracker != nil && userIdentity != "" {
		if count, err := h.usageTracker.GetMonthlyExecutionCount(ctx, userIdentity); err == nil {
			usedCount = count
		}
	}

	// Calculate limits
	monthlyLimit := h.service.GetRemainingExecutions(ctx, userIdentity, 0)
	if overrideActive {
		monthlyLimit = h.service.TierLimit(ent.Tier)
	}
	if monthlyLimit == -1 {
		monthlyLimit = -1 // Keep as unlimited
	} else if !overrideActive {
		monthlyLimit = h.service.GetRemainingExecutions(ctx, userIdentity, 0)
		// Recalculate since GetRemainingExecutions returns remaining, not limit
		monthlyLimit = usedCount + h.service.GetRemainingExecutions(ctx, userIdentity, usedCount)
	}
	if monthlyLimit < 0 {
		monthlyLimit = -1 // Normalize to -1 for unlimited
	}

	remaining := h.service.GetRemainingExecutions(ctx, userIdentity, usedCount)
	if overrideActive {
		if monthlyLimit == -1 {
			remaining = -1
		} else {
			remaining = monthlyLimit - usedCount
			if remaining < 0 {
				remaining = 0
			}
		}
	}

	response := EntitlementStatusResponse{
		UserIdentity:        userIdentity,
		Status:              string(ent.Status),
		Tier:                string(ent.Tier),
		IsActive:            ent.IsActive(),
		Features:            ent.Features,
		FeatureAccess:       h.buildFeatureAccessSummary(ent),
		MonthlyLimit:        monthlyLimit,
		MonthlyUsed:         usedCount,
		MonthlyRemaining:    remaining,
		RequiresWatermark:   h.resolveRequiresWatermark(ctx, userIdentity, ent, overrideActive),
		CanUseAI:            h.resolveCanUseAI(ctx, userIdentity, ent, overrideActive),
		CanUseRecording:     h.resolveCanUseRecording(ctx, userIdentity, ent, overrideActive),
		EntitlementsEnabled: h.service.IsEnabled() || overrideActive,
	}
	if overrideActive {
		response.OverrideTier = string(overrideTier)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SetUserIdentityRequest represents the request to set user identity.
type SetUserIdentityRequest struct {
	Email string `json:"email"`
}

// SetUserIdentity handles POST /api/v1/entitlement/identity
// Sets the user's email for entitlement verification.
func (h *EntitlementHandler) SetUserIdentity(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	var req SetUserIdentityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"INVALID_REQUEST","message":"Invalid JSON body"}}`, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	// Validate email format (basic check)
	if email != "" && !strings.Contains(email, "@") {
		http.Error(w, `{"error":{"code":"INVALID_EMAIL","message":"Invalid email format"}}`, http.StatusBadRequest)
		return
	}

	// Save to settings
	if h.settingsRepo != nil {
		if err := h.settingsRepo.SetSetting(ctx, "user_identity", email); err != nil {
			http.Error(w, `{"error":{"code":"SAVE_FAILED","message":"Failed to save user identity"}}`, http.StatusInternalServerError)
			return
		}
	}

	// Invalidate cache to force refresh
	if email != "" {
		h.service.InvalidateCache(email)
	}

	// Fetch and return new entitlement status
	h.GetEntitlementStatus(w, r.WithContext(entitlement.WithUserIdentity(ctx, email)))
}

// GetUserIdentity handles GET /api/v1/entitlement/identity
// Returns the stored user identity.
func (h *EntitlementHandler) GetUserIdentity(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	email := ""
	if h.settingsRepo != nil {
		if saved, err := h.settingsRepo.GetSetting(ctx, "user_identity"); err == nil {
			email = saved
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"email": email,
	})
}

// ClearUserIdentity handles DELETE /api/v1/entitlement/identity
// Clears the stored user identity (logout).
func (h *EntitlementHandler) ClearUserIdentity(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if h.settingsRepo != nil {
		if err := h.settingsRepo.SetSetting(ctx, "user_identity", ""); err != nil {
			http.Error(w, `{"error":{"code":"CLEAR_FAILED","message":"Failed to clear user identity"}}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"cleared"}`))
}

// GetUsageSummary handles GET /api/v1/entitlement/usage
// Returns detailed usage statistics.
func (h *EntitlementHandler) GetUsageSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	userIdentity := entitlement.UserIdentityFromContext(r.Context())
	if userIdentity == "" {
		userIdentity = strings.TrimSpace(r.URL.Query().Get("user"))
	}

	if h.usageTracker == nil {
		http.Error(w, `{"error":{"code":"USAGE_TRACKING_DISABLED","message":"Usage tracking is not available"}}`, http.StatusServiceUnavailable)
		return
	}

	summary, err := h.usageTracker.GetUsageSummary(ctx, userIdentity)
	if err != nil {
		http.Error(w, `{"error":{"code":"USAGE_ERROR","message":"Failed to get usage summary"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetEntitlementOverride handles GET /api/v1/entitlement/override
func (h *EntitlementHandler) GetEntitlementOverride(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	overrideTier := h.getOverrideTier(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"tier": string(overrideTier),
	})
}

// SetEntitlementOverride handles POST /api/v1/entitlement/override
func (h *EntitlementHandler) SetEntitlementOverride(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if h.settingsRepo == nil {
		http.Error(w, `{"error":{"code":"SETTINGS_UNAVAILABLE","message":"Settings storage unavailable"}}`, http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"INVALID_REQUEST","message":"Invalid JSON body"}}`, http.StatusBadRequest)
		return
	}

	trimmedTier := strings.TrimSpace(strings.ToLower(req.Tier))
	if trimmedTier == "" {
		if err := h.settingsRepo.SetSetting(ctx, entitlement.OverrideTierSettingKey, ""); err != nil {
			http.Error(w, `{"error":{"code":"SAVE_FAILED","message":"Failed to clear override tier"}}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tier, ok := entitlement.ParseTier(trimmedTier)
	if !ok {
		http.Error(w, `{"error":{"code":"INVALID_TIER","message":"Unknown subscription tier"}}`, http.StatusBadRequest)
		return
	}

	if err := h.settingsRepo.SetSetting(ctx, entitlement.OverrideTierSettingKey, string(tier)); err != nil {
		http.Error(w, `{"error":{"code":"SAVE_FAILED","message":"Failed to save override tier"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"tier": string(tier),
	})
}

// ClearEntitlementOverride handles DELETE /api/v1/entitlement/override
func (h *EntitlementHandler) ClearEntitlementOverride(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if h.settingsRepo == nil {
		http.Error(w, `{"error":{"code":"SETTINGS_UNAVAILABLE","message":"Settings storage unavailable"}}`, http.StatusServiceUnavailable)
		return
	}

	if err := h.settingsRepo.SetSetting(ctx, entitlement.OverrideTierSettingKey, ""); err != nil {
		http.Error(w, `{"error":{"code":"SAVE_FAILED","message":"Failed to clear override tier"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EntitlementHandler) getOverrideTier(ctx context.Context) entitlement.Tier {
	if h.settingsRepo == nil {
		return ""
	}
	value, err := h.settingsRepo.GetSetting(ctx, entitlement.OverrideTierSettingKey)
	if err != nil || value == "" {
		return ""
	}
	tier, ok := entitlement.ParseTier(value)
	if !ok {
		return ""
	}
	return tier
}

func (h *EntitlementHandler) resolveRequiresWatermark(ctx context.Context, userIdentity string, ent *entitlement.Entitlement, overrideActive bool) bool {
	if overrideActive {
		return h.service.TierRequiresWatermark(ent.Tier)
	}
	return h.service.RequiresWatermark(ctx, userIdentity)
}

func (h *EntitlementHandler) resolveCanUseAI(ctx context.Context, userIdentity string, ent *entitlement.Entitlement, overrideActive bool) bool {
	if overrideActive {
		return h.service.TierCanUseAI(ent.Tier)
	}
	return h.service.CanUseAI(ctx, userIdentity)
}

func (h *EntitlementHandler) resolveCanUseRecording(ctx context.Context, userIdentity string, ent *entitlement.Entitlement, overrideActive bool) bool {
	if overrideActive {
		return h.service.TierCanUseRecording(ent.Tier)
	}
	return h.service.CanUseRecording(ctx, userIdentity)
}

func (h *EntitlementHandler) buildFeatureAccessSummary(ent *entitlement.Entitlement) []FeatureAccessSummary {
	canUseAI := h.service.TierCanUseAI(ent.Tier)
	canUseRecording := h.service.TierCanUseRecording(ent.Tier)
	requiresWatermark := h.service.TierRequiresWatermark(ent.Tier)

	return []FeatureAccessSummary{
		{
			ID:           "ai",
			Label:        "AI-Powered Features",
			Description:  "Use AI to generate and edit workflows automatically",
			RequiredTier: string(h.service.MinTierForAI()),
			HasAccess:    canUseAI,
		},
		{
			ID:           "recording",
			Label:        "Live Recording",
			Description:  "Record browser interactions to create workflows",
			RequiredTier: string(h.service.MinTierForRecording()),
			HasAccess:    canUseRecording,
		},
		{
			ID:           "watermark-free",
			Label:        "Watermark-Free Exports",
			Description:  "Export videos and replays without watermarks",
			RequiredTier: string(h.service.MinTierWithoutWatermark()),
			HasAccess:    !requiresWatermark,
		},
	}
}

// RefreshEntitlement handles POST /api/v1/entitlement/refresh
// Forces a refresh of the cached entitlement.
func (h *EntitlementHandler) RefreshEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	userIdentity := entitlement.UserIdentityFromContext(r.Context())
	if userIdentity == "" {
		userIdentity = strings.TrimSpace(r.URL.Query().Get("user"))
	}

	if userIdentity == "" {
		http.Error(w, `{"error":{"code":"USER_REQUIRED","message":"User identity is required"}}`, http.StatusBadRequest)
		return
	}

	// Invalidate cache
	h.service.InvalidateCache(userIdentity)

	// Fetch fresh entitlement
	h.GetEntitlementStatus(w, r.WithContext(entitlement.WithUserIdentity(ctx, userIdentity)))
}
