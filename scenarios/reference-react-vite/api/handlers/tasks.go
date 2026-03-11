// DOC: docs/reference/api-endpoints.md#tasks
// DOC: docs/internal/SEAMS.md#http-handler-seam
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/tasks"
	"reference-react-vite/api/repository"
)

// TaskHandler handles HTTP requests for the tasks domain.
type TaskHandler struct {
	repo          repository.TaskRepository
	paginationCfg PaginationConfig
}

// NewTaskHandler creates a new task handler with pagination configuration.
// DOC: docs/reference/configuration.md#pagination
func NewTaskHandler(repo repository.TaskRepository, paginationCfg PaginationConfig) *TaskHandler {
	return &TaskHandler{repo: repo, paginationCfg: paginationCfg}
}

// RegisterRoutes registers task routes on the router.
func (h *TaskHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/tasks", h.List).Methods("GET")
	r.HandleFunc("/api/v1/tasks", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/tasks/{id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/tasks/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/tasks/{id}", h.Delete).Methods("DELETE")
}

// List returns tasks matching query parameters.
// GET /api/v1/tasks?project_id=X&status=Y&priority=Z&limit=N&offset=M
// Pagination is configured via PaginationConfig (see docs/reference/configuration.md).
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	// Parse pagination using shared utility
	pagination := ParsePagination(query, h.paginationCfg)

	filter := tasks.ListFilter{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}

	if v := query.Get("project_id"); v != "" {
		filter.ProjectID = &v
	}
	if v := query.Get("status"); v != "" {
		status := tasks.Status(v)
		if err := status.Validate(); err != nil {
			writeValidationError(w, r, "invalid status", map[string]interface{}{"status": v})
			return
		}
		filter.Status = &status
	}
	if v := query.Get("priority"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			writeValidationError(w, r, "invalid priority", map[string]interface{}{"priority": v})
			return
		}
		priority := tasks.Priority(p)
		if err := priority.Validate(); err != nil {
			writeValidationError(w, r, err.Error(), nil)
			return
		}
		filter.Priority = &priority
	}

	items, total, err := h.repo.List(ctx, filter)
	if err != nil {
		writeInternalError(w, r, "failed to list tasks")
		return
	}
	if items == nil {
		items = []*tasks.Task{}
	}

	writeJSON(w, http.StatusOK, newListResponse(items, total, filter.Limit, filter.Offset))
}

// Create adds a new task.
// POST /api/v1/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input tasks.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeBadRequest(w, r, "invalid request body")
		return
	}

	task, err := tasks.NewTask(input)
	if err != nil {
		writeValidationError(w, r, err.Error(), nil)
		return
	}

	if err := h.repo.Create(ctx, task); err != nil {
		writeInternalError(w, r, "failed to create task")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// Get retrieves a single task by ID.
// GET /api/v1/tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	task, err := h.repo.FindByID(ctx, id)
	if err != nil {
		writeInternalError(w, r, "failed to retrieve task")
		return
	}
	if task == nil {
		writeNotFound(w, r, "task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Update modifies an existing task.
// PATCH /api/v1/tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	task, err := h.repo.FindByID(ctx, id)
	if err != nil {
		writeInternalError(w, r, "failed to retrieve task")
		return
	}
	if task == nil {
		writeNotFound(w, r, "task")
		return
	}

	var input tasks.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeBadRequest(w, r, "invalid request body")
		return
	}

	if err := task.ApplyUpdate(input); err != nil {
		writeValidationError(w, r, err.Error(), nil)
		return
	}

	if err := h.repo.Update(ctx, task); err != nil {
		writeInternalError(w, r, "failed to update task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Delete removes a task by ID.
// DELETE /api/v1/tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if err := h.repo.Delete(ctx, id); err != nil {
		if repository.IsNotFound(err) {
			writeNotFound(w, r, "task")
			return
		}
		writeInternalError(w, r, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
