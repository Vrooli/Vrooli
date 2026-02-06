package deploy

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
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
	// For now, just verify the target exists and has required fields
	if target.ScenarioName == "" || target.RemoteProfile == "" {
		writeError(w, http.StatusBadRequest, "target missing required fields")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"name":           name,
		"scenario_name":  target.ScenarioName,
		"remote_profile": target.RemoteProfile,
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
