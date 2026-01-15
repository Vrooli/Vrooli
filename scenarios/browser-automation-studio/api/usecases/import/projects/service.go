package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
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

// WorkflowDetectionResult contains the results of workflow detection.
type WorkflowDetectionResult struct {
	Found     bool
	Count     int
	Locations []string
}

// detectWorkflows searches for workflow JSON files in any directory up to maxDepth levels.
// A JSON file is considered a workflow if it contains a "nodes" array.
func (s *Service) detectWorkflows(folderPath string, maxDepth int) *WorkflowDetectionResult {
	result := &WorkflowDetectionResult{
		Locations: []string{},
	}

	if maxDepth <= 0 {
		maxDepth = 4
	}

	_ = filepath.WalkDir(folderPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Calculate depth
		relPath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return nil
		}

		// Enforce depth limit for directories
		depth := strings.Count(relPath, string(os.PathSeparator))
		if d.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}

		// Check JSON files
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			if s.isWorkflowJSONFile(path) {
				result.Count++
				result.Locations = append(result.Locations, relPath)
				result.Found = true
			}
		}

		return nil
	})

	return result
}

// isWorkflowJSONFile checks if a JSON file contains a "nodes" array (workflow structure).
func (s *Service) isWorkflowJSONFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var content map[string]interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		return false
	}

	// Check for "nodes" array - the key indicator of a workflow file
	if nodes, ok := content["nodes"]; ok {
		if _, isArray := nodes.([]interface{}); isArray {
			return true
		}
	}

	return false
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
		Validation: shared.NewValidationSummary(),
	}

	// Check 1: Folder exists
	exists, err := s.scanner.Exists(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check folder: %w", err)
	}

	if !exists {
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "folder_exists",
			Status:      shared.ValidationStatusError,
			Label:       "Folder not found",
			Description: "The specified folder path does not exist on the filesystem.",
		})
		response.Validation.ComputeOverallStatus()
		return response, nil
	}

	response.Exists = true
	response.Validation.AddCheck(shared.ValidationCheck{
		ID:          "folder_exists",
		Status:      shared.ValidationStatusPass,
		Label:       "Folder exists",
		Description: "The specified folder path exists on the filesystem.",
	})

	// Check 2: Is directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat folder: %w", err)
	}

	response.IsDir = info.IsDir()
	if !response.IsDir {
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "is_directory",
			Status:      shared.ValidationStatusError,
			Label:       "Not a directory",
			Description: "The path must point to a directory, not a file.",
		})
		response.Validation.ComputeOverallStatus()
		return response, nil
	}

	response.Validation.AddCheck(shared.ValidationCheck{
		ID:          "is_directory",
		Status:      shared.ValidationStatusPass,
		Label:       "Is directory",
		Description: "The path points to a valid directory.",
	})

	// Check 3: Project metadata (.bas/project.json)
	meta, metaErr := s.readProjectMetadata(absPath)
	if metaErr != "" {
		response.HasBasMetadata = true
		response.MetadataError = metaErr
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "project_metadata",
			Status:      shared.ValidationStatusWarn,
			Label:       "Metadata parse error",
			Description: fmt.Sprintf("Found .bas/project.json but failed to parse: %s", metaErr),
		})
	} else if meta != nil {
		response.HasBasMetadata = true
		response.SuggestedName = strings.TrimSpace(meta.Name)
		response.SuggestedDescription = strings.TrimSpace(meta.Description)
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "project_metadata",
			Status:      shared.ValidationStatusPass,
			Label:       "Project metadata",
			Description: "Found valid .bas/project.json with project configuration.",
		})
	} else {
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "project_metadata",
			Status:      shared.ValidationStatusInfo,
			Label:       "No project metadata",
			Description: "No .bas/project.json found. This is optional - one can be created during import.",
		})
	}

	// Check 4: Workflow detection (using improved detection)
	workflowResult := s.detectWorkflows(absPath, 4)
	response.HasWorkflows = workflowResult.Found
	response.WorkflowCount = workflowResult.Count
	response.WorkflowLocations = workflowResult.Locations

	if workflowResult.Found {
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "has_workflows",
			Status:      shared.ValidationStatusPass,
			Label:       "Workflows detected",
			Description: fmt.Sprintf("Found %d workflow file(s) with valid structure.", workflowResult.Count),
			Context: map[string]any{
				"count":     workflowResult.Count,
				"locations": workflowResult.Locations,
			},
		})
	} else {
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "has_workflows",
			Status:      shared.ValidationStatusInfo,
			Label:       "No workflows found",
			Description: "No workflow files detected. You can create workflows after import.",
		})
	}

	// Check 4b: V1 workflow detection (legacy format that will be converted)
	tempProject := &database.ProjectIndex{FolderPath: absPath}
	v1Count, v1Err := workflowservice.CountV1Workflows(tempProject, 4)
	if v1Err == nil && v1Count > 0 {
		response.V1WorkflowCount = v1Count
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "v1_workflows",
			Status:      shared.ValidationStatusWarn,
			Label:       "Legacy workflows detected",
			Description: fmt.Sprintf("Found %d workflow(s) using legacy V1 format. These will be automatically converted to V2 format during import.", v1Count),
			Context: map[string]any{
				"v1_count": v1Count,
			},
		})
	}

	// Check 5: Already indexed
	existing, err := s.projecter.GetProjectByFolderPath(ctx, absPath)
	if err == nil && existing != nil {
		response.AlreadyIndexed = true
		response.IndexedProjectID = existing.ID.String()
		if strings.TrimSpace(response.SuggestedName) == "" {
			response.SuggestedName = existing.Name
		}
		response.Validation.AddCheck(shared.ValidationCheck{
			ID:          "already_indexed",
			Status:      shared.ValidationStatusWarn,
			Label:       "Already indexed",
			Description: "This folder is already registered as a project. Importing will return the existing project.",
			Context: map[string]any{
				"project_id": existing.ID.String(),
			},
		})
	}

	response.Validation.ComputeOverallStatus()
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
