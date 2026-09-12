package handlers

import (
	"io"
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// =============================================================================
// TASK HANDLERS
// =============================================================================

// CreateTask creates a new task.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.CreateTaskRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Task == nil {
		writeSimpleError(w, r, "task", "task is required")
		return
	}

	task := protoconv.TaskFromProto(req.Task)

	// Validate before sending to service
	if err := task.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.CreateTask(r.Context(), task)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateTaskResponse{
		Task: protoconv.TaskToProto(result),
	})
}

// GetTask retrieves a task by ID.
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetTaskRequest{TaskId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	task, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetTaskResponse{
		Task: protoconv.TaskToProto(task),
	})
}

// ListTasks returns all tasks.
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	limit, limitProvided, err := parseQueryIntStrict(r, "limit")
	if err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	}
	offset, offsetProvided, err := parseQueryIntStrict(r, "offset")
	if err != nil {
		writeSimpleError(w, r, "offset", "must be a number")
		return
	}
	statusRaw := queryFirst(r, "status")
	scopePrefix := queryFirst(r, "scope_prefix", "scopePrefix")

	var statusFilter *domain.TaskStatus
	var statusProto *domainpb.TaskStatus
	if statusRaw != "" {
		if parsed, ok := parseTaskStatus(statusRaw); ok {
			statusFilter = &parsed
			converted := protoconv.TaskStatusToProto(parsed)
			statusProto = &converted
		} else {
			writeSimpleError(w, r, "status", "invalid task status")
			return
		}
	}

	req := apipb.ListTasksRequest{}
	if statusProto != nil {
		req.Status = statusProto
	}
	if scopePrefix != "" {
		req.ScopePrefix = &scopePrefix
	}
	if limitProvided {
		value := int32(limit)
		req.Limit = &value
	}
	if offsetProvided {
		value := int32(offset)
		req.Offset = &value
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.ListOptions{}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if req.Offset != nil {
		opts.Offset = int(req.GetOffset())
	}
	tasks, err := h.svc.ListTasks(r.Context(), orchestration.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	if statusFilter != nil || scopePrefix != "" {
		filtered := make([]*domain.Task, 0, len(tasks))
		for _, task := range tasks {
			if statusFilter != nil && task.Status != *statusFilter {
				continue
			}
			if scopePrefix != "" && !strings.HasPrefix(task.ScopePath, scopePrefix) {
				continue
			}
			filtered = append(filtered, task)
		}
		tasks = filtered
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ListTasksResponse{
		Tasks: protoconv.TasksToProto(tasks),
		Total: int32(len(tasks)),
	})
}

// UpdateTask updates an existing task.
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.UpdateTaskRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Task == nil {
		writeSimpleError(w, r, "task", "task is required")
		return
	}
	if req.TaskId != "" {
		if req.TaskId != id.String() {
			writeSimpleError(w, r, "task_id", "task_id does not match URL")
			return
		}
	}

	task := protoconv.TaskFromProto(req.Task)
	task.ID = id

	// Validate before sending to service
	if err := task.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.UpdateTask(r.Context(), task)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.UpdateTaskResponse{
		Task: protoconv.TaskToProto(result),
	})
}

// CancelTask cancels a queued or running task.
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.CancelTaskRequest{TaskId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.CancelTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.CancelTaskResponse{Success: true, Status: "cancelled"})
}

// DeleteTask permanently removes a cancelled task.
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.DeleteTaskRequest{TaskId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.DeleteTaskResponse{Success: true})
}
