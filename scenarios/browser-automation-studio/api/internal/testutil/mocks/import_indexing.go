package mocks

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

// ImportDirectoryScanner is the canonical fake for tests that exercise import
// usecases through shared.DirectoryScanner. It keeps an in-memory filesystem
// model with explicit files and directories.
type ImportDirectoryScanner struct {
	Files       map[string][]byte
	Directories map[string][]shared.FileEntry
	Patterns    map[string][]shared.FileEntry
}

func NewImportDirectoryScanner() *ImportDirectoryScanner {
	return &ImportDirectoryScanner{
		Files:       make(map[string][]byte),
		Directories: make(map[string][]shared.FileEntry),
		Patterns:    make(map[string][]shared.FileEntry),
	}
}

func (s *ImportDirectoryScanner) PutFile(path string, content []byte) {
	s.Files[path] = content
}

func (s *ImportDirectoryScanner) PutDirectory(path string, entries []shared.FileEntry) {
	s.Directories[path] = entries
}

func (s *ImportDirectoryScanner) ScanDirectory(_ context.Context, path string) ([]shared.FileEntry, error) {
	if entries, ok := s.Directories[path]; ok {
		return entries, nil
	}
	return []shared.FileEntry{}, nil
}

func (s *ImportDirectoryScanner) ScanForPattern(_ context.Context, root string, pattern string, _ int) ([]shared.FileEntry, error) {
	if entries, ok := s.Patterns[root+"|"+pattern]; ok {
		return entries, nil
	}
	return []shared.FileEntry{}, nil
}

func (s *ImportDirectoryScanner) ReadFile(_ context.Context, path string) ([]byte, error) {
	if content, ok := s.Files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (s *ImportDirectoryScanner) WriteFile(_ context.Context, path string, content []byte, _ os.FileMode) error {
	s.Files[path] = content
	return nil
}

func (s *ImportDirectoryScanner) CopyFile(_ context.Context, src, dst string) error {
	if content, ok := s.Files[src]; ok {
		s.Files[dst] = content
		return nil
	}
	return os.ErrNotExist
}

func (s *ImportDirectoryScanner) Exists(_ context.Context, path string) (bool, error) {
	if _, ok := s.Files[path]; ok {
		return true, nil
	}
	if _, ok := s.Directories[path]; ok {
		return true, nil
	}
	return false, nil
}

func (s *ImportDirectoryScanner) IsDir(_ context.Context, path string) (bool, error) {
	_, ok := s.Directories[path]
	return ok, nil
}

func (s *ImportDirectoryScanner) Stat(_ context.Context, _ string) (os.FileInfo, error) {
	return nil, nil
}

// ImportWorkflowIndexer is the canonical fake for tests that exercise import
// usecases through shared.WorkflowIndexer.
type ImportWorkflowIndexer struct {
	Workflows map[uuid.UUID]*shared.WorkflowIndexData
}

func NewImportWorkflowIndexer() *ImportWorkflowIndexer {
	return &ImportWorkflowIndexer{Workflows: make(map[uuid.UUID]*shared.WorkflowIndexData)}
}

func (i *ImportWorkflowIndexer) CreateWorkflowIndex(_ context.Context, _ uuid.UUID, workflow *shared.WorkflowIndexData) error {
	i.Workflows[workflow.ID] = workflow
	return nil
}

func (i *ImportWorkflowIndexer) GetWorkflowByFilePath(_ context.Context, projectID uuid.UUID, filePath string) (*shared.WorkflowIndexData, error) {
	for _, workflow := range i.Workflows {
		if workflow.FilePath == filePath && workflow.ProjectID == projectID {
			return workflow, nil
		}
	}
	return nil, nil
}

func (i *ImportWorkflowIndexer) GetWorkflowByID(_ context.Context, id uuid.UUID) (*shared.WorkflowIndexData, error) {
	if workflow, ok := i.Workflows[id]; ok {
		return workflow, nil
	}
	return nil, nil
}

func (i *ImportWorkflowIndexer) UpdateWorkflowIndex(_ context.Context, workflow *shared.WorkflowIndexData) error {
	i.Workflows[workflow.ID] = workflow
	return nil
}

func (i *ImportWorkflowIndexer) ListWorkflowsByProject(_ context.Context, projectID uuid.UUID) ([]*shared.WorkflowIndexData, error) {
	var workflows []*shared.WorkflowIndexData
	for _, workflow := range i.Workflows {
		if workflow.ProjectID == projectID {
			workflows = append(workflows, workflow)
		}
	}
	return workflows, nil
}

func (i *ImportWorkflowIndexer) DeleteWorkflowIndex(_ context.Context, id uuid.UUID) error {
	delete(i.Workflows, id)
	return nil
}

// ImportProjectIndexer is the canonical fake for tests that exercise import
// usecases through shared.ProjectIndexer.
type ImportProjectIndexer struct {
	Projects map[uuid.UUID]*shared.ProjectIndexData
}

func NewImportProjectIndexer() *ImportProjectIndexer {
	return &ImportProjectIndexer{Projects: make(map[uuid.UUID]*shared.ProjectIndexData)}
}

func (i *ImportProjectIndexer) PutProject(project *shared.ProjectIndexData) {
	i.Projects[project.ID] = project
}

func (i *ImportProjectIndexer) GetProjectByID(_ context.Context, id uuid.UUID) (*shared.ProjectIndexData, error) {
	if project, ok := i.Projects[id]; ok {
		return project, nil
	}
	return nil, os.ErrNotExist
}

func (i *ImportProjectIndexer) GetProjectByFolderPath(_ context.Context, folderPath string) (*shared.ProjectIndexData, error) {
	for _, project := range i.Projects {
		if project.FolderPath == folderPath {
			return project, nil
		}
	}
	return nil, nil
}

func (i *ImportProjectIndexer) ListProjects(_ context.Context, _ int, _ int) ([]*shared.ProjectIndexData, error) {
	projects := make([]*shared.ProjectIndexData, 0, len(i.Projects))
	for _, project := range i.Projects {
		projects = append(projects, project)
	}
	return projects, nil
}

func (i *ImportProjectIndexer) CreateProject(_ context.Context, project *shared.ProjectIndexData) error {
	i.Projects[project.ID] = project
	return nil
}

var (
	_ shared.DirectoryScanner = (*ImportDirectoryScanner)(nil)
	_ shared.WorkflowIndexer  = (*ImportWorkflowIndexer)(nil)
	_ shared.ProjectIndexer   = (*ImportProjectIndexer)(nil)
)
