package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
)

type sessionProfileResponse struct {
	ID              string                           `json:"id"`
	Name            string                           `json:"name"`
	CreatedAt       string                           `json:"created_at"`
	UpdatedAt       string                           `json:"updated_at"`
	LastUsedAt      string                           `json:"last_used_at"`
	HasStorageState bool                             `json:"has_storage_state"`
	BrowserProfile  *archiveingestion.BrowserProfile `json:"browser_profile,omitempty"`
}

func toSessionProfileResponse(p *archiveingestion.SessionProfile) sessionProfileResponse {
	if p == nil {
		return sessionProfileResponse{}
	}
	return sessionProfileResponse{
		ID:              p.ID,
		Name:            p.Name,
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
		LastUsedAt:      p.LastUsedAt.Format(time.RFC3339),
		HasStorageState: len(p.StorageState) > 0,
		BrowserProfile:  p.BrowserProfile,
	}
}

// ListRecordingSessionProfiles returns all persisted session profiles.
func (h *Handler) ListRecordingSessionProfiles(w http.ResponseWriter, _ *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profiles, err := h.sessionProfiles.List()
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	response := make([]sessionProfileResponse, 0, len(profiles))
	for i := range profiles {
		response = append(response, toSessionProfileResponse(&profiles[i]))
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"profiles": response,
	})
}

// CreateRecordingSessionProfile creates a new empty session profile.
func (h *Handler) CreateRecordingSessionProfile(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	profile, err := h.sessionProfiles.Create(req.Name)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusCreated, toSessionProfileResponse(profile))
}

// UpdateRecordingSessionProfile updates an existing profile's name and/or browser profile settings.
func (h *Handler) UpdateRecordingSessionProfile(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	var req struct {
		Name           string                           `json:"name"`
		BrowserProfile *archiveingestion.BrowserProfile `json:"browser_profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// At least one field must be provided
	hasName := strings.TrimSpace(req.Name) != ""
	hasBrowserProfile := req.BrowserProfile != nil

	if !hasName && !hasBrowserProfile {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"error": "At least one of 'name' or 'browser_profile' must be provided",
		}))
		return
	}

	var profile *archiveingestion.SessionProfile
	var err error

	// Update name if provided
	if hasName {
		profile, err = h.sessionProfiles.Rename(profileID, req.Name)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
				return
			}
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": err.Error(),
			}))
			return
		}
	}

	// Update browser profile if provided
	if hasBrowserProfile {
		profile, err = h.sessionProfiles.UpdateBrowserProfile(profileID, req.BrowserProfile)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
				return
			}
			if strings.Contains(err.Error(), "invalid browser profile") {
				h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
					"error": err.Error(),
				}))
				return
			}
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": err.Error(),
			}))
			return
		}
	}

	h.respondSuccess(w, http.StatusOK, toSessionProfileResponse(profile))
}

// DeleteRecordingSessionProfile removes a profile and any in-memory session mapping.
func (h *Handler) DeleteRecordingSessionProfile(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	if err := h.sessionProfiles.Delete(profileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.removeProfileSessions(profileID)

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "deleted",
		"id":     profileID,
	})
}

// removeProfileSessions clears any active session mappings for the given profile.
func (h *Handler) removeProfileSessions(profileID string) {
	if h.sessionProfiles != nil {
		h.sessionProfiles.ClearSessionsForProfile(profileID)
	}
}

// ========================================================================
// Storage State Visibility
// ========================================================================

// storageStateCookie represents a cookie with optional value masking.
type storageStateCookie struct {
	Name        string  `json:"name"`
	Value       string  `json:"value"`
	ValueMasked bool    `json:"valueMasked"`
	Domain      string  `json:"domain"`
	Path        string  `json:"path"`
	Expires     float64 `json:"expires"`
	HttpOnly    bool    `json:"httpOnly"`
	Secure      bool    `json:"secure"`
	SameSite    string  `json:"sameSite"`
}

// storageStateLocalStorageItem represents a localStorage key-value pair.
type storageStateLocalStorageItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// storageStateOrigin represents localStorage for a specific origin.
type storageStateOrigin struct {
	Origin       string                         `json:"origin"`
	LocalStorage []storageStateLocalStorageItem `json:"localStorage"`
}

// storageStateStats provides summary statistics.
type storageStateStats struct {
	CookieCount       int `json:"cookieCount"`
	LocalStorageCount int `json:"localStorageCount"`
	OriginCount       int `json:"originCount"`
}

// storageStateResponse is the API response for GET /recordings/sessions/{profileId}/storage.
type storageStateResponse struct {
	Cookies []storageStateCookie `json:"cookies"`
	Origins []storageStateOrigin `json:"origins"`
	Stats   storageStateStats    `json:"stats"`
}

// playwrightStorageState matches the Playwright storage state format.
type playwrightStorageState struct {
	Cookies []struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expires"`
		HttpOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		SameSite string  `json:"sameSite"`
	} `json:"cookies"`
	Origins []struct {
		Origin       string `json:"origin"`
		LocalStorage []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"localStorage"`
	} `json:"origins"`
}

// GetStorageState returns the storage state (cookies and localStorage) for a session profile.
// HttpOnly cookie values are masked for security.
func (h *Handler) GetStorageState(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	profile, err := h.sessionProfiles.Get(profileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Return empty response if no storage state
	if len(profile.StorageState) == 0 {
		h.respondSuccess(w, http.StatusOK, storageStateResponse{
			Cookies: []storageStateCookie{},
			Origins: []storageStateOrigin{},
			Stats:   storageStateStats{},
		})
		return
	}

	// Parse the Playwright storage state
	var pwState playwrightStorageState
	if err := json.Unmarshal(profile.StorageState, &pwState); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse storage state: " + err.Error(),
		}))
		return
	}

	// Build response with masked httpOnly cookie values
	cookies := make([]storageStateCookie, 0, len(pwState.Cookies))
	for _, c := range pwState.Cookies {
		cookie := storageStateCookie{
			Name:     c.Name,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			HttpOnly: c.HttpOnly,
			Secure:   c.Secure,
			SameSite: c.SameSite,
		}
		if c.HttpOnly {
			cookie.Value = "[HIDDEN]"
			cookie.ValueMasked = true
		} else {
			cookie.Value = c.Value
			cookie.ValueMasked = false
		}
		cookies = append(cookies, cookie)
	}

	// Build origins with localStorage
	origins := make([]storageStateOrigin, 0, len(pwState.Origins))
	localStorageCount := 0
	for _, o := range pwState.Origins {
		items := make([]storageStateLocalStorageItem, 0, len(o.LocalStorage))
		for _, item := range o.LocalStorage {
			items = append(items, storageStateLocalStorageItem{
				Name:  item.Name,
				Value: item.Value,
			})
		}
		localStorageCount += len(items)
		origins = append(origins, storageStateOrigin{
			Origin:       o.Origin,
			LocalStorage: items,
		})
	}

	h.respondSuccess(w, http.StatusOK, storageStateResponse{
		Cookies: cookies,
		Origins: origins,
		Stats: storageStateStats{
			CookieCount:       len(cookies),
			LocalStorageCount: localStorageCount,
			OriginCount:       len(origins),
		},
	})
}

// ========================================================================
// Storage State Modification (Delete Operations)
// ========================================================================

// ClearAllStorage removes all cookies and localStorage from a session profile.
// DELETE /recordings/sessions/{profileId}/storage
func (h *Handler) ClearAllStorage(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	// Save empty storage state
	emptyState := json.RawMessage(`{"cookies":[],"origins":[]}`)
	if _, err := h.sessionProfiles.SaveStorageState(profileID, emptyState); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "cleared",
	})
}

// ClearAllCookies removes all cookies from a session profile.
// DELETE /recordings/sessions/{profileId}/storage/cookies
func (h *Handler) ClearAllCookies(w http.ResponseWriter, r *http.Request) {
	h.modifyStorageState(w, r, func(state *playwrightStorageState) {
		state.Cookies = nil
	})
}

// DeleteCookiesByDomain removes all cookies for a specific domain.
// DELETE /recordings/sessions/{profileId}/storage/cookies/{domain}
func (h *Handler) DeleteCookiesByDomain(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if strings.TrimSpace(domain) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "domain",
		}))
		return
	}

	h.modifyStorageState(w, r, func(state *playwrightStorageState) {
		filtered := make([]struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Expires  float64 `json:"expires"`
			HttpOnly bool    `json:"httpOnly"`
			Secure   bool    `json:"secure"`
			SameSite string  `json:"sameSite"`
		}, 0)
		for _, c := range state.Cookies {
			if c.Domain != domain {
				filtered = append(filtered, c)
			}
		}
		state.Cookies = filtered
	})
}

// DeleteCookie removes a specific cookie by domain and name.
// DELETE /recordings/sessions/{profileId}/storage/cookies/{domain}/{name}
func (h *Handler) DeleteCookie(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	name := chi.URLParam(r, "name")
	if strings.TrimSpace(domain) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "domain",
		}))
		return
	}
	if strings.TrimSpace(name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "name",
		}))
		return
	}

	h.modifyStorageState(w, r, func(state *playwrightStorageState) {
		filtered := make([]struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Expires  float64 `json:"expires"`
			HttpOnly bool    `json:"httpOnly"`
			Secure   bool    `json:"secure"`
			SameSite string  `json:"sameSite"`
		}, 0)
		for _, c := range state.Cookies {
			if !(c.Domain == domain && c.Name == name) {
				filtered = append(filtered, c)
			}
		}
		state.Cookies = filtered
	})
}

// ClearAllLocalStorage removes all localStorage from a session profile.
// DELETE /recordings/sessions/{profileId}/storage/origins
func (h *Handler) ClearAllLocalStorage(w http.ResponseWriter, r *http.Request) {
	h.modifyStorageState(w, r, func(state *playwrightStorageState) {
		state.Origins = nil
	})
}

// DeleteLocalStorageByOrigin removes all localStorage items for a specific origin.
// DELETE /recordings/sessions/{profileId}/storage/origins/{origin}
func (h *Handler) DeleteLocalStorageByOrigin(w http.ResponseWriter, r *http.Request) {
	origin := chi.URLParam(r, "origin")
	if strings.TrimSpace(origin) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "origin",
		}))
		return
	}

	h.modifyStorageState(w, r, func(state *playwrightStorageState) {
		filtered := make([]struct {
			Origin       string `json:"origin"`
			LocalStorage []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"localStorage"`
		}, 0)
		for _, o := range state.Origins {
			if o.Origin != origin {
				filtered = append(filtered, o)
			}
		}
		state.Origins = filtered
	})
}

// DeleteLocalStorageItem removes a specific localStorage item by origin and key.
// DELETE /recordings/sessions/{profileId}/storage/origins/{origin}/{name}
func (h *Handler) DeleteLocalStorageItem(w http.ResponseWriter, r *http.Request) {
	origin := chi.URLParam(r, "origin")
	name := chi.URLParam(r, "name")
	if strings.TrimSpace(origin) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "origin",
		}))
		return
	}
	if strings.TrimSpace(name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "name",
		}))
		return
	}

	h.modifyStorageState(w, r, func(state *playwrightStorageState) {
		for i := range state.Origins {
			if state.Origins[i].Origin == origin {
				filtered := make([]struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				}, 0)
				for _, item := range state.Origins[i].LocalStorage {
					if item.Name != name {
						filtered = append(filtered, item)
					}
				}
				state.Origins[i].LocalStorage = filtered
				// Remove origin if no items left
				if len(filtered) == 0 {
					state.Origins = append(state.Origins[:i], state.Origins[i+1:]...)
				}
				break
			}
		}
	})
}

// modifyStorageState is a helper that loads, modifies, and saves storage state.
func (h *Handler) modifyStorageState(w http.ResponseWriter, r *http.Request, modify func(*playwrightStorageState)) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	profile, err := h.sessionProfiles.Get(profileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Parse existing storage state or start with empty
	var state playwrightStorageState
	if len(profile.StorageState) > 0 {
		if err := json.Unmarshal(profile.StorageState, &state); err != nil {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": "Failed to parse storage state: " + err.Error(),
			}))
			return
		}
	}

	// Apply modification
	modify(&state)

	// Marshal and save
	newState, err := json.Marshal(state)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to serialize storage state: " + err.Error(),
		}))
		return
	}

	if _, err := h.sessionProfiles.SaveStorageState(profileID, newState); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "deleted",
	})
}

// ========================================================================
// Service Worker Management (Live Session)
// ========================================================================

// serviceWorkerResponse is the API response for service worker endpoints.
type serviceWorkerResponse struct {
	SessionID string                    `json:"session_id"`
	Workers   []serviceWorkerInfo       `json:"workers"`
	Control   serviceWorkerControl      `json:"control"`
	Message   string                    `json:"message,omitempty"`
}

// serviceWorkerInfo represents a registered service worker.
type serviceWorkerInfo struct {
	RegistrationID string `json:"registrationId"`
	ScopeURL       string `json:"scopeURL"`
	ScriptURL      string `json:"scriptURL"`
	Status         string `json:"status"`
	VersionID      string `json:"versionId,omitempty"`
}

// serviceWorkerControl represents the service worker control settings.
type serviceWorkerControl struct {
	Mode            string                          `json:"mode"`
	DomainOverrides []serviceWorkerDomainOverride  `json:"domainOverrides,omitempty"`
	BlockedDomains  []string                       `json:"blockedDomains,omitempty"`
}

// serviceWorkerDomainOverride represents per-domain service worker control.
type serviceWorkerDomainOverride struct {
	Domain string `json:"domain"`
	Mode   string `json:"mode"`
}

// GetServiceWorkers returns the service workers for an active session.
// GET /recordings/sessions/{profileId}/service-workers
func (h *Handler) GetServiceWorkers(w http.ResponseWriter, r *http.Request) {
	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	// Look up active session for this profile
	sessionID := h.getSessionForProfile(profileID)
	if sessionID == "" {
		h.respondSuccess(w, http.StatusOK, serviceWorkerResponse{
			SessionID: "",
			Workers:   []serviceWorkerInfo{},
			Control:   serviceWorkerControl{Mode: "allow"},
			Message:   "No active session for this profile",
		})
		return
	}

	// Fetch from playwright-driver
	swResp, err := h.recordModeService.GetServiceWorkers(r.Context(), sessionID)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Convert to response format
	workers := make([]serviceWorkerInfo, len(swResp.Workers))
	for i, w := range swResp.Workers {
		workers[i] = serviceWorkerInfo{
			RegistrationID: w.RegistrationID,
			ScopeURL:       w.ScopeURL,
			ScriptURL:      w.ScriptURL,
			Status:         w.Status,
			VersionID:      w.VersionID,
		}
	}

	overrides := make([]serviceWorkerDomainOverride, len(swResp.Control.DomainOverrides))
	for i, o := range swResp.Control.DomainOverrides {
		overrides[i] = serviceWorkerDomainOverride{
			Domain: o.Domain,
			Mode:   o.Mode,
		}
	}

	h.respondSuccess(w, http.StatusOK, serviceWorkerResponse{
		SessionID: swResp.SessionID,
		Workers:   workers,
		Control: serviceWorkerControl{
			Mode:            swResp.Control.Mode,
			DomainOverrides: overrides,
			BlockedDomains:  swResp.Control.BlockedDomains,
		},
		Message: swResp.Message,
	})
}

// ClearAllServiceWorkers unregisters all service workers for an active session.
// DELETE /recordings/sessions/{profileId}/service-workers
func (h *Handler) ClearAllServiceWorkers(w http.ResponseWriter, r *http.Request) {
	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	// Look up active session for this profile
	sessionID := h.getSessionForProfile(profileID)
	if sessionID == "" {
		h.respondSuccess(w, http.StatusOK, map[string]interface{}{
			"session_id":         "",
			"unregistered_count": 0,
			"message":            "No active session for this profile",
		})
		return
	}

	// Call playwright-driver to unregister all
	resp, err := h.recordModeService.UnregisterAllServiceWorkers(r.Context(), sessionID)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"session_id":         resp.SessionID,
		"unregistered_count": resp.UnregisteredCount,
		"message":            resp.Message,
	})
}

// DeleteServiceWorker unregisters a specific service worker by scope URL.
// DELETE /recordings/sessions/{profileId}/service-workers/{scopeURL}
func (h *Handler) DeleteServiceWorker(w http.ResponseWriter, r *http.Request) {
	profileID := chi.URLParam(r, "profileId")
	scopeURL := chi.URLParam(r, "scopeURL")

	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}
	if strings.TrimSpace(scopeURL) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "scopeURL",
		}))
		return
	}

	// Look up active session for this profile
	sessionID := h.getSessionForProfile(profileID)
	if sessionID == "" {
		h.respondError(w, ErrExecutionNotFound.WithMessage("No active session for this profile"))
		return
	}

	// Call playwright-driver to unregister specific SW
	resp, err := h.recordModeService.UnregisterServiceWorker(r.Context(), sessionID, scopeURL)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	if resp.Error != "" {
		h.respondError(w, ErrExecutionNotFound.WithMessage(resp.Error))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"session_id":   resp.SessionID,
		"unregistered": resp.Unregistered,
	})
}

// getSessionForProfile returns the active playwright session ID for a profile.
func (h *Handler) getSessionForProfile(profileID string) string {
	if h.sessionProfiles == nil {
		return ""
	}
	return h.sessionProfiles.GetSessionForProfile(profileID)
}

// ========================================================================
// Browser History Management
// ========================================================================

// historyResponse is the API response for GET /recordings/sessions/{profileId}/history.
type historyResponse struct {
	Entries  []archiveingestion.HistoryEntry   `json:"entries"`
	Settings *archiveingestion.HistorySettings `json:"settings"`
	Stats    historyStats                      `json:"stats"`
}

type historyStats struct {
	TotalEntries int    `json:"totalEntries"`
	OldestEntry  string `json:"oldestEntry,omitempty"`
	NewestEntry  string `json:"newestEntry,omitempty"`
}

// GetHistory returns the navigation history for a session profile.
// GET /recordings/sessions/{profileId}/history
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	entries, settings, err := h.sessionProfiles.GetHistoryWithPruning(profileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Ensure entries is never nil (nil marshals to null, empty slice marshals to [])
	if entries == nil {
		entries = []archiveingestion.HistoryEntry{}
	}

	// Build stats
	stats := historyStats{TotalEntries: len(entries)}
	if len(entries) > 0 {
		stats.NewestEntry = entries[0].Timestamp
		stats.OldestEntry = entries[len(entries)-1].Timestamp
	}

	h.respondSuccess(w, http.StatusOK, historyResponse{
		Entries:  entries,
		Settings: settings,
		Stats:    stats,
	})
}

// ClearHistory removes all history entries from a session profile.
// DELETE /recordings/sessions/{profileId}/history
func (h *Handler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	if _, err := h.sessionProfiles.ClearHistory(profileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "cleared",
	})
}

// DeleteHistoryEntry removes a specific history entry by ID.
// DELETE /recordings/sessions/{profileId}/history/{entryId}
func (h *Handler) DeleteHistoryEntry(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	entryID := chi.URLParam(r, "entryId")

	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}
	if strings.TrimSpace(entryID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "entryId",
		}))
		return
	}

	if _, err := h.sessionProfiles.DeleteHistoryEntry(profileID, entryID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage(err.Error()))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "deleted",
		"id":     entryID,
	})
}

// UpdateHistorySettings updates the history retention settings.
// PATCH /recordings/sessions/{profileId}/history/settings
func (h *Handler) UpdateHistorySettings(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	var req archiveingestion.HistorySettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	profile, err := h.sessionProfiles.UpdateHistorySettings(profileID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		if strings.Contains(err.Error(), "must be between") {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"error": err.Error(),
			}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Return the updated settings and current history count
	historyCount := len(profile.History)
	h.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"settings":     profile.HistorySettings,
		"historyCount": historyCount,
	})
}

// NavigateToHistoryURL navigates to a URL from history.
// POST /recordings/sessions/{profileId}/history/navigate
func (h *Handler) NavigateToHistoryURL(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	var req struct {
		URL       string `json:"url"`
		WaitUntil string `json:"wait_until,omitempty"`
		TimeoutMs int    `json:"timeout_ms,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "url",
		}))
		return
	}

	// Look up active session for this profile
	sessionID := h.getSessionForProfile(profileID)
	if sessionID == "" {
		h.respondError(w, ErrExecutionNotFound.WithMessage("No active session for this profile"))
		return
	}

	// Navigate directly via driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().Navigate(r.Context(), sessionID, &driver.NavigateRequest{
		URL:       req.URL,
		WaitUntil: req.WaitUntil,
		TimeoutMs: req.TimeoutMs,
	})
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"url":            resp.URL,
		"title":          resp.Title,
		"can_go_back":    resp.CanGoBack,
		"can_go_forward": resp.CanGoForward,
	})
}

// HistoryCallback receives history entries from the playwright driver.
// POST /internal/history-callback
func (h *Handler) HistoryCallback(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	var req struct {
		SessionID      string                        `json:"session_id"`
		Entry          archiveingestion.HistoryEntry `json:"entry"`
		NavigationType string                        `json:"navigation_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if strings.TrimSpace(req.SessionID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "session_id",
		}))
		return
	}

	// Look up profile for this session
	profileID := h.sessionProfiles.GetActiveSession(req.SessionID)
	if profileID == "" {
		// No profile associated with this session - might be execution mode
		h.respondSuccess(w, http.StatusOK, map[string]string{
			"status":  "ignored",
			"message": "No profile associated with session",
		})
		return
	}

	// Add the history entry
	if _, err := h.sessionProfiles.AddHistoryEntry(profileID, req.Entry); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status":     "added",
		"profile_id": profileID,
	})
}

// =============================================================================
// Tab Management Handlers
// =============================================================================

// GetSessionTabs returns the saved tabs for a session profile.
// GET /recordings/sessions/{profileId}/tabs
func (h *Handler) GetSessionTabs(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	profile, err := h.sessionProfiles.Get(profileID)
	if err != nil {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
		return
	}

	// Convert TabState to TabInfo for API response
	tabs := make([]TabInfo, 0, len(profile.OpenTabs))
	for _, tab := range profile.OpenTabs {
		tabs = append(tabs, TabInfo{
			URL:      tab.URL,
			Title:    tab.Title,
			IsActive: tab.IsActive,
			Order:    tab.Order,
		})
	}

	h.respondSuccess(w, http.StatusOK, TabsResponse{Tabs: tabs})
}

// ClearSessionTabs removes all saved tabs from a session profile.
// DELETE /recordings/sessions/{profileId}/tabs
func (h *Handler) ClearSessionTabs(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	// Save empty tabs array to clear all
	if _, err := h.sessionProfiles.SaveOpenTabs(profileID, nil); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status":     "cleared",
		"profile_id": profileID,
	})
}

// DeleteSessionTab removes a single tab by its order from a session profile.
// DELETE /recordings/sessions/{profileId}/tabs/{order}
func (h *Handler) DeleteSessionTab(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	orderStr := chi.URLParam(r, "order")
	order, err := strconv.Atoi(orderStr)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid order parameter: must be an integer",
		}))
		return
	}

	profile, err := h.sessionProfiles.Get(profileID)
	if err != nil {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
		return
	}

	// Find and remove the tab with the specified order
	var found bool
	newTabs := make([]archiveingestion.TabState, 0, len(profile.OpenTabs))
	for _, tab := range profile.OpenTabs {
		if tab.Order == order {
			found = true
			continue // Skip this tab
		}
		newTabs = append(newTabs, tab)
	}

	if !found {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Tab not found"))
		return
	}

	// Renumber remaining tabs to maintain sequential order
	for i := range newTabs {
		newTabs[i].Order = i
	}

	// Save the updated tabs
	if _, err := h.sessionProfiles.SaveOpenTabs(profileID, newTabs); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status":     "deleted",
		"profile_id": profileID,
	})
}
