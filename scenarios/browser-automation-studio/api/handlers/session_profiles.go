package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
)

type sessionProfileResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	LastUsedAt      string `json:"last_used_at"`
	HasStorageState bool   `json:"has_storage_state"`
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

// UpdateRecordingSessionProfile renames an existing profile.
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
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "name",
		}))
		return
	}

	profile, err := h.sessionProfiles.Rename(profileID, req.Name)
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
