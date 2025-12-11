package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	WorkflowID        string         `json:"workflow_id,omitempty"`
	WorkflowVersion   *int           `json:"workflow_version,omitempty"`
}

// ExecuteAdhocWorkflowRequest represents the request to execute a workflow without persistence
type ExecuteAdhocWorkflowRequest struct {
	FlowDefinition    map[string]any `json:"flow_definition"`
	Parameters        map[string]any `json:"parameters,omitempty"`
	WaitForCompletion bool           `json:"wait_for_completion"`
	Metadata          *struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	} `json:"metadata,omitempty"`
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
	WorkflowID        string         `json:"workflow_id,omitempty"`
}

// RestoreWorkflowVersionRequest allows callers to provide an optional change description when rolling back.
type RestoreWorkflowVersionRequest struct {
	ChangeDescription string `json:"change_description"`
}

// enforceProtoRequestShape marshals the decoded request into the generated proto request to catch
// unknown/invalid fields early. Returns false if validation fails and writes an error response.
func (h *Handler) enforceProtoRequestShape(w http.ResponseWriter, operation string, decoded any, msg proto.Message) bool {
	raw, err := json.Marshal(decoded)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"operation": operation,
			"error":     "failed to marshal request",
		}))
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, msg); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"operation": operation,
			"error":     fmt.Sprintf("request does not match proto schema: %v", err),
		}))
		return false
	}
	return true
}

const executionCompletionPollInterval = 250 * time.Millisecond // moved to workflow_helpers.go in v2

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

// toWorkflowVersionResponse and jsonMapToStd moved to workflow_helpers.go

func (h *Handler) validateWorkflowDefinition(w http.ResponseWriter, r *http.Request, definition map[string]any, strict bool) bool {
	if h == nil || h.workflowValidator == nil {
		return true
	}
	if definition == nil || len(definition) == 0 {
		return true
	}
	if err := validateWorkflowProtoShape(definition); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "workflow definition does not match proto schema",
		}))
		return false
	}
	result, err := h.workflowValidator.Validate(r.Context(), definition, workflowvalidator.Options{Strict: strict})
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Error("Failed to validate workflow definition")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "workflow_validation"}))
		return false
	}
	if !result.Valid {
		h.respondError(w, ErrWorkflowValidationFailed.WithDetails(result))
		return false
	}
	return true
}

func (h *Handler) validateWorkflowDefinitionStrict(w http.ResponseWriter, r *http.Request, definition map[string]any) bool {
	return h.validateWorkflowDefinition(w, r, definition, true)
}

// definitionFromNodesEdges moved to workflow_helpers.go

// CreateWorkflow handles POST /api/v1/workflows/create
func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode create workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if !h.enforceProtoRequestShape(w, "create_workflow", req, &browser_automation_studio_v1.CreateWorkflowRequest{}) {
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

	// Skip validation for empty workflows to allow manual building from scratch
	// Empty workflows (no nodes) are valid starting points for the visual builder
	hasNodes := req.FlowDefinition != nil && len(req.FlowDefinition) > 0
	if nodes, ok := req.FlowDefinition["nodes"].([]interface{}); ok {
		hasNodes = len(nodes) > 0
	}
	if hasNodes && !h.validateWorkflowDefinition(w, r, req.FlowDefinition, false) {
		return
	}

	wf, err := h.workflowCatalog.CreateWorkflowWithProject(ctx, req.ProjectID, req.Name, req.FolderPath, req.FlowDefinition, req.AIPrompt)
	if err != nil {
		h.log.WithError(err).Error("Failed to create workflow")
		var aiErr *workflow.AIWorkflowError
		switch {
		case errors.Is(err, workflow.ErrWorkflowNameConflict):
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

	pb, convErr := protoconv.CreateWorkflowResponseProto(wf)
	if convErr != nil {
		h.log.WithError(convErr).Error("Failed to convert create workflow response to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_workflow_proto"}))
		return
	}

	h.respondProto(w, http.StatusCreated, pb)
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	folderPath := r.URL.Query().Get("folder_path")
	limit, offset := parsePaginationParams(r, 0, 0)

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	workflows, err := h.workflowCatalog.ListWorkflows(ctx, folderPath, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}

	items := make([]proto.Message, 0, len(workflows))
	for idx, wf := range workflows {
		pb, convErr := protoconv.WorkflowSummaryToProto(wf)
		if convErr != nil {
			h.log.WithError(convErr).WithField("workflow_index", idx).Error("Failed to convert workflow to proto")
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "workflow_to_proto"}))
			return
		}
		if pb != nil {
			items = append(items, pb)
		}
	}

	h.respondProtoList(w, http.StatusOK, "workflows", items)
}

// GetWorkflow handles GET /api/v1/workflows/{id}
func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	wf, err := h.workflowCatalog.GetWorkflow(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get workflow")
		h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": id.String()}))
		return
	}

	pb, convErr := protoconv.WorkflowSummaryToProto(wf)
	if convErr != nil {
		h.log.WithError(convErr).WithField("workflow_id", id).Error("Failed to convert workflow to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "workflow_to_proto"}))
		return
	}

	h.respondProto(w, http.StatusOK, pb)
}

// UpdateWorkflow handles PUT /api/v1/workflows/{id}
func (h *Handler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	var req UpdateWorkflowRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).WithField("workflow_id", id).Error("Failed to decode update workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if strings.TrimSpace(req.WorkflowID) != "" && req.WorkflowID != id.String() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error":        "workflow_id in body does not match path",
			"path_id":      id.String(),
			"payload_id":   req.WorkflowID,
			"field":        "workflow_id",
			"current_path": r.URL.Path,
		}))
		return
	}
	if req.WorkflowID == "" {
		req.WorkflowID = id.String()
	}

	if !h.enforceProtoRequestShape(w, "update_workflow", req, &browser_automation_studio_v1.UpdateWorkflowRequest{}) {
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	updateInput := workflow.WorkflowUpdateInput{
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

	definitionForValidation := definitionFromNodesEdges(req.Nodes, req.Edges, req.FlowDefinition)
	if !h.validateWorkflowDefinitionStrict(w, r, definitionForValidation) {
		return
	}

	wf, err := h.workflowCatalog.UpdateWorkflow(ctx, id, updateInput)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", id).Error("Failed to update workflow")
		if errors.Is(err, workflow.ErrWorkflowVersionConflict) {
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

	pb, convErr := protoconv.UpdateWorkflowResponseProto(wf)
	if convErr != nil {
		h.log.WithError(convErr).WithField("workflow_id", id).Error("Failed to convert update workflow response to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "update_workflow_proto"}))
		return
	}

	h.respondProto(w, http.StatusOK, pb)
}

// ListWorkflowVersions handles GET /api/v1/workflows/{id}/versions
func (h *Handler) ListWorkflowVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
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

	versions, err := h.workflowCatalog.ListWorkflowVersions(ctx, id, limit, offset)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", id).Error("Failed to list workflow versions")
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": id.String()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "list_workflow_versions"}))
		return
	}

	items := make([]proto.Message, 0, len(versions))
	for idx, version := range versions {
		pb, convErr := protoconv.WorkflowVersionSummaryToProto(version)
		if convErr != nil {
			h.log.WithError(convErr).WithField("workflow_version_index", idx).Error("Failed to convert workflow version to proto")
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "workflow_version_to_proto"}))
			return
		}
		if pb != nil {
			items = append(items, pb)
		}
	}

	h.respondProtoList(w, http.StatusOK, "versions", items)
}

// GetWorkflowVersion handles GET /api/v1/workflows/{id}/versions/{version}
func (h *Handler) GetWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
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

	versionSummary, err := h.workflowCatalog.GetWorkflowVersion(ctx, id, versionNumber)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to get workflow version")
		switch {
		case errors.Is(err, workflow.ErrWorkflowVersionNotFound):
			h.respondError(w, ErrWorkflowVersionNotFound.WithDetails(map[string]string{"workflow_id": id.String(), "version": versionStr}))
		case errors.Is(err, database.ErrNotFound):
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": id.String()}))
		default:
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_workflow_version"}))
		}
		return
	}

	pb, convErr := protoconv.WorkflowVersionSummaryToProto(versionSummary)
	if convErr != nil {
		h.log.WithError(convErr).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to convert workflow version to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_workflow_version_proto"}))
		return
	}

	h.respondProto(w, http.StatusOK, pb)
}

// RestoreWorkflowVersion handles POST /api/v1/workflows/{id}/versions/{version}/restore
func (h *Handler) RestoreWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	versionStr := chi.URLParam(r, "version")
	versionNumber, convErr := strconv.Atoi(versionStr)
	if convErr != nil || versionNumber <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"version": versionStr}))
		return
	}

	var req RestoreWorkflowVersionRequest
	if err := decodeJSONBodyAllowEmpty(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithMessage("Invalid restore payload"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	versionSummary, err := h.workflowCatalog.GetWorkflowVersion(ctx, id, versionNumber)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to resolve workflow version for restore")
		h.respondError(w, ErrWorkflowVersionNotFound.WithDetails(map[string]string{"workflow_id": id.String(), "version": versionStr}))
		return
	}

	updatedWorkflow, err := h.workflowCatalog.RestoreWorkflowVersion(ctx, id, versionNumber, req.ChangeDescription)
	if err != nil {
		h.log.WithError(err).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to restore workflow version")
		switch {
		case errors.Is(err, workflow.ErrWorkflowVersionNotFound):
			h.respondError(w, ErrWorkflowVersionNotFound.WithDetails(map[string]string{"workflow_id": id.String(), "version": versionStr}))
		case errors.Is(err, workflow.ErrWorkflowRestoreProjectMismatch):
			h.respondError(w, ErrInvalidRequest.WithMessage("Workflow is not associated with a project"))
		case errors.Is(err, workflow.ErrWorkflowVersionConflict):
			h.respondError(w, ErrConflict)
		default:
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "restore_workflow_version"}))
		}
		return
	}

	restoreResp, restoreErr := protoconv.RestoreWorkflowVersionResponseProto(updatedWorkflow, versionSummary)
	if restoreErr != nil {
		h.log.WithError(restoreErr).WithFields(map[string]any{"workflow_id": id, "version": versionNumber}).Error("Failed to build restore workflow proto response")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "restore_workflow_proto"}))
		return
	}

	// ensure the workflow payload matches the latest persisted record
	if restoreResp.Workflow == nil {
		if wfSummary, err := protoconv.WorkflowSummaryToProto(updatedWorkflow); err == nil {
			restoreResp.Workflow = wfSummary
		}
	}
	if restoreResp.RestoredVersion == nil {
		if versionProto, err := protoconv.WorkflowVersionSummaryToProto(versionSummary); err == nil {
			restoreResp.RestoredVersion = versionProto
		}
	}

	h.respondProto(w, http.StatusOK, restoreResp)
}

// ExecuteWorkflow handles POST /api/v1/workflows/{id}/execute
func (h *Handler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	var req ExecuteWorkflowRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode execute workflow request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if strings.TrimSpace(req.WorkflowID) != "" && req.WorkflowID != workflowID.String() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error":        "workflow_id in body does not match path",
			"path_id":      workflowID.String(),
			"payload_id":   req.WorkflowID,
			"field":        "workflow_id",
			"current_path": r.URL.Path,
		}))
		return
	}
	if req.WorkflowID == "" {
		req.WorkflowID = workflowID.String()
	}

	if !h.enforceProtoRequestShape(w, "execute_workflow", req, &browser_automation_studio_v1.ExecuteWorkflowRequest{}) {
		return
	}

	// Use ExecutionCompletionTimeout for workflow execution since workflows may include complex
	// navigation, element interactions, and subflows that require more time than CRUD operations
	ctx, cancel := context.WithTimeout(r.Context(), constants.ExecutionCompletionTimeout)
	defer cancel()

	execution, err := h.executionService.ExecuteWorkflow(ctx, workflowID, req.Parameters)
	if err != nil {
		h.log.WithError(err).Error("Failed to execute workflow")
		h.respondError(w, ErrWorkflowExecutionFailed.WithDetails(map[string]string{
			"workflow_id": workflowID.String(),
			"error":       err.Error(),
		}))
		return
	}

	targetExecution := execution

	if req.WaitForCompletion {
		waitCtx, waitCancel := context.WithTimeout(r.Context(), constants.ExecutionCompletionTimeout)
		defer waitCancel()

		finalExecution, waitErr := h.waitForExecutionCompletion(waitCtx, execution)
		if waitErr != nil {
			if errors.Is(waitErr, context.DeadlineExceeded) || errors.Is(waitErr, context.Canceled) {
				h.respondError(w, ErrRequestTimeout.WithMessage("execution did not complete before wait_for_completion timeout"))
				return
			}
			h.log.WithError(waitErr).WithField("execution_id", execution.ID).Error("Failed while waiting for execution completion")
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "wait_for_execution", "execution_id": execution.ID.String()}))
			return
		}

		targetExecution = finalExecution
	}

	respProto, convErr := protoconv.ExecuteWorkflowResponseProto(targetExecution)
	if convErr != nil {
		h.log.WithError(convErr).WithField("execution_id", targetExecution.ID).Error("Failed to convert execute workflow response to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "execute_workflow_proto"}))
		return
	}

	h.respondProto(w, http.StatusOK, respProto)
}

// ModifyWorkflow handles POST /api/v1/workflows/{id}/modify
func (h *Handler) ModifyWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	var req ModifyWorkflowRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
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

	if !h.validateWorkflowDefinitionStrict(w, r, req.CurrentFlow) {
		return
	}

	wf, err := h.workflowCatalog.ModifyWorkflow(ctx, workflowID, req.ModificationPrompt, req.CurrentFlow)
	if err != nil {
		h.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to modify workflow via AI")
		h.respondError(w, ErrAIServiceError.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	pb, convErr := protoconv.UpdateWorkflowResponseProto(wf)
	if convErr != nil {
		h.log.WithError(convErr).WithField("workflow_id", workflowID).Error("Failed to convert modify workflow response to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "modify_workflow_proto"}))
		return
	}

	h.respondProto(w, http.StatusOK, pb)
}

// ExecuteAdhocWorkflow handles POST /api/v1/workflows/execute-adhoc
// This endpoint allows executing workflow definitions directly without persisting them to the database.
// It is particularly useful for testing scenarios where workflow pollution should be avoided.
func (h *Handler) ExecuteAdhocWorkflow(w http.ResponseWriter, r *http.Request) {
	var req ExecuteAdhocWorkflowRequest
	rawBody, err := readLimitedBody(w, r, httpjson.MaxBodyBytes())
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.respondError(w, ErrRequestTooLarge)
			return
		}
		h.log.WithError(err).Error("Failed to read adhoc workflow request body")
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "failed to read request body"}))
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.logInvalidWorkflowPayload(err, rawBody)
		h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{
			"json_error": err.Error(),
		}))
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		h.logInvalidWorkflowPayload(err, rawBody)
		h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{
			"json_error": "request body must contain a single JSON object",
		}))
		return
	}

	if len(bytes.TrimSpace(rawBody)) == 0 {
		h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{
			"json_error": "request body is empty",
		}))
		return
	}

	// Validate required fields
	if req.FlowDefinition == nil {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "flow_definition",
		}))
		return
	}

	// Use non-strict validation for adhoc workflows since they may come from other scenarios
	// with selectors not registered in this BAS instance
	if !h.validateWorkflowDefinition(w, r, req.FlowDefinition, false) {
		return
	}

	// Extract workflow name from metadata or use default
	workflowName := "adhoc-workflow"
	if req.Metadata != nil && strings.TrimSpace(req.Metadata.Name) != "" {
		workflowName = strings.TrimSpace(req.Metadata.Name)
	}

	// Use ExecutionCompletionTimeout for adhoc workflows since they may include complex
	// navigation, element interactions, and subflows that require more time than CRUD operations
	ctx, cancel := context.WithTimeout(r.Context(), constants.ExecutionCompletionTimeout)
	defer cancel()

	execution, err := h.executionService.ExecuteAdhocWorkflow(
		ctx,
		req.FlowDefinition,
		req.Parameters,
		workflowName,
	)
	if err != nil {
		h.log.WithError(err).WithField("workflow_name", workflowName).Error("Failed to execute adhoc workflow")
		h.respondError(w, ErrWorkflowExecutionFailed.WithDetails(map[string]string{
			"workflow_name": workflowName,
			"error":         err.Error(),
		}))
		return
	}

	if req.WaitForCompletion {
		waitCtx, waitCancel := context.WithTimeout(r.Context(), constants.ExecutionCompletionTimeout)
		defer waitCancel()

		finalExecution, waitErr := h.waitForExecutionCompletion(waitCtx, execution)
		if waitErr != nil {
			if errors.Is(waitErr, context.DeadlineExceeded) || errors.Is(waitErr, context.Canceled) {
				h.respondError(w, ErrRequestTimeout.WithMessage("execution did not complete before wait_for_completion timeout"))
				return
			}
			h.log.WithError(waitErr).WithField("execution_id", execution.ID).Error("Failed while waiting for adhoc execution completion")
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "wait_for_adhoc_execution", "execution_id": execution.ID.String()}))
			return
		}

		execution = finalExecution
	}

	respProto, convErr := protoconv.ExecuteAdhocResponseProto(execution, "Adhoc workflow execution started successfully")
	if convErr != nil {
		h.log.WithError(convErr).WithField("execution_id", execution.ID).Error("Failed to convert adhoc execution response to proto")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "execute_adhoc_proto"}))
		return
	}

	h.respondProto(w, http.StatusOK, respProto)
}

// Helper functions moved to workflow_helpers.go:
// - readLimitedBody
// - buildPayloadPreview
// - logInvalidWorkflowPayload
// - waitForExecutionCompletion
// - toWorkflowVersionResponse
// - jsonMapToStd
// - definitionFromNodesEdges
