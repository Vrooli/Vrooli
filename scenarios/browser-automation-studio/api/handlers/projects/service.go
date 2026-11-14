package projects

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
)

// Field length limits (mirror database schema constraints).
const (
	maxNameLength        = 255
	maxFolderPathLength  = 500
	maxBulkOperationSize = 100
	defaultListLimit     = 0 // 0 = repo default
)

// service implements projectsconnect.ProjectsServiceHandler.
type service struct {
	deps Deps
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *service) log() *logrus.Logger { return s.deps.Logger }

func parseProjectID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, errInvalidProjectID
	}
	return id, nil
}

func validateLen(field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}

// normalizeProjectRelPath mirrors the legacy helper from handlers/project_files.go.
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

func (s *service) hydrateWithStats(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, *basprojects.ProjectStats, error) {
	pb, err := s.deps.Catalog.HydrateProject(ctx, project)
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hydrate project: %w", err))
	}
	stats, err := s.deps.Catalog.GetProjectStats(ctx, project.ID)
	if err != nil {
		s.log().WithError(err).WithField("project_id", project.ID).Warn("failed to get project stats")
		stats = &database.ProjectStats{ProjectID: project.ID}
	}
	statsProto := &basprojects.ProjectStats{
		ProjectId:      project.ID.String(),
		WorkflowCount:  int32(stats.WorkflowCount),
		ExecutionCount: int32(stats.ExecutionCount),
		LastExecution:  autocontracts.TimePtrToTimestamp(stats.LastExecution),
	}
	return pb, statsProto, nil
}

func (s *service) applyPreset(project *database.ProjectIndex, preset basprojects.PresetKind, presetPaths []string) error {
	if project == nil || project.ID == uuid.Nil {
		return errors.New("project id missing")
	}
	if strings.TrimSpace(project.FolderPath) == "" {
		return errors.New("project folder path missing")
	}

	var folders []string
	switch preset {
	case basprojects.PresetKind_PRESET_KIND_UNSPECIFIED, basprojects.PresetKind_PRESET_KIND_EMPTY:
		return nil
	case basprojects.PresetKind_PRESET_KIND_RECOMMENDED:
		folders = []string{"actions", "flows", "cases", "assets"}
	case basprojects.PresetKind_PRESET_KIND_CUSTOM:
		folders = presetPaths
	default:
		return errUnknownPreset
	}

	for _, folder := range folders {
		rel, ok := normalizeProjectRelPath(folder)
		if !ok {
			return fmt.Errorf("%w: %q", errInvalidPresetPath, folder)
		}
		abs := filepath.Join(project.FolderPath, filepath.FromSlash(rel))
		if err := s.deps.Paths.MakeAll(abs, 0o755); err != nil {
			return fmt.Errorf("%w %q: %v", errProjectPresetFolderError, rel, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// CreateProject
// ---------------------------------------------------------------------------

func (s *service) CreateProject(
	ctx context.Context,
	req *connect.Request[basprojects.CreateProjectRequest],
) (*connect.Response[basprojects.CreateProjectResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	in := req.Msg
	name := strings.TrimSpace(in.GetName())
	folderPath := strings.TrimSpace(in.GetFolderPath())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNameRequired)
	}
	if folderPath == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errFolderPathRequired)
	}
	if err := validateLen("name", name, maxNameLength); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateLen("folder_path", folderPath, maxFolderPathLength); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	absPath, err := s.deps.Paths.Prepare(folderPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("folder_path: %w", err))
	}

	existing, err := s.deps.Catalog.GetProjectByName(ctx, name)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		s.log().WithError(err).WithField("name", name).Error("get project by name failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existing != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errProjectAlreadyExists)
	}

	existingByPath, err := s.deps.Catalog.GetProjectByFolderPath(ctx, absPath)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		s.log().WithError(err).WithField("folder_path", absPath).Error("get project by folder path failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existingByPath != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errProjectFolderTaken)
	}

	project := &database.ProjectIndex{Name: name, FolderPath: absPath}
	if err := s.deps.Catalog.CreateProject(ctx, project, in.GetDescription()); err != nil {
		s.log().WithError(err).Error("create project failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if in.GetPreset() != basprojects.PresetKind_PRESET_KIND_UNSPECIFIED &&
		in.GetPreset() != basprojects.PresetKind_PRESET_KIND_EMPTY {
		if err := s.applyPreset(project, in.GetPreset(), in.GetPresetPaths()); err != nil {
			s.log().WithError(err).WithFields(logrus.Fields{
				"project_id": project.ID.String(),
				"preset":     in.GetPreset().String(),
			}).Error("apply project preset failed")
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	pb, stats, err := s.hydrateWithStats(ctx, project)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&basprojects.CreateProjectResponse{Project: pb, Stats: stats}), nil
}

// ---------------------------------------------------------------------------
// ListProjects
// ---------------------------------------------------------------------------

func (s *service) ListProjects(
	ctx context.Context,
	req *connect.Request[basprojects.ListProjectsRequest],
) (*connect.Response[basprojects.ListProjectsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	limit := int(req.Msg.GetLimit())
	offset := int(req.Msg.GetOffset())
	if limit < 0 {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}

	projects, err := s.deps.Catalog.ListProjects(ctx, limit, offset)
	if err != nil {
		s.log().WithError(err).Error("list projects failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	projectIDs := make([]uuid.UUID, 0, len(projects))
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
	}

	statsByProject, err := s.deps.Catalog.GetProjectsStats(ctx, projectIDs)
	if err != nil {
		s.log().WithError(err).Error("bulk get project stats failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*basprojects.ProjectWithStats, 0, len(projects))
	for _, project := range projects {
		pb, hydErr := s.deps.Catalog.HydrateProject(ctx, project)
		if hydErr != nil {
			s.log().WithError(hydErr).WithField("project_id", project.ID).Warn("hydrate project failed; skipping")
			continue
		}
		stats := statsByProject[project.ID]
		statsProto := &basprojects.ProjectStats{ProjectId: project.ID.String()}
		if stats != nil {
			statsProto.WorkflowCount = int32(stats.WorkflowCount)
			statsProto.ExecutionCount = int32(stats.ExecutionCount)
			statsProto.LastExecution = autocontracts.TimePtrToTimestamp(stats.LastExecution)
		}
		items = append(items, &basprojects.ProjectWithStats{Project: pb, Stats: statsProto})
	}

	return connect.NewResponse(&basprojects.ListProjectsResponse{Projects: items}), nil
}

// ---------------------------------------------------------------------------
// GetProject
// ---------------------------------------------------------------------------

func (s *service) GetProject(
	ctx context.Context,
	req *connect.Request[basprojects.GetProjectRequest],
) (*connect.Response[basprojects.GetProjectResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	id, err := parseProjectID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.deps.Catalog.GetProject(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errProjectNotFound)
		}
		s.log().WithError(err).WithField("project_id", id).Error("get project failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pb, stats, err := s.hydrateWithStats(ctx, project)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&basprojects.GetProjectResponse{Project: pb, Stats: stats}), nil
}

// ---------------------------------------------------------------------------
// UpdateProject
// ---------------------------------------------------------------------------

func (s *service) UpdateProject(
	ctx context.Context,
	req *connect.Request[basprojects.UpdateProjectRequest],
) (*connect.Response[basprojects.UpdateProjectResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	in := req.Msg
	id, err := parseProjectID(in.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if name := strings.TrimSpace(in.GetName()); name != "" {
		if err := validateLen("name", name, maxNameLength); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	if folderPath := strings.TrimSpace(in.GetFolderPath()); folderPath != "" {
		if err := validateLen("folder_path", folderPath, maxFolderPathLength); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	project, err := s.deps.Catalog.GetProject(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errProjectNotFound)
		}
		s.log().WithError(err).WithField("project_id", id).Error("get project for update failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	currentProto, err := s.deps.Catalog.HydrateProject(ctx, project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hydrate project: %w", err))
	}

	if name := strings.TrimSpace(in.GetName()); name != "" {
		project.Name = name
	}
	description := currentProto.Description
	if d := in.GetDescription(); d != "" {
		description = d
	}
	if folderPath := strings.TrimSpace(in.GetFolderPath()); folderPath != "" {
		absPath, err := s.deps.Paths.Prepare(folderPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("folder_path: %w", err))
		}
		project.FolderPath = absPath
	}

	if err := s.deps.Catalog.UpdateProject(ctx, project, description); err != nil {
		s.log().WithError(err).WithField("project_id", id).Error("update project failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	updated, err := s.deps.Catalog.HydrateProject(ctx, project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hydrate project: %w", err))
	}
	return connect.NewResponse(&basprojects.UpdateProjectResponse{Project: updated}), nil
}

// ---------------------------------------------------------------------------
// DeleteProject
// ---------------------------------------------------------------------------

func (s *service) DeleteProject(
	ctx context.Context,
	req *connect.Request[basprojects.DeleteProjectRequest],
) (*connect.Response[basprojects.DeleteProjectResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	id, err := parseProjectID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	deleteFiles := req.Msg.GetDeleteFiles()

	if err := s.deps.Catalog.DeleteProject(ctx, id, deleteFiles); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errProjectNotFound)
		}
		s.log().WithError(err).WithFields(logrus.Fields{
			"project_id":   id,
			"delete_files": deleteFiles,
		}).Error("delete project failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&basprojects.DeleteProjectResponse{FilesDeleted: deleteFiles}), nil
}

// ---------------------------------------------------------------------------
// ListProjectWorkflows
// ---------------------------------------------------------------------------

func (s *service) ListProjectWorkflows(
	ctx context.Context,
	req *connect.Request[basprojects.ListProjectWorkflowsRequest],
) (*connect.Response[basapi.ListWorkflowsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	id, err := parseProjectID(req.Msg.GetProjectId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	limit := req.Msg.GetLimit()
	if limit <= 0 {
		limit = 100
	}
	offset := req.Msg.GetOffset()
	if offset < 0 {
		offset = 0
	}

	listReq := &basapi.ListWorkflowsRequest{
		ProjectId: proto.String(id.String()),
		Limit:     proto.Int32(limit),
		Offset:    proto.Int32(offset),
	}
	resp, err := s.deps.Catalog.ListWorkflows(ctx, listReq)
	if err != nil {
		s.log().WithError(err).WithField("project_id", id).Error("list project workflows failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// BulkDeleteProjectWorkflows
// ---------------------------------------------------------------------------

func (s *service) BulkDeleteProjectWorkflows(
	ctx context.Context,
	req *connect.Request[basprojects.BulkDeleteProjectWorkflowsRequest],
) (*connect.Response[basprojects.BulkDeleteProjectWorkflowsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	in := req.Msg
	id, err := parseProjectID(in.GetProjectId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if len(in.GetWorkflowIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errWorkflowIDsRequired)
	}
	if len(in.GetWorkflowIds()) > maxBulkOperationSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, errBulkOperationTooLarge)
	}

	workflowIDs := make([]uuid.UUID, 0, len(in.GetWorkflowIds()))
	for _, raw := range in.GetWorkflowIds() {
		wid, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %q", errInvalidWorkflowID, raw))
		}
		workflowIDs = append(workflowIDs, wid)
	}

	if err := s.deps.Catalog.DeleteProjectWorkflows(ctx, id, workflowIDs); err != nil {
		s.log().WithError(err).WithFields(logrus.Fields{
			"project_id": id,
			"count":      len(workflowIDs),
		}).Error("bulk delete project workflows failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	deletedIDs := make([]string, 0, len(workflowIDs))
	for _, wid := range workflowIDs {
		deletedIDs = append(deletedIDs, wid.String())
	}
	return connect.NewResponse(&basprojects.BulkDeleteProjectWorkflowsResponse{
		DeletedCount: int32(len(workflowIDs)),
		DeletedIds:   deletedIDs,
	}), nil
}

// ---------------------------------------------------------------------------
// ExecuteAllProjectWorkflows
// ---------------------------------------------------------------------------

func (s *service) ExecuteAllProjectWorkflows(
	ctx context.Context,
	req *connect.Request[basprojects.ExecuteAllProjectWorkflowsRequest],
) (*connect.Response[basprojects.ExecuteAllProjectWorkflowsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	id, err := parseProjectID(req.Msg.GetProjectId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	workflows, err := s.deps.Catalog.ListWorkflowsByProject(ctx, id, 1000, 0)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errProjectNotFound)
		}
		s.log().WithError(err).WithField("project_id", id).Error("list project workflows failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(workflows) == 0 {
		return connect.NewResponse(&basprojects.ExecuteAllProjectWorkflowsResponse{
			Message:    "No workflows found in project",
			Executions: nil,
		}), nil
	}

	results := make([]*basprojects.ProjectWorkflowExecutionResult, 0, len(workflows))
	for _, wf := range workflows {
		execution, err := s.deps.Executor.ExecuteWorkflow(ctx, wf.ID, map[string]any{})
		if err != nil {
			s.log().WithError(err).WithField("workflow_id", wf.ID).Warn("execute workflow in bulk operation failed")
			results = append(results, &basprojects.ProjectWorkflowExecutionResult{
				WorkflowId:   wf.ID.String(),
				WorkflowName: wf.Name,
				Status:       "failed",
				Error:        err.Error(),
			})
			continue
		}
		results = append(results, &basprojects.ProjectWorkflowExecutionResult{
			WorkflowId:   wf.ID.String(),
			WorkflowName: wf.Name,
			ExecutionId:  execution.ID.String(),
			Status:       execution.Status,
		})
	}

	return connect.NewResponse(&basprojects.ExecuteAllProjectWorkflowsResponse{
		Message:    fmt.Sprintf("Started execution for %d workflows", len(results)),
		Executions: results,
	}), nil
}
