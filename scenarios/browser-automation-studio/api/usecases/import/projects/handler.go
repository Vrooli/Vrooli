package projects

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for project import operations.
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
	r.Post("/projects/inspect-folder", h.InspectProjectFolder)
	r.Post("/projects/import", h.ImportProject)
}

// InspectProjectFolder handles POST /projects/inspect-folder
func (h *Handler) InspectProjectFolder(w http.ResponseWriter, r *http.Request) {
	var req InspectProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FolderPath == "" {
		h.respondError(w, http.StatusBadRequest, "folder_path is required")
		return
	}

	result, err := h.service.InspectFolder(r.Context(), req.FolderPath)
	if err != nil {
		h.log.WithError(err).WithField("folder_path", req.FolderPath).Error("Failed to inspect folder")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// ImportProject handles POST /projects/import
func (h *Handler) ImportProject(w http.ResponseWriter, r *http.Request) {
	var req ImportProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FolderPath == "" {
		h.respondError(w, http.StatusBadRequest, "folder_path is required")
		return
	}

	result, err := h.service.Import(r.Context(), &req)
	if err != nil {
		h.log.WithError(err).WithField("folder_path", req.FolderPath).Error("Failed to import project")
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
