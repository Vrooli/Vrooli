package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
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

// UpdateWorkflowRequest captures manual or autosave edits from the UI/CLI.
type UpdateWorkflowRequest struct {
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	FolderPath        string         `json:"folder_path"`
	Tags              []string       `json:"tags"`
	FlowDefinition    map[string]any `json:"flow_definition,omitempty"`
	Nodes             []any          `json:"nodes,omitempty"`
	Edges             []any          `json:"edges,omitempty"`
	ChangeDescription string         `json:"change_description,omitempty"`
	Source            string         `json:"source,omitempty"`
	ExpectedVersion   *int           `json:"expected_version,omitempty"`
}

// RestoreWorkflowVersionRequest allows callers to provide an optional change description when rolling back.
type RestoreWorkflowVersionRequest struct {
	ChangeDescription string `json:"change_description"`
}

type workflowVersionResponse struct {
	Version           int            `json:"version"`
	WorkflowID        uuid.UUID      `json:"workflow_id"`
	CreatedAt         string         `json:"created_at"`
	CreatedBy         string         `json:"created_by"`
	ChangeDescription string         `json:"change_description"`
	DefinitionHash    string         `json:"definition_hash"`
	NodeCount         int            `json:"node_count"`
	EdgeCount         int            `json:"edge_count"`
	FlowDefinition    map[string]any `json:"flow_definition"`
}

func toWorkflowVersionResponse(summary *services.WorkflowVersionSummary) workflowVersionResponse {
	if summary == nil {
		return workflowVersionResponse{}
	}
	return workflowVersionResponse{
		Version:           summary.Version,
		WorkflowID:        summary.WorkflowID,
		CreatedAt:         summary.CreatedAt.UTC().Format(time.RFC3339Nano),
		CreatedBy:         summary.CreatedBy,
		ChangeDescription: summary.ChangeDescription,
		DefinitionHash:    summary.DefinitionHash,
		NodeCount:         summary.NodeCount,
		EdgeCount:         summary.EdgeCount,
		FlowDefinition:    jsonMapToStd(summary.FlowDefinition),
	}
}

func jsonMapToStd(m database.JSONMap) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
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
		case errors.Is(err, services.ErrWorkflowNameConflict):
			details := map[string]string{"name": req.Name}
			if req.ProjectID != nil {
				details["project_id"] = req.ProjectID.String()
			}
			h.respondError(w, ErrWorkflowAlreadyExists.WithDetails(details))
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

// UpdateWorkflow handles PUT /api/v1/workflows/{id}
func (h *Handler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	var req UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).WithField("workflow_id", id).Error("Failed to decode update workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	updateInput := services.WorkflowUpdateInput{
		Name:              req.Name,
		Description:       req.Description,
		FolderPath:        req.FolderPath,
		Tags:              req.Tags,
		FlowDefinition:    req.FlowDefinition,
		Nodes:             req.Nodes,
		Edges:             req.Edges,
		ChangeDescription: req.ChangeDescription,
		Source:            req.Source,
		ExpectedVersion:   req.ExpectedVersion,
	}

	workflow, err := h.workflowService.UpdateWorkflow(ctx, id, updateInput)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", id).Error("Failed to update workflow")
		if errors.Is(err, services.ErrWorkflowVersionConflict) {
			h.respondError(w, ErrConflict.WithDetails(map[string]string{"error": err.Error()}))
			return
		}
		if strings.Contains(err.Error(), "invalid workflow definition") {
			h.respondError(w, ErrInvalidRequest.WithMessage(err.Error()))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "update_workflow"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, newWorkflowResponse(workflow))
}

// ListWorkflowVersions handles GET /api/v1/workflows/{id}/versions
func (h *Handler) ListWorkflowVersions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	query := r.URL.Query()
	limit := 50
	if limitStr := strings.TrimSpace(query.Get("limit")); limitStr != "" {
		if parsed, parseErr := strconv.Atoi(limitStr); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if offsetStr := strings.TrimSpace(query.Get("offset")); offsetStr != "" {
		if parsed, parseErr := strconv.Atoi(offsetStr); parseErr == nil && parsed >= 0 {
			offset = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	versions, err := h.workflowService.ListWorkflowVersions(ctx, id, limit, offset)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", id).Error("Failed to list workflow versions")
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": id.String()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "list_workflow_versions"}))
		return
	}

	responses := make([]workflowVersionResponse, 0, len(versions))
	for _, version := range versions {
		responses = append(responses, toWorkflowVersionResponse(version))
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"versions": responses,
	})
}

// GetWorkflowVersion handles GET /api/v1/workflows/{id}/versions/{version}
func (h *Handler) GetWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	versionStr := chi.URLParam(r, "version")
	versionNumber, convErr := strconv.Atoi(versionStr)
	if convErr != nil || versionNumber <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"version": versionStr}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	versionSummary, err := h.workflowService.GetWorkflowVersion(ctx, id, versionNumber)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to get workflow version")
		switch {
		case errors.Is(err, services.ErrWorkflowVersionNotFound):
			h.respondError(w, ErrWorkflowVersionNotFound.WithDetails(map[string]string{"workflow_id": id.String(), "version": versionStr}))
		case errors.Is(err, database.ErrNotFound):
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": id.String()}))
		default:
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_workflow_version"}))
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, toWorkflowVersionResponse(versionSummary))
}

// RestoreWorkflowVersion handles POST /api/v1/workflows/{id}/versions/{version}/restore
func (h *Handler) RestoreWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	versionStr := chi.URLParam(r, "version")
	versionNumber, convErr := strconv.Atoi(versionStr)
	if convErr != nil || versionNumber <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"version": versionStr}))
		return
	}

	var req RestoreWorkflowVersionRequest
	if r.Body != nil {
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
			h.respondError(w, ErrInvalidRequest.WithMessage("Invalid restore payload"))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	versionSummary, err := h.workflowService.GetWorkflowVersion(ctx, id, versionNumber)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to resolve workflow version for restore")
		h.respondError(w, ErrWorkflowVersionNotFound.WithDetails(map[string]string{"workflow_id": id.String(), "version": versionStr}))
		return
	}

	updatedWorkflow, err := h.workflowService.RestoreWorkflowVersion(ctx, id, versionNumber, req.ChangeDescription)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to restore workflow version")
		switch {
		case errors.Is(err, services.ErrWorkflowVersionNotFound):
			h.respondError(w, ErrWorkflowVersionNotFound.WithDetails(map[string]string{"workflow_id": id.String(), "version": versionStr}))
		case errors.Is(err, services.ErrWorkflowRestoreProjectMismatch):
			h.respondError(w, ErrInvalidRequest.WithMessage("Workflow is not associated with a project"))
		case errors.Is(err, services.ErrWorkflowVersionConflict):
			h.respondError(w, ErrConflict)
		default:
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "restore_workflow_version"}))
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"workflow":         newWorkflowResponse(updatedWorkflow),
		"restored_version": toWorkflowVersionResponse(versionSummary),
	})
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
