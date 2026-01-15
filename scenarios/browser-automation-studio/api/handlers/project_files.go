package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

type ProjectFileTreeResponse struct {
	Entries []*ProjectEntry `json:"entries"`
}

type ProjectEntryKind string

const (
	ProjectEntryKindFolder       ProjectEntryKind = "folder"
	ProjectEntryKindWorkflowFile ProjectEntryKind = "workflow_file"
	ProjectEntryKindAssetFile    ProjectEntryKind = "asset_file"
)

type ProjectEntry struct {
	ID         string           `json:"id"`
	ProjectID  string           `json:"project_id"`
	Path       string           `json:"path"`
	Kind       ProjectEntryKind `json:"kind"`
	WorkflowID *string          `json:"workflow_id,omitempty"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
}

type MkdirProjectPathRequest struct {
	Path string `json:"path"`
}

type WriteProjectWorkflowFileRequest struct {
	Path     string                   `json:"path"`
	Workflow ProjectWorkflowFileWrite `json:"workflow"`
}

type ProjectWorkflowFileWrite struct {
	Name           string         `json:"name"`
	Type           string         `json:"type,omitempty"`
	Description    string         `json:"description,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	FlowDefinition map[string]any `json:"flow_definition"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Settings       map[string]any `json:"settings,omitempty"`
}

type MoveProjectFileRequest struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
}

type DeleteProjectFileRequest struct {
	Path string `json:"path"`
}

type RevealProjectPathRequest struct {
	Path string `json:"path"`
}

type ResyncProjectFilesResponse struct {
	ProjectID        string `json:"project_id"`
	ProjectRoot      string `json:"project_root"`
	EntriesIndexed   int    `json:"entries_indexed"`
	WorkflowsIndexed int    `json:"workflows_indexed"`
	AssetsIndexed    int    `json:"assets_indexed"`
}

func normalizeProjectRelPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" || raw == "." {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", false
	}
	return clean, true
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

func workflowFolderPathFromRelPath(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "workflows/")
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(relPath)))
	if dir == "." || dir == "" {
		return "/"
	}
	return "/" + strings.TrimPrefix(dir, "/")
}

func (h *Handler) GetProjectFileTree(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
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

	// Ensure DB index reflects filesystem edits.
	_ = h.catalogService.SyncProjectWorkflows(ctx, projectID)

	workflows, err := h.repo.ListWorkflowsByProject(ctx, projectID, 10000, 0)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows_by_project"}))
		return
	}

	entries := make([]*ProjectEntry, 0, len(workflows)+4)

	folders := map[string]struct{}{}

	// Discover all top-level directories in the project folder.
	if dirEntries, readErr := os.ReadDir(project.FolderPath); readErr == nil {
		for _, entry := range dirEntries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				folders[entry.Name()] = struct{}{}
			}
		}
	}

	for _, wf := range workflows {
		if wf == nil {
			continue
		}
		// Use the actual file path from the database (no more workflows/ prefix assumption)
		filePath := filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(wf.FilePath), "/"))
		idStr := wf.ID.String()
		entries = append(entries, &ProjectEntry{
			ID:         "wf:" + idStr,
			ProjectID:  projectID.String(),
			Path:       filePath,
			Kind:       ProjectEntryKindWorkflowFile,
			WorkflowID: &idStr,
			Metadata: map[string]any{
				"folder_path": wf.FolderPath,
				"version":     wf.Version,
			},
		})

		// Add parent directories to folder list
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(filePath)))
		for dir != "." && dir != "" && dir != "/" {
			folders[dir] = struct{}{}
			next := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
			if next == dir {
				break
			}
			dir = next
		}
	}

	// Get assets from database (indexed during sync)
	assets, assetsErr := h.repo.ListAssetsByProject(ctx, projectID, 10000, 0)
	if assetsErr == nil {
		for _, asset := range assets {
			if asset == nil {
				continue
			}
			entries = append(entries, &ProjectEntry{
				ID:        "asset:" + asset.FilePath,
				ProjectID: projectID.String(),
				Path:      asset.FilePath,
				Kind:      ProjectEntryKindAssetFile,
				Metadata: map[string]any{
					"sizeBytes": asset.FileSize,
					"mimeType":  asset.MimeType,
				},
			})

			// Add parent directories to folder list
			dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(asset.FilePath)))
			for dir != "." && dir != "" && dir != "/" {
				folders[dir] = struct{}{}
				next := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
				if next == dir {
					break
				}
				dir = next
			}
		}
	}

	folderList := make([]string, 0, len(folders))
	for folder := range folders {
		folderList = append(folderList, folder)
	}
	sort.Strings(folderList)
	for _, folder := range folderList {
		entries = append(entries, &ProjectEntry{
			ID:        "folder:" + folder,
			ProjectID: projectID.String(),
			Path:      folder,
			Kind:      ProjectEntryKindFolder,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Path < entries[j].Path
	})

	h.respondSuccess(w, http.StatusOK, ProjectFileTreeResponse{Entries: entries})
}

// ReadProjectFile handles GET /api/v1/projects/{id}/files/read?path=workflows/...
// Returns the canonical protojson workflow summary when the file is a workflow file.
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

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	abs, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}

	// Check if file is a workflow JSON file
	if !strings.HasSuffix(strings.ToLower(relPath), ".json") {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "only JSON workflow files are readable"}))
		return
	}

	snapshot, err := workflowservice.ReadWorkflowSummaryFile(ctx, project, abs)
	if err != nil {
		h.respondError(w, ErrProjectFileNotFound)
		return
	}
	if snapshot.Workflow == nil {
		h.respondError(w, ErrProjectFileNotFound)
		return
	}

	h.respondProto(w, http.StatusOK, snapshot.Workflow)
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

	abs, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "mkdir_project_path"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{"path": relPath, "status": "created"})
}

// WriteProjectWorkflowFile handles POST /api/v1/projects/{id}/files/write.
// This endpoint is proto-first: it persists a WorkflowSummary protojson file and syncs the DB index.
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
	if !strings.HasPrefix(filepath.ToSlash(relPath), "workflows/") {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "path must be under workflows/"}))
		return
	}
	if !strings.HasSuffix(strings.ToLower(filepath.Base(filepath.FromSlash(relPath))), ".workflow.json") {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "workflow files must end with .workflow.json"}))
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

	folderPath := workflowFolderPathFromRelPath(relPath)
	name := strings.TrimSpace(req.Workflow.Name)
	if name == "" {
		base := filepath.Base(filepath.FromSlash(relPath))
		name = strings.TrimSuffix(base, ".workflow.json")
		if name == "" {
			name = "workflow"
		}
	}

	flow := req.Workflow.FlowDefinition
	def, defErr := workflowservice.BuildFlowDefinitionV2ForWrite(flow, req.Workflow.Metadata, req.Workflow.Settings)
	if defErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": defErr.Error()}))
		return
	}

	preferredRel := strings.TrimPrefix(filepath.ToSlash(relPath), "workflows/")
	if _, statErr := os.Stat(filepath.Join(workflowservice.ProjectWorkflowsDir(project), filepath.FromSlash(preferredRel))); statErr == nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "file already exists"}))
		return
	}

	now := autocontracts.NowTimestamp()
	workflowID := uuid.New()
	summary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		ProjectId:      projectID.String(),
		Name:           name,
		FolderPath:     folderPath,
		Description:    strings.TrimSpace(req.Workflow.Description),
		Tags:           append([]string(nil), req.Workflow.Tags...),
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
		FlowDefinition: def,
	}

	if _, _, err := workflowservice.WriteWorkflowSummaryFile(project, summary, preferredRel); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "write_workflow_file"}))
		return
	}

	index := &database.WorkflowIndex{
		ID:         workflowID,
		ProjectID:  &projectID,
		Name:       name,
		FolderPath: folderPath,
		FilePath:   preferredRel,
		Version:    1,
	}
	if err := h.repo.CreateWorkflow(ctx, index); err != nil {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_workflow_index"}))
		return
	}

	_ = h.catalogService.SyncProjectWorkflows(ctx, projectID)

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"path":       relPath,
		"workflowId": workflowID.String(),
		"warnings":   []string{},
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

	fromAbs, err := safeJoinProjectPath(project.FolderPath, fromPath)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	toAbs, err := safeJoinProjectPath(project.FolderPath, toPath)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
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

	_ = h.catalogService.SyncProjectWorkflows(ctx, projectID)
	h.respondSuccess(w, http.StatusOK, map[string]any{"status": "moved"})
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

	abs, err := safeJoinProjectPath(project.FolderPath, relPath)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	if err := os.RemoveAll(abs); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "delete_project_file"}))
		return
	}

	_ = h.catalogService.SyncProjectWorkflows(ctx, projectID)
	h.respondSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
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

	if err := h.catalogService.SyncProjectWorkflows(ctx, projectID); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "sync_project_workflows"}))
		return
	}

	workflows, _ := h.repo.ListWorkflowsByProject(ctx, projectID, 10000, 0)
	assets, _ := h.repo.ListAssetsByProject(ctx, projectID, 10000, 0)
	assetsIndexed := len(assets)

	h.respondSuccess(w, http.StatusOK, ResyncProjectFilesResponse{
		ProjectID:        projectID.String(),
		ProjectRoot:      project.FolderPath,
		EntriesIndexed:   len(workflows) + assetsIndexed,
		WorkflowsIndexed: len(workflows),
		AssetsIndexed:    assetsIndexed,
	})
}

// RevealProjectPath handles POST /api/v1/projects/{id}/files/reveal.
// Opens the file manager and highlights the file when possible.
func (h *Handler) RevealProjectPath(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	var req RevealProjectPathRequest
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

	abs, err := safeJoinProjectPath(project.FolderPath, relPath)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	info, statErr := os.Stat(abs)
	if statErr != nil {
		h.respondError(w, ErrProjectFileNotFound)
		return
	}

	if info.IsDir() {
		if err := openFolder(abs); err != nil {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "open_project_folder"}))
			return
		}
		h.respondSuccess(w, http.StatusOK, map[string]any{
			"status": "opened",
			"path":   abs,
		})
		return
	}

	if err := revealInFileManager(abs); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "reveal_project_file"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "revealed",
		"path":   abs,
	})
}

// OpenProjectFolder handles POST /api/v1/projects/{id}/open-folder.
// Opens the project root folder in the system file manager.
func (h *Handler) OpenProjectFolder(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
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

	folderPath := strings.TrimSpace(project.FolderPath)
	if folderPath == "" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "project folder path is empty"}))
		return
	}

	if info, statErr := os.Stat(folderPath); statErr != nil || !info.IsDir() {
		h.respondError(w, ErrProjectFileNotFound)
		return
	}

	if err := openFolder(folderPath); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "open_project_root"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "opened",
		"path":   folderPath,
	})
}

// ServeProjectFile handles GET /api/v1/projects/{id}/files/*
// Serves any file from a project's folder (assets, workflows, etc.)
func (h *Handler) ServeProjectFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	// Extract file path from wildcard capture
	filePath := chi.URLParam(r, "*")
	if filePath == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "file_path"}))
		return
	}

	// Remove any leading slash and normalize
	filePath = strings.TrimPrefix(filePath, "/")
	relPath, ok := normalizeProjectRelPath(filePath)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid file path"}))
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

	// Safely join paths to prevent directory traversal
	absPath, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}

	// Check if file exists and is not a directory
	info, statErr := os.Stat(absPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			h.respondError(w, ErrProjectFileNotFound)
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stat_file"}))
		return
	}
	if info.IsDir() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "path is a directory, not a file"}))
		return
	}

	// Open the file
	file, openErr := os.Open(absPath)
	if openErr != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "open_file"}))
		return
	}
	defer file.Close()

	// Determine MIME type from extension
	ext := filepath.Ext(absPath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set response headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Cache-Control", "private, max-age=60")

	// Stream file to response
	if _, err := io.Copy(w, file); err != nil {
		h.log.WithError(err).WithField("path", relPath).Error("Failed to stream project file")
	}
}
