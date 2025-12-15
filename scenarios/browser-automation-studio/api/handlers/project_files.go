package handlers

import (
	"context"
	"errors"
	"net/http"
	"os"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	Path      string                 `json:"path"`
	Kind      ProjectEntryKind       `json:"kind"`
	WorkflowID *string               `json:"workflow_id,omitempty"`
	Metadata  map[string]any         `json:"metadata,omitempty"`
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

func workflowsDir(project *database.ProjectIndex) string {
	if project == nil {
		return "workflows"
	}
	root := strings.TrimSpace(project.FolderPath)
	if root == "" {
		return "workflows"
	}
	return filepath.Join(root, "workflows")
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
	_ = h.workflowCatalog.SyncProjectWorkflows(ctx, projectID)

	workflows, err := h.repo.ListWorkflowsByProject(ctx, projectID, 10000, 0)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows_by_project"}))
		return
	}

	entries := make([]*ProjectEntry, 0, len(workflows)+4)

	folders := map[string]struct{}{
		"workflows": {},
	}
	for _, wf := range workflows {
		if wf == nil {
			continue
		}
		filePath := filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(wf.FilePath), "/"))
		fullRel := filepath.ToSlash(filepath.Join("workflows", filepath.FromSlash(filePath)))
		idStr := wf.ID.String()
		entries = append(entries, &ProjectEntry{
			ID:         "wf:" + idStr,
			ProjectID:  projectID.String(),
			Path:       fullRel,
			Kind:       ProjectEntryKindWorkflowFile,
			WorkflowID: &idStr,
			Metadata: map[string]any{
				"folder_path": wf.FolderPath,
				"version":     wf.Version,
			},
		})

		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(fullRel)))
		for dir != "." && dir != "" && dir != "/" {
			folders[dir] = struct{}{}
			next := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
			if next == dir {
				break
			}
			dir = next
		}
	}

	// Optional: include assets directory if present.
	assetsRoot := filepath.Join(project.FolderPath, "assets")
	if info, statErr := os.Stat(assetsRoot); statErr == nil && info.IsDir() {
		folders["assets"] = struct{}{}
		_ = filepath.WalkDir(assetsRoot, func(abs string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				rel, relErr := filepath.Rel(project.FolderPath, abs)
				if relErr == nil {
					rel = filepath.ToSlash(rel)
					if rel != "." && rel != "" {
						folders[rel] = struct{}{}
					}
				}
				return nil
			}
			rel, relErr := filepath.Rel(project.FolderPath, abs)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			stat, statErr := os.Stat(abs)
			if statErr != nil {
				return nil
			}
			entries = append(entries, &ProjectEntry{
				ID:        "asset:" + rel,
				ProjectID: projectID.String(),
				Path:      rel,
				Kind:      ProjectEntryKindAssetFile,
				Metadata: map[string]any{
					"sizeBytes": stat.Size(),
				},
			})
			return nil
		})
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

	_ = os.MkdirAll(workflowsDir(project), 0o755)

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

	if !strings.HasPrefix(filepath.ToSlash(relPath), "workflows/") {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "only workflow files are readable"}))
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
	nodesAny, _ := flow["nodes"]
	edgesAny, _ := flow["edges"]
	nodes := workflowservice.ToInterfaceSlice(nodesAny)
	edges := workflowservice.ToInterfaceSlice(edgesAny)
	payload := map[string]any{
		"flow_definition": flow,
		"metadata":        req.Workflow.Metadata,
		"settings":        req.Workflow.Settings,
	}
	def, convErr := workflowservice.V1NodesEdgesToV2Definition(nodes, edges, payload)
	if convErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": convErr.Error()}))
		return
	}

	preferredRel := strings.TrimPrefix(filepath.ToSlash(relPath), "workflows/")
	if _, statErr := os.Stat(filepath.Join(workflowsDir(project), filepath.FromSlash(preferredRel))); statErr == nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "file already exists"}))
		return
	}

	now := timestamppb.New(time.Now().UTC())
	workflowID := uuid.New()
	summary := &basapi.WorkflowSummary{
		Id:          workflowID.String(),
		ProjectId:   projectID.String(),
		Name:        name,
		FolderPath:  folderPath,
		Description: strings.TrimSpace(req.Workflow.Description),
		Tags:        append([]string(nil), req.Workflow.Tags...),
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
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

	_ = h.workflowCatalog.SyncProjectWorkflows(ctx, projectID)

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

	_ = h.workflowCatalog.SyncProjectWorkflows(ctx, projectID)
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

	_ = h.workflowCatalog.SyncProjectWorkflows(ctx, projectID)
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

	if err := h.workflowCatalog.SyncProjectWorkflows(ctx, projectID); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "sync_project_workflows"}))
		return
	}

	workflows, _ := h.repo.ListWorkflowsByProject(ctx, projectID, 10000, 0)
	assetsIndexed := 0
	assetsRoot := filepath.Join(project.FolderPath, "assets")
	if info, statErr := os.Stat(assetsRoot); statErr == nil && info.IsDir() {
		_ = filepath.WalkDir(assetsRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			assetsIndexed++
			return nil
		})
	}

	h.respondSuccess(w, http.StatusOK, ResyncProjectFilesResponse{
		ProjectID:        projectID.String(),
		ProjectRoot:      project.FolderPath,
		EntriesIndexed:   len(workflows) + assetsIndexed,
		WorkflowsIndexed: len(workflows),
		AssetsIndexed:    assetsIndexed,
	})
}
