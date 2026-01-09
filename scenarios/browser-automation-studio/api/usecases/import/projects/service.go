package projects

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"google.golang.org/protobuf/encoding/protojson"
)

// Service handles project import business logic.
type Service struct {
	scanner   shared.DirectoryScanner
	projecter shared.ProjectIndexer
	syncer    shared.WorkflowSyncer
	log       *logrus.Logger
}

// NewService creates a new Service.
func NewService(
	scanner shared.DirectoryScanner,
	projecter shared.ProjectIndexer,
	syncer shared.WorkflowSyncer,
	log *logrus.Logger,
) *Service {
	return &Service{
		scanner:   scanner,
		projecter: projecter,
		syncer:    syncer,
		log:       log,
	}
}

// readProjectMetadata reads project metadata from .bas/project.json.
func (s *Service) readProjectMetadata(folderPath string) (*basprojects.Project, string) {
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

// hasWorkflowFiles checks if a folder contains workflow files.
func (s *Service) hasWorkflowFiles(folderPath string) bool {
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
		if shared.IsWorkflowFile(d.Name()) {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// InspectFolder inspects a project folder for import.
func (s *Service) InspectFolder(ctx context.Context, folderPath string) (*InspectProjectResponse, error) {
	if strings.TrimSpace(folderPath) == "" {
		return nil, errors.New("folder_path is required")
	}

	// Validate and normalize path
	absPath, err := shared.NormalizePath(folderPath)
	if err != nil {
		return nil, fmt.Errorf("invalid folder path: %w", err)
	}

	response := &InspectProjectResponse{
		FolderPath: absPath,
	}

	// Check if folder exists
	exists, err := s.scanner.Exists(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check folder: %w", err)
	}

	if !exists {
		return response, nil
	}

	response.Exists = true

	// Check if it's a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat folder: %w", err)
	}

	response.IsDir = info.IsDir()
	if !response.IsDir {
		return response, nil
	}

	// Read project metadata
	meta, metaErr := s.readProjectMetadata(absPath)
	if metaErr != "" {
		response.HasBasMetadata = true
		response.MetadataError = metaErr
	} else if meta != nil {
		response.HasBasMetadata = true
		response.SuggestedName = strings.TrimSpace(meta.Name)
		response.SuggestedDescription = strings.TrimSpace(meta.Description)
	}

	// Check for workflow files
	response.HasWorkflows = s.hasWorkflowFiles(absPath)

	// Check if already indexed
	existing, err := s.projecter.GetProjectByFolderPath(ctx, absPath)
	if err == nil && existing != nil {
		response.AlreadyIndexed = true
		response.IndexedProjectID = existing.ID.String()
		if strings.TrimSpace(response.SuggestedName) == "" {
			response.SuggestedName = existing.Name
		}
	}

	return response, nil
}

// Scan scans for project folders.
func (s *Service) Scan(ctx context.Context, req *ScanProjectsRequest) (*ScanProjectsResponse, error) {
	// Determine scan path
	scanPath := req.Path
	defaultRoot := shared.DefaultProjectsRoot()

	if scanPath == "" {
		scanPath = defaultRoot
	}

	// Normalize path
	absPath, err := shared.NormalizePath(scanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists
	exists, _ := s.scanner.Exists(ctx, absPath)
	if !exists {
		return &ScanProjectsResponse{
			Path:                absPath,
			DefaultProjectsRoot: defaultRoot,
			Entries:             []ProjectEntry{},
		}, nil
	}

	// Scan for entries
	entries, err := s.scanner.ScanDirectory(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Get existing projects
	indexed, _ := s.projecter.ListProjects(ctx, 1000, 0)
	indexedByPath := make(map[string]*shared.ProjectIndexData)
	for _, p := range indexed {
		indexedByPath[p.FolderPath] = p
	}

	// Build response entries
	result := &ScanProjectsResponse{
		Path:                absPath,
		DefaultProjectsRoot: defaultRoot,
		Entries:             []ProjectEntry{},
	}

	// Calculate parent
	parent := filepath.Dir(absPath)
	if parent != absPath && parent != "" {
		result.Parent = &parent
	}

	for _, entry := range entries {
		if !entry.IsDir {
			continue
		}

		projectEntry := ProjectEntry{
			Name: entry.Name,
			Path: entry.Path,
		}

		// Check if this is a project folder
		if shared.IsProjectDir(entry.Path) {
			projectEntry.IsProject = true

			// Try to get suggested name from metadata
			meta, _ := s.readProjectMetadata(entry.Path)
			if meta != nil && strings.TrimSpace(meta.Name) != "" {
				projectEntry.SuggestedName = strings.TrimSpace(meta.Name)
			}
		}

		// Check if already indexed
		if existing, ok := indexedByPath[entry.Path]; ok {
			projectEntry.IsRegistered = true
			projectEntry.ProjectID = existing.ID.String()
			if projectEntry.SuggestedName == "" {
				projectEntry.SuggestedName = existing.Name
			}
		}

		result.Entries = append(result.Entries, projectEntry)
	}

	return result, nil
}

// Import imports a project from a folder.
func (s *Service) Import(ctx context.Context, req *ImportProjectRequest) (*shared.ProjectIndexData, error) {
	if strings.TrimSpace(req.FolderPath) == "" {
		return nil, errors.New("folder_path is required")
	}

	// Validate and normalize path
	absPath, err := shared.NormalizePath(req.FolderPath)
	if err != nil {
		return nil, fmt.Errorf("invalid folder path: %w", err)
	}

	// Check if folder exists
	exists, err := s.scanner.Exists(ctx, absPath)
	if err != nil || !exists {
		return nil, errors.New("folder does not exist")
	}

	// Check if it's a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat folder: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}

	// Check if already indexed (idempotent)
	existing, err := s.projecter.GetProjectByFolderPath(ctx, absPath)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Read metadata for defaults
	meta, _ := s.readProjectMetadata(absPath)

	// Determine name
	name := strings.TrimSpace(req.Name)
	if name == "" && meta != nil {
		name = strings.TrimSpace(meta.Name)
	}
	if name == "" {
		name = filepath.Base(absPath)
	}

	// Determine description
	description := strings.TrimSpace(req.Description)
	if description == "" && meta != nil {
		description = strings.TrimSpace(meta.Description)
	}

	// Create project
	projectID := uuid.New()

	// Try to use ID from metadata if available
	if meta != nil && strings.TrimSpace(meta.Id) != "" {
		if parsed, parseErr := uuid.Parse(strings.TrimSpace(meta.Id)); parseErr == nil {
			// Check if this ID is already in use
			byID, _ := s.projecter.GetProjectByID(ctx, parsed)
			if byID == nil {
				projectID = parsed
			}
		}
	}

	projectData := &shared.ProjectIndexData{
		ID:          projectID,
		Name:        name,
		Description: description,
		FolderPath:  absPath,
	}

	if err := s.projecter.CreateProject(ctx, projectData); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Sync workflows after project creation
	if s.syncer != nil {
		if err := s.syncer.SyncProjectWorkflows(ctx, projectID); err != nil {
			s.log.WithError(err).WithField("project_id", projectID.String()).Warn("Imported project workflow sync failed")
		}
	}

	return projectData, nil
}
