package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
)

type ProjectFileTreeResponse struct {
	Entries []*database.ProjectEntry `json:"entries"`
}

type MkdirProjectPathRequest struct {
	Path string `json:"path"`
}

type WriteProjectWorkflowFileRequest struct {
	Path     string                   `json:"path"`
	Workflow ProjectWorkflowFileWrite `json:"workflow"`
}

type ProjectWorkflowFileWrite struct {
	ID              *uuid.UUID     `json:"id,omitempty"`
	Name            string         `json:"name"`
	Type            string         `json:"type"`
	Description     string         `json:"description,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	Inputs          map[string]any `json:"inputs,omitempty"`
	Outputs         map[string]any `json:"outputs,omitempty"`
	ExpectedOutcome map[string]any `json:"expectedOutcome,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	FlowDefinition  map[string]any `json:"flow_definition"`
}

type ProjectWorkflowFileOnDisk struct {
	Version         string         `json:"version"`
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Type            string         `json:"type"`
	Description     string         `json:"description,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	Inputs          map[string]any `json:"inputs,omitempty"`
	Outputs         map[string]any `json:"outputs,omitempty"`
	ExpectedOutcome map[string]any `json:"expectedOutcome,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	FlowDefinition  map[string]any `json:"flow_definition"`
}

type MoveProjectFileRequest struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
}

type DeleteProjectFileRequest struct {
	Path string `json:"path"`
}

type ResyncProjectFilesResponse struct {
	ProjectID        string `json:"project_id"`
	ProjectRoot      string `json:"project_root"`
	EntriesIndexed   int    `json:"entries_indexed"`
	WorkflowsIndexed int    `json:"workflows_indexed"`
	AssetsIndexed    int    `json:"assets_indexed"`
}

func flowDefinitionHasAssertNode(definition map[string]any) bool {
	if definition == nil {
		return false
	}
	nodesAny, ok := definition["nodes"]
	if !ok || nodesAny == nil {
		return false
	}
	nodes, ok := nodesAny.([]any)
	if !ok {
		return false
	}
	for _, nodeAny := range nodes {
		node, ok := nodeAny.(map[string]any)
		if !ok {
			continue
		}
		typAny, ok := node["type"]
		if !ok {
			continue
		}
		typ, ok := typAny.(string)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(typ), "assert") {
			return true
		}
	}
	return false
}

func jsonMapIsNonEmpty(m map[string]any) bool {
	if m == nil {
		return false
	}
	return len(m) > 0
}

func indexProjectFolderPath(ctx context.Context, repo database.Repository, projectID uuid.UUID, relPath string) error {
	relPath, ok := normalizeProjectRelPath(relPath)
	if !ok {
		return errors.New("invalid folder path")
	}
	segments := strings.Split(relPath, "/")
	for idx := range segments {
		folder := strings.Join(segments[:idx+1], "/")
		entry := &database.ProjectEntry{
			ProjectID: projectID,
			Path:      folder,
			Kind:      database.ProjectEntryKindFolder,
			Metadata:  database.JSONMap{},
		}
		if err := repo.UpsertProjectEntry(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func normalizeProjectRelPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" {
		return "", false
	}

	parts := strings.Split(raw, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", false
		}
		clean = append(clean, part)
	}

	return strings.Join(clean, "/"), true
}

func workflowTypeFromFilename(filename string) (string, bool) {
	switch {
	case strings.HasSuffix(filename, ".action.json"):
		return "action", true
	case strings.HasSuffix(filename, ".flow.json"):
		return "flow", true
	case strings.HasSuffix(filename, ".case.json"):
		return "case", true
	default:
		return "", false
	}
}

func safeJoinProjectPath(projectRoot string, relPath string) (string, error) {
	projectRoot = filepath.Clean(strings.TrimSpace(projectRoot))
	if projectRoot == "" || projectRoot == "." {
		return "", errors.New("invalid project root")
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return "", errors.New("invalid project relative path")
	}

	abs := filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(relPath)))
	rootWithSep := projectRoot + string(filepath.Separator)
	if abs != projectRoot && !strings.HasPrefix(abs, rootWithSep) {
		return "", errors.New("path escapes project root")
	}
	return abs, nil
}

func ensureProjectFoldersIndexed(ctx context.Context, repo database.Repository, projectID uuid.UUID, relPath string) error {
	dir := path.Dir(relPath)
	if dir == "." {
		return nil
	}
	segments := strings.Split(dir, "/")
	for idx := range segments {
		folder := strings.Join(segments[:idx+1], "/")
		entry := &database.ProjectEntry{
			ProjectID: projectID,
			Path:      folder,
			Kind:      database.ProjectEntryKindFolder,
			Metadata:  database.JSONMap{},
		}
		if err := repo.UpsertProjectEntry(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) GetProjectFileTree(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	_, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	entries, err := h.repo.ListProjectEntries(ctx, projectID)
	if err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_project_entries"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, ProjectFileTreeResponse{Entries: entries})
}

func (h *Handler) ReadProjectFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}
	rawPath := r.URL.Query().Get("path")
	relPath, ok := normalizeProjectRelPath(rawPath)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid path"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	_, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	entry, err := h.repo.GetProjectEntry(ctx, projectID, relPath)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectFileNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_entry"}))
		return
	}

	if entry.Kind != database.ProjectEntryKindWorkflowFile || entry.WorkflowID == nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "only workflow files are readable in v1"}))
		return
	}

	workflow, err := h.repo.GetWorkflow(ctx, *entry.WorkflowID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_workflow"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"path": relPath,
		"workflow": map[string]any{
			"id":              workflow.ID,
			"name":            workflow.Name,
			"type":            workflow.WorkflowType,
			"description":     workflow.Description,
			"tags":            workflow.Tags,
			"inputs":          workflow.Inputs,
			"outputs":         workflow.Outputs,
			"expectedOutcome": workflow.ExpectedOutcome,
			"metadata":        workflow.WorkflowMetadata,
			"flow_definition": workflow.FlowDefinition,
		},
	})
}

func (h *Handler) MkdirProjectPath(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req MkdirProjectPathRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	relPath, ok := normalizeProjectRelPath(req.Path)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid path"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	absPath, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "mkdir_project_path"}))
		return
	}

	if err := indexProjectFolderPath(ctx, h.repo, projectID, relPath); err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "upsert_project_entry"}))
		return
	}

	h.respondSuccess(w, http.StatusCreated, map[string]any{"path": relPath})
}

func (h *Handler) WriteProjectWorkflowFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req WriteProjectWorkflowFileRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	relPath, ok := normalizeProjectRelPath(req.Path)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid path"}))
		return
	}

	typeFromExt, ok := workflowTypeFromFilename(relPath)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "workflow files must end with .action.json, .flow.json, or .case.json"}))
		return
	}
	if strings.TrimSpace(req.Workflow.Type) == "" {
		req.Workflow.Type = typeFromExt
	}
	if req.Workflow.Type != typeFromExt {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "workflow type does not match file extension"}))
		return
	}
	if strings.TrimSpace(req.Workflow.Name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "workflow.name"}))
		return
	}
	if req.Workflow.FlowDefinition == nil {
		req.Workflow.FlowDefinition = map[string]any{"nodes": []any{}, "edges": []any{}}
	}

	// Validate workflow definition when non-empty (allow empty as a starting point).
	hasNodes := len(req.Workflow.FlowDefinition) > 0
	if nodes, ok := req.Workflow.FlowDefinition["nodes"].([]any); ok {
		hasNodes = len(nodes) > 0
	}
	if hasNodes && !h.validateWorkflowDefinition(w, r, req.Workflow.FlowDefinition, false) {
		return
	}

	hasAssert := flowDefinitionHasAssertNode(req.Workflow.FlowDefinition)
	if req.Workflow.Type == "case" && hasNodes && !hasAssert && !jsonMapIsNonEmpty(req.Workflow.ExpectedOutcome) {
		h.respondError(w, ErrCaseExpectationMissing)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	var workflowID uuid.UUID
	entry, entryErr := h.repo.GetProjectEntry(ctx, projectID, relPath)
	if entryErr == nil && entry.WorkflowID != nil {
		workflowID = *entry.WorkflowID
	} else if req.Workflow.ID != nil && *req.Workflow.ID != uuid.Nil {
		workflowID = *req.Workflow.ID
	} else {
		workflowID = uuid.New()
	}

	if err := ensureProjectFoldersIndexed(ctx, h.repo, projectID, relPath); err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "ensure_project_folders"}))
		return
	}

	folder := "/"
	dir := path.Dir(relPath)
	if dir != "." {
		folder = "/" + dir
	}

	flow := database.JSONMap(req.Workflow.FlowDefinition)
	inputs := database.JSONMap(req.Workflow.Inputs)
	outputs := database.JSONMap(req.Workflow.Outputs)
	expected := database.JSONMap(req.Workflow.ExpectedOutcome)
	meta := database.JSONMap(req.Workflow.Metadata)

	wf := &database.Workflow{
		ID:               workflowID,
		ProjectID:        &projectID,
		Name:             req.Workflow.Name,
		FolderPath:       folder,
		WorkflowType:     req.Workflow.Type,
		FlowDefinition:   flow,
		Inputs:           inputs,
		Outputs:          outputs,
		ExpectedOutcome:  expected,
		WorkflowMetadata: meta,
		Description:      req.Workflow.Description,
		Tags:             database.StringArray(req.Workflow.Tags),
		LastChangeSource: "project_files_api",
	}

	absFilePath, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}
	if err := os.MkdirAll(filepath.Dir(absFilePath), 0o755); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "mkdir_project_file_dir"}))
		return
	}

	onDisk := ProjectWorkflowFileOnDisk{
		Version:         "v1",
		ID:              workflowID.String(),
		Name:            wf.Name,
		Type:            wf.WorkflowType,
		Description:     wf.Description,
		Tags:            req.Workflow.Tags,
		Inputs:          req.Workflow.Inputs,
		Outputs:         req.Workflow.Outputs,
		ExpectedOutcome: req.Workflow.ExpectedOutcome,
		Metadata:        req.Workflow.Metadata,
		FlowDefinition:  req.Workflow.FlowDefinition,
	}
	encoded, err := json.MarshalIndent(onDisk, "", "  ")
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "marshal_workflow_file"}))
		return
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(absFilePath, encoded, 0o644); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "write_workflow_file"}))
		return
	}

	if entryErr == nil && entry.WorkflowID != nil {
		if existing, err := h.repo.GetWorkflow(ctx, workflowID); err == nil && existing != nil {
			wf.Version = existing.Version + 1
		}
		if err := h.repo.UpdateWorkflow(ctx, wf); err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_workflow"}))
			return
		}
	} else {
		if err := h.repo.CreateWorkflow(ctx, wf); err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_workflow"}))
			return
		}
	}

	entryMeta := database.JSONMap{
		"workflowType": wf.WorkflowType,
	}
	entry = &database.ProjectEntry{
		ProjectID:  projectID,
		Path:       relPath,
		Kind:       database.ProjectEntryKindWorkflowFile,
		WorkflowID: &workflowID,
		Metadata:   entryMeta,
	}
	if err := h.repo.UpsertProjectEntry(ctx, entry); err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "upsert_project_entry"}))
		return
	}

	warnings := make([]string, 0, 1)
	if (wf.WorkflowType == "action" || wf.WorkflowType == "flow") && hasNodes && hasAssert {
		warnings = append(warnings, "This workflow has Assert nodes; it is usually stored as a Case.")
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"path":       relPath,
		"workflowId": workflowID.String(),
		"warnings":   warnings,
	})
}

func (h *Handler) MoveProjectFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req MoveProjectFileRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	fromPath, ok := normalizeProjectRelPath(req.FromPath)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid from_path"}))
		return
	}
	toPath, ok := normalizeProjectRelPath(req.ToPath)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid to_path"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	if err := ensureProjectFoldersIndexed(ctx, h.repo, projectID, toPath); err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "ensure_project_folders"}))
		return
	}

	entry, err := h.repo.GetProjectEntry(ctx, projectID, fromPath)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectFileNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_entry"}))
		return
	}

	fromAbs, joinErr := safeJoinProjectPath(project.FolderPath, fromPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}
	toAbs, joinErr := safeJoinProjectPath(project.FolderPath, toPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}
	if err := os.MkdirAll(filepath.Dir(toAbs), 0o755); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "mkdir_project_move_dir"}))
		return
	}
	if err := os.Rename(fromAbs, toAbs); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "move_project_file"}))
		return
	}

	if entry.Kind == database.ProjectEntryKindFolder {
		entries, err := h.repo.ListProjectEntries(ctx, projectID)
		if err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_project_entries"}))
			return
		}
		candidates := make([]*database.ProjectEntry, 0)
		prefix := fromPath + "/"
		for _, e := range entries {
			if e.Path == fromPath || strings.HasPrefix(e.Path, prefix) {
				candidates = append(candidates, e)
			}
		}
		for _, e := range candidates {
			if err := h.repo.DeleteProjectEntry(ctx, projectID, e.Path); err != nil {
				h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project_entry"}))
				return
			}
		}
		for _, e := range candidates {
			oldPath := e.Path
			suffix := strings.TrimPrefix(oldPath, fromPath)
			if suffix != "" && strings.HasPrefix(suffix, "/") {
				suffix = strings.TrimPrefix(suffix, "/")
			}
			newPath := toPath
			if suffix != "" {
				newPath = toPath + "/" + suffix
			}
			e.Path = newPath
			if err := h.repo.UpsertProjectEntry(ctx, e); err != nil {
				h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "upsert_project_entry"}))
				return
			}
			if e.Kind == database.ProjectEntryKindWorkflowFile && e.WorkflowID != nil {
				if wf, err := h.repo.GetWorkflow(ctx, *e.WorkflowID); err == nil && wf != nil {
					dir := path.Dir(newPath)
					folder := "/"
					if dir != "." {
						folder = "/" + dir
					}
					wf.FolderPath = folder
					wf.Version++
					wf.LastChangeSource = "project_files_api"
					if err := h.repo.UpdateWorkflow(ctx, wf); err != nil {
						h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_workflow"}))
						return
					}
				}
			}
		}
	} else {
		if err := h.repo.DeleteProjectEntry(ctx, projectID, fromPath); err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project_entry"}))
			return
		}

		entry.Path = toPath
		if err := h.repo.UpsertProjectEntry(ctx, entry); err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "upsert_project_entry"}))
			return
		}

		if entry.Kind == database.ProjectEntryKindWorkflowFile && entry.WorkflowID != nil {
			if wf, err := h.repo.GetWorkflow(ctx, *entry.WorkflowID); err == nil && wf != nil {
				dir := path.Dir(toPath)
				folder := "/"
				if dir != "." {
					folder = "/" + dir
				}
				wf.FolderPath = folder
				wf.Version++
				wf.LastChangeSource = "project_files_api"
				if err := h.repo.UpdateWorkflow(ctx, wf); err != nil {
					h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_workflow"}))
					return
				}
			}
		}
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"from_path": fromPath,
		"to_path":   toPath,
	})
}

func (h *Handler) DeleteProjectFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req DeleteProjectFileRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	relPath, ok := normalizeProjectRelPath(req.Path)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid path"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	entry, err := h.repo.GetProjectEntry(ctx, projectID, relPath)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectFileNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project_entry"}))
		return
	}

	absPath, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}
	if err := os.RemoveAll(absPath); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "delete_project_file"}))
		return
	}

	if entry.Kind == database.ProjectEntryKindFolder {
		entries, err := h.repo.ListProjectEntries(ctx, projectID)
		if err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_project_entries"}))
			return
		}
		prefix := relPath + "/"
		for _, e := range entries {
			if e.Path != relPath && !strings.HasPrefix(e.Path, prefix) {
				continue
			}
			if err := h.repo.DeleteProjectEntry(ctx, projectID, e.Path); err != nil {
				h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project_entry"}))
				return
			}
			if e.Kind == database.ProjectEntryKindWorkflowFile && e.WorkflowID != nil {
				_ = h.repo.DeleteWorkflow(ctx, *e.WorkflowID)
			}
		}
	} else {
		if err := h.repo.DeleteProjectEntry(ctx, projectID, relPath); err != nil {
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project_entry"}))
			return
		}

		if entry.Kind == database.ProjectEntryKindWorkflowFile && entry.WorkflowID != nil {
			_ = h.repo.DeleteWorkflow(ctx, *entry.WorkflowID)
		}
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"path":    relPath,
		"kind":    entry.Kind,
		"deleted": true,
	})
}

func (h *Handler) ResyncProjectFiles(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	if err := os.MkdirAll(project.FolderPath, 0o755); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "ensure_project_root"}))
		return
	}

	if err := h.repo.DeleteProjectEntries(ctx, projectID); err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project_entries"}))
		return
	}

	var entriesIndexed, workflowsIndexed, assetsIndexed int

	walkErr := filepath.WalkDir(project.FolderPath, func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if abs == project.FolderPath {
			return nil
		}
		rel, err := filepath.Rel(project.FolderPath, abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "./")
		if rel == "" || rel == "." {
			return nil
		}

		if d.IsDir() {
			entry := &database.ProjectEntry{
				ProjectID: projectID,
				Path:      rel,
				Kind:      database.ProjectEntryKindFolder,
				Metadata:  database.JSONMap{},
			}
			if err := h.repo.UpsertProjectEntry(ctx, entry); err != nil {
				return err
			}
			entriesIndexed++
			return nil
		}

		if err := ensureProjectFoldersIndexed(ctx, h.repo, projectID, rel); err != nil {
			return err
		}

		if wfType, ok := workflowTypeFromFilename(rel); ok {
			raw, err := os.ReadFile(abs)
			if err != nil {
				return err
			}
			var wfFile ProjectWorkflowFileOnDisk
			if err := json.Unmarshal(raw, &wfFile); err != nil {
				// Not valid workflow JSON; index as asset.
				info, statErr := os.Stat(abs)
				if statErr != nil {
					return statErr
				}
				meta := database.JSONMap{"sizeBytes": info.Size(), "error": "invalid workflow json"}
				entry := &database.ProjectEntry{ProjectID: projectID, Path: rel, Kind: database.ProjectEntryKindAssetFile, Metadata: meta}
				if err := h.repo.UpsertProjectEntry(ctx, entry); err != nil {
					return err
				}
				entriesIndexed++
				assetsIndexed++
				return nil
			}

			changedOnDisk := false
			if strings.TrimSpace(wfFile.Version) == "" {
				wfFile.Version = "v1"
				changedOnDisk = true
			}
			if strings.TrimSpace(wfFile.Type) == "" || wfFile.Type != wfType {
				wfFile.Type = wfType
				changedOnDisk = true
			}
			if strings.TrimSpace(wfFile.Name) == "" {
				base := path.Base(rel)
				base = strings.TrimSuffix(base, ".json")
				base = strings.TrimSuffix(base, ".action")
				base = strings.TrimSuffix(base, ".flow")
				base = strings.TrimSuffix(base, ".case")
				wfFile.Name = base
				changedOnDisk = true
			}

			workflowID, parseErr := uuid.Parse(strings.TrimSpace(wfFile.ID))
			if parseErr != nil || workflowID == uuid.Nil {
				workflowID = uuid.New()
				wfFile.ID = workflowID.String()
				changedOnDisk = true
			}

			if wfFile.FlowDefinition == nil {
				wfFile.FlowDefinition = map[string]any{"nodes": []any{}, "edges": []any{}}
				changedOnDisk = true
			}

			if changedOnDisk {
				encoded, err := json.MarshalIndent(wfFile, "", "  ")
				if err != nil {
					return err
				}
				encoded = append(encoded, '\n')
				if err := os.WriteFile(abs, encoded, 0o644); err != nil {
					return err
				}
			}

			folder := "/"
			dir := path.Dir(rel)
			if dir != "." {
				folder = "/" + dir
			}

			next := &database.Workflow{
				ID:               workflowID,
				ProjectID:        &projectID,
				Name:             wfFile.Name,
				FolderPath:       folder,
				WorkflowType:     wfFile.Type,
				FlowDefinition:   database.JSONMap(wfFile.FlowDefinition),
				Inputs:           database.JSONMap(wfFile.Inputs),
				Outputs:          database.JSONMap(wfFile.Outputs),
				ExpectedOutcome:  database.JSONMap(wfFile.ExpectedOutcome),
				WorkflowMetadata: database.JSONMap(wfFile.Metadata),
				Description:      wfFile.Description,
				Tags:             database.StringArray(wfFile.Tags),
				LastChangeSource: "project_files_resync",
			}

			existing, getErr := h.repo.GetWorkflow(ctx, workflowID)
			switch {
			case getErr == nil && existing != nil:
				needsUpdate := existing.Name != next.Name ||
					existing.Description != next.Description ||
					existing.FolderPath != next.FolderPath ||
					existing.WorkflowType != next.WorkflowType ||
					!reflect.DeepEqual([]string(existing.Tags), []string(next.Tags)) ||
					!reflect.DeepEqual(existing.FlowDefinition, next.FlowDefinition) ||
					!reflect.DeepEqual(existing.Inputs, next.Inputs) ||
					!reflect.DeepEqual(existing.Outputs, next.Outputs) ||
					!reflect.DeepEqual(existing.ExpectedOutcome, next.ExpectedOutcome) ||
					!reflect.DeepEqual(existing.WorkflowMetadata, next.WorkflowMetadata)
				if needsUpdate {
					next.Version = existing.Version + 1
					if err := h.repo.UpdateWorkflow(ctx, next); err != nil {
						return err
					}
				}
			case errors.Is(getErr, database.ErrNotFound):
				if err := h.repo.CreateWorkflow(ctx, next); err != nil {
					return err
				}
			default:
				return getErr
			}

			entry := &database.ProjectEntry{
				ProjectID:  projectID,
				Path:       rel,
				Kind:       database.ProjectEntryKindWorkflowFile,
				WorkflowID: &workflowID,
				Metadata:   database.JSONMap{"workflowType": next.WorkflowType},
			}
			if err := h.repo.UpsertProjectEntry(ctx, entry); err != nil {
				return err
			}

			entriesIndexed++
			workflowsIndexed++
			return nil
		}

		info, statErr := os.Stat(abs)
		if statErr != nil {
			return statErr
		}
		meta := database.JSONMap{"sizeBytes": info.Size()}
		entry := &database.ProjectEntry{
			ProjectID: projectID,
			Path:      rel,
			Kind:      database.ProjectEntryKindAssetFile,
			Metadata:  meta,
		}
		if err := h.repo.UpsertProjectEntry(ctx, entry); err != nil {
			return err
		}
		entriesIndexed++
		assetsIndexed++
		return nil
	})
	if walkErr != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "walk_project_root"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, ResyncProjectFilesResponse{
		ProjectID:        projectID.String(),
		ProjectRoot:      project.FolderPath,
		EntriesIndexed:   entriesIndexed,
		WorkflowsIndexed: workflowsIndexed,
		AssetsIndexed:    assetsIndexed,
	})
}
