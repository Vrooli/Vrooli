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
	MonthlyLimit        int      `json:"monthly_limit"` // -1 for unlimited
	MonthlyUsed         int      `json:"monthly_used"`
	MonthlyRemaining    int      `json:"monthly_remaining"` // -1 for unlimited
	RequiresWatermark   bool     `json:"requires_watermark"`
	CanUseAI            bool     `json:"can_use_ai"`
	CanUseRecording     bool     `json:"can_use_recording"`
	EntitlementsEnabled bool     `json:"entitlements_enabled"`
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
	ent, err := h.service.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Return default status on error
		ent = &entitlement.Entitlement{
			UserIdentity: userIdentity,
			Status:       entitlement.StatusInactive,
			Tier:         entitlement.TierFree,
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
	if monthlyLimit == -1 {
		monthlyLimit = -1 // Keep as unlimited
	} else {
		monthlyLimit = h.service.GetRemainingExecutions(ctx, userIdentity, 0)
		// Recalculate since GetRemainingExecutions returns remaining, not limit
		monthlyLimit = usedCount + h.service.GetRemainingExecutions(ctx, userIdentity, usedCount)
	}
	if monthlyLimit < 0 {
		monthlyLimit = -1 // Normalize to -1 for unlimited
	}

	remaining := h.service.GetRemainingExecutions(ctx, userIdentity, usedCount)

	response := EntitlementStatusResponse{
		UserIdentity:        userIdentity,
		Status:              string(ent.Status),
		Tier:                string(ent.Tier),
		IsActive:            ent.IsActive(),
		Features:            ent.Features,
		MonthlyLimit:        monthlyLimit,
		MonthlyUsed:         usedCount,
		MonthlyRemaining:    remaining,
		RequiresWatermark:   h.service.RequiresWatermark(ctx, userIdentity),
		CanUseAI:            h.service.CanUseAI(ctx, userIdentity),
		CanUseRecording:     h.service.CanUseRecording(ctx, userIdentity),
		EntitlementsEnabled: true, // We only get here if enabled
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
