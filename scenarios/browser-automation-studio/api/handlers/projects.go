package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	FolderPath  string   `json:"folder_path"`
	Preset      string   `json:"preset,omitempty"`
	PresetPaths []string `json:"preset_paths,omitempty"`
}

type InspectProjectFolderRequest struct {
	FolderPath string `json:"folder_path"`
}

type InspectProjectFolderResponse struct {
	FolderPath           string `json:"folder_path"`
	Exists               bool   `json:"exists"`
	IsDir                bool   `json:"is_dir"`
	HasBasMetadata       bool   `json:"has_bas_metadata"`
	MetadataError        string `json:"metadata_error,omitempty"`
	HasWorkflows         bool   `json:"has_workflows"`
	AlreadyIndexed       bool   `json:"already_indexed"`
	IndexedProjectID     string `json:"indexed_project_id,omitempty"`
	SuggestedName        string `json:"suggested_name,omitempty"`
	SuggestedDescription string `json:"suggested_description,omitempty"`
}

type ImportProjectRequest struct {
	FolderPath  string `json:"folder_path"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	FolderPath  string `json:"folder_path,omitempty"`
}

// BulkDeleteProjectWorkflowsRequest represents the request payload for deleting multiple workflows
type BulkDeleteProjectWorkflowsRequest struct {
	WorkflowIDs []string `json:"workflow_ids"`
}

// CreateProject handles POST /api/v1/projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode create project request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	if req.FolderPath == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "folder_path"}))
		return
	}

	// Validate and prepare folder path
	absPath, err := validateAndPrepareFolderPath(req.FolderPath, h.log)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": err.Error()}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Check if project with this name or folder path already exists
	existingProject, err := h.catalogService.GetProjectByName(ctx, req.Name)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		// Only treat as error if it's not a "not found" result
		h.log.WithError(err).WithField("name", req.Name).Error("Database error checking project name uniqueness")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_by_name"}))
		return
	}
	if existingProject != nil {
		h.respondError(w, ErrProjectAlreadyExists.WithMessage("Project with this name already exists"))
		return
	}

	existingByPath, err := h.catalogService.GetProjectByFolderPath(ctx, absPath)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		// Only treat as error if it's not a "not found" result
		h.log.WithError(err).WithField("folder_path", absPath).Error("Database error checking project folder uniqueness")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_by_folder"}))
		return
	}
	if existingByPath != nil {
		h.respondError(w, ErrProjectAlreadyExists.WithMessage("Project with this folder path already exists"))
		return
	}

	project := &database.ProjectIndex{
		Name:       req.Name,
		FolderPath: absPath,
	}

	if err := h.catalogService.CreateProject(ctx, project, req.Description); err != nil {
		h.log.WithError(err).Error("Failed to create project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_project"}))
		return
	}

	if strings.TrimSpace(req.Preset) != "" {
		if err := h.applyProjectPreset(ctx, project, req.Preset, req.PresetPaths); err != nil {
			h.log.WithError(err).WithFields(logrus.Fields{
				"project_id": project.ID.String(),
				"preset":     req.Preset,
			}).Error("Failed to apply project preset")
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"field": "preset",
				"error": err.Error(),
			}))
			return
		}
	}

	pb, err := h.catalogService.HydrateProject(ctx, project)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_project"}))
		return
	}
	h.respondProto(w, http.StatusCreated, pb)
}

func (h *Handler) applyProjectPreset(ctx context.Context, project *database.ProjectIndex, preset string, presetPaths []string) error {
	if h == nil || project == nil {
		return errors.New("invalid handler or project")
	}
	if project.ID == uuid.Nil {
		return errors.New("project id missing")
	}
	if strings.TrimSpace(project.FolderPath) == "" {
		return errors.New("project folder path missing")
	}

	preset = strings.ToLower(strings.TrimSpace(preset))
	var folders []string
	switch preset {
	case "empty":
		folders = nil
	case "recommended":
		folders = []string{"workflows/actions", "workflows/flows", "workflows/cases", "assets"}
	case "custom":
		folders = presetPaths
	default:
		return fmt.Errorf("unknown preset %q", preset)
	}

	for _, folder := range folders {
		rel, ok := normalizeProjectRelPath(folder)
		if !ok {
			return fmt.Errorf("invalid preset path %q", folder)
		}

		abs := filepath.Join(project.FolderPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("failed to create folder %q: %w", rel, err)
		}
	}

	return nil
}

func (h *Handler) readProjectMetadata(folderPath string) (*basprojects.Project, string) {
	if strings.TrimSpace(folderPath) == "" {
		return nil, "folder path missing"
	}
	metaPath := filepath.Join(folderPath, ".bas", "project.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ""
		}
		return nil, fmt.Sprintf("read metadata: %v", err)
	}
	var meta basprojects.Project
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &meta); err != nil {
		return nil, fmt.Sprintf("parse metadata: %v", err)
	}
	return &meta, ""
}

func hasWorkflowFiles(folderPath string) bool {
	workflowsRoot := filepath.Join(folderPath, "workflows")
	info, err := os.Stat(workflowsRoot)
	if err != nil || !info.IsDir() {
		return false
	}

	found := false
	_ = filepath.WalkDir(workflowsRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".workflow.json") {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// InspectProjectFolder handles POST /api/v1/projects/inspect-folder
func (h *Handler) InspectProjectFolder(w http.ResponseWriter, r *http.Request) {
	var req InspectProjectFolderRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}
	if strings.TrimSpace(req.FolderPath) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "folder_path"}))
		return
	}

	absPath, err := paths.ValidateAndNormalizeFolderPath(req.FolderPath, h.log)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": err.Error()}))
		return
	}

	resp := InspectProjectFolderResponse{FolderPath: absPath}

	info, statErr := os.Stat(absPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			h.respondSuccess(w, http.StatusOK, resp)
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": statErr.Error()}))
		return
	}

	resp.Exists = true
	resp.IsDir = info.IsDir()
	if !resp.IsDir {
		h.respondSuccess(w, http.StatusOK, resp)
		return
	}

	meta, metaErr := h.readProjectMetadata(absPath)
	if metaErr != "" {
		resp.HasBasMetadata = true
		resp.MetadataError = metaErr
	} else if meta != nil {
		resp.HasBasMetadata = true
		resp.SuggestedName = strings.TrimSpace(meta.Name)
		resp.SuggestedDescription = strings.TrimSpace(meta.Description)
	}

	resp.HasWorkflows = hasWorkflowFiles(absPath)

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	existing, getErr := h.catalogService.GetProjectByFolderPath(ctx, absPath)
	if getErr == nil && existing != nil {
		resp.AlreadyIndexed = true
		resp.IndexedProjectID = existing.ID.String()
		if strings.TrimSpace(resp.SuggestedName) == "" {
			resp.SuggestedName = existing.Name
		}
	} else if getErr != nil && !errors.Is(getErr, database.ErrNotFound) {
		h.log.WithError(getErr).WithField("folder_path", absPath).Warn("Failed to check existing project by folder path")
	}

	h.respondSuccess(w, http.StatusOK, resp)
}

// ImportProject handles POST /api/v1/projects/import
func (h *Handler) ImportProject(w http.ResponseWriter, r *http.Request) {
	var req ImportProjectRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}
	if strings.TrimSpace(req.FolderPath) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "folder_path"}))
		return
	}

	absPath, err := paths.ValidateAndNormalizeFolderPath(req.FolderPath, h.log)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": err.Error()}))
		return
	}
	info, statErr := os.Stat(absPath)
	if statErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": "folder does not exist"}))
		return
	}
	if !info.IsDir() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": "path is not a directory"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Idempotent: if already indexed, return it.
	existing, getErr := h.catalogService.GetProjectByFolderPath(ctx, absPath)
	if getErr == nil && existing != nil {
		pb, err := h.catalogService.HydrateProject(ctx, existing)
		if err != nil {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_project"}))
			return
		}
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	if getErr != nil && !errors.Is(getErr, database.ErrNotFound) {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_by_folder"}))
		return
	}

	meta, _ := h.readProjectMetadata(absPath)

	name := strings.TrimSpace(req.Name)
	if name == "" && meta != nil {
		name = strings.TrimSpace(meta.Name)
	}
	if name == "" {
		name = filepath.Base(absPath)
	}

	description := strings.TrimSpace(req.Description)
	if description == "" && meta != nil {
		description = strings.TrimSpace(meta.Description)
	}

	project := &database.ProjectIndex{
		Name:       name,
		FolderPath: absPath,
	}

	if meta != nil && strings.TrimSpace(meta.Id) != "" {
		if parsed, parseErr := uuid.Parse(strings.TrimSpace(meta.Id)); parseErr == nil {
			if byID, err := h.catalogService.GetProject(ctx, parsed); err == nil && byID != nil && byID.FolderPath != absPath {
				h.log.WithFields(logrus.Fields{
					"existing_id":   byID.ID.String(),
					"existing_path": byID.FolderPath,
					"imported_path": absPath,
					"metadata_id":   parsed.String(),
				}).Warn("Imported project metadata id already in use; generating a new id")
			} else {
				project.ID = parsed
			}
		}
	}

	if err := h.catalogService.CreateProject(ctx, project, description); err != nil {
		h.log.WithError(err).Error("Failed to import project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_project"}))
		return
	}

	if err := h.catalogService.SyncProjectWorkflows(ctx, project.ID); err != nil {
		h.log.WithError(err).WithField("project_id", project.ID.String()).Warn("Imported project workflow sync failed")
	}

	pb, err := h.catalogService.HydrateProject(ctx, project)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_project"}))
		return
	}
	h.respondProto(w, http.StatusCreated, pb)
}

// ListProjects handles GET /api/v1/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 0, 0)

	projects, err := h.catalogService.ListProjects(ctx, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list projects")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_projects"}))
		return
	}

	projectIDs := make([]uuid.UUID, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
	}

	statsByProject, err := h.catalogService.GetProjectsStats(ctx, projectIDs)
	if err != nil {
		h.log.WithError(err).Error("Failed to get project stats in bulk")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_stats"}))
		return
	}

	items := make([]proto.Message, 0, len(projects))
	for _, project := range projects {
		projectProto, err := h.catalogService.HydrateProject(ctx, project)
		if err != nil {
			continue
		}
		stats := statsByProject[project.ID]
		statsProto := &basprojects.ProjectStats{
			ProjectId:      project.ID.String(),
			WorkflowCount:  int32(stats.WorkflowCount),
			ExecutionCount: int32(stats.ExecutionCount),
		}
		if stats.LastExecution != nil {
			statsProto.LastExecution = timestamppb.New(*stats.LastExecution)
		}
		items = append(items, &basprojects.ProjectWithStats{
			Project: projectProto,
			Stats:   statsProto,
		})
	}

	h.respondProtoList(w, http.StatusOK, "projects", items)
}

// GetProject handles GET /api/v1/projects/{id}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.catalogService.GetProject(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get project")
		h.respondError(w, ErrProjectNotFound.WithDetails(map[string]string{"project_id": id.String()}))
		return
	}

	// Get project stats
	stats, err := h.catalogService.GetProjectStats(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("project_id", id).Warn("Failed to get project stats")
		stats = &database.ProjectStats{ProjectID: id}
	}

	projectProto, err := h.catalogService.HydrateProject(ctx, project)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_project"}))
		return
	}
	statsProto := &basprojects.ProjectStats{
		ProjectId:      id.String(),
		WorkflowCount:  int32(stats.WorkflowCount),
		ExecutionCount: int32(stats.ExecutionCount),
	}
	if stats.LastExecution != nil {
		statsProto.LastExecution = timestamppb.New(*stats.LastExecution)
	}
	h.respondProto(w, http.StatusOK, &basprojects.ProjectWithStats{
		Project: projectProto,
		Stats:   statsProto,
	})
}

// UpdateProject handles PUT /api/v1/projects/{id}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req UpdateProjectRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode update project request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get existing project
	project, err := h.catalogService.GetProject(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get project for update")
		h.respondError(w, ErrProjectNotFound.WithDetails(map[string]string{"project_id": id.String()}))
		return
	}

	projectProto, err := h.catalogService.HydrateProject(ctx, project)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_project"}))
		return
	}

	// Update fields if provided
	if req.Name != "" {
		project.Name = req.Name
	}
	description := projectProto.Description
	if req.Description != "" {
		description = req.Description
	}
	if req.FolderPath != "" {
		// Validate and update folder path
		absPath, err := validateAndPrepareFolderPath(req.FolderPath, h.log)
		if err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": err.Error()}))
			return
		}
		project.FolderPath = absPath
	}

	if err := h.catalogService.UpdateProject(ctx, project, description); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to update project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_project"}))
		return
	}

	updatedProto, err := h.catalogService.HydrateProject(ctx, project)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_project"}))
		return
	}
	h.respondProto(w, http.StatusOK, updatedProto)
}

// DeleteProject handles DELETE /api/v1/projects/{id}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.catalogService.DeleteProject(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to delete project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "deleted",
	})
}

// GetProjectWorkflows handles GET /api/v1/projects/{id}/workflows
func (h *Handler) GetProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	reqProto := &basapi.ListWorkflowsRequest{
		ProjectId: proto.String(projectID.String()),
		Limit:     proto.Int32(100),
		Offset:    proto.Int32(0),
	}
	respProto, err := h.catalogService.ListWorkflows(ctx, reqProto)
	if err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to list project workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}

	h.respondProto(w, http.StatusOK, respProto)
}

// BulkDeleteProjectWorkflows handles POST /api/v1/projects/{id}/workflows/bulk-delete
func (h *Handler) BulkDeleteProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req BulkDeleteProjectWorkflowsRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to decode bulk delete request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if len(req.WorkflowIDs) == 0 {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "workflow_ids"}))
		return
	}

	workflowIDs := make([]uuid.UUID, 0, len(req.WorkflowIDs))
	for _, workflowIDStr := range req.WorkflowIDs {
		workflowID, err := uuid.Parse(workflowIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowID.WithDetails(map[string]string{"workflow_id": workflowIDStr}))
			return
		}
		workflowIDs = append(workflowIDs, workflowID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	if err := h.catalogService.DeleteProjectWorkflows(ctx, projectID, workflowIDs); err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"count":      len(workflowIDs),
		}).Error("Failed to bulk delete project workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "bulk_delete_workflows"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status":        "deleted",
		"deleted_count": len(workflowIDs),
		"deleted_ids":   req.WorkflowIDs,
	})
}

// ExecuteAllProjectWorkflows handles POST /api/v1/projects/{id}/execute-all
func (h *Handler) ExecuteAllProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	// Get all workflows for this project
	workflows, err := h.catalogService.ListWorkflowsByProject(ctx, projectID, 1000, 0)
	if err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to get project workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}

	if len(workflows) == 0 {
		h.respondSuccess(w, http.StatusOK, map[string]any{
			"message":    "No workflows found in project",
			"executions": []any{},
		})
		return
	}

	// Start execution for each workflow
	executions := make([]map[string]any, 0, len(workflows))
	for _, workflow := range workflows {
		execution, err := h.executionService.ExecuteWorkflow(ctx, workflow.ID, map[string]any{})
		if err != nil {
			h.log.WithError(err).WithField("workflow_id", workflow.ID).Warn("Failed to execute workflow in bulk operation")
			executions = append(executions, map[string]any{
				"workflow_id":   workflow.ID.String(),
				"workflow_name": workflow.Name,
				"status":        "failed",
				"error":         err.Error(),
			})
			continue
		}

		executions = append(executions, map[string]any{
			"workflow_id":   workflow.ID.String(),
			"workflow_name": workflow.Name,
			"execution_id":  execution.ID.String(),
			"status":        execution.Status,
		})
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"message":    fmt.Sprintf("Started execution for %d workflows", len(executions)),
		"executions": executions,
	})
}
