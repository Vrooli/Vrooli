package codesigning

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// Handler exposes the compatibility configuration proxy. Signing validation,
// toolchain discovery, and certificate operations are owned by
// scenario-to-desktop and are intentionally not implemented here.
type Handler struct {
	repo Repository
	log  func(string, map[string]interface{})
}

func NewHandler(repo Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) addDeprecationHeaders(w http.ResponseWriter) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", "2026-06-01")
	w.Header().Set("X-Deprecation-Notice", "Use scenario-to-desktop signing API instead. See: /api/v1/signing/{scenario}")
}

func (h *Handler) GetSigning(w http.ResponseWriter, r *http.Request) {
	h.addDeprecationHeaders(w)
	profileID := mux.Vars(r)["id"]
	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	config, err := h.repo.Get(r.Context(), profileID)
	if errors.Is(err, ErrProfileNotFound) {
		h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
		return
	}
	if err != nil {
		h.logError("failed to get signing config", err)
		h.writeError(w, http.StatusInternalServerError, "failed to retrieve signing config")
		return
	}
	if config == nil {
		config = DefaultSigningConfig()
	}
	h.writeJSON(w, http.StatusOK, config)
}

func (h *Handler) SetSigning(w http.ResponseWriter, r *http.Request) {
	h.addDeprecationHeaders(w)
	profileID := mux.Vars(r)["id"]
	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	var config SigningConfig
	if err := json.Unmarshal(body, &config); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.repo.Save(r.Context(), profileID, &config); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.logError("failed to save signing config", err)
		h.writeError(w, http.StatusInternalServerError, "failed to save signing config")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"status": "updated", "config": config, "message": "Signing configuration saved successfully"})
}

func (h *Handler) SetPlatformSigning(w http.ResponseWriter, r *http.Request) {
	h.addDeprecationHeaders(w)
	vars := mux.Vars(r)
	profileID, platform := vars["id"], vars["platform"]
	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	if !IsValidPlatform(platform) {
		h.writeError(w, http.StatusBadRequest, "invalid platform: "+platform+". Valid: windows, macos, linux")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	var platformConfig interface{}
	switch platform {
	case PlatformWindows:
		var value WindowsSigningConfig
		if err := json.Unmarshal(body, &value); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid Windows config: "+err.Error())
			return
		}
		platformConfig = &value
	case PlatformMacOS:
		var value MacOSSigningConfig
		if err := json.Unmarshal(body, &value); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid macOS config: "+err.Error())
			return
		}
		platformConfig = &value
	case PlatformLinux:
		var value LinuxSigningConfig
		if err := json.Unmarshal(body, &value); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid Linux config: "+err.Error())
			return
		}
		platformConfig = &value
	}
	if err := h.repo.SaveForPlatform(r.Context(), profileID, platform, platformConfig); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.logError("failed to save platform signing config", err)
		h.writeError(w, http.StatusInternalServerError, "failed to save signing config")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"status": "updated", "platform": platform, "config": platformConfig, "message": fmt.Sprintf("%s signing configuration saved successfully", platform)})
}

func (h *Handler) DeleteSigning(w http.ResponseWriter, r *http.Request) {
	h.addDeprecationHeaders(w)
	profileID := mux.Vars(r)["id"]
	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	if err := h.repo.Delete(r.Context(), profileID); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.logError("failed to delete signing config", err)
		h.writeError(w, http.StatusInternalServerError, "failed to delete signing config")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "message": "Signing configuration removed"})
}

func (h *Handler) DeletePlatformSigning(w http.ResponseWriter, r *http.Request) {
	h.addDeprecationHeaders(w)
	vars := mux.Vars(r)
	profileID, platform := vars["id"], vars["platform"]
	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	if !IsValidPlatform(platform) {
		h.writeError(w, http.StatusBadRequest, "invalid platform: "+platform)
		return
	}
	if err := h.repo.DeleteForPlatform(r.Context(), profileID, platform); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.logError("failed to delete platform signing config", err)
		h.writeError(w, http.StatusInternalServerError, "failed to delete signing config")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "platform": platform, "message": fmt.Sprintf("%s signing configuration removed", platform)})
}

func (h *Handler) logError(message string, err error) {
	if h.log != nil {
		h.log("error", map[string]interface{}{"msg": message, "error": err.Error()})
	}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
