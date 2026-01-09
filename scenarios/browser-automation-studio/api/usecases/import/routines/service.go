package routines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

// Service handles routine/workflow import business logic.
type Service struct {
	scanner   shared.DirectoryScanner
	indexer   shared.WorkflowIndexer
	projecter shared.ProjectIndexer
	validator *Validator
	log       *logrus.Logger
}

// NewService creates a new Service.
func NewService(
	scanner shared.DirectoryScanner,
	indexer shared.WorkflowIndexer,
	projecter shared.ProjectIndexer,
	log *logrus.Logger,
) *Service {
	return &Service{
		scanner:   scanner,
		indexer:   indexer,
		projecter: projecter,
		validator: NewValidator(),
		log:       log,
	}
}

// InspectFile inspects a workflow file for import.
func (s *Service) InspectFile(ctx context.Context, projectID uuid.UUID, filePath string) (*InspectRoutineResponse, error) {
	// Validate file path
	if err := s.validator.ValidateFilePath(filePath); err != nil {
		return nil, err
	}

	response := &InspectRoutineResponse{
		FilePath: filePath,
	}

	// Check if file exists
	exists, err := s.scanner.Exists(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check file: %w", err)
	}
	response.Exists = exists

	if !exists {
		return response, nil
	}

	// Read file content
	content, err := s.scanner.ReadFile(ctx, filePath)
	if err != nil {
		response.ValidationError = fmt.Sprintf("Failed to read file: %v", err)
		return response, nil
	}

	// Validate workflow
	validation, err := s.validator.ValidateWorkflowJSON(content)
	if err != nil {
		response.ValidationError = fmt.Sprintf("Validation error: %v", err)
		return response, nil
	}

	response.IsValid = validation.IsValid
	if !validation.IsValid && len(validation.Errors) > 0 {
		response.ValidationError = validation.Errors[0].Message
	}

	// Extract preview
	preview, err := s.validator.ExtractWorkflowPreview(content)
	if err != nil {
		s.log.WithError(err).Debug("Failed to extract workflow preview")
	} else {
		response.Preview = preview
	}

	// Check if already indexed
	// Use the filename to check against existing workflows in the project
	filename := filepath.Base(filePath)
	relPath := filepath.Join(shared.WorkflowsDir, filename)

	existing, err := s.indexer.GetWorkflowByFilePath(ctx, projectID, relPath)
	if err == nil && existing != nil {
		response.AlreadyIndexed = true
		response.IndexedID = existing.ID.String()
	}

	return response, nil
}

// Import imports a workflow file into a project.
func (s *Service) Import(ctx context.Context, projectID uuid.UUID, req *ImportRoutineRequest) (*ImportRoutineResponse, error) {
	// Validate request
	if err := s.validator.ValidateFilePath(req.FilePath); err != nil {
		return nil, err
	}

	// Get project info
	project, err := s.projecter.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Check source file exists
	exists, err := s.scanner.Exists(ctx, req.FilePath)
	if err != nil || !exists {
		return nil, fmt.Errorf("source file does not exist")
	}

	// Read and validate source
	content, err := s.scanner.ReadFile(ctx, req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	validation, err := s.validator.ValidateWorkflowJSON(content)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if !validation.IsValid && len(validation.Errors) > 0 {
		return nil, fmt.Errorf("invalid workflow: %s", validation.Errors[0].Message)
	}

	// Extract preview for metadata
	preview, err := s.validator.ExtractWorkflowPreview(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Determine destination path
	destFilename := filepath.Base(req.FilePath)
	if req.DestPath != "" {
		destFilename = req.DestPath
		if !strings.HasSuffix(strings.ToLower(destFilename), ".json") {
			destFilename += ".workflow.json"
		}
	}

	// Ensure it ends with .workflow.json
	if !strings.HasSuffix(strings.ToLower(destFilename), ".workflow.json") {
		destFilename = strings.TrimSuffix(destFilename, ".json") + ".workflow.json"
	}

	relPath := filepath.Join(shared.WorkflowsDir, destFilename)
	destPath, err := shared.SafeJoin(project.FolderPath, relPath)
	if err != nil {
		return nil, fmt.Errorf("invalid destination path: %w", err)
	}

	// Check if destination exists
	destExists, _ := s.scanner.Exists(ctx, destPath)
	if destExists && !req.OverwriteIfExists {
		return nil, fmt.Errorf("workflow already exists at destination")
	}

	// Copy file to destination
	if err := s.scanner.CopyFile(ctx, req.FilePath, destPath); err != nil {
		return nil, fmt.Errorf("failed to copy workflow: %w", err)
	}

	// Determine workflow name
	name := preview.Name
	if req.Name != "" {
		name = req.Name
	}
	if name == "" {
		// Use filename without extension
		name = strings.TrimSuffix(destFilename, ".workflow.json")
		name = strings.TrimSuffix(name, ".json")
	}

	// Create or update index
	workflowID := uuid.New()
	if preview.ID != "" {
		if parsed, err := uuid.Parse(preview.ID); err == nil {
			// Check if this ID is already used
			existing, _ := s.indexer.GetWorkflowByID(ctx, parsed)
			if existing == nil {
				workflowID = parsed
			}
		}
	}

	indexData := &shared.WorkflowIndexData{
		ID:          workflowID,
		ProjectID:   projectID,
		Name:        name,
		Description: preview.Description,
		FilePath:    relPath,
		Version:     preview.Version,
		NodeCount:   preview.NodeCount,
		EdgeCount:   preview.EdgeCount,
		Tags:        preview.Tags,
	}

	if err := s.indexer.CreateWorkflowIndex(ctx, projectID, indexData); err != nil {
		// Log but don't fail - the file was copied successfully
		s.log.WithError(err).WithField("workflow_id", workflowID).Warn("Failed to create workflow index")
	}

	response := &ImportRoutineResponse{
		WorkflowID: workflowID.String(),
		Name:       name,
		Path:       relPath,
	}

	// Add warnings from validation
	for _, w := range validation.Warnings {
		response.Warnings = append(response.Warnings, w.Message)
	}

	return response, nil
}

// Scan scans a directory for workflow files.
func (s *Service) Scan(ctx context.Context, projectID uuid.UUID, req *ScanRoutinesRequest) (*ScanRoutinesResponse, error) {
	// Get project info
	project, err := s.projecter.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Determine scan path
	scanPath := req.Path
	if scanPath == "" {
		scanPath = filepath.Join(project.FolderPath, shared.WorkflowsDir)
	}

	// Validate path is within project
	isSubPath, err := shared.IsSubPath(project.FolderPath, scanPath)
	if err != nil || !isSubPath {
		return nil, fmt.Errorf("path must be within project directory")
	}

	// Check if path exists
	exists, _ := s.scanner.Exists(ctx, scanPath)
	if !exists {
		// Return empty result for non-existent workflows directory
		return &ScanRoutinesResponse{
			Path:    scanPath,
			Entries: []RoutineEntry{},
		}, nil
	}

	// Scan for entries
	entries, err := s.scanner.ScanDirectory(ctx, scanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Get existing indexed workflows for this project
	indexed, _ := s.indexer.ListWorkflowsByProject(ctx, projectID)
	indexedByPath := make(map[string]*shared.WorkflowIndexData)
	for _, w := range indexed {
		indexedByPath[w.FilePath] = w
	}

	// Build response entries
	depth := req.Depth
	if depth < 1 {
		depth = 1
	}
	if depth > 2 {
		depth = 2
	}

	result := &ScanRoutinesResponse{
		Path:    scanPath,
		Entries: []RoutineEntry{},
	}

	// Calculate parent
	parent := filepath.Dir(scanPath)
	if parent != project.FolderPath {
		result.Parent = &parent
	}

	for _, entry := range entries {
		if entry.IsDir {
			// Include directories for navigation (if depth > 1)
			if depth > 1 {
				result.Entries = append(result.Entries, RoutineEntry{
					Name: entry.Name,
					Path: entry.Path,
				})
			}
			continue
		}

		// Check if it's a workflow file
		if !shared.IsWorkflowFile(entry.Name) {
			continue
		}

		routineEntry := RoutineEntry{
			Name: entry.Name,
			Path: entry.Path,
		}

		// Check if indexed
		relPath, _ := shared.RelativePath(project.FolderPath, entry.Path)
		if existing, ok := indexedByPath[relPath]; ok {
			routineEntry.IsRegistered = true
			routineEntry.WorkflowID = existing.ID.String()
			routineEntry.PreviewName = existing.Name
		}

		// Try to validate and get preview name
		content, err := s.scanner.ReadFile(ctx, entry.Path)
		if err == nil {
			validation, _ := s.validator.ValidateWorkflowJSON(content)
			if validation != nil {
				routineEntry.IsValid = validation.IsValid
			}

			if routineEntry.PreviewName == "" {
				preview, _ := s.validator.ExtractWorkflowPreview(content)
				if preview != nil && preview.Name != "" {
					routineEntry.PreviewName = preview.Name
				}
			}
		}

		result.Entries = append(result.Entries, routineEntry)
	}

	return result, nil
}
