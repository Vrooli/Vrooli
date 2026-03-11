// DOC: docs/reference/api-endpoints.md#projects
// DOC: docs/internal/SEAMS.md#http-handler-seam
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/projects"
	"reference-react-vite/api/repository"
)

// ProjectHandler handles HTTP requests for the projects domain.
type ProjectHandler struct {
	repo          repository.ProjectRepository
	paginationCfg PaginationConfig
}

// NewProjectHandler creates a new project handler with pagination configuration.
// DOC: docs/reference/configuration.md#pagination
func NewProjectHandler(repo repository.ProjectRepository, paginationCfg PaginationConfig) *ProjectHandler {
	return &ProjectHandler{repo: repo, paginationCfg: paginationCfg}
}

// RegisterRoutes registers project routes on the router.
func (h *ProjectHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/projects", h.List).Methods("GET")
	r.HandleFunc("/api/v1/projects", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/projects/{id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/projects/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/projects/{id}", h.Delete).Methods("DELETE")
}

// List returns projects matching query parameters.
// GET /api/v1/projects?status=X&limit=N&offset=M
// Pagination is configured via PaginationConfig (see docs/reference/configuration.md).
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	// Parse pagination using shared utility
	pagination := ParsePagination(query, h.paginationCfg)

	filter := projects.ListFilter{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}

	if v := query.Get("status"); v != "" {
		status := projects.Status(v)
		if err := status.Validate(); err != nil {
			writeValidationError(w, r, "invalid status", map[string]interface{}{"status": v})
			return
		}
		filter.Status = &status
	}

	items, total, err := h.repo.List(ctx, filter)
	if err != nil {
		writeInternalError(w, r, "failed to list projects")
		return
	}
	if items == nil {
		items = []*projects.Project{}
	}

	writeJSON(w, http.StatusOK, newListResponse(items, total, filter.Limit, filter.Offset))
}

// Create adds a new project.
// POST /api/v1/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input projects.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeBadRequest(w, r, "invalid request body")
		return
	}

	project, err := projects.NewProject(input)
	if err != nil {
		writeValidationError(w, r, err.Error(), nil)
		return
	}

	if err := h.repo.Create(ctx, project); err != nil {
		writeInternalError(w, r, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// Get retrieves a single project by ID.
// GET /api/v1/projects/{id}
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	project, err := h.repo.FindByID(ctx, id)
	if err != nil {
		writeInternalError(w, r, "failed to retrieve project")
		return
	}
	if project == nil {
		writeNotFound(w, r, "project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

// Update modifies an existing project.
// PATCH /api/v1/projects/{id}
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	project, err := h.repo.FindByID(ctx, id)
	if err != nil {
		writeInternalError(w, r, "failed to retrieve project")
		return
	}
	if project == nil {
		writeNotFound(w, r, "project")
		return
	}

	var input projects.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeBadRequest(w, r, "invalid request body")
		return
	}

	if err := project.ApplyUpdate(input); err != nil {
		writeValidationError(w, r, err.Error(), nil)
		return
	}

	if err := h.repo.Update(ctx, project); err != nil {
		writeInternalError(w, r, "failed to update project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

// Delete removes a project by ID.
// DELETE /api/v1/projects/{id}
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if err := h.repo.Delete(ctx, id); err != nil {
		if err.Error() == "project not found" {
			writeNotFound(w, r, "project")
			return
		}
		writeInternalError(w, r, "failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
