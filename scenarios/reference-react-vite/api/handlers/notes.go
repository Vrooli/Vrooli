// DOC: docs/reference/api-endpoints.md#notes
// DOC: docs/internal/SEAMS.md#http-handler-seam
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/notes"
	"reference-react-vite/api/repository"
)

// NoteHandler handles HTTP requests for the notes domain.
type NoteHandler struct {
	repo          repository.NoteRepository
	taskRepo      repository.TaskRepository
	paginationCfg PaginationConfig
}

// NewNoteHandler creates a new note handler with pagination configuration.
// DOC: docs/reference/configuration.md#pagination
func NewNoteHandler(noteRepo repository.NoteRepository, taskRepo repository.TaskRepository, paginationCfg PaginationConfig) *NoteHandler {
	return &NoteHandler{repo: noteRepo, taskRepo: taskRepo, paginationCfg: paginationCfg}
}

// RegisterRoutes registers note routes on the router.
// Notes are nested under tasks as they belong to a specific task.
func (h *NoteHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/tasks/{task_id}/notes", h.List).Methods("GET")
	r.HandleFunc("/api/v1/tasks/{task_id}/notes", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/notes/{id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/notes/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/notes/{id}", h.Delete).Methods("DELETE")
}

// List returns notes for a specific task.
// GET /api/v1/tasks/{task_id}/notes?limit=N&offset=M
// Pagination is configured via PaginationConfig (see docs/reference/configuration.md).
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := mux.Vars(r)["task_id"]
	query := r.URL.Query()

	// Verify task exists
	task, err := h.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		writeInternalError(w, r, "failed to verify task")
		return
	}
	if task == nil {
		writeNotFound(w, r, "task")
		return
	}

	// Parse pagination using shared utility
	pagination := ParsePagination(query, h.paginationCfg)

	filter := notes.ListFilter{
		TaskID: taskID,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}

	items, total, err := h.repo.ListByTask(ctx, filter)
	if err != nil {
		writeInternalError(w, r, "failed to list notes")
		return
	}
	if items == nil {
		items = []*notes.Note{}
	}

	writeJSON(w, http.StatusOK, newListResponse(items, total, filter.Limit, filter.Offset))
}

// Create adds a new note to a task.
// POST /api/v1/tasks/{task_id}/notes
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := mux.Vars(r)["task_id"]

	// Verify task exists
	task, err := h.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		writeInternalError(w, r, "failed to verify task")
		return
	}
	if task == nil {
		writeNotFound(w, r, "task")
		return
	}

	var input notes.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeBadRequest(w, r, "invalid request body")
		return
	}
	input.TaskID = taskID

	note, err := notes.NewNote(input)
	if err != nil {
		writeValidationError(w, r, err.Error(), nil)
		return
	}

	if err := h.repo.Create(ctx, note); err != nil {
		writeInternalError(w, r, "failed to create note")
		return
	}

	writeJSON(w, http.StatusCreated, note)
}

// Get retrieves a single note by ID.
// GET /api/v1/notes/{id}
func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	note, err := h.repo.FindByID(ctx, id)
	if err != nil {
		writeInternalError(w, r, "failed to retrieve note")
		return
	}
	if note == nil {
		writeNotFound(w, r, "note")
		return
	}

	writeJSON(w, http.StatusOK, note)
}

// Update modifies an existing note.
// PATCH /api/v1/notes/{id}
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	note, err := h.repo.FindByID(ctx, id)
	if err != nil {
		writeInternalError(w, r, "failed to retrieve note")
		return
	}
	if note == nil {
		writeNotFound(w, r, "note")
		return
	}

	var input notes.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeBadRequest(w, r, "invalid request body")
		return
	}

	if err := note.ApplyUpdate(input); err != nil {
		writeValidationError(w, r, err.Error(), nil)
		return
	}

	if err := h.repo.Update(ctx, note); err != nil {
		writeInternalError(w, r, "failed to update note")
		return
	}

	writeJSON(w, http.StatusOK, note)
}

// Delete removes a note by ID.
// DELETE /api/v1/notes/{id}
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if err := h.repo.Delete(ctx, id); err != nil {
		if err.Error() == "note not found" {
			writeNotFound(w, r, "note")
			return
		}
		writeInternalError(w, r, "failed to delete note")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
