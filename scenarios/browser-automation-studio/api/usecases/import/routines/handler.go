package routines

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for routine import operations.
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

// RegisterRoutes registers the handler's routes on a chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/projects/{projectID}/routines/inspect", h.InspectRoutine)
	r.Post("/projects/{projectID}/routines/import", h.ImportRoutine)
}

// InspectRoutine handles POST /projects/{projectID}/routines/inspect
func (h *Handler) InspectRoutine(w http.ResponseWriter, r *http.Request) {
	// Parse project ID
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	// Parse request body
	var req InspectRoutineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.FilePath == "" {
		h.respondError(w, http.StatusBadRequest, "file_path is required")
		return
	}

	// Call service
	result, err := h.service.InspectFile(r.Context(), projectID, req.FilePath)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"file_path":  req.FilePath,
		}).Error("Failed to inspect routine")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// ImportRoutine handles POST /projects/{projectID}/routines/import
func (h *Handler) ImportRoutine(w http.ResponseWriter, r *http.Request) {
	// Parse project ID
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	// Parse request body
	var req ImportRoutineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.FilePath == "" {
		h.respondError(w, http.StatusBadRequest, "file_path is required")
		return
	}

	// Call service
	result, err := h.service.Import(r.Context(), projectID, &req)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"file_path":  req.FilePath,
		}).Error("Failed to import routine")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, result)
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
