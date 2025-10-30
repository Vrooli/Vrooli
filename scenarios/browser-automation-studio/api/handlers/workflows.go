package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/services"
)

// CreateWorkflowRequest represents the request to create a workflow
type CreateWorkflowRequest struct {
	ProjectID      *uuid.UUID     `json:"project_id,omitempty"`
	Name           string         `json:"name"`
	FolderPath     string         `json:"folder_path"`
	FlowDefinition map[string]any `json:"flow_definition,omitempty"`
	AIPrompt       string         `json:"ai_prompt,omitempty"`
}

// ExecuteWorkflowRequest represents the request to execute a workflow
type ExecuteWorkflowRequest struct {
	Parameters        map[string]any `json:"parameters,omitempty"`
	WaitForCompletion bool           `json:"wait_for_completion"`
}

// ModifyWorkflowRequest represents the request to modify a workflow with AI support
type ModifyWorkflowRequest struct {
	ModificationPrompt string         `json:"modification_prompt"`
	CurrentFlow        map[string]any `json:"current_flow,omitempty"`
}

// CreateWorkflow handles POST /api/v1/workflows/create
func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode create workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	if req.FolderPath == "" {
		req.FolderPath = "/"
	}

	timeout := constants.DefaultRequestTimeout
	if strings.TrimSpace(req.AIPrompt) != "" {
		timeout = constants.AIRequestTimeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	workflow, err := h.workflowService.CreateWorkflowWithProject(ctx, req.ProjectID, req.Name, req.FolderPath, req.FlowDefinition, req.AIPrompt)
	if err != nil {
		h.log.WithError(err).Error("Failed to create workflow")
		var aiErr *services.AIWorkflowError
		switch {
		case errors.As(err, &aiErr):
			h.respondError(w, ErrInvalidRequest.WithMessage(aiErr.Reason))
		case strings.TrimSpace(req.AIPrompt) != "":
			h.respondError(w, ErrAIServiceError.WithDetails(map[string]string{"error": err.Error()}))
		default:
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_workflow"}))
		}
		return
	}

	h.respondSuccess(w, http.StatusCreated, newWorkflowResponse(workflow))
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	folderPath := r.URL.Query().Get("folder_path")

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	workflows, err := h.workflowService.ListWorkflows(ctx, folderPath, 100, 0)
	if err != nil {
		h.log.WithError(err).Error("Failed to list workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"workflows": workflows,
	})
}

// GetWorkflow handles GET /api/v1/workflows/{id}
func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	workflow, err := h.workflowService.GetWorkflow(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get workflow")
		h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": id.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, newWorkflowResponse(workflow))
}

// ExecuteWorkflow handles POST /api/v1/workflows/{id}/execute
func (h *Handler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	var req ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode execute workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	execution, err := h.workflowService.ExecuteWorkflow(ctx, workflowID, req.Parameters)
	if err != nil {
		h.log.WithError(err).Error("Failed to execute workflow")
		h.respondError(w, ErrWorkflowExecutionFailed.WithDetails(map[string]string{
			"workflow_id": workflowID.String(),
			"error":       err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"execution_id": execution.ID,
		"status":       execution.Status,
	})
}

// ModifyWorkflow handles POST /api/v1/workflows/{id}/modify
func (h *Handler) ModifyWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	var req ModifyWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode modify workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if strings.TrimSpace(req.ModificationPrompt) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "modification_prompt"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.AIRequestTimeout)
	defer cancel()

	workflow, err := h.workflowService.ModifyWorkflow(ctx, workflowID, req.ModificationPrompt, req.CurrentFlow)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to modify workflow via AI")
		h.respondError(w, ErrAIServiceError.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, struct {
		workflowResponse
		ModificationNote string `json:"modification_note,omitempty"`
	}{
		workflowResponse: newWorkflowResponse(workflow),
		ModificationNote: "ai",
	})
}
