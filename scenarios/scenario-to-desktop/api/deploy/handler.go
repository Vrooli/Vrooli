package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	sharedenv "scenario-to-desktop-api/shared/env"
)

// Handler provides HTTP endpoints for deploy target management.
type Handler struct {
	repo *TargetRepository
}

// NewHandler creates a new deploy target handler.
func NewHandler(repo *TargetRepository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers deploy target endpoints on the router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/deploy-targets", h.handleList).Methods("GET")
	r.HandleFunc("/api/v1/deploy-targets/{name}", h.handleGet).Methods("GET")
	r.HandleFunc("/api/v1/deploy-targets/{name}", h.handleSave).Methods("PUT")
	r.HandleFunc("/api/v1/deploy-targets/{name}", h.handleDelete).Methods("DELETE")
	r.HandleFunc("/api/v1/deploy-targets/{name}/test", h.handleTest).Methods("POST")
}

func (h *Handler) handleList(w http.ResponseWriter, _ *http.Request) {
	targets, err := h.repo.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"targets": targets})
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	target, err := h.repo.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, target)
}

func (h *Handler) handleSave(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var target DeployTarget
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if target.ScenarioName == "" {
		writeError(w, http.StatusBadRequest, "scenario_name is required")
		return
	}
	if target.RemoteProfile == "" {
		writeError(w, http.StatusBadRequest, "remote_profile is required")
		return
	}
	if err := h.repo.Save(name, &target); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved", "name": name})
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.repo.Delete(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
}

func (h *Handler) handleTest(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	target, err := h.repo.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		RequireServiceAuth bool `json:"require_service_auth"`
	}
	if r.Body != nil {
		decoderErr := json.NewDecoder(r.Body).Decode(&req)
		if decoderErr != nil && !errors.Is(decoderErr, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+decoderErr.Error())
			return
		}
	}

	// For now, just verify the target exists and has required fields
	if target.ScenarioName == "" || target.RemoteProfile == "" {
		writeError(w, http.StatusBadRequest, "target missing required fields")
		return
	}

	if req.RequireServiceAuth {
		serviceToken := strings.TrimSpace(sharedenv.ResolveSecret("LPBS_SERVICE_SECRET"))
		if serviceToken == "" {
			writeError(w, http.StatusBadRequest, "LPBS_SERVICE_SECRET is not set (checked env and .vrooli/secrets.json)")
			return
		}

		client := NewLPBSClient(target.ScenarioName, serviceToken)
		status, statusErr := client.GetServiceAuthStatus(context.Background())
		if statusErr != nil {
			writeError(w, http.StatusBadGateway, statusErr.Error())
			return
		}
		if status == nil || !status.ServiceAuthConfigured {
			writeError(w, http.StatusBadRequest, "service auth is not configured in LPBS runtime")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":               "ok",
		"name":                 name,
		"scenario_name":        target.ScenarioName,
		"remote_profile":       target.RemoteProfile,
		"service_auth_checked": req.RequireServiceAuth,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
