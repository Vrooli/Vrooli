package scan

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for unified scans.
type Handler struct {
	service *Service
	log     *logrus.Logger
}

// NewHandler creates a new Handler.
func NewHandler(service *Service, log *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

// RegisterRoutes registers scan routes on a router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/fs/scan", h.Scan)
}

// Scan handles POST /fs/scan.
func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Scan(r.Context(), &req)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"mode":      req.Mode,
			"path":      req.Path,
			"projectId": req.ProjectID,
		}).Error("Failed to scan filesystem")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// respondJSON writes a JSON response.
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.WithError(err).Error("Failed to encode response")
	}
}

// respondError writes an error response.
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}
