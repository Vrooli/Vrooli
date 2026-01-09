package assets

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MaxUploadSize is the maximum file size for asset uploads (10MB).
const MaxUploadSize = 10 * 1024 * 1024

// Handler handles HTTP requests for asset import operations.
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
	r.Post("/projects/assets", h.UploadAsset)
	r.Post("/projects/{projectID}/assets/list", h.ListAssets)
	r.Post("/projects/{projectID}/assets/delete", h.DeleteAsset)
}

// UploadAsset handles POST /projects/assets
// Accepts multipart/form-data with:
// - file: the asset file
// - path: relative path within assets directory
// - project_id: the project ID
func (h *Handler) UploadAsset(w http.ResponseWriter, r *http.Request) {
	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		h.respondError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	// Get project ID
	projectIDStr := r.FormValue("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project_id")
		return
	}

	// Get path
	path := r.FormValue("path")
	if path == "" {
		h.respondError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Upload asset
	result, err := h.service.Upload(r.Context(), projectID, path, file, header.Size)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"path":       path,
		}).Error("Failed to upload asset")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, result)
}

// ListAssets handles POST /projects/{projectID}/assets/list
func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var req ListAssetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = ListAssetsRequest{}
	}

	result, err := h.service.List(r.Context(), projectID, req.Path)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"path":       req.Path,
		}).Error("Failed to list assets")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// DeleteAsset handles POST /projects/{projectID}/assets/delete
func (h *Handler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var req DeleteAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" {
		h.respondError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := h.service.Delete(r.Context(), projectID, req.Path); err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"path":       req.Path,
		}).Error("Failed to delete asset")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "deleted",
		"path":   req.Path,
	})
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
