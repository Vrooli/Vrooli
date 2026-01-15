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
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	"google.golang.org/protobuf/proto"
)

func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Decode into regular struct to allow flow_definition normalization.
	// The flow_definition contains ReactFlow fields that need to be stripped/converted
	// before proto parsing can succeed.
	var body struct {
		ProjectID      string         `json:"project_id"`
		Name           string         `json:"name"`
		FolderPath     string         `json:"folder_path"`
		FlowDefinition map[string]any `json:"flow_definition"`
		AIPrompt       string         `json:"ai_prompt"`
	}
	if err := decodeJSONBody(w, r, &body); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	// Normalize and convert flow_definition to proto.
	flowDef, err := workflowservice.BuildFlowDefinitionV2ForWrite(body.FlowDefinition, nil, nil)
	if err != nil {
		if errors.Is(err, workflowservice.ErrInvalidWorkflowFormat) {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid workflow format: nodes must have 'action' field with typed action definitions"}))
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	// Build proto request with normalized flow_definition.
	req := &basapi.CreateWorkflowRequest{
		ProjectId:      body.ProjectID,
		Name:           body.Name,
		FolderPath:     body.FolderPath,
		FlowDefinition: flowDef,
		AiPrompt:       body.AIPrompt,
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := h.catalogService.CreateWorkflow(ctx, req)
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowNameConflict) {
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

	resp, err := h.catalogService.ListWorkflows(ctx, req)
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

	resp, err := h.catalogService.GetWorkflowAPI(ctx, req)
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

	// Decode into regular struct to allow flow_definition normalization.
	// The flow_definition contains ReactFlow fields that need to be stripped/converted
	// before proto parsing can succeed.
	var body struct {
		Name              string         `json:"name"`
		Description       string         `json:"description"`
		FolderPath        string         `json:"folder_path"`
		Tags              []string       `json:"tags"`
		FlowDefinition    map[string]any `json:"flow_definition"`
		ChangeDescription string         `json:"change_description"`
		Source            string         `json:"source"`
		ExpectedVersion   int32          `json:"expected_version"`
	}
	if err := decodeJSONBody(w, r, &body); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	// Normalize and convert flow_definition to proto.
	flowDef, err := workflowservice.BuildFlowDefinitionV2ForWrite(body.FlowDefinition, nil, nil)
	if err != nil {
		if errors.Is(err, workflowservice.ErrInvalidWorkflowFormat) {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid workflow format: nodes must have 'action' field with typed action definitions"}))
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	// Build proto request with normalized flow_definition.
	req := &basapi.UpdateWorkflowRequest{
		Name:              body.Name,
		Description:       body.Description,
		FolderPath:        body.FolderPath,
		Tags:              body.Tags,
		FlowDefinition:    flowDef,
		ChangeDescription: body.ChangeDescription,
		Source:            parseChangeSource(body.Source),
		ExpectedVersion:   body.ExpectedVersion,
		WorkflowId:        proto.String(idStr),
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := h.catalogService.UpdateWorkflow(ctx, req)
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowVersionConflict) {
			h.respondError(w, ErrWorkflowVersionConflict.WithDetails(map[string]string{"workflow_id": idStr}))
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_workflow"}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

// parseChangeSource converts a string source to the proto enum value.
func parseChangeSource(source string) basbase.ChangeSource {
	switch strings.ToLower(source) {
	case "manual":
		return basbase.ChangeSource_CHANGE_SOURCE_MANUAL
	case "autosave":
		return basbase.ChangeSource_CHANGE_SOURCE_AUTOSAVE
	case "import":
		return basbase.ChangeSource_CHANGE_SOURCE_IMPORT
	case "ai_generated", "ai-generated":
		return basbase.ChangeSource_CHANGE_SOURCE_AI_GENERATED
	case "recording":
		return basbase.ChangeSource_CHANGE_SOURCE_RECORDING
	default:
		return basbase.ChangeSource_CHANGE_SOURCE_UNSPECIFIED
	}
}

func (h *Handler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := h.catalogService.DeleteWorkflow(ctx, &basapi.DeleteWorkflowRequest{WorkflowId: idStr})
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
