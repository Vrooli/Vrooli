// Package project_files hosts the BAS ProjectFilesService Connect-RPC handler.
//
// ProjectFilesService owns the read/write surface for files living inside a
// project folder: workflow JSON files, indexed assets, folder tree, OS
// reveal/open. Binary file streaming (GET /projects/{id}/files/*) is
// intentionally NOT a Connect RPC and remains a RESTException (see
// docs/internal/REST_EXCEPTIONS.md).
package project_files //nolint:revive // domain name matches REST path + proto package

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/database"
	project_filesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/project_files/project_filesconnect"
)

// ProjectRepo is the narrow seam onto database.Repository the project_files
// handler uses. Tests substitute an in-memory fake.
type ProjectRepo interface {
	GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error)
	ListAssetsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.AssetIndex, error)
	CreateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error
}

// Catalog is the narrow seam onto workflow.CatalogService for project file
// reconciliation.
type Catalog interface {
	SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error
}

// FileSystem isolates filesystem mutations so tests can swap a fake.
type FileSystem interface {
	Stat(path string) (FileInfo, error)
	MkdirAll(path string, perm uint32) error
	Rename(oldPath, newPath string) error
	RemoveAll(path string) error
	ReadDir(path string) ([]DirEntry, error)
}

// FileInfo is the subset of os.FileInfo the handler uses.
type FileInfo interface {
	IsDir() bool
}

// DirEntry is the subset of os.DirEntry the handler uses.
type DirEntry interface {
	Name() string
	IsDir() bool
}

// OSIntegration opens/reveals paths in the system file manager.
type OSIntegration interface {
	OpenFolder(path string) error
	RevealInFileManager(path string) error
}

// Deps wires the project_files handler.
type Deps struct {
	Repo    ProjectRepo
	Catalog Catalog
	FS      FileSystem
	OS      OSIntegration
	Logger  *logrus.Logger
}

// Module builds the ProjectFilesService Connect handler and returns it
// wrapped in a connectx.ServiceMount ready for connectx.RegisterChi.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("project_files.Module requires Deps.Logger")
	}
	if d.Repo == nil {
		panic("project_files.Module requires Deps.Repo")
	}
	if d.Catalog == nil {
		panic("project_files.Module requires Deps.Catalog")
	}
	if d.FS == nil {
		d.FS = defaultFS{}
	}
	if d.OS == nil {
		d.OS = defaultOS{}
	}
	path, handler := project_filesconnect.NewProjectFilesServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
