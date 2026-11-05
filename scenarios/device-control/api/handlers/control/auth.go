package control

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	authdomain "device-control/internal/auth"

	"github.com/gorilla/mux"
)

type authProfileRequest struct {
	Profile authdomain.Profile `json:"profile"`
	Actor   string             `json:"actor"`
}

type unlockRequest struct {
	ProfileID  string `json:"profile_id"`
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token"`
}

func (h *handler) listAuthProfiles(w http.ResponseWriter, r *http.Request) {
	write(w, http.StatusOK, map[string]any{"profiles": h.service.AuthProfiles(r.Context())})
}

func (h *handler) createAuthProfile(w http.ResponseWriter, r *http.Request) {
	var in authProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile JSON is invalid")
		return
	}
	profile, err := h.service.CreateAuthProfile(r.Context(), in.Profile, in.Actor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	write(w, http.StatusCreated, map[string]any{"profile": profile})
}

func (h *handler) getAuthProfile(w http.ResponseWriter, r *http.Request) {
	profile, provider, err := h.service.AuthProfileStatus(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"profile": profile, "provider": provider})
}

func (h *handler) updateAuthProfile(w http.ResponseWriter, r *http.Request) {
	var in authProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "profile JSON is invalid")
		return
	}
	profile, err := h.service.UpdateAuthProfile(r.Context(), mux.Vars(r)["id"], in.Profile, in.Actor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *handler) revokeAuthProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.RevokeAuthProfile(r.Context(), mux.Vars(r)["id"], r.Header.Get("X-Vrooli-Actor"))
	if err != nil {
		writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *handler) provisionAuthCredential(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.ProvisionAuthCredential(r.Context(), mux.Vars(r)["id"], io.LimitReader(r.Body, 4097))
	if err != nil {
		writeError(w, http.StatusConflict, "credential_provision_failed", safeCredentialError(err))
		return
	}
	write(w, http.StatusOK, map[string]any{"provider": status, "provisioned": true})
}

func (h *handler) deleteAuthCredential(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.DeleteAuthCredential(r.Context(), mux.Vars(r)["id"], r.Header.Get("X-Vrooli-Actor"))
	if err != nil {
		writeError(w, http.StatusConflict, "credential_delete_failed", safeCredentialError(err))
		return
	}
	write(w, http.StatusOK, map[string]any{"provider": status, "deleted": true})
}

func (h *handler) testAuthProfile(w http.ResponseWriter, r *http.Request) {
	profile, provider, err := h.service.AuthProfileStatus(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "profile_not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"profile": profile, "provider": provider, "outcome": providerOutcome(provider)})
}

func (h *handler) unlockDevice(w http.ResponseWriter, r *http.Request) {
	var in unlockRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "unlock JSON is invalid")
		return
	}
	result, err := h.service.UnlockDevice(r.Context(), in.ProfileID, in.DeviceID, in.Actor, in.LeaseToken)
	if err != nil {
		write(w, http.StatusConflict, map[string]any{
			"status":  "failed",
			"code":    unlockErrorCode(err),
			"message": safeUnlockError(err),
			"result":  result,
		})
		return
	}
	write(w, http.StatusOK, result)
}

func safeCredentialError(err error) string {
	if err == nil {
		return "credential provider request failed"
	}
	// Provider implementations are outside this HTTP boundary. Do not copy
	// arbitrary provider errors into an API response: they may contain a
	// resolved value, backend path, or command transcript.
	return "credential provider request failed; inspect provider status"
}

func providerOutcome(status authdomain.ProviderStatus) string {
	if status.ProviderState != "available" {
		return "credential_provider_" + status.ProviderState
	}
	if !status.Configured {
		return authdomain.OutcomeUnconfigured
	}
	return "configured"
}

func unlockErrorCode(err error) string {
	if strings.Contains(err.Error(), "lease") {
		return "lease_required"
	}
	return "unlock_failed"
}

func safeUnlockError(err error) string {
	if strings.Contains(err.Error(), "lease") {
		return "unlock requires an active device lease"
	}
	return "device unlock transaction failed; inspect the typed result and next action"
}
