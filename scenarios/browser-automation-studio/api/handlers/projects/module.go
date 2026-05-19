// Package projects hosts the BAS ProjectsService Connect-RPC handler.
//
// ProjectsService owns project CRUD plus the small set of project-scoped
// bulk operations on workflows (list, bulk-delete, execute-all). The on-disk
// file surface (workflow JSON files, asset tree, OS reveal/open) lives in a
// separate ProjectFilesService handler.
package projects

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects/projectsconnect"
)

// Catalog is the narrow seam onto workflow.CatalogService that
// ProjectsService needs. Tests substitute a fake.
type Catalog interface {
	CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error
	GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)
	GetProjectByName(ctx context.Context, name string) (*database.ProjectIndex, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.ProjectIndex, error)
	UpdateProject(ctx context.Context, project *database.ProjectIndex, description string) error
	DeleteProject(ctx context.Context, id uuid.UUID, deleteFiles bool) error
	ListProjects(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error)
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (*database.ProjectStats, error)
	GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error)
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error)
	DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
	HydrateProject(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error)
	ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error)
}

// Executor is the narrow seam onto workflow.ExecutionService used by
// ExecuteAllProjectWorkflows.
type Executor interface {
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error)
}

// PathPreparer validates a caller-supplied folder path and prepares the
// directory on disk. The default implementation calls into
// internal/paths.ValidateAndPrepareFolderPath; tests inject a fake.
type PathPreparer interface {
	Prepare(folderPath string) (string, error)
	MakeAll(absPath string, perm uint32) error
}

// Deps wires the projects handler.
type Deps struct {
	Catalog  Catalog
	Executor Executor
	Paths    PathPreparer
	Logger   *logrus.Logger
}

// Module builds the ProjectsService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("projects.Module requires Deps.Logger")
	}
	if d.Catalog == nil {
		panic("projects.Module requires Deps.Catalog")
	}
	if d.Executor == nil {
		panic("projects.Module requires Deps.Executor")
	}
	if d.Paths == nil {
		d.Paths = defaultPaths{log: d.Logger}
	}
	path, handler := projectsconnect.NewProjectsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
