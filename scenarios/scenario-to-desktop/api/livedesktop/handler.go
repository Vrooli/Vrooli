package livedesktop

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// Handler holds HTTP handlers for the livedesktop domain.
type Handler struct {
	service *Service
}

// NewHandler creates a new livedesktop handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all livedesktop API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/livedesktop/sessions", h.startSession).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/livedesktop/sessions", h.listSessions).Methods("GET")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}", h.getSession).Methods("GET")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/heartbeat", h.heartbeat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/launch", h.launchApp).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/artifact", h.findArtifact).Methods("GET")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/control", h.controlAction).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/metrics", h.getMetrics).Methods("GET")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/files/{filename}", h.serveFile).Methods("GET")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}", h.stopSession).Methods("DELETE")
	r.HandleFunc("/api/v1/livedesktop/sessions/{id}/ws", h.handleVNCProxy)
}

func extractSessionID(r *http.Request) string {
	return mux.Vars(r)["id"]
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request) {
	var cfg SessionConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	session, err := h.service.StartSession(r.Context(), cfg)
	if err != nil {
		slog.Error("failed to start desktop session", "error", err)
		status := http.StatusInternalServerError
		if session != nil {
			writeJSON(w, status, session.View())
			return
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, session.View())
}

func (h *Handler) listSessions(w http.ResponseWriter, _ *http.Request) {
	sessions := h.service.ListSessions()
	views := make([]SessionView, len(sessions))
	for i, s := range sessions {
		views[i] = s.View()
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.GetSession(extractSessionID(r))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, session.View())
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Heartbeat(extractSessionID(r)); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) launchApp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AppPath string `json:"app_path"`
	}
	// Allow empty body (auto-discover artifact)
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.service.LaunchApp(extractSessionID(r), body.AppPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "launched"})
}

func (h *Handler) findArtifact(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.GetSession(extractSessionID(r))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	artifact, err := h.service.FindArtifact(session.ScenarioName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"artifact_path": artifact})
}

func (h *Handler) stopSession(w http.ResponseWriter, r *http.Request) {
	if err := h.service.StopSession(extractSessionID(r)); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *Handler) controlAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string          `json:"action"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action is required"})
		return
	}

	result, err := h.service.ExecuteAction(r.Context(), extractSessionID(r), body.Action, body.Params)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": errMsg})
		case strings.Contains(errMsg, "unknown action"):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// DOC: docs/reference/live-desktop-api.md#process-metrics
func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.GetSession(extractSessionID(r))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	monitor := session.GetMonitor()
	if monitor == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "no_monitor"})
		return
	}
	writeJSON(w, http.StatusOK, monitor.Report())
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	filename := vars["filename"]

	// Validate filename to prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid filename"})
		return
	}

	filePath := filepath.Join(h.service.dataDir, "sessions", sessionID, filename)
	http.ServeFile(w, r, filePath)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
