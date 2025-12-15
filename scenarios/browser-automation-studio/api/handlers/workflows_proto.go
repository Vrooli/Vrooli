package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"google.golang.org/protobuf/proto"
)

func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req basapi.CreateWorkflowRequest
	if err := decodeProtoJSONBody(r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := h.workflowCatalog.CreateWorkflow(ctx, &req)
	if err != nil {
		if errors.Is(err, workflow.ErrWorkflowNameConflict) {
			h.respondError(w, ErrWorkflowAlreadyExists.WithDetails(map[string]string{"name": req.Name}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_workflow"}))
		return
	}

	h.respondProto(w, http.StatusCreated, resp)
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 50, 0)
	folderPath := strings.TrimSpace(r.URL.Query().Get("folder_path"))
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))

	req := &basapi.ListWorkflowsRequest{
		Limit:      proto.Int32(int32(limit)),
		Offset:     proto.Int32(int32(offset)),
		FolderPath: proto.String(folderPath),
	}
	if projectID != "" {
		req.ProjectId = proto.String(projectID)
	}

	resp, err := h.workflowCatalog.ListWorkflows(ctx, req)
	if err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	req := &basapi.GetWorkflowRequest{WorkflowId: idStr}
	if versionStr := strings.TrimSpace(r.URL.Query().Get("version")); versionStr != "" {
		if v, err := strconv.Atoi(versionStr); err == nil && v > 0 {
			req.Version = proto.Int32(int32(v))
		}
	}

	resp, err := h.workflowCatalog.GetWorkflow(ctx, req)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": idStr}))
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_workflow"}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

func (h *Handler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	var req basapi.UpdateWorkflowRequest
	if err := decodeProtoJSONBody(r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	// REST path param wins.
	req.WorkflowId = proto.String(idStr)

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := h.workflowCatalog.UpdateWorkflow(ctx, &req)
	if err != nil {
		if errors.Is(err, workflow.ErrWorkflowVersionConflict) {
			h.respondError(w, ErrWorkflowVersionConflict.WithDetails(map[string]string{"workflow_id": idStr}))
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_workflow"}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

func (h *Handler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := h.workflowCatalog.DeleteWorkflow(ctx, &basapi.DeleteWorkflowRequest{WorkflowId: idStr})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": idStr}))
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_workflow"}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}
