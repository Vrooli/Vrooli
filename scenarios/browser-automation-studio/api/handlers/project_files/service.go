package project_files //nolint:revive // Package name mirrors the public project_files proto and REST domain.

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	project_filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/project_files"
)

// service implements project_filesconnect.ProjectFilesServiceHandler.
type service struct {
	deps Deps
}

// ---------------------------------------------------------------------------
// Path helpers (mirror the legacy package-level helpers; unit-tested below).
// ---------------------------------------------------------------------------

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

func safeJoinProjectPath(projectRoot, relPath string) (string, error) {
	projectRoot = filepath.Clean(strings.TrimSpace(projectRoot))
	if projectRoot == "" || projectRoot == "." {
		return "", errors.New("invalid project root")
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return "", errInvalidPath
	}
	abs := filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(relPath)))
	rootWithSep := projectRoot + string(filepath.Separator)
	if abs != projectRoot && !strings.HasPrefix(abs, rootWithSep) {
		return "", errPathEscapesRoot
	}
	return abs, nil
}

func workflowFolderPathFromRelPath(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(relPath)))
	if dir == "." || dir == "" {
		return "/"
	}
	return "/" + strings.TrimPrefix(dir, "/")
}

func parseProjectID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, errInvalidProjectID
	}
	return id, nil
}

func (s *service) loadProject(ctx context.Context, rawID string) (uuid.UUID, *database.ProjectIndex, error) {
	id, err := parseProjectID(rawID)
	if err != nil {
		return uuid.Nil, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.deps.Repo.GetProject(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return uuid.Nil, nil, connect.NewError(connect.CodeNotFound, errProjectNotFound)
		}
		s.deps.Logger.WithError(err).Error("project_files.GetProject failed")
		return uuid.Nil, nil, connect.NewError(connect.CodeInternal, err)
	}
	return id, project, nil
}

// ---------------------------------------------------------------------------
// GetProjectFileTree
// ---------------------------------------------------------------------------

func (s *service) GetProjectFileTree(
	ctx context.Context,
	req *connect.Request[project_filesv1.GetProjectFileTreeRequest],
) (*connect.Response[project_filesv1.GetProjectFileTreeResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	projectID, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}

	_ = s.deps.Catalog.SyncProjectWorkflows(ctx, projectID)

	workflows, listErr := s.deps.Repo.ListWorkflowsByProject(ctx, projectID, 10000, 0)
	if listErr != nil && !errors.Is(listErr, database.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, listErr)
	}

	entries := make([]*project_filesv1.ProjectEntry, 0, len(workflows)+4)
	folders := map[string]struct{}{}

	if dirEntries, readErr := s.deps.FS.ReadDir(project.FolderPath); readErr == nil {
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
		filePath := filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(wf.FilePath), "/"))
		idStr := wf.ID.String()
		meta, _ := structpb.NewStruct(map[string]any{
			"folder_path": wf.FolderPath,
			"version":     wf.Version,
		})
		entries = append(entries, &project_filesv1.ProjectEntry{
			Id:         "wf:" + idStr,
			ProjectId:  projectID.String(),
			Path:       filePath,
			Kind:       project_filesv1.ProjectEntryKind_PROJECT_ENTRY_KIND_WORKFLOW_FILE,
			WorkflowId: idStr,
			Metadata:   meta,
		})
		addParentFolders(folders, filePath)
	}

	assets, _ := s.deps.Repo.ListAssetsByProject(ctx, projectID, 10000, 0)
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		meta, _ := structpb.NewStruct(map[string]any{
			"sizeBytes": asset.FileSize,
			"mimeType":  asset.MimeType,
		})
		entries = append(entries, &project_filesv1.ProjectEntry{
			Id:        "asset:" + asset.FilePath,
			ProjectId: projectID.String(),
			Path:      asset.FilePath,
			Kind:      project_filesv1.ProjectEntryKind_PROJECT_ENTRY_KIND_ASSET_FILE,
			Metadata:  meta,
		})
		addParentFolders(folders, asset.FilePath)
	}

	folderList := make([]string, 0, len(folders))
	for f := range folders {
		folderList = append(folderList, f)
	}
	sort.Strings(folderList)
	for _, f := range folderList {
		entries = append(entries, &project_filesv1.ProjectEntry{
			Id:        "folder:" + f,
			ProjectId: projectID.String(),
			Path:      f,
			Kind:      project_filesv1.ProjectEntryKind_PROJECT_ENTRY_KIND_FOLDER,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].GetKind() != entries[j].GetKind() {
			return entries[i].GetKind() < entries[j].GetKind()
		}
		return entries[i].GetPath() < entries[j].GetPath()
	})

	return connect.NewResponse(&project_filesv1.GetProjectFileTreeResponse{Entries: entries}), nil
}

func addParentFolders(folders map[string]struct{}, path string) {
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(path)))
	for dir != "." && dir != "" && dir != "/" {
		folders[dir] = struct{}{}
		next := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
		if next == dir {
			break
		}
		dir = next
	}
}

// ---------------------------------------------------------------------------
// ReadProjectFile
// ---------------------------------------------------------------------------

func (s *service) ReadProjectFile(
	ctx context.Context,
	req *connect.Request[project_filesv1.ReadProjectFileRequest],
) (*connect.Response[project_filesv1.ReadProjectFileResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	_, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	relPath, ok := normalizeProjectRelPath(req.Msg.GetPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidPath)
	}
	if !strings.HasSuffix(strings.ToLower(relPath), ".json") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errOnlyJSONReadable)
	}
	abs, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, joinErr)
	}
	snapshot, readErr := workflowservice.ReadWorkflowSummaryFile(ctx, project, abs)
	if readErr != nil || snapshot == nil || snapshot.Workflow == nil {
		return nil, connect.NewError(connect.CodeNotFound, errFileNotFound)
	}
	return connect.NewResponse(&project_filesv1.ReadProjectFileResponse{Workflow: snapshot.Workflow}), nil
}

// ---------------------------------------------------------------------------
// MkdirProjectPath
// ---------------------------------------------------------------------------

func (s *service) MkdirProjectPath(
	ctx context.Context,
	req *connect.Request[project_filesv1.MkdirProjectPathRequest],
) (*connect.Response[project_filesv1.MkdirProjectPathResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	_, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	relPath, ok := normalizeProjectRelPath(req.Msg.GetPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidPath)
	}
	abs, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, joinErr)
	}
	if err := s.deps.FS.MkdirAll(abs, 0o755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&project_filesv1.MkdirProjectPathResponse{
		Path:   relPath,
		Status: "created",
	}), nil
}

// ---------------------------------------------------------------------------
// WriteProjectWorkflowFile
// ---------------------------------------------------------------------------

func (s *service) WriteProjectWorkflowFile(
	ctx context.Context,
	req *connect.Request[project_filesv1.WriteProjectWorkflowFileRequest],
) (*connect.Response[project_filesv1.WriteProjectWorkflowFileResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	projectID, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	relPath, ok := normalizeProjectRelPath(req.Msg.GetPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidPath)
	}
	if !strings.HasSuffix(strings.ToLower(filepath.Base(filepath.FromSlash(relPath))), ".json") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errWorkflowFileExt)
	}

	wf := req.Msg.GetWorkflow()
	folderPath := workflowFolderPathFromRelPath(relPath)
	name := strings.TrimSpace(wf.GetName())
	if name == "" {
		base := filepath.Base(filepath.FromSlash(relPath))
		name = strings.TrimSuffix(base, ".json")
		if name == "" {
			name = "workflow"
		}
	}

	flow := wf.GetFlowDefinition().AsMap()
	meta := wf.GetMetadata().AsMap()
	settings := wf.GetSettings().AsMap()
	def, defErr := workflowservice.BuildFlowDefinitionV2ForWrite(flow, meta, settings)
	if defErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, defErr)
	}

	preferredRel := filepath.ToSlash(relPath)
	absExisting := filepath.Join(project.FolderPath, filepath.FromSlash(preferredRel))
	if _, statErr := s.deps.FS.Stat(absExisting); statErr == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errFileExists)
	}

	now := autocontracts.NowTimestamp()
	workflowID := uuid.New()
	summary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		ProjectId:      projectID.String(),
		Name:           name,
		FolderPath:     folderPath,
		Description:    strings.TrimSpace(wf.GetDescription()),
		Tags:           append([]string(nil), wf.GetTags()...),
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
		FlowDefinition: def,
	}

	if _, _, err := workflowservice.WriteWorkflowSummaryFile(project, summary, preferredRel); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	index := &database.WorkflowIndex{
		ID:         workflowID,
		ProjectID:  &projectID,
		Name:       name,
		FolderPath: folderPath,
		FilePath:   preferredRel,
		Version:    1,
	}
	if err := s.deps.Repo.CreateWorkflow(ctx, index); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	_ = s.deps.Catalog.SyncProjectWorkflows(ctx, projectID)

	return connect.NewResponse(&project_filesv1.WriteProjectWorkflowFileResponse{
		Path:       relPath,
		WorkflowId: workflowID.String(),
		Warnings:   []string{},
	}), nil
}

// ---------------------------------------------------------------------------
// MoveProjectFile
// ---------------------------------------------------------------------------

func (s *service) MoveProjectFile(
	ctx context.Context,
	req *connect.Request[project_filesv1.MoveProjectFileRequest],
) (*connect.Response[project_filesv1.MoveProjectFileResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	projectID, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	fromPath, ok := normalizeProjectRelPath(req.Msg.GetFromPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid from_path"))
	}
	toPath, ok := normalizeProjectRelPath(req.Msg.GetToPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid to_path"))
	}

	fromAbs, joinErr := safeJoinProjectPath(project.FolderPath, fromPath)
	if joinErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, joinErr)
	}
	toAbs, joinErr := safeJoinProjectPath(project.FolderPath, toPath)
	if joinErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, joinErr)
	}
	if err := s.deps.FS.MkdirAll(filepath.Dir(toAbs), 0o755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.deps.FS.Rename(fromAbs, toAbs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	_ = s.deps.Catalog.SyncProjectWorkflows(ctx, projectID)
	return connect.NewResponse(&project_filesv1.MoveProjectFileResponse{Status: "moved"}), nil
}

// ---------------------------------------------------------------------------
// DeleteProjectFile
// ---------------------------------------------------------------------------

func (s *service) DeleteProjectFile(
	ctx context.Context,
	req *connect.Request[project_filesv1.DeleteProjectFileRequest],
) (*connect.Response[project_filesv1.DeleteProjectFileResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	projectID, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	relPath, ok := normalizeProjectRelPath(req.Msg.GetPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidPath)
	}
	abs, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, joinErr)
	}
	if err := s.deps.FS.RemoveAll(abs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	_ = s.deps.Catalog.SyncProjectWorkflows(ctx, projectID)
	return connect.NewResponse(&project_filesv1.DeleteProjectFileResponse{Status: "deleted"}), nil
}

// ---------------------------------------------------------------------------
// ResyncProjectFiles
// ---------------------------------------------------------------------------

func (s *service) ResyncProjectFiles(
	ctx context.Context,
	req *connect.Request[project_filesv1.ResyncProjectFilesRequest],
) (*connect.Response[project_filesv1.ResyncProjectFilesResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	projectID, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	if err := s.deps.Catalog.SyncProjectWorkflows(ctx, projectID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	workflows, _ := s.deps.Repo.ListWorkflowsByProject(ctx, projectID, 10000, 0)
	assets, _ := s.deps.Repo.ListAssetsByProject(ctx, projectID, 10000, 0)
	return connect.NewResponse(&project_filesv1.ResyncProjectFilesResponse{
		ProjectId:        projectID.String(),
		ProjectRoot:      project.FolderPath,
		EntriesIndexed:   int32(len(workflows) + len(assets)),
		WorkflowsIndexed: int32(len(workflows)),
		AssetsIndexed:    int32(len(assets)),
	}), nil
}

// ---------------------------------------------------------------------------
// RevealProjectPath
// ---------------------------------------------------------------------------

func (s *service) RevealProjectPath(
	ctx context.Context,
	req *connect.Request[project_filesv1.RevealProjectPathRequest],
) (*connect.Response[project_filesv1.RevealProjectPathResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	_, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	relPath, ok := normalizeProjectRelPath(req.Msg.GetPath())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidPath)
	}
	abs, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, joinErr)
	}
	info, statErr := s.deps.FS.Stat(abs)
	if statErr != nil {
		return nil, connect.NewError(connect.CodeNotFound, errFileNotFound)
	}
	if info.IsDir() {
		if err := s.deps.OS.OpenFolder(abs); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&project_filesv1.RevealProjectPathResponse{
			Status: "opened",
			Path:   abs,
		}), nil
	}
	if err := s.deps.OS.RevealInFileManager(abs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&project_filesv1.RevealProjectPathResponse{
		Status: "revealed",
		Path:   abs,
	}), nil
}

// ---------------------------------------------------------------------------
// OpenProjectFolder
// ---------------------------------------------------------------------------

func (s *service) OpenProjectFolder(
	ctx context.Context,
	req *connect.Request[project_filesv1.OpenProjectFolderRequest],
) (*connect.Response[project_filesv1.OpenProjectFolderResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	_, project, err := s.loadProject(ctx, req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}
	folderPath := strings.TrimSpace(project.FolderPath)
	if folderPath == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errEmptyFolderPath)
	}
	info, statErr := s.deps.FS.Stat(folderPath)
	if statErr != nil || !info.IsDir() {
		return nil, connect.NewError(connect.CodeNotFound, errFileNotFound)
	}
	if err := s.deps.OS.OpenFolder(folderPath); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&project_filesv1.OpenProjectFolderResponse{
		Status: "opened",
		Path:   folderPath,
	}), nil
}
